# Bug Report: `remove-application` on K8s leaves orphaned resources due to removal service / worker race

> Ready to file at https://github.com/juju/juju/issues/new?template=bug-report.yml
>
> Fill in the GitHub form fields using the sections below.

---

## Description

`juju remove-application` on a Kubernetes model leaves orphaned K8s resources (Deployments, Services, Secrets, ServiceAccounts, Roles, RoleBindings, PVCs) in the model namespace. Juju reports the model as empty, but the resources remain on the cluster and are never cleaned up.

The root cause is a race condition between the **removal service** (`domain/removal/service/application.go`) and the **caasapplicationprovisioner worker** (`internal/worker/caasapplicationprovisioner/`). The removal service deletes the application's DB record before the worker can reach the `appDead()` code path that calls `app.Delete()` to clean up K8s resources.

This affects **all CAAS workload types** (StatefulSet, Deployment, DaemonSet). The only workaround is `juju destroy-model --force --destroy-storage`, which tears down the entire K8s namespace.

**Relationship to #21422**: Issue #21422 reported the same underlying race. It was closed via [PR #21441](https://github.com/juju/juju/pull/21441), but that fix only addressed the **unit sequence reset** symptom (units starting at ordinal 1 on redeploy). The K8s resource orphaning was not fixed. PR #21441 (`39bcd05b62b`) is in the current `main` branch and the bug reproduces on top of it.

## Juju version

