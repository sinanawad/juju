# Implementation Plan: Kubernetes Deployment and DaemonSet Workload Types

**Branch**: `001-k8s-deployment-types` | **Date**: 2026-02-08 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-k8s-deployment-types/spec.md`

## Summary

Enable Juju to deploy K8s applications as Deployments, StatefulSets, or DaemonSets instead of hardcoding StatefulSet for all CAAS applications. The deployment type is selected via a new `deployment-type` constraint (with automatic inference from charm storage metadata as fallback). The K8s provider layer already supports all three types; this feature wires the selection mechanism through constraints, domain persistence, worker provisioning, and status display.

## Technical Context

**Language/Version**: Go (per `go.mod`)
**Primary Dependencies**: DQLite, Sqlair, client-go (K8s), tomb/catacomb (worker lifecycle)
**Storage**: DQLite (new `deployment_type` lookup table + column on `application`)
**Testing**: `go test` with gomock for service/facade tests, Sqlair test harness for state tests
**Target Platform**: Linux (controller), any K8s cluster (target)
**Project Type**: Existing monorepo with strict layering
**Performance Goals**: No regression from current deploy/status performance
**Constraints**: Backward compatible — existing apps continue as StatefulSet
**Scale/Scope**: ~30 files modified across 6 layers, no new packages

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Everything Fails | PASS | Deployment type persisted to DB with default; idempotent on upgrade |
| II. Strict Architectural Layering | PASS | Changes respect layer boundaries: core → domain → apiserver → worker → cmd |
| III. Managed Concurrency | PASS | No new goroutines; existing worker patterns reused |
| IV. Test Discipline | PASS | Unit tests for each layer; deterministic clock usage maintained |
| V. Domain Service Encapsulation | PASS | Business logic in `domain/application/service/`; persistence in `state/` |
| VI. Access to Clouds via Providers | PASS | Provider accessed through existing broker interface; no direct provider access |
| VII. Resource Ownership | PASS | No new resources to manage |
| VIII. Simplicity and Minimalism | PASS | Reuses existing constraint mechanism; minimal new abstractions |

**Post-design re-check**: All gates continue to pass. The inference heuristic is a simple function, not a new abstraction pattern.

## Project Structure

### Documentation (this feature)

```text
specs/001-k8s-deployment-types/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 research findings
├── data-model.md        # Data model changes
├── quickstart.md        # Build/test/verify guide
├── contracts/           # API contract changes
│   ├── provisioning-info.md
│   ├── constraints.md
│   └── status.md
├── checklists/
│   └── requirements.md
└── tasks.md             # Phase 2 output (from /speckit.tasks)
```

### Source Code (repository root)

```text
core/constraints/               # New deployment-type constraint field
domain/constraints/             # Domain constraint mapping
domain/application/             # Types, service, state changes
domain/application/errors/      # New error for DaemonSet scaling
domain/schema/model/sql/        # New PATCH file for deployment_type table
domain/schema/                  # Schema registration
domain/status/service/          # Status type changes

apiserver/facades/controller/caasapplicationprovisioner/  # Facade v2, provisioning info
apiserver/facades/client/client/                          # Status facade
api/controller/caasapplicationprovisioner/                # Client-side facade
api/                                                      # Facade version list

internal/worker/caasapplicationprovisioner/  # Worker: inference + dynamic type
internal/worker/caasfirewaller/              # Firewaller: read type from state

