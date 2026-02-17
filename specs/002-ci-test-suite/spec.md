# Feature Specification: Predicate-Based CI Test Suite

**Feature Branch**: `002-ci-test-suite`
**Created**: 2026-02-15
**Status**: Draft
**Input**: User description: "New CI test suite for all Juju with predicate-based activation, test tier taxonomy (sanity/smoke/regression), path-based triggering, and GitHub Actions integration."

## Governing Principle: Iterative Improvement, Not Revolution

Every phase of this work MUST deliver a deployable improvement to the existing CI — not a parallel system that replaces it wholesale. The current tests run today; each iteration makes them better while keeping them running.

This means:
1. **Every phase ships standalone value.** No phase produces only plans, schemas, or scaffolding that requires a later phase to become useful. If A1a audits a suite and finds it needs substrate verification, that verification can be added and deployed immediately — it doesn't wait for predicate infrastructure.
2. **Changes are incremental and backward-compatible.** Adding a `predicates.yaml` to a suite doesn't change how it runs today. Adding substrate verification to a test doesn't break its existing assertions. Swapping a charm in a test doesn't restructure the test logic.
3. **The existing CI never goes dark.** At no point should a phase require disabling current workflows before new ones are ready. Old and new coexist until the new is proven, then the old is retired.
4. **Prefer enhancing over replacing.** A good test with a missing substrate check gets the check added — it doesn't get rewritten. A working suite that uses postgresql-k8s gets the charm swapped — its test logic stays.
5. **Deployability is the gate for each phase.** The question for every deliverable is: "Can this be merged and used in CI today, improving what we have?" If the answer is no, the scope is too large — break it down further.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Fast PR Feedback Loop (Priority: P1)

A developer opens a pull request that modifies files in `domain/application/`. The CI system automatically determines which test groups are relevant to the changed paths and runs only those tests. The developer receives feedback within 15 minutes on whether their change breaks core functionality, rather than waiting for the full suite or getting no integration coverage at all.

**Why this priority**: Fast, targeted feedback is the single highest-value improvement. Today, PRs run only `smoke` and `smoke_k8s` regardless of what changed — missing coverage for 44 of 50 existing suites. Developers either wait for manual test runs or merge with uncertainty.

**Independent Test**: Can be validated by opening a PR that touches a known path (e.g., `domain/secret/`) and verifying that exactly the expected test groups activate (e.g., `secrets_iaas`, `secrets_k8s`) while unrelated groups are skipped.

**Acceptance Scenarios**:

1. **Given** a PR that modifies only files under `domain/application/`, **When** the CI runs, **Then** only test groups tagged as relevant to `domain/application/` are executed (e.g., `deploy`, `deploy_caas`, `relations`), and unrelated groups (e.g., `firewall`, `spaces_ec2`) are skipped.
2. **Given** a PR that modifies only documentation or `.md` files, **When** the CI runs, **Then** no integration test groups are triggered (only static analysis and docs checks run).
3. **Given** a PR that modifies `core/` or `go.mod`, **When** the CI runs, **Then** a broader set of test groups is triggered because foundational code affects many downstream consumers.
4. **Given** a PR that modifies files in `internal/worker/caasapplication/`, **When** the CI runs, **Then** K8s-related test groups activate but IaaS-only groups do not.

---

### User Story 2 - Tiered Test Execution (Priority: P1)

The CI system classifies every test group into a well-defined tier — sanity, smoke, regression, or integration — so that different events trigger different depths of testing. A PR runs sanity + smoke. A merge to main runs sanity + smoke + regression. A nightly or release build runs all tiers including full integration.

**Why this priority**: Without tiers, the choice is binary: run everything (too slow for PRs) or run almost nothing (current state — only 6 of 50 suites on PRs). Tiers let the system scale test depth proportionally to risk.

**Independent Test**: Can be validated by configuring a tier label on each test group and verifying that a PR event triggers only sanity + smoke tiers, while a push-to-main event additionally triggers regression tier groups.

**Acceptance Scenarios**:

1. **Given** a PR event, **When** the CI evaluates test predicates, **Then** only groups classified as "sanity" or "smoke" tier are eligible to run (subject to further path filtering).
2. **Given** a push-to-main event, **When** the CI evaluates predicates, **Then** groups classified as "sanity", "smoke", or "regression" tier are eligible.
3. **Given** a nightly schedule or manual "full" trigger, **When** the CI evaluates predicates, **Then** all tiers including "integration" are eligible.
4. **Given** a test group classified as "smoke" tier, **When** a developer inspects the test metadata, **Then** the tier classification and rationale are documented alongside the group.

---

### User Story 3 - Predicate Composition (Priority: P2)

A test group has multiple activation conditions (predicates) that are composed together. For example, the `storage_k8s` suite requires: tier is "regression" OR higher, AND provider is "k8s", AND paths include `domain/storage/` or `internal/worker/caasapplication/` or `caas/`. The CI system evaluates these predicates and only runs the group when all conditions are satisfied.

**Why this priority**: Simple path-matching alone is insufficient. Juju's test suites already have implicit predicates (provider guards, tier assumptions) baked into bash `case` statements. Making these explicit and composable enables the system to make smarter scheduling decisions.

**Independent Test**: Can be validated by defining predicates for a test group and then simulating different event contexts (e.g., PR with K8s path changes vs. PR with IaaS-only changes) to verify correct activation.

**Acceptance Scenarios**:

1. **Given** a test group with predicates `tier: smoke`, `provider: k8s`, `paths: [caas/**, internal/worker/caas*/**]`, **When** a PR changes only `cmd/juju/` files, **Then** the group does not activate (path predicate not satisfied).
2. **Given** the same test group, **When** a PR changes `caas/broker.go` and the event is a PR, **Then** the group activates (tier=smoke eligible on PR, provider and path match).
3. **Given** a test group with predicate `paths: [*]` (wildcard), **When** any PR is opened, **Then** the group is eligible regardless of paths changed (acts as an "always run" for its tier).

---

### User Story 4 - Test Group Classification of Existing Suites (Priority: P2)

All 50 existing test suites are classified into the tier taxonomy and assigned path predicates, so that the new system immediately covers the existing test inventory without requiring new tests to be written. Each suite's existing provider guards are preserved as provider predicates.

