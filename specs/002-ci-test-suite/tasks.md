# Tasks: Predicate-Based CI Test Suite

**Input**: Design documents from `/specs/002-ci-test-suite/`
**Prerequisites**: plan.md (complete), spec.md (complete), research.md, data-model.md, contracts/

**Tests**: Tests are NOT explicitly requested. Test tasks are omitted. The work itself IS the test infrastructure.

**Organization**: Tasks follow two parallel tracks (A: test content, B: CI structure) mapped to user stories. The governing principle requires every task to produce a deployable improvement.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US7)
- Include exact file paths in descriptions

## Track-to-Phase Mapping

| Plan Track | Phase | Focus |
|------------|-------|-------|
| — | Phase 1: Setup | Tooling + stubs |
| B4 | Phase 2: Resource Sweeper | Independent operational tooling |
| A1a + A1b | Phase 3: Audits | Coverage + quality analysis of all 50 suites |
| A3 | Phase 4: Substrate Verification | `substrate.sh` helper library |
| A4 | Phase 5: Fail-Fast DAG | DAG-aware test runner |
| B1 | Phase 6: Predicate Evaluator | `evaluate-predicates.sh` |
| B2 | Phase 7: Classification | `predicates.yaml` for all 50 suites |
| B3 | Phase 8: Workflow | `integration-tests.yml` |
| A5 | Phase 9: New Tests | Fill coverage gaps |
| A2 | Phase 10: Charm Migration | Swap third-party → calibration charms |
| — | Phase 11: Polish | Documentation + final validation |

---

## Phase 1: Setup

**Purpose**: Establish the predicate schema documentation, validate tooling prerequisites, and create helper file stubs.

- [ ] T001 Validate `mikefarah/yq` v4 availability on CI runners (`yq --version` returns `mikefarah/yq` v4.x) and document version requirement in `specs/002-ci-test-suite/quickstart.md`
- [ ] T002 Create `tests/includes/substrate.sh` with empty function stubs and source guard in `tests/includes/juju.sh`
- [ ] T003 [P] Create `tests/includes/predicates.sh` with empty function stubs for predicate evaluation helpers

**Checkpoint**: Tooling confirmed, helper files exist (empty stubs), no behavior changes to existing CI.

---

## Phase 2: Resource Sweeper (B4 — Independent, No Dependencies)

**Purpose**: Automated cleanup of stale CI resources. Independent of all other work — ships immediately.

**Independent Test**: Run `ci-sweeper.sh` when no stale resources exist — verify it exits cleanly with no errors (idempotent).

- [ ] T004 [P] Create `tests/ci-sweeper.sh` — list all Juju controllers matching `ci-*` pattern, destroy those older than configurable TTL (default 4h), sweep orphaned LXD containers and MicroK8s namespaces matching `ci-*`. Idempotent (FR-035)
- [ ] T005 [P] Create `.github/workflows/ci-sweeper.yml` — scheduled hourly, runs `tests/ci-sweeper.sh` with 4h TTL, plus manual dispatch for emergency cleanup
- [ ] T006 Update `tests/includes/juju.sh` `bootstrap()` function — prefix controller names with `ci-${GITHUB_RUN_ID:-local}-` when `CI=true` env var is set. Backward-compatible: local runs keep current naming (FR-033)

**Checkpoint**: Resource sweeper deployed. CI-created resources have naming convention and automated cleanup.

---

## Phase 3: Foundational — Audits (A1a + A1b)

**Purpose**: Understand the current state of all 50 suites before making any changes. MUST complete before meaningful implementation in any user story.

**Why blocking**: Every subsequent phase (substrate verification, DAG, classification, new tests) depends on knowing what each suite does, what charms it uses, and what its implicit dependencies are.

