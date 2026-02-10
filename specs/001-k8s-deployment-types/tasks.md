# Tasks: Kubernetes Deployment and DaemonSet Workload Types

**Input**: Design documents from `/specs/001-k8s-deployment-types/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story. Tests are included within implementation tasks (not as separate TDD tasks) since no explicit TDD approach was requested.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1+2+4, US3, US5)
- Exact file paths included in all descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Database schema changes and core type additions that all stories depend on.

- [x] T001 Create DB PATCH file with `deployment_type` lookup table and `deployment_type_id` column on `application` table in `domain/schema/model/sql/0046-deployment-type.PATCH.sql`
- [x] T002 Register new PATCH file in `domain/schema/model.go` via `modelPostPatchFilesByVersion`

**Checkpoint**: Schema changes ready — domain and upper layers can now reference deployment type.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core constraint field and domain type changes that ALL user stories depend on.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [x] T003 [P] Add `DeploymentType *string` field to `Value` struct in `core/constraints/constraints.go` with parsing, validation (values: `stateless`, `stateful`, `daemon`), and serialization support
- [x] T004 [P] Add `DeploymentType` field to `AddCAASApplicationArg` struct in `domain/application/types.go`
- [x] T005 Map new `deployment-type` constraint in `domain/constraints/constraints.go`
- [x] T006 Persist `deployment_type_id` during `CreateCAASApplication` in `domain/application/state/application.go` (read from `AddCAASApplicationArg`, map to lookup table ID, INSERT into `application` row)
- [x] T007 Add `GetApplicationDeploymentType(ctx, appName) (string, error)` method to application service in `domain/application/service/application.go` (query `deployment_type` join from `application` table)

**Checkpoint**: Foundation ready — constraint parsing, persistence, and retrieval all functional. User story implementation can now begin.

---

## Phase 3: User Story 1+2+4 — Foundation + Constraint + Backward Compat (Priority: P1) 🎯 MVP

**Goal**: Charms deploy with the correct K8s workload type based on constraint or inference, while all existing charms continue as StatefulSet.

**Independent Test**: Deploy a charm with no storage → verify Deployment. Deploy with storage → verify StatefulSet. Deploy with explicit constraint → verify override works. Deploy existing charm → verify StatefulSet preserved.

**Validates**: FR-001, FR-002, FR-003, FR-004, FR-006, FR-008, FR-009, FR-010, FR-011, FR-012 | SC-001, SC-003, SC-005, SC-007

### Wire Types & API Versioning

- [x] T008 [P] [US1+2+4] Add `DeploymentType string` field to `CAASApplicationProvisioningInfo` struct in `rpc/params/caas.go` (or the file containing this struct) with JSON tag `"deployment-type,omitempty"`
- [x] T009 [P] [US1+2+4] Bump CAASApplicationProvisioner facade version from v1 to v2 in `apiserver/facades/controller/caasapplicationprovisioner/register.go`
- [x] T010 [P] [US1+2+4] Update client facade version for `CAASApplicationProvisioner` to 2 in `api/facadeversions.go`

### Facade: Populate Deployment Type in Provisioning Info

- [x] T011 [US1+2+4] Populate `DeploymentType` field in provisioning info response by calling `GetApplicationDeploymentType()` in `apiserver/facades/controller/caasapplicationprovisioner/provisioner.go` (within the ProvisioningInfo assembly method)

### API Client: Parse New Field

- [x] T012 [US1+2+4] Parse `DeploymentType` from provisioning info response in `api/controller/caasapplicationprovisioner/client.go`

### Worker: Dynamic Deployment Type Selection

- [x] T013 [US1+2+4] Implement `DetermineDeploymentType(constraint *string, hasStorage bool) caas.DeploymentType` function in `internal/worker/caasapplicationprovisioner/ops.go` — logic: (1) if constraint set → map to caas type, (2) if charm has storage → `DeploymentStateful`, (3) else → `DeploymentStateless`
- [x] T014 [US1+2+4] Replace hardcoded `caas.DeploymentStateful` with call to `DetermineDeploymentType()` using provisioning info in `internal/worker/caasapplicationprovisioner/application.go:149`
- [x] T015 [US1+2+4] Replace hardcoded `caas.DeploymentStateful` with deployment type from state in `internal/worker/caasfirewaller/appfirewaller.go:81`

### Domain: Replace Hardcoded StatefulSet in Provider Service

- [x] T016 [P] [US1+2+4] Replace hardcoded `caas.DeploymentStateful` at line 442 in `domain/application/service/provider.go` with deployment type read from application state
- [x] T017 [P] [US1+2+4] Replace hardcoded `caas.DeploymentStateful` at line 538 in `domain/application/service/provider.go` with deployment type read from application state

### Immutability Enforcement (FR-006)

- [x] T018 [US1+2+4] Add `DeploymentTypeImmutable` error constant to `domain/application/errors/errors.go` with message: "deployment type cannot be changed for a running application; redeploy to use a different workload type"
- [x] T019 [US1+2+4] Add deployment type immutability check in `domain/application/service/provider.go` — when `SetApplicationConstraints` is called, if the new constraints include `deployment-type` and differ from the persisted value, return `DeploymentTypeImmutable` error

### Warning for Storage Mismatch (FR-012)

- [x] T020 [US1+2+4] Add warning log when `deployment-type=stateless` but charm declares persistent storage, emitted during deploy in the worker or domain service layer (exact location: near `DetermineDeploymentType` usage in `internal/worker/caasapplicationprovisioner/application.go`)

**Checkpoint**: At this point, deploying charms with any workload type works end-to-end. Existing charms continue as StatefulSet. Constraint validation, inference, persistence, and provisioning all functional.

---

## Phase 4: User Story 3 — DaemonSet Scale Blocking (Priority: P2)

**Goal**: Operators can deploy charms as DaemonSets with proper scale blocking. Manual scale operations are rejected with a clear error.

**Independent Test**: Deploy with `deployment-type=daemon` → verify DaemonSet created. Try `juju scale-application` → verify clear error. Add node to cluster → verify new pod appears.

**Validates**: FR-005 | SC-002, SC-006

**Depends on**: Phase 3 (deployment type must be persistable and queryable)

### Implementation

- [x] T021 [P] [US3] Add `DaemonSetScaleNotSupported` error constant to `domain/application/errors/errors.go` with message: "scaling is not supported for DaemonSet applications; scale is determined by the number of cluster nodes"
- [ ] T022 [US3] Add deployment type check in scale validation in `domain/application/service/application.go` — before processing scale change, query application deployment type; if `daemon`, return `DaemonSetScaleNotSupported` error

**Checkpoint**: DaemonSet applications correctly reject manual scaling. Combined with Phase 3, the full DaemonSet workflow (deploy + scale blocking) is testable.

---

## Phase 5: User Story 5 — Status Visibility (Priority: P2)

**Goal**: Operators can see the workload type for each K8s application in `juju status` output — both in the summary table and per-application detail.

**Independent Test**: Deploy applications with different workload types → run `juju status` → verify "Type" column shows correct values for CAAS model. Verify IAAS model has no Type column.

**Validates**: FR-007 | SC-004

**Depends on**: Phase 3 (deployment type must be persisted and queryable)

### Domain & Wire Types

- [ ] T023 [P] [US5] Add `DeploymentType *string` field to `Application` struct in `domain/status/service/types.go`
- [ ] T024 [P] [US5] Add `DeploymentType string` field to `ApplicationStatus` struct in `rpc/params/status.go` with JSON tag `"deployment-type,omitempty"`
- [ ] T025 [P] [US5] Add `DeploymentType string` field to `applicationStatus` struct in `cmd/juju/status/formatted.go`

### Status Assembly (API Server)

- [ ] T026 [US5] Populate `DeploymentType` in CAAS application status assembly in `apiserver/facades/client/client/status.go` (within the CAAS-specific block around line 940-956) by reading from the domain status Application struct

### Status Display (CLI)

- [ ] T027 [US5] Map `DeploymentType` from `params.ApplicationStatus` to `applicationStatus` in `cmd/juju/status/formatter.go`
- [ ] T028 [US5] Add "Type" column header to CAAS application table headers at `cmd/juju/status/output_tabular.go:142` (after "Exposed", before "Message")
- [ ] T029 [US5] Print deployment type value (display names: "Deployment", "StatefulSet", "DaemonSet") in the CAAS application table rendering loop in `cmd/juju/status/output_tabular.go`

**Checkpoint**: `juju status` shows workload type for all CAAS applications. IAAS models unaffected.

---

## Phase 6: Migration Support (Priority: P1 — Release Blocker)

**Goal**: Deployment type survives model migration between controllers.

**Depends on**: Phase 3 (deployment type must be persisted and queryable)

### Implemented (in-tree)

- [x] T036 [P] [Migration] Fix YAML constraint validation bypass — route `UnmarshalYAML` through `setDeploymentType()` in `core/constraints/constraints.go`
- [x] T037 [P] [Migration] Add `DeploymentType string` field to `InsertApplicationArgs` in `domain/application/types.go`
- [x] T038 [Migration] Set `DeploymentTypeID` from `args.DeploymentType` in `InsertMigratingApplication` in `domain/application/state/migration.go`
- [x] T039 [Migration] Re-infer deployment type from charm metadata during CAAS import when no explicit constraint is set in `domain/application/service/migration.go`

### External Dependency (RELEASE BLOCKER)

- [ ] T040 [Migration] PR to `github.com/juju/description`: add `DeploymentType string` to `ConstraintsArgs` and `DeploymentType() string` to `Constraints` interface
- [ ] T041 [Migration] Bump `description` dependency in `go.mod` after T040 merges
- [ ] T042 [Migration] Update `exportApplicationConstraints()` in `domain/application/modelmigration/export.go` to include `DeploymentType` field
- [ ] T043 [Migration] Update `DecodeConstraints()` in `domain/constraints/modelmigration/decode.go` to read `DeploymentType` from description constraints

**Checkpoint**: After T040–T043, all deployment types (including explicit `daemon` and constraint overrides) survive migration round-trip. Without T040–T043, only inferred stateless/stateful types are preserved.

---

## Cross-Cutting Tests (Post-Implementation Validation)

**Purpose**: Validation, cleanup, and verification across all stories. Run after all implementation phases are complete.

### Test Runs

- [ ] T030 Run `go test ./core/constraints/...` to verify constraint parsing, validation, and serialization
- [ ] T031 Run `go test ./domain/application/...` to verify persistence, retrieval, scale validation, and immutability enforcement
- [ ] T032 Run `go test ./internal/worker/caasapplicationprovisioner/...` to verify deployment type determination and dynamic provisioning
- [ ] T033 Run `go test ./apiserver/facades/controller/caasapplicationprovisioner/...` to verify facade v2 and provisioning info population
- [ ] T034 Run `go test ./cmd/juju/status/...` to verify status display with Type column
- [ ] T035 Run quickstart.md validation: build with `make go-build` and verify no compilation errors

### New Tests Required (Constitution IV — Test Discipline)

- [ ] T055 [US1+2+4] Add unit tests for `RegisterCAASUnit` with `OrderedScale=false` in `domain/application/state/state_test.go` — verify Deployment/DaemonSet units can register when `ordinal < appScale.Scale` even when `Scaling=false`; verify rejection when `ordinal >= appScale.Scale`
- [ ] T056 [US1+2+4] Add unit tests for `GetCAASUnitNameByProviderID` in `domain/application/state/state_test.go` — verify lookup returns correct unit name for known provider ID; returns `(empty, false, nil)` for unknown provider ID
- [ ] T057 [US1+2+4] Add unit tests for `GetNextCAASUnitOrdinal` in `domain/application/state/state_test.go` — verify returns 0 for app with no units; returns `max+1` for app with existing units; handles gaps correctly
- [ ] T058 [US1+2+4] Add unit tests for `RegisterCAASUnit` service-layer branching in `domain/application/service/provider_test.go` — verify StatefulSet derives ordinal from pod name; Deployment looks up by provider ID then falls back to next ordinal

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 (schema must exist before persistence code)
- **Story 1+2+4 (Phase 3)**: Depends on Phase 2 (constraint + domain types must exist)
- **Provider Fixes (Phase 3a)**: Depends on Phase 3 (deployment type must flow through to provider)
- **Unit Registration (Phase 3b)**: Depends on Phase 3 (deployment type must be queryable in domain layer)
- **Story 3 (Phase 4)**: Depends on Phase 3 (deployment type must be persistable/queryable)
- **Story 5 (Phase 5)**: Depends on Phase 3 (deployment type must be persisted)
- **Migration (Phase 6)**: Depends on Phase 3; T040–T043 are a **release blocker** (external dep on `description` library)
- **Cross-Cutting Tests**: Depends on all implementation phases being complete (T030-T035 run after Phases 3-6; T055-T058 run after Phase 3b)

### User Story Dependencies

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
                                                          └──> Phase 9 (US7: Storage attachment cleanup) ← P1 GATE
                                                                    ├──> Phase 10 (US8: PVC cleanup) ← P1
                                                                    │         └──> Phase 13 (US8: PVC scaling)
                                                                    ├──> Phase 11 (US9+10: Access mode) ← P2
                                                                    └──> Phase 12 (US11: Ephemeral) ← P2
                                                                    Phases 10-13 all complete ──> Phase 14 (Test runs)
```

