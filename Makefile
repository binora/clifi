.PHONY: build test lint clean install fmt check install-hooks

# Build variables
BINARY_NAME=clifi
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Default target
all: build

# Build the binary
build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/clifi

# Build for multiple platforms
build-all:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/clifi
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/clifi
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/clifi
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 ./cmd/clifi

# Install to $GOPATH/bin
install:
	go install $(LDFLAGS) ./cmd/clifi

# Run tests
test:
	go test -v -race -cover ./...

# Run tests with coverage report
test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-integration:
	OPENROUTER_API_KEY=$$OPENROUTER_API_KEY go test -tags=integration -v ./internal/llm -run OpenRouter

# Run linter (auto-installs golangci-lint if missing)
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

# Format code
fmt:
	go fmt ./...
	@which goimports > /dev/null && goimports -w . || true

# Tidy dependencies
tidy:
	go mod tidy

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Run the application
run:
	go run ./cmd/clifi

# Development mode with hot reload (requires air)
dev:
	air

# Generate mocks (requires mockgen)
generate:
	go generate ./...

# Run all checks (fmt, lint, test) - used by git pre-commit hook
check: fmt lint test

# Install git hooks
install-hooks:
	@echo "Installing git hooks..."
	@mkdir -p .git/hooks
	@cp scripts/pre-commit .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Git hooks installed successfully"
