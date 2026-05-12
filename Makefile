.PHONY: help build test clean install-xk6 examples lint fmt coverage mod-tidy docker-build docker-up docker-down docker-logs all

MILVUS_PKG_V3_VERSION := v3.0.0-20260512023210-c5ee59af8de5

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install-xk6: ## Install xk6 tool
	go install go.k6.io/xk6/cmd/xk6@latest

build: ## Build k6 with xk6-milvus extension
	$(shell go env GOPATH)/bin/xk6 build \
		--with github.com/mmga-lab/xk6-milvus=. \
		--replace github.com/milvus-io/milvus/pkg/v3=github.com/milvus-io/milvus/pkg/v3@$(MILVUS_PKG_V3_VERSION)

test: ## Run tests
	go test -v -race ./pkg/milvus

test-verbose: ## Run tests with verbose output
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./pkg/milvus

test-integration: ## Run integration tests (requires MILVUS_HOST)
	@if [ -z "$(MILVUS_HOST)" ]; then \
		echo "Warning: MILVUS_HOST not set, using default localhost:19530"; \
		echo "To set: export MILVUS_HOST=localhost:19530"; \
		export MILVUS_HOST=localhost:19530; \
	fi
	go test -tags=integration -v -race ./pkg/milvus

test-integration-coverage: ## Run integration tests with coverage
	@if [ -z "$(MILVUS_HOST)" ]; then \
		echo "Warning: MILVUS_HOST not set, using default localhost:19530"; \
		export MILVUS_HOST=localhost:19530; \
	fi
	go test -tags=integration -v -race -coverprofile=integration-coverage.txt -covermode=atomic ./pkg/milvus
	go tool cover -html=integration-coverage.txt -o integration-coverage.html

test-e2e: ## Run E2E tests (requires running Milvus)
	go test -tags e2e -v ./pkg/milvus

test-all: test test-integration ## Run all tests (unit + integration)

test-with-examples: ## Run all tests including k6 examples (requires MILVUS_HOST)
	./scripts/test-all.sh $(MILVUS_HOST)

coverage: test-verbose ## Generate and view coverage report
	go tool cover -html=coverage.txt

lint: ## Run linters
	go vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin"; \
	fi

fmt: ## Format code
	go fmt ./...
	gofmt -s -w .

fmt-md: ## Format markdown files
	@if command -v prettier >/dev/null 2>&1; then \
		prettier --write "**/*.md"; \
	else \
		echo "prettier not installed. Install with: npm install -g prettier"; \
		exit 1; \
	fi

lint-md: ## Lint markdown files
	@if command -v markdownlint-cli2 >/dev/null 2>&1; then \
		markdownlint-cli2 "**/*.md"; \
	elif command -v prettier >/dev/null 2>&1; then \
		prettier --check "**/*.md"; \
	else \
		echo "No markdown linter found. Install markdownlint-cli2 or prettier"; \
		echo "  npm install -g markdownlint-cli2"; \
		echo "  or: npm install -g prettier"; \
		exit 1; \
	fi

fix-md: ## Auto-fix markdown lint issues
	@if command -v markdownlint-cli2 >/dev/null 2>&1; then \
		markdownlint-cli2 --fix "**/*.md"; \
	else \
		echo "markdownlint-cli2 not installed"; \
		echo "Install with: npm install -g markdownlint-cli2"; \
		exit 1; \
	fi

mod-tidy: ## Tidy go modules
	go mod tidy

mod-download: ## Download go modules
	go mod download

clean: ## Clean build artifacts
	rm -f k6 coverage.txt coverage.html
	rm -rf dist/

examples: build ## Build k6 and show available examples
	@echo "k6 built successfully!"
	@./k6 version
	@echo ""
	@echo "Available examples:"
	@echo "  Beginner:"
	@echo "    - examples/basic-operations.js"
	@echo "    - examples/collection-management.js"
	@echo "  Intermediate:"
	@echo "    - examples/vector-search.js"
	@echo "  Advanced:"
	@echo "    - examples/hybrid-search.js"
	@echo "    - examples/full-text-search.js"
	@echo ""
	@echo "Run: make run-example FILE=<filename>"

run-examples: build ## Run all examples in sequence
	@echo "Running all examples..."
	@./k6 run examples/basic-operations.js
	@./k6 run examples/collection-management.js
	@./k6 run examples/vector-search.js
	@./k6 run examples/hybrid-search.js
	@./k6 run examples/full-text-search.js

run-example: build ## Run specific example (usage: make run-example FILE=basic-operations.js)
	@if [ -z "$(FILE)" ]; then \
		echo "Error: Please specify FILE parameter"; \
		echo "Usage: make run-example FILE=basic-operations.js"; \
		exit 1; \
	fi
	./k6 run examples/$(FILE)

docker-build: ## Build Docker image
	docker build -t k6-milvus:latest .

docker-run: docker-build ## Run k6 in Docker
	docker run --rm k6-milvus:latest version

docker-up: ## Start Milvus using docker-compose
	@echo "Starting Milvus cluster..."
	docker compose -f deployment/docker-compose.yml up -d --wait
	@echo "Milvus is ready!"

docker-down: ## Stop and remove Milvus containers
	@echo "Stopping Milvus cluster..."
	docker compose -f deployment/docker-compose.yml down -v

docker-logs: ## Show Milvus logs
	docker compose -f deployment/docker-compose.yml logs -f standalone

docker-status: ## Check Milvus container status
	docker compose -f deployment/docker-compose.yml ps

test-integration-local: docker-up ## Run integration tests with local Milvus (starts/stops containers)
	@echo "Running integration tests..."
	@MILVUS_HOST=localhost:19530 go test -tags=integration -v -race ./pkg/milvus || \
		(echo "Integration tests failed, showing logs..."; docker compose -f deployment/docker-compose.yml logs standalone; docker compose -f deployment/docker-compose.yml down -v; exit 1)
	@echo "Integration tests passed, stopping Milvus..."
	@$(MAKE) docker-down

test-e2e-local: build docker-up ## Run E2E tests with local Milvus (starts/stops containers)
	@echo "Running E2E tests..."
	@export MILVUS_HOST=localhost:19530 && \
		./k6 run --quiet examples/basic-operations.js && \
		./k6 run --quiet examples/collection-management.js && \
		./k6 run --quiet examples/vector-search.js && \
		./k6 run --quiet examples/hybrid-search.js && \
		./k6 run --quiet examples/full-text-search.js || \
		(echo "E2E tests failed, showing logs..."; docker compose -f deployment/docker-compose.yml logs standalone; docker compose -f deployment/docker-compose.yml down -v; exit 1)
	@echo "E2E tests passed, stopping Milvus..."
	@$(MAKE) docker-down

test-all-local: test-integration-local test-e2e-local ## Run all tests with local Milvus (unit + integration + E2E)

generate-vectors: ## Generate sample test vectors
	@echo "Generating sample vectors..."
	@if [ -f examples/data/generate_vectors.go ]; then \
		cd examples/data && go run generate_vectors.go; \
	else \
		echo "Vector generator not yet implemented"; \
	fi

all: clean mod-download build test ## Run all: clean, download, build, test

.DEFAULT_GOAL := help