**Why this priority**: The new system must be backward-compatible with the existing suite structure. Value comes from classifying what already exists, not just from adding new tests.

**Independent Test**: Can be validated by producing a classification table of all 50 suites and verifying that: (a) every suite has a tier, (b) every suite has a provider predicate, (c) every suite has path predicates, and (d) the existing provider guards in bash are consistent with the predicate metadata.

**Acceptance Scenarios**:

1. **Given** the existing `smoke_k8s` suite, **When** it is classified, **Then** it has tier "smoke", provider "k8s", and paths that include broad K8s-related directories.
2. **Given** the existing `deploy_caas` suite, **When** it is classified, **Then** it has tier "regression", provider "k8s", and paths that include deployment-related code.
3. **Given** a suite currently not run in any GitHub Actions workflow (e.g., `relations`), **When** it is classified as tier "regression" with paths `[domain/relation/**, apiserver/facades/**/relation**]`, **Then** it becomes automatically eligible when a PR touches relation code and is promoted to main.

---

### User Story 5 - GitHub Actions Workflow Consolidation (Priority: P3)

The current 19 GitHub Actions workflows are consolidated into a smaller number of predicate-aware workflows that invoke the bash test runner with the appropriate filters. Instead of one workflow per test scenario (smoke.yml, postgresql-k8s.yml, upgrade.yml, etc.), a unified integration workflow evaluates predicates and dispatches matching test groups.

**Why this priority**: The current workflow sprawl (19 files, some disabled, inconsistent patterns) makes maintenance difficult and creates gaps in coverage. Consolidation reduces duplication while the predicate system handles activation logic.

**Independent Test**: Can be validated by creating a single integration workflow that replaces `smoke.yml` and `postgresql-k8s.yml` while producing identical test coverage for the same events.

**Acceptance Scenarios**:

1. **Given** the consolidated workflow, **When** a PR is opened, **Then** it produces the same test results as the current `smoke.yml` would (smoke + deploy on LXD, smoke_k8s on MicroK8s).
2. **Given** a push to main, **When** the consolidated workflow runs, **Then** it additionally runs regression-tier suites that were not previously automated.
3. **Given** the consolidated workflow, **When** a new test group is added with appropriate predicates, **Then** it is automatically picked up by the workflow without modifying any `.yml` file.

---

### User Story 6 - New Test Coverage for Gaps (Priority: P3)

New test groups are proposed to fill identified coverage gaps. These include K8s constraint testing (currently skipped), K8s deployment-type variants, cross-provider regression suites, and controller lifecycle tests that are currently manual-only.

**Why this priority**: While classifying existing suites delivers most of the value, specific gaps cannot be addressed by classification alone — they require new test content.

**Independent Test**: Can be validated by implementing one new test group (e.g., K8s constraints) and verifying it integrates with the predicate system and activates on relevant PRs.

**Acceptance Scenarios**:

1. **Given** a new `constraints_k8s` test group, **When** a PR modifies constraint-related code, **Then** the group activates and tests K8s-specific constraint behavior.
2. **Given** a new `deploy_caas_deployment_type` test group, **When** a PR modifies K8s provider code, **Then** the group activates and tests Deployment/DaemonSet/StatefulSet variants.

---

### User Story 7 - Test Quality: Sterility, Fail-Fast, and Outcome Verification (Priority: P1)

Tests must validate Juju's actual behavior, not external charm behavior. Each test suite uses purpose-built calibration charms (e.g., norma-k8s) instead of third-party charms (e.g., postgresql-k8s) wherever possible — isolating Juju failures from upstream charm bugs. Tests within a suite declare dependencies so that if a foundational step fails (e.g., bootstrap), dependent tests are skipped immediately rather than wasting CI minutes on guaranteed failures. And every test verifies its outcome at the substrate level (e.g., after `juju destroy-model`, confirm the namespace is gone from K8s), not just that the CLI returned exit code 0.

**Why this priority**: Without these properties, more test coverage just means more noise. A test that passes because postgresql-k8s happened to work tells us nothing about a Juju regression. A suite that runs 30 tests after bootstrap failed wastes 30 minutes. A test that checks only CLI output misses silent substrate failures.

**Independent Test**: Can be validated by auditing one existing suite (e.g., `deploy_caas`) against all three criteria: (a) does it use calibration charms or 3rd-party? (b) does it skip dependent tests when a prerequisite fails? (c) does it verify outcomes on the substrate?

**Acceptance Scenarios**:

1. **Given** a test suite that deploys a K8s charm, **When** the suite is reviewed against the sterility principle, **Then** it uses the norma-k8s calibration charm (or equivalent purpose-built charm) rather than a third-party charm like postgresql-k8s, unless the test specifically validates integration with that third-party charm.
2. **Given** a test suite where test B depends on the outcome of test A (e.g., test A bootstraps, test B deploys), **When** test A fails, **Then** test B is skipped immediately with a clear "skipped: dependency failed" status — not executed and not counted as a separate failure.
3. **Given** a test that performs `juju destroy-model mymodel`, **When** the command returns successfully, **Then** the test additionally verifies on the substrate (e.g., `kubectl get namespace mymodel` returns NotFound) before marking the test as passed.
4. **Given** a test suite with 5 tests where test 1 is "bootstrap" and tests 2-5 depend on it, **When** bootstrap fails, **Then** the suite completes in approximately the time of the bootstrap attempt — not 5x that time.
5. **Given** a suite that needs common setup (bootstrap, deploy calibration charm), **When** the suite runs, **Then** setup is performed once as a suite prerequisite, individual tests run against the prepared environment, and each test is atomic (its failure does not corrupt state for subsequent tests).

---

### Edge Cases

- What happens when a PR touches only test infrastructure (e.g., `tests/includes/`) without touching application code? All test groups should be eligible since helper changes could break any suite.
- What happens when a developer force-pushes over a PR? The predicate evaluation must be re-run against the new diff, not cached from the previous push.
- What happens when the predicate metadata file itself is modified? All test groups should be eligible to validate the metadata change.
- What happens when a test group's predicate matches but the group fails to bootstrap (e.g., MicroK8s unavailable)? The failure must be reported as infrastructure failure, not test failure, and must not block unrelated groups.
- What happens when a new suite is added without predicate metadata? It should default to the most conservative behavior: tier "integration" (only runs on full builds), provider "all", paths wildcard.
- What happens when a calibration charm itself has a bug? The failure should be distinguishable from a Juju failure — the charm's test actions should include self-diagnostics and the test output should clearly attribute the failure source.
- What happens when substrate verification times out (e.g., K8s namespace deletion is slow)? The test should use reasonable timeouts with clear reporting: "substrate verification timed out after 120s — namespace still exists" rather than silently passing or hanging.
- What happens when a test within a suite corrupts shared state (e.g., removes a model that other tests depend on)? Tests must be designed to be atomic — each test should either operate on its own resources or explicitly declare shared-state dependencies so the runner can order them correctly.