- **Story 3 and Story 5 are independent of each other** — they can run in parallel after Phase 3
- **Story 1+2+4 is the MVP** — once complete, the core feature works end-to-end
- **Phase 8 depends on Phase 7** — resilience testing validates pod recovery + all prior phases
- **Phase 9 is the gate** for all storage stories — extends existing pod recovery with storage cleanup
- **Phases 10, 11, 12 are independent** — can run in parallel after Phase 9

### Within Each Phase

- Tasks marked [P] can run in parallel (different files, no dependencies)
- Sequential tasks must complete in order (earlier tasks establish types/methods used by later tasks)

### Parallel Opportunities

**Phase 2** (3 parallel groups):
```
T003 (core/constraints) ─┐
T004 (domain/types)     ─┤── parallel (different files)
                          └──> T005 (domain/constraints mapping) ──> T006 (state) ──> T007 (service)
```

**Phase 3** (parallel within sub-groups):
```
T008 (rpc/params)     ─┐
T009 (facade register) ┤── parallel (different files)
T010 (api versions)   ─┘
         └──> T011 (facade populate) ──> T012 (api client) ──> T013 (worker ops) ──> T014 (worker app)
                                                                                  ──> T015 (firewaller)
T016 (provider.go:442) ─┐
T017 (provider.go:538) ─┘── parallel (same file but independent changes)
```

