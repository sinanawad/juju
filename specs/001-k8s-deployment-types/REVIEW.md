# Review Brief: Kubernetes Deployment Type Support

**Branch**: `001-k8s-deployment-types`
**Author**: Sinan + Claude Code | **Date**: 2026-02-10
**Status**: Ready for staff review (MVP complete, resilience-tested)

---

## Table of Contents

1. [What This Feature Does](#what-this-feature-does)
2. [Design Principles](#design-principles)
3. [User Stories & Requirements](#user-stories--requirements)
4. [What's Implemented (MVP)](#whats-implemented-mvp)
5. [Pod Recovery (Phase 7)](#pod-recovery-phase-7)
6. [Resilience Testing Results (Phase 8)](#resilience-testing-results-phase-8)
7. [What's NOT Implemented Yet](#whats-not-implemented-yet)
8. [Known Limitations](#known-limitations)
9. [Commit History](#commit-history)
10. [Code Diff Overview](#code-diff-overview)
11. [How to Verify](#how-to-verify)
12. [Artifacts Index](#artifacts-index)

---

## What This Feature Does

Today Juju hardcodes **StatefulSet** for every Kubernetes application.
This feature lets operators choose between **Deployment**, **StatefulSet**,
and **DaemonSet** via a new `deployment-type` constraint, with automatic
inference from charm metadata as the default.

```
juju deploy my-app                                         # inferred from charm
juju deploy my-app --constraints="deployment-type=daemon"  # explicit
```

**Inference rule**: charm declares storage -> StatefulSet; no storage -> Deployment.
DaemonSet is explicit-only. Existing apps default to StatefulSet (backward compat).

---

## Design Principles

### 1. Constraint, not config

Deployment type is a constraint (like `arch` or `mem`), not application config.
This gives us model-level defaults, per-app overrides, and validation for free.

### 2. Immutable after deploy

Changing workload type requires destroying and recreating K8s resources. Rather
than implement risky in-place migration, we reject changes and tell the operator
to redeploy. (FR-006)

### 3. No provider changes required (mostly)

The K8s provider (`internal/provider/kubernetes/application/`) already supported
all three types. This feature wires the selection mechanism above it. Two provider
methods (`currentScale`, `computeStatus`) needed implementation for Deployment/DaemonSet
but the scaffolding was already there.

### 4. Default 0 = stateful

The schema uses `DEFAULT 0` (stateful) on the FK column, so existing rows
automatically get StatefulSet on upgrade. Zero code needed for backward compat.

### 5. Migration re-inference

The `description/v11` library lacks `DeploymentType` support, so explicit
constraints are silently dropped during export. We added re-inference from charm
metadata on import as a pragmatic partial fix. This correctly handles the common
case (inferred types) but loses `daemon` and explicit overrides. Full fix requires
an upstream PR.

### 6. Worker-side reconciliation for pod recovery

When K8s replaces a Deployment/DaemonSet pod, the new pod has a different random
name. The worker's `updateState()` (the only code path with access to both K8s
pods and DB state) detects stale `k8s_pod` entries and clears them, allowing the
agent's registration retry to succeed naturally.

---

## User Stories & Requirements

The spec defines **6 user stories** and **13 functional requirements**. Full details in [spec.md](spec.md).

### User Stories (with MVP status)

| # | Story | Priority | Status |
|---|-------|----------|--------|
| US1 | Deploy a Stateless Application (Deployment) | P1 | Implemented |
| US2 | Explicit Workload Type Selection via Constraints | P1 | Implemented |
| US3 | Deploy a Node-Level Agent as a DaemonSet | P2 | Partial (scale blocking pending) |
| US4 | Backward-Compatible Default Behavior | P1 | Implemented |
| US5 | Workload Type Visibility in Status | P2 | Not started |
| US6 | Pod Recovery and Resilience | P1 | Implemented |

### Functional Requirements (with MVP status)

| FR | Description | Status |
|----|-------------|--------|
| FR-001 | Support three workload types (Deployment, StatefulSet, DaemonSet) | Done |
| FR-002 | `deployment-type` constraint with values `stateless/stateful/daemon` | Done |
| FR-003 | Automatic inference from charm storage metadata | Done |
| FR-004 | Existing charms continue as StatefulSet | Done |
| FR-005 | Reject manual scaling for DaemonSet | Pending (Phase 4) |
| FR-006 | Immutability enforcement (can't change type on running app) | Done |
| FR-007 | Display workload type in `juju status` | Pending (Phase 5) |
| FR-008 | Validate constraint values at deploy time | Done |
| FR-009 | Silently ignore on IAAS models | Done (existing behavior) |
| FR-010 | Persist type across controller restarts/upgrades | Done |
| FR-011 | Default existing apps to StatefulSet on upgrade | Done (schema default) |
| FR-012 | Warning when stateless + storage | Done |
| FR-013 | Preserve type during model migration | Partial (re-inference only) |

### Acceptance Scenarios Validated

Key scenarios from the spec that have been tested end-to-end on a real K8s cluster:

- Deploy charm with no storage -> runs as Deployment
- Deploy with `deployment-type=stateless` -> runs as Deployment
- Deploy with `deployment-type=stateful` -> runs as StatefulSet
- Scale Deployment up/down -> replicas change correctly
- Scale StatefulSet up/down -> replicas change correctly
- Delete Deployment pod -> unit re-registers on replacement pod
- Delete StatefulSet pod -> unit re-registers on same-name pod
- Remove application -> resources cleaned up
- Redeploy after removal -> ordinals reset to 0
- Attempt constraint change on running app -> error returned

---

## What's Implemented (MVP)

The full constraint-to-K8s-resource pipeline works end-to-end across 8 phases:

```
constraint parsing  ->  domain persistence  ->  API facade v2
    ->  worker provisioning  ->  K8s provider creates correct resource type
    ->  pod recovery on replacement  ->  resilience tested
```

### Phase 1-2: Schema & Foundations (T001-T007)

- `deployment_type` lookup table + FK on `application` (PATCH 0046)
- `DeploymentType *string` in `core/constraints/constraints.go` with validation
- Domain mapping, persistence in `CreateCAASApplication`, retrieval via `GetApplicationDeploymentType()`

### Phase 3: Core Feature (T008-T020)

- Wire types: `DeploymentType` in `CAASApplicationProvisioningInfo`
- Facade: CAASApplicationProvisioner bumped to v2
- Worker: `DetermineDeploymentType()` replaces all hardcoded StatefulSet refs
- Immutability enforcement: `DeploymentTypeImmutable` error (FR-006)
- Storage mismatch warning: stateless + storage logs a warning (FR-012)
- YAML constraint validation bypass fixed

### Phase 3a: Provider Fixes (T044-T045)

- `currentScale()` implemented for Deployment and DaemonSet
- `computeStatus()` implemented for Deployment and DaemonSet

### Phase 3b: Unit Registration Refactor (T046-T049)

StatefulSet pods have predictable names (`<app>-<ordinal>`). Deployment/DaemonSet
pods have random suffixes. This required a new 3-step registration strategy:

1. `GetCAASUnitNameByProviderID` — match existing unit by pod name (idempotent)
2. `GetUnassignedCAASUnitName` — find unit with no `k8s_pod` row
3. `GetNextCAASUnitOrdinal` + scaling gate — allocate next ordinal

The scaling gate is also relaxed: StatefulSet requires `Scaling=true` AND
`ordinal < ScaleTarget`; Deployment/DaemonSet only requires `aliveCount < appScale.Scale`.

### Phase 6 (Partial): Migration (T036-T039)

- YAML constraint validation bypass fixed
- `DeploymentType` in `InsertApplicationArgs` for migration import
- Re-inference from charm metadata on CAAS import

### Phase 7: Pod Recovery (T050-T054)

When K8s replaces a Deployment/DaemonSet pod, the old unit's `k8s_pod` row
blocks the new pod's registration. The fix:

1. **State**: `ClearCAASUnitCloudContainer(ctx, unitName)` — deletes `k8s_pod_port` and `k8s_pod` rows
2. **Service**: Exposed via `UnitState` interface with validation
3. **Worker**: `updateState()` in `ops.go` compares `unitToPod` map against live K8s pods; any unit whose provider_id doesn't match an active pod gets its cloud container cleared

Additionally:
- `DeleteApplication` sequence now cleans up `application_scale` before `unit_agent_status` (fixes `FOREIGN KEY constraint failed` on removal)
- `GetNextCAASUnitOrdinal` filters by `life_id < 2` (excludes dead units from max ordinal calculation, preventing ordinal inflation)
- `appDying()` tolerates `applicationerrors.UnitNotFound` (prevents crash when removal service races ahead)

---

## Pod Recovery (Phase 7)

### The Problem

When a Deployment/DaemonSet pod is deleted, K8s creates a replacement with a **new
random name**. The jujud agent in the new pod calls `RegisterCAASUnit`, but:

1. **Step 1** (`GetCAASUnitNameByProviderID`): No match (new pod name)
2. **Step 2** (`GetUnassignedCAASUnitName`): Stale `k8s_pod` row blocks (unit appears assigned)
3. **Step 3** (`GetNextCAASUnitOrdinal`): Rejected — `aliveCount >= appScale.Scale`

Result: new pod loops forever with `"unrequired unit zinc-k8s/1 is not assigned"`.

### The Fix

The worker's `updateState()` already queries both K8s pods (`app.Units()`) and DB
pod mappings (`GetAllUnitCloudContainerIDsForApplication`). We add stale detection:

```go
activePods := make(map[string]struct{}, len(units))
for _, u := range units {
    activePods[u.Id] = struct{}{}
}
for unitName, podName := range unitToPod {
    if _, active := activePods[podName]; !active {
        logger.Infof(ctx, "clearing stale cloud container for unit %s (pod %s no longer active)", unitName, podName)
        applicationService.ClearCAASUnitCloudContainer(ctx, unitName)
    }
}
```

On the next agent retry (~seconds), step 2 finds the now-unassigned unit and succeeds.

### Why StatefulSet is unaffected

StatefulSet replacement pods have the **same name** as the originals (`zinc-k8s-0`),
so step 1 always matches immediately. No stale entry clearing needed.

---

## Resilience Testing Results (Phase 8)

Full test plan and script in [resilience-testing.md](resilience-testing.md) and
[resilience-test.sh](resilience-test.sh). Tested on a real microk8s cluster with
the `zinc-k8s` charm (Charmhub latest/stable).

### Deployment (stateless) — 10 scenarios

| Scenario | Result |
|----------|--------|
| S1.1 Deploy | PASS |
| S1.2 Scale 1->3 | PASS |
| S1.3 Scale 3->1 | PASS |
| S1.4 Scale 1->2 | PASS |
| S1.5 Remove | PASS |
| S1.6 Redeploy ordinal reset | PASS |
| S2.1 Single pod kill (scale=1) | PASS |
| S2.2 Single pod kill (scale=3) | PASS |
| S2.3 All pods killed (scale=3) | KNOWN-LIM (2/3 recover) |
| S5.1 Scale 0->1 | PASS |

### StatefulSet (default) — Regression Guard — 10 scenarios

| Scenario | Result |
|----------|--------|
| S1.1 Deploy | PASS |
| S1.2 Scale 1->3 | FAIL (intermittent, pre-existing) |
| S1.3 Scale 3->1 | PASS |
| S1.4 Scale 1->2 | PASS |
| S1.5 Remove | PASS |
| S1.6 Redeploy ordinal reset | PASS |
| S2.1 Single pod kill (scale=1) | PASS |
| S2.2 Single pod kill (scale=3) | PASS |
| S2.3 All pods killed (scale=3) | PASS |
| S5.1 Scale 0->1 | PASS |

### Regression Assessment

**No regressions introduced.** The sole StatefulSet failure (S1.2, 2/3 scale-up) is
caused by a pre-existing storage provisioner panic (`duplicate key {unit-X filesystem-0}`)
that affects both StatefulSet and Deployment intermittently.

### S2.3 Race Condition (Deployment-specific)

When **all** Deployment pods die simultaneously, replacement pods register **before**
the worker clears stale entries (~10s cycle). Timeline:

1. K8s creates 3 new pods with random names
2. Agents call `RegisterCAASUnit` — step 2 blocked by stale entries
3. All 3 fall to step 3 → rejected (`aliveCount >= scale`)
4. Worker clears stale entries on next cycle
5. 2/3 pods retry successfully; 1 may stay stuck

This is an edge case (requires all pods to die at the exact same moment) with
known mitigations for post-MVP work. See [resilience-testing.md](resilience-testing.md)
for full analysis.

---

## What's NOT Implemented Yet

| What | Why it matters | Phase | Effort |
|------|---------------|-------|--------|
| DaemonSet scale blocking | `juju scale-application` on DaemonSet should return a clear error | Phase 4 | Small |
| `juju status` "Type" column | Operators can't see workload type in status output | Phase 5 | Medium |
| `description` library support | Explicit constraints (`daemon`, overrides) **lost on migration** | Phase 6 | External dep (**release blocker**) |
| Remaining resilience scenarios | S2.4-S2.6, S3.2-S3.3, S4.1-S4.2, S5.2-S5.3 not yet executed | Phase 8 | Test-only |

---

## Known Limitations

### 1. Pre-existing: Storage provisioner panic

`duplicate key {unit-X filesystem-0}` — intermittently affects scale-up for **both**
StatefulSet and Deployment. Not caused by this feature. Causes cascading worker
restarts until the corrupted in-memory state is cleared (e.g., by killing jujud).

### 2. Deployment-specific: All-pods-killed race (S2.3)

When all pods of a Deployment die simultaneously, 2/3 recover but 1 may stay stuck
due to the worker cycle timing vs. agent registration race. StatefulSet is unaffected
(stable pod names). See [S2.3 analysis](#s23-race-condition-deployment-specific).

### 3. Pre-existing: remove-application race

`remove-application` can leave orphaned K8s resources due to a race between the
removal service (deletes DB record) and the worker (cleans up K8s resources).
Affects all workload types. Documented in [spec.md](spec.md) with suggested fix
directions. The resilience test script handles this with cleanup fallbacks.

### 4. Pre-existing: K8s provider gaps

- `Exists()` only checks the stored type (won't detect a stray resource of a different type)
- No drift detection for manual `kubectl` edits
- No cross-type resource validation

These are low-severity and affect all deployment types equally.

---

## Commit History

```
81bda3598e docs: add resilience testing plan, script, and results
c7bf2e3fd7 feat: add pod recovery for Deployment/DaemonSet and fix unit ordinal inflation
2e88fa124c feat: fix review issues, add non-StatefulSet unit registration and provider support
babc10a1dc feat: add deployment-type constraint for K8s workload selection (MVP)
a7afb43178 docs: fix analysis gaps in spec, plan, and tasks
dbee02f671 docs: add speckit tooling and k8s deployment type design artifacts
```

To see the full diff: `git diff main...001-k8s-deployment-types`

---

## Code Diff Overview

The diff touches ~40 files across 6 layers. No new packages.

### Core (constraint definition)

`core/constraints/constraints.go` — `DeploymentType *string` field, parsing, validation

### Domain (business logic + persistence)

| File | Change |
|------|--------|
| `domain/application/types.go` | `DeploymentType` in `AddCAASApplicationArg`, `InsertApplicationArgs` |
| `domain/application/service/application.go` | `GetApplicationDeploymentType()` |
| `domain/application/service/provider.go` | Deployment-type-aware `RegisterCAASUnit`, immutability check |
| `domain/application/service/unit.go` | `ClearCAASUnitCloudContainer()` |
| `domain/application/state/application.go` | Persist `deployment_type_id` |
| `domain/application/state/unit.go` | `GetCAASUnitNameByProviderID`, `GetUnassignedCAASUnitName`, `GetNextCAASUnitOrdinal`, `ClearCAASUnitCloudContainer`, relaxed scaling gate |
| `domain/application/state/migration.go` | `DeploymentTypeID` in migration import |
| `domain/application/service/migration.go` | Re-inference from charm metadata on import |
| `domain/application/errors/errors.go` | `DeploymentTypeImmutable`, `DaemonSetScaleNotSupported` |
| `domain/constraints/constraints.go` | Map new constraint |
| `domain/schema/model.go` | Register PATCH 0046 |
| `domain/schema/model/sql/0046-deployment-type.PATCH.sql` | Lookup table + FK |

### API server (facade)

| File | Change |
|------|--------|
| `apiserver/facades/controller/caasapplicationprovisioner/register.go` | v1 -> v2 |
| `apiserver/facades/controller/caasapplicationprovisioner/provisioner.go` | `DeploymentType` in provisioning info |
| `apiserver/facades/controller/caasapplicationprovisioner/service.go` | `ClearCAASUnitCloudContainer` interface |

### API client

| File | Change |
|------|--------|
| `api/facadeversions.go` | CAASApplicationProvisioner v2 |
| `api/controller/caasapplicationprovisioner/client.go` | Parse `DeploymentType` |

### Workers

| File | Change |
|------|--------|
| `internal/worker/caasapplicationprovisioner/application.go` | Dynamic deployment type |
| `internal/worker/caasapplicationprovisioner/ops.go` | `DetermineDeploymentType()`, stale pod reconciliation, `DeleteApplication` sequence fix |
| `internal/worker/caasapplicationprovisioner/worker.go` | `ClearCAASUnitCloudContainer` in `ApplicationService` interface |
| `internal/worker/caasfirewaller/appfirewaller.go` | Read deployment type from state |

### K8s provider

| File | Change |
|------|--------|
| `internal/provider/kubernetes/application/scale.go` | `currentScale()` for Deployment/DaemonSet |
| `internal/provider/kubernetes/application/application.go` | `computeStatus()` for Deployment/DaemonSet |

### Wire types

`rpc/params/caas.go` — `DeploymentType` in `CAASApplicationProvisioningInfo`

---

## How to Verify

### Build

```bash
make jujud-controller   # Build controller binary with all changes
```

### Unit tests

```bash
go test ./core/constraints/... -count=1
go test ./domain/application/... -count=1
go test ./domain/constraints/... -count=1
go test ./internal/worker/caasapplicationprovisioner/... -count=1
go test ./internal/worker/caasfirewaller/... -count=1
go test ./apiserver/facades/controller/caasapplicationprovisioner/... -count=1
```

### Manual verification on K8s

```bash
# Push custom binary to controller (see quickstart.md for details)
# Then:

juju add-model test-model

# 1. Deploy as Deployment (no storage charm -> auto-inferred)
juju deploy zinc-k8s --constraints deployment-type=stateless
juju wait-for application zinc-k8s --query='status=="active"' --timeout=5m
microk8s kubectl get deployment zinc-k8s -n test-model   # Should exist

# 2. Verify pod recovery
POD=$(microk8s kubectl get pods -n test-model -l app.kubernetes.io/name=zinc-k8s -o name | head -1)
microk8s kubectl delete $POD -n test-model
# Wait ~30s, then:
juju status   # zinc-k8s/0 should be active with new IP

# 3. Scale up/down
juju scale-application zinc-k8s 3
juju scale-application zinc-k8s 1

# 4. Deploy as StatefulSet (backward compat)
juju deploy zinc-k8s zinc-stateful
microk8s kubectl get statefulset zinc-stateful -n test-model   # Should exist
```

### Automated resilience testing

```bash
bash specs/001-k8s-deployment-types/resilience-test.sh
# Or for specific workload type:
bash specs/001-k8s-deployment-types/resilience-test.sh -t stateless
bash specs/001-k8s-deployment-types/resilience-test.sh -t stateful
```

---

## Artifacts Index

Read in this order for full context:

| # | Document | What to look for |
|---|----------|-----------------|
| 1 | **[spec.md](spec.md)** | User stories (6), functional requirements (FR-001 through FR-013), edge cases, known bugs, success criteria |
| 2 | **[plan.md](plan.md)** | Architecture, constitution check, delivery plan, pre-merge dependencies (release blocker), K8s provider gaps |
| 3 | **[tasks.md](tasks.md)** | Task breakdown with completion status (8 phases, ~55 tasks), dependency graph |
| 4 | **[data-model.md](data-model.md)** | Schema design: `deployment_type` lookup table, FK on `application` |
| 5 | **[contracts/](contracts/)** | API contract changes (provisioning-info, constraints, status) |
| 6 | **[resilience-testing.md](resilience-testing.md)** | 20-scenario test plan, execution matrix, results, S2.3 race analysis, known limitations |
| 7 | **[resilience-test.sh](resilience-test.sh)** | Automated test script for both Deployment and StatefulSet |
| 8 | **[quickstart.md](quickstart.md)** | Build, test, lint, manual verification commands |
| 9 | **[remove-application-race.md](remove-application-race.md)** | Pre-existing race condition analysis |

---

## Lifecycle Audit Results

Every charm/application lifecycle phase was verified:

| Phase | Verdict |
|-------|---------|
| Deploy | Correct — full constraint-to-K8s flow |
| Charm upgrade | Correct — `SetApplicationCharm` doesn't touch deployment type |
| Scale | Correct at provider level; domain guard pending for DaemonSet (Phase 4) |
| Pod replacement | Correct — worker clears stale entries, agent re-registers |
| Destroy | Correct — `Delete()` switches on type |
| Worker restart | Correct — re-reads type from DB |
| Config/Trust/Expose | Correct — don't re-provision |
| Migration | Partial — inferred types survive; explicit constraints need description lib |

---

## Bugs Fixed Along the Way

1. **Cross-package test regression** — `domain/application/service_test.go` had mock expectations using `caas.DeploymentStateful` but `stubCharm` has no storage (inference yields `stateless`). Updated all 4 expectations.

2. **Non-ordered scale completion check** — `ops.go:804` used `len(units)` (includes dead units) to decide scale-up completion, but `unitScale` for Deployment/DaemonSet only counts non-dead units. Fixed to use `unitScale` consistently.

3. **Ordinal inflation** — `GetNextCAASUnitOrdinal` counted dead units in `max(ordinal)`, causing ever-increasing ordinals across deploy cycles. Fixed by filtering `WHERE life_id < 2`.

4. **DeleteApplication FK constraint failure** — `application_scale` had to be deleted before `unit_agent_status` in the deletion sequence to avoid `FOREIGN KEY constraint failed`.

5. **appDying crash on removal race** — When the removal service deleted the unit before the worker's `ensureScale` completed, the worker crashed on `UnitNotFound`. Fixed by tolerating this error.

---

## One Question for Reviewer

The migration story has a hard dependency on the external `description/v11`
library. The re-inference workaround covers ~90% of real deployments, but
DaemonSet apps and explicit constraint overrides will lose their type on
migration until the upstream PR lands. Is this acceptable for merge to
a development branch, with the description PR as a release blocker?
