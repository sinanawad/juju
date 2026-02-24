# K8s Charm Usage Research

Research document cataloging every K8s charm deployed in the Juju integration test suites, what Juju operations are exercised against each charm, and what is actually being verified.

**Scope**: All test suites under `tests/suites/` that deploy charms to a K8s provider.

---

## Summary

| Suite | K8s Charms | Juju Operations | Focus |
|-------|-----------|-----------------|-------|
| smoke_k8s | snappass-test | deploy, wait | Basic K8s smoke |
| smoke_k8s_psql | postgresql-k8s, postgresql-test-app | deploy, integrate, actions | DB relation + writes |
| deploy_caas | discourse-k8s, postgresql-k8s, redis-k8s, nginx-ingress-integrator | deploy, trust, integrate, actions | Multi-app orchestration |
| sidecar | snappass-test, juju-qa-pebble-notices, juju-qa-pebble-checks, juju-qa-credential-get-k8s, sidecar-non-root*, sidecar-sudoer* | deploy, remove, force-remove, ssh, actions, debug-log | Sidecar lifecycle, pebble, rootless |
| storage_k8s | postgresql-k8s | deploy, import-filesystem, attach-storage, add-unit, remove-unit | PV lifecycle, storage attach |
| secrets_k8s | alertmanager-k8s, nginx-ingress-integrator, snappass-test | deploy, scale, exec (secret-*), grant-secret, model-secret-backend | Secret lifecycle, drain, parallel |
| caasadmission | (none - kubectl only) | kubectl apply/patch | Admission controller labels |
| controllercharm | prometheus-k8s | deploy, offer, relate, remove-relation | Metrics endpoint, CMR |
| coslite | cos-lite bundle (alertmanager, grafana, prometheus, traefik) | deploy bundle, config, actions | Bundle deploy, health checks |
| dashboard | juju-dashboard | deploy, expose, relate | Dashboard relation (any provider) |
| ck | charmed-kubernetes bundle, snappass-test | deploy bundle, scp, add-k8s | CK on IAAS, CAAS workload |
| kubeflow | kubeflow bundle | deploy bundle, trust | Bundle deploy, UI access |
| deploy_aks | juju-qa-dummy-sink, juju-qa-dummy-source | deploy, relate, config | AKS cloud (SKIPPED) |
| resources | juju-qa-container-resource | deploy, attach-resource, refresh | OCI container resources |

\* = custom test charm packed locally

---

## 1. smoke_k8s

**Location**: `tests/suites/smoke_k8s/`

### Charms Deployed

| Charm | Channel | Source |
|-------|---------|--------|
| snappass-test | stable (rev 8) | CharmHub |

### Operations

| Operation | Command | Assertion |
|-----------|---------|-----------|
| Deploy | `juju deploy snappass-test --revision 8 --channel stable` | Deployed |
| Wait | `wait_for` idle condition | juju-status==idle, workload-status!=error |
| Destroy | `destroy_model` | Model removed |

### What It Tests
Minimal K8s deployment smoke test. Confirms Juju can deploy a sidecar charm and reach idle state. No relations, no actions, no HTTP verification.

---

## 2. smoke_k8s_psql

**Location**: `tests/suites/smoke_k8s_psql/`

### Charms Deployed

| Charm | Channel | Notes |
|-------|---------|-------|
| postgresql-k8s | 16/edge | `--trust` (cluster scope) |
| postgresql-test-app | latest/edge | base: ubuntu@22.04 |

### Operations

| Operation | Command | Assertion |
|-----------|---------|-----------|
| Deploy postgresql | `juju deploy postgresql-k8s --trust --channel 16/edge` | Deployed |
| Deploy test app | `juju deploy postgresql-test-app --channel latest/edge --base ubuntu@22.04` | Deployed |
| Integrate | `juju integrate postgresql-k8s postgresql-test-app:database` | Relation created |
| Wait | Wait for both active/idle | Both apps active |
| Wait | Check for "received database credentials" | Credentials exchanged |
| Action | `juju run postgresql-test-app/0 start-continuous-writes` | status==completed |
| Action | `juju run postgresql-test-app/0 stop-continuous-writes` | status==completed |
| Validate | Check writes > 3 | Database writes persisted |

