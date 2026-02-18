# CI Test Suite: Design Proposal

**Index**: 002 | **Status**: Draft | **Author**: Sinan / Claude
**Companion artifacts**: [spec.md](spec.md) (requirements), [data-model.md](data-model.md) (schema), [test-quality-research.md](test-quality-research.md) (findings), [coverage-analysis.md](coverage-analysis.md) (audit)

---

## Abstract

This proposal introduces a predicate-driven CI test system for Juju: per-suite metadata that declares when, where, and why each test runs, combined with quality standards that ensure every test validates Juju behavior at the substrate level. The same framework classifies all existing suites, enables smart test selection on PRs, and defines the contract for purpose-built calibration charms that isolate Juju testing from third-party charm failures.

## Rationale

Juju CI today runs 6 of 50 integration test suites on PRs. The remaining 44 suites exist but are not automated — they run manually or not at all. When tests do run, they:

- **Check status, not substance** — 30+ suites treat `juju status active/idle` as proof that an operation succeeded, without verifying the substrate (K8s pods exist, LXD containers removed, PVCs created).
- **Hang instead of timing out** — A bug in `wait_for()` means timeouts never fire. Tests hang until CI kills the job externally.
- **Pass silently on assertion failure** — Multiple test files use `|| true` on assertions, making them no-ops.
- **Conflate Juju bugs with charm bugs** — Tests using postgresql-k8s or discourse-k8s can't distinguish a Juju regression from an upstream charm update.
- **Waste CI time after setup fails** — If bootstrap fails, all downstream tests in the suite still attempt to run and fail individually.

The result: low coverage, low confidence, and no clear path to improving either. A developer changing `domain/storage/` code gets no feedback on whether storage tests pass until someone runs them manually.

## Governing Principle

**Iterative improvement, not revolution.** Every change ships standalone value. The existing test framework is preserved — predicates are metadata about suites, not modifications to suites. A developer can still run `./main.sh deploy` locally with no awareness of predicates. The CI never goes dark: old and new coexist until the new is proven.

---

## Design

### 1. Predicate Metadata

Each test suite gets a `predicates.yaml` file co-located with its `task.sh`. This file declares:

```yaml
schema: "1.0"
tier: regression              # when: sanity | smoke | regression | integration
provider: k8s                 # where: all | k8s | iaas | ec2 | gce | azure | maas
level: MUST                   # criticality: MUST | SHOULD | MAY
paths:                        # why: which source changes trigger this suite
  - domain/storage/**
  - internal/worker/caas*/**
charms:                       # sterility: what charms does this suite use
  - name: norma-k8s
    type: calibration
rationale: "K8s storage lifecycle including PVC management"
tests:                        # fail-fast: intra-suite dependency DAG
  - name: test_setup
    type: prerequisite
  - name: test_storage_attach
    type: test
    depends_on: [test_setup]
```

**No workflow changes required.** Adding a `predicates.yaml` to a suite automatically includes it in predicate-based activation. Suites without predicates default to `tier: integration, provider: all, paths: ["*"]` — they run only on nightly/manual builds until explicitly classified.

### 2. Test Tiers

| Tier | Intent | Target Time | When It Runs |
|------|--------|-------------|-------------|
| **Sanity** | Compilation, linting, schema validation | <5 min | Always |
| **Smoke** | Minimal e2e per provider: bootstrap, deploy, verify | <15 min | PR (blocking), push, nightly |
| **Regression** | Feature-area verification: storage, relations, secrets, hooks | <60 min | PR (informational), push (blocking), nightly |
| **Integration** | Cross-feature, upgrade paths, ecosystem bundles | <4 hours | Nightly, manual, release builds |

Event-to-tier mapping:

| Event | Eligible Tiers | Blocking Behavior |
|-------|---------------|-------------------|
| `pull_request` | Sanity + Smoke + Regression | Sanity + Smoke block merge; Regression is informational |
| `push` to main | Sanity + Smoke + Regression | All blocking |
| `schedule` (nightly) | All tiers | All blocking |
| `workflow_dispatch` | Configurable (default: all) | All blocking |

### 3. Requirement Levels

Each suite declares a `level` that indicates its criticality:

| Level | Meaning | Failure Handling |
|-------|---------|-----------------|
| **MUST** | Core functionality. Failure blocks the pipeline for the event type. | Always blocking when the tier is active |
| **SHOULD** | Expected to pass. Failure requires documented justification to proceed. | Blocking, but waivable with sign-off |
| **MAY** | Supplementary coverage. Informational only. | Never blocking |