- [ ] T007 [US7] Read and catalog all 50 `tests/suites/*/task.sh` files — for each suite, list: test functions, charms deployed, provider guards, bootstrap/destroy lifecycle, and Juju capabilities exercised. Output to `specs/002-ci-test-suite/coverage-analysis.md` (update existing v2)
- [ ] T008 [US7] Cross-reference suite catalog (T007) against the 175-capability inventory in `specs/002-ci-test-suite/coverage-analysis.md` — identify NONE/MINIMAL gaps and over-testing (multiple suites testing same capability identically)
- [ ] T009 [US7] For each suite, score on three quality dimensions in `specs/002-ci-test-suite/coverage-analysis.md`: sterility (0-2: calibration/mixed/third-party), substrate verification (0-2: present/partial/absent), fail-fast readiness (0-2: DAG declared/implicit ordering/no ordering). Also document per FR-045 where substrate verification is impractical and the fallback verification strategy
- [ ] T010 [US7] Assign per-suite verdict (keep/enhance/migrate/rewrite) in `specs/002-ci-test-suite/coverage-analysis.md` — default is keep or enhance per FR-057. Document justification for any migrate or rewrite verdict
- [ ] T011 [US7] For each suite, map the implicit test dependency DAG — which test functions depend on bootstrap, which depend on prior deployments. Output as proposed `tests` section per suite in `specs/002-ci-test-suite/coverage-analysis.md`
- [ ] T012 [US7] Produce per-suite migration plan (for suites verdicted as migrate): which charm to swap, what substrate checks to add, what DAG to declare. Append to `specs/002-ci-test-suite/coverage-analysis.md`
- [ ] T013 [P] [US7] Feed calibration charm contract (CC-01 through CC-M11) from `specs/002-ci-test-suite/contracts/charm-contract.yaml` (source of truth) into norma-k8s and norma charm specs as external requirements. Document handoff in `specs/002-ci-test-suite/coverage-analysis.md`

**Checkpoint**: Complete understanding of all 50 suites. Verdicts, DAG maps, migration plans, and coverage gaps documented. All subsequent phases have their input data.

---

## Phase 4: US7 — Substrate Verification Helpers (Priority: P1)

**Goal**: Every mutating test can verify its outcome on the substrate. Ships as a shared helper library — existing tests can adopt incrementally.

**Independent Test**: Source `substrate.sh` in any suite, call `substrate_check_namespace_gone` after `juju destroy-model`, verify it works on MicroK8s.

- [ ] T014 [US7] Implement K8s substrate verification functions in `tests/includes/substrate.sh`: `substrate_check_pod_exists`, `substrate_check_pod_count`, `substrate_check_namespace_exists`, `substrate_check_namespace_gone`, `substrate_check_pvc_exists`, `substrate_check_pvc_count`, `substrate_check_service_exists`
- [ ] T015 [P] [US7] Implement LXD substrate verification functions in `tests/includes/substrate.sh`: `substrate_check_container_exists`, `substrate_check_container_gone`, `substrate_check_container_count`
- [ ] T016 [US7] Implement provider-aware dispatch functions in `tests/includes/substrate.sh`: `substrate_verify_deploy`, `substrate_verify_destroy_model`, `substrate_verify_scale` — detect `BOOTSTRAP_PROVIDER` and call appropriate K8s/LXD function
- [ ] T017 [US7] Add configurable timeout (default 60s) and clear error messages to all substrate functions in `tests/includes/substrate.sh` — format: "substrate verification: expected <X> but found <Y> after <T>s"
- [ ] T018 [US7] Add substrate verification calls to `tests/suites/smoke/deploy.sh` after `juju deploy` and `juju destroy-model` — validate the helper works in a real suite without changing test logic
- [ ] T019 [P] [US7] Add substrate verification calls to `tests/suites/smoke_k8s/task.sh` after K8s deploy — validate K8s-specific helpers work

**Checkpoint**: `substrate.sh` is a working library. Two smoke suites enhanced with substrate verification. Can be merged independently.

---

## Phase 5: US7 — Fail-Fast DAG (Priority: P1)

**Goal**: When bootstrap fails, dependent tests skip instantly instead of running and failing one by one.

**Independent Test**: Run a suite where bootstrap is intentionally broken — verify dependent tests report "skipped: dependency failed" and suite completes in bootstrap-attempt time, not N*test time.