**Phase 4+5** (independent stories):
```
Phase 4 (US3: T021, T022) ─┐
Phase 5 (US5: T023-T029)  ─┘── parallel (different layers, no dependencies)
```

---

## Implementation Strategy

### MVP First (Story 1+2+4 Only)

1. Complete Phase 1: Setup (schema)
2. Complete Phase 2: Foundational (constraints, types, persistence)
3. Complete Phase 3: Story 1+2+4 (wire types, facade, worker, provider)
4. Complete Phase 3a: Provider fixes (currentScale, computeStatus for Deployment/DaemonSet)
5. Complete Phase 3b: Unit registration refactor (naming, scaling gate for Deployment/DaemonSet)
6. **STOP and VALIDATE**: Deploy charms with different types, verify inference + constraint override + backward compat
7. Run `go test` for core, domain, worker, and facade packages

### Incremental Delivery

1. Phases 1+2+3+3a+3b → Core feature works end-to-end → Test independently → **MVP**
2. Add Phase 4 (Story 3) → DaemonSet scale blocking → Test independently
3. Add Phase 5 (Story 5) → Status visibility → Test independently
4. Phase 6 → Full validation pass
5. Each story adds value without breaking previous stories

### Storage Adaptation (Post-MVP)

6. Phase 9 (US7) → Storage attachment cleanup → **GATE** for all storage work
7. Phases 10 + 11 + 12 in parallel → PVC cleanup + access mode warnings + ephemeral validation
8. Phase 13 → PVC scaling stability verification
9. Phase 14 → Full test run across all storage-modified packages