This applies uniformly — there is no reduced strictness for different release types. A release build runs all tiers; quality expectations are the same whether shipping a patch or a major release. If a test matters, it's MUST. If it's nice to have, it's SHOULD. If it's exploratory, it's MAY.

### 4. Path-Based Activation

The predicate evaluator matches changed files against suite path globs. This enables targeted testing:

| Changed Path | Activated Suites |
|-------------|-----------------|
| `domain/secret/` | secrets_iaas, secrets_k8s |
| `domain/storage/` | storage, storage_k8s |
| `domain/relation/` | relations, cmr |
| `domain/application/` | deploy, deploy_caas, refresh, resources |
| `caas/`, `internal/worker/caas*/` | All K8s suites |
| `cmd/juju/` | cli, charmhub |
| `core/`, `go.mod` | **All suites** (wildcard — foundational code) |
| `tests/includes/` | **All suites** (wildcard — framework changes) |

Path predicates use curated glob patterns, not automated dependency analysis. This is a deliberate choice — curated globs are transparent, auditable, and don't require build tooling. Automated Go Transitive Analysis (GTA) is a documented future enhancement that the schema is forward-compatible with.

### 5. Fail-Fast Test Dependencies

Each suite can declare a DAG of test dependencies. If a prerequisite fails, all dependent tests are skipped immediately:

```
test_setup (prerequisite)
    ├── test_storage_attach
    │       └── test_storage_detach
    └── test_storage_persist  (independent of attach)
```

If `test_setup` fails → all 3 tests skipped in seconds, not after 3 separate timeout failures. The DAG is declared in `predicates.yaml` and executed by an enhanced `run.sh`.

### 6. Substrate Verification

Tests must verify outcomes at the infrastructure level, not just Juju CLI responses:

| Juju Operation | What Tests Do Today | What Tests Must Do |
|---------------|--------------------|--------------------|
| `juju deploy app` | Check `juju status` shows active | **Also** verify pod/container exists on substrate |
| `juju destroy-model` | Check command exits 0 | **Also** verify K8s namespace / LXD containers gone |
| `juju scale-application -n 3` | Check status shows 3 units | **Also** verify 3 pods / 3 machines on substrate |
| `juju relate a b` | Check status shows relation | **Also** verify relation data visible to charm via action |

Substrate verification uses provider-aware helpers in `tests/includes/substrate.sh` that dispatch to `kubectl` (K8s) or `lxc` (LXD) based on the current provider.

### 7. Test Sterility (Calibration Charms)

Tests must validate **Juju behavior**, not charm behavior. A test that passes because postgresql-k8s happened to work tells us nothing about a Juju regression.

**Current state** (48 suites):

| Sterility | Count | Description |
|-----------|-------|-------------|
| Sterile | 20 | Only tests Juju behavior; charms are simple deploy targets |
| Mixed | 18 | Some charm-specific assertions, but mostly Juju testing |
| Charm-coupled | 10 | Tests charm behavior directly |

**Target state**: All smoke and regression tier suites use only calibration charms (**norma-k8s** for K8s, **norma** for IaaS). Third-party charms are used only in the integration tier for ecosystem validation.

**Version pinning policy**: Third-party charms that remain (integration tier) must be pinned to specific revisions. Team-owned charms use channel pinning.

### 8. Calibration Charm Contract

Two calibration charms serve as sterile test targets. Their development is **out of scope** — this design defines the contract they must fulfill.

**Shared capabilities** (both charms):

| ID | Capability | Action/Interface |
|----|-----------|-----------------|
| CC-01 | Event lifecycle logging | `get-event-log` → ordered event ledger |
| CC-02 | Configuration (all types) | `get-config` + string/int/float/bool/secret options |
| CC-03 | Status reporting | `set-status` → active/blocked/waiting/maintenance |
| CC-04 | Actions with params | `run-check <capability>` → structured JSON pass/fail |
| CC-05 | Peer relations | `norma-peers` endpoint + `get-peer-data` |
| CC-06 | Provides/requires relations | `calibration-provider` + `calibration-requirer` |
| CC-07 | Cross-model relations | Same endpoints via `juju offer`/`juju consume` |
| CC-08 | Scaling | `get-cluster-info` → unit count, leader, peers |
| CC-09 | Secrets | Full lifecycle: create, rotate, expire, share, remove |
| CC-10 | Storage (filesystem) | `check-storage` → persistence marker verification |
| CC-11 | Upgrade/refresh | `get-version` → charm_version, workload_version |
| CC-12 | Action error handling | `fail-action` → intentional failure with message |