## Clarifications

### Session 2026-02-16

- Q: How should the spec handle flaky tests? → A: Auto-quarantine — tests failing >5% over a 7-day rolling window are moved to optional (non-blocking) with a 30-day fix-or-delete SLO.
- Q: Where should predicate metadata live? → A: Per-suite YAML file co-located with the test group at `tests/suites/<name>/predicates.yaml`.
- Q: Should regression-tier tests run on PRs? → A: Yes, but as informational-only (non-blocking) status checks. Merge requires only sanity + smoke to pass.
- Q: How should CI handle leaked cloud resources from failed/timed-out jobs? → A: Naming convention (`ci-<run-id>-` prefix) + scheduled sweeper with configurable TTL. No test code changes required.
- Q: Should predicate path filtering use Go transitive dependency analysis (GTA) or curated globs? → A: Start with curated globs (with `core/` and `go.mod` as wildcards); add GTA as a documented future enhancement.

### Session 2026-02-17

- Q: What is the concrete `predicates.yaml` schema? → A: Single file per suite containing all metadata — tier, provider, paths, charms used (with type), rationale, and intra-suite test dependency DAG.
- Q: Where should flaky test tracking data be stored? → A: Defer flaky test management (FR-028–032) to a future phase. Ship predicate system and test quality improvements first.
- Q: Should the spec define runner concurrency limits per event type? → A: Defer — ship without limits, measure actual usage, then add limits based on real data.

## Requirements *(mandatory)*

### Functional Requirements

#### Test Tier Taxonomy

- **FR-001**: The system MUST define exactly four test tiers with clear semantics:
  - **Sanity**: Compilation, static analysis, schema validation — no runtime required. Target: under 5 minutes.
  - **Smoke**: Minimal end-to-end verification per provider — bootstrap, deploy one charm, verify active/idle. Target: under 15 minutes.
  - **Regression**: Feature-area verification — storage, relations, secrets, hooks, constraints, lifecycle. Target: under 60 minutes per provider.
  - **Integration**: Full cross-feature, cross-provider, and upgrade/migration testing. Target: under 4 hours.
- **FR-002**: Each existing test suite MUST be assigned exactly one tier based on its scope and runtime characteristics.
- **FR-003**: Tier assignment MUST be documented with rationale for each suite.

#### Predicate System

- **FR-004**: Every test group MUST have a predicate definition specifying: tier, provider(s), and relevant source paths.
- **FR-005**: Predicates MUST be stored as a per-suite YAML file at `tests/suites/<name>/predicates.yaml`, co-located with the test group's `task.sh`. Adding or modifying predicates MUST NOT require changing GitHub Actions workflow files. The schema MUST include:
  ```yaml
  tier: <sanity|smoke|regression|integration>
  provider: <all|k8s|iaas|ec2|gce|azure|maas|manual|unmanaged>
  paths:
    - <glob pattern>
  charms:
    - name: <charm-name>
      type: <calibration|third-party>
  rationale: "<why this suite exists and its tier>"
  tests:
    - name: <test-function-name>
      type: <prerequisite|test>       # prerequisite = setup step
      depends_on: [<other-test-names>] # optional, forms the DAG
  ```
- **FR-006**: The predicate evaluation MUST support the following event types as inputs: pull_request, push (to main/release), schedule (nightly), and workflow_dispatch (manual with optional tier override).
- **FR-007**: The predicate evaluation MUST support path-based filtering using manually curated glob patterns: given a list of changed files, determine which test groups are relevant. Changes to foundational paths (`core/`, `go.mod`) MUST trigger wildcard behavior (all suites eligible for the active tier) as a conservative fallback.
- **FR-008**: Predicates MUST compose as logical AND across dimensions: a group runs only if its tier is eligible for the event AND its provider is available AND at least one of its paths matches the changeset (or the group has a wildcard path).
- **FR-009**: A test group with no predicate metadata MUST default to: tier "integration", provider "all", paths wildcard — ensuring it runs only on full builds until explicitly classified.

#### Event-to-Tier Mapping

- **FR-010**: The system MUST define a default mapping from events to eligible tiers:
  - `pull_request` → sanity + smoke (required/blocking) + regression (informational/non-blocking)
  - `push` to main/release → sanity + smoke + regression (all required)
  - `schedule` (nightly) → all tiers
  - `workflow_dispatch` → configurable (default: all tiers)
- **FR-011**: The event-to-tier mapping MUST be overridable per workflow dispatch (e.g., a developer can manually trigger regression tier on a PR branch for pre-merge confidence).

#### Path Predicate Mapping

- **FR-012**: The system MUST define path-to-suite mappings for at least the following code areas:
  - `domain/application/` → `deploy`, `deploy_caas`, `refresh`, `resources`
  - `domain/secret/` → `secrets_iaas`, `secrets_k8s`
  - `domain/storage/` → `storage`, `storage_k8s`
  - `domain/relation/` → `relations`, `cmr`
  - `caas/`, `internal/worker/caas*/` → all K8s suites
  - `internal/worker/` → worker-specific suites
  - `apiserver/facades/` → facade-specific suites
  - `cmd/juju/` → `cli`, `charmhub`, client-tests
  - `core/`, `go.mod` → wildcard (all suites eligible for the active tier)
  - `tests/includes/` → wildcard (all suites eligible)
  - `domain/schema/` → DDL validation + any suites that exercise database operations
- **FR-013**: Path predicates MUST support glob patterns (e.g., `caas/**`, `internal/worker/caas*/**`).
- **FR-014**: When multiple suites share a path prefix, all matching suites MUST be activated.

#### Existing Suite Classification