---

## Phase 3a: Provider Layer Fixes (Discovered During Integration Testing)

**Purpose**: The K8s provider layer already had scaffolding for all 3 deployment types, but key methods only had full implementations for StatefulSet — `currentScale()` and `computeStatus()` returned `NotSupported` for Deployment/DaemonSet. These are required for the feature to work end-to-end.

- [x] T044 [P] [US1+2+4] Add `DeploymentStateless` and `DeploymentDaemon` cases to `currentScale()` in `internal/provider/kubernetes/application/scale.go` — Deployment reads `*d.Spec.Replicas`; DaemonSet reads `ds.Status.DesiredNumberScheduled`
- [x] T045 [P] [US1+2+4] Add `DeploymentStateless` and `DeploymentDaemon` cases to `computeStatus()` in `internal/provider/kubernetes/application/application.go` — checks deletion timestamp, ready replicas, and warning events (same pattern as StatefulSet)

**Checkpoint**: Provider layer now correctly reports scale and status for all 3 workload types.

---

## Phase 3b: Unit Registration Refactor (Discovered During Integration Testing)

**Purpose**: StatefulSet pods have predictable names (`<app>-<ordinal>`), so unit registration could derive the unit ordinal from the pod name. Deployment/DaemonSet pods have random suffixes (e.g., `nginx-759b4f4b68-5mk8l`), requiring a different strategy for unit naming and a relaxed scaling gate for unit registration.

### State Layer: New Queries for Non-StatefulSet Unit Registration

- [x] T046 [P] [US1+2+4] Add `GetCAASUnitNameByProviderID(ctx, appUUID, providerID) (unitName, found, error)` to `domain/application/state/unit.go` — queries `unit JOIN k8s_pod` to find an existing unit by its cloud container provider ID (for idempotent re-registration after pod restart)
- [x] T047 [P] [US1+2+4] Add `GetNextCAASUnitOrdinal(ctx, appName) (int, error)` to `domain/application/state/unit.go` — queries all existing unit names, parses their ordinal suffixes, and returns `max + 1` (or 0 if no units exist)

### Service Layer: Deployment-Type-Aware Unit Registration

- [x] T048 [US1+2+4] Refactor `RegisterCAASUnit()` in `domain/application/service/provider.go` to branch on deployment type: StatefulSet derives ordinal from pod name; Deployment/DaemonSet looks up existing unit by provider ID (T046) or assigns next ordinal (T047). Set `OrderedScale = (deploymentType == "stateful")` on the register args.

### State Layer: Relaxed Scaling Gate for Non-StatefulSet

- [x] T049 [US1+2+4] Modify `RegisterCAASUnit` scaling gate in `domain/application/state/unit.go` to differentiate by `OrderedScale` flag: StatefulSet requires `appScale.Scaling == true` AND `ordinal < ScaleTarget` (strict gate); Deployment/DaemonSet only requires `ordinal < appScale.Scale` (relaxed gate — pods start before `EnsureScale` sets `Scaling=true`)

**Checkpoint**: Init containers in Deployment/DaemonSet pods can successfully register units via UnitIntroduction without being blocked by the StatefulSet-specific scaling gate.

---

## Phase 7: Pod Recovery and Resilience (User Story 6)

**Purpose**: Fix Deployment/DaemonSet pod replacement — when K8s replaces a pod with a new random name, the stale k8s_pod entry blocks re-registration. The worker-side reconciliation detects and clears stale entries.

### State Layer: Clear Stale Cloud Container

- [x] T050 [US6] Add `ClearCAASUnitCloudContainer(ctx, unitName)` to `domain/application/state/unit.go` — deletes k8s_pod_port and k8s_pod rows for the given unit name. Add tests in `domain/application/state/unit_test.go`.

### Service Layer: Expose Clear Method

- [x] T051 [US6] Add `ClearCAASUnitCloudContainer` to `UnitState` interface in `domain/application/service/unit.go` and add delegating `Service` method with validation. Add mock-based tests in `domain/application/service/unit_test.go`.

### Worker Layer: Stale Pod Reconciliation

- [x] T052 [US6] Add `ClearCAASUnitCloudContainer` to `ApplicationService` interface in `internal/worker/caasapplicationprovisioner/worker.go`.
- [x] T053 [US6] Add stale pod detection in `updateState()` in `internal/worker/caasapplicationprovisioner/ops.go` — after building `unitToPod` and querying K8s pods, identify units whose provider_id doesn't match any active pod and call `ClearCAASUnitCloudContainer` for each. Add tests in `ops_test.go`.