**K8s-specific** (norma-k8s): Pebble lifecycle (CC-K1), health checks (CC-K2), file/exec ops (CC-K3), custom notices (CC-K4), multi-container (CC-K5), OCI resources (CC-K6), non-root (CC-K7), ports (CC-K8), COS (CC-K9), multi-storage (CC-K10), event deferral (CC-K11).

**Machine-specific** (norma): Systemd (CC-M1), SSH (CC-M2), constraints (CC-M3), LXD containers (CC-M4), subordinate (CC-M5), spaces (CC-M6), block storage (CC-M7), base verification (CC-M8), agent tools (CC-M9), payloads (CC-M10), subordinate variant (CC-M11).

**Design principles**: Same action names across charms (test script reuse); structured JSON results; self-diagnosing (charm bugs distinguishable from Juju bugs); no external runtime dependencies; active/idle within 120 seconds.

### 9. Predicate Evaluator

A bash script (`tests/evaluate-predicates.sh`) that reads all `predicates.yaml` files, evaluates them against the event context (event type, changed files, target provider), and outputs a JSON matrix consumed by GitHub Actions:

```json
{
  "include": [
    {"suite": "smoke_k8s", "provider": "microk8s", "tier": "smoke", "required": true},
    {"suite": "storage_k8s", "provider": "microk8s", "tier": "regression", "required": false}
  ]
}
```

The evaluator is a pure function: inputs in, JSON out. No state, no side effects. It runs as a GitHub Actions job step, and its output feeds `strategy.matrix` for parallel test dispatch.

### 10. GitHub Actions Architecture

**Keep unchanged**: `static-analysis.yml`, `cla.yml`, `merge.yml`, `context-tests.yml`

**New**: A single `integration-tests.yml` that:
1. Runs `evaluate-predicates.sh` to determine which suites to activate
2. Spawns parallel jobs per (suite, provider) pair via matrix strategy
3. Each job runs `./main.sh -p <provider> --fail-fast <suite>`
4. Regression-tier jobs on PRs use `continue-on-error: true` (informational)

**Retire** (after validation): `smoke.yml`, `postgresql-k8s.yml`, `upgrade.yml`, `microk8s-tests.yml`

**Transition**: Both old and new workflows run in parallel for 2 weeks. Once parity is confirmed, old workflows are removed.

### 11. Resource Cleanup

CI-created resources use a `ci-<GITHUB_RUN_ID>-<suite>` naming convention. A scheduled sweeper (`ci-sweeper.yml`, hourly) discovers and destroys resources older than a configurable TTL (default: 4 hours). The sweeper is idempotent — running it when no stale resources exist produces no errors.

---

## Current State Assessment

### Coverage Audit (completed)

175 Juju capabilities mapped across 7 domains:

| Domain | Capabilities | GOOD | PARTIAL | MINIMAL | NONE |
|--------|-------------|------|---------|---------|------|
| Deployment & App Lifecycle | 30 | 80% | 13% | 7% | 0% |
| Relations & Integrations | 15 | 60% | 33% | 0% | 7% |
| Model & Controller Mgmt | 22 | 41% | 45% | 14% | 0% |
| Cloud, Credentials & Network | 25 | 56% | 20% | 24% | 0% |
| Secrets, Storage & Resources | 30 | 90% | 10% | 0% | 0% |
| K8s/CAAS Specific | 18 | 44% | 33% | 11% | 11% |
| Constraints, Machines & Agents | 35 | 63% | 23% | 14% | 0% |
| **Total** | **175** | **65%** | **23%** | **10%** | **2%** |

Only 12% of features are covered by automated CI (6 of 50 suites on PRs). The predicate system unlocks the other 44 suites.

### Quality Audit (completed)

