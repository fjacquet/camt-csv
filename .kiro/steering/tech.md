# Technical Stack

## Language & Runtime

- **Go**: Version 1.24.2
- **Build System**: Go modules with standard `go build`

## Core Dependencies

### CLI & Configuration

- `spf13/cobra`: Command-line interface framework
- `spf13/viper`: Hierarchical configuration management
- `joho/godotenv`: Environment variable loading from .env files

### Data Processing

- `gocarina/gocsv`: CSV parsing and generation
- `shopspring/decimal`: Precise decimal arithmetic for financial calculations
- `golang.org/x/net`: XML parsing utilities
- `gopkg.in/xmlpath.v2`: XPath queries for XML documents
- `gopkg.in/yaml.v3`: YAML configuration file parsing

### Logging & Testing

- `sirupsen/logrus`: Structured logging
- `stretchr/testify`: Testing assertions and mocks

### External Services

- Google Gemini API: Optional AI-powered transaction categorization

## External Tools

- **poppler-utils** (`pdftotext`): Required for PDF parsing functionality
  - macOS: `brew install poppler`
  - Ubuntu: `apt-get install poppler-utils`

## Common Commands

### Development

```bash
# Install dependencies
go mod download
go mod tidy

# Run tests
go test ./...
go test ./... -v                    # Verbose output
go test ./... -cover                # With coverage
go test -coverprofile=coverage.out ./...

# Run linters
golangci-lint run

# Format code
go fmt ./...
goimports -w .
```

### Building

```bash
# Development build
go build -o camt-csv

# Production build (optimized)
go build -ldflags="-s -w" -o camt-csv

# Build with version info
VERSION=$(git describe --tags --always)
go build -ldflags "-X main.Version=${VERSION}" -o camt-csv

# Cross-platform builds
GOOS=linux GOARCH=amd64 go build -o camt-csv-linux-amd64
GOOS=darwin GOARCH=arm64 go build -o camt-csv-darwin-arm64
GOOS=windows GOARCH=amd64 go build -o camt-csv-windows-amd64.exe
```

### Running

```bash
# Basic conversion
./camt-csv camt -i input.xml -o output.csv
./camt-csv pdf -i statement.pdf -o output.csv
./camt-csv revolut -i export.csv -o output.csv

# Batch processing
./camt-csv batch -i input_dir/ -o output_dir/

# With configuration
./camt-csv --log-level debug camt -i file.xml -o output.csv

# Using environment variables
export GEMINI_API_KEY=your_key
export LOG_LEVEL=debug
./camt-csv camt -i file.xml -o output.csv
```

### Configuration

```bash
# Configuration file locations (in order of precedence)
# 1. CLI flags (highest priority)
# 2. Environment variables
# 3. Config file: ~/.camt-csv/config.yaml
# 4. Config file: ./.camt-csv/config.yaml
# 5. Default values (lowest priority)
```

## Project Structure Conventions

- Use Go modules exclusively for dependency management
- Pin dependency versions in go.mod for reproducible builds
- Follow semantic versioning (MAJOR.MINOR.PATCH)
- All code must pass `golangci-lint` checks before commit
