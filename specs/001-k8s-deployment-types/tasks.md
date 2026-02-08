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

- [ ] T001 Create DB PATCH file with `deployment_type` lookup table and `deployment_type_id` column on `application` table in `domain/schema/model/sql/NNNN-deployment-type.PATCH.sql`
- [ ] T002 Register new PATCH file in `domain/schema/model.go` via `modelPostPatchFilesByVersion`

**Checkpoint**: Schema changes ready — domain and upper layers can now reference deployment type.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core constraint field and domain type changes that ALL user stories depend on.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [ ] T003 [P] Add `DeploymentType *string` field to `Value` struct in `core/constraints/constraints.go` with parsing, validation (values: `stateless`, `stateful`, `daemon`), and serialization support
- [ ] T004 [P] Add `DeploymentType` field to `AddCAASApplicationArg` struct in `domain/application/types.go`
- [ ] T005 Map new `deployment-type` constraint in `domain/constraints/constraints.go`
- [ ] T006 Persist `deployment_type_id` during `CreateCAASApplication` in `domain/application/state/application.go` (read from `AddCAASApplicationArg`, map to lookup table ID, INSERT into `application` row)
- [ ] T007 Add `GetApplicationDeploymentType(ctx, appName) (string, error)` method to application service in `domain/application/service/application.go` (query `deployment_type` join from `application` table)

**Checkpoint**: Foundation ready — constraint parsing, persistence, and retrieval all functional. User story implementation can now begin.

---

## Phase 3: User Story 1+2+4 — Foundation + Constraint + Backward Compat (Priority: P1) 🎯 MVP

**Goal**: Charms deploy with the correct K8s workload type based on constraint or inference, while all existing charms continue as StatefulSet.

**Independent Test**: Deploy a charm with no storage → verify Deployment. Deploy with storage → verify StatefulSet. Deploy with explicit constraint → verify override works. Deploy existing charm → verify StatefulSet preserved.

**Validates**: FR-001, FR-002, FR-003, FR-004, FR-008, FR-009, FR-010, FR-011, FR-012 | SC-001, SC-003, SC-005, SC-007

### Wire Types & API Versioning

- [ ] T008 [P] [US1+2+4] Add `DeploymentType string` field to `CAASApplicationProvisioningInfo` struct in `rpc/params/caas.go` (or the file containing this struct) with JSON tag `"deployment-type,omitempty"`
- [ ] T009 [P] [US1+2+4] Bump CAASApplicationProvisioner facade version from v1 to v2 in `apiserver/facades/controller/caasapplicationprovisioner/register.go`
- [ ] T010 [P] [US1+2+4] Update client facade version for `CAASApplicationProvisioner` to 2 in `api/facadeversions.go`

### Facade: Populate Deployment Type in Provisioning Info

- [ ] T011 [US1+2+4] Populate `DeploymentType` field in provisioning info response by calling `GetApplicationDeploymentType()` in `apiserver/facades/controller/caasapplicationprovisioner/provisioner.go` (within the ProvisioningInfo assembly method)

### API Client: Parse New Field

- [ ] T012 [US1+2+4] Parse `DeploymentType` from provisioning info response in `api/controller/caasapplicationprovisioner/client.go`

### Worker: Dynamic Deployment Type Selection

- [ ] T013 [US1+2+4] Implement `DetermineDeploymentType(constraint *string, hasStorage bool) caas.DeploymentType` function in `internal/worker/caasapplicationprovisioner/ops.go` — logic: (1) if constraint set → map to caas type, (2) if charm has storage → `DeploymentStateful`, (3) else → `DeploymentStateless`
- [ ] T014 [US1+2+4] Replace hardcoded `caas.DeploymentStateful` with call to `DetermineDeploymentType()` using provisioning info in `internal/worker/caasapplicationprovisioner/application.go:149`
- [ ] T015 [US1+2+4] Replace hardcoded `caas.DeploymentStateful` with deployment type from state in `internal/worker/caasfirewaller/appfirewaller.go:81`

### Domain: Replace Hardcoded StatefulSet in Provider Service

- [ ] T016 [P] [US1+2+4] Replace hardcoded `caas.DeploymentStateful` at line 442 in `domain/application/service/provider.go` with deployment type read from application state
- [ ] T017 [P] [US1+2+4] Replace hardcoded `caas.DeploymentStateful` at line 538 in `domain/application/service/provider.go` with deployment type read from application state

### Warning for Storage Mismatch (FR-012)

- [ ] T018 [US1+2+4] Add warning log when `deployment-type=stateless` but charm declares persistent storage, emitted during deploy in the worker or domain service layer (exact location: near `DetermineDeploymentType` usage in `internal/worker/caasapplicationprovisioner/application.go`)

**Checkpoint**: At this point, deploying charms with any workload type works end-to-end. Existing charms continue as StatefulSet. Constraint validation, inference, persistence, and provisioning all functional.

---

## Phase 4: User Story 3 — DaemonSet Scale Blocking (Priority: P2)

**Goal**: Operators can deploy charms as DaemonSets with proper scale blocking. Manual scale operations are rejected with a clear error.

**Independent Test**: Deploy with `deployment-type=daemon` → verify DaemonSet created. Try `juju scale-application` → verify clear error. Add node to cluster → verify new pod appears.

**Validates**: FR-005, FR-006 | SC-002, SC-006

**Depends on**: Phase 3 (deployment type must be persistable and queryable)

