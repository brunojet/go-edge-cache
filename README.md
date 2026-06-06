# go-edge-cache

CloudFront Media Proxy com cache inteligente em S3 e fallback via Lambda (Go).

Serve mídia por uma CDN (CloudFront) com signed URLs. No cache miss, uma Lambda
busca o arquivo na origem (proxy do ServiceNow via S3), grava em `/cdn/` e
redireciona o cliente para a URL assinada. Objetos imutáveis (identificados por
`sysid`) e gestão de custo via S3 Intelligent-Tiering.

## Arquitetura (resumo)

```
Cliente → CloudFront ──(hit)──→ S3 /cdn/{sysid}
              │
              └──(miss / 4xx,5xx)──→ Lambda fallback
                                        ├── lock distribuído (S3)
                                        ├── download da origem (S3 root)
                                        ├── upload streaming → S3 /cdn/
                                        └── 302 → signed URL
```

Detalhes completos em [`context-docs/arquitetura-cloudfront-media-proxy.docx.md`](context-docs/arquitetura-cloudfront-media-proxy.docx.md).

## Estrutura do repositório

| Caminho | Conteúdo |
|---|---|
| `cmd/fallback/` | Lambda handler do fallback CloudFront (streaming download→upload) |
| `cmd/sign-url/` | CLI para gerar CloudFront signed URLs |
| `internal/cdn/` | Assinatura de URLs do CloudFront |
| `internal/secrets/` | Leitura de credenciais no Secrets Manager |
| `internal/models/` | Tipos compartilhados |
| `terraform/` | IaC: S3, CloudFront, Lambda, IAM, monitoring (módulos em `terraform/modules/`) |
| `env/` | `terraform.tfvars` por ambiente (`env/dev/`) |
| `scripts/` | Build da Lambda, diagnóstico, invalidação de cache |
| `tools/` | Utilitários de signed URL |
| `bootstrap/` | Passos pré-Terraform (ex: backend de state) |
| `docs/` | Documentação operacional e de design |

## Comandos úteis

```bash
# Testes
go test ./...

# Build da Lambda (arm64, provided.al2)
bash scripts/build-lambda.sh

# Deploy (dev)
cd terraform
terraform init
terraform apply -var-file=../env/dev/terraform.tfvars

# Formatar Terraform
terraform fmt -recursive
```

## Documentação

Veja o índice completo em [`docs/README.md`](docs/README.md). Atalhos:

- [Deploy com Terraform](docs/deployment-terraform.md)
- [Build & deploy da Lambda](docs/deployment-lambda.md)
- [Desenvolvimento local](docs/local-development.md)
- [Configuração do CloudFront](docs/cloudfront.md)
- [Signed URLs](docs/signed-urls.md)
- [Plano de mitigação de riscos](docs/risk-mitigation-plan.md)

## Convenções

Commits seguem [Conventional Commits](CLAUDE.md). Terraform é formatado com `terraform fmt`.