### Mock Regeneration

- [x] T054 [US6] Regenerate mocks: `go generate ./domain/application/service/...` and `go generate ./internal/worker/caasapplicationprovisioner/...`

**Checkpoint**: Deployment/DaemonSet pod replacement self-heals — the worker clears stale k8s_pod entries and the agent's retry succeeds via step 2 (GetUnassignedCAASUnitName).

---

## Phase 8: Resilience Testing (User Story 6 + Regression Guard)

**Purpose**: Systematic end-to-end verification of the MVP under stress, chaos, and lifecycle churn. Every scenario is executed for both Deployment and StatefulSet to guard against regressions.

**Test plan**: See [`resilience-testing.md`](resilience-testing.md) for the full scenario matrix.

### Juju Lifecycle Scenarios

- [ ] T060 [US1+4+6] S1.1-S1.6: Deploy, scale up (1->3), scale down (3->1), scale up (1->2), remove, redeploy — verify ordinal reset, correct unit count, clean lifecycle. Run for both Deployment and StatefulSet.

### Substrate Chaos Scenarios

- [ ] T061 [US6] S2.1-S2.3: Single pod deletion at scale=1, single pod deletion at scale=3, all pods deleted at scale=3 — verify unit recovery, no phantom units. Run for both Deployment and StatefulSet.
- [ ] T062 [US6] S2.4-S2.6: Rapid pod cycling, pod deletion during scale-up, pod deletion during scale-down — verify convergence to correct state. Run for both Deployment and StatefulSet.

### Removal and Redeployment Scenarios

- [ ] T063 [US6] S3.1-S3.3: Remove and redeploy (clean cycle), remove scaled app and redeploy, redeploy as different type — verify ordinal reset, correct K8s resource type. Run for both Deployment and StatefulSet.

### Worker Restart Scenarios

- [ ] T064 [US6] S4.1-S4.2: Controller jujud restart during normal operation, controller restart combined with pod deletion — verify no state loss, unit recovery. Run for both Deployment and StatefulSet.

### Edge Case Scenarios

- [ ] T065 [US6] S5.1-S5.3: Scale to 0 and back, rapid scale oscillation, kill pod during startup — verify convergence, no orphaned units. Run for both Deployment and StatefulSet.

**Checkpoint**: All 20 scenarios in the execution matrix pass for both Deployment and StatefulSet.

---

## Phase 9: User Story 7 — Storage Attachment Cleanup on Pod Replacement (Priority: P1)

**Goal**: When the worker clears a stale `k8s_pod` entry for a replaced Deployment/DaemonSet pod, also cascade-delete the stale `storage_filesystem_attachment` and `storage_volume_attachment` records for that unit's net_node. This prevents duplicate key errors during re-registration with storage.

**Independent Test**: Deploy Deployment with persistent storage → delete pod → verify replacement pod re-registers without duplicate key errors and remounts storage.

**Validates**: FR-014 | SC-008

**Depends on**: Phase 7 (T050-T054 must be complete — `ClearCAASUnitCloudContainer` must exist before extending it)

### State Layer: Cascade Storage Attachment Cleanup

- [ ] T070 [US7] Extend `ClearCAASUnitCloudContainer()` in `domain/application/state/unit.go` (line ~2180) to add DELETE statements within the existing `db.Txn()` block for `storage_filesystem_attachment` rows where `net_node_uuid` matches the unit's net_node, and `storage_volume_attachment` rows where `net_node_uuid` matches the unit's net_node. Use the same subquery pattern already used for `k8s_pod` deletion: `WHERE net_node_uuid = (SELECT net_node_uuid FROM unit WHERE name = $unitName.name)`. Ensure DELETE order respects foreign key constraints (filesystem/volume attachments before any parent rows).

- [ ] T071 [US7] Add unit tests in `domain/application/state/unit_test.go` for the extended `ClearCAASUnitCloudContainer`:
  - Test that after clearing, `storage_filesystem_attachment` rows for the target unit's net_node are deleted while rows for other units remain.
  - Test that after clearing, `storage_volume_attachment` rows for the target unit's net_node are deleted while rows for other units remain.
  - Test idempotency: calling on a unit with no storage attachments completes without error.
  - Test that `k8s_pod` and `k8s_pod_port` rows are still deleted (no regression from T050).

- [ ] T072 [US7] Regenerate mocks if the state interface signature changed: `go generate ./domain/application/service/...`

**Checkpoint**: `ClearCAASUnitCloudContainer` now atomically clears cloud container AND storage attachment records. Replacement pods can re-register with fresh storage bindings.

---

## Phase 10: User Story 8 — PVC Cleanup on Non-StatefulSet Application Removal (Priority: P1)

**Goal**: When a Deployment or DaemonSet application is removed, delete all Juju-created standalone PVCs from the K8s namespace. StatefulSet PVC behavior is unchanged.

**Independent Test**: Deploy Deployment with storage → verify PVCs exist → `juju remove-application` → verify PVCs deleted from namespace.

**Validates**: FR-015 | SC-009