rpc/params/                     # Wire types
cmd/juju/status/                # Status display
```

**Structure Decision**: No new packages. All changes fit within existing directory structure, following established patterns in each location.

## Incremental Delivery Plan

Each user story is independently implementable and testable. Stories are ordered by dependency — earlier stories establish the foundation that later stories build on, but each produces a testable, demoable increment.

### Story 1+2+4: Foundation + Constraint + Backward Compat (P1)

**Goal**: Charms deploy with the correct K8s workload type based on constraint or inference, while all existing charms continue as StatefulSet.

**Layers touched**: core → domain → apiserver → worker

**Steps**:
1. Add `DeploymentType *string` to `core/constraints/constraints.go` Value struct
2. Add parsing, validation, and serialization for `deployment-type` constraint
3. Map new constraint in `domain/constraints/constraints.go`
4. Create DB PATCH: `deployment_type` lookup table + column on `application`
5. Register PATCH in `domain/schema/model.go`
6. Add `DeploymentType` field to `AddCAASApplicationArg` in `domain/application/types.go`
7. Persist deployment type in `domain/application/state/application.go` during CreateCAASApplication
8. Add `GetApplicationDeploymentType()` to application service
9. Add `DeploymentType` to `params.CAASApplicationProvisioningInfo`
10. Bump CAASApplicationProvisioner facade to v2; populate deployment type in provisioning info
11. Update `api/facadeversions.go` with new version
12. Implement `DetermineDeploymentType()` in worker ops: constraint → storage heuristic → default
13. Replace hardcoded `caas.DeploymentStateful` in `application.go:149` with dynamic type
14. Replace hardcoded `caas.DeploymentStateful` in `caasfirewaller/appfirewaller.go:81`
15. Replace hardcoded `caas.DeploymentStateful` in `domain/application/service/provider.go:442,538`
16. Add warning when `deployment-type=stateless` but charm has storage (FR-012)

**Test**: Deploy a charm with no storage → verify it runs as Deployment. Deploy with storage → verify StatefulSet. Deploy with explicit constraint → verify override works. Verify existing charms continue as StatefulSet.

**Validates**: FR-001, FR-002, FR-003, FR-004, FR-008, FR-009, FR-010, FR-011, FR-012 | SC-001, SC-003, SC-005, SC-007

---

### Story 3: DaemonSet Support (P2)

**Goal**: Operators can deploy charms as DaemonSets with proper scale blocking.

**Layers touched**: domain

**Steps**:
1. Add `DaemonSetScaleNotSupported` error to `domain/application/errors/errors.go`
2. Add deployment type check in scale validation (`domain/application/service/application.go`) — reject scale operations for daemon type
3. Test: Attempt to scale a DaemonSet app → verify error returned

**Test**: Deploy with `deployment-type=daemon` → verify DaemonSet created. Try `juju scale-application` → verify clear error. Add node to cluster → verify new pod appears.

**Validates**: FR-005, FR-006 | SC-002, SC-006

---

### Story 5: Status Visibility (P2)

**Goal**: Operators can see the workload type for each K8s application in status output.

**Layers touched**: domain → apiserver → rpc → cmd

**Steps**:
1. Add `DeploymentType *string` to `domain/status/service/types.go` Application struct
2. Populate deployment type in status assembly (`apiserver/facades/client/client/status.go`)
3. Add `DeploymentType string` to `rpc/params/status.go` ApplicationStatus struct
4. Add `DeploymentType string` to `cmd/juju/status/formatted.go` applicationStatus struct
5. Map field in `cmd/juju/status/formatter.go`
6. Add "Type" column to CAAS application table headers in `output_tabular.go:142`
7. Print deployment type value in the table rendering loop
8. Test: Deploy multiple apps with different types → verify status shows correct types

**Test**: Run `juju status` on a model with mixed deployment types → verify "Type" column shows correct values. Verify IAAS model has no Type column.

**Validates**: FR-007 | SC-004

## Complexity Tracking

No constitution violations. All changes use existing patterns.

## Files Modified (~30)

### Core Layer
- `core/constraints/constraints.go` — Add DeploymentType field
- `core/constraints/constraints_test.go` — Test new constraint

### Domain Layer
- `domain/constraints/constraints.go` — Map new constraint
- `domain/schema/model/sql/NNNN-deployment-type.PATCH.sql` — NEW: lookup table + column
- `domain/schema/model.go` — Register PATCH
- `domain/application/types.go` — Add DeploymentType to AddCAASApplicationArg
- `domain/application/state/application.go` — Persist deployment type
- `domain/application/service/application.go` — GetApplicationDeploymentType, scale validation
- `domain/application/service/provider.go` — Replace hardcoded StatefulSet
- `domain/application/errors/errors.go` — DaemonSetScaleNotSupported error
- `domain/status/service/types.go` — Add DeploymentType to Application

### API Server Layer
- `apiserver/facades/controller/caasapplicationprovisioner/register.go` — v1 → v2
- `apiserver/facades/controller/caasapplicationprovisioner/provisioner.go` — Add DeploymentType to ProvisioningInfo
- `apiserver/facades/client/client/status.go` — Populate deployment type in status

### API Client Layer
- `api/facadeversions.go` — Add CAASApplicationProvisioner v2
- `api/controller/caasapplicationprovisioner/client.go` — Parse DeploymentType

### Worker Layer
- `internal/worker/caasapplicationprovisioner/application.go` — Dynamic deployment type
- `internal/worker/caasapplicationprovisioner/ops.go` — DetermineDeploymentType heuristic
- `internal/worker/caasfirewaller/appfirewaller.go` — Read deployment type

### RPC/Wire Types
- `rpc/params/status.go` — Add DeploymentType to ApplicationStatus
- `rpc/params/caas.go` or equivalent — Add DeploymentType to CAASApplicationProvisioningInfo

### CLI Layer
- `cmd/juju/status/formatted.go` — Add DeploymentType field
- `cmd/juju/status/formatter.go` — Map DeploymentType
- `cmd/juju/status/output_tabular.go` — Add "Type" column for CAAS

### Test Files (updates to existing)
- `core/constraints/constraints_test.go`
- `domain/application/state/application_test.go`
- `domain/application/service/application_test.go`
- `domain/application/service/provider_test.go`
- `apiserver/facades/controller/caasapplicationprovisioner/provisioner_test.go`
- `internal/worker/caasapplicationprovisioner/application_test.go`
- `cmd/juju/status/output_tabular_test.go`
