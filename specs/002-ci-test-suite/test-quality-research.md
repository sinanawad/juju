# Test Quality Research: Juju CI Test Suite

**Date**: 2026-02-17 | **Spec**: [spec.md](spec.md) | **Coverage Audit**: [coverage-analysis.md](coverage-analysis.md)
**Scope**: All 48 integration test suites in `tests/suites/`

## Executive Summary

Five parallel research tracks analyzed every test suite across isolation, flakiness,
assertion quality, charm coupling, and error handling. The findings reveal a test
framework with solid architectural bones but critical implementation gaps that
undermine reliability and coverage confidence.

**Top 3 systemic issues**:
1. **Broken timeout arithmetic** in `wait_for()` — affects all 48 suites, causes CI hangs
2. **Silent assertion failures** — multiple tests use `|| true` on assertions, making them no-ops
3. **Status-only testing** — 30+ suites treat `juju status active/idle` as proof of correctness

**Key metrics across 48 suites**:

| Dimension | Distribution |
|-----------|-------------|
| Isolation | 14 ISOLATED, 8 SELF-CONTAINED, 12 SEQUENTIAL, 3 MIXED (11 not parallelizable) |
| Assertion quality | 4 A-rated, 16 B, 17 C, 11 D (median: 1.3 assertions/scenario) |
| Sterility | 20 sterile (Juju-only), 18 mixed, 10 charm-coupled |
| Charm migration | 20 EASY, 18 MODERATE, 10 HARD |
| Negative testing | Only 10 of 48 suites test any failure scenarios |
| Cleanup verification | 0 of 48 suites verify destroy actually cleaned up resources |

---

## 1. Framework-Level Issues

These affect the entire test suite — every suite inherits these problems.

### 1.1 Broken Timeout Arithmetic (CRITICAL)

**File**: `tests/includes/wait-for.sh`, lines 29 and 352

```bash
# CURRENT (broken) — produces string "1707000050-1707000000", not integer 50
elapsed=$(date -u +%s)-$start_time
if [[ ${elapsed} -ge ${timeout} ]]; then  # string comparison, never triggers

# CORRECT
elapsed=$(( $(date -u +%s) - start_time ))
```

**Impact**: `wait_for()` and `wait_for_storage()` timeouts never fire. If a condition
is never met, the test hangs until CI kills the job externally. This is the single
highest-impact bug in the test framework.

### 1.2 Infinite Polling Loops (CRITICAL)

Five `wait_for_*` functions have **no timeout at all**:

| Function | File:Line | Used By |
|----------|-----------|---------|
| `wait_for_machine_agent_status()` | wait-for.sh:155 | bootstrap, deploy, network |
| `wait_for_container_agent_status()` | wait-for.sh:186 | deploy (LXD containers) |
| `wait_for_machine_netif_count()` | wait-for.sh:220 | spaces/network tests |
| `wait_for_subordinate_count()` | wait-for.sh:246 | hooks/reboot |
| `wait_for_model()` | wait-for.sh:279 | migration tests |

Additionally, `controller/enable_ha.sh:1-23` has tight infinite loops with no sleep
or timeout for `wait_for_controller_no_leader` and `wait_for_controller_leader`.

### 1.3 Destruction Errors Suppressed (HIGH)

Both `destroy_model` (juju.sh:507) and `destroy_controller` (juju.sh:576) use
`|| true`, swallowing all errors. The error-check code has its `exit 1` commented
out with a "workaround" note. This means resource leaks in CI go undetected — the
test suite reports success even when cleanup fails.

### 1.4 SSH Key Setup Returns Success on Failure (HIGH)

`add_client_ssh_key_to_juju_model` (juju.sh:453-477) uses `return` without an error
code on all failure paths, defaulting to `return 0` (success). If the SSH key file
is missing or `juju add-ssh-key` fails, the function reports success. Tests that
depend on SSH access fail later with confusing errors.

---

## 2. Test Isolation

### 2.1 Classification

| Classification | Count | Description |
|---------------|-------|-------------|
| **ISOLATED** | 14 | Each test function creates its own model via `ensure()` and calls `destroy_model` |
| **SELF-CONTAINED** | 8 | Each test bootstraps its own controller — strongest isolation |
| **SEQUENTIAL** | 12 | Tests share a single bootstrap model, must run in order |
| **MIXED** | 3 | Some tests isolated, some share state |

