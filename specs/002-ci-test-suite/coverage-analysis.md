# Juju CI Test Suite: Feature Coverage Analysis (v4)

> Comprehensive cross-reference of Juju's feature surface against existing test
> coverage. Based on deep codebase research across 7 capability domains plus
> analysis of the norma-k8s calibration charm and per-suite audit of all 48
> integration test suites.
>
> **Updated**: 2026-02-17 (v4 — adds quality audit scores, implicit DAG mapping, migration plans, and charm contract handoff)

## Methodology

Eight parallel research agents examined the codebase, each covering a distinct capability domain:

| Domain | Scope |
|--------|-------|
| Deployment & App Lifecycle | deploy, refresh, config, expose, scale, remove, run/exec, ssh |
| Relations & Integrations | integrate, CMR (offer/consume), subordinates, peer, hooks, egress |
| Model & Controller Mgmt | bootstrap, add/destroy-model, users, permissions, upgrade, migrate |
| Cloud, Credentials & Network | clouds, credentials, spaces, subnets, firewall, endpoint binding |
| Secrets, Storage & Resources | secrets (user + charm), secret backends, storage lifecycle, resources |
| K8s/CAAS Specific | StatefulSet, Deployment, DaemonSet, Pebble, RBAC, PVC, providers |
| Constraints, Machines & Agents | constraint types, machines, containers, manual provisioning, agent upgrade |
| Norma-K8s Charm | Calibration charm capabilities and CI test suitability |

Each capability was rated:
- **GOOD**: Unit tests + integration tests cover main flows
- **PARTIAL**: Some tests exist but key scenarios missing
- **MINIMAL**: Only basic unit tests or incidental coverage
- **NONE**: No test coverage found

---

## 1. Coverage Scorecard

### By Domain

| Domain | Capabilities | GOOD | PARTIAL | MINIMAL | NONE | Score |
|--------|-------------|------|---------|---------|------|-------|
| Deployment & App Lifecycle | 30 | 24 (80%) | 4 (13%) | 2 (7%) | 0 | 80% |
| Relations & Integrations | 15 | 9 (60%) | 5 (33%) | 0 | 1 (7%) | 60% |
| Model & Controller Mgmt | 22 | 9 (41%) | 10 (45%) | 3 (14%) | 0 | 41% |
| Cloud, Credentials & Network | 25 | 14 (56%) | 5 (20%) | 6 (24%) | 0 | 56% |
| Secrets, Storage & Resources | 30 | 27 (90%) | 3 (10%) | 0 | 0 | 90% |
| K8s/CAAS Specific | 18 | 8 (44%) | 6 (33%) | 2 (11%) | 2 (11%) | 44% |
| Constraints, Machines & Agents | 35 | 22 (63%) | 8 (23%) | 5 (14%) | 0 | 63% |
| **Total** | **175** | **113 (65%)** | **41 (23%)** | **18 (10%)** | **3 (2%)** | **65%** |

### By Test Type

| Test Type | Description | Coverage |
|-----------|-------------|----------|
| Unit tests (Go) | Package-level `_test.go` files | ~85% of implemented features |
| Integration tests (bash) | `tests/suites/` shell scripts | ~35% of features (only 6 of 50 suites automated in CI) |
| CI automation (GH Actions) | Automated on PR/push | ~12% of features |

---

## 2. Deployment & Application Lifecycle

### Strong Coverage (GOOD)

| Capability | Implementation | Tests | Notes |
|------------|---------------|-------|-------|
| deploy (charms) | `cmd/juju/application/deploy.go` → facade → domain | Unit (37 cases) + integration (`deploy_charms.sh`) | Placement, series, LXD profile tested |
| deploy (bundles) | `cmd/juju/application/deployer/bundle.go` | Unit + integration (`deploy_bundles.sh`) | Overlays, fixed revisions tested |
| deploy with channels/revisions | deploy.go + deployer/charm.go | Unit + integration (`deploy_revision.sh`) | Channel parsing, revision selection |
| deploy with constraints | deploy.go + constraints | Unit + integration (`constraints/`) | Per-provider constraint tests |
| deploy with config | deploy.go → SetConfigs | Unit + integration | Config at deploy time |
| deploy with base/series | deploy.go Base field | Unit + integration | Default base, specific series |
| refresh (upgrade charm) | `cmd/juju/application/refresh.go` | Unit (1353 lines) + integration (`refresh.sh`) | Channel change, switch URL |
| config get/set | `cmd/juju/application/config.go` | Unit (20+ cases) + integration | Get, set, YAML file override |
| expose | `cmd/juju/application/expose.go` | Unit + integration (`firewall/`) | Endpoints, spaces, CIDRs |
| scale-application (CAAS) | `cmd/juju/application/scaleapplication.go` | Unit + integration (`deploy_caas/`) | Scale up/down |
| add-unit / remove-unit | addunit.go / removeunit.go | Unit + integration | Placement, force removal |
| remove-application | removeapplication.go | Unit + integration | With subordinates, relations |
| run / exec | `cmd/juju/action/run.go`, exec.go | Unit + integration (`actions/`) | Multi-unit execution |
| ssh / scp | `cmd/juju/ssh/` | Unit + integration | Proxy, key verification |
| application status | `cmd/juju/status/status.go` | Unit + integration | Formatting, filters |

### Gaps

| Capability | Rating | Gap |
|------------|--------|-----|
| deploy with resources | PARTIAL | Resource versioning conflicts, OCI auth edge cases |
| deploy with storage | PARTIAL | Storage pool selection at deploy time |
| deploy with trust | MINIMAL | No integration test for trust credential propagation |
| config unset | PARTIAL | No integration test for unset-to-default behavior |
| unexpose | MINIMAL | Only basic unit tests |
| resolved (retry hooks) | PARTIAL | Hook output context not tested |

---

## 3. Relations & Integrations

### Strong Coverage (GOOD)

| Capability | Implementation | Tests |
|------------|---------------|-------|
| integrate / add-relation | `cmd/juju/application/integrate.go` → facade → domain | Unit (15+ cases) + integration (`relations/`) |
| remove-relation | removerelation.go | Unit + integration (`cmr/`) |
| CMR offer | `cmd/juju/application/offer.go` | Unit (7+ cases) + integration (`offer_consume.sh`) |
| CMR consume | consume.go | Unit (7 cases) + integration |
| CMR remove-saas | removeremoteapplication.go | Unit + integration |
| relation data exchange | uniter facade (relation-get/set) | Unit (40+ cases) + integration (`relation_data_exchange.sh`) |
| peer relations | domain/relation + uniter | Unit + implicit in scaling tests |
| relation hooks (joined/changed/departed/broken) | `internal/worker/uniter/relation/` | Unit (extensive) + integration |
| CMR cross-controller events | `crossmodelrelations` facade | Unit (10+ cases) + integration |

### Gaps

| Capability | Rating | Gap |
|------------|--------|-----|
| relation suspension/resume | PARTIAL | Only CMR-tested; same-model suspension not tested |
| subordinate relations | PARTIAL | No end-to-end integration test for full subordinate deployment |
| network egress CIDRs | PARTIAL | Validated at API level but no traffic enforcement test |
| endpoint binding post-deploy | PARTIAL | Binding changes after deploy not tested |
| relation constraints (limit, interface) | PARTIAL | No test for endpoint limit enforcement or interface mismatch |

---

## 4. Model & Controller Management

### Strong Coverage (GOOD)

| Capability | Tests |
|------------|-------|
| bootstrap | Unit (extensive) + integration (`bootstrap/`) |
| add-model / destroy-model | Unit + integration (`model/multi.sh`, `model/destroy.sh`) |
| switch | Unit + implicit in all tests |
| models / show-model | Unit + integration |
| controllers / destroy-controller | Unit + integration (cleanup helpers) |
| add-user / remove-user / change-password | Unit + integration (`user/manage.sh`) |
| grant / revoke (model + cloud) | Unit + integration (`user/manage.sh`) |
| model-config | Unit + integration (`model/config.sh`) |
| migrate | Unit + integration (`model/migration.sh`) — covers secrets, units, relations |

### Gaps

| Capability | Rating | Gap |
|------------|--------|-----|
| disable-command / enable-command | MINIMAL | **No CLI wrapper found**; API-only with minimal unit tests; no integration tests |
| upgrade-controller | PARTIAL | HA upgrade sequencing, rollback, version skipping not covered |
| upgrade-model | PARTIAL | Multi-model orchestrated upgrades, cross-major-version blocking |
| controller-config | PARTIAL | Object store config, HA settings not tested |
| model-defaults | PARTIAL | Cloud region-specific defaults, inheritance not fully tested |
| enable-ha | PARTIAL | Failure scenarios, multi-DC failover not tested |
| register / unregister | PARTIAL/MINIMAL | Complex flows, state consistency not tested |
| login / logout | PARTIAL | SSO, macaroon auth, session cleanup not tested |

---

## 5. Cloud, Credentials & Network

### Strong Coverage (GOOD)

| Capability | Tests |
|------------|-------|
| add-cloud / remove-cloud / clouds / show-cloud | Unit (41+ tests) + integration (`cli/display_clouds.sh`) |
| add-credential / remove-credential / update-credential | Unit (44+ tests) + integration (`credential/`) |
| credentials (list) | Unit (20 tests) + integration |
| add-space / show-space / spaces (list) | Unit + integration (`spaces_ec2/`, `spaces_gce/`) |
| expose with space-level firewall rules | Unit + integration (`firewall/expose_app.sh`) |
| endpoint binding via --bind | Unit + integration (`spaces_ec2/juju_bind.sh`) |
| cloud provider suites | Integration (`cloud_azure/`, `cloud_gce/`) |
| network health | Integration (`network/network_health.sh`) |

### Gaps

