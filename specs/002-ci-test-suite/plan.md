# Implementation Plan: Predicate-Based CI Test Suite

**Branch**: `002-ci-test-suite` | **Date**: 2026-02-17 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/002-ci-test-suite/spec.md`

## Summary

Overhaul Juju's integration test CI from a static, coverage-poor system (6 of 50 suites on PRs) to a predicate-driven, quality-audited test suite. Two parallel tracks: **Track A** improves test content (coverage audit, quality audit, calibration charm migration, substrate verification, fail-fast DAG); **Track B** adds smart CI structure (predicate schema, suite classification, unified workflow, resource sweeper). Every deliverable is independently mergeable and improves CI from the moment it lands.

## Technical Context

**Language/Version**: Bash (test framework, predicate evaluator), YAML (predicates, GitHub Actions), Go (per `go.mod` — Juju codebase under test)
**Primary Dependencies**: GitHub Actions, `dorny/paths-filter@v3` (existing), `jq` (existing, used in test assertions), MicroK8s (K8s CI provider), LXD (IaaS CI provider)
**Storage**: YAML files (`tests/suites/<name>/predicates.yaml`) — no database
**Testing**: Integration tests via `tests/main.sh`; predicate evaluator tested via unit tests (bash or Go)
**Target Platform**: GitHub Actions self-hosted runners (Ubuntu, quad-xlarge/xxlarge)
**Project Type**: CI infrastructure (scripts + workflows + metadata)
**Performance Goals**: PR smoke feedback <15 min; push-to-main regression <90 min wall-clock; nightly all tiers
**Constraints**: Backward-compatible with existing `tests/main.sh` invocation; no changes to existing suite code for predicate system; iterative deployment (no big-bang cutover)
**Scale/Scope**: 50 existing suites, 19 GitHub Actions workflows, ~175 Juju capabilities to cover

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Applicable? | Status | Notes |
|-----------|------------|--------|-------|
| I. Everything Fails | Yes | PASS | Predicate evaluation is stateless; test runner already handles cleanup on failure; resource sweeper adds idempotent cleanup |
| II. Strict Architectural Layering | Minimal | PASS | This work touches `tests/` and `.github/workflows/`, not the Go codebase layers |
| III. Managed Concurrency | No | N/A | No new goroutines; CI parallelism is GitHub Actions job-level |
| IV. Test Discipline | Yes | PASS | Core goal: improve test determinism (fail-fast DAG), sterility (calibration charms), and verification (substrate checks) |
| V. Domain Service Encapsulation | No | N/A | No domain service changes |
| VI. Access to Clouds via Providers | No | N/A | Tests access providers via Juju CLI, not Go imports |
| VII. Resource Ownership | Yes | PASS | Resource sweeper (FR-033–035) ensures CI-created resources have explicit ownership and cleanup |
| VIII. Simplicity and Minimalism | Yes | PASS | Reuse-first principle (FR-055–057); enhance existing tests, don't rewrite; single YAML file per suite |

**Gate result: PASS** — No violations. Constitution principles IV (Test Discipline) and VIII (Simplicity) are actively advanced by this work.

## Project Structure

### Documentation (this feature)

```text
specs/002-ci-test-suite/
├── spec.md              # Feature specification (complete)
├── plan.md              # This file
├── research.md          # Phase 0 output — test framework analysis
├── data-model.md        # Phase 1 output — predicates.yaml schema + entities
├── quickstart.md        # Phase 1 output — how to add/modify predicates
├── contracts/           # Phase 1 output — calibration charm contract (CC-*)
│   └── charm-contract.yaml
├── coverage-analysis.md # Pre-existing — 175 capabilities mapped
├── ci-best-practices-research.md  # Pre-existing — industry research
├── ci-test-suite-reference.md     # Pre-existing — current suite inventory
└── github-actions-reference.md    # Pre-existing — current workflow inventory
```

### Source Code (repository root)

```text
tests/
├── main.sh                          # EXISTING — test dispatcher (enhanced with --fail-fast, --dag)
├── includes/
│   ├── run.sh                       # EXISTING — enhanced with DAG-aware skip logic
│   ├── juju.sh                      # EXISTING — enhanced with ci-<run-id> naming
│   ├── wait-for.sh                  # EXISTING — no changes
│   ├── check.sh                     # EXISTING — no changes
│   ├── substrate.sh                 # NEW — substrate verification helpers (kubectl, lxc)
│   └── predicates.sh               # NEW — predicate evaluation functions (read YAML, match paths)
├── suites/
│   ├── smoke/
│   │   ├── task.sh                  # EXISTING — no changes
│   │   └── predicates.yaml          # NEW — tier, provider, paths, charms, test DAG
│   ├── deploy/
│   │   ├── task.sh                  # EXISTING — no changes (or minimal charm swap)
│   │   └── predicates.yaml          # NEW
│   ├── ... (50 suites, each gets predicates.yaml)
│   └── constraints_k8s/             # NEW suite
│       ├── task.sh
│       └── predicates.yaml
├── evaluate-predicates.sh           # NEW — reads all predicates.yaml, outputs matrix JSON
└── ci-sweeper.sh                    # NEW — discovers and destroys stale ci-* resources

