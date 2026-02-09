<!--
  Sync Impact Report
  ==================
  Version change: 1.2.0 → 1.2.1
  Modified principles: None
  Added constraints:
    - "Migration serialization boundary" in Architectural Constraints.
      New persisted fields must survive the description library
      round-trip or have documented re-derivation strategies.
      Motivated by k8s deployment-type migration gap analysis.
  Removed sections: None
  Templates requiring updates: None
  Follow-up TODOs: None

  History:
    0.0.0 → 1.0.0: Initial adoption (7 principles)
    1.0.0 → 1.1.0: Replaced "Incremental Delivery" with
      "Access to Clouds via Providers"; renumbered VI-VIII
    1.1.0 → 1.2.0: Codebase verification audit (2026-02-08).
      Clarified II, III, VI to reflect known exceptions.
      Added Codebase Verification section with test results.
    1.2.0 → 1.2.1: Added migration serialization boundary constraint
      from deployment-type lifecycle audit (2026-02-08).
-->

# Juju Constitution

## Core Principles

### I. Everything Fails

All code MUST be idempotent and resumable. User operations record
intent to a database and MUST NOT depend on the client remaining
connected. Agent operations MUST retry indefinitely until the
declared model state is achieved. No operation may leave the system
in an unrecoverable state on partial failure. It is mandatory that each
transaction is independent, and does not use any values from previous
attempts.

**Rationale**: Juju orchestrates distributed infrastructure where
networks drop, processes die, and disks fill. Every code path MUST
assume failure at any point and recover gracefully.

### II. Strict Architectural Layering

The codebase MUST respect the following import hierarchy. Layers
MUST NOT import upward:

```
cmd/              → may import all layers below
api/              → may import core/, domain/deployment/charm
apiserver/        → may import domain/, core/; MUST NOT import cmd/
domain/           → MUST NOT import apiserver/, cmd/, internal/worker/
core/             → may import core/ sub-packages, externals, and
                    domain/deployment/charm (shared type packages)
internal/worker/  → may import domain/, api/; MUST NOT import cmd/
```

New cross-layer dependencies MUST NOT be introduced. Existing
local conventions MUST be followed before introducing new
abstractions.

**Known exceptions** (legacy; do not extend):
- `core/` imports `domain/deployment/charm` and
  `domain/deployment/charm/resource` for shared charm types
  (19 files). These packages act as shared type definitions.
- `api/` imports `domain/deployment/charm` for the same reason
  (16 files).
- `internal/worker/apiserver/manifold.go` imports `cmd/juju/commands`
  for embedded CLI execution.
- `internal/worker/uniter/runner/jujuc/goal-state.go` imports
  `cmd/juju/common` for `FormatTime()`.

**Rationale**: Strict layering prevents circular dependencies,
enables independent testing of each layer, and keeps business logic
decoupled from transport and CLI concerns.

### III. Managed Concurrency

All long-lived goroutines MUST be managed via the worker framework
(`tomb.Tomb`, `catacomb.Catacomb`, `worker.dependency.Engine`).
Workers MUST be restartable, deterministic, and observe cancellation
via `context.Context`. Request-scoped goroutines in API handlers
may use `context.Context` cancellation and `sync.WaitGroup` as an
alternative to tomb/catacomb when the goroutine lifetime is bounded
by the request.

**Known exceptions** (legacy; do not extend):
- `apiserver/observer/observer.go` — bare goroutines with WaitGroup
- `apiserver/logsink/logsink.go` — bare goroutine for log reception
- `apiserver/debuglog.go` — bare goroutine for context monitoring
- `apiserver/embeddedcli.go` — bare goroutines for CLI command I/O
- `domain/leaseservice.go` — bare goroutine for lease expiry waiting

**Rationale**: Unmanaged concurrency causes resource leaks, data
races, and ungraceful shutdowns in a long-running distributed
system.

### IV. Test Discipline

Unit tests and integration tests are both required. Tests MUST be
deterministic: never use `time.Now()` directly — inject
`clock.Clock` and use `*testing.Clock` in tests. State-layer tests
MUST use Sqlair and the established database test harness. Contract
tests MUST cover cross-service boundaries.

**Rationale**: Juju is a large, concurrent system where
non-deterministic tests erode confidence and waste CI resources.

### V. Domain Service Encapsulation

Business logic MUST reside in `domain/*/service/` packages.
Persistence MUST be abstracted behind `domain/*/state/` interfaces.
No transaction or database details may leak out of state packages.
State method arguments MUST be simple types or domain-internal
types. Services depend on state indirections, not implementations.
API facades MUST be thin orchestration — auth, encoding/decoding,
and delegation to domain services only.

**Rationale**: Clean separation of business logic from persistence
and transport enables independent evolution, testing, and reasoning
about each concern.

**Verified**: Facades contain zero sqlair/database imports. All
persistence code lives in `domain/*/state/` packages. Services
use interface abstractions, not concrete state implementations.

### VI. Access to Clouds via Providers

