# Manual Testing Guide: Kubernetes Deployment Type Support (MVP)

This guide walks through manual verification of the deployment type selection
feature (Phases 1-3). It covers environment setup, building Juju from the
feature branch, deploying applications with each workload type, and verifying
the correct Kubernetes resources are created.

## Prerequisites

- A machine with Go installed (version per `go.mod`)
- DQLite build dependencies (for `make install`)
- A Kubernetes cluster (microk8s, k3s, kind, or any conformant cluster)
- `kubectl` configured to talk to the cluster
- `juju` CLI (will be built from source below)

## 1. Build Juju from the Feature Branch

```bash
cd /path/to/juju
git checkout 001-k8s-deployment-types
make install
```

Verify the binary is on your `PATH`:

```bash
juju version
```

## 2. Set Up a Kubernetes Cluster

Pick one of the options below. If you already have a cluster, skip to
section 3.

### Option A: microk8s (Ubuntu)

```bash
sudo snap install microk8s --classic --channel=1.28/stable
sudo microk8s enable dns storage
sudo microk8s status --wait-ready

# Add the cloud to Juju
juju add-k8s my-k8s --client
```

### Option B: k3s

```bash
curl -sfL https://get.k3s.io | sh -
mkdir -p ~/.kube
sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
sudo chown $(id -u):$(id -g) ~/.kube/config

juju add-k8s my-k8s --client
```

### Option C: kind

```bash
kind create cluster --name juju-test
juju add-k8s my-k8s --client
```

## 3. Bootstrap a Controller

```bash
juju bootstrap my-k8s test-ctrl
```

Wait for the controller to become ready:

```bash
juju controllers
```

## 4. Create a Test Model

```bash
juju add-model deploy-types
```

Note the Kubernetes namespace Juju creates. It will be your model name
(typically the model name itself):

```bash
kubectl get ns | grep deploy-types
```

Save it for later:

```bash
export NS=$(juju models --format=json | jq -r '.models[] | select(.name=="admin/deploy-types") | .["model-uuid"]' | cut -c1-5)-deploy-types
# Or simply:
export NS=deploy-types
```

> The exact namespace format depends on your Juju version. Use
> `kubectl get ns` to find the one containing your model name.

## 5. Create a Minimal Test Charm

We need a single charm that we can deploy in different ways. Create a
minimal OCI-based charm that has **no storage** by default but can
optionally declare storage.

### 5a. Stateless Charm (no storage)

```bash
mkdir -p /tmp/test-charm-stateless
```

Create `/tmp/test-charm-stateless/metadata.yaml`:

```yaml
name: test-webserver
summary: Minimal test charm for deployment type verification
description: |
  A minimal charm that runs an nginx container.
  Declares no persistent storage, so Juju should infer
  the "stateless" deployment type (Kubernetes Deployment).
assumes:
  - k8s-api
containers:
  webserver:
    resource: oci-image
resources:
  oci-image:
    type: oci-image
    description: OCI image for the webserver container
```

Create `/tmp/test-charm-stateless/charmcraft.yaml`:

```yaml
type: charm
bases:
  - build-on:
      - name: ubuntu
        channel: "22.04"
    run-on:
      - name: ubuntu
        channel: "22.04"
parts:
  charm:
    plugin: charm
```

Create `/tmp/test-charm-stateless/src/charm.py`:

```python
#!/usr/bin/env python3
import ops

class TestWebserverCharm(ops.CharmBase):
    def __init__(self, framework):
        super().__init__(framework)
        framework.observe(self.on["webserver"].pebble_ready, self._on_pebble_ready)

    def _on_pebble_ready(self, event):
        container = self.unit.get_container("webserver")
        container.add_layer("webserver", {
            "services": {
                "webserver": {
                    "override": "replace",
                    "command": "nginx -g 'daemon off;'",
                    "startup": "enabled",
                }
            }
        }, combine=True)
        container.autostart()
        self.unit.status = ops.ActiveStatus("ready")

if __name__ == "__main__":
    ops.main(TestWebserverCharm)
```

Pack the charm:

```bash
cd /tmp/test-charm-stateless
charmcraft pack
```

### 5b. Stateful Charm (with storage)

```bash
cp -r /tmp/test-charm-stateless /tmp/test-charm-stateful
```

Edit `/tmp/test-charm-stateful/metadata.yaml` to add storage:

```yaml
name: test-webserver-storage
summary: Test charm with persistent storage
description: |
  Same minimal charm but declares persistent storage.
  Juju should infer the "stateful" deployment type (Kubernetes StatefulSet).
assumes:
  - k8s-api
containers:
  webserver:
    resource: oci-image
    mounts:
      - storage: data
        location: /data
resources:
  oci-image:
    type: oci-image
    description: OCI image for the webserver container
storage:
  data:
    type: filesystem
    minimum-size: 1G
```

