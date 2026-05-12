# Repository Guidelines

## Project Structure & Module Organization

This repository builds `xk6-milvus`, a k6 extension for load testing Milvus. Go
extension code lives in `pkg/milvus/`, with registration in `register.go`.
JavaScript examples are in `examples/`, docs are in `docs/` and `api-docs/`, and
local Milvus Docker Compose assets are in `deployment/`. The Python benchmark
CLI is under `xk6-milvus-bench/`, with source in `src/xk6_milvus_bench/`, tests
in `tests/`, configs in `configs/`, and Jinja templates in `templates/`.

## Build, Test, and Development Commands

- `make help` lists supported repository commands.
- `make install-xk6` installs the xk6 builder.
- `make build` builds a local `./k6` binary with this extension.
- `make test` runs Go unit tests for `./pkg/milvus` with race detection.
- `make test-integration-local` starts Milvus via Docker Compose and runs
  integration tests.
- `make test-e2e-local` builds k6, starts Milvus, and runs representative k6
  examples.
- `make lint` runs `go vet` and `golangci-lint` when available.
- `make fmt` runs `go fmt ./...` and `gofmt -s -w .`.

For the Python CLI, work from `xk6-milvus-bench/` and use:

```bash
uv venv -p 3.10
uv pip install -e ".[dev]"
pytest
ruff check .
```

## Coding Style & Naming Conventions

Use standard Go formatting and keep implementation files focused by operation:
`client.go`, `collection.go`, `data.go`, `search.go`, `index.go`,
`snapshot.go`, and related tests. Exported Go identifiers need purpose-focused
comments. Go tests should be table-driven for multiple cases. Python uses Ruff
with a 100-character line length, `py310` target, and first-party imports under
`xk6_milvus_bench`.

## Testing Guidelines

Place Go tests beside implementation files as `*_test.go`; use names like
`TestFunctionName_Scenario`. Integration tests require a Milvus endpoint and
normally read `MILVUS_HOST`, defaulting to `localhost:19530` in Makefile flows.
Clean up clients and temporary collections in tests and examples. Python tests
use `pytest`, with files named `test_*.py` and functions named `test_*`.

## Commit & Pull Request Guidelines

Recent history follows Conventional Commits, for example `feat: ...`,
`fix(test): ...`, and `ci: ...`. Keep scopes short and meaningful. Pull requests
should include a clear summary, linked issues when relevant, test results, and
documentation updates for API or behavior changes. Include example script output
or screenshots only when they clarify user-visible benchmark behavior.

## Security & Configuration Tips

Do not commit credentials, tokens, generated benchmark output, coverage files,
or local build artifacts. Prefer `MILVUS_HOST` and `MILVUS_TOKEN` environment
variables for local testing. Use `make docker-down` after local integration runs
to remove Milvus containers and volumes.