- **FR-015**: All 50 existing test suites MUST be classified with tier, provider, and path predicates.
- **FR-016**: The classification MUST preserve existing provider guards — a suite currently guarded with `case "k8s"` MUST have provider predicate "k8s".
- **FR-017**: Suites currently run in GitHub Actions (`smoke`, `smoke_k8s`, `deploy`, `smoke_k8s_psql`) MUST remain at smoke tier to avoid regression in PR coverage.

#### Proposed New Test Groups

- **FR-018**: The system MUST propose new test groups to fill identified coverage gaps:
  - `constraints_k8s` — K8s-specific constraint testing (currently skipped entirely)
  - `deploy_caas_lifecycle` — Full CAAS application lifecycle (deploy, scale, config, remove) beyond the current single-charm smoke test
  - `controller_lifecycle` — Controller bootstrap, HA, and teardown (currently manual-only in GitHub Actions)
- **FR-019**: Each proposed test group MUST include tier classification, provider predicate, path predicates, and a description of what it validates.

#### Test Quality Standards

##### Test Sterility (Calibration Charms)

- **FR-036**: Test suites MUST prefer purpose-built calibration charms (e.g., norma-k8s) over third-party charms (e.g., postgresql-k8s, ubuntu) for validating Juju behavior. Third-party charms are permitted ONLY when the test explicitly validates integration with that specific charm.
- **FR-037**: Each test suite's `predicates.yaml` MUST declare which charms it uses and whether they are calibration or third-party, enabling audit of sterility across the test inventory.
- **FR-038**: Calibration charms MUST be designed to exercise specific Juju capabilities (relations, storage, config, actions, resources) without introducing external failure modes (database startup, complex operator logic, upstream version incompatibility).

##### Fail-Fast Test Dependencies

- **FR-039**: Test suites MUST support declaring intra-suite dependencies — a directed acyclic graph (DAG) of tests where a test can declare prerequisite tests that must pass before it runs.
- **FR-040**: When a prerequisite test fails, all transitively dependent tests MUST be skipped immediately with a "skipped: dependency `<test-name>` failed" status. Skipped tests MUST NOT count as failures in CI reporting.
- **FR-041**: Common suite setup (bootstrap, deploy calibration charm, prepare environment) MUST be expressed as an explicit prerequisite step in the dependency graph — not inlined into individual tests. If setup fails, all tests in the suite are skipped.
- **FR-042**: The test runner MUST report the dependency graph in its output so that developers can understand why a test was skipped and which upstream failure caused it.

##### Substrate Outcome Verification

- **FR-043**: Tests that mutate Juju state (deploy, destroy, scale, relate, remove) MUST verify the outcome at the substrate level, not solely via Juju CLI/API responses. Examples:
  - After `juju deploy`: verify the workload pod/container exists on the substrate
  - After `juju destroy-model`: verify the namespace/resources are gone from the substrate
  - After `juju scale-application`: verify the correct number of pods/machines exist
  - After `juju relate`: verify the relation data is visible to the charm (via action or status)
- **FR-044**: Substrate verification MUST use the appropriate substrate tool (`kubectl` for K8s, `lxc` for LXD, cloud CLI for IaaS) and MUST be clearly separated from Juju verification in test output — so that a failure says whether Juju reported success but the substrate disagrees.
- **FR-045**: Where substrate verification is not practical (e.g., external cloud providers in integration tier), the test MUST document why and use the deepest available Juju verification (model status, `juju show-*`, debug-log inspection) as a fallback.

#### GitHub Actions Integration

- **FR-020**: The system MUST propose a workflow architecture that invokes the predicate evaluation and dispatches matching test groups.
- **FR-021**: The workflow architecture MUST preserve the existing `context-tests.yml` orchestration pattern (path-based gating of reusable workflows) for non-integration checks (build, gen, docs, ddl, snap).
- **FR-022**: The workflow architecture MUST support matrix expansion: one workflow can spawn parallel jobs for each activated test group.
- **FR-023**: Static analysis, CLA, and merge-monitoring workflows MUST remain unchanged — they are outside the scope of integration test predicates.
- **FR-024**: The system MUST define how failure in one test group is reported without blocking unrelated groups. On PRs, regression-tier checks MUST be reported as informational (non-blocking) status checks — their failure MUST NOT prevent merge. Only sanity and smoke tier checks are merge-blocking on PRs.

#### Flaky Test Management (Deferred — Future Phase)

*Deferred to a future phase. The requirements below are retained for reference but are NOT in scope for initial implementation. Ship predicate system and test quality improvements first.*

- **FR-028**: The system MUST track per-test-group pass/fail rates over a 7-day rolling window.
- **FR-029**: A test group that fails more than 5% of runs over the rolling window WITHOUT a corresponding code change MUST be automatically quarantined — moved from required to optional (non-blocking) status.
- **FR-030**: Quarantined test groups MUST continue to run but MUST NOT block PR merge or post-merge gating.
- **FR-031**: Quarantined test groups MUST have a 30-day fix-or-delete SLO. If not fixed within 30 days, the group MUST be disabled and an issue filed.
- **FR-032**: The system MUST provide visibility into quarantine status — developers must be able to see which groups are quarantined and why.

#### Calibration Charm Contract

- **FR-046**: All K8s test suites that currently use third-party charms (postgresql-k8s, ubuntu, juju-qa-test) MUST migrate to norma-k8s unless the test specifically validates integration with that third-party charm. Migration is blocked until the charm meets its contract (CC-01 through CC-K11).
- **FR-047**: All IaaS test suites that currently use ad-hoc or third-party charms MUST migrate to the norma machine charm once it meets its contract (CC-01 through CC-M11).
- **FR-048**: The calibration charm contract defined in this spec (CC-01 through CC-M11) MUST be fed into the respective charm specs (norma-k8s, norma) as external requirements. Charm development is out of scope for this spec.
- **FR-049**: Both calibration charms MUST expose a `run-check <capability>` action that returns structured JSON pass/fail results with substrate-level evidence, enabling per-capability validation from test scripts.

#### Test Coverage Audit

