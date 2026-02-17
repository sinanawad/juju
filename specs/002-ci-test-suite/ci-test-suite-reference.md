# Juju Integration Test Suite Reference

> Generated for spec authoring reference. Describes the bash-based integration
> test framework under `tests/`.

## Framework Overview

| Component | Path | Purpose |
|-----------|------|---------|
| Runner | `tests/main.sh` | Orchestrates suite execution, flag parsing, cleanup |
| Suites | `tests/suites/<name>/` | Self-contained test groups, each with `task.sh` entry |
| Helpers | `tests/includes/` | Shared utilities (bootstrap, assertions, retry, wait) |
| Env docs | `tests/ENV.md` | Environment variable reference |

### Execution Model

```
main.sh [flags] [suite] [test] [subtest]
  ├─ Parse flags (-p provider, -l local controller, -r reuse, -v verbose)
  ├─ Source includes/*.sh helpers
  ├─ For each suite:
  │   └─ Source suites/<suite>/*.sh → call test_<suite>()
  │       ├─ skip() guard (RUN_LIST / SKIP_LIST)
  │       ├─ Bootstrap controller (or reuse with -r / -l)
  │       ├─ run "run_<subtest>" invocations
  │       └─ Destroy controller
  └─ Cleanup & report
```

### Key Flags

| Flag | Effect |
|------|--------|
| `-A` | Run all suites |
| `-p <provider>` | Set provider (lxd/ec2/gce/azure/k8s/manual/microk8s) |
| `-r` | Reuse existing controller |
| `-l <name>` | Reuse named local controller |
| `-v` / `-V` | Verbose / debug (set -x) |

### Key Environment Variables

| Variable | Purpose |
|----------|---------|
| `BOOTSTRAP_PROVIDER` | Provider type (k8s, lxd, ec2, etc.) |
| `BOOTSTRAP_CLOUD` | Cloud name |
| `BOOTSTRAP_REUSE` | Reuse existing controller |
| `BOOTSTRAP_BASE` | OS base for controller |
| `BOOTSTRAP_ARCH` | Architecture constraint |
| `OPERATOR_IMAGE_ACCOUNT` | Container image registry (K8s) |
| `CONTROLLER_CHARM_PATH_CAAS` | K8s controller charm name |
| `RUN_SUBTEST` | Run a specific subtest only |
| `TEST_INSPECT` | Pause for manual inspection |

### Helper Libraries (`tests/includes/`)

| File | Provides |
|------|----------|
| `run.sh` | `run()` wrapper with timing & error capture |
| `juju.sh` | `bootstrap()`, `ensure()`, controller lifecycle |
| `cleanup.sh` | `add_clean_func()`, teardown registration |
| `controller.sh` | Alternative bootstrap, custom controller setup |
| `check.sh` | Assertion utilities (`check`, `check_contains`) |
| `wait-for.sh` | `wait_for` polling against `juju status --format=json` |
| `storage.sh` | `wait_for_storage`, storage assertions |
| `retry.sh` | Retry logic for flaky operations |
| `network.sh` | Network helpers |
| `secrets.sh` | Secret management helpers |

### Common Test Patterns

```bash
# Provider guard
case "${BOOTSTRAP_PROVIDER:-}" in
  "k8s") run_k8s_test ;;
  *)     echo "==> TEST SKIPPED: K8s only" ;;
esac

# Bootstrap + model
ensure "model-name" "${file}"

# Wait for status
wait_for "app-name" "$(active_idle_condition "app-name")"
wait_for_storage "attached" '.storage["pgdata/0"]["status"].current'

# Cleanup registration
add_clean_func "destroy_model ${model}"
```

---

## All Test Suites (50 total)

### K8s / CAAS Suites

#### `deploy_caas` — K8s Charm Deployment
- **Provider**: K8s only
- **Tests**: `test_deploy_charm`
- **What it verifies**: Deploys charms (discourse-k8s, postgresql-k8s, redis-k8s, nginx-ingress-integrator), validates relations, active/idle status, `juju run` on K8s units.