### What It Tests
K8s database relation lifecycle. Confirms charm-to-charm integration via database endpoint, credential exchange, and actual data writes through the relation.

---

## 3. deploy_caas

**Location**: `tests/suites/deploy_caas/`

### Charms Deployed

| Charm | Channel | Notes |
|-------|---------|-------|
| discourse-k8s | latest/stable | Central app |
| postgresql-k8s | latest/stable | DB backend |
| redis-k8s | edge | Stable too old |
| nginx-ingress-integrator | latest/stable | `--trust --scope=cluster` |

### Operations

| Operation | Command | Assertion |
|-----------|---------|-----------|
| Deploy (x4) | `juju deploy <charm>` | All 4 deployed |
| Trust | `juju trust nginx-ingress-integrator --scope=cluster` | Cluster access granted |
| Integrate (x3) | discourse-k8s <-> postgresql, redis, nginx | Relations formed |
| Wait | All 4 apps active/idle | Healthy |
| Action | `juju run discourse-k8s/0 create-user admin=true email=user@example.com` | Output contains "user: user@example.com" |

### What It Tests
Multi-application K8s orchestration. Confirms 4-charm integration graph with 3 relations, trust delegation, and charm action execution. Most complex relationship topology in K8s test suites.

---

## 4. sidecar

**Location**: `tests/suites/sidecar/`

### Charms Deployed

| Charm | Source | Notes |
|-------|--------|-------|
| snappass-test | CharmHub | Deploy + HTTP verify |
| juju-qa-pebble-notices | CharmHub | Pebble notice lifecycle |
| juju-qa-pebble-checks | CharmHub | Pebble health checks |
| juju-qa-credential-get-k8s | CharmHub | K8s API credential-get |
| sidecar-non-root | Local testcharm | `--resource ubuntu=public.ecr.aws/ubuntu/ubuntu:22.04` |
| sidecar-sudoer | Local testcharm | `--resource ubuntu=public.ecr.aws/ubuntu/ubuntu:22.04` |

### Operations

**test_deploy_and_remove_application**:
- Deploy snappass-test, wait active, HTTP verify (`curl http://<addr>:5000` contains "Snappass"), `juju remove-application`, wait for 0 apps

**test_deploy_and_force_remove_application**:
- Same as above but uses `juju remove-application snappass-test --force --no-prompt`

**test_pebble_notices**:
- Deploy juju-qa-pebble-notices, wait active
- `juju ssh --container redis juju-qa-pebble-notices/0 /charm/bin/pebble notify foo.com/bar key=val`
- Wait for workload-status=="maintenance" with message "notice type=custom key=foo.com/bar"
- Repeat with second notice key

**test_pebble_checks**:
- Deploy juju-qa-pebble-checks
- Wait for maintenance + "check failed: exec-check" (initial fail state)
- `juju ssh --container ubuntu juju-qa-pebble-checks/0 mkdir /trigger/`
- Wait for active + "check recovered: exec-check"

**test_credential_get_k8s**:
- Deploy juju-qa-credential-get-k8s, `juju trust --scope=cluster`
- `juju run juju-qa-credential-get-k8s/0 hit-k8s-api-default` (in-cluster)
- `juju run juju-qa-credential-get-k8s/0 hit-k8s-api-credential-get` (credential-get)
- Assert both outputs are identical

**test_rootless** (2 sub-tests):
- `juju deploy $(pack_charm ./testcharms/charms/sidecar-non-root) --resource ubuntu=...`
- Wait idle, `juju debug-log --replay` contains "charm=170", "sudo=no", "rootless=10000", "rootful=0"
- Same for sidecar-sudoer: "charm=171", "sudo=yes", "rootless=10000", "rootful=0"

### What It Tests
Comprehensive sidecar charm lifecycle. Covers: normal removal, forced removal, Pebble notice/check event handling, K8s credential-get vs in-cluster credentials, and rootless/sudoer execution modes. This is the deepest sidecar-specific test suite.

---

## 5. storage_k8s

**Location**: `tests/suites/storage_k8s/`

### Charms Deployed

| Charm | Channel | Notes |
|-------|---------|-------|
| postgresql-k8s | 14/stable | `--trust`, scaled to 1 or 3 units |