**Well-isolated suites (model-per-test pattern)**:
deploy, model, secrets_iaas, caasadmission, cli, sidecar, storage_k8s

**Sequential (shared state) suites — cannot be reordered or parallelized**:
authorized_keys, firewall, hooks, refresh, relations, resources, secrets_k8s,
smoke, spaces_ec2, static_analysis, storage, user

### 2.2 Optimization Opportunities

Some sequential coupling exists for good reason — bootstrapping a controller takes
minutes and should be amortized across tests. The goal is not to eliminate shared
controllers but to ensure **test independence within a shared controller**:

- **Model-per-test**: Each test creates a fresh model on the shared controller,
  deploys, asserts, and destroys the model. The controller is shared; the model
  state is isolated. This is the pattern used by `deploy`, `model`, `secrets_iaas`.
- **Substrate verification as isolation proof**: After `destroy_model`, verify on
  the substrate (K8s namespace gone, LXD containers removed) before the next test
  starts. This proves the controller is clean for the next test without requiring
  a fresh bootstrap.
- **Static analysis parallelism**: The `static_analysis` suite has 8 independent
  linting checks that run sequentially but could safely run in parallel — no
  controller or model involved.

### 2.3 High-Risk Isolation Gaps

| Suite | Risk | Issue |
|-------|------|-------|
| `controller` | HIGH | `test_enable_ha` permanently adds HA nodes; affects `test_query_tracing` |
| `authorized_keys` | HIGH | SSH keys and machines persist across tests in shared model |
| `user` | HIGH | User accounts persist across tests on shared controller |
| `relations` | HIGH | 4 tests deploy interconnected charms in shared model |
| `secrets_k8s` | HIGH | 5 tests manipulate secrets on shared model/controller |
| `cli` | MEDIUM | `run_controller_clouds` adds "my-ec2" cloud to controller, never removes it |

---

## 3. Flakiness Patterns

### 3.1 Hardcoded Sleeps (20+ instances)

Fixed `sleep` calls replace proper condition-based waiting. These are either too
short (flaky on slow CI) or too long (wasting CI time).

**Critical (30s+)**:

| File | Sleep | Purpose |
|------|-------|---------|
| `secrets_iaas/vault.sh:218` | 60s | Wait for vault initialization |
| `model/metrics.sh:116` | 120s | Wait for disabled telemetry check |
| `model/metrics.sh:32,74` | 45s | Wait for charmrevisioner restart |
| `model/migration.sh:206,273,341` | 30s | Wait for relation join before migrate |

**Recommended fix pattern**: Replace `sleep N` with polling loops that check the
actual condition, using the existing `wait_for` pattern (after fixing its timeout bug).

### 3.2 Silent Assertion Failures (5+ instances)

Tests that use `grep -q "pattern" || true` on assertions — the `|| true` makes
the assertion always pass regardless of whether the pattern was found:

| File | Line | Assertion That Can Never Fail |
|------|------|-------------------------------|
| `hooks/dispatch.sh` | 17 | `juju debug-log \| grep -q "via hook dispatching script" \|\| true` |
| `hooks/dispatch.sh` | 25 | `juju debug-log \| grep -q 'ran "update-status" hook' \|\| true` |
| `cli/block.sh` | 11,36-39,67-71 | All block command assertions use `\|\| true` |
| `authorized_keys/machine.sh` | 19 | SSH key distribution assertion |
| `deploy/deploy_charms.sh` | 63 | Unsupported series rejection assertion |

**Fix**: Use `check_contains` from `check.sh` instead of `grep -q ... || true`.

### 3.3 Non-Deterministic Log Assertions

Tests that `grep` streaming `juju debug-log` output without `--replay --no-tail`:
- `hooks/dispatch.sh:17,25` — `debug-log` streams forever, `grep -q` + `|| true` means result is never checked
- `model/metrics.sh:36` — missing `--replay --no-tail`, will hang

### 3.4 Tight/Inadequate Polling

| File | Retries | Total Wait | Concern |
|------|---------|-----------|---------|
| `model/multi.sh:56` | 2 | ~10s | Only 2 attempts for config change |
| `firewall/ssh_allow.sh:28` | 5 | ~5s | AWS security group propagation |
| `firewall/ssh_allow.sh:68` | 10 | ~10s | GCE firewall propagation |
| `wait-for.sh:318` | 3 | ~15s | Systemd unit file appearance |
| `sidecar/sidecar.sh:67` | 3 | ~15s | HTTP endpoint readiness |