#### `storage_k8s` — K8s Storage (PV/PVC)
- **Provider**: K8s only
- **Tests**: `test_import_filesystem`, `test_force_import_filesystem`, `test_deploy_attach_storage`, `test_add_unit_attach_storage`, `test_add_unit_duplicate_pvc_exists`, `test_add_unit_attach_storage_scaling_race_condition`
- **What it verifies**: PersistentVolume import, reclaim policy, claimRef handling, storage attachment on deploy, scaling with concurrent PVC attachment, race condition handling.
- **Deployment-type relevance**: HIGH — Deployment/DaemonSet types handle storage differently from StatefulSet. Must verify no regression.

#### `sidecar` — Sidecar Charms (Pebble)
- **Provider**: K8s only
- **Tests**: `test_deploy_and_remove_application`, `test_deploy_and_force_remove_application`, `test_pebble_notices`, `test_pebble_checks`, `test_credential_get_k8s`, `test_rootless`
- **What it verifies**: Sidecar charm lifecycle, Pebble service management, force removal, rootless containers, K8s credential retrieval.

#### `caasadmission` — K8s Admission & Namespaces
- **Provider**: K8s only
- **Tests**: `test_controller_model_admission`, `test_new_model_admission`, `test_model_chicken_and_egg`
- **What it verifies**: Controller model namespace isolation, admission webhooks for new models, bootstrap edge cases.

#### `smoke_k8s` — K8s Smoke Test
- **Provider**: K8s only
- **Tests**: `test_deploy`
- **What it verifies**: Basic charm deployment cycle on K8s.

#### `smoke_k8s_psql` — K8s PostgreSQL Smoke
- **Provider**: K8s only
- **Tests**: `test_deploy_postgresql`
- **What it verifies**: PostgreSQL charm deployment on K8s.

#### `secrets_k8s` — K8s Secret Management
- **Provider**: K8s only
- **Tests**: `test_secrets`, `test_secret_drain`, `test_user_secrets`, `test_user_secret_drain`, `test_add_multiple_secrets_parallel`
- **What it verifies**: Secret creation, lifecycle, user secrets, concurrent operations on K8s.

#### `controllercharm` — Controller Charm (CaaS)
- **Provider**: K8s only
- **Tests**: `test_prometheus`
- **What it verifies**: Controller charm Prometheus metrics endpoint.

#### `coslite` — COS Lite Bundle
- **Provider**: K8s only
- **Tests**: `test_deploy_coslite`
- **What it verifies**: Canonical Observability Stack Lite bundle deployment.

#### `kubeflow` — Kubeflow
- **Provider**: K8s only
- **Tests**: `test_deploy_kubeflow`
- **What it verifies**: Kubeflow ML bundle deployment.

#### `deploy_aks` — Azure Kubernetes Service
- **Provider**: AKS only (currently disabled)
- **Tests**: `test_deploy_aks_charms`
- **What it verifies**: Charm deployment on AKS (skipped pending snap tooling).

#### `ck` — Charmed Kubernetes
- **Provider**: K8s
- **Tests**: `test_deploy_ck`
- **What it verifies**: Full Charmed Kubernetes bundle deployment.

---

### Deployment & Operations Suites

#### `deploy` — Core Deployment
- **Provider**: All
- **Tests**: `test_deploy_charms`, `test_deploy_bundles`, `test_cmr_bundles_export_overlay`, `test_deploy_revision`, `test_deploy_default_series`
- **What it verifies**: Charm and bundle deployment, specific revisions, default base/series, cross-model bundle overlays.

#### `smoke` — Basic Smoke Test
- **Provider**: All
- **Tests**: `test_build`, `test_deploy`
- **What it verifies**: Build verification, basic end-to-end deployment.

#### `machine` — Machine Operations (IaaS)
- **Provider**: IaaS only (NOT K8s)
- **Tests**: `test_logs`
- **What it verifies**: Machine log aggregation.

#### `manual` — Manual Provider
- **Provider**: Manual
- **Tests**: `test_deploy_manual`, `test_spaces_manual`
- **What it verifies**: Manual machine registration and deployment.

#### `unmanaged` — Unmanaged Provider
- **Provider**: Unmanaged
- **Tests**: Provider-specific
- **What it verifies**: Unmanaged machine management (Juju 4+).

---

### Constraints

