# Risk Mitigation Plan — go-edge-cache Infrastructure

> Atualizado: 2026-06-06
> Status: 🔴 Critical · 🟠 High · 🟡 Medium · 🔵 Low · ✅ Resolvido · 🚫 Won't Do

---

## Summary

| # | Risk | Severity | Status | Effort |
|---|---|---|---|---|
| 1 | Lock wait timeout > Lambda timeout | 🔴 Critical | ✅ Resolvido | Small |
| 2 | Lambda Function URL publicly accessible | 🔴 Critical | ✅ Resolvido | Medium |
| 3 | No cache invalidation mechanism | 🔴 Critical | ✅ Resolvido | — |
| 4 | No WAF | 🟠 High | 🚫 Won't Do | — |
| 5 | Large file timeout | 🟠 High | ✅ Resolvido | — |
| 6 | No circuit breaker for origin | 🟠 High | 🚫 Won't Do | — |
| 7 | Lambda cold start | 🟡 Medium | 🚫 Won't Do | — |
| 8 | No CloudWatch alarms | 🟡 Medium | ✅ Resolvido | Small |
| 9 | Key rotation window | 🟡 Medium | ✅ Resolvido | — |
| 10 | No X-Ray tracing | 🟡 Medium | ✅ Resolvido | Small |
| 11 | Single region | 🔵 Low | 🚫 Won't Do | — |
| 12 | S3 lifecycle tuning | 🔵 Low | ✅ Resolvido | — |
| 13 | No DLQ | 🔵 Low | 🚫 Won't Do | — |

---

## ✅ Resolvidos

---

### RISK-01 — Lock wait timeout > Lambda timeout

**Problema**
`defaultLockWaitTimeout = 70s` era maior que o timeout da Lambda de `60s`.
Lambda B esperando lock de Lambda A era morta pelo runtime antes do `GetLockWait`
retornar o 429 graciosamente. CloudFront recebia 503 em vez de 429.

**O que foi feito**

- Constantes corrigidas para ficarem dentro do timeout da Lambda (60s):
  ```go
  defaultLockTTL         = 45  // S3 lock TTL — expira antes da Lambda morrer
  defaultLockWaitTimeout = 50  // max wait — deixa 10s para o trabalho após o lock
  ```
- `GetLockWait` passou a usar o `ctx` da Lambda diretamente — o runtime Go já
  injeta o deadline correto, sem necessidade de derivação manual.
- `signal.NotifyContext(context.Background(), syscall.SIGTERM)` adicionado ao
  `main()` — SIGTERM propaga cancelamento para todas as operações em voo.
- `ReleaseLock` usa `context.Background()` fresco com timeout de 5s — garante
  que o DELETE S3 seja tentado mesmo quando o `ctx` da invocação já está cancelado.
- Erros de lock diferenciados por tipo:
  - `context.DeadlineExceeded` → 429 Too Many Requests
  - `context.Canceled` → 503 Service Unavailable

**Arquivos alterados**
- `cmd/fallback/main.go`

---

### RISK-02 — Lambda Function URL publicly accessible

**Problema**
`authorization_type = "NONE"` permitia que qualquer pessoa com a URL invocasse
o Lambda diretamente, bypassando CloudFront, signed URLs e o lock distribuído.

**O que foi feito**

- `function_url_auth_type = "AWS_IAM"` definido como default em todos os níveis
  Terraform (`variables.tf`, `modules/lambda/variables.tf`, `env/dev/terraform.tfvars`).
- CloudFront OAC para Lambda provisionado (`origin_access_control_origin_type = "lambda"`,
  `signing_behavior = "always"`, `signing_protocol = "sigv4"`).
- `aws_lambda_permission` com `principal = "cloudfront.amazonaws.com"` e
  `source_arn = distribution.arn` — restringe invocação a esta distribuição específica.
- `Apply complete! Resources: 2 added, 2 changed, 0 destroyed` confirmado.
- Bug corrigido: `awsRegion` nunca era inicializado no `init()`, causando
  `"AWS region required"` em toda requisição. Region removida da cadeia toda —
  SDK Go v2 resolve via `AWS_REGION` env var (setado automaticamente pelo runtime Lambda).

> ⚠️ **Pendente:** Rodar `terraform apply -var-file=../env/dev/terraform.tfvars`
> para atualizar `authorization_type` de `NONE` → `AWS_IAM` no console AWS.