| Dimension | Finding |
|-----------|---------|
| **Framework bugs** | Broken timeout arithmetic (all 48 suites), 5 infinite loops, suppressed destruction errors |
| **Assertion quality** | 4 A-rated suites, 16 B, 17 C, 11 D. Median: 1.3 assertions/scenario |
| **Isolation** | 14 isolated, 8 self-contained, 12 sequential, 3 mixed |
| **Negative testing** | Only 10 of 48 suites test any failure scenario |
| **Cleanup verification** | 0 of 48 suites verify destroy actually cleaned up resources |
| **Sterility** | 20 sterile, 18 mixed, 10 charm-coupled |
| **Charm migration** | 20 EASY, 18 MODERATE, 10 HARD (4 non-migratable: ck, coslite, kubeflow, dashboard) |

### Gold Standard: secrets_iaas / secrets_k8s

These suites (authored primarily by Kelvin Liu and Ian Booth) demonstrate best-practice patterns: actual content verification (not just status checks), full lifecycle testing, access control assertions (negative tests), and cross-model behavior. All other suites should aspire to this depth.

---

## Suite Classification

All 48 existing suites classified:

### Sanity (1 suite)

| Suite | Provider | Rationale |
|-------|----------|-----------|
| static_analysis | N/A | Linting, code quality — no runtime |

### Smoke (6 suites)

| Suite | Provider | Rationale |
|-------|----------|-----------|
| smoke | All | Basic build + deploy verification |
| smoke_k8s | K8s | Basic K8s charm deployment |
| smoke_k8s_psql | K8s | Database charm on K8s (operator + storage) |
| deploy | All | Core deployment workflows |
| cli | All | CLI command verification |
| charmhub | All | Charm discovery and download |

### Regression (26 suites)

deploy_caas, storage, storage_k8s, sidecar, caasadmission, secrets_iaas, secrets_k8s, relations, cmr, resources, hooks, hooktools, actions, model, controller, bootstrap, agents, refresh, constraints, authorized_keys, credential, user, network, machine, appdata, dashboard

### Integration (15 suites)

upgrade, controllercharm, coslite, kubeflow, ck, deploy_aks, spaces_ec2, spaces_gce, cloud_azure, cloud_gce, firewall, ovs_maas, manual, unmanaged, examples

---

## Proposed New Suites

| Suite | Tier | Provider | What It Validates | Gap Addressed |
|-------|------|----------|-------------------|---------------|
| constraints_k8s | Regression | K8s | K8s-specific constraints (cpu, memory, deployment-type) | Constraints suite explicitly skips K8s |
| deploy_caas_lifecycle | Regression | K8s | Full CAAS app lifecycle: deploy, scale, config, action, remove | deploy_caas only deploys and checks status |
| controller_lifecycle | Regression | All | Bootstrap, enable-ha, disable-ha, controller config, teardown | controller suite only tests metrics/HA/tracing |

---

## External Dependencies

| Dependency | Status | Impact |
|-----------|--------|--------|
| **norma-k8s** charm | Out of scope — contract defined (CC-K1–K11) | Blocks K8s suite migration (A2) |
| **norma** charm | Out of scope — contract defined (CC-M1–M11) | Blocks IaaS suite migration (A2) |
| **Charmhub** | Accepted external dependency | Required for charm fetching; norma charms will also come from charmhub |
| **Snap store** | Accepted external dependency | Required for environment preparation; cannot be self-contained |
| **GitHub Actions runners** | Assumed available (self-hosted, quad-xlarge/xxlarge) | Predicate system relies on parallelism |
| **MicroK8s** | Assumed as K8s CI provider | Provider predicate "k8s" maps to MicroK8s |

Audit phases (A1a, A1b), structural phases (B1–B4), and fail-fast DAG (A4) are NOT blocked on charm readiness.

---

## Success Criteria

1. All 50 suites classified with tier, provider, path predicates, and requirement level
2. PRs touching a single domain trigger ≤10 suites (down from all-or-nothing)
3. PR smoke feedback completes within 15 minutes
4. Push-to-main runs 20+ suites without exceeding 90 minutes wall-clock
5. Nightly builds execute all 50+ suites across available providers
6. Adding a new suite requires zero workflow file changes
7. All smoke and regression suites use calibration charms
8. Every mutating test includes substrate-level verification
9. Every suite declares intra-suite dependencies; bootstrap failure skips dependents within 30 seconds
10. Calibration charm contract documented and fed into norma charm specs

---

## Open Issues

- Confirm `yq` version compatibility across all GitHub Actions runner images (v4 required for YAML processing)
- Determine whether `predicates.yaml` validation should be a pre-commit hook or CI-only check
- Decide on flaky test management approach (auto-quarantine deferred to future phase; see FR-028–032)
