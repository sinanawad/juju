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
**Scale/Scope**: ~38 files modified across 6 layers (including ~8 for storage stories), no new packages

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
| VII. Resource Ownership | PASS | US8: Juju-created standalone PVCs must be deleted on app removal. Ownership is clear: Juju creates → Juju deletes. |
| VIII. Simplicity and Minimalism | PASS | Reuses existing constraint mechanism; minimal new abstractions |

**Post-design re-check (MVP)**: All gates continue to pass. The inference heuristic is a simple function, not a new abstraction pattern.

**Post-storage-stories re-check (2026-02-10)**: Principle VII now applies — US8 introduces PVC lifecycle management for standalone PVCs. All other gates unchanged. No new goroutines, no new abstractions, no new packages.

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
17. Add `DeploymentTypeImmutable` error and enforce immutability in `SetApplicationConstraints` (FR-006)

**Test**: Deploy a charm with no storage → verify it runs as Deployment. Deploy with storage → verify StatefulSet. Deploy with explicit constraint → verify override works. Verify existing charms continue as StatefulSet. Attempt to change deployment-type on running app → verify error.

**Validates**: FR-001, FR-002, FR-003, FR-004, FR-006, FR-008, FR-009, FR-010, FR-011, FR-012 | SC-001, SC-003, SC-005, SC-007

---

### Story 3: DaemonSet Support (P2)

**Goal**: Operators can deploy charms as DaemonSets with proper scale blocking.

**Layers touched**: domain

**Steps**:
1. Add `DaemonSetScaleNotSupported` error to `domain/application/errors/errors.go`
2. Add deployment type check in scale validation (`domain/application/service/application.go`) — reject scale operations for daemon type
3. Test: Attempt to scale a DaemonSet app → verify error returned

**Test**: Deploy with `deployment-type=daemon` → verify DaemonSet created. Try `juju scale-application` → verify clear error. Add node to cluster → verify new pod appears.

**Validates**: FR-005 | SC-002, SC-006

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

## Pre-Merge Dependencies

### RELEASE BLOCKER: `description/v11` — Add `DeploymentType` to Constraints

The `github.com/juju/description/v11` library serializes model data during migration. Its `ConstraintsArgs` struct and `Constraints` interface do **not** include a `DeploymentType` field. This means:

- The `deployment-type` constraint is silently dropped during export/import
- **All DaemonSet apps will change workload type after migration** (daemon cannot be re-inferred)
- Explicitly overridden types (e.g., `stateless` on a charm with storage) are lost

**Mitigation in place**: Import re-infers deployment type from charm metadata when no constraint is available. This correctly handles the common case (inferred stateless/stateful) but cannot recover explicit overrides or `daemon`.

**Required before merge to main**:
1. PR to `github.com/juju/description` adding `DeploymentType string` to `ConstraintsArgs` and `DeploymentType() string` to `Constraints` interface
2. Bump `description` dependency in `go.mod`
3. Update `domain/application/modelmigration/export.go` `exportApplicationConstraints()` to include `DeploymentType`
4. Update `domain/constraints/modelmigration/decode.go` `DecodeConstraints()` to read `DeploymentType`

**Validates**: FR-013

## K8s Provider: Known Gaps (Pre-Existing)

The K8s provider layer (`internal/provider/kubernetes/application/`) already supports all three deployment types, but the following gaps were identified during substrate analysis. These are **pre-existing** and affect all deployment types — they are not introduced by this feature and are not blockers.