| Capability | Rating | Gap |
|------------|--------|-----|
| update-cloud | MINIMAL | No integration tests; only unit tests for endpoint/region updates |
| autoload-credentials | MINIMAL | No integration test for auto-detection from environment |
| show-credential | MINIMAL | Only 7 unit tests; no integration test |
| default-credential | MINIMAL | Limited tests (9 unit tests) |
| remove-space / rename-space | PARTIAL/MINIMAL | No integration tests |
| move-to-space (subnet reassignment) | MINIMAL | 6 unit tests only; no integration validation |
| reload-spaces | MINIMAL | No unit tests (used in integration tests only) |
| add-subnet / list-subnets | MINIMAL | 6-11 unit tests; no integration tests |

---

## 6. Secrets, Storage & Resources

**This domain has the strongest coverage in the codebase.**

### Strong Coverage (GOOD) — Nearly Complete

| Capability | Tests |
|------------|-------|
| add/update/remove/show/list-secret | Unit + integration (`secrets_iaas/juju.sh`, 18KB) |
| grant-secret / revoke-secret | Unit + integration |
| secret backends (add/update/remove/show/list) | Unit + integration (`secrets_iaas/vault.sh`) |
| charm-managed secrets (all 8 hook commands) | Unit + integration |
| secret rotation & expiry | Unit + integration (daily/hourly/monthly policies) |
| K8s secrets | Integration (`secrets_k8s/k8s.sh`, 22KB) |
| storage lifecycle (add/attach/detach/remove) | Unit + integration (`storage/charm_storage.sh`) |
| storage inspection (list/show/pools) | Unit + integration |
| storage pool management (create/remove) | Unit + integration |
| K8s PVC storage | Integration (`storage_k8s/deploy.sh`, `import.sh`) |
| attach-resource | Unit + integration (`resources/basic.sh`) |
| resources (list) | Unit (17KB) + integration |
| OCI image resources | Unit + integration (`resources/containers/`) |
| deploy/refresh with resources | Unit (16KB) + integration |
| resource-get (hook command) | Unit + integration |

### Minor Gaps

| Capability | Rating | Gap |
|------------|--------|-----|
| update-storage-pool | PARTIAL | Integration test coverage minimal |
| update-secret-backend | PARTIAL | Integration tests minimal |
| Client storage facade | MINIMAL | Only 45 lines of unit tests (functionality tested via CLI) |

---

## 7. K8s/CAAS Specific

### Strong Coverage (GOOD)

| Capability | Tests |
|------------|-------|
| StatefulSet workload management | Unit (50+ test functions) + integration (`storage_k8s/`, `smoke_k8s/`) |
| Sidecar/Pebble charm deployment | Unit + integration (`sidecar/`, `deploy_caas/`) |
| K8s RBAC & service accounts | Unit + integration (`caasadmission/`) |
| K8s namespace management | Unit + integration (model isolation) |
| CAAS application provisioner | Unit (worker, ops, application tests) + indirect integration |
| MicroK8s bootstrap & add-k8s | Unit + integration (`bootstrap/`, `smoke_k8s/`) |
| Multi-container support | Unit + integration (`sidecar/`) |
| K8s annotations & labels | Unit + integration (`caasadmission/`) |

### Gaps

| Capability | Rating | Gap |
|------------|--------|-----|
| Deployment workload type | PARTIAL | Unit tests exist; **NO integration tests** |
| DaemonSet workload type | PARTIAL | Unit tests exist; **NO integration tests** |
| deployment-type constraint | PARTIAL | Core constraint added; **NO K8s integration tests** (microk8s tests skip constraints) |
| K8s service/ingress exposure | PARTIAL | Basic tests; advanced routing, LoadBalancer gaps |
| K8s PVC for Deployment/DaemonSet | MINIMAL | RWO/RWX validation missing; shared PVC untested at scale |
| CAAS firewaller | PARTIAL | No Deployment/DaemonSet-specific firewall rules tested |
| Init containers | PARTIAL | Multiple sequential init containers, failure handling |
| Pod recovery (US6) | MINIMAL | Stale pod entry cleanup, multi-pod replacement untested |
| PVC cleanup on app removal (US8) | **NONE** | Delete method doesn't clean up PVCs |
| Storage for Deployment/DaemonSet (US9-10) | MINIMAL | RWO + Deployment at scale > 1 validation missing |

---

## 8. Constraints, Machines & Agents

### Strong Coverage (GOOD)

| Capability | Tests |
|------------|-------|
| Constraint types & validation | Unit (956 lines) + integration (`constraints/`) |
| Application constraints get/set | Unit + integration |
| Model constraints get/set | Unit + integration (`constraints_model.sh`) |
| Constraint enforcement | Unit + integration (provisioner tests) |
| Add machine | Unit + integration (baseline via bootstrap) |
| Remove machine | Unit + integration (force, keep-instance, dry-run) |
| Show/list machines | Unit + integration |
| Machine cloud instance tracking | Unit + integration |
| Upgrade model/controller | Unit + integration (`upgrade/streams.sh`) |
| Agent binary management | Unit + integration |
| Unit agent hooks | Unit + integration (charm deployment tests) |

### Gaps

| Capability | Rating | Gap |
|------------|--------|-----|
| deployment-type constraint (K8s) | PARTIAL | **NO integration test** for K8s deployment-type |
| Machine containers (LXD) | MINIMAL | CLI parsing exists; **NO integration tests** for `juju add-machine lxd` |
| Manual provisioning | MINIMAL | CLI exists; **NO integration tests** for `juju add-machine ssh:user@host` |
| Machine reboot | MINIMAL | Domain tracking implemented; **NO integration tests** |
| Retry provisioning | MINIMAL | Command exists; **minimal integration coverage** |
| Agent upgrade flows | PARTIAL | No multi-machine agent version coordination tests |
| Mixed-version agents | NONE-equivalent | No tests for partially-upgraded environments |
| Agent binary syncing (offline) | MINIMAL | Limited coverage |
| Constraint merging (model+app) | PARTIAL | Merging happens at placement; limited explicit tests |

---

## 9. Norma-K8s Calibration Charm

The charm at `/data/dev/juju-norma-k8s` is a **purpose-built calibration charm** for CI testing.

### Charm Profile

| Attribute | Value |
|-----------|-------|
| Framework | Python `ops` (v3.x) |
| Type | K8s sidecar charm (2 containers) |
| Workload | Custom Go HTTP server (metrics, health toggles) |
| Actions | 17 (covering lifecycle, config, relations, storage, secrets, pebble, networking, security) |
| Events handled | 22 types |
| Config options | 5 (string, int, float, bool, secret) |
| Relations | 5 endpoints (1 peer, 2 provides, 2 requires) |
| Storage | 1 filesystem (1G, persistent) |
| Security | Non-root charm + workload, chiselled ROCK |
| Observability | Prometheus, Grafana, Loki (COS) integration |
| Test framework | `jubilant` (not pytest-operator) |

### Juju Features Exercisable via Norma

| Juju Feature | Norma Capability | Replaces |
|--------------|-----------------|----------|
| Lifecycle events | All 22 events + event ledger | noop charm |
| Config management | All 5 types with validation | Custom test charms |
| Status reporting | All 4 statuses + priority aggregation | Manual status charms |
| Actions | 17 actions with params/results | Action test charms |
| Peer relations & leadership | Multi-unit coordination, leader failover | Multi-unit test charms |
| Provides/requires relations | Bidirectional + self-relation + CMR | Relation test charms |
| Scaling | Scale up/down with data consistency | postgresql-k8s (partially) |
| Pebble workload mgmt | Service lifecycle, restart, replan | Container test charms |
| Pebble health checks | HTTP/TCP/exec + failure simulation | Health check charms |
| Pebble file ops | push/pull/make-dir/exec/remove | File operation charms |
| Pebble custom notices | Workload-to-charm signaling | Notice test charms |
| Juju secrets | Create, rotate, grant/revoke, expiry | Secret test charms |
| Storage | StatefulSet PVCs, marker files | Storage test charms |
| OCI resources | Pod restart on image refresh | container-resource charm |
| Multi-container | 2 independent containers | Multi-container test charms |
| Networking | Port management, network bindings | Network test charms |
| COS observability | Prometheus, Grafana, Loki | Observability test charms |
| Security & trust | Non-root, cloud credentials | Security test charms |
| Event deferral | Defer/re-emit ordering | Deferral test charms |
| Charm upgrade | upgrade-charm event, version tracking | Upgrade test charms |
| Introspection | Full state report action | Debug/inspection charms |

### NOT Covered by Norma

- IAAS machine lifecycle (Norma is K8s-only)
- Storage types beyond filesystem (block devices, EBS)
- LXD container placement
- Cross-provider scenarios
- TLS/certificate management (noted exception in charm spec)

---

## 10. Critical Gaps Summary

### Priority 1 — No Coverage (NONE)

| Gap | Impact | Notes |
|-----|--------|-------|
| PVC cleanup on app removal | Resource leak in K8s | `Delete()` doesn't clean up standalone PVCs |
| Backup/restore | N/A | **Not implemented in codebase** — remove from gap list |

### Priority 2 — Minimal Coverage (High Risk)

| Gap | Impact | Existing Coverage |
|-----|--------|-------------------|
| disable-command / enable-command | Command blocking untested | API-only; no CLI wrapper found |
| LXD container creation | Container provisioning untested | CLI parsing only |
| Manual machine provisioning | Manual provider untested | CLI parsing only |
| Machine reboot | Reboot recovery untested | Domain tracking only |
| Retry provisioning | Provisioning recovery untested | Command exists; minimal tests |
| deployment-type K8s constraint | New feature unvalidated | Unit tests only |
| Deployment/DaemonSet workloads | New feature unvalidated | Unit tests only |
| Pod recovery for Deployment/DaemonSet | Pod replacement broken | Stale entries not cleaned |
| Storage for Deployment/DaemonSet | RWO/RWX access mode | No validation at deploy time |
| Agent binary syncing (offline) | Offline deployment untested | Minimal coverage |

### Priority 3 — Partial Coverage (Medium Risk)