Pack the charm:

```bash
cd /tmp/test-charm-stateful
charmcraft pack
```

## 6. Test Scenarios

### Test 1: Automatic Inference - Stateless (no storage -> Deployment)

Deploy the charm with no storage declarations and no explicit constraint:

```bash
juju deploy /tmp/test-charm-stateless/test-webserver_*.charm \
  --resource oci-image=docker.io/library/nginx:latest \
  test-stateless
```

Wait for the unit to start:

```bash
juju status --watch 5s
# Wait until the application appears (it may show "waiting" while the
# container image pulls)
```

Verify that a **Deployment** was created (not a StatefulSet):

```bash
# Should return a Deployment resource
kubectl get deployment test-stateless -n "$NS"

# Should return "No resources found" — no StatefulSet for this app
kubectl get statefulset test-stateless -n "$NS" 2>&1

# Should return "No resources found" — no DaemonSet for this app
kubectl get daemonset test-stateless -n "$NS" 2>&1
```

**Expected**: `kubectl get deployment` succeeds. The other two return
"not found".

### Test 2: Automatic Inference - Stateful (has storage -> StatefulSet)

Deploy the charm that declares persistent storage:

```bash
juju deploy /tmp/test-charm-stateful/test-webserver-storage_*.charm \
  --resource oci-image=docker.io/library/nginx:latest \
  test-stateful
```

Verify:

```bash
# Should return a StatefulSet resource
kubectl get statefulset test-stateful -n "$NS"

# Should return "No resources found"
kubectl get deployment test-stateful -n "$NS" 2>&1
```

**Expected**: `kubectl get statefulset` succeeds.

### Test 3: Explicit Constraint Override - Force Stateless on Storage Charm

Deploy the storage charm but explicitly request a Deployment:

```bash
juju deploy /tmp/test-charm-stateful/test-webserver-storage_*.charm \
  --resource oci-image=docker.io/library/nginx:latest \
  --constraints="deployment-type=stateless" \
  test-forced-stateless
```

Verify:

```bash
# Should return a Deployment (constraint overrides inference)
kubectl get deployment test-forced-stateless -n "$NS"

# Should return "No resources found"
kubectl get statefulset test-forced-stateless -n "$NS" 2>&1
```

**Expected**: Deployment created despite the charm having storage. The
controller log should contain a warning about deploying a stateless
workload with persistent storage (FR-012).

Check logs for the warning:

```bash
juju debug-log --include caas-application-provisioner --replay \
  | grep -i "stateless.*storage\|storage.*stateless"
```

### Test 4: Explicit Constraint - DaemonSet

Deploy the stateless charm as a DaemonSet:

```bash
juju deploy /tmp/test-charm-stateless/test-webserver_*.charm \
  --resource oci-image=docker.io/library/nginx:latest \
  --constraints="deployment-type=daemon" \
  test-daemon
```

Verify:

```bash
# Should return a DaemonSet resource
kubectl get daemonset test-daemon -n "$NS"

# Should return "No resources found"
kubectl get deployment test-daemon -n "$NS" 2>&1
kubectl get statefulset test-daemon -n "$NS" 2>&1
```

Check that the number of pods matches the number of cluster nodes:

```bash
# Count cluster nodes
kubectl get nodes --no-headers | wc -l

# Count DaemonSet pods (should match node count)
kubectl get pods -n "$NS" -l app.kubernetes.io/name=test-daemon --no-headers | wc -l
```

**Expected**: One pod per node. DaemonSet created, no Deployment or
StatefulSet.

### Test 5: Explicit Constraint - Force Stateful on Stateless Charm

```bash
juju deploy /tmp/test-charm-stateless/test-webserver_*.charm \
  --resource oci-image=docker.io/library/nginx:latest \
  --constraints="deployment-type=stateful" \
  test-forced-stateful
```

Verify:

```bash
kubectl get statefulset test-forced-stateful -n "$NS"
kubectl get deployment test-forced-stateful -n "$NS" 2>&1
```

**Expected**: StatefulSet created even though the charm has no storage.

### Test 6: Immutability - Cannot Change Deployment Type (FR-006)

Try to change the deployment type on a running application:

```bash
juju set-constraints test-stateless deployment-type=stateful
```

**Expected**: Error message:
`deployment type cannot be changed for a running application; redeploy to use a different workload type`

### Test 7: Invalid Constraint Value

```bash
juju deploy /tmp/test-charm-stateless/test-webserver_*.charm \
  --resource oci-image=docker.io/library/nginx:latest \
  --constraints="deployment-type=invalid" \
  test-invalid
```

**Expected**: Error at deploy time listing valid values
(`stateless`, `stateful`, `daemon`).

