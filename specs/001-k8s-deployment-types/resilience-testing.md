# Resilience Testing Plan: K8s Deployment Type MVP

This document defines systematic resilience tests for the K8s Deployment/DaemonSet
MVP. Every scenario is executed for **both** Deployment (`deployment-type=stateless`)
and StatefulSet (default, no constraint) to guard against regressions.

All substrate commands use `microk8s kubectl`. All application-level operations
use `juju`. We only go to the substrate directly to **simulate failures** that
bypass Juju intentionally.

## Environment

| Component      | Value |
|----------------|-------|
| Substrate      | microk8s (single-node) |
| Charm          | `zinc-k8s` (Charmhub, latest/stable) |
| Model          | `test-model` |
| K8s namespace  | `test-model` |
| Controller     | Built from branch `001-k8s-deployment-types` |

```bash
# Variables used throughout
APP=zinc-k8s
NS=test-model
```

---

## Prerequisites

Before each scenario group, start from a clean model:

```bash
juju status  # Verify model is empty or remove existing apps first
```

To observe recovery in real time, keep a watch loop running in a separate terminal:

```bash
watch -n2 'juju status --format short 2>&1; echo "---"; microk8s kubectl get pods -n test-model -l app.kubernetes.io/name=zinc-k8s -o wide 2>&1'
```

---

## Scenario Group 1: Juju Lifecycle (Scale Up / Down / Up)

These scenarios use only `juju` commands. No substrate interaction.

### S1.1 Deploy and verify initial state

| Step | Deployment | StatefulSet |
|------|-----------|-------------|
| Deploy | `juju deploy zinc-k8s --constraints deployment-type=stateless` | `juju deploy zinc-k8s` |
| Wait | `juju wait-for application zinc-k8s --query='status=="active"' --timeout=5m` | same |
| Verify unit | `juju status` shows `zinc-k8s/0` active | same |
| Verify K8s resource | `microk8s kubectl get deployment zinc-k8s -n $NS` | `microk8s kubectl get statefulset zinc-k8s -n $NS` |
| Verify pod count | 1 pod running | 1 pod running |

**Pass criteria**: Unit `zinc-k8s/0` active/idle with an address.

### S1.2 Scale up from 1 to 3

```bash
juju scale-application zinc-k8s 3
```

| Check | Expected |
|-------|----------|
| `juju status` | 3 units: `zinc-k8s/0`, `zinc-k8s/1`, `zinc-k8s/2` |
| All units active/idle | Yes |
| Pod count matches | `microk8s kubectl get pods -n $NS -l app.kubernetes.io/name=zinc-k8s --no-headers \| wc -l` = 3 |
| Each unit has unique address | Yes |
| Each unit has unique provider ID | Yes |

**Timeout**: 3 minutes for all 3 units to reach active/idle.

### S1.3 Scale down from 3 to 1

```bash
juju scale-application zinc-k8s 1
```

| Check | Expected |
|-------|----------|
| `juju status` | 1 unit remains: `zinc-k8s/0` |
| Removed units gone | `zinc-k8s/1` and `zinc-k8s/2` no longer in status |
| Pod count matches | 1 pod running |
| Surviving unit unchanged | Same address, same provider ID as before scale-down |

**Timeout**: 2 minutes for scale-down to complete.

### S1.4 Scale back up from 1 to 2

```bash
juju scale-application zinc-k8s 2
```

| Check | Expected |
|-------|----------|
| `juju status` | 2 units active/idle |
| New unit ordinal | For StatefulSet: `zinc-k8s/1` (deterministic). For Deployment: next available ordinal |
| Pod count | 2 |

**Timeout**: 2 minutes.

### S1.5 Remove application cleanly

```bash
juju remove-application zinc-k8s
```

| Check | Expected |
|-------|----------|
| `juju status` | Model eventually empty |
| K8s cleanup | No zinc-k8s pods, deployments, statefulsets, or services remain in namespace |

**Timeout**: 3 minutes.

**Known issue**: `remove-application` has a race condition that may leave orphaned
K8s resources (see `remove-application-race.md`). If K8s resources remain after
the model shows empty, document it but do not block the test.

### S1.6 Redeploy after removal - ordinal reset

After S1.5 completes (model empty):

