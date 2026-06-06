# Arquitetura — go-edge-cache

CloudFront Media Proxy com cache em S3 e fallback via Lambda (Go). Serve mídia
imutável (identificada por `sysid`) por uma CDN com signed URLs. No cache miss,
uma Lambda busca o artefato na origem, grava em `/cdn/` e redireciona o cliente
para a URL assinada.

> Diagramas em [Mermaid](https://mermaid.js.org/) — renderizam direto no GitHub.

> **Origem:** no mundo real a origem é a **API do ServiceNow**, que fornece os
> artefatos para download. Na PoC atual ela é simulada por um bucket S3 (download
> bucket-to-bucket). Diferenças relevantes estão anotadas nos diagramas — em
> especial, a API do ServiceNow **não** expõe `HEAD`, então a guarda de existência
> (404) e tamanho (413) feita hoje via `HeadObject` é específica da PoC.

## Componentes

```mermaid
flowchart LR
    Client(["Cliente"])
    CF["CloudFront<br/>signed URLs + OAC"]
    OG{{"Origin Group<br/>S3 primary / Lambda failover"}}
    S3C[("S3 /cdn/<br/>cache + Intelligent-Tiering")]
    L["Lambda fallback<br/>Go provided.al2 arm64"]
    SN["ServiceNow API<br/>origem dos artefatos<br/>(PoC: bucket S3 stand-in)"]
    SM["Secrets Manager<br/>chave de assinatura"]

    Client -->|"GET signed URL"| CF
    CF --> OG
    OG -->|"primary"| S3C
    OG -.->|"failover 403/404/5xx"| L
    L -->|"download artefato"| SN
    L -->|"upload streaming"| S3C
    L -->|"fetch key"| SM
    L -.->|"302 -> signed URL"| Client
```

| Componente | Papel |
|---|---|
| **CloudFront** | CDN com signed URLs (trusted key group). OAC sigv4 para S3 e Lambda. |
| **Origin Group** | S3 como origem primária; failover para a Lambda nos status `403, 404, 500, 502, 503, 504`. |
| **S3 `/cdn/`** | Cache de objetos servidos. Intelligent-Tiering + expiração de 365 dias. |
| **Lambda fallback** | Em cache miss: lock distribuído → valida origem → download → upload streaming → 302. |
| **ServiceNow API** | Origem real dos artefatos (download). **PoC:** simulada por um bucket S3 lido bucket-to-bucket. Sem `HEAD` e sem Range — download all-or-nothing. |
| **Secrets Manager** | Chave privada de assinatura (rotacionada por `go-edge-key-management`). |

## Fluxo de requisição (sequência)

```mermaid
sequenceDiagram
    autonumber
    actor C as Cliente
    participant CF as CloudFront
    participant S3C as S3 /cdn (cache)
    participant L as Lambda fallback
    participant SN as ServiceNow API (origem)
    participant SM as Secrets Manager

    C->>CF: GET /images/{sysid} (signed URL)
    CF->>S3C: busca /cdn/{path}

    alt cache hit no edge
        S3C-->>CF: 200 objeto
        CF-->>C: 200 (servido do cache)
    else cache miss (403/404) -> failover
        S3C-->>CF: 403/404
        CF->>L: invoca Lambda (OAC sigv4)

        L->>S3C: GetLockWait(cdn + path)
        Note over L,S3C: TTL 45s / espera 50s (< Lambda 60s)<br/>timeout->429 · cancelado->503

        L->>S3C: HeadObject /cdn/{path} (isCached)
        alt já populado por requisição concorrente
            S3C-->>L: 200
        else precisa buscar na origem
            Note over L,SN: checkOrigin: PoC usa HeadObject (S3) p/ guarda 404/413.<br/>ServiceNow API não tem HEAD — em prod validar pela resposta do download.
            L->>SN: checkOrigin (existência + tamanho)
            alt não existe
                SN-->>L: 404
                L-->>CF: 404
                CF-->>C: 404 (cacheado 300s)
            else excede MAX_FILE_SIZE_MB (256)
                SN-->>L: 200 metadata
                L-->>CF: 413
                CF-->>C: 413
            else ok
                SN-->>L: 200 metadata
                L->>SN: download do artefato (GET)
                SN-->>L: stream do corpo
                L->>S3C: PutObject /cdn/{path} (streaming)
            end
        end

        L->>SM: busca chave de assinatura
        SM-->>L: chave privada
        L->>L: assina URL (TTL 900s)
        L->>S3C: ReleaseLock (ctx fresco)
        L-->>CF: 302 -> signed URL
        CF-->>C: 302
        C->>CF: GET signed URL (re-request)
        CF->>S3C: busca /cdn/{path}
        S3C-->>CF: 200 objeto
        CF-->>C: 200
    end
```

**Pontos-chave do fluxo:**

- O **lock é adquirido antes** da checagem de cache — serializa requisições
  concorrentes pelo mesmo `path`, evitando downloads duplicados da origem.
- O `isCached` dentro da Lambda cobre a corrida em que outra invocação populou
  `/cdn/` enquanto esta esperava o lock.
- `checkOrigin` é a **dupla guarda** da origem: 404 (não existe) e 413 (maior que
  o limite). Na PoC isso é um `HeadObject` único no S3, evitando um `GetObject`
  desperdiçado. Como a **API do ServiceNow não tem `HEAD`**, em produção essa
  validação passa a vir dos headers/metadata da própria resposta de download.
- A Lambda **não devolve o corpo**: retorna `302` para a signed URL. O cliente
  re-requisita e aí o objeto já está em `/cdn/`, servido pelo S3.
- `ReleaseLock` usa `context.Background()` fresco (o ctx da invocação pode estar
  cancelado por timeout/SIGTERM no momento do `defer`).

### TTLs de cache de erro (CloudFront)

| Status | TTL | Racional |
|---|---|---|
| 404 | 300s | arquivo não existe na origem; improvável aparecer em 5 min |
| 403 | 60s | auth temporário; pode resolver após rotação de chave |
| 500 | 10s | erro interno da origem; retry rápido |
| 502 | 30s | conectividade; aguarda recuperação |
| 503 | 10s | origem sobrecarregada; retry rápido |
| 504 | 60s | timeout; origem lenta, não adianta retry imediato |

## Ciclo de vida do objeto em `/cdn/`

Dois relógios independentes: o tiering interno do Intelligent-Tiering (por
**último acesso**, reseta a cada GET) e a expiração do lifecycle (por **idade**,
não reseta).

```mermaid
stateDiagram-v2
    [*] --> Frequent: PutObject (lifecycle days=0)
    Frequent --> Infrequent: 30 dias sem acesso
    Infrequent --> ArchiveInstant: 90 dias sem acesso
    Infrequent --> Frequent: acesso (GET)
    ArchiveInstant --> Frequent: acesso (GET)
    Frequent --> [*]: 365 dias de idade (DELETE)
    Infrequent --> [*]: 365 dias de idade (DELETE)
    ArchiveInstant --> [*]: 365 dias de idade (DELETE)

    note right of Frequent
        ~$0.023/GB (= Standard)
    end note
    note right of Infrequent
        ~$0.0125/GB (-46%)
        acesso instantâneo
    end note
    note right of ArchiveInstant
        ~$0.004/GB (-68%)
        acesso instantâneo
    end note
```

- Os 3 tiers são **instant access** (sem `restore`, sem retrieval fee). Os tiers
  assíncronos (Archive / Deep Archive) **não** são habilitados — exigiriam
  restore, incompatível com CDN.
- Thresholds de 30/90 dias são **fixos da AWS** (não configuráveis). O DELETE de
  365 dias é nosso (`s3_cache_cleanup_days`).
- O DELETE é por idade, não por frieza: como o conteúdo é imutável, deletar e
  forçar redownload pontual é seguro.

Detalhes de risco e custo em [risk-mitigation-plan.md](risk-mitigation-plan.md).