#### `constraints` — Resource Constraints
- **Provider**: All EXCEPT K8s (explicitly skipped for microk8s)
- **Tests**: `test_constraints_common` (LXD, AWS, GCE, OpenStack, VM variants), `test_constraints_model`
- **What it verifies**: CPU, memory, cores, arch, virt-type constraints at deploy and model level.
- **Note**: K8s constraint tests do not exist in this suite. K8s constraints may need a separate approach.

---

### Storage

#### `storage` — IaaS Storage
- **Provider**: All (provider-specific sub-tests)
- **Tests**: `test_charm_storage`, `test_model_storage_block`, `test_model_storage_filesystem`, `test_persistent_storage`
- **What it verifies**: Block/filesystem/loop/rootfs/tmpfs/ebs storage pools, charm-defined storage, persistent storage lifecycle.

#### `storage_k8s` — See K8s section above.

---

### Resources

#### `resources` — Charm Resources
- **Provider**: All (`test_container_resources` K8s only)
- **Tests**: `test_basic_resources`, `test_upgrade_resources`, `test_empty_resources`, `test_container_resources`
- **What it verifies**: Resource upload/download, upgrade handling, empty resource edge cases, K8s container resources.

---

### Relations & Cross-Model

#### `relations` — Charm Relations
- **Provider**: All
- **Tests**: `test_relation_data_exchange`, `test_relation_departing_unit`, `test_relation_list_app`, `test_relation_model_get`
- **What it verifies**: Relation hook data flow, departing unit cleanup, relation-list for apps, model data retrieval.

#### `cmr` — Cross-Model Relations
- **Provider**: All
- **Tests**: `test_offer_consume`
- **What it verifies**: Offer/consume workflow across models.

---

### Secrets

#### `secrets_iaas` — IaaS Secrets
- **Provider**: IaaS only
- **Tests**: `test_secrets_juju`, `test_secrets_cmr`
- **What it verifies**: Secret creation, rotation, grants; CMR secret sharing.

#### `secrets_k8s` — See K8s section above.

---

### Hooks & Actions

#### `hooks` — Hook Dispatch
- **Provider**: All
- **Tests**: `test_dispatching_script`, `test_start_hook_fires_after_reboot`
- **What it verifies**: Hook dispatch mechanism, hook timing after unit restart.

#### `hooktools` — Hook Tool Access
- **Provider**: All
- **Tests**: `test_state_hook_tools`
- **What it verifies**: Hook tool availability during charm lifecycle.

#### `actions` — Charm Actions
- **Provider**: All
- **Tests**: `test_actions_params`
- **What it verifies**: Action parameter passing and execution via `juju run`.

---

### Model & Controller

#### `model` — Model Lifecycle
- **Provider**: All
- **Tests**: `test_model_config`, `test_model_migration`, `test_model_migration_version`, `test_model_migration_saas_common`, `test_model_migration_saas_external`, `test_model_multi`, `test_model_metrics`, `test_model_destroy`, `test_model_status`
- **What it verifies**: Model configuration (provisioner-harvest-mode etc.), migration between controllers, multi-model management, metrics, status, teardown.

#### `controller` — Controller Operations
- **Provider**: All
- **Tests**: `test_metrics`, `test_enable_ha`, `test_query_tracing`
- **What it verifies**: Controller metrics, HA setup, query tracing.

#### `bootstrap` — Bootstrap
- **Provider**: All
- **Tests**: `test_bootstrap_simplestream`
- **What it verifies**: Bootstrap with custom simplestreams metadata.

#### `agents` — Agent Operations
- **Provider**: All
- **Tests**: `test_charmrevisionupdater`
- **What it verifies**: Charm revision update background agent.

---

### Network & Spaces

#### `network` — Network Health
- **Provider**: All
- **Tests**: `test_network_health`
- **What it verifies**: Network diagnostic and connectivity.

#### `spaces_ec2` — EC2 Spaces
- **Provider**: EC2 only
- **What it verifies**: AWS EC2 network spaces.

#### `spaces_gce` — GCE Spaces
- **Provider**: GCE only
- **What it verifies**: GCE network spaces.

#### `ovs_maas` — OVS on MAAS
- **Provider**: MAAS only
- **Tests**: `test_ovs_netplan_config`
- **What it verifies**: Open vSwitch netplan configuration.

---

### Cloud-Specific