- [ ] T020 [US7] Extend `tests/includes/run.sh` with DAG-aware functions: `load_test_dag` (parse `predicates.yaml` `tests` section via `mikefarah/yq` v4), `check_dependencies` (check if all `depends_on` tests passed), `record_test_result` (write to `$TEST_DIR/test-results.tmp`)
- [ ] T021 [US7] Implement topological sort in `tests/includes/run.sh` for test execution ordering — use bash associative arrays, handle DAG cycles with error message
- [ ] T022 [US7] Extend `skip()` function in `tests/includes/run.sh` to check dependency results before each test — if any dependency failed, print "SKIPPED <test>: dependency '<dep>' failed" and return without executing
- [ ] T023 [US7] Add `--fail-fast` flag to `tests/main.sh` — when set, stop suite execution on first non-prerequisite test failure (prerequisites always run to establish the dependency chain)
- [ ] T024 [US7] Print DAG summary at suite start when `predicates.yaml` has a `tests` section — format: "Test execution order: setup → deploy → {test_a, test_b} → test_c"
- [ ] T025 [US7] Make DAG opt-in: suites without `tests` section in `predicates.yaml` run in existing sequential order from `task.sh` — zero behavior change for unclassified suites
- [ ] T026 [US7] Add `tests` DAG section to `tests/suites/smoke/predicates.yaml` (create file if not exists) mapping `test_build` and `test_deploy` with bootstrap as prerequisite — validate DAG works on a real suite

**Checkpoint**: `run.sh` supports DAG-aware execution. Smoke suite has a working DAG. Suites without predicates.yaml are unaffected.

---

## Phase 6: US1 + US2 + US3 — Predicate Evaluation System (Priority: P1/P2)

**Goal**: A script reads all `predicates.yaml` files, evaluates against event context and changeset, and outputs a JSON matrix of activated suites for GitHub Actions.

**Independent Test**: Run `evaluate-predicates.sh --event-type pull_request --changed-files <(echo "domain/secret/service.go") --provider all` and verify only secrets-related suites appear in output.

- [ ] T027 [US1] Implement core predicate evaluator in `tests/evaluate-predicates.sh`: argument parsing for `--event-type`, `--changed-files`, `--provider`, `--tier-override`
- [ ] T028 [US2] Implement tier eligibility logic in `tests/evaluate-predicates.sh`: map event types to eligible tiers per FR-010 (PR → sanity+smoke required + regression informational; push → sanity+smoke+regression; schedule → all; dispatch → configurable)
- [ ] T029 [US1] Implement path matching logic in `tests/evaluate-predicates.sh`: for each suite's `predicates.yaml`, check if any glob pattern matches any changed file. Handle wildcards: `core/`, `go.mod`, `tests/includes/` → all suites eligible (FR-007)
- [ ] T030 [US3] Implement predicate composition (AND logic) in `tests/evaluate-predicates.sh`: suite activates only if tier eligible AND provider matches AND paths match (FR-008)
- [ ] T031 [US1] Implement default handling in `tests/evaluate-predicates.sh`: suites without `predicates.yaml` default to tier=integration, provider=all, paths=wildcard (FR-009)
- [ ] T032 [US2] Implement JSON matrix output in `tests/evaluate-predicates.sh`: format as `{"include": [{"suite": "...", "provider": "...", "tier": "...", "required": true/false}]}` compatible with GitHub Actions `strategy.matrix`
- [ ] T033 [US3] Implement `predicates.yaml` schema validation in `tests/evaluate-predicates.sh`: warn on missing required fields, error on invalid tier/provider values
- [ ] T034 [P] [US1] Write evaluator unit tests as bash test cases in `tests/test-evaluate-predicates.sh`: test path matching, tier eligibility, default handling, wildcard paths, predicate composition — at least 10 test cases covering spec acceptance scenarios

**Checkpoint**: `evaluate-predicates.sh` is a working, tested script. Can be run locally to preview what suites would activate for any changeset.

---

## Phase 7: US4 — Classify All 50 Existing Suites (Priority: P2)