Provider implementations (`internal/provider`) MUST NOT be accessed
directly from `apiserver/` or `internal/worker/` (except
`providertracker`), but through interfaces. Providers MUST NOT be
used outside the `domain/*/service` packages, which get providers
via worker dependency. Providers MUST NOT be instantiated by logic
other than in `internal/worker/providertracker`, except during
bootstrap.

`cmd/` packages may import provider packages for registration
(via `internal/provider/all`), constants, and bootstrap operations.

**Known exceptions** (legacy; do not extend):
- `domain/modelprovider/service/service.go` imports
  `internal/provider/kubernetes` for the `ExecRBACResourceName`
  constant.

**Rationale**: Manipulating the cloud via providers is the
responsibility of the controller. Provider instantiation can be
costly, so the version cached by the provider tracker is re-used.

### VII. Resource Ownership

If you start, open, or create a resource (file, connection, session,
goroutine, worker), you MUST either stop/close/destroy it or
explicitly hand ownership to another component. Every resource MUST
have exactly one owner at all times. Workers MUST clean up promptly
and gracefully when their `Dying` channel fires.

**Rationale**: Resource leaks in a long-running orchestration engine
compound over time, eventually causing instability in production
deployments.

### VIII. Simplicity and Minimalism

Start simple. Avoid premature abstraction, speculative generality,
and unnecessary indirection. New patterns, abstractions, and
dependencies MUST NOT be introduced unless absolutely necessary.
Prefer minimal diffs that improve clarity, safety, or correctness.
Follow existing local conventions in each directory.

**Rationale**: Juju is a large, long-lived codebase. Every
unnecessary abstraction increases cognitive load and maintenance
burden for all contributors.

## Architectural Constraints

- **Language**: Go (current version per `go.mod`)
- **Database**: DQLite for controller persistence; Sqlair for query
  construction in state packages
- **API protocol**: RPC facades with bulk arguments (accept and
  return arrays, process all items)
- **Migration serialization boundary**: New persisted fields MUST
  survive the model migration round-trip through the `description`
  library. If the `description` library lacks support for a new
  field, it MUST be added before the feature ships, or an
  equivalent re-derivation strategy MUST be documented and tested
- **Watcher contract**: Watchers MUST send an initial event
  indicating baseline state; clients read full state on every event
- **Import grouping**: stdlib, 3rd-party, juju — each
  alphabetically sorted
- **Line length**: 120 characters (code), 80 characters (comments)
- **Formatting**: All code MUST pass `go fmt`
- **Error documentation**: Methods returning errors MUST document
  possible error types in doc comments using the pattern:
  `// The following errors may be returned:`

## Development Workflow

- **Build**: `make install` (full) or `make go-build` (no schema)
- **Unit tests**: `go test ./path/to/package` or `make test`
- **Lint**: `make pre-check` (golangci-lint)
- **Integration tests**: `cd tests && ./main.sh <suite>`
- **Pre-merge gate**: `make check` (lint + tests) MUST pass
- All changes MUST be reviewed and MUST comply with the principles
  in this constitution
- When modifying a subsystem, examine all files in the directory to
  maintain consistency
- Avoid adding new global state

## Governance

This constitution is the authoritative reference for Juju
development principles. It supersedes ad-hoc guidance when
conflicts arise.

**Amendment procedure**: Any change to this constitution MUST be
documented with a version bump, rationale, and migration plan for
affected code. Amendments follow semantic versioning (see below).

**Versioning policy**:
- MAJOR: Backward-incompatible principle removal or redefinition
- MINOR: New principle or materially expanded guidance
- PATCH: Clarifications, wording fixes, non-semantic refinements

**Compliance**: All pull requests and code reviews MUST verify
compliance with these principles. Violations MUST be justified in
a Complexity Tracking table (see plan template) or rejected.

**Canonical sources**: `CLAUDE.md`, `AGENTS.md`, `CODING.md`, and
`STYLE.md` provide supplementary detail. This constitution provides
the governing principles; those files provide operational guidance.

### Codebase Verification (2026-02-08)

Each principle was tested against the codebase via automated
analysis. Full violation details with file paths, line numbers,
and suggested fixes: [constitution-violations.md](constitution-violations.md)

Results:

| Principle | Result | Notes |
|-----------|--------|-------|
| I. Everything Fails | Design principle | Not statically testable |
| II. Strict Architectural Layering | Partial | 37 known exceptions documented; domain/ and apiserver/ clean |
| III. Managed Concurrency | Partial | Workers compliant (139 files); 7 bare goroutines in apiserver/domain |
| IV. Test Discipline | Verified | Deterministic patterns, error docs confirmed |
| V. Domain Service Encapsulation | Verified | Zero DB imports in facades; clean service/state separation |
| VI. Access to Clouds via Providers | Mostly | providertracker centralised; 1 domain exception |
| VII. Resource Ownership | Design principle | Not statically testable |
| VIII. Simplicity and Minimalism | Design principle | Not statically testable |

**Version**: 1.2.1 | **Ratified**: 2026-02-06 | **Last Amended**: 2026-02-08