| Gap | Impact |
|-----|--------|
| Subordinate charm relations | No end-to-end integration test |
| Network egress CIDRs (via-cidrs) | API validated but traffic enforcement untested |
| Endpoint binding changes post-deploy | Mutation path untested |
| Upgrade controller/model (HA) | Sequencing, rollback untested |
| Enable-HA failure scenarios | Failover untested |
| autoload-credentials | Auto-detection from environment untested |
| Space remove/rename/move-to-space | Limited or no integration tests |
| Subnets (add/list) | No integration tests |
| CAAS firewaller for Deployment/DaemonSet | No type-specific rules tested |
| Controller-config / model-defaults | Limited integration testing |

---

## 11. Recommendations

### For the CI Test Suite Spec

1. **Remove backup/restore from gap list** — it's not implemented in the codebase
2. **Adopt norma-k8s as primary K8s test charm** — it covers 22 user stories and can replace noop, container-resource, and partially replace postgresql-k8s
3. **Prioritize Deployment/DaemonSet integration tests** — the 001-k8s-deployment-types feature has no integration test coverage
4. **Add machine lifecycle tests** — LXD containers, manual provisioning, and reboot are entirely unintegration-tested
5. **Add block command tests** — disable-command/enable-command may need CLI wrappers first

### Proposed New Test Groups

| Test Group | Covers | Priority | Charm |
|------------|--------|----------|-------|
| `deploy_caas_deployment_type` | Deployment/DaemonSet constraint, PVC, scaling | P1 | norma-k8s |
| `caas_lifecycle` | Pod recovery, PVC cleanup, workload restart | P1 | norma-k8s |
| `constraints_k8s` | deployment-type on microk8s | P1 | norma-k8s |
| `machine_containers` | LXD container creation, placement | P2 | ubuntu (IAAS) |
| `machine_manual` | Manual provisioning via SSH | P2 | ubuntu (IAAS) |
| `controller_lifecycle` | enable-ha, controller-config, block commands | P2 | norma-k8s + IAAS |
| `storage_k8s_deployment` | PVC for Deployment/DaemonSet, access modes | P2 | norma-k8s |
| `network_spaces` | Space CRUD, subnet management, binding | P3 | IAAS (EC2/GCE) |
| `credential_management` | autoload, show, default-credential | P3 | any |
| `upgrade_agents` | Agent upgrade sequencing, mixed versions | P3 | multi-unit |

### Integration Test Automation Priority

Currently only 6 of 50 suites run in CI. Recommended automation order:

1. **smoke_k8s** — already partially automated; extend with norma-k8s
2. **deploy_caas** — CAAS deployment validation
3. **constraints** — constraint enforcement verification
4. **relations** — relation lifecycle
5. **secrets_k8s** — K8s secret management (22KB test file, high value)
6. **storage_k8s** — PVC lifecycle
7. **cmr** — cross-model relations
8. **user** — user management and permissions
9. **credential** — credential management
10. **upgrade** — upgrade paths (expensive; nightly tier)

---

## 12. Per-Suite Audit Catalog (A1a)

> Generated 2026-02-17. All 48 `tests/suites/*/task.sh` files read and cataloged.

### Master Table

| # | Suite | Tests | Charms | Provider | Lifecycle | Key Capabilities |
|---|-------|-------|--------|----------|-----------|-----------------|
| 1 | actions | 2 | juju-qa-action (3P) | all | bootstrap→ensure→destroy_ctl | Action execution, params, operation/task status |
| 2 | agents | 1 | ubuntu (3P) | all | bootstrap(AGENT_TESTING)→ensure→destroy_ctl | Charm revision updater worker |
| 3 | appdata | 1 | juju-qa-appdata-{source,sink} (3P) | all | bootstrap→ensure→destroy_model | Relation appdata, unit scaling, config propagation |
| 4 | authorized_keys | 4 | none | all | ensure→bootstrap(auth-keys)→migrate→destroy_ctl | SSH key CRUD, import (GH/LP), migration |
| 5 | bootstrap | 1 | ubuntu-lite (3P) | lxd | custom bootstrap→add-model→deploy→cleanup | Simplestream metadata server, agent binary discovery |
| 6 | caasadmission | 3 | none | k8s | ensure→kubectl→destroy_model | K8s admission webhook, RBAC, label propagation |
| 7 | charmhub | 4 | juju-qa-test, ubuntu (3P) | all | find/info(no ctl)→bootstrap→deploy→destroy_ctl | Charmhub find/info/download, legacy store rejection |
| 8 | ck | 2 | charmed-kubernetes, cloud integrators (3P) | ec2/gce/azure | ensure→deploy CK→kubectl→destroy_model | Charmed K8s deployment, cloud overlays, CAAS workload |
| 9 | cli | 8 | ubuntu-lite, ntp (3P) | all | ensure→cloud/config mgmt→destroy_model | Cloud display, model-config, constraints, block commands |
| 10 | cloud_azure | 2 | ubuntu-lite, postgresql (3P) | azure | bootstrap(constraints)→HA→deploy→destroy_ctl | Managed identity, storage account type |
| 11 | cloud_gce | 5 | ubuntu, postgresql (3P) | gce | ensure→deploy→bootstrap(SA)→destroy_model | Pro images, GPU, storage pools, SA credential |
| 12 | cmr | 2 | juju-qa-dummy-{source,sink} (3P) | all | ensure→deploy→offer→consume→destroy_model/ctl | CMR single+cross-controller, offer/consume lifecycle |
| 13 | constraints | 3 | ubuntu-lite (3P) | all (per-provider) | ensure→constraints→show-machine→destroy_ctl | Model/app constraints, per-provider validation |
| 14 | controller | 3 | ubuntu-lite (3P) | all | bootstrap→metrics→HA→tracing→destroy_ctl | Controller metrics, enable-HA, DQLite quorum, tracing |
| 15 | controllercharm | 3 | prometheus-k8s (3P) | k8s (cross-ctl) | bootstrap(per-test)→deploy prom→destroy_ctl | Controller charm metrics, CMR cross-controller |
| 16 | coslite | 1 | cos-lite bundle (3P) | k8s | bootstrap→deploy COS→destroy_ctl(KILL) | COS Lite deployment, health checks |
| 17 | credential | 2 | none | all | JUJU_DATA→add/remove cred→bootstrap→destroy_ctl | Credential add/remove, client-local, controller-bound |
| 18 | dashboard | 1 | juju-dashboard (3P) | all | bootstrap→deploy dashboard→destroy_ctl | Dashboard charm, controller relation |
| 19 | deploy | 33 | 20+ charms (mix 3P+local) | all (LXD-specific) | bootstrap→multiple deploys→destroy_ctl | Charm/bundle deploy, placement, LXD profiles, revision, resources, CMR bundle |
| 20 | deploy_aks | 1 | juju-qa-dummy-{sink,source} (3P) | k8s (AKS) | bootstrap(AKS)→deploy→destroy_ctl | **SKIPPED** — AKS k8s cloud registration |
| 21 | deploy_caas | 1 | discourse-k8s, postgresql-k8s, redis-k8s, nginx-ingress (3P) | k8s | bootstrap→deploy stack→destroy_ctl | CAAS charm deploy, trust/RBAC, multi-charm |
| 22 | examples | 3 | none | all | ensure→checks→destroy_model | Template/example test patterns |
| 23 | firewall | 4 | ubuntu-lite (3P) | ec2/gce | bootstrap→provider-specific tests→destroy_ctl | SSH-allow, expose with CIDRs, endpoint exposure, security groups |
| 24 | hooks | 3 | ubuntu-plus (local), juju-qa-test (3P) | all | bootstrap→ensure→destroy_ctl | Hook dispatch, start-after-reboot, subordinate hooks, refresh hooks |
| 25 | hooktools | 1 | ubuntu-lite (3P) | all | bootstrap→ensure→destroy_ctl | state-get/set/delete, uniter state clash |
| 26 | kubeflow | 1 | kubeflow (3P) | k8s | bootstrap→deploy kubeflow→destroy_ctl(KILL) | Kubeflow deployment, metallb, ingress |
| 27 | machine | 2 | juju-qa-test (3P) | all | bootstrap→ensure→destroy_ctl | Agent logging, log permissions |
| 28 | manual | 3 | ubuntu (3P) | lxd/ec2 | create VMs→add-cloud→bootstrap→add-machine(ssh)→destroy_ctl | Manual cloud, SSH machine addition, HA |
| 29 | model | 10 | ubuntu, easyrsa, etcd, juju-qa-dummy-{src,sink} (3P) | all | bootstrap(primary+alt)→ensure→migrate→destroy_ctls | Model config, migration (cross-ver, secrets), multi-model, SAAS, metrics, destroy tracking |
| 30 | network | 2 | ubuntu, juju-qa-network-health (3P sub) | all (IP-change: lxd) | bootstrap→ensure→deploy sub→destroy_ctl | Subordinate deploy, network health, connectivity, IP change |
| 31 | ovs_maas | 1 | juju-qa-space-invader (3P) | maas | bootstrap(tags=ovs)→ensure→deploy→destroy_ctl | OVS bridge, netplan merge, space binding |
| 32 | refresh | 7 | ubuntu, juju-qa-test, juju-qa-refresher (3P) | all | bootstrap→ensure→refresh tests→destroy_ctl | Local/channel/revision refresh, resource refresh, charm switch |
| 33 | relations | 4 | juju-qa-dummy-{sink,source}, departer (local) | all | bootstrap→ensure→relation tests→destroy_ctl | Relation data exchange, departing hook, relation-list, model-get, CMR |
| 34 | resources | 5 | juju-qa-test, juju-qa-container-resource (3P) | all (container: k8s) | bootstrap→ensure→resource tests→destroy_ctl | Repo/local resources, attach, refresh, container images, large files |
| 35 | secrets_iaas | 8 | juju-qa-dummy-{source,sink} (3P) | all | bootstrap→ensure→secrets tests→destroy_ctl(KILL) | User/charm secrets, rotation, grant/revoke, CMR, vault backend, drain |
| 36 | secrets_k8s | 6 | alertmanager-k8s, hello (3P) | k8s | bootstrap→ensure→ingress→secrets→destroy_ctl(KILL) | K8s secret backend, RBAC roles, scale-down cleanup, parallel creation |
| 37 | sidecar | 6 | snappass-test, juju-qa-pebble-* (mix 3P+test) | k8s | ensure→per-test deploy→destroy_model | Pebble notices/checks, credential-get, rootless, force-remove |
| 38 | smoke | 2 | juju-qa-refresher, juju-qa-test (3P) | all | bootstrap→deploy→destroy_ctl | Build validation, basic deploy from charmhub |
| 39 | smoke_k8s | 1 | snappass-test (3P) | all | bootstrap→deploy→destroy_ctl | Basic K8s charm deploy |
| 40 | smoke_k8s_psql | 1 | postgresql-k8s, postgresql-test-app (3P) | k8s | bootstrap→deploy(trust)→integrate→destroy_ctl | PostgreSQL on K8s, relations, storage, actions |
| 41 | spaces_ec2 | 3 | space-defender (test) | ec2 | bootstrap→setup NIC→space tests→destroy_ctl | Space CRUD, machine-in-space, bind, charm refresh+bind |
| 42 | spaces_gce | 1 | none | gce | bootstrap→setup VPC→space tests→destroy_ctl | GCE VPC, space CRUD, machine-in-space |
| 43 | static_analysis | 8 | none | N/A (no bootstrap) | pure static checks | Copyright, license, doc.go, linting, schema, primary keys |
| 44 | storage | 4 | dummy-storage-* (5 local test charms) | all | bootstrap→storage tests→destroy_ctl | Pool CRUD, attach/detach, fs/block/tmpfs, persistent reattach |
| 45 | storage_k8s | 6 | postgresql-k8s (3P) | k8s | ensure→per-test→destroy_model | K8s PVC, import-filesystem, attach on deploy/add-unit, scaling race |
| 46 | unmanaged | 3 | ubuntu (3P) | lxd/ec2 | bootstrap(unmanaged)→add-machine(ssh)→deploy→destroy_ctl | Unmanaged cloud, manual SSH provisioning, HA |
| 47 | upgrade | 2 | ubuntu-lite (3P) | lxd | bootstrap(prior ver)→upgrade-controller→upgrade-model→destroy_ctl | Multi-version upgrades, agent metadata, version verification |
| 48 | user | 5 | none | all | bootstrap→ensure→user tests→destroy_ctl | User CRUD, grant/revoke, external users, disable/enable, login/password, register |