### Implementation

- [ ] T019 [P] [US3] Add `DaemonSetScaleNotSupported` error constant to `domain/application/errors/errors.go` with message: "scaling is not supported for DaemonSet applications; scale is determined by the number of cluster nodes"
- [ ] T020 [US3] Add deployment type check in scale validation in `domain/application/service/application.go` — before processing scale change, query application deployment type; if `daemon`, return `DaemonSetScaleNotSupported` error

**Checkpoint**: DaemonSet applications correctly reject manual scaling. Combined with Phase 3, the full DaemonSet workflow (deploy + scale blocking) is testable.

---

## Phase 5: User Story 5 — Status Visibility (Priority: P2)

**Goal**: Operators can see the workload type for each K8s application in `juju status` output — both in the summary table and per-application detail.

**Independent Test**: Deploy applications with different workload types → run `juju status` → verify "Type" column shows correct values for CAAS model. Verify IAAS model has no Type column.

**Validates**: FR-007 | SC-004

**Depends on**: Phase 3 (deployment type must be persisted and queryable)

### Domain & Wire Types

- [ ] T021 [P] [US5] Add `DeploymentType *string` field to `Application` struct in `domain/status/service/types.go`
- [ ] T022 [P] [US5] Add `DeploymentType string` field to `ApplicationStatus` struct in `rpc/params/status.go` with JSON tag `"deployment-type,omitempty"`
- [ ] T023 [P] [US5] Add `DeploymentType string` field to `applicationStatus` struct in `cmd/juju/status/formatted.go`

### Status Assembly (API Server)

- [ ] T024 [US5] Populate `DeploymentType` in CAAS application status assembly in `apiserver/facades/client/client/status.go` (within the CAAS-specific block around line 940-956) by reading from the domain status Application struct

### Status Display (CLI)

- [ ] T025 [US5] Map `DeploymentType` from `params.ApplicationStatus` to `applicationStatus` in `cmd/juju/status/formatter.go`
- [ ] T026 [US5] Add "Type" column header to CAAS application table headers at `cmd/juju/status/output_tabular.go:142` (after "Exposed", before "Message")
- [ ] T027 [US5] Print deployment type value (display names: "Deployment", "StatefulSet", "DaemonSet") in the CAAS application table rendering loop in `cmd/juju/status/output_tabular.go`

**Checkpoint**: `juju status` shows workload type for all CAAS applications. IAAS models unaffected.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Validation, cleanup, and verification across all stories.

- [ ] T028 Run `go test ./core/constraints/...` to verify constraint parsing, validation, and serialization
- [ ] T029 Run `go test ./domain/application/...` to verify persistence, retrieval, and scale validation
- [ ] T030 Run `go test ./internal/worker/caasapplicationprovisioner/...` to verify deployment type determination and dynamic provisioning
- [ ] T031 Run `go test ./apiserver/facades/controller/caasapplicationprovisioner/...` to verify facade v2 and provisioning info population
- [ ] T032 Run `go test ./cmd/juju/status/...` to verify status display with Type column
- [ ] T033 Run quickstart.md validation: build with `make go-build` and verify no compilation errors

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 (schema must exist before persistence code)
- **Story 1+2+4 (Phase 3)**: Depends on Phase 2 (constraint + domain types must exist)
- **Story 3 (Phase 4)**: Depends on Phase 3 (deployment type must be persistable/queryable)
- **Story 5 (Phase 5)**: Depends on Phase 3 (deployment type must be persisted)
- **Polish (Phase 6)**: Depends on all story phases being complete

### User Story Dependencies

```
Phase 1 (Setup)
    └──> Phase 2 (Foundational)
              └──> Phase 3 (US1+2+4: Core feature) 🎯 MVP
                        ├──> Phase 4 (US3: DaemonSet scale blocking)
                        └──> Phase 5 (US5: Status visibility)
                                    └──> Phase 6 (Polish)
```

- **Story 3 and Story 5 are independent of each other** — they can run in parallel after Phase 3
- **Story 1+2+4 is the MVP** — once complete, the core feature works end-to-end

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
Phase 4 (US3: T019, T020) ─┐
Phase 5 (US5: T021-T027)  ─┘── parallel (different layers, no dependencies)
```

---

## Implementation Strategy

### MVP First (Story 1+2+4 Only)

1. Complete Phase 1: Setup (schema)
2. Complete Phase 2: Foundational (constraints, types, persistence)
3. Complete Phase 3: Story 1+2+4 (wire types, facade, worker, provider)
4. **STOP and VALIDATE**: Deploy charms with different types, verify inference + constraint override + backward compat
5. Run `go test` for core, domain, worker, and facade packages

### Incremental Delivery

1. Phases 1+2+3 → Core feature works → Test independently → **MVP**
2. Add Phase 4 (Story 3) → DaemonSet scale blocking → Test independently
3. Add Phase 5 (Story 5) → Status visibility → Test independently
4. Phase 6 → Full validation pass
5. Each story adds value without breaking previous stories

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks
- [Story] label maps task to specific user story for traceability
- Story 1+2+4 are grouped because they are deeply intertwined (constraint → persistence → provisioning → backward compat)
- Story 3 and Story 5 are fully independent of each other and can be implemented in either order
- No new packages are created — all changes fit within existing directory structure
- Provider layer (`internal/provider/kubernetes/application/`) requires NO changes — it already supports all 3 types
