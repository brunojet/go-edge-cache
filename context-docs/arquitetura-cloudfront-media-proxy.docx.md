  
**CloudFront Media Proxy**

Arquitetura de Distribuição de Mídia com Cache Inteligente

Documento de Arquitetura Técnica  •  v1.0  •  2026

# **1\. Visão Geral**

Este documento descreve a arquitetura de um sistema de distribuição de arquivos de mídia utilizando Amazon CloudFront como CDN, Amazon S3 como cache permanente e AWS Lambda como mecanismo de fallback para busca de arquivos em uma API externa.

O sistema funciona como um proxy espelho inteligente: qualquer path requisitado ao CloudFront é espelhado identicamente no S3 e na API externa de origem. Quando o arquivo não está em cache no S3, o Lambda o busca automaticamente, armazena e entrega ao usuário — de forma transparente.

| Objetivo principal: Servir arquivos de mídia com alta performance via CDN, reduzindo chamadas à API externa através de cache permanente no S3, com acesso controlado por CloudFront Signed URLs. |
| :---- |

# **2\. Componentes da Arquitetura**

| Componente | Serviço AWS | Responsabilidade |
| :---- | :---- | :---- |
| CDN / Ponto de entrada | Amazon CloudFront | Recebe requisições, valida Signed URL, serve do cache |
| Cache permanente | Amazon S3 (privado) | Armazena arquivos de mídia espelhando o path original |
| Fallback / Proxy | AWS Lambda (via CloudFront) | Busca arquivo na API externa e salva no S3 |
| Autenticação de acesso | CloudFront Signed URLs | Controla quem pode baixar cada arquivo e por quanto tempo |
| Acesso seguro ao S3 | CloudFront OAC | Permite que CloudFront acesse o S3 sem expô-lo publicamente |
| Segredos | AWS Secrets Manager | Armazena a API Key da origem externa com segurança |

# **3\. Fluxo de Requisição**

## **3.1 Arquivo em cache no S3 (caminho feliz)**

  Usuário

    │

    │  GET /produtos/categoria/imagem.jpg?Expires=...\&Signature=...\&Key-Pair-Id=...

    ▼

  CloudFront

    │  1\. Valida Signed URL (expiração \+ assinatura criptográfica)

    │  2\. Verifica cache local do CloudFront

    │  3\. Cache miss → consulta S3

    ▼

  S3 (privado)  ──→  arquivo existe  ──→  retorna 200 \+ binário

    │

    ▼

  CloudFront entrega ao usuário  ✅

  (guarda em cache para próximas requisições)

## **3.2 Arquivo ausente no S3 (fallback via Lambda)**

  Usuário

    │

    │  GET /produtos/categoria/imagem.jpg?\[Signed URL params\]

    ▼

  CloudFront

    │  1\. Valida Signed URL

    │  2\. Consulta S3  →  404 Not Found

    ▼

  Origin Group: aciona Origin Secundária (API Gateway \+ Lambda)

    │

    │  Lambda recebe o path: /produtos/categoria/imagem.jpg

    │  3\. Monta URL da API externa: https://api-origem.com/produtos/categoria/imagem.jpg

    │  4\. Faz GET com API Key no header (X-Api-Key: \*\*\*)

    │  5\. Recebe binário da imagem

    │  6\. Salva no S3: s3://bucket/produtos/categoria/imagem.jpg

    │  7\. Retorna o binário ao CloudFront

    ▼

  CloudFront entrega ao usuário  ✅

  (próxima requisição virá do S3 direto)

# **4\. Espelhamento de Paths**

O sistema adota a estratégia de proxy espelho: o path da URL requisitada é preservado integralmente em todos os componentes. Não há mapeamento, tradução ou transformação de paths.

| Componente | Exemplo de Path |
| :---- | :---- |
| URL pública (CloudFront) | https://cdn.empresa.com/produtos/eletronicos/foto-01.jpg |
| Objeto no S3 | s3://media-bucket/produtos/eletronicos/foto-01.jpg |
| Chamada à API externa | https://api-origem.com/produtos/eletronicos/foto-01.jpg |

Esta abordagem garante:

* Consistência total entre os três sistemas

* Debug simplificado — o path é o mesmo em todos os logs

* Sem necessidade de tabela de mapeamento ou banco de dados

* Suporte a qualquer estrutura de path sem alteração de código

# **5\. Controle de Acesso e Segurança**

## **5.1 CloudFront Signed URLs**

Todos os acessos ao CloudFront exigem uma Signed URL válida, gerada pela aplicação backend do cliente. A URL contém:

| Parâmetro | Descrição |
| :---- | :---- |
| Expires | Timestamp Unix de expiração do acesso |
| Signature | Assinatura RSA-SHA1 com a chave privada do Key Pair |
| Key-Pair-Id | Identificador da chave pública cadastrada no CloudFront |

Requisições sem Signed URL ou com URL expirada recebem HTTP 403 Forbidden.

## **5.2 S3 Completamente Privado**

* Block Public Access habilitado em todas as configurações

* Sem Bucket Policy pública

* Acesso permitido exclusivamente via CloudFront OAC (Origin Access Control)

* Usuários nunca acessam o S3 diretamente

## **5.3 API Key da Origem**

* Armazenada no AWS Secrets Manager — nunca em variáveis de ambiente ou código