**Depends on**: Phase 9 (storage attachment cleanup must work before testing full PVC lifecycle)

### K8s Provider: PVC Deletion in Delete()

- [ ] T073 [US8] In `Delete()` in `internal/provider/kubernetes/application/application.go` (after the DaemonSet listing block around line ~1199, before `applier.Delete()` at line ~1210), add PVC listing and deletion for non-StatefulSet workloads:
  - List PVCs using `resources.ListPersistentVolumeClaims()` with the existing `resourceLabels` selector (same `utils.LabelsForAppCreated()` labels used for all other resources)
  - Append matching PVCs to the `resourcesToDelete` slice (same pattern as StatefulSets, Services, Secrets, etc.)
  - Guard: only list and delete PVCs when `a.deploymentType != caas.DeploymentStateful` — StatefulSet PVCs are managed by K8s VolumeClaimTemplates retention policy

- [ ] T074 [US8] Remove or update the TODO comment at `internal/provider/kubernetes/application/application.go:186`: `// TODO: storage handling for deployment/daemonset enhancement.` — replace with a brief comment noting that PVC cleanup is handled in `Delete()`.

- [ ] T075 [P] [US8] Add unit tests in `internal/provider/kubernetes/application/application_test.go` for PVC cleanup in `Delete()`:
  - Test Deployment removal: PVCs with Juju app labels are listed and deleted
  - Test DaemonSet removal: PVCs with Juju app labels are listed and deleted
  - Test StatefulSet removal: no PVC listing or deletion occurs (PVCs unchanged)
  - Test idempotency: calling Delete() when no PVCs exist completes without error

**Checkpoint**: `juju remove-application` for Deployment/DaemonSet now cleans up standalone PVCs. StatefulSet behavior unchanged.

**Caveat**: SC-009 ("zero PVCs within 60s") depends on `Delete()` being called. The pre-existing `remove-application` race can prevent this. The race fix is out of scope (see spec.md Known Bugs).

---

## Phase 11: User Story 9+10 — Storage Access Mode Validation (Priority: P2)

**Goal**: Warn operators at deploy time when storage access modes are incompatible with the workload type. The warning is non-blocking — deployment proceeds.

**Independent Test**: Deploy storage-bearing charm as Deployment with RWO storage at scale=3 → verify non-blocking warning in both CLI and controller logs.

**Validates**: FR-016 | SC-010

**Depends on**: Phase 9 (storage foundation must be in place)

### Worker: Access Mode Warning

- [ ] T076 [US9+10] In `internal/worker/caasapplicationprovisioner/ops.go` (near line 260, alongside the existing stateless+storage warning), add access mode validation:
  - For each storage declaration in `pi.CharmMeta.Storage`, determine the access mode from the charm storage metadata and Juju storage pool config available in provisioning info. The access mode default is `ReadWriteOnce` unless explicitly overridden in the storage pool. **Do NOT import** `internal/provider/kubernetes/storage/` — that violates Constitution Principles II and VI. Instead, read the access mode from the storage constraints already available in provisioning info (e.g., `pi.Filesystems[].Attributes["storage-mode"]`), or define a simple helper in the worker package itself.
  - If `pi.DeploymentType == "stateless"` AND access mode is `ReadWriteOnce` AND requested scale > 1: emit `logger.Warningf()` with message: "application %q uses deployment-type=stateless with ReadWriteOnce storage %q at scale %d; pods on different nodes may fail to mount. Consider using a ReadWriteMany storage class"
  - If `pi.DeploymentType == "daemon"` AND access mode is `ReadWriteOnce`: emit `logger.Warningf()` with message: "application %q uses deployment-type=daemon with ReadWriteOnce storage %q; pods on nodes other than the PV-bound node will fail to mount. Consider using a ReadWriteMany storage class or ephemeral storage"
  - Skip the check if storage provider type is `rootfs` or `tmpfs` (ephemeral — no PVC involved)

### CLI: Surface Warning to Operator

- [ ] T077 [US9+10] Surface the access mode warning to the CLI via the application's status message. The T020 stateless+storage warning is only emitted via `logger.Warningf()` to controller logs — CLI users don't see it. To surface to operators: after emitting the logger warning in T076, call `SetOperatorStatus()` with a warning-level status message containing the access mode text (application-level warning, not per-unit). This makes the warning visible in `juju status` output. Location: `internal/worker/caasapplicationprovisioner/ops.go` (same function as T076, immediately after the logger call).

### Tests

- [ ] T079 [P] [US9+10] Add unit tests in `internal/worker/caasapplicationprovisioner/ops_test.go` for access mode validation:
  - Deployment + RWO + scale=1: no warning emitted
  - Deployment + RWO + scale=3: warning emitted containing "ReadWriteOnce" and "ReadWriteMany"
  - Deployment + RWX + scale=3: no warning emitted
  - DaemonSet + RWO: warning emitted containing "ReadWriteOnce"
  - DaemonSet + ephemeral (tmpfs): no warning emitted
  - StatefulSet + RWO: no warning emitted (RWO is expected per-pod for StatefulSet)

**Checkpoint**: Operators deploying Deployment/DaemonSet with RWO storage see a clear non-blocking warning in both controller logs and CLI output.

