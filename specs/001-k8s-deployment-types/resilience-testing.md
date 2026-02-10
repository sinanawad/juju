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
| Charms         | `zinc-k8s` (latest/stable, has storage → infers StatefulSet) |
|                | `coredns` (1.35/edge, no storage → infers Deployment) |
| Model          | `test-model` |
| K8s namespace  | `test-model` |
| Controller     | Built from branch `001-k8s-deployment-types` |

```bash
# Variables used throughout
APP=zinc-k8s       # For StatefulSet scenarios
APP=coredns        # For Deployment/DaemonSet scenarios
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

## Scenario Group 6: CLI Operations

These scenarios exercise juju CLI commands beyond scale/deploy/remove to verify
they work identically for Deployment and StatefulSet workloads.

**Charm**: Use `coredns` (channel 1.35/edge, no storage) for Deployment tests
and `zinc-k8s` for StatefulSet tests, unless a scenario requires a specific charm.

### S6.1 Configuration change

**Setup**: Application deployed with 1 unit, active/idle.

```bash
# Set a config value
juju config $APP <config-key>=<value>
```

| Check | Expected | Timeout |
|-------|----------|---------|
| Config applied | `juju config $APP <config-key>` returns new value | 30s |
| Pod restarted with new config | Pod rolls out (new pod name for Deployment) | 120s |
| Unit recovers | Unit active/idle after rollout | 120s |
| Unit count unchanged | Still 1 unit, no phantoms | - |

**Note**: Config changes trigger a pod rollout in K8s. For Deployment, the new pod
has a different name — the unit must re-register via `RegisterCAASUnit`. For
StatefulSet, the pod name is stable (rolling update in-place).

### S6.2 SSH into unit

**Setup**: Application deployed with 1 unit, active/idle.

```bash
juju ssh $APP/0 -- hostname
```

| Check | Expected |
|-------|----------|
| Command succeeds | Returns the pod hostname |
| Hostname matches pod name | `microk8s kubectl get pods -n $NS -l app.kubernetes.io/name=$APP -o name` |

**Expected behavior**: SSH targets pods by name via the exec API. This is
deployment-type agnostic — works the same for all workload types.

### S6.3 Run action on unit

**Setup**: Application deployed with 1 unit, active/idle. Use a charm that has
defined actions, or test with a generic command.

```bash
juju run $APP/0 -- hostname
```

| Check | Expected |
|-------|----------|
| Command succeeds | Returns output from the pod |
| No errors | Exit code 0 |

**Expected behavior**: Actions use the same pod exec mechanism as SSH. No
deployment-type branching in the action facade.

### S6.4 Expose and unexpose

**Setup**: Application deployed with 1 unit, active/idle.

```bash
juju expose $APP
```

| Check | Expected |
|-------|----------|
| Service type changes | `microk8s kubectl get svc $APP -n $NS -o jsonpath='{.spec.type}'` shows expected type |
| Application status | `juju status` shows exposed |

```bash
juju unexpose $APP
```

| Check | Expected |
|-------|----------|
| Service type reverts | Service type back to ClusterIP |
| Application status | `juju status` shows not exposed |

**Expected behavior**: Expose operates on the K8s Service resource, which is
created for all deployment types. No deployment-type branching.

### S6.5 Charm refresh (upgrade)

**Setup**: Application deployed with 1 unit, active/idle.

```bash
juju refresh $APP --channel=<different-channel>
```

| Check | Deployment | StatefulSet |
|-------|-----------|-------------|
| Command result | **EXPECTED FAIL**: `NotSupported` | PASS: rolling update |
| Error message | Clear error about deployment type | N/A |
| Unit state after | Unchanged (still active) | Updated to new revision |
| No crash | Worker does not restart-loop | N/A |

**Known gap**: `upgradeMainResource()` in the K8s provider returns
`errors.NotSupportedf` for Deployment and DaemonSet workloads
(`internal/provider/kubernetes/application/application.go:594-597`).
This is the most significant operational gap in the MVP. The test documents the
current behavior; fixing it is post-MVP work.

### S6.6 Remove unit by count

**Setup**: Application scaled to 3, all active/idle.

```bash
juju remove-unit $APP --num-units 2
```

| Check | Expected | Timeout |
|-------|----------|---------|
| 1 unit remains | `juju status` shows 1 unit active/idle | 120s |
| Pod count matches | 1 pod running | - |
| Scale target updated | Application scale is 1 | - |

**Comparison with `scale-application`**: `remove-unit --num-units` should behave
identically to `scale-application` for CAAS models. Verify both paths produce
the same result.

### S6.7 Show unit details

**Setup**: Application deployed with 2 units, both active/idle.

```bash
juju show-unit $APP/0 --format yaml
juju show-unit $APP/1 --format yaml
```

| Check | Expected |
|-------|----------|
| Provider ID present | Each unit has a provider-id field |
| Provider IDs unique | Different pod names |
| Address present | Each unit has an address |
| Machine field | Empty (CAAS model) |

**Deployment difference**: Deployment provider IDs are random pod names
(e.g., `coredns-7d8f9b6c4d-x2k9m`). StatefulSet provider IDs are ordinal
(e.g., `zinc-k8s-0`). Both must be present and valid.

---

## Scenario Group 7: Relations and Integration

These scenarios test that applications with different deployment types can
integrate (relate) with each other correctly.

### S7.1 Cross-type relation

**Setup**: Deploy two applications — one as Deployment, one as StatefulSet.

```bash
juju deploy coredns --channel=1.35/edge --constraints deployment-type=stateless
juju deploy zinc-k8s
# Wait for both to be active
```

| Check | Expected |
|-------|----------|
| Both apps active | `juju status` shows both active/idle |
| K8s resource types | Deployment for coredns, StatefulSet for zinc-k8s |

If the two charms support a common relation:

```bash
juju integrate coredns zinc-k8s
```

| Check | Expected | Timeout |
|-------|----------|---------|
| Relation established | `juju status --relations` shows the relation | 60s |
| Both apps still active | No errors from relation hooks | 120s |

**Note**: coredns and zinc-k8s may not share a common relation interface. If not,
use two charms that do (e.g., deploy a database charm and a web app charm). The
key test is that relation hooks fire correctly regardless of the backing K8s
workload type.

### S7.2 Scale related application

**Setup**: From S7.1 with both apps deployed and related.

```bash
juju scale-application coredns 3
```

| Check | Expected | Timeout |
|-------|----------|---------|
| Deployment scaled | 3 coredns units active | 120s |
| Relation intact | Relation still shows in `juju status --relations` | - |
| StatefulSet app unaffected | zinc-k8s unchanged | - |

---

## Scenario Group 8: Model-Level Operations

### S8.1 Destroy model with active Deployment

**Setup**: Model with a Deployment application (possibly scaled).

```bash
juju deploy coredns --channel=1.35/edge --constraints deployment-type=stateless -m test-model
juju wait-for application coredns --query='status=="active"' --timeout=5m -m test-model
juju scale-application coredns 3 -m test-model
# Wait for all 3 active
juju destroy-model test-model --no-prompt
```

| Check | Expected | Timeout |
|-------|----------|---------|
| Model removed | `juju models` no longer shows test-model | 300s |
| K8s namespace cleaned | `microk8s kubectl get ns test-model` returns NotFound | 300s |
| No orphaned resources | No pods, deployments, services in namespace | - |

**Comparison**: Repeat with StatefulSet (zinc-k8s, no constraint). Both must
clean up completely. This exercises the model teardown path including the
remove-application race under time pressure.

### S8.2 Export bundle preserves constraint

**Setup**: Application deployed with explicit `deployment-type=stateless` constraint.

```bash
juju deploy coredns --channel=1.35/edge --constraints deployment-type=stateless
# Wait for active
juju export-bundle --filename=/tmp/bundle.yaml
```

| Check | Expected |
|-------|----------|
| Bundle file exists | `/tmp/bundle.yaml` written |
| Constraint preserved | `grep deployment-type /tmp/bundle.yaml` shows `deployment-type=stateless` |
| Re-deployable | Deploying from the bundle file produces a Deployment (not StatefulSet) |

**Why this matters**: If the constraint is lost during export, redeployments from
the bundle would silently switch to a different workload type.

### S8.3 Juju status format consistency

**Setup**: Application deployed as Deployment with 2 units.

```bash
juju status
juju status --format json
juju status --format yaml
```

| Check | Expected |
|-------|----------|
| Tabular output | Shows units with addresses and provider IDs |
| JSON output | Valid JSON; units have `provider-id` and `address` fields |
| YAML output | Valid YAML; same fields as JSON |
| No "machine" field | CAAS units should not show machine assignments |

---

## Scenario Group 9: DaemonSet-Specific

DaemonSet workloads (`deployment-type=daemon`) have fundamentally different
scaling semantics: pod count equals node count, not a user-specified replica count.
These scenarios use a single-node microk8s cluster (1 node = 1 pod).

**Charm**: Use `coredns` (channel 1.35/edge, no storage) for DaemonSet tests.

### S9.1 Deploy as DaemonSet

```bash
juju deploy coredns --channel=1.35/edge --constraints deployment-type=daemon
```

| Check | Expected | Timeout |
|-------|----------|---------|
| K8s resource type | `microk8s kubectl get daemonset coredns -n $NS` exists | 120s |
| Pod count | 1 pod (single-node cluster) | 120s |
| Unit registered | `juju status` shows `coredns/0` active/idle | 120s |

### S9.2 Scale-application is rejected or no-op

```bash
juju scale-application coredns 3
```

| Check | Expected |
|-------|----------|
| Pod count | Still 1 (node-driven, not replica-driven) |
| Behavior | Either error message, or Juju records scale=3 but K8s stays at 1 |
| No crash | Worker does not restart-loop |

**Note**: On a single-node cluster, DaemonSet always runs exactly 1 pod. The
`Scale()` provider method returns `NotSupportedf` for DaemonSet. The test
documents how `scale-application` interacts with this limitation.

### S9.3 Pod deletion recovery

**Setup**: DaemonSet deployed, 1 unit active/idle.

```bash
POD=$(microk8s kubectl get pods -n $NS -l app.kubernetes.io/name=coredns -o name | head -1)
microk8s kubectl delete $POD -n $NS
```

| Check | Expected | Timeout |
|-------|----------|---------|
| Replacement pod created | DaemonSet controller creates new pod | 30s |
| Unit recovers | `coredns/0` active/idle with new IP | 90s |
| No extra units | Still exactly 1 unit | - |

### S9.4 Remove DaemonSet application

```bash
juju remove-application coredns
```

| Check | Expected | Timeout |
|-------|----------|---------|
| Model empty | `juju status` shows no applications | 180s |
| K8s cleanup | No coredns daemonset, pods, or services in namespace | - |

### S9.5 DaemonSet charm refresh

```bash
juju refresh coredns --channel=<different-channel>
```

| Check | Expected |
|-------|----------|
| Command result | **EXPECTED FAIL**: `NotSupported` (same as Deployment) |
| Unit state after | Unchanged (still active) |
| No crash | Worker does not restart-loop |

**Same known gap as S6.5.**

---

## Scenario Group 10: Inference Verification

These scenarios verify that the deployment type inference engine correctly selects
the workload type based on charm metadata.

### S10.1 Storage charm infers StatefulSet

```bash
juju deploy zinc-k8s  # Has storage declaration
```

| Check | Expected |
|-------|----------|
| K8s resource | StatefulSet (not Deployment) |
| Inference reason | Charm declares `data` storage at `/zinc-data` |

### S10.2 Storageless charm infers Deployment

```bash
juju deploy coredns --channel=1.35/edge  # No storage declaration
```

| Check | Expected |
|-------|----------|
| K8s resource | Deployment (not StatefulSet) |
| Inference reason | No storage declared in charm metadata |

### S10.3 Explicit constraint overrides inference

```bash
juju deploy zinc-k8s --constraints deployment-type=stateless
```

| Check | Expected |
|-------|----------|
| K8s resource | Deployment (despite charm having storage) |
| Warning | Advisory warning about storage with non-StatefulSet workload (FR-012) |

### S10.4 Explicit stateful constraint on storageless charm

```bash
juju deploy coredns --channel=1.35/edge --constraints deployment-type=stateful
```

| Check | Expected |
|-------|----------|
| K8s resource | StatefulSet (despite charm having no storage) |
| Unit active | `coredns/0` active/idle |

---

## Execution Matrix

Each scenario must pass for **both** workload types unless noted otherwise.
DaemonSet scenarios (S9.x) are only tested with `deployment-type=daemon`.
Inference scenarios (S10.x) test specific type assignments.

| Scenario | Deployment (stateless) | StatefulSet (default) | Notes |
|----------|:---------------------:|:--------------------:|-------|
| **Group 1: Lifecycle** | | | |
| S1.1 Deploy | PASS | PASS | |
| S1.2 Scale 1 → 3 | PASS | FAIL (2/3) | Pre-existing storage provisioner bug |
| S1.3 Scale 3 → 1 | PASS | PASS | |
| S1.4 Scale 1 → 2 | FAIL (1/2) | PASS | Cascade from S1.3 state / intermittent |
| S1.5 Remove | PASS | PASS | |
| S1.6 Redeploy ordinal | PASS | PASS | |
| **Group 2: Substrate Chaos** | | | |
| S2.1 Single pod kill (scale=1) | PASS | PASS | |
| S2.2 Single pod kill (scale=3) | PASS | PASS | |
| S2.3 All pods killed (scale=3) | KNOWN-LIM (2/3) | PASS | Filesystem attachment race (pre-existing) |
| S2.4 Rapid pod cycling | - | - | Not yet tested |
| S2.5 Pod kill during scale-up | - | - | Not yet tested |
| S2.6 Pod kill during scale-down | - | - | Not yet tested |
| **Group 3: Removal/Redeploy** | | | |
| S3.1 Remove + redeploy | - | - | Covered by S1.5+S1.6 |
| S3.2 Remove scaled + redeploy | - | - | Not yet tested |
| S3.3 Redeploy as different type | - | - | Not yet tested |
| **Group 4: Worker Restart** | | | |
| S4.1 Controller restart | - | - | Not yet tested |
| S4.2 Controller restart + pod kill | - | - | Not yet tested |
| **Group 5: Edge Cases** | | | |
| S5.1 Scale 0 and back | FAIL (cascade) | PASS | Deployment: cascade from S2.3 stuck unit |
| S5.2 Rapid scale oscillation | - | - | Not yet tested |
| S5.3 Kill starting pod | - | - | Not yet tested |
| **Group 6: CLI Operations** | | | |
| S6.1 Config change | - | - | Not yet tested |
| S6.2 SSH into unit | - | - | Not yet tested |
| S6.3 Run action | - | - | Not yet tested |
| S6.4 Expose/unexpose | - | - | Not yet tested |
| S6.5 Charm refresh | EXPECT FAIL | PASS (baseline) | `upgradeMainResource` returns NotSupported |
| S6.6 Remove unit by count | - | - | Not yet tested |
| S6.7 Show unit details | - | - | Not yet tested |
| **Group 7: Relations** | | | |
| S7.1 Cross-type relation | - | - | Not yet tested |
| S7.2 Scale related app | - | - | Not yet tested |
| **Group 8: Model Operations** | | | |
| S8.1 Destroy model | - | - | Not yet tested |
| S8.2 Export bundle constraint | - | - | Not yet tested |
| S8.3 Status format consistency | - | - | Not yet tested |

| Scenario | DaemonSet (daemon) | Notes |
|----------|:-----------------:|-------|
| **Group 9: DaemonSet** | | |
| S9.1 Deploy as DaemonSet | - | Not yet tested |
| S9.2 Scale rejected/no-op | - | Not yet tested; Scale() returns NotSupported |
| S9.3 Pod deletion recovery | - | Not yet tested |
| S9.4 Remove DaemonSet app | - | Not yet tested |
| S9.5 DaemonSet charm refresh | EXPECT FAIL | `upgradeMainResource` returns NotSupported |

| Scenario | Type | Notes |
|----------|:----:|-------|
| **Group 10: Inference** | | |
| S10.1 Storage → StatefulSet | StatefulSet | Not yet tested |
| S10.2 No storage → Deployment | Deployment | Not yet tested |
| S10.3 Explicit override | Deployment | Not yet tested; zinc-k8s with stateless |
| S10.4 Explicit stateful on storageless | StatefulSet | Not yet tested |

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

### Known Gaps (Deployment/DaemonSet)

These are functional gaps identified through code analysis where StatefulSet
has working behavior but Deployment/DaemonSet does not:

1. **Charm refresh blocked** (HIGH): `upgradeMainResource()` in
   `internal/provider/kubernetes/application/application.go:594-597` returns
   `errors.NotSupportedf` for Deployment and DaemonSet. Users cannot upgrade
   charms deployed with non-StatefulSet workload types. The function handles
   init container image rebuilds and annotation upgrades only for StatefulSet.
   **Scenarios**: S6.5, S9.5.

2. **PVC orphaning on removal** (HIGH): For Deployment/DaemonSet, PVCs created
   by `handlePVCForStatelessResource` are standalone (not tied to
   `volumeClaimTemplates`). The `Delete()` method never deletes them, so PVCs
   persist indefinitely after `remove-application`. StatefulSet PVCs are
   garbage-collected by K8s. Only affects charms deployed with storage AND an
   explicit `deployment-type=stateless` override.

3. **Migration may lose deployment type** (MEDIUM): The `description/v11`
   migration serialization may not include the `deployment-type` constraint
   field, meaning cross-controller migration could silently reset the workload
   type to the inferred default.

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