**Goal**: Every existing suite has a `predicates.yaml` with tier, provider, paths, charms, and rationale.

**Independent Test**: Run `evaluate-predicates.sh` against sample changesets and verify: (a) every suite has metadata, (b) `domain/secret/` change triggers only secrets suites, (c) `core/` change triggers all suites for the active tier.

- [ ] T035 [P] [US4] Create `predicates.yaml` for all 6 smoke-tier suites: `tests/suites/smoke/predicates.yaml`, `tests/suites/smoke_k8s/predicates.yaml`, `tests/suites/smoke_k8s_psql/predicates.yaml`, `tests/suites/deploy/predicates.yaml`, `tests/suites/cli/predicates.yaml`, `tests/suites/charmhub/predicates.yaml` — using tier, provider, and path data from audit (T007-T010)
- [ ] T036 [P] [US4] Create `predicates.yaml` for first batch of 10 regression-tier suites: `deploy_caas`, `storage`, `storage_k8s`, `sidecar`, `caasadmission`, `secrets_iaas`, `secrets_k8s`, `relations`, `cmr`, `resources` — using audit data
- [ ] T037 [P] [US4] Create `predicates.yaml` for second batch of 15 regression-tier suites: `hooks`, `hooktools`, `actions`, `model`, `controller`, `bootstrap`, `agents`, `refresh`, `constraints`, `authorized_keys`, `credential`, `user`, `network`, `machine`, `appdata`, `dashboard` — using audit data
- [ ] T038 [P] [US4] Create `predicates.yaml` for the sanity-tier suite: `tests/suites/static_analysis/predicates.yaml`
- [ ] T039 [P] [US4] Create `predicates.yaml` for all 15 integration-tier suites: `upgrade`, `controllercharm`, `coslite`, `kubeflow`, `ck`, `deploy_aks`, `spaces_ec2`, `spaces_gce`, `cloud_azure`, `cloud_gce`, `firewall`, `ovs_maas`, `manual`, `unmanaged`, `examples` — using audit data
- [ ] T040 [US4] Validate all 50 `predicates.yaml` files: run `evaluate-predicates.sh` against 5 sample changesets and verify expected activation:
  - `domain/secret/service.go` → `secrets_iaas`, `secrets_k8s` only (+ smoke if wildcard)
  - `core/network/address.go` → all suites for the active tier (wildcard)
  - `caas/broker.go` → all K8s suites (`deploy_caas`, `storage_k8s`, `sidecar`, etc.)
  - `cmd/juju/application/deploy.go` → `cli`, `deploy`, `deploy_caas`, `charmhub`
  - `docs/README.md` → no integration suites activated

**Checkpoint**: All 50 suites classified. Evaluator correctly selects suites based on changesets. Can be merged — predicates.yaml files are inert until the workflow reads them.

---

## Phase 8: US5 — GitHub Actions Workflow Consolidation (Priority: P3)

**Goal**: A single `integration-tests.yml` replaces scattered test workflows, dispatching matching suites via matrix.

**Independent Test**: Open a PR, verify `integration-tests.yml` triggers the same suites as `smoke.yml` plus additional regression suites (informational). Verify a push to main triggers regression as blocking.