| Gap | Severity | Impact | File | Status |
|-----|----------|--------|------|--------|
| ~~`computeStatus()` incomplete~~ | ~~Medium~~ | ~~Returns `NotSupported` for Deployment and DaemonSet~~ | `application.go` | **FIXED** (T045) — implemented for all 3 types |
| ~~`currentScale()` incomplete~~ | ~~Medium~~ | ~~Returns `NotSupported` for Deployment and DaemonSet~~ | `scale.go` | **FIXED** (T044) — implemented for all 3 types |
| `Exists()` single-type check | Low | Only checks the resource matching stored deployment type. Won't detect a stray resource of a different type with the same name. | `application.go` | Open |
| No drift detection | Low | Manual `kubectl` changes to resource type are not detected by Juju. | N/A | Open |
| No cross-type resource validation | Low | If a wrong resource type exists with the same name, `Exists()` returns false rather than an error. | `application.go` | Open |

**Remaining gaps**: `Exists()` single-type check, drift detection, and cross-type validation are low-severity and can be addressed in a follow-up PR.

---

## Storage Adaptation Stories (US7-US11) — Added 2026-02-10

These phases extend the MVP to support persistent and ephemeral storage for Deployment/DaemonSet workloads. They were added after the storage codebase research session and `/speckit.specify` update.

**Prerequisites**: Phases 1-8 (MVP + Pod Recovery) complete.

### Phase 9: Story 7 — Storage Attachment Cleanup on Pod Replacement (P1)

**Goal**: When the worker clears a stale `k8s_pod` entry for a replaced Deployment/DaemonSet pod, also cascade-delete the stale `storage_filesystem_attachment` and `storage_volume_attachment` records for that unit's net_node. This prevents duplicate key errors during re-registration with storage.

**Layers touched**: domain (state)

**Steps**:
1. Extend `ClearCAASUnitCloudContainer()` in `domain/application/state/unit.go` (line ~2180) to add DELETE statements within the existing `db.Txn()` for:
   - `storage_filesystem_attachment WHERE net_node_uuid = (SELECT net_node_uuid FROM unit WHERE name = $unitName)`
   - `storage_volume_attachment WHERE net_node_uuid = (SELECT net_node_uuid FROM unit WHERE name = $unitName)`
2. Add unit tests in `domain/application/state/unit_test.go` verifying that after `ClearCAASUnitCloudContainer`, filesystem/volume attachment rows for the unit are gone while attachments for other units remain.
3. Verify idempotency: calling on a unit with no storage attachments is a no-op.

**Test**: Deploy Deployment with persistent storage → delete a pod → verify replacement pod re-registers without duplicate key errors and remounts storage.

**Validates**: FR-014 | SC-008

**Note**: This also subsumes the S2.3 filesystem race (Difficulty 5 in the storage analysis) — once stale storage attachments are cleaned alongside stale cloud containers, the timing window is eliminated.

---

### Phase 10: Story 8 — PVC Cleanup on Non-StatefulSet Application Removal (P1)

**Goal**: When a Deployment or DaemonSet application is removed, delete all Juju-created standalone PVCs from the K8s namespace. StatefulSet PVC behavior is unchanged.

**Layers touched**: K8s provider

**Steps**:
1. In `Delete()` in `internal/provider/kubernetes/application/application.go` (line ~1199, after DaemonSet listing and before `applier.Delete()`), add PVC listing and deletion:
   - List PVCs using the existing `resourceLabels` selector (`utils.LabelsForAppCreated()` — labels include `app.kubernetes.io/managed-by=juju` and `app.kubernetes.io/name={appName}`)
   - Use `resources.ListPersistentVolumeClaims()` (already available in the codebase)
   - Append matching PVCs to `resourcesToDelete` slice (same pattern as StatefulSets, Services, etc.)
2. Guard with deployment type: only delete PVCs for non-StatefulSet applications. StatefulSet PVCs are managed by K8s VolumeClaimTemplates retention policy.
3. Add unit tests for `Delete()` verifying:
   - Deployment removal: PVCs with Juju labels are deleted
   - DaemonSet removal: PVCs with Juju labels are deleted
   - StatefulSet removal: PVC behavior unchanged (no PVCs deleted by Juju)
4. Address the TODO at line 186: `// TODO: storage handling for deployment/daemonset enhancement.`

