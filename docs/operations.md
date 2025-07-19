# Operations & Deployment Guide - CAMT-CSV Project

## Overview

This document provides comprehensive guidance for building, deploying, monitoring, and maintaining the CAMT-CSV application in production environments.

## Build Process

### 1. Local Development Build

```bash
# Clean build
go clean -cache
go mod tidy
go mod verify

# Build for current platform
go build -o bin/camt-csv cmd/camt-csv/main.go

# Build with version information
VERSION=$(git describe --tags --always --dirty)
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT=$(git rev-parse HEAD)

go build -ldflags "\
  -X main.Version=${VERSION} \
  -X main.BuildTime=${BUILD_TIME} \
  -X main.Commit=${COMMIT}" \
  -o bin/camt-csv cmd/camt-csv/main.go
```

### 2. Cross-Platform Builds

```bash
# Build for multiple platforms
PLATFORMS="darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64"

for platform in $PLATFORMS; do
    GOOS=${platform%/*}
    GOARCH=${platform#*/}
    
    echo "Building for $GOOS/$GOARCH..."
    
    output="bin/camt-csv-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        output="${output}.exe"
    fi
    
    GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags "-X main.Version=${VERSION}" \
        -o $output cmd/camt-csv/main.go
done
```

### 3. Docker Build

```dockerfile
# Dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-w -s" \
    -o camt-csv cmd/camt-csv/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates poppler-utils
WORKDIR /root/

COPY --from=builder /app/camt-csv .
COPY --from=builder /app/database ./database

ENTRYPOINT ["./camt-csv"]
```

```bash
# Build Docker image
docker build -t camt-csv:latest .

# Multi-platform build
docker buildx build --platform linux/amd64,linux/arm64 -t camt-csv:latest .
```

## Release Process

### 1. Version Management

**Semantic Versioning**: Follow semver (MAJOR.MINOR.PATCH)
- **MAJOR**: Breaking changes to CLI interface or file formats
- **MINOR**: New features, new parser support
- **PATCH**: Bug fixes, performance improvements

### 2. Release Workflow

```bash
# 1. Prepare release
git checkout main
git pull origin main

# 2. Update version
VERSION="v1.2.3"
echo $VERSION > VERSION

# 3. Update CHANGELOG.md
# Add release notes, breaking changes, new features

# 4. Commit and tag
git add VERSION CHANGELOG.md
git commit -m "Release $VERSION"
git tag -a $VERSION -m "Release $VERSION"

# 5. Push
git push origin main
git push origin $VERSION
```

### 3. Automated Release (GitHub Actions)

```yaml
# .github/workflows/release.yml
name: Release
on:
  push:
    tags: ['v*']

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - uses: actions/setup-go@v3
      with:
        go-version: '1.22'
    
    - name: Build releases
      run: |
        make build-all
    
    - name: Create Release
      uses: softprops/action-gh-release@v1
      with:
        files: |
          bin/camt-csv-*
          checksums.txt
        generate_release_notes: true
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Deployment Strategies

### 1. Standalone Binary Deployment

**Advantages**: Simple, no dependencies, fast startup
**Use Cases**: Personal use, CI/CD pipelines, development

```bash
# Download and install
curl -L https://github.com/user/camt-csv/releases/latest/download/camt-csv-linux-amd64 \
  -o /usr/local/bin/camt-csv
chmod +x /usr/local/bin/camt-csv

# Verify installation
camt-csv --version
```

### 2. Container Deployment

**Advantages**: Consistent environment, easy scaling, isolation
**Use Cases**: Cloud environments, microservices, batch processing

```bash
# Run with Docker
docker run --rm \
  -v $(pwd)/data:/data \
  -e GEMINI_API_KEY=$GEMINI_API_KEY \
  camt-csv:latest convert --input /data/input.xml --output /data/output.csv

# Docker Compose for batch processing
version: '3.8'
services:
  camt-csv:
    image: camt-csv:latest
    volumes:
      - ./input:/input
      - ./output:/output
    environment:
      - GEMINI_API_KEY=${GEMINI_API_KEY}
      - LOG_LEVEL=info
    command: convert --input /input/statements.xml --output /output/transactions.csv