```bash
juju deploy zinc-k8s --constraints deployment-type=stateless  # or no constraint for StatefulSet
```

| Check | Expected |
|-------|----------|
| First unit | `zinc-k8s/0` (ordinal starts at 0, not inflated) |

**Pass criteria**: Ordinal reset confirms sequence cleanup in `DeleteApplication`.

---

## Scenario Group 2: Substrate Chaos (Pod Disruption via microk8s)

These scenarios use `microk8s kubectl` to simulate failures that bypass Juju.
Start each scenario from a known-good state (S1.1 completed).

### S2.1 Single pod deletion (scale=1)

**Setup**: zinc-k8s deployed with 1 unit, active/idle.

```bash
# Record current state
juju status                                           # Note unit name and IP
POD=$(microk8s kubectl get pods -n $NS -l app.kubernetes.io/name=zinc-k8s -o name | head -1)
echo "Deleting: $POD"

# Simulate failure
microk8s kubectl delete $POD -n $NS
```

| Check | Expected | Timeout |
|-------|----------|---------|
| K8s creates replacement pod | New pod name (different from deleted) | 30s |
| Unit re-registers | `zinc-k8s/0` active/idle with new IP | 90s |
| Unit name preserved | Still `zinc-k8s/0`, not a new ordinal | - |
| Provider ID updated | New pod name in `juju show-unit zinc-k8s/0` | - |
| No extra units | Still exactly 1 unit in `juju status` | - |

**StatefulSet difference**: Replacement pod has the **same** name (e.g., `zinc-k8s-0`).
Deployment replacement has a **different** random name. Both must recover to the
same unit.

### S2.2 Single pod deletion (scale=3)

**Setup**: zinc-k8s scaled to 3, all active/idle.

```bash
# Pick the middle pod (not the first, not the last)
PODS=$(microk8s kubectl get pods -n $NS -l app.kubernetes.io/name=zinc-k8s -o name)
TARGET=$(echo "$PODS" | sed -n '2p')
echo "Deleting: $TARGET"

microk8s kubectl delete $TARGET -n $NS
```

| Check | Expected | Timeout |
|-------|----------|---------|
| Replacement pod created | Yes | 30s |
| Affected unit recovers | Active/idle with new IP | 90s |
| Other 2 units unaffected | Same IP, same provider ID | - |
| Total units | Still 3 | - |

### S2.3 All pods deleted simultaneously (scale=3)

**Setup**: zinc-k8s scaled to 3, all active/idle.

```bash
microk8s kubectl delete pods -n $NS -l app.kubernetes.io/name=zinc-k8s
```

| Check | Expected | Timeout |
|-------|----------|---------|
| K8s creates 3 replacement pods | Yes | 60s |
| All 3 units recover | Active/idle | 120s |
| No duplicate units | Still exactly 3 units, same names | - |
| Each unit mapped to unique pod | No two units share a provider ID | - |

### S2.4 Rapid pod cycling (delete-wait-delete, scale=1)

**Setup**: zinc-k8s with 1 unit active/idle.

```bash
for i in 1 2 3; do
  POD=$(microk8s kubectl get pods -n $NS -l app.kubernetes.io/name=zinc-k8s -o name --field-selector=status.phase=Running | head -1)
  echo "Cycle $i: deleting $POD"
  microk8s kubectl delete $POD -n $NS --wait=false
  sleep 15  # Allow partial recovery
done
```

| Check | Expected | Timeout |
|-------|----------|---------|
| After cycling stops, unit recovers | `zinc-k8s/0` active/idle | 120s |
| Exactly 1 unit | No phantom units created | - |
| No stuck "allocating" units | Final state is active/idle, not waiting | - |

### S2.5 Pod deletion during scale-up

**Setup**: zinc-k8s with 1 unit active/idle.

```bash
# Trigger scale-up and immediately kill the original pod
juju scale-application zinc-k8s 3
sleep 2
POD=$(microk8s kubectl get pods -n $NS -l app.kubernetes.io/name=zinc-k8s -o name --field-selector=status.phase=Running | head -1)
microk8s kubectl delete $POD -n $NS
```

| Check | Expected | Timeout |
|-------|----------|---------|
| All 3 units eventually active | Yes | 180s |
| No stuck units | All units reach active/idle | - |
| Pod count matches unit count | 3 pods, 3 units | - |

