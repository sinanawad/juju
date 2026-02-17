# Juju CI Test Suite: Feature Coverage Analysis (v2)

> Comprehensive cross-reference of Juju's feature surface against existing test
> coverage. Based on deep codebase research across 7 capability domains plus
> analysis of the norma-k8s calibration charm.
>
> **Updated**: 2026-02-15 (v2 — replaces surface-level inventory with per-capability implementation+test analysis)

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