### Operations

**test_import_filesystem**:
- Deploy postgresql-k8s, wait for pgdata/0 storage
- Capture PV provider-id via `juju storage --format json`
- `juju remove-application`, `juju remove-storage pgdata/0 --no-destroy`
- Patch PV reclaim policy via kubectl, delete PVC, clear claimRef
- `juju import-filesystem kubernetes <PV> pgdata`

**test_force_import_filesystem**:
- Same as above but with `juju import-filesystem kubernetes <PV> pgdata --force`
- Tests label mismatch scenarios

**test_deploy_attach_storage**:
- Import PV in second model, deploy with `--attach-storage pgdata/0`
- Verify PV is Bound, PVC labels correct (`storage.juju.is/name`, `app.kubernetes.io/managed-by`)

**test_add_unit_attach_storage**:
- Deploy with 3 units, capture all 3 PVs
- Import in second model, deploy with `--attach-storage pgdata/0`
- `juju add-unit psql-k8s --attach-storage pgdata/1`
- `juju add-unit psql-k8s --attach-storage pgdata/2`
- Verify all PVs bound with correct labels

**test_add_unit_duplicate_pvc_exists**:
- Tests scaling failure when PVC has incorrect labels
- Patches PVC label to `storage.juju.is/name=not-pgdata`, verifies scaling blocks
- Restores label, verifies scaling succeeds

**test_add_unit_attach_storage_scaling_race_condition**:
- Rapidly add then remove units to test race conditions
- `juju add-unit` x2 then `juju remove-unit --num-units 2` then `--num-units 1`
- Verifies storage detaches correctly

### What It Tests
K8s PersistentVolume lifecycle management. Covers: PV import, reclaim policy handling, storage attachment across models, multi-unit storage, PVC label verification, and scaling race conditions. Heavy kubectl substrate verification.

---

## 6. secrets_k8s

**Location**: `tests/suites/secrets_k8s/`

### Charms Deployed

| Charm | Test | Notes |
|-------|------|-------|
| alertmanager-k8s | run_secrets | App + unit secret ownership |
| nginx-ingress-integrator | run_secrets | Secret consumer (rev 83, `--trust`) |
| snappass-test | run_user_secrets, run_secret_drain, run_user_secret_drain | User secret lifecycle |

### Operations

**run_secrets** (application-owned secrets):
- Create model with `--config secret-backend=auto`
- `juju exec --unit alertmanager-k8s/0 -- secret-add foo=bar` (app-owned)
- `juju exec --unit alertmanager-k8s/0 -- secret-add --owner unit foo=bar2` (unit-owned)
- Verify secrets in K8s: `microk8s kubectl -n <model> get secrets`
- Scale up/down: `juju scale-application alertmanager-k8s 2` then `1` then `0`
- Verify unit secrets deleted when units removed, app secrets persist until app removed
- Grant secret via relation: `secret-grant <uri> -r <relation_id>`
- Verify consumer reads: `juju exec --unit nginx/0 -- secret-get <uri>`
- Revoke: `secret-revoke <uri> --relation <id>` and `--app nginx`
- Verify K8s RBAC role rules for secret access

**run_user_secrets**:
- `juju add-secret mysecret owned-by="<model>-1"`
- `juju show-secret mysecret --revisions --format yaml`
- `juju update-secret <uri> --info info owned-by="<model>-2"`
- `juju grant-secret <uri> snappass-test`
- `juju update-secret <uri> --auto-prune=true` (verify old revisions pruned)
- `juju exec --unit snappass-test/0 -- secret-get <uri> --peek`
- `juju exec --unit snappass-test/0 -- secret-get <uri> --refresh`
- `juju revoke-secret`, `juju remove-secret`

**run_secret_drain** / **run_user_secret_drain**:
- Deploy Vault backend, `juju add-secret-backend myvault vault endpoint=<addr> token=<token>`
- Switch backend: `juju model-secret-backend myvault`
- Verify secrets drained to Vault (K8s backend cleared)
- Switch back: `juju model-secret-backend auto`
- Verify secrets drained back to K8s

**run_test_add_multiple_secrets_parallel**:
- Create 100 secrets in parallel: `seq 1 100 | xargs -P5 -I{} juju add-secret "test{}" "foo=bar{}"`
- Verify all secret IDs exist

