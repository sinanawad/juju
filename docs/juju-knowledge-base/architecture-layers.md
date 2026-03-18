# Juju Architecture Layers

## Layer Hierarchy

```
┌─────────────────────────────────────────────────────────────────┐
│  cmd/                                                           │
│  CLI entry points: juju (client), jujud (agent), jujuc (hooks), │
│  jujud-controller (controller daemon)                           │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│  api/                                                           │
│  Client-side API consumers. Thin translation layer:             │
│  wire types (params) → model types. Avoid "State" types that    │
│  mimic remote objects.                                          │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│  apiserver/                                                     │
│  RPC facade implementations. Thin orchestration only:           │
│  - Authorization checks                                         │
│  - Encoding/decoding wire data                                  │
│  - Calling domain services                                      │
│  Must NOT contain business logic.                               │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│  domain/                                                        │
│  Business logic layer:                                          │
│  - service/ subdirs: orchestration, validation, cross-domain    │
│  - state/ subdirs: persistence via Sqlair, transaction mgmt     │
│  Services depend on State interfaces, not implementations.      │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│  core/                                                          │
│  Shared primitives with minimal dependencies:                   │
│  - Identity types (UUID, Name wrappers with validation)         │
│  - Status enums, lifecycle states                               │
│  - Network types, constraints                                   │
│  - Interfaces (Watcher, Repository)                             │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│  internal/worker/                                               │
│  Background task actors:                                        │
│  - Long-running operations                                      │
│  - tomb.Tomb or catacomb.Catacomb for lifecycle                 │
│  - dependency.Engine for orchestration                          │
│  Each worker: restartable, deterministic, observes cancellation │
└─────────────────────────────────────────────────────────────────┘
```

## Import Rules

| Package | May Import | Must NOT Import |
|---------|------------|-----------------|
| `domain` | `core` | `apiserver`, `cmd`, `internal/worker` |
| `apiserver` | `domain`, `core` | `cmd` |
| `internal/worker` | `domain`, `api`, `core` | `cmd` |
| `core` | other `core/*` subpackages, external | any other juju packages |
| `api` | `core`, params types | `domain`, `apiserver` |
| `cmd` | anything (top of stack) | - |

## Directory Purposes

### `cmd/`
Entry points for all binaries:
- `cmd/juju/` - Client CLI
- `cmd/jujud/` - Agent daemon (machines, units)
- `cmd/jujuc/` - Unit hook tool
- `cmd/jujud-controller/` - Controller daemon (requires dqlite)
- `cmd/containeragent/` - Container-based agent
- `cmd/plugins/juju-metadata/` - Metadata plugin

### `domain/`
62+ subdomains organized by business concept:
- `domain/application/` - Application lifecycle, charms, units
- `domain/machine/` - Machine provisioning, instances
- `domain/model/` - Model metadata and lifecycle
- `domain/relation/` - Relations between applications
- `domain/secret/` - Secret management
- `domain/services/` - Factory for creating services (special)

Each subdomain has:
```
domain/{name}/
├── service/          # Business logic
├── state/            # Data persistence (Sqlair)
├── errors/           # Domain-specific errors
├── types.go          # Domain types
└── modelmigration/   # Migration support (optional)
```

### `apiserver/`
RPC endpoint implementations:
- `apiserver/facades/agent/` - Agent-facing facades
- `apiserver/facades/client/` - Client-facing facades
- `apiserver/facades/controller/` - Controller-specific facades
- `apiserver/common/` - Shared utilities (auth helpers, block checking)
- `apiserver/facade/` - Registry and interfaces

### `core/`
69 subdirectories of primitives:
- `core/application/`, `core/machine/`, `core/unit/` - Identity types
- `core/status/` - Status enums
- `core/life/` - Lifecycle states (Alive, Dying, Dead)
- `core/network/` - Address types with scope
- `core/constraints/` - Hardware constraints
- `core/watcher/` - Watcher interface
- `core/permission/` - Access levels

### `internal/worker/`
123+ worker implementations:
- `internal/worker/uniter/` - Unit agent main worker
- `internal/worker/machiner/` - Machine agent worker
- `internal/worker/provisioner/` - Machine provisioning
- `internal/worker/apiserver/` - API server worker
- `internal/worker/dbaccessor/` - Database access
- `internal/worker/fortress/`, `internal/worker/gate/` - Synchronization primitives

### `agent/`
Agent configuration and engine setup:
- `agent/engine/` - Manifold helpers, housing decorator
- Agent manifold composition

### Other Key Directories
- `rpc/params/` - Wire format types (never change field names)
- `environs/` - Cloud environment abstractions
- `caas/` - Kubernetes/container orchestration
- `tests/` - Integration tests (bash-based)
