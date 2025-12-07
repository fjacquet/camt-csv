.PHONY: build test lint security clean coverage help install-tools

# Build variables
BINARY_NAME=camt-csv
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Go variables
GOTEST=go test
GOCOVER=-coverprofile=coverage.txt -covermode=atomic
GORACE=-race

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

## build: Build the application
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

## build-prod: Build optimized production binary
build-prod:
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINARY_NAME) .

## test: Run all tests
test:
	$(GOTEST) -v ./...

## test-race: Run tests with race detector
test-race:
	$(GOTEST) -v $(GORACE) ./...

## coverage: Run tests with coverage report
coverage:
	$(GOTEST) -v $(GOCOVER) ./...
	go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report: coverage.html"

## coverage-summary: Show coverage summary per package
coverage-summary:
	$(GOTEST) -cover ./... | grep -v "no test files"

## lint: Run golangci-lint
lint:
	golangci-lint run --timeout=5m

## security: Run security scan with gosec
security:
	gosec -exclude=G304 -fmt=sarif -out=security.sarif ./...
	@echo "Security report: security.sarif"

## clean: Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.txt coverage.html
	rm -f security.sarif
	rm -f debug_pdf_extract.txt

## install-tools: Install development tools
install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest

## mod-tidy: Tidy go modules
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

## all: Run lint, test, and build
all: lint test build