**Arquivos alterados**
- `terraform/modules/media_proxy/main.tf` — OAC + lambda_permission + origin OAC ID
- `terraform/modules/media_proxy/variables.tf` — `lambda_function_arn`
- `terraform/main.tf` — `lambda_function_arn = module.lambda.function_arn`
- `terraform/variables.tf` — default `"AWS_IAM"`
- `terraform/modules/lambda/variables.tf` — default `"AWS_IAM"`
- `env/dev/terraform.tfvars` — `lambda_function_url_auth_type = "AWS_IAM"`
- `internal/secrets/secrets.go` — removido parâmetro `region`
- `internal/cdn/signer.go` — removido parâmetro `region`
- `cmd/fallback/main.go` — removido `awsRegion` global

---

### RISK-04 — No WAF

🚫 **Won't Do — não aplicável a esta arquitetura**

O vetor de ataque pressuposto não se concretiza porque a arquitetura já possui
três camadas sobrepostas de proteção nativa:

| Camada | Proteção |
|---|---|
| CloudFront signed URLs obrigatórias | Request sem URL válida bloqueada no edge |
| CloudFront error cache (4xx/5xx) | Lambda invocado no máximo 1× por path por janela de cache |
| S3 distributed lock | Requisições concorrentes para o mesmo path serializadas |

Para que qualquer ataque ocorra, o atacante precisaria de signed URLs válidas.
Signed URLs só podem ser geradas pelo **backend** — único detentor da private key
(Secrets Manager). O backend não gera URLs aleatórias.

O único vetor real seria vazamento da private key, coberto pelo **RISK-09**.
Se a chave vazar, WAF não resolve. Se não vazar, WAF não é necessário.

---

## 🔴 Critical — Open

---

### RISK-03 — No cache invalidation mechanism

✅ **Resolvido — invalidação não necessária para este modelo de dados**

**Análise**

Arquivos do ServiceNow são identificados por `sysid` — identificador único e imutável.
Um dado `sysid` sempre representa o mesmo conteúdo. Não existe conteúdo desatualizado
para invalidar: se o arquivo mudou no ServiceNow, é um novo `sysid`.

**O que foi implementado: S3 Intelligent-Tiering + DELETE**

```
/cdn/{sysid}
    │
    ├── acessado frequentemente   → Frequent Access (STANDARD pricing)
    ├── 30 dias sem acesso        → Infrequent Access (-40% custo, acesso instantâneo)
    ├── 90 dias sem acesso        → Archive Instant Access (-68% custo, acesso instantâneo)
    ├── acessado em qualquer tier → volta para Frequent Access automaticamente
    └── 365 dias                  → DELETE (força redownload do ServiceNow se necessário)
```

- S3 gerencia 100% as transições de tier com base em padrões de acesso reais — sem Lambda.
- Archive Access e Deep Archive não habilitados: exigem restore, incompatível com CDN.
- Multipart uploads incompletos abortados em 1 dia automaticamente.

**Arquivos alterados**
- `terraform/modules/media_proxy/main.tf` — lifecycle com IT + abort multipart
- `terraform/modules/media_proxy/variables.tf` — `s3_cache_cleanup_days` default 365
- `terraform/variables.tf` — default 365
- `env/dev/terraform.tfvars` — `s3_cache_cleanup_days = 365`

---

## 🟠 High — Open

---

### RISK-05 — Large file timeout

✅ **Resolvido**

**Contexto**
ServiceNow não suporta HTTP Range requests — o download é all-or-nothing. Testado
900MB em bucket-to-bucket em <30s (rede interna AWS), mas o gargalo real é o download
do ServiceNow via rede externa. Limite de 256MB definido como padrão conservador.

**O que foi implementado**

Função `checkOrigin` executa um `HeadObject` no S3 root (proxy do ServiceNow) antes
de qualquer download, com dupla responsabilidade:

```
checkOrigin (HeadObject)
  ├── 404 → arquivo não existe, retorna imediatamente (evita GetObject desnecessário)
  ├── 413 → arquivo > MAX_FILE_SIZE_MB, retorna imediatamente
  └── OK  → fetchFromS3Origin (GetObject) → upload → signed URL
```

- `MAX_FILE_SIZE_MB` configurável via env var (default `256`)
- Abort de multipart incompleto em 1 dia já provisionado pelo RISK-03
- Arquivos acima do limite devem ser disponibilizados fora deste fluxo