### What It Tests
Gold-standard secret lifecycle testing. Covers: app vs unit ownership, scaling behavior, cross-application grants via relations, K8s RBAC verification, secret revisions with auto-prune, backend drain to/from Vault, and parallel creation stress test. One of the two "gold standard" suites (with secrets_iaas).

---

## 7. caasadmission

**Location**: `tests/suites/caasadmission/`

### Charms Deployed
**None** -- this suite operates directly on K8s resources via kubectl.

### Operations

**test_controller_model_admission** / **test_new_model_admission**:
- Create ServiceAccount, Role, RoleBinding in model namespace via kubectl
- Create bearer token: `kubectl create token <name> -n <namespace>`
- Apply ConfigMap using limited-permission kubeconfig
- Verify ConfigMap gets Juju admission label: `app.juju.is/created-by=test-app`

**test_model_chicken_and_egg**:
- Delete modeloperator service: `kubectl delete svc modeloperator -n <namespace>`
- Patch modeloperator deployment with test label
- Verify deployment comes back up (model operator can restart without self-validation)

### What It Tests
K8s admission controller behavior. Confirms that Juju's admission webhook correctly labels resources created by non-Juju ServiceAccounts, and that the model operator can recover from service deletion (chicken-and-egg problem).

---

## 8. controllercharm

**Location**: `tests/suites/controllercharm/`

### Charms Deployed

| Charm | Channel | Notes |
|-------|---------|-------|
| prometheus-k8s | 1/stable | `--trust`, deployed as p1/p2 aliases |

### Operations

**run_prometheus**:
- `juju offer controller.controller:metrics-endpoint`
- `juju deploy prometheus-k8s --channel 1/stable --trust`
- `juju relate prometheus-k8s controller.controller`
- Verify controller in Prometheus targets via HTTP: `curl http://<prom-ip>:9090/api/v1/targets`
- `juju remove-relation prometheus-k8s controller`
- Verify controller removed from targets

**run_prometheus_multiple_units**:
- Deploy two instances (p1, p2) with different aliases
- Relate both to controller, `juju add-unit p1` (scale to 2)
- Verify targets for each unit
- `juju remove-unit p1 --num-units 1`, verify targets update

**run_prometheus_cross_controller**:
- Bootstrap second controller on K8s
- Deploy prometheus-k8s in second controller
- Cross-controller relation: `juju relate prometheus-k8s "${CONTROLLER_NAME}:controller.controller"`
- Verify targets across controllers

### What It Tests
Controller charm metrics integration. Confirms Prometheus can scrape Juju controller metrics via relation, handles multi-unit scaling, and works across controllers (CMR). K8s-only for 2/3 tests.

---

## 9. coslite

**Location**: `tests/suites/coslite/`

### Charms Deployed

| Charm | Channel | Notes |
|-------|---------|-------|
| cos-lite (bundle) | stable | `--trust` |
| ubuntu-lite | default | Used for HTTP verification |

Bundle components: alertmanager, grafana, prometheus, traefik

### Operations

- `juju deploy cos-lite --trust --channel=stable`
- `juju config traefik external_hostname=test-coslite.com`
- Wait for all units idle (30-minute timeout)
- `juju run grafana/0 get-admin-password --wait=2m`
- HTTP health checks via `juju ssh ubuntu-lite/0 curl`:
  - alertmanager: `http://<ip>:9093/-/ready` (200)
  - grafana: `http://<ip>:3000/api/health` (200)
  - prometheus: `http://<ip>:9090/-/ready` (200)

### What It Tests
COS Lite bundle deployment and health. Confirms the full observability stack deploys on K8s, all components reach idle, and HTTP endpoints respond. Uses `KILL_CONTROLLER=true` teardown (K8s model cleanup workaround).

---

## 10. dashboard

**Location**: `tests/suites/dashboard/`

### Charms Deployed

| Charm | Notes |
|-------|-------|
| juju-dashboard (alias: dashboard) | Deployed to controller model |

### Operations

- `juju switch controller`
- `juju deploy juju-dashboard dashboard`
- `juju expose dashboard`
- `juju relate dashboard controller`
- `juju dashboard 2>&1` -- expects "not implemented" error
- Same check from a non-controller model