**Legend**: 3P = third-party charm, test = local test charm, sub = subordinate

### Charm Usage Summary

| Charm | Type | Used By | Replaceable by norma? |
|-------|------|---------|----------------------|
| juju-qa-test | 3P test | smoke, agents, hooks, machine, deploy, resources | Yes (norma) |
| juju-qa-dummy-{source,sink} | 3P test | appdata, cmr, model, relations, secrets_iaas | Yes (norma) |
| juju-qa-action | 3P test | actions | Yes (norma) |
| juju-qa-refresher | 3P test | smoke, refresh | Yes (norma) |
| juju-qa-appdata-{source,sink} | 3P test | appdata | Yes (norma) |
| juju-qa-pebble-* | test | sidecar | Partial (norma-k8s covers pebble) |
| juju-qa-network-health | 3P sub | network | No (subordinate-specific) |
| juju-qa-space-invader | 3P test | ovs_maas | No (MAAS-specific) |
| ubuntu-lite | 3P | smoke, cli, controller, constraints, deploy, firewall, hooktools, cloud_*, upgrade | Yes (norma machine) |
| ubuntu | 3P | agents, manual, model, network, unmanaged, cloud_gce | Yes (norma machine) |
| snappass-test | 3P | smoke_k8s, sidecar, ck | Yes (norma-k8s) |
| postgresql-k8s | 3P | smoke_k8s_psql, storage_k8s, deploy_caas | No (tests PG-specific behavior) |
| discourse-k8s | 3P | deploy_caas | Yes (norma-k8s) |
| prometheus-k8s | 3P | controllercharm | No (tests Prometheus integration) |
| cos-lite bundle | 3P | coslite | No (tests COS stack) |
| charmed-kubernetes | 3P | ck | No (tests CK deployment) |
| kubeflow | 3P | kubeflow | No (tests Kubeflow deployment) |
| juju-dashboard | 3P | dashboard | No (tests dashboard specifically) |
| ntp | 3P | cli | Yes (norma machine) |
| dummy-storage-* | local test | storage | Keep (purpose-built for storage tests) |
| departer | local test | relations | Keep (purpose-built for departing hook) |
| space-defender | test | spaces_ec2 | Keep (purpose-built for space tests) |

---

## 13. Suite-to-Capability Cross-Reference (A1a)

### Domain 1: Deployment & App Lifecycle (30 capabilities)

| Capability | Primary Suite(s) | Also Tested In |
|------------|-----------------|----------------|
| deploy (charms) | **deploy** (33 tests) | smoke, smoke_k8s, deploy_caas, most suites |
| deploy (bundles) | **deploy** | coslite, ck |
| deploy with channels/revisions | **deploy**, **refresh** | smoke |
| deploy with constraints | **deploy**, **constraints** | cli |
| deploy with config | **deploy** | appdata, cmr |
| deploy with base/series | **deploy** | hooks |
| refresh (upgrade charm) | **refresh** (7 tests) | deploy, smoke |
| config get/set | **cli**, **model** | appdata, cmr, deploy |
| expose | **firewall** | cli, network |
| scale-application (CAAS) | **deploy_caas** | secrets_k8s, storage_k8s |
| add-unit / remove-unit | **deploy** | controller, model, secrets |
| remove-application | **deploy** | sidecar, smoke_k8s_psql |
| run / exec | **actions** | hooks, machine, model, secrets |
| ssh / scp | **authorized_keys** | machine, hooks, spaces |
| application status | **cli** | all (via wait_for) |
| deploy with resources | **resources** (5 tests) | deploy |
| deploy with storage | **storage** (4 tests) | storage_k8s |
| deploy with trust | **deploy_caas** | smoke_k8s_psql |
| config unset | — | — (gap) |
| unexpose | **cli** | — |
| resolved (retry hooks) | **deploy** (resolve charm) | — |

### Domain 2: Relations & Integrations (15 capabilities)

| Capability | Primary Suite(s) | Also Tested In |
|------------|-----------------|----------------|
| integrate / add-relation | **relations** (4 tests) | appdata, cmr, deploy |
| remove-relation | **relations** | cmr, controllercharm |
| CMR offer | **cmr** (2 tests) | model, relations |
| CMR consume | **cmr** | model, relations |
| CMR remove-saas | **cmr** | — |
| relation data exchange | **relations** | appdata, cmr, model |
| peer relations | **relations** | — (implicit in scaling) |
| relation hooks | **hooks** (3 tests) | relations |
| CMR cross-controller | **cmr** | controllercharm |
| relation suspension/resume | — | — (gap) |
| subordinate relations | **network** | hooks |
| network egress CIDRs | — | — (gap, API-only) |
| endpoint binding post-deploy | **spaces_ec2** | — |
| relation constraints (limit) | — | — (gap) |

### Domain 3: Model & Controller Mgmt (22 capabilities)

| Capability | Primary Suite(s) | Also Tested In |
|------------|-----------------|----------------|
| bootstrap | **bootstrap** | smoke, upgrade, manual, unmanaged |
| add-model / destroy-model | **model** (10 tests) | most suites via ensure() |
| switch | **model** | all (via ensure) |
| models / show-model | **cli** | model, firewall |
| controllers / destroy-controller | **controller** | all (cleanup) |
| add-user / remove-user | **user** (5 tests) | — |
| grant / revoke | **user** | — |
| model-config | **cli**, **model** | most suites |
| migrate | **model** | authorized_keys |
| disable-command / enable-command | **cli** (block_commands) | — |
| upgrade-controller | **upgrade** (2 tests) | — |
| upgrade-model | **upgrade** | — |
| controller-config | **controller** (tracing) | — |
| model-defaults | **cli** | — |
| enable-ha | **controller** | cloud_gce, manual, unmanaged |
| register / unregister | **user** (register), **cli** (unregister) | — |
| login / logout | **user** (login_password) | — |

### Domain 4: Cloud, Credentials & Network (25 capabilities)

| Capability | Primary Suite(s) | Also Tested In |
|------------|-----------------|----------------|
| add-cloud / remove-cloud / clouds | **cli** (display_clouds) | manual, unmanaged, credential |
| add-credential / remove-credential | **credential** (2 tests) | — |
| credentials (list) | **credential** | cloud_gce |
| add-space / spaces | **spaces_ec2** (3 tests), **spaces_gce** | manual |
| expose with firewall rules | **firewall** (4 tests) | — |
| endpoint binding | **spaces_ec2** | ovs_maas |
| cloud provider suites | **cloud_azure**, **cloud_gce** | — |
| network health | **network** (2 tests) | — |
| update-cloud | — | — (gap) |
| autoload-credentials | — | — (gap) |
| show-credential | **cloud_gce** | credential |
| remove-space / rename-space | — | — (gap) |
| move-to-space | — | — (gap) |
| reload-spaces | **spaces_ec2**, **spaces_gce** | — |
| add-subnet / list-subnets | — | — (gap) |

### Domain 5: Secrets, Storage & Resources (30 capabilities)

| Capability | Primary Suite(s) | Also Tested In |
|------------|-----------------|----------------|
| secrets (all CRUD) | **secrets_iaas** (8 tests) | — |
| K8s secrets | **secrets_k8s** (6 tests) | — |
| secret backends | **secrets_iaas** (vault) | — |
| secret rotation & expiry | **secrets_iaas** | — |
| secret drain | **secrets_iaas**, **secrets_k8s** | — |
| storage lifecycle | **storage** (4 tests) | — |
| K8s PVC storage | **storage_k8s** (6 tests) | — |
| storage pool management | **storage**, **cloud_gce** | — |
| resources (all ops) | **resources** (5 tests) | deploy |
| OCI image resources | **resources** (container) | — |
| secret migration | **model** | — |