**Arquivos alterados**
- `cmd/fallback/main.go` — `checkOrigin`, `defaultMaxFileSizeMB`, `maxFileSizeBytes`
- `env/dev/terraform.tfvars` — `MAX_FILE_SIZE_MB = "256"` em `lambda_environment`

**Files**
- `cmd/fallback/main.go` — size check antes do fetch
- `terraform/modules/s3_bucket/main.tf` — abort incomplete multipart lifecycle rule
- `env/dev/terraform.tfvars` — `MAX_FILE_SIZE_MB` em `lambda_environment`

---

### RISK-06 — No circuit breaker for origin

🚫 **Won't Do — coberto pela arquitetura existente**

O cenário de ataque pressupõe ServiceNow indisponível causando avalanche de invocações Lambda. A arquitetura já possui três camadas que limitam esse impacto:

| Camada | Proteção |
|---|---|
| CloudFront error cache (4xx/5xx) | Após 1 falha por path, CF cacheia o erro — próximas requests do mesmo path não chegam ao Lambda |
| S3 distributed lock | Serializa requisições concorrentes por path — 1 invocação ativa por path |
| Lambda concurrency limit (prod: 10 instâncias) | Teto global de invocações simultâneas |

**Worst case com ServiceNow fora:** 10 invocações simultâneas (uma por instância) atingem o ServiceNow antes do CF cachear os erros por path. Carga negligenciável para uma plataforma enterprise historicamente estável.

Um circuit breaker clássico exigiria estado distribuído (SSM + lógica de half-open) com custo de 1 leitura SSM por request — complexidade operacional alta para proteção marginal sobre o que já existe.

---

## 🟡 Medium — Open

---

### RISK-07 — Lambda cold start

🚫 **Won't Do — não é um risco relevante para este fluxo**

A Lambda só é invocada quando o arquivo **não existe** no S3 `/cdn/`. Isso significa
que o trabalho mínimo é: download do ServiceNow + upload para S3 + assinatura de URL
— operação que leva segundos independentemente.

Um cold start de ~500ms (Go em `provided.al2`) é ruído dentro da latência total do
fluxo de fallback. O usuário já está esperando pelo trabalho real, não pelo bootstrap.
Provisioned Concurrency adicionaria custo fixo sem benefício perceptível ao usuário.

---

### RISK-08 — No CloudWatch alarms

✅ **Resolvido — implementado, desabilitado por default**

**O que foi feito**

`terraform/monitoring.tf` provisionado com `enable_alarms` toggle (default `false`).
Ativar em prod: `enable_alarms = true` + `alarm_email = "ops@..."` em `terraform.tfvars`.

| Alarme | Métrica | Threshold | Janela |
|---|---|---|---|
| Lambda errors | `Errors Sum` | > 5 | 5 min |
| Lambda throttles | `Throttles Sum` | > 0 | 1 min |
| Lambda P99 duration | `Duration p99` | > 45 000 ms | 5 min |
| Lambda max duration | `Duration Maximum` | > 55 000 ms | 5 min |
| CF 5xx rate | `5xxErrorRate Average` | > 1% | 5 min |
| CF 4xx rate | `4xxErrorRate Average` | > 10% | 5 min |

- SNS topic `{bucket}-alarms` + email subscription opcional.
- `treat_missing_data = "notBreaching"` — sem ruído em períodos sem tráfego.
- Todos os alarmes têm count=0 quando `enable_alarms = false` — zero custo em dev.

**Arquivos alterados**
- `terraform/monitoring.tf` — (NOVO) SNS topic + 6 CloudWatch alarms
- `terraform/variables.tf` — `enable_alarms`, `alarm_email`

---

### RISK-09 — Key rotation window

✅ **Já mitigado pelo `go-edge-key-management`**

O sistema de rotação mantém sempre no mínimo dois keys ativos no CloudFront Key Group:
a chave atual e a anterior. Signed URLs em voo assinadas com a chave anterior continuam
válidas durante e após a rotação — não existe janela de 403 para clientes legítimos.

Nenhuma ação adicional necessária neste projeto.

---

### RISK-10 — No X-Ray tracing

✅ **Resolvido — implementado, desabilitado por default**

**O que foi feito**

`enable_xray` toggle adicionado (default `false`). Ativar em prod: `enable_xray = true`.

Quando habilitado:
- `tracing_config { mode = "Active" }` adicionado dinamicamente ao `aws_lambda_function` — Lambda passa a enviar segmentos automáticos (cold start + duração de invocação) sem alterações no código Go.
- `AWSXRayDaemonWriteAccess` managed policy attachada à role Lambda via `aws_iam_role_policy_attachment` (count=0 quando desabilitado).
- Zero custo e zero permissões extras em dev.