- [ ] T041 [US5] Create `.github/workflows/integration-tests.yml` with evaluate job: checkout, get changed files (via `dorny/paths-filter` or git diff), run `tests/evaluate-predicates.sh`, output matrix as job output
- [ ] T042 [US5] Add test matrix job to `.github/workflows/integration-tests.yml`: spawn parallel jobs from evaluator output, each runs `tests/main.sh -p <provider> --fail-fast <suite>`, configure `fail-fast: false` on matrix strategy
- [ ] T043 [US5] Configure blocking vs informational in `.github/workflows/integration-tests.yml`: regression-tier jobs on PRs use `continue-on-error: true` (non-blocking); all tiers on push-to-main are blocking
- [ ] T044 [US5] Add a post-test summary step to `.github/workflows/integration-tests.yml` that generates a PR comment or check annotation listing informational (regression-tier) results — ensures developers see actual pass/fail status even when the check is green (FR-024)
- [ ] T045 [US5] Add workflow_dispatch trigger to `.github/workflows/integration-tests.yml` with tier override input (dropdown: smoke, regression, all) per FR-011
- [ ] T046 [US5] Add schedule trigger (nightly) to `.github/workflows/integration-tests.yml` with all tiers eligible
- [ ] T047 [US5] Configure runner labels in `.github/workflows/integration-tests.yml`: LXD suites on quad-xlarge, K8s suites on xxlarge (matching current `smoke.yml` runner config)
- [ ] T048 [US5] Run `integration-tests.yml` alongside `smoke.yml` for validation — both trigger on same events, compare results over 2-week transition period
- [ ] T049 [US5] After transition validation: remove `smoke.yml`, `postgresql-k8s.yml`, `upgrade.yml`, `microk8s-tests.yml` from `.github/workflows/` and add `terraform-smoke.yml`, `migrate.yml` to retired list

**Checkpoint**: Unified workflow running in production. Old scattered workflows removed. Adding a new suite requires only `predicates.yaml` — zero workflow changes.

---

## Phase 9: US6 — New Test Coverage for Gaps (Priority: P3)

**Goal**: Fill identified coverage gaps with new test suites using calibration charms.

**Independent Test**: Deploy `constraints_k8s` suite locally via `./main.sh -p microk8s constraints_k8s`, verify it tests K8s-specific constraints and has working `predicates.yaml`.

- [ ] T050 [P] [US6] Create `tests/suites/constraints_k8s/task.sh` and `tests/suites/constraints_k8s/predicates.yaml` — test K8s-specific constraints (cpu, memory, deployment-type) using norma-k8s charm. Tier: regression, provider: k8s, paths: `domain/constraint/**`, `internal/provider/kubernetes/**`
- [ ] T051 [P] [US6] Create `tests/suites/deploy_caas_lifecycle/task.sh` and `tests/suites/deploy_caas_lifecycle/predicates.yaml` — full CAAS lifecycle (deploy, scale, config, action, remove) using norma-k8s. Tier: regression, provider: k8s
- [ ] T052 [P] [US6] Create `tests/suites/controller_lifecycle/task.sh` and `tests/suites/controller_lifecycle/predicates.yaml` — controller bootstrap, enable-ha, disable-ha, controller config, teardown. Tier: regression, provider: all
- [ ] T053 [P] [US6] Create `tests/suites/deploy_caas_deployment_type/task.sh` and `tests/suites/deploy_caas_deployment_type/predicates.yaml` — Deployment/DaemonSet/StatefulSet variants using norma-k8s. Tier: regression, provider: k8s
- [ ] T054 [P] [US6] Create `tests/suites/storage_k8s_deployment/task.sh` and `tests/suites/storage_k8s_deployment/predicates.yaml` — storage behavior with Deployment type (no stable PVC naming). Tier: regression, provider: k8s
- [ ] T055 [US6] Create additional test suites for gaps identified by coverage audit (T008) — exact suites TBD from audit results, each with `task.sh` + `predicates.yaml`

**Checkpoint**: Coverage gaps filled. New suites integrated into predicate system and activated on relevant PRs.

---

## Phase 10: US7 — Charm Migration (Priority: P1, Blocked on External)

**Goal**: Replace third-party charms with norma calibration charms in smoke and regression tiers.

**Independent Test**: Run migrated suite (e.g., `smoke_k8s`) — verify it uses norma-k8s, passes all existing assertions, and includes substrate verification.

**External dependency**: Blocked on norma-k8s meeting contract CC-01 through CC-K11, and norma meeting CC-01 through CC-M11 (source of truth: `specs/002-ci-test-suite/contracts/charm-contract.yaml`). Start when charms are ready.

