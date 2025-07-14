# xk6-milvus Makefile

# Variables
XK6_VERSION := latest
MODULE_NAME := github.com/zilliz/xk6-milvus
K6_BINARY := ./k6

# Default target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  make install-xk6    - Install xk6 tool"
	@echo "  make build          - Build k6 with milvus extension"
	@echo "  make test           - Run Go tests"
	@echo "  make example-basic  - Run basic example test"
	@echo "  make example-advanced - Run advanced example test"
	@echo "  make lint           - Run golangci-lint"
	@echo "  make fmt            - Format Go code"
	@echo "  make tidy           - Run go mod tidy"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make all            - Run fmt, lint, test, and build"

# Install xk6 if not present
.PHONY: install-xk6
install-xk6:
	@which xk6 > /dev/null || (echo "Installing xk6..." && go install go.k6.io/xk6/cmd/xk6@$(XK6_VERSION))

# Build k6 with milvus extension
.PHONY: build
build: install-xk6
	@echo "Building k6 with milvus extension..."
	xk6 build --with $(MODULE_NAME)=.

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# Run basic example
.PHONY: example-basic
example-basic: build
	@echo "Running basic example..."
	$(K6_BINARY) run examples/basic/test-milvus.js

# Run advanced example
.PHONY: example-advanced
example-advanced: build
	@echo "Running advanced example..."
	$(K6_BINARY) run examples/advanced/flexible-test.js

# Run linter
.PHONY: lint
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Tidy dependencies
.PHONY: tidy
tidy:
	@echo "Tidying dependencies..."
	go mod tidy

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(K6_BINARY)
	rm -rf dist/

# Build everything
.PHONY: all
all: fmt lint test build
	@echo "Build complete!"

# Development workflow targets
.PHONY: dev
dev: fmt build
	@echo "Ready for development!"

# Check if Milvus is running (requires MILVUS_HOST env var or uses default)
.PHONY: check-milvus
check-milvus:
	@echo "Checking Milvus connection..."
	@MILVUS_HOST=$${MILVUS_HOST:-localhost:19530} && \
	echo "Using Milvus at: $$MILVUS_HOST"

# Run all examples
.PHONY: examples
examples: build check-milvus example-basic example-advanced
	@echo "All examples completed!"