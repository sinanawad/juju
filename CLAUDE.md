# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
make install              # Build and install all Juju binaries (includes schema rebuild)
make build                # Build all targets with schema rebuild
make go-build             # Build without schema rebuild
make juju                 # Build just the juju client
make jujud-controller     # Build the controller (requires dqlite)
```

Cross-compilation:
```bash
CLIENT_PACKAGE_PLATFORMS="linux/amd64 linux/arm64" make go-client-build
AGENT_PACKAGE_PLATFORMS="linux/amd64" make go-agent-build
```

## Testing

```bash
# Unit tests
make test                                    # Run all unit tests
go test ./path/to/package                   # Run specific package
go test -run 'TestName' ./path/to/package   # Run specific test by regex

# With race detector or coverage
make race-test
make cover-test

# Integration tests (bash-based, in tests/ directory)
cd tests && ./main.sh -h                    # Help
cd tests && ./main.sh deploy                # Run deploy suite
cd tests && ./main.sh deploy test_deploy_bundles  # Run specific test
cd tests && ./main.sh -A                    # Run all integration tests
```

## Linting

```bash
make pre-check            # Static analysis (golangci-lint)
make check                # Static analysis + tests
```

## Architecture Overview

Juju is an orchestration engine for deploying and managing applications via "charms." The codebase follows strict layering:

### Layer Hierarchy (respect import boundaries)

1. **`domain/`** - Business logic and services
   - Services grouped by workflow concern
   - Services depend on state abstractions, not implementations
   - State sub-packages use Sqlair for database queries
   - No transaction/database details leak out of state packages

2. **`apiserver/`** - RPC facade implementations
   - Thin orchestration only (not business logic)
   - Responsibilities: auth, encoding/decoding, calling domain services
   - Must NOT import `cmd` packages

3. **`api/`** - Client-side API consumers
   - Thin translation layer (wire types to model types)

4. **`internal/worker/`** - Background task actors
   - Each worker must be restartable, deterministic, observe cancellation
   - Use `catacomb.Catacomb` or `tomb.Tomb` for lifetime management
   - Managed via `worker.dependency.Engine`

5. **`cmd/`** - CLI entry points
   - `cmd/juju` (client), `cmd/jujud` (agent), `cmd/jujuc` (unit hooks), `cmd/jujud-controller`

6. **`core/`** - Shared primitives
   - Cross-cutting concerns only
   - Should only import other `core` sub-packages or external packages

### Import Rules
- `apiserver` → may import `domain` and `core`, NOT `cmd`
- `internal/worker` → may depend on `domain` and `api`, NOT `cmd`
- `domain` → must NOT import `apiserver`, `cmd`, or `internal/worker`
- `core` → should only import `core` sub-packages or external packages

## Key Patterns

### Worker Pattern
```go
type Worker struct {
    config   Config
    catacomb catacomb.Catacomb
}

func (w *Worker) Kill() { w.catacomb.Kill(nil) }
func (w *Worker) Wait() error { return w.catacomb.Wait() }

func New(config Config) (*Worker, error) {
    if err := config.Validate(); err != nil {
        return nil, errors.Trace(err)
    }
    w := &Worker{config: config}
    err := catacomb.Invoke(catacomb.Plan{
        Site: &w.catacomb,
        Work: w.loop,
    })
    return w, err
}
```

### Watcher Pattern
- Always send initial events indicating baseline state
- Client reads state on every event (no special "initial" handling)
- Prefer notification watchers (`struct{}`) over data-heavy events

### Clock Usage
- Never use `time.Now()` directly; inject `clock.Clock`
- Tests must use `*testing.Clock` for deterministic timing

## Code Style

- Line length: 120 chars (code), 80 chars (comments)
- Imports grouped: stdlib, 3rd party, juju (each alphabetically sorted)
- All code formatted with `go fmt`
- Document errors in method docs:
  ```go
  // DoSomething does something.
  // The following errors may be returned:
  // - [errors.NotFound] when the thing cannot be found.
  ```

## Critical Principles

1. **EVERYTHING FAILS** - All code must be idempotent and resumable
2. **Resource cleanup** - If you start it, you manage or hand off its lifetime
3. **No unmanaged goroutines** - Use worker patterns with proper lifecycle
4. **Context propagation** - Always propagate `context.Context`; never ignore cancellation
5. **Bulk arguments** - API facades should accept bulk arguments

## Key Documentation Files

- `CODING.md` - Development philosophy and patterns
- `AGENTS.md` - Agent-specific architecture guidelines
- `STYLE.md` - Code formatting and style rules
- `CONTRIBUTING.md` - Contribution workflow

## Active Technologies
- Go (per `go.mod`) + DQLite, Sqlair, client-go (K8s), tomb/catacomb (worker lifecycle) (001-k8s-deployment-types)
- DQLite (new `deployment_type` lookup table + column on `application`) (001-k8s-deployment-types)
- Bash (test framework, predicate evaluator), YAML (predicates, GitHub Actions), Go (per `go.mod` — Juju codebase under test) + GitHub Actions, `dorny/paths-filter@v3` (existing), `jq` (existing, used in test assertions), MicroK8s (K8s CI provider), LXD (IaaS CI provider) (002-ci-test-suite)
- YAML files (`tests/suites/<name>/predicates.yaml`) — no database (002-ci-test-suite)

## Recent Changes
- 001-k8s-deployment-types: Added Go (per `go.mod`) + DQLite, Sqlair, client-go (K8s), tomb/catacomb (worker lifecycle)