Note: `firewall/ssh_allow.sh:29-30,69-70` prints errors but does NOT exit —
the test continues and passes even when the firewall assertion fails.

### 3.5 External Dependencies (Accepted)

The following external dependencies are **acceptable** per project constraints:
- **Charmhub**: Required for fetching charms. Future calibration charms (norma-k8s,
  norma) will also be fetched from charmhub. No change needed.
- **Snap store**: Required for environment preparation (snap info, snap install).
  Cannot be self-contained. Accepted as-is.

These should be documented in the spec as known external dependencies rather than
treated as flakiness risks.

---

## 4. Assertion Quality and Coverage Depth

### 4.1 Quality Ratings

| Rating | Count | Suites | Criteria |
|--------|-------|--------|----------|
| **A** | 4 | secrets_iaas, secrets_k8s, appdata, hooks | Thorough multi-layer assertions, verify actual data/behavior |
| **B** | 16 | actions, authorized_keys, charmhub, cli, cmr, controllercharm, deploy, firewall, hooktools, model, network, refresh, relations, sidecar, static_analysis, storage, storage_k8s, user | Good assertions covering main success path |
| **C** | 17 | caasadmission, ck, cloud_gce, constraints, controller, coslite, credential, dashboard, deploy_caas, machine, manual, resources, smoke, smoke_k8s_psql, spaces_ec2, unmanaged, upgrade | Minimal assertions, mostly status checks |
| **D** | 11 | agents, bootstrap, cloud_azure, deploy_aks, examples, kubeflow, ovs_maas, smoke_k8s, spaces_gce | Few or no meaningful assertions |

### 4.2 Assertion Density

**Best**: secrets_k8s (7.9/scenario), secrets_iaas (6.6), cmr (4.5), appdata (4.0)
**Median**: ~1.3 assertions per test scenario
**Worst**: examples (0.0), cloud_azure (0.2), agents (0.3), bootstrap (0.3)

### 4.3 Systemic Coverage Gaps

| Gap | Affected | Description |
|-----|----------|-------------|
| Status-only testing | 30+ suites | Treat `juju status active/idle` as proof of correctness |
| No cleanup verification | ALL 48 | Call `destroy_model` but never verify resources gone |
| No negative testing | 38 suites | Zero error-path or failure scenarios |
| No idempotency testing | ALL 48 | No repeated-operation tests |
| No concurrent operation testing | ALL 48 | No parallel-operation tests |

### 4.4 Gold Standard: secrets_iaas / secrets_k8s

These suites demonstrate best-practice assertion patterns:
- Verify actual secret content values (not just "secret exists")
- Test lifecycle transitions (create → grant → get → rotate → expire → remove)
- Test access control (denied access produces expected error)
- Test cross-model behavior (CMR secrets)
- Test backend-specific behavior (K8s secret objects in namespace)

All other suites should aspire to this depth of verification.

---

## 5. Charm Coupling and Sterility

### 5.1 Principle

Tests should verify **Juju behavior**, not charm behavior. Charms are deployment
vehicles — the test asserts that Juju correctly deployed, configured, related,
scaled, or removed the application. The charm is an implementation detail.

Future state: all smoke and regression tier suites use only calibration charms
(norma-k8s for K8s, norma for IaaS). Third-party charms are used only in the
integration tier for full-stack validation.

### 5.2 Sterility Distribution

| Rating | Count | Description |
|--------|-------|-------------|
| **2 (sterile)** | 20 | Tests only Juju behavior; charms are simple deploy targets |
| **1 (mixed)** | 18 | Some charm-specific assertions, but mostly Juju testing |
| **0 (charm-coupled)** | 10 | Tests charm behavior directly |

### 5.3 Version Pinning Gaps

Many charmhub deploys are **unpinned** — channel updates can silently break tests:

- `juju deploy ubuntu-lite` (no revision) — 15+ suites
- `juju deploy snappass-test` — sidecar, smoke_k8s
- `juju deploy discourse-k8s` — deploy_caas
- `juju deploy postgresql-k8s` — multiple suites (channel 14/stable pinned)
- `juju deploy cos-lite` — coslite (channel stable)
- `juju deploy charmed-kubernetes` — ck (no revision)

**Recommendation**: Pin revisions for all third-party charms we don't own. For
juju-qa-* charms (owned by the team), channel pinning is sufficient. For future
calibration charms (norma-k8s, norma), channel pinning with explicit channel in
predicates.yaml.