### Test 8: Backward Compatibility - Existing Apps Default to StatefulSet

This verifies FR-011. Existing applications (deployed before this
feature) should continue running as StatefulSets.

If you have an existing Juju K8s model from before this branch:

1. Upgrade the controller to the feature branch build.
2. Verify all existing applications still show StatefulSet resources:

```bash
kubectl get statefulset -n "$NS"
```

The schema migration adds `deployment_type_id` with `DEFAULT 0`
(stateful), so all existing rows automatically get the stateful type.

### Test 9: Scaling Behavior

Verify scaling works correctly for each type:

```bash
# Deployment scales normally
juju scale-application test-stateless 3
kubectl get deployment test-stateless -n "$NS" -o jsonpath='{.spec.replicas}'
# Expected: 3

# StatefulSet scales normally
juju scale-application test-stateful 2
kubectl get statefulset test-stateful -n "$NS" -o jsonpath='{.spec.replicas}'
# Expected: 2
```

> **Note**: DaemonSet scale blocking (T021-T022) is not yet implemented.
> The K8s provider rejects DaemonSet scaling at the broker level, but
> the domain service guard that returns a clear error message to the
> user is pending (Phase 4).

## 7. Verification Cheat Sheet

One-liner to check all deployed applications and their K8s resource types:

```bash
for app in test-stateless test-stateful test-forced-stateless test-daemon test-forced-stateful; do
  TYPE="none"
  kubectl get deployment "$app" -n "$NS" &>/dev/null && TYPE="Deployment"
  kubectl get statefulset "$app" -n "$NS" &>/dev/null && TYPE="StatefulSet"
  kubectl get daemonset "$app" -n "$NS" &>/dev/null && TYPE="DaemonSet"
  echo "$app: $TYPE"
done
```

Expected output:

```
test-stateless: Deployment
test-stateful: StatefulSet
test-forced-stateless: Deployment
test-daemon: DaemonSet
test-forced-stateful: StatefulSet
```

## 8. Inspect Juju Labels on K8s Resources

All Juju-managed resources carry standard labels. Verify them:

```bash
kubectl get deployment test-stateless -n "$NS" -o jsonpath='{.metadata.labels}' | jq .
```

Expected labels include:

```json
{
  "app.kubernetes.io/managed-by": "juju",
  "app.kubernetes.io/name": "test-stateless"
}
```

List all resources for a specific application:

```bash
kubectl get all -n "$NS" -l app.kubernetes.io/name=test-stateless
```

## 9. Using Charmhub Charms Instead (Alternative)

If you prefer not to build test charms, you can use existing Charmhub
charms. Choose charms based on whether they declare storage.

### Charms without storage (should create Deployment)

```bash
# Any simple charm without storage declarations
juju deploy hello-juju --channel=latest/edge
```

### Charms with storage (should create StatefulSet)

```bash
juju deploy postgresql-k8s --channel=14/stable --trust
```

### Force a different type via constraint

```bash
juju deploy hello-juju --channel=latest/edge \
  --constraints="deployment-type=daemon" \
  hello-daemon
```

> **Caveat**: Not all Charmhub charms may be compatible with all
> deployment types. A charm designed for StatefulSet semantics (stable
> network identity, ordered deployment) may not work correctly as a
> Deployment. The constraint override is an expert feature.

## 10. Cleanup

```bash
juju destroy-model deploy-types --destroy-storage --no-prompt
juju destroy-controller test-ctrl --destroy-all-models --no-prompt
```

## What is NOT Covered by MVP

The following features are planned but not yet implemented:

| Feature | Phase | Status |
|---------|-------|--------|
| DaemonSet scale blocking (clear error on `juju scale-application`) | Phase 4 (T021-T022) | Pending |
| `juju status` showing a "Type" column for CAAS apps | Phase 5 (T023-T029) | Pending |
| Full migration round-trip for explicit constraints (daemon, overrides) | Phase 6 (T040-T043) | Blocked on `description` library |
| `computeStatus()` for Deployment/DaemonSet in K8s provider | Follow-up | Pre-existing gap |

## Troubleshooting

### Charm stays in "waiting" state

The OCI image may still be pulling. Check pod status:

```bash
kubectl get pods -n "$NS" -l app.kubernetes.io/name=<app-name>
kubectl describe pod <pod-name> -n "$NS"
```

### "deployment type" constraint not recognised

Make sure you are running the Juju binary built from the feature branch,
not a system-installed version:

```bash
which juju
juju version
```

### Finding the model namespace

```bash
juju show-model deploy-types --format=json | jq -r '."deploy-types"."model-uuid"'
# The namespace is typically: <model-name>
kubectl get ns
```

### Controller logs

```bash
juju debug-log --include caas-application-provisioner --replay
```
