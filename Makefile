.PHONY: help build test clean install-xk6 examples lint fmt coverage mod-tidy docker-build all

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install-xk6: ## Install xk6 tool
	go install go.k6.io/xk6/cmd/xk6@latest

build: ## Build k6 with xk6-milvus extension
	xk6 build --with github.com/mmga-lab/xk6-milvus=.

test: ## Run tests
	go test -v -race ./pkg/milvus

test-verbose: ## Run tests with verbose output
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./pkg/milvus

test-e2e: ## Run E2E tests (requires running Milvus)
	go test -tags e2e -v ./pkg/milvus

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

generate-vectors: ## Generate sample test vectors
	@echo "Generating sample vectors..."
	@if [ -f examples/data/generate_vectors.go ]; then \
		cd examples/data && go run generate_vectors.go; \
	else \
		echo "Vector generator not yet implemented"; \
	fi

all: clean mod-download build test ## Run all: clean, download, build, test

.DEFAULT_GOAL := help