- [ ] T056 [US7] Migrate `tests/suites/smoke_k8s/` to use norma-k8s instead of juju-qa-test — swap charm reference, update wait conditions, preserve test logic and assertions
- [ ] T057 [P] [US7] Migrate `tests/suites/smoke_k8s_psql/` to use norma-k8s instead of postgresql-k8s — swap charm, add storage verification via norma actions, preserve test flow
- [ ] T058 [P] [US7] Migrate `tests/suites/deploy_caas/` to use norma-k8s — swap charm, preserve deployment test logic
- [ ] T059 [P] [US7] Migrate `tests/suites/storage_k8s/` to use norma-k8s storage capabilities — swap charm, add `check-storage` action calls
- [ ] T060 [P] [US7] Migrate `tests/suites/relations/` to use norma self-relation (`calibration-provider` ↔ `calibration-requirer`) — swap charm, preserve relation test logic
- [ ] T061 [US7] Migrate `tests/suites/deploy/` to use norma (machine charm) — swap ubuntu/juju-qa-test references, preserve deployment test logic
- [ ] T062 [US7] Update `charms` section in all migrated suites' `predicates.yaml` to reflect `type: calibration`
- [ ] T063 [US7] Add substrate verification (from `substrate.sh`) to all migrated suites — call `substrate_verify_deploy`, `substrate_verify_destroy_model`, `substrate_verify_scale` after corresponding Juju operations
- [ ] T064 [US7] Validate SC-009: run `yq '.charms[].type' tests/suites/*/predicates.yaml` for all smoke and regression tier suites and verify only `calibration` appears — zero third-party charm dependencies in these tiers

**Checkpoint**: Smoke and regression tier suites use calibration charms. Third-party charm dependencies removed from these tiers. Substrate verification active.

---

## Phase 11: Polish & Cross-Cutting Concerns

**Purpose**: Workflow retirement, documentation, final end-to-end validation.

- [ ] T065 [P] Retire disabled workflows: remove `terraform-smoke.yml` and `migrate.yml` from `.github/workflows/` (both currently have `if: false`)
- [ ] T066 [P] Evaluate `jaas-smoke.yml` — determine if it should be integrated into predicate system or kept standalone (external JIMM dependency). Document decision in `specs/002-ci-test-suite/plan.md`
- [ ] T067 Validate end-to-end: open a test PR touching `domain/secret/`, verify only secrets-related suites activate, regression tier is informational, smoke tier is blocking
- [ ] T068 Update `specs/002-ci-test-suite/quickstart.md` with lessons learned from implementation — add troubleshooting section, update examples with real suite names

**Checkpoint**: CI infrastructure complete. Resource sweeper running. All documentation up to date.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies — start immediately
- **Phase 2 (Resource Sweeper)**: No dependencies — start immediately, parallel with Phase 1
- **Phase 3 (Audits)**: No dependencies — start immediately, parallel with Phase 1+2
- **Phase 4 (Substrate)**: Depends on T002 (stub file from Phase 1)
- **Phase 5 (Fail-Fast DAG)**: Depends on Phase 3 (audit provides DAG mapping)
- **Phase 6 (Predicate Evaluator)**: No dependencies — can start Day 1 in parallel with audits
- **Phase 7 (Classification)**: Depends on Phase 3 (audit data for tier/path assignment) + Phase 6 (schema)
- **Phase 8 (Workflow)**: Depends on Phase 6 (evaluator) + Phase 7 (at least smoke tier classified)
- **Phase 9 (New Tests)**: Depends on Phase 3 (gap identification) + calibration charms (external)
- **Phase 10 (Charm Migration)**: Depends on Phase 3 (migration plans) + calibration charms meeting contract (external)
- **Phase 11 (Polish)**: Depends on Phase 8 (workflow to configure)

### User Story Dependencies

- **US7 (P1)**: Phases 3, 4, 5, 10 — audit starts Day 1; substrate and DAG after audit; migration when charms ready
- **US1 (P1)**: Phase 6 — predicate evaluator starts Day 1, independent of audits
- **US2 (P1)**: Phase 6 — tier logic in evaluator, starts Day 1
- **US3 (P2)**: Phase 6 — composition logic in evaluator, starts Day 1
- **US4 (P2)**: Phase 7 — classification, depends on audit + evaluator
- **US5 (P3)**: Phase 8 — workflow, depends on evaluator + classification
- **US6 (P3)**: Phase 9 — new tests, depends on audit + charms