```

### 3. Kubernetes Deployment

**Use Cases**: Large-scale processing, enterprise environments

```yaml
# k8s-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: camt-csv-processor
spec:
  replicas: 3
  selector:
    matchLabels:
      app: camt-csv
  template:
    metadata:
      labels:
        app: camt-csv
    spec:
      containers:
      - name: camt-csv
        image: camt-csv:latest
        env:
        - name: GEMINI_API_KEY
          valueFrom:
            secretKeyRef:
              name: ai-credentials
              key: gemini-api-key
        - name: LOG_LEVEL
          value: "info"
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

## Configuration Management

### 1. Environment-Specific Configuration

```yaml
# config/production.yaml
log:
  level: "warn"
  format: "json"

ai:
  enabled: true
  requests_per_minute: 60
  
csv:
  delimiter: ","

# config/development.yaml  
log:
  level: "debug"
  format: "text"

ai:
  enabled: false
```

### 2. Secret Management

**Environment Variables**:
```bash
# Production secrets
export GEMINI_API_KEY="$(cat /etc/secrets/gemini-api-key)"
export LOG_LEVEL="warn"
```

**Kubernetes Secrets**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: ai-credentials
type: Opaque
data:
  gemini-api-key: <base64-encoded-key>
```

**Docker Secrets**:
```bash
# Create secret
echo "your-api-key" | docker secret create gemini_api_key -

# Use in service
docker service create \
  --secret gemini_api_key \
  --env GEMINI_API_KEY_FILE=/run/secrets/gemini_api_key \
  camt-csv:latest
```

## Monitoring & Observability

### 1. Logging Strategy

**Structured Logging**:
```go
log.WithFields(logrus.Fields{
    "file":         filePath,
    "parser":       "camt",
    "transactions": len(transactions),
    "duration":     time.Since(start),
}).Info("File processing completed")
```

**Log Aggregation**:
```yaml
# docker-compose.yml with ELK stack
version: '3.8'
services:
  camt-csv:
    image: camt-csv:latest
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
    environment:
      - LOG_FORMAT=json
      
  filebeat:
    image: elastic/filebeat:7.15.0
    volumes:
      - /var/lib/docker/containers:/var/lib/docker/containers:ro
      - ./filebeat.yml:/usr/share/filebeat/filebeat.yml
```

### 2. Metrics Collection

**Application Metrics**:
```go
// Add to main application
import "github.com/prometheus/client_golang/prometheus"

var (
    filesProcessed = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "camt_files_processed_total",
            Help: "Total number of files processed",
        },
        []string{"parser", "status"},
    )
    
    processingDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "camt_processing_duration_seconds",
            Help: "Time spent processing files",
        },
        []string{"parser"},
    )
)
```

**System Metrics**:
```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'camt-csv'
    static_configs:
      - targets: ['localhost:8080']
```

### 3. Health Checks

```go
// Health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
    health := map[string]interface{}{
        "status":    "healthy",
        "timestamp": time.Now(),
        "version":   Version,
        "uptime":    time.Since(startTime),
    }
    
    // Check dependencies
    if aiEnabled {
        if err := checkAIService(); err != nil {
            health["status"] = "degraded"
            health["ai_service"] = "unavailable"
        }
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(health)
}
```

## Performance Optimization

### 1. Resource Management

**Memory Optimization**:
```go
// Stream processing for large files
func processLargeFile(filePath string) error {
    file, err := os.Open(filePath)
    if err != nil {
        return err
    }
    defer file.Close()
    
    decoder := xml.NewDecoder(file)
    
    for {
        token, err := decoder.Token()
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }
        
        // Process tokens incrementally
        if se, ok := token.(xml.StartElement); ok {
            if se.Name.Local == "Ntry" {
                var entry Entry
                if err := decoder.DecodeElement(&entry, &se); err != nil {
                    return err
                }
                processEntry(entry)
            }
        }
    }
    
    return nil
}
```

**CPU Optimization**:
```go
// Parallel processing
func processTransactions(transactions []models.Transaction) []models.Transaction {
    numWorkers := runtime.NumCPU()
    jobs := make(chan models.Transaction, len(transactions))
    results := make(chan models.Transaction, len(transactions))
    
    // Start workers
    for w := 0; w < numWorkers; w++ {
        go worker(jobs, results)
    }
    
    // Send jobs
    for _, tx := range transactions {
        jobs <- tx
    }
    close(jobs)
    
    // Collect results
    processed := make([]models.Transaction, 0, len(transactions))
    for i := 0; i < len(transactions); i++ {
        processed = append(processed, <-results)
    }
    
    return processed
}
```

### 2. Caching Strategy

```go
// Category cache
type CategoryCache struct {
    cache map[string]*models.Category
    mutex sync.RWMutex
    ttl   time.Duration
}

