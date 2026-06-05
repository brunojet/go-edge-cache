 # CloudFront Media Proxy — Guia de Implementação (Prompt base)

 Este arquivo é a versão formatada do prompt gerado e deve ser usado como base junto com o documento de arquitetura: [Arquitetura — CloudFront Media Proxy](arquitetura-cloudfront-media-proxy.docx.md).

 ## Objetivo 🎯

 - Fornecer um prompt claro para o Copilot gerar a infraestrutura e o código base de um CloudFront Media Proxy.
 - Produzir Terraform (S3, CloudFront, Lambda, API Gateway, Secrets Manager), uma Lambda em Go com streaming, scripts de deploy e uma estrutura de projeto organizada.

 ## Prompt principal (resumo)

 Preciso implementar um CloudFront Media Proxy conforme o documento de arquitetura. O resultado esperado inclui:

 - Terraform completo para AWS (S3, CloudFront, Lambda, API Gateway, Secrets Manager).
 - Lambda function em Go que funciona como fallback COM STREAMING.
 - Estrutura de projeto organizada e scripts de deploy.

 Requisitos de capacidade e configuração:

 - S3 privado com Origin Access Control (OAC).
 - CloudFront com Origin Group: S3 (primary) e Lambda Function URL (secondary).
 - Lambda Go com streaming response (suporte até 200 MB).
 - Function URL com streaming habilitado (InvokeMode: RESPONSE_STREAM).
 - Secrets Manager para API key externa.
 - IAM roles com permissões mínimas necessárias.
 - Timeout da Lambda: 900s (15 minutos).
 - Memory: 3008 MB.

 ## Prompts específicos por componente

 ### 1) Infraestrutura (Terraform)

 Gerar Terraform que crie e configure:

 - S3 bucket privado (Block Public Access, OAC configurado).
 - CloudFront distribution com Origin Group (S3 primary + Lambda Function URL secondary) e failover para erros 403/404/5xx.
 - Lambda function empacotada (runtime provided.al2023) com Function URL em modo de streaming.
 - Secrets Manager para armazenar API key externa.
 - IAM roles e policies com permissões mínimas (ex.: s3:GetObject, s3:PutObject, secretsmanager:GetSecretValue).
 - CloudWatch Log Groups para Lambda e API Gateway.

 Exemplo de política S3 (ajustar ARN):

 ```json
 {
   "Version": "2012-10-17",
   "Statement": [
     {
       "Effect": "Allow",
       "Principal": { "Service": "cloudfront.amazonaws.com" },
       "Action": "s3:GetObject",
       "Resource": "arn:aws:s3:::bucket-name/*",
       "Condition": {
         "StringEquals": { "AWS:SourceArn": "arn:aws:cloudfront::account:distribution/distribution-id" }
       }
     }
   ]
 }
 ```

 ### 2) Lambda Streaming em Go (requisitos e comportamento)

 A função Lambda deve:

 - Ser invocada via Function URL com streaming de resposta habilitado.
 - Receber o path do arquivo na requisição HTTP.
 - Recuperar a API key do Secrets Manager.
 - Fazer requisição à API externa e repassar o body em streaming para o cliente (sem carregar tudo em memória).
 - Paralelamente ao streaming de resposta, realizar upload multipart para o S3.
 - Fornecer headers corretos (Content-Type, Content-Length quando possível).
 - Tratar erros e timeouts (usar context com timeout apropriado).

 Exemplo de esqueleto de handler (Go):

 ```go
 package main

 import (
   "context"
   "io"
   "net/http"
   "time"

   "github.com/aws/aws-lambda-go/events"
   "github.com/aws/aws-lambda-go/lambda"
 )

 func handler(ctx context.Context, req events.LambdaFunctionURLRequest) (*events.LambdaFunctionURLStreamingResponse, error) {
   // 1. Extrair path da requisição (req.RawPath / req.Path)
   // 2. Recuperar secret do Secrets Manager
   // 3. Montar requisição HTTP para a API externa
   // 4. Encadear o response.Body para o streaming de saída e para um multipart upload para o S3
   // 5. Retornar eventos.LambdaFunctionURLStreamingResponse com streaming habilitado
   return &events.LambdaFunctionURLStreamingResponse{
     StatusCode:      200,
     Headers:         map[string]string{"Content-Type": "application/octet-stream"},
     IsBase64Encoded: false,
   }, nil
 }

 func main() {
   lambda.StartWithOptions(handler, lambda.WithEnableResponseStreaming())
 }
 ```

 Observação: o handler deve usar pipes (`io.Pipe`) e `io.TeeReader` para multiplexar o stream entre resposta e upload ao S3.

 ### 3) CloudFront (configuração)

 - Origin Group: S3 como primary; Lambda Function URL como secondary (failover).
 - Failover criteria: 403, 404, 500, 502, 503, 504.
 - Cache behaviors otimizados para mídia (cache key baseado em path, compressão habilitada quando aplicável).
 - Trusted Key Group para Signed URLs (se necessário).
 - Redirecionamento para HTTPS.

 ## Estrutura de projeto sugerida

 ```text
 cloudfront-media-proxy/
 ├── terraform/
 │   ├── main.tf
 │   ├── variables.tf
 │   ├── outputs.tf
 │   ├── cloudfront.tf
 │   ├── lambda.tf
 │   ├── s3.tf
 │   └── iam.tf
 ├── lambda/
 │   ├── main.go
 │   ├── go.mod
 │   ├── go.sum
 │   ├── handler/
 │   │   ├── streaming.go
 │   │   └── s3upload.go
 │   └── internal/
 │       ├── config/
 │       └── client/
 ├── scripts/
 │   ├── deploy.sh
 │   ├── build-lambda.sh
 │   └── generate-keypair.sh
 └── docs/
     └── arquitetura-cloudfront-media-proxy.docx.md
 ```

 ## Checklist de implementação (faseada)

 - Fase 1 — Infraestrutura base
   - S3 privado com OAC
   - Secrets Manager com API key
   - IAM roles e policies mínimas
   - CloudWatch Log Groups

 - Fase 2 — Lambda Function
   - Lambda com streaming habilitado
   - Function URL configurada
   - Código Go implementado e testado
   - Deploy automatizado (scripts)

 - Fase 3 — CloudFront
   - Distribution com Origin Group
   - OAC configurado para S3
   - Cache behaviors otimizados
   - Trusted Key Group para Signed URLs

 - Fase 4 — Testes
   - Teste cache hit (arquivo existente no S3)
   - Teste fallback (arquivo ausente → Lambda)
   - Testes com arquivos grandes (>50 MB)
   - Testes de Signed URLs (válidos / inválidos)
   - Teste de concorrência

 ## Monitoramento e logs

 - CloudWatch Metrics:
   - Lambda: Duration, Errors, Throttles, Streaming Duration
   - CloudFront: Cache Hit Ratio, Origin Latency, 4xx/5xx Errors
   - S3: NumberOfObjects, BucketSizeBytes

 - Logs importantes:
   - Lambda Logs: `/aws/lambda/media-proxy-function`
   - CloudFront Access Logs: bucket S3 dedicado
   - API Gateway / Function URL logs

 ## Otimizações de performance

 - Streaming: usar `io.TeeReader` / `io.Pipe` e multipart upload para S3 (>5 MB).
 - Reutilizar `http.Client` com pooling e timeouts.
 - TTL longo para mídia estática (quando apropriado).
 - Usar S3 Intelligent Tiering / Lifecycle para arquivos antigos.

 ## Segurança

 - Signed URLs: gerar no BFF (ex.: função em Go que assina URLs CloudFront).

 ```go
 func (s *MediaService) GetSignedURL(filePath string, ttl time.Duration) (string, error) {
   resourceURL := fmt.Sprintf("https://%s/%s", s.cloudfrontDomain, filePath)
   return s.signer.SignURL(resourceURL, time.Now().Add(ttl))
 }
 ```

 - Cabeçalhos de resposta recomendados: `Strict-Transport-Security`, `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`.

 ## Comandos de deploy rápidos

 ```bash
 # Build da Lambda
 cd lambda && GOOS=linux GOARCH=amd64 go build -o bootstrap main.go

 # Deploy da infraestrutura
 cd terraform && terraform init && terraform apply

 # Atualizar código da Lambda
 aws lambda update-function-code \
   --function-name media-proxy-function \
   --zip-file fileb://function.zip
 ```

 ## Observações e próximos passos

 - Use este arquivo como prompt base no Copilot e combine-o com o documento de arquitetura ([Arquitetura — CloudFront Media Proxy](arquitetura-cloudfront-media-proxy.docx.md)).
 - Revise a configuração de streaming da Lambda com atenção (events.LambdaFunctionURLStreamingResponse e lambda.WithEnableResponseStreaming()).
 - Faça testes iniciais com arquivos pequenos e, em seguida, valide o fluxo com arquivos maiores (até 200 MB).

 ---

 Arquivo gerado/formatado automaticamente a partir do prompt original; um backup foi salvo como `prompt.original.md` no mesmo diretório.