### Domain 6: K8s/CAAS Specific (18 capabilities)

| Capability | Primary Suite(s) | Also Tested In |
|------------|-----------------|----------------|
| StatefulSet workload | **deploy_caas**, **smoke_k8s** | storage_k8s, sidecar |
| Sidecar/Pebble | **sidecar** (6 tests) | — |
| K8s RBAC | **caasadmission** (3 tests) | secrets_k8s |
| K8s namespace management | **caasadmission** | model (via destroy) |
| CAAS provisioner | **deploy_caas** | — |
| MicroK8s bootstrap | **smoke_k8s** | most k8s suites |
| Multi-container | **sidecar** | — |
| K8s annotations/labels | **caasadmission** | — |
| Deployment workload type | — | — (**GAP: no integration test**) |
| DaemonSet workload type | — | — (**GAP: no integration test**) |
| deployment-type constraint | — | — (**GAP: no integration test**) |
| K8s service/ingress | **kubeflow** (ingress) | deploy_caas (nginx-ingress) |
| K8s PVC for Deployment/DaemonSet | — | — (**GAP**) |
| Init containers | — | — (gap, partial unit tests) |
| Pod recovery | — | — (**GAP**) |
| PVC cleanup on app removal | — | — (**GAP: NONE coverage**) |

### Domain 7: Constraints, Machines & Agents (35 capabilities)

| Capability | Primary Suite(s) | Also Tested In |
|------------|-----------------|----------------|
| Constraint types & validation | **constraints** (3 tests) | cli |
| Model constraints get/set | **constraints**, **cli** | — |
| Constraint enforcement | **constraints** | cloud_azure, cloud_gce |
| Add/remove machine | **deploy** | constraints, controller, manual |
| Machine cloud instance | **constraints** | cloud_azure, cloud_gce |
| Upgrade model/controller | **upgrade** (2 tests) | — |
| Agent binary management | **upgrade** | bootstrap |
| Unit agent hooks | **hooks** (3 tests) | deploy, refresh |
| deployment-type constraint (K8s) | — | — (**GAP**) |
| Machine containers (LXD) | **deploy** (lxd placement) | — (no `add-machine lxd` test) |
| Manual provisioning | **manual** (3 tests) | unmanaged |
| Machine reboot | **hooks** (reboot test) | — |
| Retry provisioning | — | — (gap) |
| Agent upgrade flows | **upgrade** | — (no mixed-version test) |

---

## 14. Over-Testing Analysis

Suites testing the **same capability identically** (candidates for consolidation):

| Capability | Overlap | Verdict |
|------------|---------|---------|
| Basic charm deploy | smoke + deploy + smoke_k8s + deploy_caas | **Acceptable** — different providers, different depths |
| CMR offer/consume | cmr + relations + model (migration_saas) | **Minor overlap** — cmr is comprehensive, relations tests relation-model-get, model tests CMR in migration context. Keep all. |
| Relation data exchange | relations + appdata | **Acceptable** — appdata tests file-based appdata specifically, relations tests databag access |
| User secrets | secrets_iaas + secrets_k8s + model (secret migration) | **Acceptable** — different backends (juju vs k8s vs vault), model tests migration specifically |
| Model constraints | constraints + cli (model_constraints) | **Minor overlap** — cli tests basic get/set, constraints tests enforcement per-provider. Keep both. |
| Storage pools | storage + cloud_gce (create-storage-pool) | **Acceptable** — cloud_gce tests GCE-specific pool types |
| enable-HA | controller + cloud_gce + manual + unmanaged | **Acceptable** — controller tests HA formally, others use HA as setup |
| Manual SSH provisioning | manual + unmanaged | **Significant overlap** — both test `add-machine ssh:user@host`. Consider consolidating into one suite. |

**Conclusion**: No critical over-testing requiring immediate action. The manual/unmanaged overlap is the most significant but they test different cloud types (manual cloud vs unmanaged cloud).

---

## 15. Coverage Gap Analysis (Updated)

### Gaps NOT Covered by Any Suite

| Gap | Impact | Severity | Proposed New Suite |
|-----|--------|----------|-------------------|
| Deployment workload type (K8s) | New feature entirely untested | HIGH | `deploy_caas_deployment_type` |
| DaemonSet workload type (K8s) | New feature entirely untested | HIGH | `deploy_caas_deployment_type` |
| deployment-type constraint (K8s) | New feature entirely untested | HIGH | `constraints_k8s` |
| K8s PVC for Deployment/DaemonSet | RWO/RWX access mode | HIGH | `storage_k8s_deployment` |
| PVC cleanup on app removal | Resource leak in K8s | HIGH | `deploy_caas_lifecycle` |
| Pod recovery (Deployment/DaemonSet) | Stale pod entries | MEDIUM | `deploy_caas_lifecycle` |
| relation suspension/resume | Same-model untested | LOW | enhance `relations` |
| config unset | No integration test | LOW | enhance `cli` |
| update-cloud | No integration test | LOW | enhance `credential` |
| autoload-credentials | No integration test | LOW | enhance `credential` |
| remove-space / rename-space | No integration test | LOW | enhance `spaces_ec2` |
| add-subnet / list-subnets | No integration test | LOW | enhance `spaces_ec2` |
| Init containers | Partial unit tests only | MEDIUM | enhance `sidecar` |
| Agent mixed versions | No test | MEDIUM | enhance `upgrade` |

### Gaps Partially Covered (Enhancement Opportunities)

| Gap | Current State | Enhancement |
|-----|--------------|-------------|
| Machine containers (LXD `add-machine lxd`) | deploy tests LXD placement but not `add-machine lxd` | Add to `deploy` or new `machine_containers` |
| Subordinate end-to-end | network uses sub, hooks tests sub hooks, but no full lifecycle test | Enhance `relations` |
| Retry provisioning | Command exists, minimal coverage | Enhance `constraints` or `deploy` |
| Controller-config | Only tracing tested in controller suite | Enhance `controller` |
| Model-defaults | Only basic test in cli suite | Enhance `model` |

---

## 16. Per-Suite Verdicts (A1a)

Default is **keep** or **enhance** per FR-057. Justification required for migrate or rewrite.

| Suite | Verdict | Justification |
|-------|---------|---------------|
| actions | **enhance** | Swap juju-qa-action → norma; add substrate verification |
| agents | **keep** | Unique capability (revision updater), minimal charm dependency |
| appdata | **enhance** | Swap juju-qa-appdata-* → norma self-relation; add substrate verification |
| authorized_keys | **keep** | No charms deployed; tests pure juju CLI operations |
| bootstrap | **keep** | Tests simplestream metadata (unique); minimal charm dependency |
| caasadmission | **keep** | No charms deployed; tests K8s admission (unique, well-isolated) |
| charmhub | **keep** | Tests charmhub API directly; charm deploy is incidental |
| ck | **keep** | Tests Charmed Kubernetes specifically (cannot use norma) |
| cli | **enhance** | Add config-unset test, enhance constraint coverage; charm swap minimal (ubuntu-lite → norma machine) |
| cloud_azure | **keep** | Tests Azure-specific capabilities; cannot use norma |
| cloud_gce | **keep** | Tests GCE-specific capabilities; cannot use norma |
| cmr | **enhance** | Swap juju-qa-dummy-* → norma; add substrate verification |
| constraints | **enhance** | Add deployment-type constraint test for K8s; swap ubuntu-lite → norma |
| controller | **enhance** | Add controller-config coverage; swap ubuntu-lite → norma |
| controllercharm | **keep** | Tests Prometheus integration specifically; cannot use norma |
| coslite | **keep** | Tests COS Lite bundle specifically; cannot use norma |
| credential | **enhance** | Add autoload-credentials test; no charm changes needed |
| dashboard | **keep** | Tests juju-dashboard specifically; cannot use norma |
| deploy | **enhance** | Swap some juju-qa-test → norma in applicable tests; add substrate verification to key deploy paths |
| deploy_aks | **keep** | Currently skipped; re-enable when AKS available |
| deploy_caas | **migrate** | Swap discourse-k8s/postgresql-k8s/redis-k8s → norma-k8s; current charms test charm behavior not Juju behavior. Norma-k8s provides equivalent deploy+relate+trust flow. |
| examples | **keep** | Template suite; no changes needed |
| firewall | **keep** | Tests provider-specific firewall rules; charm is incidental (ubuntu-lite) |
| hooks | **enhance** | Swap juju-qa-test → norma; add substrate verification |
| hooktools | **keep** | Tests hook tools directly; charm is incidental (ubuntu-lite) |
| kubeflow | **keep** | Tests Kubeflow specifically; cannot use norma |
| machine | **enhance** | Swap juju-qa-test → norma; minimal changes |
| manual | **keep** | Tests manual provisioning specifically; ubuntu is appropriate |
| model | **enhance** | Swap juju-qa-dummy-* → norma for migration tests; add model-defaults coverage |
| network | **keep** | Uses subordinate charm (juju-qa-network-health) specifically; cannot swap |
| ovs_maas | **keep** | Tests MAAS OVS specifically; uses space-invader charm appropriately |
| refresh | **enhance** | Some charm swaps possible (juju-qa-test → norma); core refresh logic must keep current charms |
| relations | **enhance** | Swap juju-qa-dummy-* → norma; add relation suspension/resume test |
| resources | **keep** | Tests resource system specifically; requires resource-enabled charms |
| secrets_iaas | **enhance** | Swap juju-qa-dummy-* → norma; tests are well-structured, charm swap is straightforward |
| secrets_k8s | **enhance** | Swap alertmanager-k8s → norma-k8s where feasible; some tests need specific charm behavior |
| sidecar | **keep** | Uses purpose-built pebble test charms; partially norma-k8s compatible |
| smoke | **enhance** | Swap juju-qa-refresher/juju-qa-test → norma; add substrate verification |
| smoke_k8s | **migrate** | Swap snappass-test → norma-k8s; current charm tests charm, not Juju. Norma-k8s provides equivalent K8s deploy validation. |
| smoke_k8s_psql | **migrate** | Swap postgresql-k8s → norma-k8s with storage; current suite tests PG-specific behavior. Norma-k8s storage + actions provides equivalent Juju validation. |
| spaces_ec2 | **enhance** | Add remove-space, rename-space, subnet tests; charm is appropriate |
| spaces_gce | **keep** | Tests GCE spaces specifically; well-structured |
| static_analysis | **keep** | No charms; pure static analysis |
| storage | **keep** | Uses purpose-built storage test charms; cannot swap to norma |
| storage_k8s | **migrate** | Swap postgresql-k8s → norma-k8s with storage capabilities; current suite tests PG storage behavior, not generic Juju K8s storage. |
| unmanaged | **keep** | Tests unmanaged cloud specifically; ubuntu is appropriate |
| upgrade | **keep** | Tests upgrade paths specifically; charm is incidental |
| user | **keep** | No charms; tests user management CLI |