### 5.4 Migration Difficulty

| Difficulty | Count | Suites |
|-----------|-------|--------|
| **EASY** | 20 | No charms or charms used as simple deploy targets |
| **MODERATE** | 18 | Need specific actions, relations, config, or resources |
| **HARD** | 10 | Deeply coupled to charm internals |

See Section 7 for detailed migration analysis of MODERATE and HARD suites.

---

## 6. Error Handling

### 6.1 Summary by Severity

| Severity | Count | Key Areas |
|----------|-------|-----------|
| **CRITICAL** | 6 | Broken timeout arithmetic (2), missing timeouts (4) |
| **HIGH** | 8 | Silent assertions (4), destruction errors suppressed (2), exit code gaps (2) |
| **MEDIUM** | 3 | Expected-failure tests not validating failure, fixed sleeps |

### 6.2 Key Patterns

**Pattern: Error swallowing on expected failures**

Tests that check for expected errors use `command | grep -q "error message" || true`.
If the command succeeds unexpectedly (Juju bug), the test silently passes. Found in:
cli/block.sh, deploy/deploy_charms.sh, model/config.sh.

**Pattern: Subshell exit code loss**

Some `test_*` functions run in subshells where `set -e` may be toggled by
`set_verbosity`. A failure inside a `run()` call within the subshell may not
propagate to the parent.

**Pattern: Cleanup-on-failure gaps**

SEQUENTIAL suites rely entirely on `destroy_controller` at suite end. If a test
fails mid-way, intermediate state (deployed apps, relations, storage) is not
cleaned up before the next test starts — the suite aborts entirely via `set -e`.
There is no try/finally pattern for per-test cleanup in shared-state suites.

---

## 7. Calibration Charm Migration Analysis

This section details what each non-EASY suite requires from charms and whether
the calibration charms (norma-k8s, norma) can satisfy those requirements. CC-*
IDs reference `specs/002-ci-test-suite/contracts/charm-contract.yaml`.

### 7.1 HARD Migration Suites (10)

| Suite | Juju Capabilities Tested | Can Calibration Replace? | Required CC-* | Approach |
|-------|--------------------------|--------------------------|---------------|----------|
| **appdata** | Config propagation, relation data exchange, file I/O via SSH, scaling, subordinates | PARTIAL | CC-02, CC-06, CC-08 | Need file I/O capability added to contract. Subordinate support via CC-M5. |
| **ck** | Cloud provider integration, CK bundle, add-k8s | NO | — | Infrastructure test. Keep as integration tier. Not a calibration target. |
| **controllercharm** | Controller metrics-endpoint, Prometheus scrape targets, cross-controller relations | PARTIAL | CC-06, CC-08, CC-K9 | Need CC-K9 (COS observability) implemented. Prometheus-specific interface required. |
| **coslite** | COS Lite bundle deployment, HTTP health endpoints | NO | — | Monitoring stack test. Keep as integration tier. Not a calibration target. |
| **deploy_caas** | K8s multi-charm deploy, relations, actions, trust | YES | CC-04, CC-06, CC-K1 | Replace discourse/postgresql/redis/nginx with 2 norma-k8s instances (provider + requirer). |
| **kubeflow** | Large bundle deploy, scale, LoadBalancer | NO | — | ML platform test. Keep as integration tier. Not a calibration target. |
| **network** | Subordinate relations, HTTP connectivity checks, IP change detection, multi-base | PARTIAL | CC-06, CC-M5, CC-K8, CC-M6 | Need CC-M5 (subordinate) + HTTP server for connectivity checks. |
| **sidecar** | Pebble lifecycle, notices, health checks, rootless, multi-container, credential-get | YES | CC-K1, CC-K2, CC-K4, CC-K5, CC-K7 | norma-k8s with full K8s contract (CC-K1 through CC-K7). |
| **smoke_k8s_psql** | K8s relations, actions, data exchange between provider/requirer | PARTIAL | CC-04, CC-06, CC-03 | Use two norma-k8s instances with provider/requirer relation. No real DB needed. |
| **relations** | All relation hook tools, CMR, peer relations, leader permissions, departing lifecycle | YES | CC-05, CC-06, CC-07, CC-04 | norma with full relation endpoint coverage (peer + provides + requires + CMR). |

