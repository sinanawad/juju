# Constitution Violations & Known Exceptions

**Audit date**: 2026-02-08
**Constitution version**: 1.2.0
**Status**: Documented — violations marked as legacy exceptions in the constitution. Do not extend.

---

## Principle II: Strict Architectural Layering

### core/ imports domain/ (19 files)

**Rule violated**: `core/` MUST only import `core/` sub-packages or externals.

**Root cause**: `domain/deployment/charm` and `domain/deployment/charm/resource` contain shared type definitions (charm metadata, resource types) that both `core/` and upper layers need. These packages act as shared type libraries rather than domain business logic.

**Potential fix**: Move `domain/deployment/charm` types into `core/charm` (or a new `core/deployment/charm` package) so the dependency flows correctly downward.

| # | File | Violating Import |
|---|------|-----------------|
| 1 | `core/assumes/featureset.go` | `domain/deployment/charm/assumes` |
| 2 | `core/assumes/sat_checker.go` | `domain/deployment/charm/assumes` |
| 3 | `core/base/base.go` | `domain/deployment/charm` |
| 4 | `core/charm/adaptor.go` | `domain/deployment/charm` |
| 5 | `core/charm/channel.go` | `domain/deployment/charm` |
| 6 | `core/charm/charmpath.go` | `domain/deployment/charm` |
| 7 | `core/charm/computedbase.go` | `domain/deployment/charm` |
| 8 | `core/charm/origin.go` | `domain/deployment/charm` |
| 9 | `core/charm/repository.go` | `domain/deployment/charm`, `domain/deployment/charm/resource` |
| 10 | `core/crossmodel/interface.go` | `domain/deployment/charm` |
| 11 | `core/crossmodel/params.go` | `domain/deployment/charm` |
| 12 | `core/operation/action.go` | `domain/deployment/charm` |
| 13 | `core/relation/key.go` | `domain/deployment/charm` |
| 14 | `core/resource/application.go` | `domain/deployment/charm/resource` |
| 15 | `core/resource/content.go` | `domain/deployment/charm/resource` |
| 16 | `core/resource/resource.go` | `domain/deployment/charm/resource` |
| 17 | `core/resource/serialization.go` | `domain/deployment/charm/resource` |
| 18 | `core/resource/testing/resource.go` | `domain/deployment/charm/resource` |
| 19 | `core/settings/settings.go` | `domain/deployment/charm` |

---

### api/ imports domain/ (16 files)

**Rule violated**: `api/` should only import `core/`.

**Root cause**: Same as above — `domain/deployment/charm` types are used as shared type definitions across layers.

**Potential fix**: Same as above — relocate charm types to `core/`.

| # | File | Violating Import |
|---|------|-----------------|
| 1 | `api/agent/provisioner/provisioner.go` | `domain/network` |
| 2 | `api/agent/uniter/endpoint.go` | `domain/deployment/charm` |
| 3 | `api/agent/uniter/relation.go` | `domain/deployment/charm` |
| 4 | `api/agent/uniter/unit.go` | `domain/deployment/charm` |
| 5 | `api/client/application/client.go` | `domain/deployment/charm` |
| 6 | `api/client/applicationoffers/client.go` | `domain/deployment/charm` |
| 7 | `api/client/charms/client.go` | `domain/deployment/charm` |
| 8 | `api/client/charms/downloader_s3.go` | `domain/deployment/charm` |
| 9 | `api/client/charms/localcharmclient.go` | `domain/deployment/charm` |
| 10 | `api/client/resources/client.go` | `domain/deployment/charm/resource` |
| 11 | `api/client/resources/helpers.go` | `domain/deployment/charm/resource` |
| 12 | `api/client/resources/upload.go` | `domain/deployment/charm/resource` |
| 13 | `api/common/charm/charmorigin.go` | `domain/deployment/charm` |
| 14 | `api/common/charms/common.go` | `domain/deployment/charm`, `domain/deployment/charm/resource` |
| 15 | `api/common/secretbackends/client.go` | `domain/deployment/charm` |
| 16 | `api/controller/migrationmaster/client.go` | `domain/deployment/charm` |

---

### internal/worker/ imports cmd/ (2 files)

**Rule violated**: `internal/worker/` MUST NOT import `cmd/`.

**Potential fix**: Extract the needed functionality into a shared package or pass it via dependency injection.

| # | File | Violating Import | Usage |
|---|------|-----------------|-------|
| 1 | `internal/worker/apiserver/manifold.go` (line 22) | `cmd/juju/commands` | Calls `commands.NewJujuCommandWithStore()` for embedded CLI execution |
| 2 | `internal/worker/uniter/runner/jujuc/goal-state.go` (line 10) | `cmd/juju/common` | Calls `common.FormatTime()` for time formatting |