**Test**: Deploy Deployment with storage → verify PVCs exist → `juju remove-application` → verify PVCs deleted from namespace.

**Validates**: FR-015 | SC-009

**Caveat**: SC-009 ("zero PVCs within 60s") depends on `Delete()` actually being called. The pre-existing `remove-application` race (documented in spec.md Known Bugs) can prevent this. The race fix is out of scope for this story.

---

### Phase 11: Story 9+10 — Storage Access Mode Validation (P2)

**Goal**: Warn operators at deploy time when storage access modes are incompatible with the workload type. The warning is non-blocking — deployment proceeds.

**Layers touched**: worker (provisioner)

**Steps**:
1. In the provisioning flow in `internal/worker/caasapplicationprovisioner/ops.go` (near line 260, alongside the existing stateless+storage warning), add access mode validation:
   - If `deployment-type=stateless` AND charm declares persistent storage AND access mode is ReadWriteOnce AND scale > 1: emit warning via `logger.Warningf()` to controller logs
   - If `deployment-type=daemon` AND charm declares persistent storage AND access mode is ReadWriteOnce: emit warning (DaemonSet on multi-node → Multi-Attach error)
   - Read access mode from provisioning info storage constraints (do NOT import provider packages — Constitution Principles II and VI)
2. Surface warning to CLI output via application status message:
   - After emitting the logger warning, set a warning-level status message on the application so it appears in `juju status`
   - The warning should appear in both controller logs (via `logger.Warningf()`) and CLI output (via status message)
   - All warning logic stays in the worker layer — the provider layer should NOT contain deployment-type-aware warning logic (Constitution Principles II and VI)
3. Add unit tests verifying:
   - Deployment + RWO + scale=1: no warning
   - Deployment + RWO + scale=3: warning emitted
   - Deployment + RWX + scale=3: no warning
   - DaemonSet + RWO: warning emitted
   - DaemonSet + ephemeral only: no warning
   - StatefulSet + RWO: no warning (RWO is expected per-pod)

**Test**: Deploy storage-bearing charm as Deployment with RWO storage class at scale=3 → verify non-blocking warning in both CLI and logs.

**Validates**: FR-016 | SC-010

**Note**: K8s `StorageClass` objects do not expose supported access modes in their spec. The validation uses the access mode from provisioning info storage constraints (default: ReadWriteOnce unless overridden in storage pool config). The system cannot dynamically query whether a storage class supports RWX — this is documented as a limitation.

---

### Phase 12: Story 11 — Ephemeral Storage for Stateless Workloads (P2)

**Goal**: Verify and ensure that EmptyDir/tmpfs storage works correctly for Deployment and DaemonSet workloads. No PVCs should be created for ephemeral storage.

**Layers touched**: K8s provider (validation), worker (test)

**Steps**:
1. Verify the existing code path: `VolumeSourceForFilesystem()` in `internal/provider/kubernetes/storage/storage.go` (line 57) already returns a non-nil `VolumeSource` for `rootfs` and `tmpfs` provider types, which bypasses PVC creation in `filesystemToVolumeInfo()` (line 2302). Confirm this path works for Deployment and DaemonSet by tracing through the `Ensure()` method.
2. Add validation: when a charm declares only ephemeral storage and is deployed as a Deployment/DaemonSet, confirm no PVCs are created in the namespace.
3. Verify pod replacement: when a Deployment pod with ephemeral storage is replaced, the replacement gets a fresh EmptyDir volume and re-registers without storage attachment errors (since ephemeral volumes have no attachment rows).
4. Add unit tests verifying:
   - Deployment + rootfs storage: EmptyDir volume created, no PVCs
   - DaemonSet + tmpfs storage: EmptyDir with Memory medium, no PVCs
   - Mixed (persistent + ephemeral): persistent gets PVC, ephemeral gets EmptyDir