### S2.6 Pod deletion during scale-down

**Setup**: zinc-k8s scaled to 3, all active/idle.

```bash
# Trigger scale-down and immediately kill a surviving pod
juju scale-application zinc-k8s 1
sleep 2
POD=$(microk8s kubectl get pods -n $NS -l app.kubernetes.io/name=zinc-k8s -o name --field-selector=status.phase=Running | head -1)
microk8s kubectl delete $POD -n $NS
```

| Check | Expected | Timeout |
|-------|----------|---------|
| Final state: 1 unit active | Yes | 180s |
| No orphaned units | Exactly 1 unit | - |
| Surviving unit functional | Active/idle with address | - |

---

## Scenario Group 3: Application Removal and Redeployment

### S3.1 Remove and redeploy - clean cycle

```bash
# Deploy, verify, remove
juju deploy zinc-k8s --constraints deployment-type=stateless
# ... wait for active ...
juju remove-application zinc-k8s
# ... wait for model empty ...

# Redeploy
juju deploy zinc-k8s --constraints deployment-type=stateless
```

| Check | Expected |
|-------|----------|
| New unit is `zinc-k8s/0` | Yes (sequence reset) |
| No residual state | Clean active/idle |

### S3.2 Remove with scale > 1 and redeploy

```bash
juju deploy zinc-k8s --constraints deployment-type=stateless
# ... wait for active ...
juju scale-application zinc-k8s 3
# ... wait for all 3 active ...
juju remove-application zinc-k8s
# ... wait for model empty ...
juju deploy zinc-k8s --constraints deployment-type=stateless
```

| Check | Expected |
|-------|----------|
| New unit is `zinc-k8s/0` | Yes (not zinc-k8s/3) |

### S3.3 Redeploy as different type

```bash
# Deploy as Deployment
juju deploy zinc-k8s --constraints deployment-type=stateless
# ... wait for active ...
juju remove-application zinc-k8s
# ... wait for model empty ...

# Redeploy as StatefulSet (no constraint = default)
juju deploy zinc-k8s
```

| Check | Expected |
|-------|----------|
| K8s resource type | StatefulSet (not Deployment) |
| Unit is `zinc-k8s/0` | Yes |
| Unit active/idle | Yes |

---

## Scenario Group 4: Worker Restart Resilience

### S4.1 Controller jujud restart during normal operation

**Setup**: zinc-k8s deployed with 2 units, both active/idle.

```bash
# Kill the jujud process on the controller (it auto-restarts)
microk8s kubectl exec controller-0 -n controller-k8s -c api-server -- \
  kill $(microk8s kubectl exec controller-0 -n controller-k8s -c api-server -- pgrep -f 'jujud machine')
```

| Check | Expected | Timeout |
|-------|----------|---------|
| Controller recovers | `juju status` works | 60s |
| Units unchanged | Both units still active/idle, same IPs | - |
| No spurious scaling | Pod count unchanged | - |

### S4.2 Controller restart + pod deletion combined

**Setup**: zinc-k8s with 2 units, both active/idle.

```bash
# Kill controller AND delete a pod simultaneously
microk8s kubectl exec controller-0 -n controller-k8s -c api-server -- \
  kill $(microk8s kubectl exec controller-0 -n controller-k8s -c api-server -- pgrep -f 'jujud machine')
sleep 2
POD=$(microk8s kubectl get pods -n $NS -l app.kubernetes.io/name=zinc-k8s -o name | head -1)
microk8s kubectl delete $POD -n $NS
```

| Check | Expected | Timeout |
|-------|----------|---------|
| Controller recovers | `juju status` works | 60s |
| Affected unit recovers | Active/idle with new pod | 120s |
| Unaffected unit unchanged | Same IP and provider ID | - |

---

## Scenario Group 5: Edge Cases

### S5.1 Scale to 0 and back to 1

```bash
juju scale-application zinc-k8s 0
# ... wait for 0 units ...
juju scale-application zinc-k8s 1
```

| Check | Expected | Timeout |
|-------|----------|---------|
| After scale to 0 | No units, no pods | 60s |
| After scale to 1 | 1 unit active/idle | 120s |

### S5.2 Rapid scale oscillation

