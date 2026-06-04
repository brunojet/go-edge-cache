# Quick Start - Local Debug

## 1. Start LocalStack

```bash
docker-compose up -d localstack
```

## 2. Setup Local Environment

```bash
bash scripts/setup-local.sh
```

This will:
- Create `test-bucket`
- Upload test files (root and nested)
- Configure AWS env vars for LocalStack

## 3. Build Fallback CLI

```bash
go build -o fallback ./cmd/fallback
```

## 4. Test Cache Miss (First Run)

```bash
./fallback \
  -bucket test-bucket \
  -endpoint http://localhost:4566 \
  -path /test-file-1.txt \
  -v
```

**Output:**
```
=== Lambda Fallback Simulation ===
Bucket: test-bucket
Region: us-east-1
Endpoint: http://localhost:4566
Path: /test-file-1.txt

✗ Cache MISS: Not found in cdn/test-file-1.txt

Step 2: Fetching from origin: test-file-1.txt
✓ Origin FOUND: test-file-1.txt
  Content-Type: text/plain
  Size: 20 bytes

Step 3: Caching to: cdn/test-file-1.txt
✓ Cached successfully

=== Next request will hit cache ===
```

## 5. Test Cache Hit (Second Run)

```bash
./fallback \
  -bucket test-bucket \
  -endpoint http://localhost:4566 \
  -path /test-file-1.txt \
  -v
```

**Output:**
```
=== Lambda Fallback Simulation ===
Bucket: test-bucket
Region: us-east-1
Endpoint: http://localhost:4566
Path: /test-file-1.txt

✓ Cache HIT: Found cdn/test-file-1.txt
  Content-Type: text/plain
  Size: 20 bytes
```

## 6. Verify S3 Contents

```bash
export AWS_ENDPOINT_URL_S3=http://localhost:4566
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test

# List all files
aws s3 ls s3://test-bucket/ --recursive

# Output should show:
# 2024-01-15 10:30:45   20 cdn/test-file-1.txt  (cached)
# 2024-01-15 10:30:30   20 test-file-1.txt      (origin)
```

## 7. Test with Custom Path

```bash
./fallback \
  -bucket test-bucket \
  -endpoint http://localhost:4566 \
  -path /images/nested-test.txt \
  -v
```

## Architecture Overview

```
Your Local Machine
├── LocalStack (Docker)
│   └── S3 Bucket: test-bucket/
│       ├── test-file-1.txt         ← origin (root)
│       ├── test-file-2.txt         ← origin (root)
│       ├── images/nested-test.txt  ← origin (nested)
│       └── cdn/                    ← cache (populated by fallback)
│           ├── test-file-1.txt     (after first run)
│           └── images/nested-test.txt (after first run)
│
└── Fallback CLI (./fallback)
    └── Uses go-infra-adapters
        └── Simulates Lambda handler locally
```

## What's Happening

1. **Request**: `GET /test-file-1.txt`
2. **CloudFront Logic**: Try S3 `/cdn/test-file-1.txt`
3. **Cache Miss**: Not found → Lambda fallback
4. **Lambda Fallback**:
   - Fetch: S3 `/test-file-1.txt` (origin/root)
   - Stream to: S3 `/cdn/test-file-1.txt` (cache)
5. **Next Request**: Cache hit on `/cdn/test-file-1.txt`

## Debugging Tips

- Use `-v` flag for verbose logging
- Monitor LocalStack logs:
  ```bash
  docker logs -f go-edge-cache-localstack
  ```
- Check bucket state:
  ```bash
  aws s3 ls s3://test-bucket/ --recursive
  ```
- Test different paths in the same bucket

## Using Real AWS S3

To test against real AWS instead of LocalStack:

```bash
# Set your AWS credentials
export AWS_ACCESS_KEY_ID=your-key
export AWS_SECRET_ACCESS_KEY=your-secret
export AWS_DEFAULT_REGION=us-east-1

# Run without -endpoint flag (uses real AWS)
./fallback -bucket your-bucket -path /images/photo.jpg -v
```

## Cleanup

```bash
# Stop LocalStack
docker-compose down

# Remove test bucket (if using LocalStack)
docker-compose down -v
```
