# CloudFront Signed URL Generator

Ferramentas para gerar URLs assinadas do CloudFront para acessar arquivos com autenticação.

## Pré-requisitos

- AWS credentials configuradas (via `~/.aws/credentials` ou variáveis de ambiente)
- Secret Manager com chaves em `/go-edge-key-management/rotator`
- Para Go: `go 1.24+`
- Para Python: `python3.8+` + `cryptography` + `boto3`

## Setup Python

```bash
pip install cryptography boto3
```

## Uso

### Python (Recomendado - mais simples)

```bash
# Gera URL assinada com expiração padrão (1 hora)
python3 tools/sign_url.py --file /image.jpg

# URL com expiração custom (24 horas)
python3 tools/sign_url.py --file /video.mp4 --expires 86400

# Com domain customizado
python3 tools/sign_url.py --domain media.brunojet.com.br --file /photo.png
```

### Go

```bash
cd cmd/sign-url

# Build
go build -o sign-url main.go

# Run
./sign-url --file /image.jpg

# Com opções customizadas
./sign-url \
  --domain media.brunojet.com.br \
  --file /video.mp4 \
  --expires 86400
```

## Saída

A URL assinada contém 3 parâmetros QueryString:

- `Policy`: Política de acesso em base64
- `Signature`: Assinatura RSA-SHA1
- `Key-Pair-Id`: ID da chave pública usada para validar

**Exemplo:**
```
https://media.brunojet.com.br/image.jpg?Policy=eyJT...&Signature=abcd...&Key-Pair-Id=33bd9f...
```

## Fluxo

1. **Fetch Secret**: Lê `/go-edge-key-management/rotator` do AWS Secrets Manager
2. **Parse Keys**: Extrai private_key, public_key, key_group_id do secret
3. **Create Policy**: Gera política JSON com expiração
4. **Sign**: Assina com RSA-SHA1
5. **Encode**: Converte para Base64 URL-safe
6. **Generate URL**: Monta URL com parâmetros

## Estrutura do Secret

O secret deve conter um JSON com:

```json
{
  "private_key": "-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----",
  "public_key": "-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----",
  "key_group_id": "33bd9f09-5f1c-4976-806a-0fb5b8b70241"
}
```

A **fonte da verdade é sempre o Secrets Manager**. Você pode fazer rotação de chaves atualizando o secret sem alterar o código.

## Variáveis de Ambiente

Configure defaults via env vars:

```bash
export CF_DOMAIN=media.brunojet.com.br
export CF_SECRET=/go-edge-key-management/rotator
export AWS_REGION=us-east-1

python3 tools/sign_url.py --file /image.jpg
```

## Troubleshooting

### "Failed to get secret"
- Verifica AWS credentials: `aws sts get-caller-identity`
- Verifica se o secret existe: `aws secretsmanager get-secret-value --secret-id /go-edge-key-management/rotator`

### "key_group_id not found in secret"
- Certifica que o secret contém o campo `key_group_id`
- Ou passa `--key-group XXX` explicitamente

### URL retornada não funciona
- Verifica se o arquivo existe no S3 bucket
- Verifica se o domain name está correto
- Verifica se a expiração não passou

## Referências

- [CloudFront Signed URLs](https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/private-content-signed-urls.html)
- [AWS Secrets Manager](https://docs.aws.amazon.com/secretsmanager/)