### What It Tests
Dashboard charm deployment. **Not K8s-specific** -- runs on any provider. Currently only verifies the `juju dashboard` command returns "not implemented" (functionality pending reimplementation in controller charm).

---

## 11. ck (Charmed Kubernetes)

**Location**: `tests/suites/ck/`

### Charms Deployed

| Charm | Provider | Notes |
|-------|----------|-------|
| charmed-kubernetes (bundle) | IAAS (ec2/gce/azure) | `--trust`, with provider overlays |
| Provider integrators (aws-integrator, gcp-integrator, azure-integrator) | IAAS | Via bundle |
| snappass-test | K8s (CAAS workload) | Deployed to CK cluster |

### Operations

**test_deploy_ck**:
- `juju deploy charmed-kubernetes --overlay <provider> --overlay ./overlay.yaml --trust`
- Wait for kubernetes-control-plane, kubernetes-worker, integrators to reach active (1800s)
- `juju scp kubernetes-control-plane/0:config ~/.kube/config`
- `kubectl cluster-info`, `kubectl get ns`
- `juju run "$integrator_app_name/leader" --wait=10m purge-subnet-tags`

**test_deploy_caas_workload**:
- `juju add-k8s <cloud> --storage <provider-storage> --controller <name>`
- `juju add-model <name> <k8s-cloud>`
- `juju deploy snappass-test`
- Wait for idle

### What It Tests
Full Charmed Kubernetes lifecycle on IAAS. Bootstraps CK on cloud providers, extracts kubeconfig, deploys a CAAS workload to the CK cluster. Tests Juju managing K8s *and* deploying to K8s.

---

## 12. kubeflow

**Location**: `tests/suites/kubeflow/`

### Charms Deployed

| Charm | Channel | Notes |
|-------|---------|-------|
| kubeflow (bundle) | 1.9 | `--trust` |

### Operations

- MetalLB setup: `sudo microk8s enable "metallb:10.64.140.43-10.64.140.49"`
- `juju deploy kubeflow --trust --channel 1.9`
- Wait for training-operator active/idle (1800s)
- Extract Jupyter IP: `microk8s kubectl -n kubeflow get svc istio-ingressgateway-workload`
- `curl ${jupyter_ip}` contains "Found" (HTTP 302)
- `KILL_CONTROLLER=true` for teardown

### What It Tests
Kubeflow bundle deployment on K8s. Confirms the ML/AI stack deploys, training-operator activates, and Jupyter UI is accessible via LoadBalancer. K8s-only.

---

## 13. deploy_aks

**Location**: `tests/suites/deploy_aks/`

**STATUS: SKIPPED** (`if [ true ]` guard -- pending K8s tooling in strict snap)

### Charms (if enabled)

| Charm | Notes |
|-------|-------|
| juju-qa-dummy-sink | base: ubuntu@22.04 |
| juju-qa-dummy-source | base: ubuntu@22.04 |

### Operations (if enabled)
- `az aks create`, `juju add-k8s --aks`, bootstrap on AKS
- Deploy + relate dummy-sink/source
- `juju config dummy-source token=yeah-boi`, verify in dummy-sink status

### What It Would Test
AKS cloud integration. Currently disabled.

---

## 14. resources (container subset)

**Location**: `tests/suites/resources/` (specifically `container.sh`)

### Charms Deployed

| Charm | Notes |
|-------|-------|
| juju-qa-container-resource | OCI/container resource support |

### Container Images
- `localhost:5000/resource-1` (built from tests/suites/resources/containers/resource-1/)
- `localhost:5000/resource-2` (built from tests/suites/resources/containers/resource-2/)
- CharmHub app-image revisions 3 and 4

### Operations

- Start local registry: `podman run -d -p 5000:5000 registry:2.7`
- Build + push 2 container images
- `juju deploy juju-qa-container-resource --resource app-image=localhost:5000/resource-1`
- Verify workload status: "I am resource 1"
- `juju attach-resource juju-qa-container-resource app-image=localhost:5000/resource-2`
- Verify: "I am resource 2"
- `juju refresh juju-qa-container-resource --resource app-image=3` (CharmHub rev)
- Verify: "I am the charmhub resource (revision 3)"
- `juju refresh juju-qa-container-resource --resource app-image=4`
- Verify: "I am the charmhub resource (revision 4)"