### Verdict Summary

| Verdict | Count | Suites |
|---------|-------|--------|
| **keep** | 24 | agents, authorized_keys, bootstrap, caasadmission, charmhub, ck, cloud_azure, cloud_gce, controllercharm, coslite, dashboard, deploy_aks, examples, firewall, hooktools, kubeflow, manual, network, ovs_maas, resources, sidecar, spaces_gce, static_analysis, storage, unmanaged, upgrade, user |
| **enhance** | 20 | actions, appdata, cli, cmr, constraints, controller, credential, deploy, hooks, machine, model, refresh, relations, secrets_iaas, secrets_k8s, smoke, spaces_ec2 |
| **migrate** | 4 | deploy_caas, smoke_k8s, smoke_k8s_psql, storage_k8s |
| **rewrite** | 0 | — |

**Migration rationale**: The 4 suites verdicted as "migrate" all use heavyweight third-party charms (discourse-k8s, postgresql-k8s, snappass-test, redis-k8s) where the test validates charm behavior rather than Juju behavior. Swapping to norma-k8s (calibration charm) isolates the Juju feature being tested and reduces external dependencies per FR-037.

---

## 17. Quality Audit Scores (A1b)

> Generated 2026-02-17. Each suite scored on three quality dimensions per plan A1b.

**Scoring rubric**:

| Dimension | 2 | 1 | 0 |
|-----------|---|---|---|
| **Sterility (S)** | No charms or calibration charms | Lightweight test charms (juju-qa-*, ubuntu-lite, local test) | Heavyweight 3P charms testing charm behavior |
| **Substrate Verification (V)** | Checks substrate state (kubectl/lxc/cloud CLI) | Partial checks or setup-only substrate access | No substrate verification — Juju status only |
| **Fail-fast Readiness (F)** | Tests independent or trivial linear chain | Implicit ordering, shared state but separable | Tightly coupled, complex shared state |

### Per-Suite Quality Scores

| Suite | S | V | F | Total | Notes |
|-------|---|---|---|-------|-------|
| actions | 1 | 0 | 1 | 2 | juju-qa-action (test charm); 2 sequential tests sharing model |
| agents | 1 | 0 | 2 | 3 | ubuntu (lightweight); single test, trivial DAG |
| appdata | 1 | 0 | 2 | 3 | juju-qa-appdata (test charm); single test |
| authorized_keys | 2 | 0 | 1 | 3 | No charms; test_machine_ssh depends on test_user_ssh_keys state |
| bootstrap | 1 | 0 | 2 | 3 | ubuntu-lite (incidental); single test |
| caasadmission | 2 | 2 | 2 | 6 | No charms; heavy kubectl verification; tests independent via ensure |
| charmhub | 1 | 0 | 1 | 2 | 3P deploy incidental; 2 offline + 2 bootstrap tests, mixed ordering |
| ck | 0 | 1 | 1 | 2 | Charmed K8s (tests CK, not Juju); kubectl for CK verification |
| cli | 1 | 0 | 1 | 2 | ubuntu-lite (incidental); 8 tests, model_defaults modifies controller state |
| cloud_azure | 1 | 0 | 1 | 2 | ubuntu-lite + postgresql (incidental); sequential with shared ctl |
| cloud_gce | 1 | 0 | 1 | 2 | ubuntu + postgresql (incidental); test_serviceaccount independent bootstrap |
| cmr | 1 | 0 | 1 | 2 | juju-qa-dummy (test charm); cross-controller needs sequential |
| constraints | 1 | 0 | 1 | 2 | ubuntu-lite (incidental); per-provider execution |
| controller | 1 | 0 | 1 | 2 | ubuntu-lite (incidental); enable-HA modifies controller state |
| controllercharm | 0 | 0 | 2 | 2 | prometheus-k8s (tests Prometheus); each subtest bootstraps independently |
| coslite | 0 | 0 | 2 | 2 | cos-lite bundle (tests COS); single test |
| credential | 2 | 0 | 1 | 3 | No charms; 2 sequential tests |
| dashboard | 0 | 0 | 2 | 2 | juju-dashboard (tests dashboard); single test |
| deploy | 1 | 0 | 0 | 1 | 20+ charms (mix); 33 tests sharing single bootstrap, deeply coupled |
| deploy_aks | 0 | 0 | 2 | 2 | juju-qa-dummy (3P); single test (SKIPPED) |
| deploy_caas | 0 | 0 | 2 | 2 | discourse/postgresql/redis (3P); single test |
| examples | 2 | 0 | 2 | 4 | No charms; template tests, all independent |
| firewall | 1 | 0 | 1 | 2 | ubuntu-lite (incidental); per-provider tests, independent models |
| hooks | 1 | 0 | 1 | 2 | ubuntu-plus (local) + juju-qa-test; tests share bootstrap |
| hooktools | 1 | 0 | 2 | 3 | ubuntu-lite (incidental); single test |
| kubeflow | 0 | 1 | 2 | 3 | Kubeflow (tests KF); kubectl for KF health checks; single test |
| machine | 1 | 0 | 2 | 3 | juju-qa-test; 2 independent tests via ensure |
| manual | 1 | 1 | 1 | 3 | ubuntu (incidental); lxc for VM IP; sequential VM + bootstrap |
| model | 1 | 0 | 1 | 2 | ubuntu, juju-qa-dummy (test charms); 10 tests, metrics restarts jujud |
| network | 1 | 0 | 2 | 3 | ubuntu + sub (specific); 2 independent tests via ensure |
| ovs_maas | 1 | 0 | 2 | 3 | space-invader (specific); single test |
| refresh | 1 | 0 | 1 | 2 | ubuntu, juju-qa-* (test charms); test_basic chain is sequential |
| relations | 1 | 0 | 1 | 2 | juju-qa-dummy + departer (mixed); 4 tests, independent models |
| resources | 1 | 0 | 1 | 2 | juju-qa-test; 5 tests, independent but sequential |
| secrets_iaas | 1 | 1 | 1 | 3 | juju-qa-dummy (test charm); kubectl for K8s backend setup; 7 tests |
| secrets_k8s | 0 | 2 | 1 | 3 | alertmanager-k8s (3P); heavy kubectl verification of K8s secrets/RBAC |
| sidecar | 1 | 0 | 2 | 3 | juju-qa-pebble-* (test charms); model-per-test, fully independent |
| smoke | 1 | 0 | 1 | 2 | juju-qa-refresher + juju-qa-test; test_build → test_deploy sequential |
| smoke_k8s | 0 | 0 | 2 | 2 | snappass-test (3P); single test |
| smoke_k8s_psql | 0 | 0 | 2 | 2 | postgresql-k8s (3P); single test |
| spaces_ec2 | 1 | 0 | 1 | 2 | space-defender (test charm); 3 tests with VPC state dependency |
| spaces_gce | 2 | 0 | 2 | 4 | No charms; single test, VPC setup |
| static_analysis | 2 | 0 | 2 | 4 | No charms/bootstrap; all 8 tests fully independent |
| storage | 1 | 0 | 1 | 2 | dummy-storage-* (local test); 4 tests with shared bootstrap |
| storage_k8s | 0 | 2 | 2 | 4 | postgresql-k8s (3P); heavy kubectl PV/PVC verification; model-per-test |
| unmanaged | 1 | 1 | 1 | 3 | ubuntu (incidental); lxc for VM IP; sequential VM + bootstrap |
| upgrade | 1 | 0 | 1 | 2 | ubuntu-lite (incidental); sequential version upgrades |
| user | 2 | 0 | 1 | 3 | No charms; login_password modifies admin password |

### Score Distribution

| Total Score | Count | Suites |
|-------------|-------|--------|
| 6 | 1 | caasadmission |
| 4 | 4 | examples, spaces_gce, static_analysis, storage_k8s |
| 3 | 14 | agents, appdata, authorized_keys, bootstrap, credential, hooktools, kubeflow, machine, manual, network, ovs_maas, secrets_iaas, secrets_k8s, sidecar, unmanaged, user |
| 2 | 28 | actions, charmhub, ck, cli, cloud_azure, cloud_gce, cmr, constraints, controller, controllercharm, coslite, dashboard, deploy_aks, deploy_caas, firewall, hooks, model, refresh, relations, resources, smoke, smoke_k8s, smoke_k8s_psql, spaces_ec2, storage, upgrade |
| 1 | 1 | deploy |

### Dimension Summaries

| Dimension | Score 2 | Score 1 | Score 0 |
|-----------|---------|---------|---------|
| Sterility | 7 (no charms) | 30 (lightweight) | 11 (heavyweight 3P) |
| Substrate Verification | 3 (present) | 5 (partial) | 40 (absent) |
| Fail-fast Readiness | 21 (independent) | 26 (implicit ordering) | 1 (tightly coupled) |

### FR-045 Notes: Where Substrate Verification Is Impractical

Per FR-045, the following categories of tests cannot practically verify at the substrate level. Fallback verification strategies documented:

