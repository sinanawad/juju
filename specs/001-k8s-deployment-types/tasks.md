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

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Validation, cleanup, and verification across all stories.

### Test Runs

- [ ] T030 Run `go test ./core/constraints/...` to verify constraint parsing, validation, and serialization
- [ ] T031 Run `go test ./domain/application/...` to verify persistence, retrieval, scale validation, and immutability enforcement
- [ ] T032 Run `go test ./internal/worker/caasapplicationprovisioner/...` to verify deployment type determination and dynamic provisioning
- [ ] T033 Run `go test ./apiserver/facades/controller/caasapplicationprovisioner/...` to verify facade v2 and provisioning info population
- [ ] T034 Run `go test ./cmd/juju/status/...` to verify status display with Type column
- [ ] T035 Run quickstart.md validation: build with `make go-build` and verify no compilation errors

### New Tests Required (Constitution IV — Test Discipline)

- [ ] T050 [US1+2+4] Add unit tests for `RegisterCAASUnit` with `OrderedScale=false` in `domain/application/state/state_test.go` — verify Deployment/DaemonSet units can register when `ordinal < appScale.Scale` even when `Scaling=false`; verify rejection when `ordinal >= appScale.Scale`
- [ ] T051 [US1+2+4] Add unit tests for `GetCAASUnitNameByProviderID` in `domain/application/state/state_test.go` — verify lookup returns correct unit name for known provider ID; returns `(empty, false, nil)` for unknown provider ID
- [ ] T052 [US1+2+4] Add unit tests for `GetNextCAASUnitOrdinal` in `domain/application/state/state_test.go` — verify returns 0 for app with no units; returns `max+1` for app with existing units; handles gaps correctly
- [ ] T053 [US1+2+4] Add unit tests for `RegisterCAASUnit` service-layer branching in `domain/application/service/provider_test.go` — verify StatefulSet derives ordinal from pod name; Deployment looks up by provider ID then falls back to next ordinal

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
- **Polish (Phase 7)**: Depends on all story phases being complete

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
```

- **Story 3 and Story 5 are independent of each other** — they can run in parallel after Phase 3
- **Story 1+2+4 is the MVP** — once complete, the core feature works end-to-end
- **Phase 8 depends on Phase 7** — resilience testing validates pod recovery + all prior phases

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
- Edge case 2 (DaemonSet + non-shared storage access mode): Deferred — the provider layer
  already uses standalone PVCs for DaemonSets (`handlePVCForStatelessResource`), avoiding the
  identity-dependent VolumeClaimTemplate pattern. No additional validation needed for MVP.
- ~~**K8s provider gaps (pre-existing, not blockers)**: `computeStatus()` only fully implemented for StatefulSet (returns `NotSupported` for Deployment/DaemonSet)~~ **CORRECTED**: `computeStatus()` and `currentScale()` are now implemented for all 3 types (T044, T045). `Exists()` still only checks the stored deployment type (no cross-type detection); no drift detection for manual kubectl edits. These remaining gaps can be addressed in a follow-up PR.