4.0.2 (built from `main` branch as of 2026-02-10, includes PR #21441 fix)

## Cloud

Kubernetes (microk8s, but the race is provider-agnostic within CAAS)

## Expected behaviour

After `juju remove-application` completes and the model reports as empty, **all** K8s resources created for that application should be deleted from the model namespace. The namespace should contain only the model operator resources.

## Reproduce / Test

```bash
juju add-model test-model
juju deploy zinc-k8s
# Wait for "active" status (~90 seconds)
juju status -m test-model
# Confirm zinc-k8s is active

juju remove-application -m test-model zinc-k8s --no-prompt
# Wait for model to report empty (~30 seconds)
juju status -m test-model
# Shows: Model "test-model" is empty.

# Check K8s — these resources should not exist but they do:
microk8s kubectl -n test-model get deployments.apps,svc,sa,roles,rolebindings,secrets,pvc
```

**Expected**: Only `modeloperator` resources remain.

**Actual**: 14+ orphaned `zinc-k8s` resources remain:

| Orphaned Resource | Name |
|---|---|
| Deployment | `zinc-k8s` (0/0 replicas) |
| ReplicaSet | `zinc-k8s-*` |
| Service | `zinc-k8s` |
| Secrets | `zinc-k8s-application-config`, `zinc-k8s-zinc-secret` |
| ServiceAccounts | `zinc-k8s`, `unit-zinc-k8s-0` |
| Roles | `zinc-k8s`, `unit-zinc-k8s-0` |
| RoleBindings | `zinc-k8s`, `unit-zinc-k8s-0` |
| PVC | `zinc-k8s-data-*` |
| Endpoints | `zinc-k8s` |

## Notes & References

### Race condition analysis

Two independent actors race during application removal:

**Actor 1 — Removal Service** (`domain/removal/service/application.go:273-331`):
1. `RemoveApplication()` sets the app to `Dying` and schedules a background removal job
2. `processApplicationRemovalJob()` polls: is the app dying? are there 0 units?
3. At line 310: `DeleteApplication()` — **deletes the DB record** as soon as units reach 0

**Actor 2 — caasapplicationprovisioner worker** (`internal/worker/caasapplicationprovisioner/application.go:240-346`):
1. Detects `life.Dying` at line 268 → calls `AppDying()` → `ensureScale(0)` → scales K8s to 0 replicas, removes DB unit records
2. On next event, calls `handleChange()` → `GetApplicationLife()` → `ApplicationNotFound`
3. Line 242: treats "not found" as `life.Dead`
4. Calls `AppDying()` again at line 333 → `ensureScale()` → `updateProvisioningState()` → **"application not found" error → worker crashes**
5. Worker restarts, `GetApplicationName()` at line 131 returns `ApplicationNotFound` → **exits cleanly**
6. `appDead()` at line 338 is **never reached** → `app.Delete()` is **never called** → K8s resources orphaned

### Sequence diagram

```
Time  Removal Service                    Worker (appWorker)
─────┬─────────────────────────────────┬────────────────────────────────
  T1 │ Set app → Dying                │
  T2 │ Schedule removal job           │
  T3 │                                │ Detects Dying
  T4 │                                │ AppDying() → ensureScale(0)
  T5 │                                │ K8s: scale to 0 replicas
  T6 │                                │ DB: remove units → 0 remaining
  T7 │ Job sees 0 units               │
  T8 │ ★ DeleteApplication() ★        │
     │ (DB record GONE)               │
  T9 │                                │ handleChange() → GetApplicationLife()
     │                                │ → ApplicationNotFound → treat as Dead
 T10 │                                │ AppDying() → updateProvisioningState()
     │                                │ → "application not found" → CRASH
 T11 │                                │ Worker restarts → app gone → clean exit
     │                                │ ★ appDead() NEVER CALLED ★
     │                                │ ★ app.Delete() NEVER CALLED ★
     │                                │ ★ K8s resources ORPHANED ★
```

### The cleanup that never runs

`appDead()` in `ops.go:426-453` is the **only** code path that calls `app.Delete()`, which triggers the K8s provider cleanup at `internal/provider/kubernetes/application/application.go:985-1214`. That method deletes Deployments/StatefulSets/DaemonSets, Services, Secrets, ConfigMaps, Roles, RoleBindings, ClusterRoles, ClusterRoleBindings, ServiceAccounts, PVCs, CRDs, Ingresses, and webhook configs.

### Existing TODO in code

`internal/worker/caasapplicationprovisioner/ops.go:447-451`:
```go
// TODO(k8s): re-implement this to prevent a dead app from going away through
// creating a new domain concept that holds the application until this worker
// has destroyed all the k8s resources.
//
// Clear "has-resources" flag so state knows it can now remove the application.
```

### What PR #21441 fixed (and what it didn't)

PR #21441 (merged 2025-12-10, commit `70f1b147`) closed #21422 by adding `deleteSequencesForApplication()` in `domain/removal/state/model/application.go`. This reset the unit sequence counter so redeploying starts at ordinal 0.

It did **not** address:
- The removal service / worker race
- K8s resources being orphaned after removal
- The `appDead()` → `app.Delete()` path never being reached
- The TODO at `ops.go:447-451`

### Related issues

| Issue | Status | Relationship |
|---|---|---|
| #21422 | Closed | Same root cause. Only sequence-reset symptom was fixed. |
| #21605 | Open | Inverse symptom — app stuck in "dying" state (worker loops without progress) |
| #21257 | Open | Wedged K8s app cannot be removed |
| #21503 | Open | Consequence — stale PVCs break redeploy |
| #21722 | Open | Consequence — StatefulSet volume claim mismatch after force-remove + redeploy |
| #21177 | Closed | Same race class — caas firewaller crashes accessing deleted DB record |
| #21608 | Open | Related — K8s app with no units can't be removed |

### Suggested fix directions

**Option A: "has-resources" flag** (as suggested by the TODO at `ops.go:447`). Add a `has_cloud_resources` boolean to the `application` table. Removal service checks this flag before deleting the DB record. Worker clears it after `app.Delete()` completes.

**Option B: Make `appDying()` tolerant of "not found"**. When the worker enters the Dead case (including "not found"), skip `ensureScale()` if the app is gone from the DB and fall through to `appDead()` → `app.Delete()`. Minimal change but doesn't fix the fundamental lifecycle ordering.

**Option C: Hybrid — worker-confirmed deletion**. Removal service sets Dying → worker scales to 0 → worker calls `app.Delete()` → worker sets "cleanup-complete" → removal service deletes DB record. Correct lifecycle but requires coordination mechanism and timeout for unresponsive workers.