- **FR-050**: A coverage audit MUST be performed mapping every existing suite to the Juju capabilities it exercises, producing a capability-to-suite matrix.
- **FR-051**: The coverage audit MUST identify over-testing (multiple suites redundantly testing the same capability) and recommend consolidation.
- **FR-052**: The coverage audit MUST identify missing coverage (Juju capabilities with no test or only MINIMAL/NONE coverage) and propose new test groups to fill gaps.
- **FR-053**: A quality audit MUST be performed for each suite evaluating: charm sterility (calibration vs third-party), substrate verification (present/absent), intra-suite dependency DAG (explicit/implicit), and whether the test validates Juju behavior or external charm behavior.
- **FR-054**: The quality audit MUST produce a per-suite migration plan: which charms to replace, which substrate verifications to add, and which intra-suite dependencies to declare.

#### CI Resource Cleanup

- **FR-033**: All CI-created resources (models, controllers) MUST use a naming convention with a `ci-<run-id>-` prefix to enable automated discovery of CI-owned resources.
- **FR-034**: A scheduled sweeper job MUST run on a recurring cadence (e.g., hourly) to discover and destroy CI-prefixed resources older than a configurable TTL (default: 4 hours).
- **FR-035**: The sweeper MUST be idempotent — running it multiple times must not produce errors or side effects when no stale resources exist.

#### Backward Compatibility & Reuse-First Principle

- **FR-025**: The existing `tests/main.sh` runner and suite structure (`tests/suites/<name>/task.sh`) MUST be preserved. The predicate system is an overlay, not a replacement.
- **FR-026**: A developer MUST still be able to run any suite locally using the existing `./main.sh <suite>` invocation, independent of predicates.
- **FR-027**: Existing suites MUST NOT require code changes to work with the predicate system — predicates are metadata about suites, not modifications to suites.
- **FR-055**: The audits (A1a, A1b) MUST default to **keep and enhance** existing tests rather than rewrite. A test that correctly validates Juju behavior MUST be preserved as-is, even if its style or structure differs from the ideal. Rewriting is justified ONLY when a test: (a) validates external charm behavior rather than Juju, (b) lacks substrate verification and cannot be augmented without restructuring, or (c) is fundamentally broken or redundant with another suite.
- **FR-056**: When migrating a suite to calibration charms (A2), the migration MUST be incremental — swap the charm reference and add substrate verification, but preserve the test logic, assertions, and flow. Do not rewrite working test logic as part of charm migration.
- **FR-057**: The coverage audit (A1a) verdict categories MUST be: **keep** (test is good), **enhance** (add substrate verification or DAG declaration to existing test), **migrate** (swap charm only), or **rewrite** (last resort, with documented justification). The default verdict MUST be keep or enhance.

### Key Entities

- **Test Tier**: A classification level (sanity, smoke, regression, integration) that determines when a test group is eligible to run based on the triggering event.
- **Test Predicate**: A declarative condition set (tier + provider + paths) attached to a test group that governs its activation.
- **Test Group**: A self-contained test suite (existing `tests/suites/<name>/` directory) that can be independently activated and executed.
- **Event Context**: The combination of trigger event type (PR, push, schedule, manual) and changeset (list of modified files) that is evaluated against predicates.
- **Tier Schedule**: The mapping from event types to which tiers are eligible, defining the "depth" of testing for each event.
- **Calibration Charm**: A purpose-built charm designed specifically for CI testing. Exercises Juju capabilities (relations, storage, config, actions) without external dependencies or complex operator logic. norma-k8s is the reference calibration charm for K8s.
- **Test Dependency Graph**: A DAG within a suite declaring prerequisite relationships between tests. Enables fail-fast: when a prerequisite fails, all dependent tests are skipped.
- **Substrate Verification**: The practice of confirming Juju's claimed outcome by inspecting the underlying infrastructure (K8s, LXD, cloud) directly, ensuring Juju didn't just report success while the substrate is in a different state.
- **Norma Charm Family**: The pair of calibration charms (norma-k8s for K8s, norma for machines) that serve as sterile test targets. Same action interface, different substrate lifecycle.
- **Suite Audit**: The process of evaluating each existing test suite against the three quality dimensions: sterility (charm choice), fail-fast (dependency DAG), and substrate verification.

## Preliminary Research: Test Tier Taxonomy for Juju

### Proposed Classification of All 50 Existing Suites

#### Sanity Tier (no runtime, fast feedback)

| Suite              | Provider | Rationale                                            |
|--------------------|----------|------------------------------------------------------|
| `static_analysis`  | N/A      | Linting, code quality — no Juju runtime needed       |

#### Smoke Tier (minimal e2e, per provider)

| Suite              | Provider | Rationale                                              |
|--------------------|----------|--------------------------------------------------------|
| `smoke`            | All      | Basic build + deploy verification                      |
| `smoke_k8s`        | K8s      | Basic K8s charm deployment                             |
| `smoke_k8s_psql`   | K8s      | Database charm on K8s (validates operator + storage)    |
| `deploy`           | All      | Core deployment workflows (charms, bundles, revisions)  |
| `cli`              | All      | CLI command verification (low cost, high signal)        |
| `charmhub`         | All      | Charm discovery and download (external dependency)      |

#### Regression Tier (feature-area verification)

| Suite              | Provider  | Rationale                                             |
|--------------------|-----------|-------------------------------------------------------|
| `deploy_caas`      | K8s       | K8s-specific deployment beyond smoke                   |
| `storage`          | IaaS      | IaaS storage pools and lifecycle                       |
| `storage_k8s`      | K8s       | K8s PV/PVC handling and race conditions                |
| `sidecar`          | K8s       | Pebble/sidecar charm lifecycle                         |
| `caasadmission`    | K8s       | K8s namespace isolation and webhooks                   |
| `secrets_iaas`     | IaaS      | Secret creation, rotation, CMR sharing                 |
| `secrets_k8s`      | K8s       | K8s secret lifecycle and concurrency                   |
| `relations`        | All       | Relation data exchange and lifecycle                   |
| `cmr`              | All       | Cross-model relation offer/consume                     |
| `resources`        | All       | Charm resource upload/download/upgrade                 |
| `hooks`            | All       | Hook dispatch and timing                               |
| `hooktools`        | All       | Hook tool availability during lifecycle                |
| `actions`          | All       | Action parameter passing and execution                 |
| `model`            | All       | Model config, migration, multi-model, status           |
| `controller`       | All       | Controller metrics, HA, tracing                        |
| `bootstrap`        | All       | Bootstrap with simplestreams                           |
| `agents`           | All       | Background agent operations                            |
| `refresh`          | All       | Charm refresh/upgrade, channel switching               |
| `constraints`      | IaaS      | Resource constraints (currently skips K8s)              |
| `authorized_keys`  | All       | SSH key management                                     |
| `credential`       | All       | Cloud credential lifecycle                             |
| `user`             | All       | User management and access control                     |
| `network`          | All       | Network health diagnostics                             |
| `machine`          | IaaS      | Machine operations and logging                         |
| `appdata`          | All       | Application data integration                           |
| `dashboard`        | All       | Dashboard deployment                                   |