func (c *CategoryCache) Get(key string) (*models.Category, bool) {
    c.mutex.RLock()
    defer c.mutex.RUnlock()
    
    category, exists := c.cache[key]
    return category, exists
}
```

## Troubleshooting

### 1. Common Issues

**File Processing Errors**:
```bash
# Check file format
camt-csv validate --input suspicious_file.xml

# Debug with verbose logging
CAMT_LOG_LEVEL=debug camt-csv convert --input file.xml --output file.csv

# Test with minimal example
camt-csv convert --input samples/minimal.xml --output test.csv
```

**Memory Issues**:
```bash
# Monitor memory usage
top -p $(pgrep camt-csv)

# Use streaming for large files
camt-csv convert --streaming --input large_file.xml --output output.csv
```

**AI Service Issues**:
```bash
# Test AI connectivity
curl -H "Authorization: Bearer $GEMINI_API_KEY" \
  https://generativelanguage.googleapis.com/v1/models

# Disable AI fallback
CAMT_AI_ENABLED=false camt-csv convert --input file.xml --output file.csv
```

### 2. Diagnostic Commands

```bash
# System information
camt-csv version --verbose

# Configuration dump
camt-csv config --show

# Performance profiling
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
```

### 3. Log Analysis

**Error Patterns**:
```bash
# Find parsing errors
grep "parse error" /var/log/camt-csv.log | tail -20

# Check AI API issues
grep "ai service" /var/log/camt-csv.log | grep ERROR

# Performance analysis
grep "processing_duration" /var/log/camt-csv.log | \
  awk '{print $NF}' | sort -n | tail -10
```

## Backup & Recovery

### 1. Data Backup

**Configuration Backup**:
```bash
# Backup user configuration
tar -czf camt-csv-config-$(date +%Y%m%d).tar.gz \
  ~/.camt-csv/ \
  ~/.env
```

**Database Backup**:
```bash
# Backup category mappings
cp -r database/ backup/database-$(date +%Y%m%d)/
```

### 2. Disaster Recovery

**Recovery Procedures**:
```bash
# Restore from backup
tar -xzf camt-csv-config-20241219.tar.gz -C ~/

# Verify configuration
camt-csv config --validate

# Test functionality
camt-csv convert --input samples/test.xml --output test.csv --dry-run
```

## Security Considerations

### 1. Secure Deployment

**File Permissions**:
```bash
# Secure configuration files
chmod 600 ~/.camt-csv/config.yaml
chmod 600 ~/.env

# Secure binary
chmod 755 /usr/local/bin/camt-csv
```

**Network Security**:
```yaml
# Kubernetes NetworkPolicy
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: camt-csv-policy
spec:
  podSelector:
    matchLabels:
      app: camt-csv
  policyTypes:
  - Egress
  egress:
  - to: []
    ports:
    - protocol: TCP
      port: 443  # HTTPS only
```

### 2. Security Monitoring

**Audit Logging**:
```go
log.WithFields(logrus.Fields{
    "user":      os.Getenv("USER"),
    "file":      filePath,
    "operation": "convert",
    "timestamp": time.Now(),
}).Info("File processing initiated")
```

**Vulnerability Scanning**:
```bash
# Scan dependencies
go list -json -m all | nancy sleuth

# Container scanning
docker scan camt-csv:latest
```

This operations guide provides comprehensive coverage for deploying, monitoring, and maintaining the CAMT-CSV application in production environments while ensuring security, performance, and reliability.
