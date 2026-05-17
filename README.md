# go-edge-cache

CloudFront Media Proxy — estrutura inicial do projeto.

Este repositório contém o esqueleto para implementar um Media Proxy usando CloudFront, Lambda (Go) e S3.

Fases iniciais

1. Infraestrutura (Terraform) — placeholder em `terraform/`.
2. Lambda em Go (`lambda/`) — handler e testes.
3. Pacotes internos em `internal/` para configurações e clientes.

Comandos úteis

```bash
# Rodar testes
go test ./...

# Build Lambda (Linux binary)
cd lambda
GOOS=linux GOARCH=amd64 go build -o bootstrap main.go
zip -j function.zip bootstrap

# Deploy Terraform (placeholder)
cd terraform
terraform init
terraform apply
```
