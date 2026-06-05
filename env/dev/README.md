# `env/dev` — Development environment

Conteúdo:
- `terraform.tfvars` — arquivo de variáveis para usar com `terraform plan -var-file=env/dev/terraform.tfvars`.

## Como usar

```bash
# inicializar e aplicar terraform a partir da pasta terraform/
cd terraform
terraform init
terraform plan -var-file=../env/dev/terraform.tfvars
terraform apply -var-file=../env/dev/terraform.tfvars
```

## Configuração

- **S3 bucket**: `brunojet-media-proxy-dev`
- **CloudFront aliases**: `media.brunojet.com.br`
- **Signed URLs**: Habilitado com keygroup `go-edge-key-group` (provisionado no projeto `go-edge-key-management`)
- **Lambda**: Desabilitado (pode ser habilitado alterando `enable_lambda = true`)
- **Secrets**: Desabilitado (pode ser habilitado alterando `enable_secrets = true`)

## Notas

- CloudFront ref o keygroup `go-edge-key-group` que já existe na infra
- Se usar Lambda, fornecer `lambda_image_uri` ou `lambda_s3_bucket` + `lambda_s3_key`
- ACM certificate é obrigatório para usar aliases customizados