#### Integration Tier (cross-feature, upgrade, provider-specific)

| Suite              | Provider  | Rationale                                             |
|--------------------|-----------|-------------------------------------------------------|
| `upgrade`          | All       | Version upgrade path validation                        |
| `controllercharm`  | K8s       | Controller charm Prometheus metrics                    |
| `coslite`          | K8s       | Full COS Lite bundle (multi-charm)                     |
| `kubeflow`         | K8s       | Full Kubeflow bundle (heavy, multi-charm)              |
| `ck`               | K8s       | Full Charmed Kubernetes bundle                         |
| `deploy_aks`       | AKS       | AKS-specific (currently disabled)                      |
| `spaces_ec2`       | EC2       | EC2 network spaces                                     |
| `spaces_gce`       | GCE       | GCE network spaces                                     |
| `cloud_azure`      | Azure     | Azure managed identity and storage                     |
| `cloud_gce`        | GCE       | GCE images, GPU, storage                               |
| `firewall`         | EC2       | EC2 security group rules                               |
| `ovs_maas`         | MAAS      | OVS netplan on MAAS                                    |
| `manual`           | Manual    | Manual provider deployment                             |
| `unmanaged`        | Unmanaged | Unmanaged provider                                     |
| `examples`         | All       | Example charm operations                               |

### Proposed New Test Groups

| Group                         | Tier       | Provider | What It Validates                                                | Gap Addressed                                    |
|-------------------------------|------------|----------|------------------------------------------------------------------|--------------------------------------------------|
| `constraints_k8s`             | Regression | K8s      | K8s-specific constraints (cpu, memory, deployment-type)          | Constraints suite explicitly skips K8s           |
| `deploy_caas_lifecycle`       | Regression | K8s      | Full CAAS app lifecycle: deploy, scale, config, action, remove   | `deploy_caas` only deploys and checks status     |
| `controller_lifecycle`        | Regression | All      | Bootstrap, enable-ha, disable-ha, controller config, teardown    | `controller` suite only tests metrics/HA/tracing |
| `deploy_caas_deployment_type` | Regression | K8s      | Deployment/DaemonSet/StatefulSet variants                        | No deployment-type variant testing exists         |
| `storage_k8s_deployment`      | Regression | K8s      | Storage behavior with Deployment type (no stable PVC naming)     | `storage_k8s` assumes StatefulSet PVC patterns   |

### Proposed GitHub Actions Workflow Architecture

The current architecture has 19 workflow files, 4 disabled, and inconsistent patterns. The proposed architecture preserves what works and replaces the scattered integration test workflows with a unified predicate-aware system.

**Keep unchanged**:
- `static-analysis.yml` — Always runs, no predicate needed
- `cla.yml` — PR administrative check
- `merge.yml` — Branch maintenance notification
- `docs-sphinx-python-dependency-build-checks.yml` — Scheduled maintenance
- `context-tests.yml` — Orchestrates non-integration checks (build, gen, docs, ddl, snap)

**Consolidate into a single predicate-aware workflow** (`integration-tests.yml`):
- Replaces: `smoke.yml`, `postgresql-k8s.yml`, `upgrade.yml`, `microk8s-tests.yml`
- Reads predicate metadata, evaluates against event context and changeset, generates a matrix of (suite, provider) pairs, dispatches parallel jobs
- Event mapping: PR → smoke tier; push-to-main → smoke + regression; schedule → all; manual → configurable

**Retire**:
- `terraform-smoke.yml` — Currently disabled (`if: false`)
- `migrate.yml` — Currently disabled (`if: false`)
- `jaas-smoke.yml` — Evaluate whether to integrate into predicate system or keep standalone (external dependency on JIMM)

### Calibration Charm Requirements Contract

CI tests must use purpose-built calibration charms instead of third-party charms. Two charms form the calibration family: **norma-k8s** (K8s/CAAS) and **norma** (machines/IaaS). Development of these charms is **out of scope** for this spec — they are maintained in separate repositories. This section defines the **contract**: what capabilities the CI test suite expects from each charm. These requirements must be fed back into the respective charm specs.

#### Shared Contract (Both Charms)

Both calibration charms MUST provide:

| ID | Capability | Required Action/Interface | What CI Tests Need It For |
|----|-----------|--------------------------|--------------------------|
| CC-01 | Event lifecycle logging | `get-event-log` action → returns ordered event ledger | Verify Juju dispatches correct events in correct order |
| CC-02 | Configuration (all types) | `get-config` action + string, int, float, bool, secret config options | Test `juju config`, config-changed event, type coercion |
| CC-03 | Status reporting | `set-status` action + all 4 status types (active, blocked, waiting, maintenance) | Test `juju status` accuracy and status transitions |
| CC-04 | Actions with params | `run-check <capability>` action → structured pass/fail with evidence | Per-capability validation from inside the charm |
| CC-05 | Peer relations | `norma-peers` peer endpoint + `get-peer-data` action | Test leader election, peer data exchange, unit coordination |
| CC-06 | Provides/requires relations | `calibration-provider` + `calibration-requirer` endpoints | Test relation lifecycle, data exchange, self-relation |
| CC-07 | Cross-model relations | Same endpoints work via `juju offer`/`juju consume` | Test CMR offer, consume, data flow across models |
| CC-08 | Scaling | `get-cluster-info` action → unit count, leader, peer list | Test `juju scale-application`, add-unit, remove-unit |
| CC-09 | Secrets | `get-secret-info` action + secret create/rotate/expire/share | Test full secret lifecycle |
| CC-10 | Storage (filesystem) | `check-storage` action → persistence marker verification | Test storage attach, detach, persistence across restarts |
| CC-11 | Upgrade/refresh | `get-version` action + upgrade-charm handling | Test `juju refresh`, channel switching |
| CC-12 | Action error handling | `fail-action` action → intentional failure with message | Test action failure reporting in `juju run` |