| Category | Suites | Why Impractical | Fallback Strategy |
|----------|--------|-----------------|-------------------|
| CLI-only operations | authorized_keys, cli, credential, user | No infrastructure created/modified | `juju status --format=json` + `jq` assertions |
| Model config/constraints | cli, constraints, model | Configuration is internal state | `juju model-config --format=json`, `juju constraints` |
| Charmhub API | charmhub | Remote API calls, no local substrate | HTTP response verification via `juju info/find` |
| Static analysis | static_analysis | Source-only, no runtime | Exit codes + output assertions |
| Template/examples | examples | Demonstration suite, no real operations | N/A |
| External cloud (integration tier) | cloud_azure, cloud_gce, spaces_ec2, spaces_gce | Cloud CLI access inconsistent in CI | `gcloud`/`aws` CLI when available; `juju show-machine` as fallback |
| Multi-controller migration | model (migration tests) | Cross-controller state hard to verify | `juju status` on both controllers + `juju show-model` |

---

## 18. Implicit Test Dependency DAGs (A1b)

> Generated 2026-02-17. Maps implicit inter-test dependencies within each suite.
> Format: `setup → {independent_tests}` or `setup → test_A → test_B` for sequential chains.

### DAG Legend

- `→` = depends on (must complete first)
- `{a, b, c}` = parallel group (no dependencies between items)
- `[P]` = provider-specific subset
- `*` = creates own bootstrap (independent of suite bootstrap)

### Trivial DAGs (1-2 tests)

| Suite | DAG | Model Isolation |
|-------|-----|-----------------|
| agents | bootstrap → test_revision_update_enabled | ensure (isolated) |
| appdata | bootstrap → test_relation_data | ensure (isolated) |
| bootstrap | custom_bootstrap → test_bootstrap_simplestreams | own bootstrap |
| coslite | bootstrap → test_coslite | ensure (isolated) |
| dashboard | bootstrap → test_dashboard | ensure (isolated) |
| deploy_aks | bootstrap → test_deploy_aks (SKIPPED) | ensure (isolated) |
| deploy_caas | bootstrap → test_deploy_charm | ensure (isolated) |
| hooktools | bootstrap → test_hooktools | ensure (isolated) |
| kubeflow | bootstrap → test_kubeflow | ensure (isolated) |
| ovs_maas | bootstrap → test_ovs | ensure (isolated) |
| smoke_k8s | bootstrap → test_deploy | ensure (isolated) |
| smoke_k8s_psql | bootstrap → test_deploy_postgresql | ensure (isolated) |
| spaces_gce | bootstrap → test_spaces_gce | ensure (isolated) |

### Small DAGs (3-5 tests)

**actions** (2 tests):
```
bootstrap → {test_actions, test_task_operations}
```
Both share model via ensure — independent.

**authorized_keys** (4 tests):
```
bootstrap → test_user_ssh_keys → test_machine_ssh
         ↘ test_bootstrap_authorized_keys*
         ↘ test_migrate_authorized_keys*
```
test_machine_ssh needs machines from test_user_ssh_keys. Bootstrap/migrate tests create own controllers.

**caasadmission** (3 tests):
```
bootstrap → {test_admission_label_propagation, test_admission_rbac, test_admission_label_propagation_new}
```
All tests independent via ensure, each creates own model.

**charmhub** (4 tests):
```
{test_charmhub_find, test_charmhub_info} → bootstrap → {test_charmhub_download, test_charmhub_deploy_lxd}
```
Find/info need no bootstrap; download/deploy need controller.

**cmr** (2 tests):
```
bootstrap → run_offer_consume → run_offer_consume_cross_controller
```
Cross-controller test bootstraps secondary controller; sequential within test_offer_consume.

**constraints** (3 tests):
```
bootstrap → {test_constraints_common, test_constraints_model*}
```
test_constraints_model creates own bootstrap with constraints.

**controller** (3 tests):
```
bootstrap → test_metrics → test_enable_ha → test_query_tracing
```
Sequential: enable-HA modifies controller state (3→1 machines), tracing runs after.

**credential** (2 tests):
```
bootstrap → test_credential_add_remove → test_credential_client_local
```
Sequential within shared bootstrap.

**examples** (3 tests):
```
bootstrap → {test_example_1, test_example_2, test_example_3}
```
All independent via ensure.

**firewall** (4 tests) [P: ec2/gce]:
```
bootstrap → {test_firewall_ssh_ec2[ec2], test_firewall_ssh_gce[gce], test_expose_app_ec2[ec2], test_bundle_with_exposed_endpoints}
```
All independent models; provider-gated subtests.

**hooks** (3 tests):
```
bootstrap → {test_dispatching_script, test_start_hook_fires_after_reboot}
```
Independent — each creates own models. Subordinate hook test is within dispatching.

**machine** (2 tests):
```
bootstrap → {test_machine_logging, test_machine_logging_permissions}
```
Both independent via ensure.

**manual** (3 tests):
```
test_deploy_manual* [lxd/ec2] → cleanup
test_spaces_manual* [ec2] → cleanup
```
Each test creates own VMs, bootstrap, and controller. Fully independent.

**network** (2 tests):
```
bootstrap → {test_network_health, test_ip_change}
```
Independent via ensure. IP change is LXD-specific.

**relations** (4 tests):
```
bootstrap → {test_relation_data_exchange, test_relation_departing_unit, test_relation_list_app, test_relation_model_get}
```
All create isolated models — fully independent.

**resources** (5 tests):
```
bootstrap → {test_basic_resources, test_upgrade_resources, test_empty_resources, test_container_resources[k8s]}
```
All create isolated models. Container resources requires K8s + podman.

**smoke** (2 tests):
```
bootstrap → test_build → test_deploy
```
Sequential: build validates `juju` binary, deploy uses it.

**storage** (4 tests):
```
bootstrap → {test_charm_storage, test_model_storage_block[iaas], test_model_storage_filesystem, test_persistent_storage}
```
All create isolated models. Block storage skips on LXD.

**unmanaged** (3 tests):
```
test_deploy_unmanaged* [lxd/ec2] → cleanup
test_spaces_unmanaged* [ec2] → cleanup
```
Each test creates own VMs and controller. Fully independent.

**upgrade** (2 tests):
```
bootstrap(prior_ver) → test_upgrade → test_upgrade_multi
```
Sequential: version upgrades must preserve controller state.

**user** (5 tests):
```
bootstrap → test_user_manage → test_user_login_password → test_user_register
```
Sequential: manage creates users, login_password changes admin password, register uses admin context.

### Complex DAGs (6+ tests)

**cli** (8 tests):
```
bootstrap → {test_display_clouds, test_unregister}
         → test_model_config
         → test_model_defaults → test_model_constraints
         → test_block_commands
```
test_model_defaults modifies controller-level defaults (saves/restores). test_display_clouds and test_unregister use isolated JUJU_DATA.

**cloud_gce** (5 tests):
```
bootstrap → {test_pro_images, test_deploy_gpu_instance, test_create_storage_pool}
test_serviceaccount_credential*
```
First 3 share bootstrap, create isolated models. SA credential bootstraps its own controller with enable-HA.

**controllercharm** (3 tests):
```
{run_prometheus*, run_prometheus_multiple_units*, run_prometheus_cross_controller*}
```
Each subtest bootstraps its own controller — fully independent.

**deploy** (33 tests — most complex):
```
bootstrap → {test_deploy_charms, test_deploy_bundles, test_cmr_bundles_export_overlay, test_deploy_revision, test_deploy_default_series}
```
Each top-level test creates isolated models via ensure. Within each, sub-tests (run_*) are sequential in shared model but each creates/destroys its own model. **All 5 top-level tests can run in parallel** after bootstrap.

**model** (10 tests):
```
bootstrap → {test_model_config, test_model_migration, test_model_migration_version, test_model_migration_saas_common, test_model_migration_saas_external, test_model_multi, test_model_status, test_model_destroy}
                                                                                                                                                                    ↓
                                                                                                                                              test_model_metrics (restarts jujud — run last)
```
Most tests create isolated models. Migration tests bootstrap alt controllers. Metrics restarts controller jujud — should run after other tests.

**refresh** (7 tests):
```
bootstrap → test_basic: run_refresh_local → run_refresh_local_resources → run_refresh_channel → run_refresh_channel_no_new_revision → run_refresh_revision
         → test_switch: run_refresh_switch_local_to_ch_channel
```
test_basic is a sequential chain (shared model state). test_switch is independent.

**secrets_iaas** (8 tests):
```
bootstrap → {test_secrets_juju, test_secrets_cmr, test_obsolete_revisions, test_secrets_vault, test_secret_drain, test_user_secret_drain, test_secrets_k8s[k8s]}
```
All create isolated models. Vault tests (3) share vault setup but each manages own model.

**secrets_k8s** (6 tests):
```
bootstrap → {test_secrets, test_user_secrets, test_secret_drain, test_user_secret_drain, test_add_multiple_secrets_parallel}
```
All create isolated models. Parallel secrets test uses KILL_CONTROLLER=true cleanup.

**sidecar** (6 tests):
```
bootstrap → {test_deploy_and_remove_application, test_deploy_and_force_remove_application, test_pebble_notices, test_pebble_checks, test_credential_get_k8s, test_rootless}
```
All fully independent — each creates own model via ensure and destroys it.

**static_analysis** (8 tests):
```
{test_copyright, test_licence, test_doc_go, test_versions, test_static_analysis_shell, test_static_analysis_python, test_schema, test_text_primary_key}
→ test_static_analysis_go (slow)
```
No bootstrap. All 8 quick tests fully independent. Go analysis runs last (slow).

**storage_k8s** (6 tests):
```
bootstrap → {test_import_filesystem, test_force_import_filesystem, test_deploy_attach_storage, test_add_unit_attach_storage, test_add_unit_duplicate_pvc_exists, test_add_unit_attach_storage_scaling_race_condition}
```
All fully independent — each creates own model via ensure.

**cloud_azure** (2 tests):
```
bootstrap(constraints) → test_managed_identity → test_storage_account_type
```
Sequential with shared bootstrap using managed identity constraints.

---

## 19. Migration Plans (A1b)

