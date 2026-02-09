# Review Brief: Kubernetes Deployment Type Support

**Branch**: `001-k8s-deployment-types` — ready for staff review
**Author**: Sinan + Claude Code | **Date**: 2026-02-08

---

## What This Feature Does

Today Juju hardcodes **StatefulSet** for every Kubernetes application.
This feature lets operators choose between **Deployment**, **StatefulSet**,
and **DaemonSet** via a new `deployment-type` constraint, with automatic
inference from charm metadata as the default.

```
juju deploy my-app                                    # inferred from charm
juju deploy my-app --constraints="deployment-type=daemon"   # explicit
```

**Inference rule**: charm declares storage -> StatefulSet; no storage -> Deployment.
DaemonSet is explicit-only. Existing apps default to StatefulSet (backward compat).

## What's Implemented (MVP, ~20 tasks across 30 files)

The full constraint-to-K8s-resource pipeline works end-to-end:

```
constraint parsing  ->  domain persistence  ->  API facade v2
    ->  worker provisioning  ->  K8s provider creates correct resource type
```

Specifically:
- `deployment-type` constraint with validation (`stateless|stateful|daemon`)
- Schema: `deployment_type` lookup table + FK on `application` (PATCH 0046)
- Domain service: `GetApplicationDeploymentType()` for retrieval
- Facade: CAASApplicationProvisioner bumped to v2, returns deployment type
- Worker: `DetermineDeploymentType()` replaces all hardcoded StatefulSet refs
- Immutability enforcement — cannot change type on a running app (FR-006)
- Storage mismatch warning — stateless + storage logs a warning (FR-012)
- Migration: re-inference from charm metadata on import (partial fix)
- YAML constraint validation bypass fixed

## What's NOT Implemented Yet

| What | Why it matters | Phase |
|------|---------------|-------|
| DaemonSet scale blocking | `juju scale-application` on a DaemonSet should return a clear error instead of failing at the provider | Phase 4 |
| `juju status` "Type" column | Operators can't see workload type in status output yet | Phase 5 |
| `description` library support | Explicit constraints (daemon, overrides) are **lost on migration** — the external serialization library doesn't carry `deployment-type` | Phase 6 (**release blocker**) |

## Key Design Decisions

1. **Constraint, not config** — Deployment type is a constraint (like `arch` or `mem`), not application config. This gives us model-level defaults, per-app overrides, and validation for free.

2. **Immutable after deploy** — Changing workload type requires destroying and recreating K8s resources. Rather than implement risky in-place migration, we reject changes and tell the operator to redeploy.

3. **No provider changes** — The K8s provider (`internal/provider/kubernetes/application/`) already supports all three types. This feature only wires the selection mechanism above it.

4. **Default 0 = stateful** — The schema uses `DEFAULT 0` (stateful) on the FK column, so existing rows automatically get StatefulSet on upgrade. Zero code needed for backward compat.

5. **Migration re-inference** — The `description/v11` library lacks `DeploymentType` support, so explicit constraints are silently dropped during export. We added re-inference from charm metadata on import as a pragmatic partial fix. This correctly handles the common case (inferred types) but loses `daemon` and explicit overrides. Full fix requires an upstream PR.

## Known Gaps (Pre-Existing, Not Introduced)

The K8s provider has gaps that affect all deployment types equally:
- `computeStatus()` only implemented for StatefulSet (Deployment/DaemonSet return NotSupported)
- No drift detection for manual `kubectl` edits
- `Exists()` only checks the stored type, won't detect a stray resource of a different type

These are documented but not in scope for this feature.

## Lifecycle Audit Results

Every charm/application lifecycle phase was verified:

| Phase | Verdict |
|-------|---------|
| Deploy | Correct — full constraint-to-K8s flow |
| Charm upgrade | Correct — `SetApplicationCharm` doesn't touch deployment type |
| Scale | Correct at provider level; domain guard pending (Phase 4) |
| Destroy | Correct — `Delete()` switches on type |
| Worker restart | Correct — re-reads type from DB |
| Config/Trust/Expose | Correct — don't re-provision |
| Migration | Partial — inferred types survive; explicit constraints need description lib |

## Artifacts to Review

Read in this order:

| # | Document | What to look for |
|---|----------|-----------------|
| 1 | **[spec.md](spec.md)** | Requirements (FR-001 through FR-013), user stories, edge cases, success criteria |
| 2 | **[plan.md](plan.md)** | Architecture, constitution check, delivery plan, pre-merge dependencies (release blocker), K8s provider gaps |
| 3 | **[tasks.md](tasks.md)** | Task breakdown with completion status, dependency graph, phase structure |
| 4 | **[data-model.md](data-model.md)** | Schema design, lookup table, FK relationship |
| 5 | **[contracts/](contracts/)** | API contract changes (provisioning-info, constraints, status) |
| 6 | **[manual-testing.md](manual-testing.md)** | How to verify the MVP on a real K8s cluster |
| 7 | **[quickstart.md](quickstart.md)** | Build, test, lint commands |

## Code Diff Overview

The diff touches ~30 files across 6 layers. No new packages.

**Core** (constraint definition):
`core/constraints/constraints.go`

**Domain** (business logic + persistence):
`domain/application/types.go`, `service/application.go`, `service/provider.go`,
`state/application.go`, `state/types.go`, `errors/errors.go`,
`service/migration.go`, `state/migration.go`
`domain/constraints/constraints.go`, `domain/schema/model.go`
`domain/schema/model/sql/0046-deployment-type.PATCH.sql` (new file)

**API server** (facade):
`apiserver/facades/controller/caasapplicationprovisioner/` (provisioner, register, service)

**API client**:
`api/controller/caasapplicationprovisioner/client.go`, `api/facadeversions.go`

**Workers**:
`internal/worker/caasapplicationprovisioner/` (application, ops)
`internal/worker/caasfirewaller/appfirewaller.go`

**Wire types**: `rpc/params/caas.go`

To see the full diff: `git diff main...001-k8s-deployment-types`

## One Question for Reviewer

The migration story has a hard dependency on the external `description/v11`
library. The re-inference workaround covers ~90% of real deployments, but
DaemonSet apps and explicit constraint overrides will lose their type on
migration until the upstream PR lands. Is this acceptable for merge to
a development branch, with the description PR as a release blocker?