#### norma-k8s Specific Contract (K8s/CAAS Only)

| ID | Capability | Required Action/Interface | What CI Tests Need It For |
|----|-----------|--------------------------|--------------------------|
| CC-K1 | Pebble workload management | Service start/stop/restart via Pebble layers | Test K8s workload lifecycle |
| CC-K2 | Pebble health checks | `toggle-health` action + HTTP/TCP/exec checks | Test pebble-check-failed/recovered events |
| CC-K3 | Pebble file/exec ops | `test-pebble-ops` action → file push/pull/exec | Test container file operations |
| CC-K4 | Pebble custom notices | `trigger-notice` action → workload-to-charm notification | Test async workload communication |
| CC-K5 | Multiple containers | 2+ containers with independent lifecycle | Test multi-container charms |
| CC-K6 | OCI resources | Image resource refresh via `juju attach-resource` | Test OCI resource lifecycle |
| CC-K7 | Non-root execution | Runs as non-root UID in container | Test security constraints |
| CC-K8 | Port management | `test-networking` action + open-port/close-port | Test K8s service/port exposure |
| CC-K9 | COS observability | prometheus_scrape, grafana_dashboard, loki_push_api endpoints | Test COS integration (optional, integration tier only) |
| CC-K10 | Multiple storages | 2+ named storage definitions (e.g., data + logs) | Test multiple PVC management |
| CC-K11 | Event deferral | `test-defer` action → arm/verify deferral | Test event.defer() and re-emission |

#### norma Specific Contract (Machines/IaaS Only)

| ID | Capability | Required Action/Interface | What CI Tests Need It For |
|----|-----------|--------------------------|--------------------------|
| CC-M1 | Systemd service management | Go binary managed as systemd unit | Test machine workload lifecycle (no Pebble) |
| CC-M2 | SSH accessibility | Charm reachable via `juju ssh` | Test SSH access, authorized-keys |
| CC-M3 | Machine constraints | Deploy with cores/mem/root-disk/virt-type and verify | Test machine placement, constraint enforcement |
| CC-M4 | LXD container placement | Deploy to `lxd:N` and report container status | Test container placement on machines |
| CC-M5 | Subordinate interface | Provides subordinate endpoint for injection testing | Test subordinate deployment pattern |
| CC-M6 | Spaces & network bindings | `test-networking` action reports `network-get` bindings | Test network spaces, endpoint bindings |
| CC-M7 | Block storage | `check-storage` action for block devices via `lsblk` | Test IaaS block storage (not just filesystem) |
| CC-M8 | Base/series verification | Reports OS version via action | Test `--base ubuntu@22.04` enforcement |
| CC-M9 | Agent tools verification | Reports jujud version and tools path | Test agent upgrade path, tools distribution |
| CC-M10 | Payload registration | Register payload, verify via `juju list-payloads` | Test payload lifecycle tracking |
| CC-M11 | Subordinate variant (norma-sub) | Separate subordinate charm for injection tests | Test subordinate relation + principal interaction |

#### Substrate Verification Contract

Both charms MUST support substrate-level verification via their actions. The test suite will call charm actions AND independently verify on the substrate:

| Juju Operation | norma-k8s Substrate Check | norma Machine Substrate Check |
|---------------|--------------------------|------------------------------|
| `juju deploy` | `kubectl get pod -l app.kubernetes.io/name=norma-k8s` exists | `juju ssh <unit> systemctl is-active norma` returns active |
| `juju scale-application -n 3` | `kubectl get pods` count = 3 | `juju status --format=json` machine count = 3 |
| `juju destroy-model` | `kubectl get namespace <model>` returns NotFound | LXD containers / cloud instances terminated |
| `juju config app key=val` | Charm action `get-config` returns new value | Same |
| `juju relate a b` | Charm action `get-relation-data` shows data | Same |
| `juju remove-relation a b` | Charm action `get-relation-data` shows empty/absent | Same |
| `juju add-storage` | `kubectl get pvc` shows new PVC | `lsblk` or `df -h` shows new mount |
| `juju run app/0 get-secret-info` | Secret content accessible | Same |

#### Design Principles (Contract to Charm Specs)

1. **Same action names across charms** — `get-event-log`, `get-config`, `check-storage`, `run-check`, etc. — so test scripts can be reused across providers with minimal branching
2. **Structured JSON results** — All actions return JSON with consistent schema so test scripts can parse programmatically
3. **Self-diagnosing** — Charm bugs must be distinguishable from Juju bugs. Actions should include a self-test that validates the charm's own state before reporting on Juju behavior
4. **No external dependencies** — Calibration charms must not depend on external services (databases, APIs, registries) at runtime. The Go workload binary is self-contained
5. **Fast startup** — Active/idle within 120 seconds of deploy. CI time is expensive

### Execution Approach: Two Parallel Tracks

The work proceeds on two parallel tracks. **Track A (test content) has priority** — great tests matter more than great structure. Track B runs alongside but does not block Track A. Calibration charm development (norma-k8s, norma) is **out of scope** — this spec defines the contract they must fulfill, and that contract is fed into their respective specs.

#### Track A — Test Content Quality (Priority)

Focus: Make sure every test is worth running. **Reuse-first**: keep existing tests that work, enhance them where needed, rewrite only as a last resort.

1. **A1a — Coverage audit** — For each of the 50 suites, evaluate WHAT is tested:
   - Map every suite to the Juju capabilities it exercises
   - Identify over-testing: suites that redundantly test the same capability
   - Identify missing coverage: Juju capabilities with no test or only partial coverage
   - Verdict per suite: **keep** (good as-is), **enhance** (add substrate verification/DAG), **migrate** (swap charm only), or **rewrite** (last resort, justified). Default is keep or enhance.
   - Produce a capability-to-suite coverage matrix

2. **A1b — Quality audit** — For each of the 50 suites, evaluate HOW it's tested:
   - Does it use calibration charms or third-party? Mark for migration.
   - Does it verify outcomes at the substrate level? Mark gaps.
   - Does it have intra-suite dependencies declared? Map the implicit DAG.
   - Is the test actually testing Juju behavior, or just that a charm works?
   - Produce per-suite migration plan (charm swap, substrate checks to add, DAG to declare)