**Note**: K8s `StorageClass` objects do not expose supported access modes. The validation checks the access mode from Juju storage pool config (default: ReadWriteOnce). The system cannot dynamically determine if a storage class supports RWX.

---

## Phase 12: User Story 11 — Ephemeral Storage for Stateless Workloads (Priority: P2)

**Goal**: Verify and ensure that EmptyDir/tmpfs storage works correctly for Deployment and DaemonSet workloads without creating PVCs.

**Independent Test**: Deploy charm with tmpfs storage as Deployment → verify EmptyDir mounts → scale to 3 → verify each pod has independent ephemeral volume → delete pod → verify replacement gets fresh volume.

**Validates**: FR-017 | SC-011

**Depends on**: Phase 9 (extended `ClearCAASUnitCloudContainer` must handle units with no storage attachments)

### Verification and Validation

- [ ] T080 [US11] Trace the existing code path to confirm EmptyDir/tmpfs works for Deployment/DaemonSet: `VolumeSourceForFilesystem()` in `internal/provider/kubernetes/storage/storage.go` (line 57) returns non-nil `VolumeSource` for `rootfs` and `tmpfs` → `filesystemToVolumeInfo()` in `internal/provider/kubernetes/application/application.go` (line ~2302) bypasses PVC creation when `VolumeSource != nil` → `Ensure()` applies the pod spec with EmptyDir volumes. Verify no code path differences exist between StatefulSet and Deployment/DaemonSet for this flow. If differences exist, document and fix them.

- [ ] T081 [US11] Verify that the `ClearCAASUnitCloudContainer` extension (T070) handles units with no storage attachments gracefully — the DELETE statements for `storage_filesystem_attachment` and `storage_volume_attachment` should be no-ops when no rows match the unit's net_node. This is critical for ephemeral-only workloads where pods are replaced but no storage attachment records exist.

### Tests

- [ ] T082 [P] [US11] Add unit tests in `internal/provider/kubernetes/application/application_test.go` for ephemeral storage with Deployment/DaemonSet:
  - Deployment + rootfs storage: EmptyDir volume created in pod spec, no PVCs created
  - DaemonSet + tmpfs storage: EmptyDir with Memory medium in pod spec, no PVCs created
  - Mixed (persistent + ephemeral): persistent storage gets shared PVC, ephemeral gets per-pod EmptyDir

- [ ] T083 [P] [US11] Add unit test in `domain/application/state/unit_test.go` verifying `ClearCAASUnitCloudContainer` completes without error when the unit has no `storage_filesystem_attachment` or `storage_volume_attachment` rows (ephemeral-only unit scenario)

**Checkpoint**: EmptyDir/tmpfs storage confirmed working for all deployment types. No PVCs created. Pod replacement creates fresh ephemeral volumes.

---

## Phase 13: User Story 8 Supplement — PVC Stability During Scaling (Priority: P2)

**Goal**: Verify that scale operations on Deployment/DaemonSet do not create or delete PVCs. The shared PVC remains stable regardless of replica count changes.

**Independent Test**: Deploy Deployment with storage at scale=1 → scale to 3 → verify single PVC → scale to 1 → verify PVC remains → scale to 3 → verify same PVC.

**Validates**: FR-018

**Depends on**: Phase 10 (PVC lifecycle must be understood before testing scaling behavior)

### Verification and Validation

- [ ] T084 [US8] Trace the `Ensure()` flow in `internal/provider/kubernetes/application/application.go` for Deployment/DaemonSet when an application already exists (scale change). Verify that:
  - `handlePVCForStatelessResource` uses `Apply()` semantics (create-or-update via `resources.NewPersistentVolumeClaim`), making re-calls on existing PVCs a no-op
  - Scale changes only modify the Deployment's `replicas` field, not PVC configuration
  - Scale-down does NOT trigger PVC deletion (PVCs are only deleted in `Delete()`, not during scale operations)

### Tests

- [ ] T085 [P] [US8] Add unit tests in `internal/provider/kubernetes/application/application_test.go` for PVC stability during scaling:
  - Scale up from 1 to 3: verify no new PVCs created (same PVC count before and after)
  - Scale down from 3 to 1: verify no PVCs deleted
  - Scale up after scale-down (1 → 3 → 1 → 3): verify PVC count remains 1 throughout

**Checkpoint**: PVC count is stable across all scaling operations. Shared PVC model works correctly for Deployment/DaemonSet.

---

## Phase 14: Storage Stories — Test Runs and Integration

**Purpose**: Run test suites for all packages modified by storage stories. Cross-cutting validation.

- [ ] T086 Run `go test ./domain/application/state/...` to verify storage attachment cascade cleanup (T070-T071) and ephemeral no-op (T083)
- [ ] T087 Run `go test ./domain/application/service/...` to verify mock regeneration (T072) didn't break existing tests
- [ ] T088 Run `go test ./internal/provider/kubernetes/application/...` to verify PVC cleanup in Delete() (T075), ephemeral storage (T082), and PVC scaling stability (T085)
- [ ] T089 Run `go test ./internal/worker/caasapplicationprovisioner/...` to verify access mode warnings (T079)
- [ ] T090 Run `make go-build` to verify no compilation errors across all modified packages