.github/workflows/
├── static-analysis.yml              # EXISTING — no changes
├── cla.yml                          # EXISTING — no changes
├── merge.yml                        # EXISTING — no changes
├── context-tests.yml                # EXISTING — no changes
├── integration-tests.yml            # NEW — unified predicate-aware workflow
└── ci-sweeper.yml                   # NEW — scheduled resource cleanup
```

**Structure Decision**: Enhancement of existing `tests/` directory. New files added alongside existing ones. No restructuring of existing suite directories. Two new helpers in `tests/includes/`, one new evaluator script at `tests/`, one new workflow.

## Phase 0: Research

### Research Findings

All research was conducted prior to plan creation via dedicated research agents. Key findings consolidated below; full details in pre-existing reference documents.

#### R1: Existing Test Framework Architecture

**Decision**: Enhance the existing bash test framework; do not replace it.

**Rationale**: The framework (`main.sh` → `run.sh` → `suites/*/task.sh`) is well-structured, understood by the team, and covers the dispatch, filtering, bootstrap, and assertion lifecycle. The `skip()` function in `run.sh` already provides whitelist/blacklist filtering — extending it for DAG-aware skipping is straightforward.

**Alternatives considered**:
- Go test binary for integration tests → rejected: would require rewriting all 50 suites
- Python pytest framework → rejected: adds dependency, no team familiarity
- Keep bash, no changes → rejected: misses fail-fast and predicate goals

#### R2: Predicate Evaluation Strategy

**Decision**: Bash script (`evaluate-predicates.sh`) that reads all `predicates.yaml` files, accepts event context (type, changed files, provider) as input, and outputs a JSON matrix of activated suites for GitHub Actions.

**Rationale**: Bash is consistent with the existing test framework. The evaluator is a pure function (inputs → outputs) with no state. `yq` (already available on runners) parses YAML. Output format matches GitHub Actions `matrix` input natively.

**Alternatives considered**:
- Go binary → rejected: adds build step, overkill for YAML parsing + glob matching
- GitHub Actions composite action → rejected: locks logic into GitHub-specific construct
- `dorny/paths-filter` extension → rejected: doesn't support per-suite YAML metadata

#### R3: Fail-Fast DAG Implementation

**Decision**: Extend `run.sh` to read the `tests` section of `predicates.yaml`, topologically sort tests, and skip dependents when a prerequisite fails. Track pass/fail state in a temporary file per suite run.

**Rationale**: The DAG is small (typically 3–10 tests per suite). Topological sort in bash is simple for small graphs. The `skip()` function already exists — extending it to check a `FAILED_TESTS` file is minimal.

**Alternatives considered**:
- External DAG runner (e.g., `make` with dependencies) → rejected: adds complexity, doesn't integrate with existing `run()` function
- No DAG, just `set -e` (stop on first error) → rejected: too coarse — kills the entire suite instead of selectively skipping dependents

#### R4: Substrate Verification Approach

**Decision**: New `tests/includes/substrate.sh` providing functions like `substrate_check_pod_exists()`, `substrate_check_namespace_gone()`, `substrate_check_pvc_count()`. Tests call these after Juju operations. Provider-aware: functions detect current provider and use appropriate tool (kubectl/lxc/cloud CLI).

**Rationale**: Centralizing substrate checks in a shared helper ensures consistency and reuse. Provider detection is already done in `juju.sh` (bootstrap provider selection).

**Alternatives considered**:
- Inline substrate checks in each test → rejected: duplication, inconsistency
- Charm-side verification only (via actions) → rejected: misses cases where Juju and charm disagree

#### R5: GitHub Actions Workflow Architecture

**Decision**: Single new `integration-tests.yml` workflow that:
1. Calls `evaluate-predicates.sh` to get activated suite matrix
2. Uses `matrix` strategy with the output to spawn parallel jobs
3. Each job runs `main.sh <suite> --fail-fast` with provider from matrix
4. Regression-tier jobs on PRs use `continue-on-error: true` (informational)

Existing `smoke.yml` is kept during transition, deprecated once `integration-tests.yml` proves equivalent, then removed.

**Rationale**: GitHub Actions native `matrix` strategy handles parallelism. `continue-on-error` achieves non-blocking regression checks. Keeping `smoke.yml` during transition honors the "CI never goes dark" principle.

**Alternatives considered**:
- Reusable workflows per tier → rejected: still results in workflow sprawl
- Separate workflow per provider → rejected: duplicates predicate evaluation logic

#### R6: Resource Sweeper Design

**Decision**: Standalone `ci-sweeper.sh` script + `ci-sweeper.yml` scheduled workflow (hourly). Lists all Juju controllers/models matching `ci-*` pattern, destroys those older than TTL (default 4h). Also sweeps stale LXD containers and MicroK8s namespaces.

**Rationale**: Simple, idempotent, no external dependencies. Already proven pattern (Terraform CI resource sweepers).

## Phase 1: Design & Contracts

### Data Model

See [data-model.md](data-model.md) for full schema. Summary:

**Primary entity**: `predicates.yaml` — one per suite, co-located at `tests/suites/<name>/predicates.yaml`

```yaml
# Schema version for forward compatibility
schema: "1.0"

# Predicate dimensions (FR-004, FR-008)
tier: regression              # sanity|smoke|regression|integration
provider: k8s                 # all|k8s|iaas|ec2|gce|azure|maas|manual|unmanaged
paths:                        # glob patterns (FR-013)
  - domain/storage/**
  - internal/worker/caas*/**
  - caas/**

# Test quality metadata (FR-037)
charms:
  - name: norma-k8s
    type: calibration          # calibration|third-party

# Documentation (FR-003)
rationale: "K8s PV/PVC handling and race conditions"

# Intra-suite test DAG (FR-039)
tests:
  - name: setup_bootstrap
    type: prerequisite
  - name: deploy_norma
    type: prerequisite
    depends_on: [setup_bootstrap]
  - name: test_storage_attach
    depends_on: [deploy_norma]
  - name: test_storage_persist_restart
    depends_on: [deploy_norma]
  - name: test_storage_detach
    depends_on: [test_storage_attach]
```

**Evaluator output** (JSON for GitHub Actions matrix):

```json
{
  "include": [
    {"suite": "smoke", "provider": "localhost", "tier": "smoke", "required": true},
    {"suite": "smoke_k8s", "provider": "microk8s", "tier": "smoke", "required": true},
    {"suite": "storage_k8s", "provider": "microk8s", "tier": "regression", "required": false}
  ]
}
```

**State transitions**: None — predicates are static metadata. The evaluator is a pure function.

### Contracts

See [contracts/charm-contract.yaml](contracts/charm-contract.yaml) for the full calibration charm contract extracted from the spec (CC-01 through CC-M11).

### Architecture Decisions

#### AD-1: Predicate Evaluator is a Script, Not a Service

The evaluator is a stateless bash script. It reads YAML files and CLI arguments, outputs JSON. No daemon, no database, no API. This maximizes simplicity (Constitution VIII) and makes it testable locally.

#### AD-2: Existing Workflows Coexist During Transition

`smoke.yml` continues to run alongside `integration-tests.yml` until the new workflow proves equivalent over 2 weeks of production use. Then `smoke.yml` is removed. This honors the governing principle ("CI never goes dark").

#### AD-3: DAG is Per-Suite, Not Cross-Suite

Test dependencies are declared within a suite's `predicates.yaml`. There is no cross-suite dependency system. If suite B depends on suite A passing (e.g., both need bootstrap), each suite bootstraps independently. This keeps suites independently executable (FR-026).

#### AD-4: Substrate Helpers are Opt-In

`substrate.sh` provides helper functions. Existing tests are not forced to use them immediately. Tests are enhanced incrementally (reuse-first principle). New tests and migrated tests use them from the start.

## Execution Phases (Two Parallel Tracks)

### Track A — Test Content Quality (PRIORITY)

#### A1a: Coverage Audit

**Goal**: Map every existing suite to Juju capabilities. Identify gaps and redundancy.
**Deliverable**: Updated `coverage-analysis.md` with per-suite verdicts (keep/enhance/migrate/rewrite).
**Mergeable**: Yes — the analysis document itself is a deliverable.
**Dependencies**: None — starts day 1.

**Work items**:
1. Read all 50 `task.sh` files and map each test function to the Juju capability it exercises
2. Cross-reference against the 175-capability inventory in `coverage-analysis.md`
3. Identify over-testing (multiple suites testing the same capability identically)
4. Identify NONE/MINIMAL coverage gaps and propose new test groups
5. Assign verdict per suite: keep, enhance, migrate, or rewrite (default: keep/enhance per FR-057)

#### A1b: Quality Audit

**Goal**: Evaluate HOW each suite tests — sterility, substrate verification, DAG.
**Deliverable**: Per-suite migration plan in `coverage-analysis.md`.
**Mergeable**: Yes — the analysis document is a deliverable.
**Dependencies**: None — starts day 1, can run parallel with A1a.

**Work items**:
1. For each suite, catalog: which charms used (calibration vs third-party), whether substrate verification exists, what the implicit test dependency order is
2. Score each suite on three dimensions: sterility (0-2), substrate verification (0-2), fail-fast readiness (0-2)
3. Produce migration plan per suite: what charm to swap (when norma is ready), what substrate checks to add, what DAG to declare
4. Flag suites where test logic validates charm behavior rather than Juju behavior

#### A2: Migrate Suites to Calibration Charms

**Goal**: Replace third-party charms with norma-k8s/norma in smoke and regression tiers.
**Deliverable**: Modified `task.sh` files with charm references swapped. One PR per suite.
**Mergeable**: Yes — each suite migration is an independent PR.
**Dependencies**: A1b (migration plans), norma charms meeting contract (external).

**Work items** (per suite):
1. Swap charm reference (e.g., `postgresql-k8s` → `norma-k8s`)
2. Update deployment commands and wait conditions
3. Preserve all existing test assertions and flow
4. Add `predicates.yaml` charm declaration with `type: calibration`
5. Verify suite passes locally before PR

#### A3: Add Substrate Verification

**Goal**: Every mutating test verifies outcome on the substrate.
**Deliverable**: New `tests/includes/substrate.sh` + enhanced test functions. One PR per batch of suites.
**Mergeable**: Yes — each batch is independent.
**Dependencies**: A2 (migrated suites are easier to verify, but verification can be added to non-migrated suites too).

**Work items**:
1. Create `tests/includes/substrate.sh` with provider-aware verification functions:
   - `substrate_check_pod_exists <app> <namespace>` (K8s)
   - `substrate_check_namespace_gone <model>` (K8s)
   - `substrate_check_pod_count <app> <expected> <namespace>` (K8s)
   - `substrate_check_container_exists <name>` (LXD)
   - `substrate_check_container_gone <name>` (LXD)
   - `substrate_check_machine_count <expected>` (IaaS)
2. Add substrate verification calls after key Juju operations in each suite
3. Ensure verification failures report clearly: "Juju reported success but substrate disagrees"

#### A4: Implement Fail-Fast DAG

**Goal**: Suite setup failures skip dependent tests instantly.
**Deliverable**: Enhanced `tests/includes/run.sh` + `predicates.yaml` test DAGs. One PR.
**Mergeable**: Yes — DAG is opt-in. Suites without `tests` section in predicates.yaml run as before.
**Dependencies**: A1b (DAG mapping), B1 (predicates.yaml schema — but DAG section is independent).

**Work items**:
1. Extend `run.sh`:
   - On suite start, read `predicates.yaml` `tests` section if present
   - Build in-memory DAG (bash associative arrays)
   - Topologically sort tests
   - Before each test, check if any `depends_on` test has failed → skip with message
   - Track results in `$TEST_DIR/test-results.tmp`
2. Extend `main.sh`:
   - Add `--fail-fast` flag (optional, for CI use)
   - When set, stop suite on first non-prerequisite failure
3. Add `tests` section to `predicates.yaml` for each suite (can be done incrementally)
4. Report: print dependency graph at suite start, print skip reasons

#### A5: Write New Tests for Coverage Gaps

**Goal**: Fill NONE/MINIMAL coverage gaps with new test groups.
**Deliverable**: New suite directories with `task.sh` and `predicates.yaml`. One PR per new suite.
**Mergeable**: Yes — each new suite is independent.
**Dependencies**: A1a (gap identification), calibration charms (for new tests using norma).

**Work items** (from spec FR-018):
1. `constraints_k8s` — K8s constraint testing (tier: regression, provider: k8s)
2. `deploy_caas_lifecycle` — Full CAAS lifecycle (tier: regression, provider: k8s)
3. `controller_lifecycle` — Controller bootstrap/HA/teardown (tier: regression, provider: all)
4. `deploy_caas_deployment_type` — Deployment/DaemonSet/StatefulSet (tier: regression, provider: k8s)
5. `storage_k8s_deployment` — Storage with Deployment type (tier: regression, provider: k8s)
6. Additional gaps identified by A1a audit

### Track B — CI Structure (PARALLEL)

#### B1: Define Predicate Schema and Evaluator

**Goal**: Predicate evaluation script + schema documentation.
**Deliverable**: `tests/evaluate-predicates.sh`, schema docs in `data-model.md`. One PR.
**Mergeable**: Yes — the evaluator is a standalone script, doesn't change existing behavior.
**Dependencies**: None — starts day 1.

**Work items**:
1. Define `predicates.yaml` schema (done in spec, formalize in `data-model.md`)
2. Write `evaluate-predicates.sh`:
   - Input: `--event-type <pr|push|schedule|dispatch>`, `--changed-files <file>`, `--provider <name>`, `--tier-override <tier>`
   - Logic: scan all `tests/suites/*/predicates.yaml`, evaluate tier eligibility, path matching, provider matching
   - Output: JSON matrix for GitHub Actions
   - Handle defaults: missing predicates.yaml → tier=integration, provider=all, paths=wildcard (FR-009)
   - Handle wildcards: `core/`, `go.mod`, `tests/includes/` changes → all suites eligible (FR-007)
3. Write unit tests for the evaluator (bash test cases with known inputs/outputs)
4. Write `tests/includes/predicates.sh` for shared predicate functions usable by other scripts

#### B2: Classify All 50 Suites

**Goal**: Every existing suite has a `predicates.yaml` file.
**Deliverable**: 50 new YAML files. Can be done in batches (smoke tier first, then regression, then integration).
**Mergeable**: Yes — each batch is independent. Predicates are inert until the workflow reads them.
**Dependencies**: A1a/A1b (audit data for tier/path assignment), B1 (schema).

**Work items**:
1. Smoke tier (6 suites): `smoke`, `smoke_k8s`, `smoke_k8s_psql`, `deploy`, `cli`, `charmhub`
2. Regression tier (25 suites): `deploy_caas`, `storage`, `storage_k8s`, `sidecar`, ... (per spec table)
3. Integration tier (15 suites): `upgrade`, `controllercharm`, `coslite`, ... (per spec table)
4. Sanity tier (1 suite): `static_analysis`
5. For each: determine paths by analyzing which Go packages the suite exercises (from A1a)
6. Validate: run evaluator against sample changesets and verify expected activation

#### B3: Build Unified Workflow

**Goal**: `integration-tests.yml` that replaces scattered test workflows.
**Deliverable**: New workflow file. Deployed alongside existing `smoke.yml` during transition.
**Mergeable**: Yes — runs in parallel with existing workflows. No disruption.
**Dependencies**: B1 (evaluator), B2 (at least smoke tier classified).

**Work items**:
1. Create `.github/workflows/integration-tests.yml`:
   - Triggers: `pull_request`, `push` (main/release), `schedule` (nightly), `workflow_dispatch`
   - Job 1: `evaluate` — checkout, get changed files, run `evaluate-predicates.sh`, output matrix
   - Job 2: `test` — matrix strategy from Job 1 output, each runs `main.sh <suite> --fail-fast`
   - Regression tier on PRs: `continue-on-error: true` (non-blocking)
   - All tiers on push-to-main: `continue-on-error: false` (blocking)
   - Manual dispatch: tier override via `workflow_dispatch` input
2. Configure runner labels per provider (existing: quad-xlarge for LXD, xxlarge for K8s)
3. Run in parallel with `smoke.yml` for 2 weeks
4. Verify equivalent or better coverage
5. Deprecate `smoke.yml`, `postgresql-k8s.yml`, `upgrade.yml`, `microk8s-tests.yml`

#### B4: Add Resource Sweeper

**Goal**: Automated cleanup of stale CI resources.
**Deliverable**: `tests/ci-sweeper.sh` + `.github/workflows/ci-sweeper.yml`. One PR.
**Mergeable**: Yes — independent of all other work.
**Dependencies**: None — can start anytime.

**Work items**:
1. Create `tests/ci-sweeper.sh`:
   - List all Juju controllers matching `ci-*` pattern
   - For each: check age via `juju show-controller --format=json`
   - Destroy controllers older than TTL (default 4h, configurable via env var)
   - Also sweep: LXD containers matching `ci-*`, MicroK8s namespaces matching `ci-*`
   - Idempotent: no errors if nothing to sweep
2. Update `tests/includes/juju.sh` `bootstrap()`:
   - Prefix controller names with `ci-${GITHUB_RUN_ID:-local}-` when `CI=true`
   - Backward-compatible: local runs keep current naming
3. Create `.github/workflows/ci-sweeper.yml`:
   - Schedule: every hour
   - Runs `ci-sweeper.sh` with 4h TTL
   - Manual dispatch for emergency cleanup

## Dependency Graph

```
START
  ├─► A1a (coverage audit) ─────────────► A5 (new tests)
  │         │
  │         └──► B2 (classify suites) ──► B3 (unified workflow) ──► retire old workflows
  │
  ├─► A1b (quality audit) ──► A2 (migrate charms) ──► A3 (substrate verify)
  │         │                    ↑
  │         └──► A4 (fail-fast)  │ (blocked on norma charms - external)
  │
  └─► B1 (predicate schema) ──► B2 (classify) ──► B3 (workflow)
  │
  └─► B4 (resource sweeper) — independent, anytime