* Lambda lê a secret em runtime via SDK

* Rotação de chave possível sem redeploy do Lambda

# **6\. Tratamento de Concorrência**

Quando múltiplos usuários requisitam o mesmo arquivo simultaneamente e ele ainda não está no S3, pode ocorrer que mais de um Lambda tente buscá-lo e salvá-lo ao mesmo tempo.

## **6.1 Por que não é um problema**

O arquivo é idempotente: dado o mesmo path, a API externa sempre retorna o mesmo conteúdo binário. Portanto:

  Lambda A  →  busca /img/foto.jpg  →  salva no S3  ✅

  Lambda B  →  busca /img/foto.jpg  →  salva no S3  ✅  (sobrescreve com conteúdo idêntico)

  Resultado final: arquivo correto no S3  ✅

  Nenhum dado corrompido, nenhuma inconsistência.

## **6.2 CloudFront Request Collapsing**

O próprio CloudFront mitiga o problema nativamente: quando centenas de usuários requisitam o mesmo arquivo ao mesmo tempo, o CloudFront encaminha apenas UMA requisição à origin (Lambda). As demais ficam em espera e recebem a mesma resposta.

| Decisão de arquitetura: Não será implementado mecanismo de lock (DynamoDB, Redis, arquivo .lock no S3). O custo de duplicar uma geração idempotente é mínimo e a complexidade de um lock distribuído não se justifica para este caso. |
| :---- |

# **7\. Decisões Técnicas e Alternativas Consideradas**

## **7.1 Origin Group vs Lambda@Edge**

| Critério | Origin Group (escolhido) | Lambda@Edge |
| :---- | :---- | :---- |
| Complexidade | Baixa — configuração nativa do CloudFront | Alta — código executado no edge |
| Custo | Lambda regional (mais barato) | Lambda@Edge (preço maior por invocação) |
| Latência do fallback | Maior (vai para região AWS) | Menor (edge global) |
| Limite de payload | Sem limite prático | 1 MB no corpo da resposta |
| Manutenção | Simples | Complexa (replicação global, versões) |
| Flexibilidade | Limitada ao que o CF suporta | Total (código arbitrário) |

Decisão: Origin Group foi escolhido pela menor complexidade operacional e custo, dado que os arquivos de mídia podem ultrapassar 1 MB, o que inviabilizaria Lambda@Edge.

## **7.2 Mecanismo de Cache no S3**

| Alternativa | Avaliação |
| :---- | :---- |
| Cache no S3 (permanente) | ✅ Escolhido — simples, barato, durável, sem TTL para gerenciar |
| ElastiCache / Redis | ❌ Custo alto, complexidade operacional, não adequado para binários grandes |
| Somente cache do CloudFront | ❌ Cache volátil, não persiste entre invalidações, custo de re-download |
| Banco de dados de metadados | ❌ Desnecessário — o path já é a chave única |

# **8\. Recursos de Infraestrutura**

| Recurso | Tipo | Configuração Principal |
| :---- | :---- | :---- |
| media-bucket | S3 Bucket | Privado, Block Public Access, versionamento opcional |
| media-distribution | CloudFront Distribution | Origin Group, Trusted Key Group, HTTPS only |
| media-oac | CloudFront OAC | Signing protocol: sigv4, Origin type: S3 |
| media-fallback-fn | Lambda Function | Runtime Node.js 20, timeout 30s, memória 512 MB |
| media-fallback-api | API Gateway (HTTP API) | Proxy integration com Lambda, rota ANY /{proxy+} |
| media-origin-secret | Secrets Manager Secret | API Key da origem, rotação manual |
| media-lambda-role | IAM Role | s3:PutObject no bucket \+ secretsmanager:GetSecretValue |

# **9\. Ordem de Implantação**

1. Criar o S3 Bucket com Block Public Access

2. Criar o Secret no Secrets Manager com a API Key da origem

3. Criar IAM Role do Lambda com permissões ao S3 e Secrets Manager

4. Fazer deploy da função Lambda com o código de fallback

5. Criar API Gateway HTTP API apontando para o Lambda

6. Criar CloudFront Distribution com Origin Group (S3 primário \+ API GW secundário)

7. Criar CloudFront OAC e aplicar Bucket Policy no S3

8. Gerar Key Pair para Signed URLs e cadastrar Trusted Key Group no CloudFront

9. Validar fluxo completo: arquivo existente, arquivo ausente, acesso não autorizado

# **10\. Premissas e Limitações**

## **Premissas**

* A API externa é idempotente: o mesmo path sempre retorna o mesmo binário

* A API externa é estável e disponível — não há mecanismo de retry complexo

* Os arquivos de mídia não têm prazo de expiração (cache permanente é adequado)

* O controle de quais usuários podem acessar quais arquivos é feito pela aplicação que gera as Signed URLs

## **Limitações conhecidas**

* Lambda timeout de 30s: arquivos muito grandes na API externa podem estourar o limite

* Primeiro acesso de cada arquivo sempre terá latência adicional (download da API \+ save no S3)

* Sem invalidação automática de cache: se um arquivo mudar na API externa, o S3 precisa ser atualizado manualmente

| Próximos passos: Com este documento aprovado, o próximo passo é a implementação via Terraform (IaC) e o código do Lambda em Node.js 20, cobrindo todos os cenários descritos nos fluxos acima. |
| :---- |

