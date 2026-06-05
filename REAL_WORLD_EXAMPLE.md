# Real World Example

## URL: https://media.brunojet.com.br/images/cyril-mzn-WSvth_lwCi0-unsplash.jpg

### How It Works

**Step 1: Client Request**
```
GET https://media.brunojet.com.br/images/cyril-mzn-WSvth_lwCi0-unsplash.jpg
```

**Step 2: CloudFront Processing**
- CloudFront origin has `origin_path = /cdn`
- CloudFront adds prefix: `/cdn` + `/images/cyril-mzn-WSvth_lwCi0-unsplash.jpg`
- CloudFront requests S3: `/cdn/images/cyril-mzn-WSvth_lwCi0-unsplash.jpg`

**Step 3: First Request (Cache Miss)**
```
S3 Key: /cdn/images/cyril-mzn-WSvth_lwCi0-unsplash.jpg
Status: 404 Not Found
```

**Step 4: Lambda Fallback Triggered**
- Lambda receives original path: `/images/cyril-mzn-WSvth_lwCi0-unsplash.jpg`
- Lambda fetches from origin: `S3: /images/cyril-mzn-WSvth_lwCi0-unsplash.jpg` ✓

**Step 5: Stream to Cache**
```
Origin (root):  /images/cyril-mzn-WSvth_lwCi0-unsplash.jpg
                         ↓ (stream)
Cache (/cdn):  /cdn/images/cyril-mzn-WSvth_lwCi0-unsplash.jpg
```

**Step 6: Second Request (Cache Hit)**
```
CloudFront requests: S3: /cdn/images/cyril-mzn-WSvth_lwCi0-unsplash.jpg
Status: 200 OK (from cache)
```

### Testing Locally

**1. Setup test file at origin (root)**
```bash
# Start LocalStack
docker-compose up -d

# Run setup
bash scripts/setup-local.sh

# Verify structure:
# test-bucket/
#   images/
#     cyril-mzn-WSvth_lwCi0-unsplash.jpg  ← origin (root)
```

**2. Test with fallback CLI**
```bash
# Test with real path
go build -o fallback ./cmd/fallback

./fallback \
  -bucket test-bucket \
  -endpoint http://localhost:4566 \
  -path /images/cyril-mzn-WSvth_lwCi0-unsplash.jpg \
  -v
```

**3. Verify flow**

First run (cache miss):
```
=== Lambda Fallback Simulation ===
Path: /images/cyril-mzn-WSvth_lwCi0-unsplash.jpg

✗ Cache MISS: Not found in cdn/images/cyril-mzn-WSvth_lwCi0-unsplash.jpg
Step 2: Fetching from origin: images/cyril-mzn-WSvth_lwCi0-unsplash.jpg
✓ Origin FOUND: images/cyril-mzn-WSvth_lwCi0-unsplash.jpg
Step 3: Caching to: cdn/images/cyril-mzn-WSvth_lwCi0-unsplash.jpg
✓ Cached successfully
```

Second run (cache hit):
```
=== Lambda Fallback Simulation ===
Path: /images/cyril-mzn-WSvth_lwCi0-unsplash.jpg

✓ Cache HIT: Found cdn/images/cyril-mzn-WSvth_lwCi0-unsplash.jpg
```

### S3 Bucket Structure

```
brunojet-media-proxy-dev/
├── images/
│   ├── cyril-mzn-WSvth_lwCi0-unsplash.jpg  ← Origin (root)
│   └── (other images)
├── cdn/
│   └── images/
│       ├── cyril-mzn-WSvth_lwCi0-unsplash.jpg  ← Cache (after first request)
│       └── (other cached images)
└── (S3 Lifecycle expires /cdn/* after 90 days)
```

### Path Transformation

| Stage | S3 Key |
|-------|--------|
| User URL path | `/images/cyril-mzn-WSvth_lwCi0-unsplash.jpg` |
| CloudFront → S3 (with origin_path=/cdn) | `/cdn/images/cyril-mzn-WSvth_lwCi0-unsplash.jpg` |
| Lambda receives | `/images/cyril-mzn-WSvth_lwCi0-unsplash.jpg` |
| Lambda fetches origin | `images/cyril-mzn-WSvth_lwCi0-unsplash.jpg` (no leading /) |
| Lambda caches | `cdn/images/cyril-mzn-WSvth_lwCi0-unsplash.jpg` (no leading /) |

### Testing with Real AWS S3

```bash
# Prepare real bucket with origin files
aws s3 cp image.jpg s3://brunojet-media-proxy-dev/images/cyril-mzn-WSvth_lwCi0-unsplash.jpg

# Test locally (without Lambda, just fallback simulation)
./fallback \
  -bucket brunojet-media-proxy-dev \
  -region us-east-1 \
  -path /images/cyril-mzn-WSvth_lwCi0-unsplash.jpg \
  -v

# Verify cache was created
aws s3 ls s3://brunojet-media-proxy-dev/cdn/images/
# Should show: cyril-mzn-WSvth_lwCi0-unsplash.jpg
```

### CloudFront Signed URL (if enabled)

Using `cmd/sign-url` to generate signed URLs:

```bash
./sign-url \
  -domain media.brunojet.com.br \
  -path /images/cyril-mzn-WSvth_lwCi0-unsplash.jpg \
  -expires 3600

# Output:
# https://media.brunojet.com.br/cdn/images/cyril-mzn-WSvth_lwCi0-unsplash.jpg?Expires=...&Signature=...&Key-Pair-Id=...
```

Note: The signed URL automatically includes `/cdn` prefix (see `cmd/sign-url/main.go` line 86)

### Caching Headers

When Lambda returns the cached object:
```
Status: 200 OK
Content-Type: image/jpeg
Cache-Control: (CloudFront default: 86400s = 24h)
X-Cache: Hit from cloudfront  ← on second request
```

### Lifecycle Cleanup

After 90 days, S3 lifecycle rule removes:
- `cdn/images/cyril-mzn-WSvth_lwCi0-unsplash.jpg`
- All other files in `/cdn/*` prefix

Next request will trigger fallback again:
1. CloudFront 404 on `/cdn/images/...`
2. Lambda fetches fresh from origin
3. Cache renewed in `/cdn/images/...`

### Production Flow

```
User → HTTPS (media.brunojet.com.br)
         ↓
     CloudFront (distribution)
         ├─ Cache first: /cdn/* (origin_path=/cdn)
         ├─ Miss? → Lambda fallback (origin group)
         └─ Return to user
         
Lambda → S3 GetObject(/images/...)
      ↓
      S3 PutObject(/cdn/images/...)
      ↓
      Return to CloudFront
      
CloudFront → User
Next 24h: Cache hit, no Lambda invocation
```