### Within Each Phase

- Tasks marked [P] can run in parallel
- Tasks without [P] must run sequentially within their phase
- Phases 1, 2, 3, 6 can all start Day 1 (no cross-dependencies)

### External Blockers

- **Calibration charms** (norma-k8s, norma): Block Phase 9 and Phase 10. All other phases proceed independently.

### Parallel Opportunities

- **Day 1 parallel starts**: Phase 1 (setup) ‖ Phase 2 (sweeper) ‖ Phase 3 (audit) ‖ Phase 6 (evaluator)
- **Within Phase 3**: T007 starts, then T008-T013 sequentially (T013 in parallel)
- **After Phase 3**: Phase 4 ‖ Phase 5 ‖ Phase 7 (substrate ‖ DAG ‖ classification)
- **Within Phase 7**: T035 ‖ T036 ‖ T037 ‖ T038 ‖ T039 (all suite batches in parallel)
- **Within Phase 9**: T050 ‖ T051 ‖ T052 ‖ T053 ‖ T054 (all new suites in parallel)
- **Within Phase 10**: T057 ‖ T058 ‖ T059 ‖ T060 (K8s suite migrations in parallel)

---

## Parallel Example: Day 1

```bash
# Track A — Test Content (Agent/Developer 1):
Task: T007 "Catalog all 50 suites in coverage-analysis.md"

# Track B — CI Structure (Agent/Developer 2):
Task: T027 "Implement core predicate evaluator in tests/evaluate-predicates.sh"
Task: T028 "Implement tier eligibility logic"

# Track B — Independent (Agent/Developer 3):
Task: T004 "Create tests/ci-sweeper.sh"
Task: T005 "Create .github/workflows/ci-sweeper.yml"
```

---

## Implementation Strategy

### MVP First (Phases 1 + 2 + 3 + 6)

1. Complete Phase 1: Setup (stubs)
2. Complete Phase 2: Resource sweeper (ships immediately, independent value)
3. Complete Phase 3: Audit all 50 suites (delivers coverage-analysis.md v3)
4. Complete Phase 6: Predicate evaluator (delivers working `evaluate-predicates.sh`)
5. **STOP and VALIDATE**: Run evaluator against sample changesets, verify correct suite selection
6. This MVP already provides: resource sweeper running + complete audit + working evaluator

### Incremental Delivery

1. **Phase 1+2+3+6** → Setup + sweeper + audit + evaluator → Can preview what suites would activate (sweeper already running)
2. **Phase 4** → Substrate helpers → Two smoke suites enhanced, mergeable independently
3. **Phase 5** → Fail-fast DAG → Smoke suite has working DAG, mergeable independently
4. **Phase 7** → Classification → All 50 suites have predicates.yaml, mergeable independently
5. **Phase 8** → Unified workflow → Predicate-driven CI live, runs alongside existing workflows
6. **Phase 9** → New tests → Coverage gaps filled, integrated into predicate system
7. **Phase 10** → Charm migration → When norma charms ready, migrate suites incrementally
8. **Phase 11** → Polish → Cleanup, retire old workflows, final documentation

Each phase ships standalone value. No phase requires a later phase to be useful.

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Every phase produces a deployable improvement (governing principle)
- Default verdict is keep/enhance, not rewrite (FR-055-057)
- Charm migration (Phase 10) is blocked on external — all other phases proceed independently
- The evaluator unit tests (T034) cover all spec acceptance scenarios for US1, US2, US3
- Substrate verification (Phase 4) and fail-fast DAG (Phase 5) can be added to suites incrementally — no big-bang rollout
- Calibration charm contract source of truth: `specs/002-ci-test-suite/contracts/charm-contract.yaml` (spec.md tables are a rendered summary)
- `yq` refers to `mikefarah/yq` v4 (Go binary) throughout — NOT `kislyuk/yq` (Python)