5. Verify that `ClearCAASUnitCloudContainer` (extended in Phase 9) handles the case where a unit has no storage attachments gracefully (no-op for the storage DELETE statements).

**Test**: Deploy charm with tmpfs storage as Deployment → verify EmptyDir mounts → scale to 3 → verify each pod has independent ephemeral volume → delete pod → verify replacement gets fresh volume.

**Validates**: FR-017 | SC-011

---

### Phase 13: Story 8 Supplement — PVC Stability During Scaling (P2)

**Goal**: Verify that scale operations on Deployment/DaemonSet do not create or delete PVCs. The shared PVC remains stable across scaling.

**Layers touched**: K8s provider (validation), worker (test)

**Steps**:
1. Verify that `Ensure()` in the K8s provider for Deployment/DaemonSet does not call `handlePVCForStatelessResource` on scale changes (only on initial creation). Trace the `Ensure()` flow for an existing application to confirm PVC creation is skipped when PVCs already exist.
2. Verify scale-down: reducing replicas from 3 to 1 does not delete the shared PVC. The PVC remains for the surviving pod.
3. Verify scale-up after scale-down: scaling from 1 back to 3, new pods mount the existing shared PVC without creating new PVCs.
4. Add unit tests verifying:
   - Scale up from 1 to 3: no new PVCs created
   - Scale down from 3 to 1: no PVCs deleted
   - Scale up after scale-down: existing PVC reused

**Test**: Deploy Deployment with storage at scale=1 → scale to 3 → verify single PVC → scale to 1 → verify PVC remains → scale to 3 → verify same PVC.

**Validates**: FR-018

---

## Updated Dependency Graph

```
Phase 1 (Setup)
    └──> Phase 2 (Foundational)
              └──> Phase 3 (US1+2+4: Core feature) 🎯 MVP
                        ├──> Phase 3a (Provider: currentScale + computeStatus)
                        ├──> Phase 3b (Unit registration: naming + scaling gate)
                        ├──> Phase 4 (US3: DaemonSet scale blocking)
                        ├──> Phase 5 (US5: Status visibility)
                        └──> Phase 6 (Migration) ⚠️ T040-T043 = RELEASE BLOCKER
                                    └──> Phase 7 (Pod Recovery)
                                                └──> Phase 8 (Resilience Testing)
                                                          └──> Phase 9 (US7: Storage attachment cleanup) ← P1
                                                                    ├──> Phase 10 (US8: PVC cleanup on removal) ← P1
                                                                    │         └──> Phase 13 (US8: PVC scaling stability) ← P2
                                                                    ├──> Phase 11 (US9+10: Access mode validation) ← P2
                                                                    └──> Phase 12 (US11: Ephemeral storage) ← P2
```

**Phase 9 is the gate** for all storage stories — it fixes the storage attachment cleanup that all other stories depend on.

**Phases 10, 11, 12 are independent** of each other and can run in parallel after Phase 9. **Phase 13 depends on Phase 10** (PVC lifecycle must be understood before testing scaling stability).

---

## Complexity Tracking

No constitution violations. All changes use existing patterns.

## Files Modified (Storage Stories — ~8 additional)

### Domain Layer (Phase 9)
- `domain/application/state/unit.go` — Extend `ClearCAASUnitCloudContainer` with storage attachment DELETEs
- `domain/application/state/unit_test.go` — Tests for cascaded storage cleanup

### K8s Provider Layer (Phases 10, 12, 13)
- `internal/provider/kubernetes/application/application.go` — Add PVC listing+deletion to `Delete()`
- `internal/provider/kubernetes/application/application_test.go` — Tests for PVC cleanup and ephemeral storage

### Worker Layer (Phase 11)
- `internal/worker/caasapplicationprovisioner/ops.go` — Access mode warnings alongside existing stateless+storage warning
- `internal/worker/caasapplicationprovisioner/ops_test.go` — Tests for access mode warning logic

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