**Checkpoint**: All storage adaptation tests pass. No regressions in existing MVP tests.

---

## Storage Stories — Dependencies & Execution Order

### Phase Dependencies

- **Phase 9 (US7)**: Depends on Phase 7 (T050-T054 complete). **GATE** for all storage stories.
- **Phase 10 (US8)**: Depends on Phase 9 (storage attachment cleanup must work)
- **Phase 11 (US9+10)**: Depends on Phase 9 (storage foundation in place). Independent of Phase 10.
- **Phase 12 (US11)**: Depends on Phase 9 (ClearCAASUnitCloudContainer must handle no-op). Independent of Phase 10, 11.
- **Phase 13 (US8 suppl.)**: Depends on Phase 10 (PVC lifecycle understood). Independent of Phase 11, 12.
- **Phase 14 (Test runs)**: Depends on all story phases (9-13) being complete.

### Dependency Graph

```
Phase 8 (Resilience Testing — MVP)
    └──> Phase 9 (US7: Storage attachment cleanup) ← P1 GATE
              ├──> Phase 10 (US8: PVC cleanup on removal) ← P1
              │         └──> Phase 13 (US8: PVC scaling stability) ← P2
              ├──> Phase 11 (US9+10: Access mode validation) ← P2
              └──> Phase 12 (US11: Ephemeral storage) ← P2
              Phases 10-13 all complete ──> Phase 14 (Test runs)
```

### Parallel Opportunities

**After Phase 9 completes** (3 independent tracks):
```
Phase 10 (T073-T075) ─┐
Phase 11 (T076-T077, T079) ┤── parallel (different files, different layers)
Phase 12 (T080-T083)  ─┘
```

**After Phase 10 completes**:
```
Phase 13 (T084-T085) ── can start immediately
```

### Implementation Strategy (Storage Stories)

1. **Phase 9 first** — small, focused change to `ClearCAASUnitCloudContainer`. This is the gate.
2. **Phases 10, 11, 12 in parallel** — different files, different layers. Can be done simultaneously.
3. **Phase 13 after Phase 10** — verification/testing of PVC scaling behavior.
4. **Phase 14 last** — run all test suites to confirm no regressions.

---

## Notes

- FR-009 (silently ignore deployment-type on IAAS) requires no new code — the existing constraint
  system already ignores unsupported constraints for IAAS models. Covered by T003 validation logic.
- FR-011 (default existing apps to StatefulSet on upgrade) requires no new code beyond T001 —
  the `DEFAULT 0` on the `deployment_type_id` column ensures existing rows get `stateful`.
- Edge case 4 (model-level deployment-type constraint) requires no new code — the existing
  constraint system already supports model-level defaults with per-application overrides.
- [P] tasks = different files, no dependencies on incomplete tasks
- [Story] label maps task to specific user story for traceability
- Story 1+2+4 are grouped because they are deeply intertwined (constraint → persistence → provisioning → backward compat)
- Story 3 and Story 5 are fully independent of each other and can be implemented in either order
- No new packages are created — all changes fit within existing directory structure
- ~~Provider layer (`internal/provider/kubernetes/application/`) requires NO changes — it already supports all 3 types~~ **CORRECTED**: Provider layer required `currentScale()` and `computeStatus()` implementations for Deployment/DaemonSet (T044, T045). The scaffolding existed but returned `NotSupported`.
- Edge case 2 (DaemonSet + non-shared storage access mode): Addressed by Phase 11 (US9+10,
  T076-T079). The worker emits a non-blocking warning when DaemonSet storage uses ReadWriteOnce
  access mode. The provider layer uses standalone PVCs (`handlePVCForStatelessResource`).
- ~~**K8s provider gaps (pre-existing, not blockers)**: `computeStatus()` only fully implemented for StatefulSet (returns `NotSupported` for Deployment/DaemonSet)~~ **CORRECTED**: `computeStatus()` and `currentScale()` are now implemented for all 3 types (T044, T045). `Exists()` still only checks the stored deployment type (no cross-type detection); no drift detection for manual kubectl edits. These remaining gaps can be addressed in a follow-up PR.
- **Storage stories (T070-T090)**: Added 2026-02-10 after storage codebase research session.
  Task IDs start at T070 to avoid collision with existing T050-T065 range.
- FR-016 access mode warning is non-blocking (per clarification Q2). The system warns but does not
  block deployment because the operator explicitly overrode the inference heuristic.
- SC-009 (zero PVCs after removal) depends on `Delete()` being called. The pre-existing
  `remove-application` race (spec.md Known Bugs) can prevent this. Race fix is out of scope.
- US8 scenario 4 (force-remove with `--force`): Satisfied by existing Juju labels on
  standalone PVCs (`app.kubernetes.io/managed-by=juju`, `app.kubernetes.io/name={appName}`).
  The `--force` path triggers the same `Delete()` flow with the same label-based cleanup.
  No additional task needed.
- The `FilesystemProvisioningInfo()` facade stub is out of scope (per clarification Q1).
  US9 scaling with storage depends on this facade being implemented separately.
- Ephemeral storage (US11) is largely a verification exercise — the existing `VolumeSourceForFilesystem()`
  code path already handles EmptyDir/tmpfs for all deployment types.
