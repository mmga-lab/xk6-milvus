# Contributing to xk6-milvus

Thank you for your interest in contributing to xk6-milvus! This document provides guidelines and instructions for contributing to the project.

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/version/2/0/code_of_conduct/). By participating, you are expected to uphold this code.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the existing issues to avoid duplicates. When creating a bug report, include:

- **Clear title and description**
- **Steps to reproduce** the issue
- **Expected vs actual behavior**
- **Environment details** (OS, Go version, k6 version, Milvus version)
- **Sample code** or test scripts (if applicable)
- **Error messages** and stack traces
- **Milvus server logs** (if relevant)

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion:

- **Use a clear and descriptive title**
- **Provide a detailed description** of the proposed functionality
- **Explain why this enhancement would be useful** to xk6-milvus users
- **List any alternative solutions** you've considered
- **Provide examples** of how the feature would be used

### Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Make your changes** following the coding standards
3. **Add tests** for any new functionality
4. **Update documentation** as needed (README, API docs, CLAUDE.md)
5. **Ensure tests pass** (`make test` or `go test ./...`)
6. **Run linters** (`make lint` or `golangci-lint run`)
7. **Submit a pull request**

## Development Setup

### Prerequisites

- Go 1.24 or later
- xk6 (`go install go.k6.io/xk6/cmd/xk6@latest`)
- Milvus server (for testing)
- Git
- Make (optional, for using Makefile commands)

### Setting Up Your Development Environment

```bash
# Fork and clone the repository
git clone https://github.com/YOUR_USERNAME/xk6-milvus.git
cd xk6-milvus

# Add upstream remote
git remote add upstream https://github.com/mmga-lab/xk6-milvus.git

# Create a feature branch
git checkout -b feature/your-feature-name

# Install dependencies
go mod download

# Build with xk6
make build
# Or manually:
xk6 build --with github.com/mmga-lab/xk6-milvus=.
```

### Running Milvus for Development

Using Docker Compose (recommended):

```bash
# Using Makefile (recommended)
make docker-up

# Or manually
docker compose -f deployment/docker-compose.yml up -d --wait

# Check status
make docker-status

# View logs
make docker-logs

# Stop and cleanup
make docker-down
```

Or use the Milvus standalone Docker image:

```bash
docker run -d --name milvus \
  -p 19530:19530 \
  -p 9091:9091 \
  milvusdb/milvus:latest
```

### Running Tests

```bash
# Run all tests
make test
# Or:
go test ./...

# Run tests with coverage
make coverage
# Or:
go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

# Run tests for a specific package
go test -v ./pkg/milvus

# Run a specific test
go test -v -run TestConverters ./pkg/milvus

# Run E2E tests (requires running Milvus)
go test -tags e2e -v ./pkg/milvus
```

### Building

```bash
# Build with Makefile
make build

# Or build manually with xk6
xk6 build --with github.com/mmga-lab/xk6-milvus=.

# Build for specific platform
GOOS=linux GOARCH=amd64 xk6 build --with github.com/mmga-lab/xk6-milvus=.
```

### Running Examples

```bash
# Set Milvus host (optional, defaults to localhost:19530)
export MILVUS_HOST=localhost:19530

# Run examples
./k6 run examples/basic-operations.js
./k6 run examples/vector-search.js
./k6 run examples/hybrid-search.js
```

## Project Structure

```text
xk6-milvus/
├── register.go              # Extension registration entry point
├── pkg/milvus/              # Core implementation
│   ├── module.go            # k6 module registration
│   ├── client.go            # Client creation and management
│   ├── collection.go        # Collection operations
│   ├── data.go              # Insert/upsert/delete operations
│   ├── search.go            # Search and query operations
│   ├── index.go             # Index management
│   ├── converters.go        # Type conversions
│   ├── types.go             # Type definitions
│   ├── errors.go            # Error handling
│   ├── config.go            # Configuration structs
│   ├── helpers.go           # Helper functions
│   └── *_test.go            # Tests
├── examples/                # Usage examples
├── docs/                    # Documentation
├── .github/                 # GitHub workflows and templates
└── deployment/              # Deployment configurations
```

## Coding Standards

### Go Code Style

- Follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` to format your code (or `make fmt`)
- Use `go vet` to check for common mistakes
- Run `golangci-lint run` for comprehensive linting
- Keep functions focused and under 50 lines when possible
- Add godoc comments for exported functions and types

### Code Organization

- Place implementation in `pkg/milvus/`
- Keep files focused on specific functionality:
  - `client.go` - Client creation
  - `collection.go` - Collection operations
  - `data.go` - Data operations (insert/upsert/delete)
  - `search.go` - Search and query
  - `index.go` - Index operations
- Co-locate tests with implementation (`*_test.go`)

### Testing

- Write table-driven tests for functions with multiple cases
- Use descriptive test names: `TestFunctionName_Scenario`
- Test both success and error cases
- Use `check` package in examples to validate results
- Ensure tests clean up resources (close clients, drop test collections)

Example test structure:

```go
func TestSearch_Success(t *testing.T) {
    tests := []struct {
        name    string
        vectors [][]float32
        topK    int
        params  map[string]interface{}
        want    bool
    }{
        {
            name:    "single vector search",
            vectors: [][]float32{{0.1, 0.2}},
            topK:    10,
            params:  map[string]interface{}{"vectorField": "embedding"},
            want:    true,
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Documentation

- Update `README.md` for user-facing changes
- Update `docs/API.md` for new methods or parameters
- Update `CLAUDE.md` for implementation details
- Add examples in `examples/` for new features
- Use clear, concise comments in code
- Document complex algorithms or patterns

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/) specification:

```text
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

Types:

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

Examples:

```text
feat(search): add hybrid search support

Implement multi-vector hybrid search with RRF and weighted reranking.
Supports multiple vector fields in a single query.

fix(client): handle connection timeout properly

Previously, client creation would hang indefinitely if Milvus was unreachable.
Now returns a timeout error after 30 seconds.

docs: update API documentation for createCollection

Add examples for BM25 function configuration and complex schemas.
```

### API Design Principles

This project follows the [Locust Milvus client](https://github.com/locustio/locust/blob/master/locust/contrib/milvus.py) design pattern:

1. **Unified OperationResult**: All operations return `OperationResult` structure

   ```go
   type OperationResult struct {
       Success         bool    `json:"success"`
       ResponseTimeMS  float64 `json:"response_time_ms"`
       Result          any     `json:"result"`
       Error           string  `json:"error"`
       Empty           bool    `json:"empty,omitempty"`
       Recall          float64 `json:"recall,omitempty"`
   }
   ```

2. **Built-in Metrics**: Every operation tracks response time automatically

3. **Collection-Bound Clients**: Support `clientWithCollection()` for cleaner code

4. **Recall Exposure**: Search operations include recall metric

5. **Error Handling**: Return errors in result, don't throw exceptions

When adding new methods, ensure they follow these patterns.

## Making Changes

### Before Submitting

1. **Run tests**: `make test` or `go test ./...`
2. **Run linters**: `make lint` or `golangci-lint run`
3. **Format code**: `make fmt` or `gofmt -w .`
4. **Update docs**: README, API docs, CLAUDE.md as needed
5. **Test examples**: Run example scripts to ensure they work
6. **Update CHANGELOG.md**: Add entry for your changes

### Pull Request Process

1. Update the README.md with details of changes if needed
2. Update the docs/API.md if you're adding or changing methods
3. Update CHANGELOG.md under "Unreleased" section
4. The PR will be merged once you have approval from maintainers

### Pull Request Guidelines

- **Title**: Use conventional commit format
- **Description**: Explain what and why (not how)
- **Link issues**: Reference related issues (#123)
- **Screenshots**: Add screenshots for UI/output changes
- **Breaking changes**: Clearly mark and explain breaking changes
- **Testing**: Describe how you tested the changes

## Release Process

Releases are fully automated using GitHub Actions workflow (`.github/workflows/release.yml`).

### Creating a Release

1. **Update CHANGELOG.md**
   - Move changes from "Unreleased" to a new version section
   - Add release date
   - Follow [Semantic Versioning](https://semver.org/)

2. **Create and push a version tag**

   ```bash
   # Ensure you're on main branch and up to date
   git checkout main
   git pull origin main

   # Create annotated tag (format: v*.*.*)
   git tag -a v0.2.0 -m "Release v0.2.0"

   # Push the tag to trigger release workflow
   git push origin v0.2.0
   ```

3. **Automated Build Process**
   The release workflow automatically:
   - Builds k6 binaries with xk6-milvus for multiple platforms:
     - **OS**: Linux, Windows, macOS
     - **Architectures**: amd64, arm64
   - Creates a GitHub Release with all binaries attached
   - Uses official Grafana xk6 build system for consistency

4. **Verify Release**
   - Check GitHub Releases page
   - Verify all platform binaries are present
   - Test download and run on at least one platform

### Release Checklist

- [ ] All tests pass (`make test`)
- [ ] Linters pass (`make lint`)
- [ ] CHANGELOG.md updated
- [ ] Version follows semantic versioning
- [ ] Tag format is `v*.*.*` (e.g., `v0.2.0`)
- [ ] Breaking changes documented
- [ ] Migration guide provided (if needed)

### Supported Platforms

The release workflow builds for:

- Linux (amd64, arm64)
- Windows (amd64, arm64)
- macOS/Darwin (amd64, arm64)

Total: 6 platform binaries per release

## Getting Help

- **GitHub Issues**: For bugs and feature requests
- **GitHub Discussions**: For questions and community support
- **Documentation**: Check README, API docs, and CLAUDE.md

## Development Commands (Makefile)

```bash
make help                    # Show all available commands
make build                   # Build k6 with extension
make test                    # Run unit tests
make test-integration        # Run integration tests (requires MILVUS_HOST)
make test-integration-local  # Run integration tests with auto Milvus setup
make test-e2e-local          # Run E2E tests with auto Milvus setup
make test-all-local          # Run all tests with auto Milvus setup
make coverage                # Generate coverage report
make lint                    # Run linters
make fmt                     # Format code
make clean                   # Clean build artifacts
make docker-up               # Start Milvus cluster
make docker-down             # Stop Milvus cluster
make docker-logs             # Show Milvus logs
make docker-status           # Check Milvus status
make examples                # Run all examples
```

## Additional Resources

- [k6 Extension Development](https://k6.io/docs/extensions/)
- [Milvus Go SDK Documentation](https://milvus.io/docs/sdk-go.md)
- [Go Testing](https://golang.org/pkg/testing/)
- [Conventional Commits](https://www.conventionalcommits.org/)

Thank you for contributing to xk6-milvus!