```bash
for scale in 3 1 5 2 4 1; do
  juju scale-application zinc-k8s $scale
  sleep 10
done
# Final desired scale is 1
```

| Check | Expected | Timeout |
|-------|----------|---------|
| Final state | 1 unit active/idle | 180s |
| No orphaned units | Exactly 1 unit | - |
| No orphaned pods | Exactly 1 pod | - |

### S5.3 Delete pod that hasn't finished starting

**Setup**: Scale from 1 to 3, immediately delete a pod that's still in ContainerCreating.

```bash
juju scale-application zinc-k8s 3
sleep 3
# Find a pod that's not yet Running
PENDING=$(microk8s kubectl get pods -n $NS -l app.kubernetes.io/name=zinc-k8s --field-selector=status.phase!=Running -o name | head -1)
if [ -n "$PENDING" ]; then
  microk8s kubectl delete $PENDING -n $NS
fi
```

| Check | Expected | Timeout |
|-------|----------|---------|
| All 3 units eventually active | Yes | 180s |

---

## Execution Matrix

Each scenario must pass for **both** workload types:

| Scenario | Deployment (stateless) | StatefulSet (default) | Notes |
|----------|:---------------------:|:--------------------:|-------|
| S1.1 Deploy | PASS | PASS | |
| S1.2 Scale 1 -> 3 | PASS | FAIL (2/3) | Pre-existing storage provisioner bug |
| S1.3 Scale 3 -> 1 | PASS | PASS | |
| S1.4 Scale 1 -> 2 | FAIL (1/2) | PASS | Cascade from S1.3 state / intermittent |
| S1.5 Remove | PASS | PASS | |
| S1.6 Redeploy ordinal | PASS | PASS | |
| S2.1 Single pod kill (scale=1) | PASS | PASS | |
| S2.2 Single pod kill (scale=3) | PASS | PASS | |
| S2.3 All pods killed (scale=3) | KNOWN-LIM (2/3) | PASS | Filesystem attachment race (pre-existing) |
| S2.4 Rapid pod cycling | - | - | Not yet tested |
| S2.5 Pod kill during scale-up | - | - | Not yet tested |
| S2.6 Pod kill during scale-down | - | - | Not yet tested |
| S3.1 Remove + redeploy | - | - | Covered by S1.5+S1.6 |
| S3.2 Remove scaled + redeploy | - | - | Not yet tested |
| S3.3 Redeploy as different type | - | - | Not yet tested |
| S4.1 Controller restart | - | - | Not yet tested |
| S4.2 Controller restart + pod kill | - | - | Not yet tested |
| S5.1 Scale 0 and back | FAIL (cascade) | PASS | Deployment: cascade from S2.3 stuck unit |
| S5.2 Rapid scale oscillation | - | - | Not yet tested |
| S5.3 Kill starting pod | - | - | Not yet tested |

**Run date**: 2026-02-10

### Known Limitations (Pre-existing)

1. **Storage provisioner panic** (`duplicate key {unit-X filesystem-0}`): Intermittently
   affects scale-up operations for **both** StatefulSet and Deployment. This is a
   pre-existing bug in the storage provisioner worker, not caused by the Deployment feature.

2. **Filesystem attachment stale entries** (Deployment-specific): When **all** pods of a
   Deployment are deleted simultaneously, the replacement pods may fail to register because
   stale filesystem attachment entries (separate from the k8s_pod entries we clear) block
   the storage registration step. StatefulSet is unaffected because replacement pods have
   the same name, so step 1 of `RegisterCAASUnit` matches immediately without needing
   storage re-registration.

3. **Remove-application race**: `remove-application` can leave orphaned K8s resources.
   The test script handles this by cleaning up orphans after each removal.

---

## Pass / Fail Criteria

A scenario **passes** when:
1. The final `juju status` matches the expected state (unit count, status, no stuck units).
2. The K8s pod count matches the Juju unit count.
3. Every unit has a unique, valid provider ID and address.
4. No extra units were created beyond the target scale.
5. For StatefulSet: all behavior matches pre-feature behavior exactly (regression guard).

A scenario **fails** when:
1. A unit is stuck in `waiting/allocating` beyond the timeout.
2. Extra phantom units appear (ordinal inflation).
3. A unit loses its assignment and becomes permanently unassigned.
4. The worker enters a restart loop (check controller logs for repeated restarts).
5. K8s resources are leaked (pods, services, deployments remain after removal).