#### `cloud_azure` — Azure Features
- **Provider**: Azure only
- **Tests**: `test_managed_identity`, `test_storage_account_type`
- **What it verifies**: Azure managed identity, storage pool types.

#### `cloud_gce` — GCE Features
- **Provider**: GCE only
- **Tests**: `test_pro_images`, `test_deploy_gpu_instance`, `test_create_storage_pool`
- **What it verifies**: Pro images, GPU instances, GCP storage pools.

#### `firewall` — Firewall Rules
- **Provider**: EC2 only
- **Tests**: `test_firewall_ssh_ec2`
- **What it verifies**: EC2 security group SSH rules.

---

### User & Access

#### `user` — User Management
- **Provider**: All
- **What it verifies**: User creation, credential management.

#### `authorized_keys` — SSH Key Management
- **Provider**: All
- **Tests**: `test_user_ssh_keys`, `test_machine_ssh`, `test_bootstrap_authorized_keys`
- **What it verifies**: SSH public key lifecycle, machine SSH access.

#### `credential` — Cloud Credentials
- **Provider**: All
- **Tests**: `test_add_remove_credential`, `test_controller_credentials`
- **What it verifies**: Credential lifecycle, controller credential binding.

---

### CLI & UI

#### `cli` — CLI Commands
- **Provider**: All
- **Tests**: `test_display_clouds`, `test_local_charms`, `test_model_config`
- **What it verifies**: Cloud display, local charm handling, CLI config.

#### `charmhub` — Charmhub Integration
- **Provider**: All
- **Tests**: `test_charmhub_find`, `test_charmhub_info`, `test_charmhub_download`
- **What it verifies**: Charm discovery, info retrieval, download.

#### `refresh` — Charm Refresh
- **Provider**: All
- **Tests**: `test_basic`, `test_switch`
- **What it verifies**: Charm refresh/upgrade, channel switching.

#### `dashboard` — Dashboard
- **Provider**: All
- **Tests**: `test_dashboard_deploy`
- **What it verifies**: Dashboard charm deployment.

---

### Upgrade & Quality

#### `upgrade` — Version Upgrades
- **Provider**: All
- **What it verifies**: Juju version upgrade process.

#### `static_analysis` — Code Quality
- **Provider**: N/A (code analysis, no provider)
- **What it verifies**: Go linting (golangci-lint), shellcheck, static analysis.

#### `appdata` — Application Data
- **Provider**: All
- **Tests**: `test_appdata_int`
- **What it verifies**: Application data integration.

#### `examples` — Example Charms
- **Provider**: All
- **Tests**: `test_example`, `test_other`
- **What it verifies**: Example charm operations.

---

## Relevance for 001-k8s-deployment-types

### High-Impact Suites (must pass with no regression)

| Suite | Risk | Reason |
|-------|------|--------|
| `deploy_caas` | **Critical** | Core K8s charm deployment — currently StatefulSet-only |
| `storage_k8s` | **Critical** | PV/PVC handling differs between StatefulSet and Deployment |
| `sidecar` | **High** | Sidecar charms depend on pod identity patterns |
| `smoke_k8s` | **High** | Baseline K8s functionality |
| `secrets_k8s` | **Medium** | Secrets may reference pod names tied to StatefulSet |
| `caasadmission` | **Medium** | Namespace/admission may need awareness of resource type |
| `resources` | **Medium** | `test_container_resources` runs on K8s |

### Gaps to Address in New Tests

1. **No `deployment-type` constraint test** — The `constraints` suite skips K8s entirely.
2. **No Deployment/DaemonSet lifecycle test** — All K8s tests assume StatefulSet.
3. **No storage test for non-StatefulSet** — `storage_k8s` tests PVC patterns that don't apply to Deployments.
4. **No scale test for Deployment** — Scaling semantics differ (named pods vs. anonymous replicas).

### Recommended New Test Coverage

```
tests/suites/deploy_caas/
  ├─ deploy_deployment_type.sh    # Deploy with deployment-type=deployment
  ├─ deploy_daemonset_type.sh     # Deploy with deployment-type=daemonset
  └─ deploy_statefulset_default.sh # Verify StatefulSet remains default

tests/suites/storage_k8s/
  └─ deployment_storage.sh        # Verify storage behavior with Deployment type
```