```

**Day 1 starts**: A1a, A1b, B1, B4
**External blocker**: A2 (charm migration) waits for norma charms meeting contract

## Post-Phase 1: Constitution Re-Check

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Everything Fails | PASS | Evaluator is stateless; DAG handles test failures gracefully; sweeper is idempotent |
| II. Strict Architectural Layering | PASS | No Go codebase layer violations; all work in tests/ and .github/ |
| IV. Test Discipline | PASS | Actively advanced: DAG for determinism, substrate verification for correctness, calibration charms for isolation |
| VII. Resource Ownership | PASS | Sweeper ensures CI resources have owner and TTL |
| VIII. Simplicity and Minimalism | PASS | Reuse-first (FR-055–057); single YAML per suite; bash evaluator; no external services |

**Gate result: PASS** — No violations post-design.

## Complexity Tracking

No constitution violations to justify. The design follows existing patterns and adds minimal new abstractions.

## Artifacts Generated

| Artifact | Status | Path |
|----------|--------|------|
| plan.md | Complete | `specs/002-ci-test-suite/plan.md` (this file) |
| research.md | Complete | `specs/002-ci-test-suite/research.md` |
| data-model.md | Complete | `specs/002-ci-test-suite/data-model.md` |
| quickstart.md | Complete | `specs/002-ci-test-suite/quickstart.md` |
| contracts/charm-contract.yaml | Complete | `specs/002-ci-test-suite/contracts/charm-contract.yaml` |