### What It Tests
OCI container resource lifecycle. Confirms local registry resources, CharmHub resources, resource attachment, and resource refresh all work on K8s. Tests the juju-managed container image pipeline.

---

## Analysis

### Charm Dependency Map

Charms used by multiple suites (risk if charm breaks):

| Charm | Used By | Count |
|-------|---------|-------|
| snappass-test | smoke_k8s, sidecar, secrets_k8s, ck | 4 |
| postgresql-k8s | smoke_k8s_psql, deploy_caas, storage_k8s | 3 |
| nginx-ingress-integrator | deploy_caas, secrets_k8s | 2 |
| prometheus-k8s | controllercharm, coslite (via bundle) | 2 |
| alertmanager-k8s | secrets_k8s, coslite (via bundle) | 2 |

### Juju Operation Coverage

| Juju Operation | Suites That Exercise It |
|----------------|------------------------|
| deploy | All (except caasadmission) |
| integrate/relate | smoke_k8s_psql, deploy_caas, secrets_k8s, controllercharm, coslite, dashboard, ck, deploy_aks |
| remove-application | sidecar, storage_k8s, controllercharm, dashboard |
| remove-application --force | sidecar, controllercharm |
| scale-application | secrets_k8s |
| add-unit / remove-unit | storage_k8s, controllercharm |
| trust | smoke_k8s_psql, deploy_caas, sidecar, secrets_k8s, controllercharm, coslite, ck, kubeflow |
| actions (juju run) | smoke_k8s_psql, deploy_caas, sidecar, secrets_k8s, controllercharm, coslite |
| exec (juju exec) | secrets_k8s |
| ssh (juju ssh) | sidecar, coslite |
| config | coslite, deploy_aks |
| expose | dashboard |
| import-filesystem | storage_k8s |
| attach-storage | storage_k8s |
| attach-resource | resources |
| refresh (with resources) | resources |
| add-secret / grant-secret | secrets_k8s |
| model-secret-backend | secrets_k8s |
| offer (CMR) | controllercharm |
| add-k8s | ck |
| debug-log | sidecar |

### Key Observations

1. **snappass-test is the most reused charm** (4 suites) -- breakage would cascade widely
2. **No suite uses the calibration charm (norma-k8s)** -- all depend on real-world charms from CharmHub
3. **Substrate verification varies widely**: storage_k8s and secrets_k8s verify K8s resources directly; most others only check Juju status
4. **Forced teardown is common**: coslite, secrets_k8s, and kubeflow all set `KILL_CONTROLLER=true`, indicating K8s model cleanup issues
5. **deploy_aks is dead code** -- permanently skipped
6. **dashboard is not K8s-specific** -- runs on any provider
7. **Pebble operations are sidecar-exclusive** -- only tested in the sidecar suite
8. **Secret drain (Vault) is only tested in secrets_k8s** -- no other suite touches secret backends
9. **Bundle deployments have extreme timeouts** -- kubeflow and coslite use 1800s (30 min) waits

### Implications for Calibration Charm Design

The research reveals these Juju operations that norma-k8s would need to support for comprehensive testing:

| Priority | Capability | Currently Tested By |
|----------|-----------|-------------------|
| P1 | deploy + wait idle | All suites |
| P1 | remove-application (normal + forced) | sidecar |
| P1 | integrate (basic relation) | smoke_k8s_psql, deploy_caas |
| P1 | scale-application (up/down) | secrets_k8s |
| P1 | add-unit / remove-unit | storage_k8s, controllercharm |
| P2 | actions (juju run) | Multiple |
| P2 | secret-add / secret-get / secret-grant | secrets_k8s |
| P2 | config set/get | deploy_aks, coslite |
| P2 | trust (--scope=cluster) | Multiple |
| P2 | storage (attach, import) | storage_k8s |
| P3 | ssh (into container) | sidecar |
| P3 | pebble notices/checks | sidecar |
| P3 | container resources (OCI) | resources |
| P3 | expose | dashboard |
| P3 | CMR (cross-model offer/consume) | controllercharm |