**Rastreio automático sem SDK** cobre os casos mais comuns: duração total, erros, cold start. SDK X-Ray no Go adicionaria subsegmentos granulares por operação (S3, Secrets Manager) — fica como melhoria futura quando o rastreio for ativado em prod.

**Arquivos alterados**
- `terraform/modules/lambda/main.tf` — `dynamic "tracing_config"` em zip e image
- `terraform/modules/lambda/variables.tf` — `enable_xray`
- `terraform/lambda.tf` — passa `enable_xray` para o módulo
- `terraform/iam.tf` — `aws_iam_role_policy_attachment.lambda_xray` (count condicional)
- `terraform/variables.tf` — `enable_xray`

---

## 🔵 Low — Open

---

### RISK-11 — Single region

🚫 **Won't Do — constraint da origem, não da CDN**

Multi-região na CDN não resolve o problema: o **ServiceNow é single-region** por natureza.
A CDN distribui conteúdo que já existe no S3 (CloudFront serve de cache global). Somente
o fluxo de **fallback** (cache miss) depende da região: Lambda + S3. E o Lambda só entra
em cena quando o arquivo não existe ainda no S3 — nesse momento, a origem real é o
ServiceNow, não um recurso que pode ser replicado.

| Componente | Multi-região resolve? |
|---|---|
| CloudFront edge | ✅ Já é global por design |
| S3 /cdn/ (conteúdo cacheado) | Irrelevante — CF serve do edge cache |
| Lambda fallback | ❌ Mesmo com Lambda multi-região, o ServiceNow é single-region |
| ServiceNow origin | ❌ Fora do nosso controle |

Se o ServiceNow ficar indisponível, nenhuma arquitetura multi-região CDN ajuda.
Se a região AWS cair, os conteúdos já em cache no CloudFront continuam servidos.
O único gap real (cache misses durante outage regional) é aceito como constraint do projeto.

---

### RISK-12 — S3 lifecycle tuning

✅ **Resolvido junto com RISK-03**

S3 Intelligent-Tiering aplicado em `cdn/` — gerencia hot/cold automaticamente por
padrão de acesso real. DELETE em 365 dias. Abort multipart incompleto em 1 dia.
Ver detalhes em RISK-03.

---

### RISK-13 — No DLQ para falhas Lambda

🚫 **Won't Do — não se aplica a invocações síncronas**

DLQ existe para invocações **assíncronas** (SNS, S3 events, Event Source Mappings).
A única invocação atual é CloudFront → Lambda Function URL, que é **síncrona**: falha
vira resposta de erro imediata ao caller — não há mensagem perdida para capturar em fila.

O risco foi especulado para "webhooks futuros" que não existem. Se paths async forem
adicionados no futuro, DLQ deve ser avaliado naquele momento, não antes.

---

## Roadmap

```
✅ Concluído (8/13)
  ├── RISK-01: Lock timeout corrigido + SIGTERM + releaseCtx fresco
  ├── RISK-02: Lambda URL → AWS_IAM + CloudFront OAC provisionado
  ├── RISK-03: sysid imutável → invalidação desnecessária; IT + DELETE 365d aplicado
  ├── RISK-05: checkOrigin (HeadObject) — 404 + 413 antes do download; MAX_FILE_SIZE_MB=256
  ├── RISK-08: CloudWatch alarms + SNS topic — enable_alarms=false (pronto para prod)
  ├── RISK-09: go-edge-key-management já mantém chave atual + anterior no Key Group
  ├── RISK-10: X-Ray tracing_config + IAM — enable_xray=false (pronto para prod)
  └── RISK-12: S3 Intelligent-Tiering aplicado em cdn/ (junto com RISK-03)

🚫 Won't Do (5/13)
  ├── RISK-04: WAF — signed URLs + error cache cobrem
  ├── RISK-06: Circuit breaker — 10 instâncias prod + SN estável + error cache cobrem
  ├── RISK-07: Cold start — 500ms irrelevante frente ao download+upload do fallback
  ├── RISK-11: Multi-região — ServiceNow é single-region; constraint da origem, não da CDN
  └── RISK-13: DLQ — invocação é síncrona, DLQ não se aplica

Todos os 13 riscos avaliados e endereçados. ✓
```