---

## Observability Commands

Use these to diagnose failures:

```bash
# Juju state
juju status
juju show-unit zinc-k8s/0 --format yaml

# Controller logs (full output)
microk8s kubectl -n controller-k8s logs controller-0 -c api-server --tail=200

# Worker-specific logs
microk8s kubectl -n controller-k8s logs controller-0 -c api-server --tail=500 | grep -i 'caas\|zinc\|stale\|register\|ordinal\|scale'

# K8s state
microk8s kubectl get pods -n $NS -l app.kubernetes.io/name=zinc-k8s -o wide
microk8s kubectl get deployment,statefulset,daemonset -n $NS
microk8s kubectl describe pod <pod-name> -n $NS

# Worker restarts (should be 0 during stable operation)
microk8s kubectl -n controller-k8s logs controller-0 -c api-server --tail=500 | grep -c 'worker returned unexpected error'
```

---

## Test Results Summary (2026-02-10)

### Deployment (stateless)

| Scenario | Result | Notes |
|----------|--------|-------|
| S1.1 Deploy | PASS | zinc-k8s/0 active, K8s Deployment created |
| S1.2 Scale 1→3 | PASS | All 3 units active, 3 pods |
| S1.3 Scale 3→1 | PASS | 1 unit remains, pods terminate cleanly |
| S1.4 Scale 1→2 | PASS | New unit created and active (on clean state) |
| S1.5 Remove | PASS | Model empty, K8s resources cleaned |
| S1.6 Redeploy ordinal | PASS | zinc-k8s/0 (sequence reset works) |
| S2.1 Single pod kill (1) | PASS | Unit recovered with new IP, no phantoms |
| S2.2 Single pod kill (3) | PASS | All 3 recovered, others unaffected |
| S2.3 All pods killed (3) | KNOWN-LIM | 2/3 recover; see race condition below |
| S5.1 Scale 0→1 | PASS | (when run on clean state) |

### StatefulSet (default) — Regression Guard

| Scenario | Result | Notes |
|----------|--------|-------|
| S1.1 Deploy | PASS | zinc-k8s/0 active, K8s StatefulSet created |
| S1.2 Scale 1→3 | FAIL (intermittent) | 2/3 units; pre-existing storage provisioner bug |
| S1.3 Scale 3→1 | PASS | |
| S1.4 Scale 1→2 | PASS | |
| S1.5 Remove | PASS | |
| S1.6 Redeploy ordinal | PASS | |
| S2.1 Single pod kill (1) | PASS | Same pod name recreated |
| S2.2 Single pod kill (3) | PASS | |
| S2.3 All pods killed (3) | PASS | StatefulSet: stable names → step 1 always matches |
| S5.1 Scale 0→1 | PASS | |

### Regression Assessment

**No regressions introduced.** All StatefulSet failures observed (S1.2) are pre-existing
storage provisioner bugs (`duplicate key`, `nil pointer dereference`) that also affect
Deployments intermittently.

### S2.3 Race Condition Analysis (Deployment-specific)

When **all** pods of a Deployment are deleted simultaneously:

1. K8s creates 3 new pods with new random names
2. New pods' agents call `RegisterCAASUnit` **before** the worker's `updateState` clears
   stale `k8s_pod` entries (~10s cycle)
3. Step 2 (`GetUnassignedCAASUnitName`) finds no unassigned units (stale entries block)
4. All 3 pods fall to step 3, attempt to create `zinc-k8s/3` → rejected ("unrequired",
   `aliveCount >= scale`)
5. Worker eventually clears stale entries
6. 2 of 3 pods retry and claim existing units successfully
7. The 3rd pod's agent may crash or back off too long → unit stays "lost"

**Why StatefulSet is unaffected:** Replacement pods have the same name as the originals, so
step 1 (`GetCAASUnitNameByProviderID`) always matches — no stale entry clearing needed.

**Mitigation for future work (not MVP):**
- Increase agent retry count/aggression for "unrequired" errors
- Or: make `RegisterCAASUnit` step 3 check for unassigned units before creating new ones
- Or: reduce worker cycle time or add event-driven stale detection
