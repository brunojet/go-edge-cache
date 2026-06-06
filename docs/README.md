# Documentação — go-edge-cache

Índice da documentação operacional e de design do projeto.

## Arquitetura

| Doc | Descrição |
|---|---|
| [architecture.md](architecture.md) | Visão geral, diagrama de componentes e de sequência (Mermaid), ciclo de vida do objeto |

## Deploy & infraestrutura

| Doc | Descrição |
|---|---|
| [deployment-terraform.md](deployment-terraform.md) | Workflow de deploy com Terraform e `-var-file` por ambiente |
| [deployment-lambda.md](deployment-lambda.md) | Build e deploy do handler Lambda do fallback |
| [lambda-activation.md](lambda-activation.md) | Troubleshooting de ativação da Lambda no CloudFront |
| [cloudfront.md](cloudfront.md) | Configuração de cache, comportamentos e Origin Shield |

## Desenvolvimento & testes

| Doc | Descrição |
|---|---|
| [local-development.md](local-development.md) | Debug local (LocalStack, build, simulação de requests) |
| [debugging.md](debugging.md) | Guia de depuração do fallback e fluxo completo |
| [testing.md](testing.md) | Testes manuais de cache control e Origin Shield |

## Segurança & operação

| Doc | Descrição |
|---|---|
| [signed-urls.md](signed-urls.md) | Workflow recomendado de CloudFront signed URLs |
| [risk-mitigation-plan.md](risk-mitigation-plan.md) | Plano de mitigação de riscos (13 riscos avaliados) |

## Design / contexto

A arquitetura corrente (com diagramas Mermaid) está em [architecture.md](architecture.md).
Material de design original e diretrizes em [`../context-docs/`](../context-docs/):

- [Arquitetura — CloudFront Media Proxy (documento base)](../context-docs/arquitetura-cloudfront-media-proxy.docx.md)
- [Diretrizes de GC e complexidade](../context-docs/GC_GUIDELINES.md)

## READMEs co-localizados

Cada componente mantém seu próprio README:

- [`../terraform/README.md`](../terraform/README.md) — módulos Terraform
- [`../env/README.md`](../env/README.md) — configuração por ambiente
- [`../tools/README.md`](../tools/README.md) — utilitários de signed URL
- [`../cmd/sign-url/README.md`](../cmd/sign-url/README.md) — CLI sign-url
- [`../bootstrap/README.md`](../bootstrap/README.md) — passos pré-Terraform