**Key findings — 3 suites are NOT calibration targets**:
- `ck`, `coslite`, `kubeflow` are full-stack integration tests that validate
  third-party ecosystem bundles, not Juju behavior. They should remain in the
  integration tier with their current charms. They are NOT candidates for
  calibration charm migration.

**Key findings — 4 suites need contract expansion**:
- `appdata` needs file I/O capability (write to disk, verify via SSH)
- `controllercharm` needs CC-K9 (COS observability) — Prometheus metrics endpoint
- `network` needs CC-M5 (subordinate) + HTTP server
- `smoke_k8s_psql` needs two-charm provider/requirer pattern (no real DB)

**Key findings — 3 suites are fully migratable**:
- `deploy_caas` → 2 norma-k8s with relations
- `sidecar` → norma-k8s with CC-K1 through CC-K7
- `relations` → norma with CC-05, CC-06, CC-07

### 7.2 MODERATE Migration Suites (18)

| Suite | Key Charm Capability Needed | CC-* Contracts | Migration Path |
|-------|-----------------------------|----------------|----------------|
| **actions** | Parameterized actions, error reporting | CC-04, CC-12 | norma with `run-check` and `fail-action` actions |
| **agents** | Charm with multiple revisions for revision updater testing | CC-11 | norma published at multiple revisions on charmhub |
| **charmhub** | Charm with resources for download testing | CC-11 | norma with foo-file resource definition |
| **cloud_azure** | Simple deploy target only | CC-03 | Direct swap to norma. No special capabilities. |
| **cloud_gce** | Simple deploy target only | CC-03 | Direct swap to norma. No special capabilities. |
| **cmr** | Two charms with provider/requirer relation, config-driven data exchange | CC-02, CC-06, CC-07 | Two norma instances with `calibration-provider`/`calibration-requirer` endpoints |
| **dashboard** | Simple deploy target only | CC-03 | Keep juju-dashboard (tests dashboard-specific behavior) |
| **deploy** | Multi-base support, LXD profiles, error states, resource files | CC-03, CC-11 | norma with multi-base builds. Local testcharms (lxd-profile, simple-resolve) likely stay. |
| **deploy_aks** | K8s deploy target (currently skipped) | CC-K1 | norma-k8s when AKS support is ready |
| **hooks** | Event lifecycle logging, subordinate endpoint, reboot detection | CC-01, CC-M5 | norma with `get-event-log` action + subordinate variant (norma-sub) |
| **model** | Relations for migration tests, subordinates for metrics | CC-02, CC-06 | norma pairs (source/sink) for migration. Some sub-tests need no charms. |
| **ovs_maas** | Space bindings in metadata | CC-M6 | norma with space-aware endpoint bindings |
| **refresh** | Multiple revisions, multiple channels, resources | CC-11, CC-02 | norma published across channels (stable, edge) with resource definitions |
| **resources** | File resources (small + 100MB), OCI resources (K8s) | CC-10, CC-K6 | norma with foo-file resource. norma-k8s with OCI image resource. |
| **secrets_iaas** | Full secret lifecycle, CMR secrets, backend config | CC-09, CC-06, CC-07 | norma with secret create/grant/revoke/rotate/expire via actions |
| **secrets_k8s** | K8s secret backend, scale-aware secrets, RBAC | CC-09, CC-K1, CC-06 | norma-k8s with secret lifecycle. Kubectl verification of K8s Secret objects. |
| **spaces_gce** | No charm capabilities needed (infrastructure test) | CC-M6 | No migration needed. Uses `add-machine` only. |
| **storage_k8s** | K8s storage (PVC), import, attach, multi-unit | CC-K10, CC-10 | norma-k8s with `pgdata`-equivalent storage definition |

**Key findings — infrastructure-only tests (no charm migration needed)**:
cloud_azure, cloud_gce, spaces_gce — these test Juju's cloud/provider integration,
not charms. They use ubuntu/ubuntu-lite as simple deploy targets. Swap to norma is
trivial but low priority.

**Key findings — charmhub/revision-dependent tests**:
agents, charmhub, refresh — these require norma to be published on charmhub with
multiple revisions and channels. This is a charmhub publishing workflow requirement,
not a charm code requirement.

**Key findings — dashboard stays as-is**:
The dashboard suite tests juju-dashboard-specific behavior. It should keep
juju-dashboard since that's the system under test.

### 7.3 Minimum Viable Contract for Migration

Based on the analysis, the calibration charms need these capabilities to unblock
migration of all EASY + MODERATE suites plus the migratable HARD suites:

**norma (IaaS) minimum viable contract**:
- CC-01: Event lifecycle logging (`get-event-log`)
- CC-02: Configuration (string, int, float, bool, secret)
- CC-03: Status reporting (`set-status`)
- CC-04: Actions with params (`run-check`)
- CC-05: Peer relations (`norma-peers`)
- CC-06: Provides/requires relations (`calibration-provider`/`calibration-requirer`)
- CC-07: Cross-model relations (same endpoints via offer/consume)
- CC-09: Secrets (full lifecycle)
- CC-10: Storage (filesystem)
- CC-11: Upgrade/refresh (`get-version`)
- CC-12: Action error handling (`fail-action`)
- CC-M5: Subordinate interface (norma-sub variant)
- CC-M6: Spaces & network bindings

**norma-k8s (K8s) minimum viable contract**:
- All shared contracts (CC-01 through CC-12)
- CC-K1: Pebble workload management
- CC-K2: Pebble health checks
- CC-K4: Pebble custom notices
- CC-K5: Multiple containers
- CC-K6: OCI resources
- CC-K7: Non-root execution
- CC-K10: Multiple storages

**Contracts NOT needed for migration** (integration tier only):
- CC-K3: Pebble file/exec ops (nice-to-have, not blocking)
- CC-K8: Port management (network suite needs HTTP server, covered by CC-K1)
- CC-K9: COS observability (controllercharm only — defer to integration tier)
- CC-K11: Event deferral (sidecar has test-defer but not critical path)
- CC-M1 through CC-M4: Machine-specific (covered by IaaS norma naturally)
- CC-M7 through CC-M11: Machine-specific advanced (defer)

### 7.4 Suites That Cannot Be Migrated

These 4 suites test third-party ecosystem integration, not Juju behavior.
They should remain in the integration tier with their current charms:

| Suite | Why Not Migratable | Recommendation |
|-------|-------------------|----------------|
| `ck` | Tests Charmed Kubernetes bundle (50+ components) | Integration tier, as-is |
| `coslite` | Tests COS Lite monitoring stack | Integration tier, as-is |
| `kubeflow` | Tests Kubeflow ML platform | Integration tier, as-is |
| `dashboard` | Tests juju-dashboard charm specifically | Keep juju-dashboard charm |

---

## 8. Recommendations

### 8.1 Immediate Fixes (No Spec Changes)

These can be merged today as standalone improvements:

1. **Fix `wait_for` timeout arithmetic** — Change `elapsed=$(date -u +%s)-$start_time`
   to `elapsed=$(( $(date -u +%s) - start_time ))` in wait-for.sh lines 29 and 352.
   Single highest-impact fix.

2. **Add timeouts to infinite loops** — Add `if [[ ${attempt} -gt MAX ]]; then exit 1; fi`
   to all 5 `wait_for_*` functions and the HA leader wait loops.

3. **Fix silent assertions** — Replace `grep -q ... || true` with `check_contains`
   in hooks/dispatch.sh, cli/block.sh, authorized_keys/machine.sh.

4. **Fix SSH key setup error handling** — Change `return` to `return 1` on error
   paths in `add_client_ssh_key_to_juju_model`.

### 8.2 Spec Amendments

Based on research findings, the following should be incorporated into spec.md:

1. **External dependencies are accepted**: Charmhub and snap store dependencies are
   inherent to Juju testing. Document as known dependencies, not flakiness risks.

2. **Isolation via model-per-test**: The recommended pattern is shared controller +
   per-test models + substrate verification of cleanup. This optimizes bootstrap
   time while maintaining test independence.

3. **Version pinning policy**: Third-party charms must be pinned to specific
   revisions. Team-owned charms (juju-qa-*, norma-*) use channel pinning.

4. **Calibration charm contract gaps**: The HARD migration analysis (Section 7.1)
   identifies specific CC-* contract capabilities that the calibration charms must
   implement to enable migration.

### 8.3 Priority Order for Suite Improvements

1. Framework fixes (Section 8.1) — affects all 48 suites
2. Convert SEQUENTIAL suites to model-per-test where feasible
3. Add substrate verification to smoke and regression tier suites
4. Add negative test scenarios to D-rated and C-rated suites
5. Migrate EASY suites to calibration charms (20 suites, minimal effort)
6. Migrate MODERATE suites (18 suites, requires calibration charm capabilities)
7. Migrate HARD suites (10 suites, requires contract expansion or tier reclassification)