> Per-suite migration plans for the 4 suites verdicted as "migrate" in section 16.

### 19.1 deploy_caas → norma-k8s

**Current state**: Deploys discourse-k8s + postgresql-k8s + redis-k8s + nginx-ingress-integrator. Validates multi-charm K8s deployment with trust, relations, and an action call to create a Discourse user.

**Problem**: Test validates Discourse charm behavior (user creation action) rather than Juju's K8s deployment mechanics (deploy, relate, trust, scale).

**Migration plan**:
1. **Swap charms**: Replace 4 heavyweight charms with 2 norma-k8s instances:
   - `norma-k8s` as `norma-provider` (replaces discourse-k8s role)
   - `norma-k8s` as `norma-requirer` (replaces postgresql-k8s role)
   - Remove redis-k8s and nginx-ingress-integrator (not testing Juju features)
2. **Preserve Juju capabilities tested**:
   - `juju deploy` on K8s (**CC-01**: event lifecycle confirms deploy events)
   - `juju trust` (**CC-K1**: Pebble verifies workload is managed)
   - `juju integrate` (**CC-06**: calibration-provider/requirer relation)
   - Wait for active/idle via `juju status`
3. **Add substrate verification**:
   - `substrate_check_pod_exists norma-provider` after deploy
   - `substrate_check_pod_count norma-provider 1` after deploy
   - `substrate_check_namespace_gone` after destroy-model
4. **Replace action call**: Instead of `discourse-k8s/0 create-user`, use `norma-provider/0 run-check capability=relations` (**CC-04**)
5. **Add DAG**: Single test, trivial DAG — no change needed
6. **Charm contracts required**: CC-01, CC-04, CC-06, CC-K1

### 19.2 smoke_k8s → norma-k8s

**Current state**: Deploys snappass-test (revision 8, stable channel). Waits for idle. That's it.

**Problem**: Tests that snappass-test charm works, not that Juju can deploy a K8s charm. Any charm failure looks like a Juju failure.

**Migration plan**:
1. **Swap charm**: Replace `snappass-test` with `norma-k8s`
   - `juju deploy norma-k8s`
   - `wait_for "norma-k8s" "$(idle_condition "norma-k8s")"`
2. **Add verification**:
   - `juju run norma-k8s/0 run-check capability=deploy` (**CC-04**: confirms charm received install + start events)
   - `substrate_check_pod_exists norma-k8s` after deploy
3. **Preserve smoke semantics**: Keep it fast (single charm, no relations, no scaling)
4. **Add DAG**: Single test, trivial DAG — no change needed
5. **Charm contracts required**: CC-01, CC-04, CC-K1

### 19.3 smoke_k8s_psql → norma-k8s with storage

**Current state**: Deploys postgresql-k8s (trust, 16/edge) + postgresql-test-app. Integrates them. Runs continuous-write action to verify DB connectivity. Complex charm-specific behavior.

**Problem**: Tests PostgreSQL's continuous-write capability and charm-level database integration, not Juju's deploy+integrate+trust+storage mechanics.

**Migration plan**:
1. **Swap charms**: Replace postgresql-k8s + postgresql-test-app with norma-k8s pair:
   - `norma-k8s` as `norma-db` with storage: `juju deploy norma-k8s norma-db --trust --storage data=1G` (**CC-K10**)
   - `norma-k8s` as `norma-client`: `juju deploy norma-k8s norma-client`
2. **Preserve Juju capabilities tested**:
   - `juju deploy --trust` (RBAC token)
   - `juju deploy --storage` (PVC creation)
   - `juju integrate norma-db norma-client` (**CC-06**: relation lifecycle)
   - Wait for active/idle
3. **Replace continuous-write verification**:
   - `juju run norma-db/0 check-storage` (**CC-10**: confirms storage attached and writable)
   - `juju run norma-client/0 run-check capability=relations` (**CC-04**: confirms relation data received)
4. **Add substrate verification**:
   - `substrate_check_pvc_exists` for norma-db storage
   - `substrate_check_pod_count norma-db 1`
5. **Add DAG**: Single test, trivial DAG — no change needed
6. **Charm contracts required**: CC-01, CC-04, CC-06, CC-10, CC-K1, CC-K10

### 19.4 storage_k8s → norma-k8s with storage

**Current state**: Uses postgresql-k8s to test K8s PVC import-filesystem, deploy-attach-storage, add-unit-attach-storage, duplicate PVC handling, and scaling race conditions. Heavy `microk8s kubectl` usage for PV/PVC manipulation.

**Problem**: Tests PVC mechanics through postgresql-k8s lens. PG-specific storage names (pgdata), PG-specific StatefulSet behavior. Norma-k8s provides equivalent storage lifecycle with cleaner isolation.

**Migration plan**:
1. **Swap charm**: Replace `postgresql-k8s` with `norma-k8s` across all 6 sub-tests:
   - `juju deploy norma-k8s --storage data=1G` (instead of `postgresql-k8s --storage pgdata=...`)
   - Update all PVC name references from `pgdata` to `data`
   - Update all StatefulSet name references
2. **Preserve Juju capabilities tested** (all 6 tests):
   - `test_import_filesystem`: `juju import-filesystem` with retained PVs — same mechanics, different charm (**CC-K10**)
   - `test_force_import_filesystem`: `juju import-filesystem --force` — same mechanics
   - `test_deploy_attach_storage`: Deploy into retained PV — same mechanics (**CC-10**)
   - `test_add_unit_attach_storage`: `juju add-unit` with pre-existing PVs — same mechanics
   - `test_add_unit_duplicate_pvc_exists`: Label collision recovery — same mechanics
   - `test_add_unit_attach_storage_scaling_race_condition`: Race condition fix — same mechanics
3. **Keep existing substrate verification**: This suite already has excellent kubectl verification (score 2). Only update resource names.
4. **Update kubectl references**:
   - PVC names: `pgdata` → `data`
   - StatefulSet: `postgresql-k8s` → `norma-k8s`
   - Storage labels: `storage.juju.is/name` assertions stay the same
5. **Add DAG**: All 6 tests already independent (model-per-test) — just declare in predicates.yaml
6. **Charm contracts required**: CC-01, CC-04, CC-10, CC-K1, CC-K10
7. **Risk**: Import-filesystem and PV retention tests depend on specific StatefulSet PVC naming conventions. Norma-k8s must produce equivalent PVC patterns. Verify with `microk8s kubectl get pvc` after deploy.

---

## 20. Charm Contract Handoff (A1b)

> Maps calibration charm contract capabilities (CC-*) from `contracts/charm-contract.yaml` to test suite requirements.

### Contract-to-Suite Mapping

| Contract | Capability | Required By (migrate) | Required By (enhance) |
|----------|------------|----------------------|----------------------|
| **CC-01** | Event lifecycle logging | deploy_caas, smoke_k8s, smoke_k8s_psql, storage_k8s | actions, hooks, sidecar |
| **CC-02** | Configuration (all types) | — | cli, appdata, model |
| **CC-03** | Status reporting | — | smoke, deploy |
| **CC-04** | Actions with params | deploy_caas, smoke_k8s, smoke_k8s_psql, storage_k8s | actions, machine |
| **CC-05** | Peer relations | — | relations (peer test) |
| **CC-06** | Provides/requires relations | deploy_caas, smoke_k8s_psql | cmr, relations, appdata |
| **CC-07** | Cross-model relations | — | cmr |
| **CC-08** | Scaling | — | deploy (add/remove-unit tests) |
| **CC-09** | Secrets | — | secrets_iaas, secrets_k8s |
| **CC-10** | Storage (filesystem) | smoke_k8s_psql, storage_k8s | storage |
| **CC-11** | Upgrade/refresh | — | refresh |
| **CC-12** | Action error handling | — | actions |
| **CC-K1** | Pebble workload | deploy_caas, smoke_k8s, smoke_k8s_psql, storage_k8s | sidecar |
| **CC-K2** | Pebble health checks | — | sidecar (pebble_checks) |
| **CC-K3** | Pebble file/exec | — | sidecar (pebble_notices) |
| **CC-K10** | Multiple storages | smoke_k8s_psql, storage_k8s | — |
| **CC-K11** | Event deferral | — | hooks |
| **CC-M1** | Systemd service | — | machine, agents |
| **CC-M2** | SSH accessibility | — | authorized_keys |
| **CC-M3** | Machine constraints | — | constraints |
| **CC-M5** | Subordinate interface | — | network, hooks |

### Minimum Viable Contract for Migration

The 4 migrate suites require only these contracts to unblock migration:

| Contract | Priority | Blocking |
|----------|----------|----------|
| CC-01 | **P0** | All 4 suites need event lifecycle |
| CC-04 | **P0** | All 4 suites need action verification |
| CC-06 | **P1** | deploy_caas + smoke_k8s_psql need relations |
| CC-10 | **P1** | smoke_k8s_psql + storage_k8s need storage |
| CC-K1 | **P0** | All 4 suites need Pebble workload |
| CC-K10 | **P2** | smoke_k8s_psql + storage_k8s need multi-storage |

**Total**: 6 of 41 contracts needed for migration. Norma-k8s must implement CC-01, CC-04, CC-06, CC-10, CC-K1, CC-K10 before any suite can migrate.

### Handoff Documentation

The charm contract (`specs/002-ci-test-suite/contracts/charm-contract.yaml`) is the **source of truth** for norma-k8s and norma charm requirements. When norma charm development begins:

1. Feed the full contract YAML into the norma-k8s charm spec as external requirements
2. Feed the `machine:` section into the norma (IaaS) charm spec
3. Prioritize implementation of the P0 contracts (CC-01, CC-04, CC-K1) to unblock smoke_k8s migration first (simplest migration)
4. Test each contract capability against the test suite that will consume it
5. Report back contract compliance to update the `charms` section of each suite's `predicates.yaml`

**External dependency**: Norma charm development is OUT OF SCOPE for this spec. This handoff establishes the interface contract. The CI test suite can proceed with all other work (predicate system, DAG, substrate verification, classification) while norma development happens in parallel.