---

## Principle III: Managed Concurrency

### Bare goroutines in apiserver/ (4 files, 6 goroutines)

**Rule violated**: All goroutines MUST be managed via tomb/catacomb/dependency.Engine.

**Context**: These are request-scoped goroutines in API handlers. Their lifetime is bounded by the request, but they lack formal lifecycle management.

**Potential fix**: Refactor to use `context.Context` cancellation exclusively without spawning goroutines, or wrap in a request-scoped tomb.

| # | File | Line(s) | Description | Severity |
|---|------|---------|-------------|----------|
| 1 | `apiserver/observer/observer.go` | 125-128, 132-135 | Two bare goroutines in `mapConcurrent()` — fan-out with `sync.WaitGroup` | Low — bounded by observer count |
| 2 | `apiserver/logsink/logsink.go` | 300 | Bare goroutine for websocket log reception | Medium — long-lived websocket handler |
| 3 | `apiserver/debuglog.go` | 140-146 | Bare goroutine monitoring context cancellation. Code comment (line 137): *"This should really use a tomb"* | Medium — acknowledged tech debt |
| 4 | `apiserver/embeddedcli.go` | 112-136, 293-309 | Two bare goroutines for CLI command I/O (receiver + executor) | Medium — websocket handler |

### Bare goroutine in domain/ (1 file, 1 goroutine)

| # | File | Line(s) | Description | Severity |
|---|------|---------|-------------|----------|
| 5 | `domain/leaseservice.go` | 110-128 | Bare goroutine in `WithLeader()` waiting for lease expiry | High — domain layer should not spawn goroutines |

---

## Principle VI: Access to Clouds via Providers

### Domain service imports provider package (1 file)

**Rule violated**: Providers MUST NOT be accessed directly from domain services.

**Potential fix**: Move the `ExecRBACResourceName` constant to `core/` or pass it via configuration/dependency injection.

| # | File | Line | Violating Import | Usage |
|---|------|------|-----------------|-------|
| 1 | `domain/modelprovider/service/service.go` | 20, 97 | `internal/provider/kubernetes` | References `k8sprovider.ExecRBACResourceName` (a string constant) |

### cmd/ imports provider packages beyond bootstrap (6 files)

**Context**: The constitution allows `cmd/` to import providers for registration and bootstrap. Most of these are borderline — importing constants or performing K8s-specific CLI operations. Listed here for completeness.

| # | File | Import | Usage | Assessment |
|---|------|--------|-------|------------|
| 1 | `cmd/juju/common/controller.go` | `internal/provider/kubernetes` | `DecideControllerNamespace()` | Borderline — CLI needs namespace logic |
| 2 | `cmd/juju/caas/add.go` | `internal/provider/kubernetes` | K8s cloud registration | Acceptable — cloud management CLI |
| 3 | `cmd/juju/caas/update.go` | `internal/provider/kubernetes`, `internal/provider/kubernetes/proxy` | K8s cloud update | Acceptable — cloud management CLI |
| 4 | `cmd/juju/ssh/ssh_container.go` | `internal/provider/kubernetes`, `internal/provider/kubernetes/exec` | K8s exec for SSH | Borderline — could use interface |
| 5 | `cmd/juju/controller/listcontrollersconverters.go` | `internal/provider/kubernetes/constants` | Constants only | Acceptable |
| 6 | `cmd/juju/storage/poolcreate.go` | `internal/provider/kubernetes/constants` | Constants only | Acceptable |

---

## Summary by Severity

| Severity | Count | Description |
|----------|-------|-------------|
| **High** | 1 | `domain/leaseservice.go` bare goroutine — domain layer should never spawn goroutines |
| **Medium** | 5 | Bare goroutines in apiserver handlers (3 files), domain provider import (1 file), worker→cmd imports (2 files) |
| **Low** | 1 | Observer fan-out goroutines with WaitGroup |
| **Structural** | 35 | `core/` and `api/` importing `domain/deployment/charm` — requires package relocation to fix properly |

**Total violations**: 42 across 3 principles

---

## Principles With No Violations

| Principle | Status |
|-----------|--------|
| I. Everything Fails | Design principle (not statically testable) |
| IV. Test Discipline | Verified — deterministic patterns and error documentation consistently applied |
| V. Domain Service Encapsulation | Verified — zero DB imports in facades, clean service/state separation |
| VII. Resource Ownership | Design principle (not statically testable) |
| VIII. Simplicity and Minimalism | Design principle (not statically testable) |