3. **A2 — Migrate suites to calibration charms** — Starting with smoke and regression tiers (blocked on norma charms meeting the contract):
   - `smoke_k8s` → use norma-k8s instead of juju-qa-test
   - `smoke_k8s_psql` → replace with norma-k8s (validates operator+storage without PostgreSQL dependency)
   - `deploy` → use norma for machine deployment tests
   - `deploy_caas` → use norma-k8s
   - `storage_k8s` → use norma-k8s storage capabilities
   - `relations` → use norma self-relation + norma-k8s cross-relation

4. **A3 — Add substrate verification** — For each migrated suite, add verification that checks the substrate after every mutating Juju operation.

5. **A4 — Implement fail-fast DAG** — Add dependency declarations to each suite so that setup failures skip dependent tests instantly.

6. **A5 — Write new tests for coverage gaps** — Based on A1a's missing coverage findings, create new test groups using calibration charms.

#### Track B — CI Structure (Parallel)

Focus: Make the CI system smart about when to run which tests.

1. **B1 — Define predicate schema** — `predicates.yaml` format and evaluation logic.
2. **B2 — Classify all 50 suites** — Tier, provider, paths for each (uses A1a/A1b audit data).
3. **B3 — Build unified workflow** — `integration-tests.yml` with matrix expansion.
4. **B4 — Add operational tooling** — Resource sweeper (`ci-<run-id>-` naming + scheduled cleanup).

#### External Dependencies (Out of Scope, Contract-Driven)

- **norma-k8s**: Must meet CC-01 through CC-12 + CC-K1 through CC-K11 before A2 can migrate K8s suites
- **norma (machines)**: Must meet CC-01 through CC-12 + CC-M1 through CC-M11 before A2 can migrate IaaS suites
- Charm development progress does not block A1a, A1b, A4, B1, B2, B3 — those phases work with existing charms or produce plans

#### Interaction Between Tracks

- A1a (coverage audit) tells us what new tests to write (A5) and feeds classification data to B2
- A1b (quality audit) tells us what to fix in existing tests and produces the migration plan for A2
- Track A produces better tests → Track B ensures they run at the right time
- Track A's dependency DAG (A4) integrates into Track B's `predicates.yaml` (dependency field)
- A1a and A1b, B1 all start day 1 — no blocking dependencies

### Future Enhancements (Out of Initial Scope)

- **FE-001**: Replace curated glob patterns with Go Transitive Analysis (GTA) for automatic dependency-graph-aware path filtering. GTA analyzes the Go import graph to determine which packages are transitively affected by a changeset, reducing both over-triggering (running unrelated suites) and under-triggering (missing suites affected via transitive imports). The predicate YAML schema (`paths` field) is designed to be forward-compatible — GTA would augment or replace the glob list without schema changes.
- **FE-002**: Flaky test management — auto-quarantine tests failing >5% over a 7-day rolling window, 30-day fix-or-delete SLO, quarantine visibility dashboard. Requires deciding on a tracking datastore (git-tracked JSON, GitHub artifacts, or external DB). See deferred FR-028 through FR-032.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All 50 existing test suites are classified with tier, provider, and path predicates — 100% coverage of existing inventory.
- **SC-002**: PRs that touch only a single domain area (e.g., `domain/secret/`) trigger no more than 10 test groups (reduced from the current "all or nothing" approach), while PRs that touch foundational code (`core/`, `go.mod`) trigger the full set for the active tier.
- **SC-003**: PR integration test feedback (smoke tier) completes within 15 minutes for path-filtered runs, compared to the current 60-minute timeout.
- **SC-004**: Push-to-main runs include at least 20 test groups (regression tier), up from the current 4 automated suites, without exceeding 90 minutes total wall-clock time (via parallelism).
- **SC-005**: Nightly full builds execute all 50+ test groups across all available providers, providing complete regression coverage on a daily cadence.
- **SC-006**: Adding a new test suite to the system requires only creating the suite directory and a predicate metadata file — zero GitHub Actions workflow file changes.
- **SC-007**: The predicate metadata format is self-documenting: a new contributor can read one suite's metadata and understand when and why it runs.
- **SC-008**: No regression in existing automated coverage — everything that runs today on PRs continues to run with identical or better reliability.
- **SC-009**: All smoke and regression tier suites use calibration charms (norma-k8s or norma) — zero third-party charm dependencies in these tiers.
- **SC-010**: Every test that performs a mutating Juju operation includes substrate-level verification of the outcome.
- **SC-011**: Every test suite declares intra-suite dependencies; a bootstrap failure skips all dependent tests within 30 seconds (not after attempting and failing each one).
- **SC-012**: The calibration charm contract (CC-01 through CC-M11) is documented and fed into the norma-k8s and norma charm specs as external requirements.
- **SC-013**: The coverage audit identifies all Juju capabilities with NONE/MINIMAL test coverage and proposes new test groups for each gap.
- **SC-014**: The quality audit produces a migration plan for every smoke and regression tier suite.

### Assumptions

- Self-hosted GitHub Actions runners (quad-xlarge, xxlarge, etc.) remain available with the same capacity. The predicate system schedules more work but relies on parallelism rather than serial execution.
- MicroK8s remains the K8s provider for CI. The predicate system uses "k8s" as the provider label, which maps to MicroK8s in GitHub Actions.
- The existing bash test framework (`tests/main.sh`, `tests/includes/`, `tests/suites/`) is stable and does not require architectural changes. Predicates are an external metadata layer.
- Suites currently not run in GitHub Actions (but present in `tests/suites/`) are functional and can be activated by the predicate system without code fixes. If some are broken, they will be discovered during initial activation and fixed as part of the rollout.
- Path-based filtering uses the git diff changeset between the PR branch and its base. The mechanism for obtaining this changeset is already proven in `context-tests.yml` via `dorny/paths-filter`.
- Calibration charms (norma-k8s, norma) are developed in separate repositories outside the scope of this spec. This spec defines the contract they must meet (CC-01 through CC-M11). Suite migration (A2) is blocked until the respective charm meets its contract.
- Both norma charms are maintained in separate repositories and versioned independently of Juju. CI references them by charm revision or local build.
- The audit phases (A1a, A1b) and structural phases (B1–B4) are NOT blocked on charm readiness — they work with existing charms and produce plans that activate when charms are ready.
