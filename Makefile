# Canonical Go Makefile — fjacquet/ci standard interface (do not rename targets)
.DEFAULT_GOAL := all
DIST  ?= dist
COVER ?= coverage.out
GOLANGCI_VERSION ?= v2.8.0
GORELEASER_VERSION ?= v2.7.0
GOVULNCHECK_VERSION ?= v1.1.4

# Build variables
BINARY_NAME=camt-csv
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(BUILD_TIME)"

.PHONY: all clean install tools lint format test build vuln sbom security docs coverage-upload release ci \
        build-prod test-race coverage coverage-summary mod-tidy mod-update run help

## all: Run lint, test, and build
all: clean lint test build

## clean: Clean build artifacts
clean:
	rm -rf $(DIST) site $(COVER) *.sarif
	rm -f $(BINARY_NAME) coverage.txt coverage.html debug_pdf_extract.txt sbom.cdx.json

## install: Download Go modules
install:
	go mod download

## tools: Install development tools (including goreleaser)
tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)
	go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
	go install github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)

## lint: Run golangci-lint
lint:
	golangci-lint run --timeout=5m

## format: Run golangci-lint formatter
format:
	golangci-lint fmt

## test: Run tests with race detector and coverage
test:
	go test -race -coverprofile=$(COVER) -covermode=atomic ./...

## build: Build the application
build:
	go build $(LDFLAGS) -v -o $(BINARY_NAME) .

## vuln: Run govulncheck
vuln:
	go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

## sbom: Generate Software Bill of Materials (CycloneDX)
sbom:
	mkdir -p $(DIST)
	go run github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest mod -json -output $(DIST)/sbom.cdx.json

## security: Run Semgrep SAST (canonical Go security scanner)
security:
	uvx semgrep scan --config auto --error --skip-unknown-extensions

## docs: Build MkDocs documentation
docs:
	uvx --with mkdocs-material --with pymdown-extensions mkdocs build --strict --site-dir site

## coverage-upload: Upload coverage to Codecov
coverage-upload:
	uvx --from codecov-cli codecov upload-process --file $(COVER) || true

## release: Cut a release with goreleaser
release:
	goreleaser release --clean

## ci: Lint, test, build, and vuln check (matches reusable workflow)
ci: lint test build vuln

# ---------------------------------------------------------------------------
# Convenience targets (preserved from original Makefile)
# ---------------------------------------------------------------------------

## build-prod: Build optimised production binary (CGO disabled)
build-prod:
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINARY_NAME) .

## test-race: Run tests with race detector (verbose)
test-race:
	go test -v -race ./...

## coverage: Run tests and generate HTML coverage report
coverage:
	go test -v -coverprofile=coverage.txt -covermode=atomic ./...
	go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report: coverage.html"

## coverage-summary: Show coverage summary per package
coverage-summary:
	go test -cover ./... | grep -v "no test files"

## mod-tidy: Tidy Go modules
mod-tidy:
	go mod tidy
	go mod verify

## mod-update: Update all dependencies
mod-update:
	go get -u ./...
	go mod tidy

## run: Build and run the application
run: build
	./$(BINARY_NAME)

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
