# Tasks: Predicate-Based CI Test Suite

**Input**: Design documents from `/specs/002-ci-test-suite/`
**Prerequisites**: plan.md (complete), spec.md (complete), research.md, data-model.md, contracts/

**Tests**: Tests are NOT explicitly requested. Test tasks are omitted. The work itself IS the test infrastructure.

**Organization**: Tasks follow two parallel tracks (A: test content, B: CI structure) mapped to user stories. The governing principle requires every task to produce a deployable improvement.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US7)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: Establish the predicate schema documentation and validate tooling prerequisites.

- [ ] T001 Validate `yq` availability on CI runners and document version requirement in `specs/002-ci-test-suite/quickstart.md`
- [ ] T002 Create `tests/includes/substrate.sh` with empty function stubs and source guard in `tests/includes/juju.sh`
- [ ] T003 [P] Create `tests/includes/predicates.sh` with empty function stubs for predicate evaluation helpers

**Checkpoint**: Tooling confirmed, helper files exist (empty stubs), no behavior changes to existing CI.

---

## Phase 2: Foundational — Audits (A1a + A1b)

**Purpose**: Understand the current state of all 50 suites before making any changes. MUST complete before meaningful implementation in any user story.

**Why blocking**: Every subsequent phase (substrate verification, DAG, classification, new tests) depends on knowing what each suite does, what charms it uses, and what its implicit dependencies are.

- [ ] T004 [P] [US7] Read and catalog all 50 `tests/suites/*/task.sh` files — for each suite, list: test functions, charms deployed, provider guards, bootstrap/destroy lifecycle, and Juju capabilities exercised. Output to `specs/002-ci-test-suite/coverage-analysis.md` (update existing v2)
- [ ] T005 [P] [US7] Cross-reference suite catalog (T004) against the 175-capability inventory in `specs/002-ci-test-suite/coverage-analysis.md` — identify NONE/MINIMAL gaps and over-testing (multiple suites testing same capability identically)
- [ ] T006 [US7] For each suite, score on three quality dimensions in `specs/002-ci-test-suite/coverage-analysis.md`: sterility (0-2: calibration/mixed/third-party), substrate verification (0-2: present/partial/absent), fail-fast readiness (0-2: DAG declared/implicit ordering/no ordering)
- [ ] T007 [US7] Assign per-suite verdict (keep/enhance/migrate/rewrite) in `specs/002-ci-test-suite/coverage-analysis.md` — default is keep or enhance per FR-057. Document justification for any migrate or rewrite verdict
- [ ] T008 [US7] For each suite, map the implicit test dependency DAG — which test functions depend on bootstrap, which depend on prior deployments. Output as proposed `tests` section per suite in `specs/002-ci-test-suite/coverage-analysis.md`
- [ ] T009 [US7] Produce per-suite migration plan (for suites verdicted as migrate): which charm to swap, what substrate checks to add, what DAG to declare. Append to `specs/002-ci-test-suite/coverage-analysis.md`
- [ ] T010 [P] [US7] Feed calibration charm contract (CC-01 through CC-M11) from `specs/002-ci-test-suite/contracts/charm-contract.yaml` into norma-k8s and norma charm specs as external requirements. Document handoff in `specs/002-ci-test-suite/coverage-analysis.md`

**Checkpoint**: Complete understanding of all 50 suites. Verdicts, DAG maps, migration plans, and coverage gaps documented. All subsequent phases have their input data.

---

## Phase 3: US7 — Substrate Verification Helpers (Priority: P1)

**Goal**: Every mutating test can verify its outcome on the substrate. Ships as a shared helper library — existing tests can adopt incrementally.

**Independent Test**: Source `substrate.sh` in any suite, call `substrate_check_namespace_gone` after `juju destroy-model`, verify it works on MicroK8s.

- [ ] T011 [US7] Implement K8s substrate verification functions in `tests/includes/substrate.sh`: `substrate_check_pod_exists`, `substrate_check_pod_count`, `substrate_check_namespace_exists`, `substrate_check_namespace_gone`, `substrate_check_pvc_exists`, `substrate_check_pvc_count`, `substrate_check_service_exists`
- [ ] T012 [P] [US7] Implement LXD substrate verification functions in `tests/includes/substrate.sh`: `substrate_check_container_exists`, `substrate_check_container_gone`, `substrate_check_container_count`
- [ ] T013 [US7] Implement provider-aware dispatch functions in `tests/includes/substrate.sh`: `substrate_verify_deploy`, `substrate_verify_destroy_model`, `substrate_verify_scale` — detect `BOOTSTRAP_PROVIDER` and call appropriate K8s/LXD function
- [ ] T014 [US7] Add configurable timeout (default 60s) and clear error messages to all substrate functions in `tests/includes/substrate.sh` — format: "substrate verification: expected <X> but found <Y> after <T>s"
- [ ] T015 [US7] Add substrate verification calls to `tests/suites/smoke/deploy.sh` after `juju deploy` and `juju destroy-model` — validate the helper works in a real suite without changing test logic
- [ ] T016 [P] [US7] Add substrate verification calls to `tests/suites/smoke_k8s/task.sh` after K8s deploy — validate K8s-specific helpers work

**Checkpoint**: `substrate.sh` is a working library. Two smoke suites enhanced with substrate verification. Can be merged independently.

---

## Phase 4: US7 — Fail-Fast DAG (Priority: P1)

**Goal**: When bootstrap fails, dependent tests skip instantly instead of running and failing one by one.

**Independent Test**: Run a suite where bootstrap is intentionally broken — verify dependent tests report "skipped: dependency failed" and suite completes in bootstrap-attempt time, not N*test time.

- [ ] T017 [US7] Extend `tests/includes/run.sh` with DAG-aware functions: `load_test_dag` (parse `predicates.yaml` `tests` section via `yq`), `check_dependencies` (check if all `depends_on` tests passed), `record_test_result` (write to `$TEST_DIR/test-results.tmp`)
- [ ] T018 [US7] Implement topological sort in `tests/includes/run.sh` for test execution ordering — use bash associative arrays, handle DAG cycles with error message
- [ ] T019 [US7] Extend `skip()` function in `tests/includes/run.sh` to check dependency results before each test — if any dependency failed, print "SKIPPED <test>: dependency '<dep>' failed" and return without executing
- [ ] T020 [US7] Add `--fail-fast` flag to `tests/main.sh` — when set, stop suite execution on first non-prerequisite test failure (prerequisites always run to establish the dependency chain)
- [ ] T021 [US7] Print DAG summary at suite start when `predicates.yaml` has a `tests` section — format: "Test execution order: setup → deploy → {test_a, test_b} → test_c"
- [ ] T022 [US7] Make DAG opt-in: suites without `tests` section in `predicates.yaml` run in existing sequential order from `task.sh` — zero behavior change for unclassified suites
- [ ] T023 [US7] Add `tests` DAG section to `tests/suites/smoke/predicates.yaml` (create file if not exists) mapping `test_build` and `test_deploy` with bootstrap as prerequisite — validate DAG works on a real suite

**Checkpoint**: `run.sh` supports DAG-aware execution. Smoke suite has a working DAG. Suites without predicates.yaml are unaffected.

---

## Phase 5: US1 + US2 + US3 — Predicate Evaluation System (Priority: P1/P2)

**Goal**: A script reads all `predicates.yaml` files, evaluates against event context and changeset, and outputs a JSON matrix of activated suites for GitHub Actions.

**Independent Test**: Run `evaluate-predicates.sh --event-type pull_request --changed-files <(echo "domain/secret/service.go") --provider all` and verify only secrets-related suites appear in output.

- [ ] T024 [US1] Implement core predicate evaluator in `tests/evaluate-predicates.sh`: argument parsing for `--event-type`, `--changed-files`, `--provider`, `--tier-override`
- [ ] T025 [US2] Implement tier eligibility logic in `tests/evaluate-predicates.sh`: map event types to eligible tiers per FR-010 (PR → sanity+smoke required + regression informational; push → sanity+smoke+regression; schedule → all; dispatch → configurable)
- [ ] T026 [US1] Implement path matching logic in `tests/evaluate-predicates.sh`: for each suite's `predicates.yaml`, check if any glob pattern matches any changed file. Handle wildcards: `core/`, `go.mod`, `tests/includes/` → all suites eligible (FR-007)
- [ ] T027 [US3] Implement predicate composition (AND logic) in `tests/evaluate-predicates.sh`: suite activates only if tier eligible AND provider matches AND paths match (FR-008)
- [ ] T028 [US1] Implement default handling in `tests/evaluate-predicates.sh`: suites without `predicates.yaml` default to tier=integration, provider=all, paths=wildcard (FR-009)
- [ ] T029 [US2] Implement JSON matrix output in `tests/evaluate-predicates.sh`: format as `{"include": [{"suite": "...", "provider": "...", "tier": "...", "required": true/false}]}` compatible with GitHub Actions `strategy.matrix`
- [ ] T030 [US3] Implement `predicates.yaml` schema validation in `tests/evaluate-predicates.sh`: warn on missing required fields, error on invalid tier/provider values
- [ ] T031 [P] [US1] Write evaluator unit tests as bash test cases in `tests/test-evaluate-predicates.sh`: test path matching, tier eligibility, default handling, wildcard paths, predicate composition — at least 10 test cases covering spec acceptance scenarios

**Checkpoint**: `evaluate-predicates.sh` is a working, tested script. Can be run locally to preview what suites would activate for any changeset.

---

## Phase 6: US4 — Classify All 50 Existing Suites (Priority: P2)

**Goal**: Every existing suite has a `predicates.yaml` with tier, provider, paths, charms, and rationale.

**Independent Test**: Run `evaluate-predicates.sh` against sample changesets and verify: (a) every suite has metadata, (b) `domain/secret/` change triggers only secrets suites, (c) `core/` change triggers all suites for the active tier.

- [ ] T032 [P] [US4] Create `predicates.yaml` for all 6 smoke-tier suites: `tests/suites/smoke/predicates.yaml`, `tests/suites/smoke_k8s/predicates.yaml`, `tests/suites/smoke_k8s_psql/predicates.yaml`, `tests/suites/deploy/predicates.yaml`, `tests/suites/cli/predicates.yaml`, `tests/suites/charmhub/predicates.yaml` — using tier, provider, and path data from audit (T004-T007)
- [ ] T033 [P] [US4] Create `predicates.yaml` for first batch of 10 regression-tier suites: `deploy_caas`, `storage`, `storage_k8s`, `sidecar`, `caasadmission`, `secrets_iaas`, `secrets_k8s`, `relations`, `cmr`, `resources` — using audit data
- [ ] T034 [P] [US4] Create `predicates.yaml` for second batch of 15 regression-tier suites: `hooks`, `hooktools`, `actions`, `model`, `controller`, `bootstrap`, `agents`, `refresh`, `constraints`, `authorized_keys`, `credential`, `user`, `network`, `machine`, `appdata`, `dashboard` — using audit data
- [ ] T035 [P] [US4] Create `predicates.yaml` for the sanity-tier suite: `tests/suites/static_analysis/predicates.yaml`
- [ ] T036 [P] [US4] Create `predicates.yaml` for all 15 integration-tier suites: `upgrade`, `controllercharm`, `coslite`, `kubeflow`, `ck`, `deploy_aks`, `spaces_ec2`, `spaces_gce`, `cloud_azure`, `cloud_gce`, `firewall`, `ovs_maas`, `manual`, `unmanaged`, `examples` — using audit data
- [ ] T037 [US4] Validate all 50 `predicates.yaml` files: run `evaluate-predicates.sh` against 5 sample changesets (domain/secret change, core/ change, caas/ change, cmd/juju/ change, docs-only change) and verify expected activation for each

**Checkpoint**: All 50 suites classified. Evaluator correctly selects suites based on changesets. Can be merged — predicates.yaml files are inert until the workflow reads them.

---

## Phase 7: US5 — GitHub Actions Workflow Consolidation (Priority: P3)

**Goal**: A single `integration-tests.yml` replaces scattered test workflows, dispatching matching suites via matrix.

**Independent Test**: Open a PR, verify `integration-tests.yml` triggers the same suites as `smoke.yml` plus additional regression suites (informational). Verify a push to main triggers regression as blocking.

- [ ] T038 [US5] Create `.github/workflows/integration-tests.yml` with evaluate job: checkout, get changed files (via `dorny/paths-filter` or git diff), run `tests/evaluate-predicates.sh`, output matrix as job output
- [ ] T039 [US5] Add test matrix job to `.github/workflows/integration-tests.yml`: spawn parallel jobs from evaluator output, each runs `tests/main.sh -p <provider> <suite>`, configure `fail-fast: false` on matrix strategy
- [ ] T040 [US5] Configure blocking vs informational in `.github/workflows/integration-tests.yml`: regression-tier jobs on PRs use `continue-on-error: true` (non-blocking); all tiers on push-to-main are blocking
- [ ] T041 [US5] Add workflow_dispatch trigger to `.github/workflows/integration-tests.yml` with tier override input (dropdown: smoke, regression, all) per FR-011
- [ ] T042 [US5] Add schedule trigger (nightly) to `.github/workflows/integration-tests.yml` with all tiers eligible
- [ ] T043 [US5] Configure runner labels in `.github/workflows/integration-tests.yml`: LXD suites on quad-xlarge, K8s suites on xxlarge (matching current `smoke.yml` runner config)
- [ ] T044 [US5] Run `integration-tests.yml` alongside `smoke.yml` for validation — both trigger on same events, compare results over 2-week transition period
- [ ] T045 [US5] After transition validation: remove `smoke.yml`, `postgresql-k8s.yml`, `upgrade.yml`, `microk8s-tests.yml` from `.github/workflows/` and add `terraform-smoke.yml`, `migrate.yml` to retired list

**Checkpoint**: Unified workflow running in production. Old scattered workflows removed. Adding a new suite requires only `predicates.yaml` — zero workflow changes.

---

## Phase 8: US6 — New Test Coverage for Gaps (Priority: P3)

**Goal**: Fill identified coverage gaps with new test suites using calibration charms.

**Independent Test**: Deploy `constraints_k8s` suite locally via `./main.sh -p microk8s constraints_k8s`, verify it tests K8s-specific constraints and has working `predicates.yaml`.

- [ ] T046 [P] [US6] Create `tests/suites/constraints_k8s/task.sh` and `tests/suites/constraints_k8s/predicates.yaml` — test K8s-specific constraints (cpu, memory, deployment-type) using norma-k8s charm. Tier: regression, provider: k8s, paths: `domain/constraint/**`, `internal/provider/kubernetes/**`
- [ ] T047 [P] [US6] Create `tests/suites/deploy_caas_lifecycle/task.sh` and `tests/suites/deploy_caas_lifecycle/predicates.yaml` — full CAAS lifecycle (deploy, scale, config, action, remove) using norma-k8s. Tier: regression, provider: k8s
- [ ] T048 [P] [US6] Create `tests/suites/controller_lifecycle/task.sh` and `tests/suites/controller_lifecycle/predicates.yaml` — controller bootstrap, enable-ha, disable-ha, controller config, teardown. Tier: regression, provider: all
- [ ] T049 [P] [US6] Create `tests/suites/deploy_caas_deployment_type/task.sh` and `tests/suites/deploy_caas_deployment_type/predicates.yaml` — Deployment/DaemonSet/StatefulSet variants using norma-k8s. Tier: regression, provider: k8s
- [ ] T050 [P] [US6] Create `tests/suites/storage_k8s_deployment/task.sh` and `tests/suites/storage_k8s_deployment/predicates.yaml` — storage behavior with Deployment type (no stable PVC naming). Tier: regression, provider: k8s
- [ ] T051 [US6] Create additional test suites for gaps identified by coverage audit (T005) — exact suites TBD from audit results, each with `task.sh` + `predicates.yaml`

**Checkpoint**: Coverage gaps filled. New suites integrated into predicate system and activated on relevant PRs.

---

## Phase 9: US7 — Charm Migration (Priority: P1, Blocked on External)

**Goal**: Replace third-party charms with norma calibration charms in smoke and regression tiers.

**Independent Test**: Run migrated suite (e.g., `smoke_k8s`) — verify it uses norma-k8s, passes all existing assertions, and includes substrate verification.

**External dependency**: Blocked on norma-k8s meeting contract CC-01 through CC-K11, and norma meeting CC-01 through CC-M11. Start when charms are ready.

- [ ] T052 [US7] Migrate `tests/suites/smoke_k8s/` to use norma-k8s instead of juju-qa-test — swap charm reference, update wait conditions, preserve test logic and assertions
- [ ] T053 [P] [US7] Migrate `tests/suites/smoke_k8s_psql/` to use norma-k8s instead of postgresql-k8s — swap charm, add storage verification via norma actions, preserve test flow
- [ ] T054 [P] [US7] Migrate `tests/suites/deploy_caas/` to use norma-k8s — swap charm, preserve deployment test logic
- [ ] T055 [P] [US7] Migrate `tests/suites/storage_k8s/` to use norma-k8s storage capabilities — swap charm, add `check-storage` action calls
- [ ] T056 [P] [US7] Migrate `tests/suites/relations/` to use norma self-relation (`calibration-provider` ↔ `calibration-requirer`) — swap charm, preserve relation test logic
- [ ] T057 [US7] Migrate `tests/suites/deploy/` to use norma (machine charm) — swap ubuntu/juju-qa-test references, preserve deployment test logic
- [ ] T058 [US7] Update `charms` section in all migrated suites' `predicates.yaml` to reflect `type: calibration`
- [ ] T059 [US7] Add substrate verification (from `substrate.sh`) to all migrated suites — call `substrate_verify_deploy`, `substrate_verify_destroy_model`, `substrate_verify_scale` after corresponding Juju operations

**Checkpoint**: Smoke and regression tier suites use calibration charms. Third-party charm dependencies removed from these tiers. Substrate verification active.

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Resource sweeper, documentation, workflow retirement, final validation.

- [ ] T060 [P] Create `tests/ci-sweeper.sh` — list all Juju controllers matching `ci-*` pattern, destroy those older than configurable TTL (default 4h), sweep orphaned LXD containers and MicroK8s namespaces matching `ci-*`. Idempotent.
- [ ] T061 [P] Create `.github/workflows/ci-sweeper.yml` — scheduled hourly, runs `tests/ci-sweeper.sh` with 4h TTL, plus manual dispatch for emergency cleanup
- [ ] T062 Update `tests/includes/juju.sh` `bootstrap()` function — prefix controller names with `ci-${GITHUB_RUN_ID:-local}-` when `CI=true` env var is set. Backward-compatible: local runs keep current naming
- [ ] T063 [P] Retire disabled workflows: remove `terraform-smoke.yml` and `migrate.yml` from `.github/workflows/` (both currently have `if: false`)
- [ ] T064 [P] Evaluate `jaas-smoke.yml` — determine if it should be integrated into predicate system or kept standalone (external JIMM dependency). Document decision in `specs/002-ci-test-suite/plan.md`
- [ ] T065 Add `--fail-fast` flag to all `main.sh` invocations in `.github/workflows/integration-tests.yml`
- [ ] T066 Validate end-to-end: open a test PR touching `domain/secret/`, verify only secrets-related suites activate, regression tier is informational, smoke tier is blocking
- [ ] T067 Update `specs/002-ci-test-suite/quickstart.md` with lessons learned from implementation — add troubleshooting section, update examples with real suite names

**Checkpoint**: CI infrastructure complete. Resource sweeper running. All documentation up to date.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies — start immediately
- **Phase 2 (Audits)**: No dependencies — start immediately, parallel with Phase 1
- **Phase 3 (Substrate)**: Depends on T002 (stub file from Phase 1)
- **Phase 4 (Fail-Fast DAG)**: Depends on Phase 2 (audit provides DAG mapping)
- **Phase 5 (Predicate Evaluator)**: No dependencies — can start Day 1 in parallel with audits
- **Phase 6 (Classification)**: Depends on Phase 2 (audit data for tier/path assignment) + Phase 5 (schema)
- **Phase 7 (Workflow)**: Depends on Phase 5 (evaluator) + Phase 6 (at least smoke tier classified)
- **Phase 8 (New Tests)**: Depends on Phase 2 (gap identification) + calibration charms (external)
- **Phase 9 (Charm Migration)**: Depends on Phase 2 (migration plans) + calibration charms meeting contract (external)
- **Phase 10 (Polish)**: Depends on Phase 7 (workflow to configure)

### User Story Dependencies

- **US7 (P1)**: Phases 2, 3, 4, 9 — audit starts Day 1; substrate and DAG after audit; migration when charms ready
- **US1 (P1)**: Phase 5 — predicate evaluator starts Day 1, independent of audits
- **US2 (P1)**: Phase 5 — tier logic in evaluator, starts Day 1
- **US3 (P2)**: Phase 5 — composition logic in evaluator, starts Day 1
- **US4 (P2)**: Phase 6 — classification, depends on audit + evaluator
- **US5 (P3)**: Phase 7 — workflow, depends on evaluator + classification
- **US6 (P3)**: Phase 8 — new tests, depends on audit + charms

### Within Each Phase

- Tasks marked [P] can run in parallel
- Tasks without [P] must run sequentially within their phase
- Phases 1, 2, 3, 5 can all start Day 1 (no cross-dependencies)

### External Blockers

- **Calibration charms** (norma-k8s, norma): Block Phase 8 and Phase 9. All other phases proceed independently.

### Parallel Opportunities

- **Day 1 parallel starts**: Phase 1 (setup) ‖ Phase 2 (audit) ‖ Phase 5 (evaluator)
- **Within Phase 2**: T004 ‖ T005 (catalog and cross-reference)
- **After Phase 2**: Phase 3 ‖ Phase 4 ‖ Phase 6 (substrate ‖ DAG ‖ classification)
- **Within Phase 6**: T032 ‖ T033 ‖ T034 ‖ T035 ‖ T036 (all suite batches in parallel)
- **Within Phase 8**: T046 ‖ T047 ‖ T048 ‖ T049 ‖ T050 (all new suites in parallel)
- **Within Phase 9**: T053 ‖ T054 ‖ T055 ‖ T056 (K8s suite migrations in parallel)

---

## Parallel Example: Day 1

```bash
# Track A — Test Content (Agent/Developer 1):
Task: T004 "Catalog all 50 suites in coverage-analysis.md"
Task: T005 "Cross-reference against capability inventory"

# Track B — CI Structure (Agent/Developer 2):
Task: T024 "Implement core predicate evaluator in tests/evaluate-predicates.sh"
Task: T025 "Implement tier eligibility logic"

# Track B — Independent (Agent/Developer 3):
Task: T060 "Create tests/ci-sweeper.sh"
Task: T061 "Create .github/workflows/ci-sweeper.yml"
```

---

## Implementation Strategy

### MVP First (Phases 1 + 2 + 5)

1. Complete Phase 1: Setup (stubs)
2. Complete Phase 2: Audit all 50 suites (delivers coverage-analysis.md v3)
3. Complete Phase 5: Predicate evaluator (delivers working `evaluate-predicates.sh`)
4. **STOP and VALIDATE**: Run evaluator against sample changesets, verify correct suite selection
5. This MVP already provides: complete audit + working evaluator = informed decisions + smart test selection

### Incremental Delivery

1. **Phase 1+2+5** → Audit + evaluator → Can preview what suites would activate (no CI changes yet)
2. **Phase 3** → Substrate helpers → Two smoke suites enhanced, mergeable independently
3. **Phase 4** → Fail-fast DAG → Smoke suite has working DAG, mergeable independently
4. **Phase 6** → Classification → All 50 suites have predicates.yaml, mergeable independently
5. **Phase 7** → Unified workflow → Predicate-driven CI live, runs alongside existing workflows
6. **Phase 8** → New tests → Coverage gaps filled, integrated into predicate system
7. **Phase 9** → Charm migration → When norma charms ready, migrate suites incrementally
8. **Phase 10** → Polish → Sweeper, cleanup, retire old workflows

Each phase ships standalone value. No phase requires a later phase to be useful.

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Every phase produces a deployable improvement (governing principle)
- Default verdict is keep/enhance, not rewrite (FR-055-057)
- Charm migration (Phase 9) is blocked on external — all other phases proceed independently
- The evaluator unit tests (T031) cover all spec acceptance scenarios for US1, US2, US3
- Substrate verification (Phase 3) and fail-fast DAG (Phase 4) can be added to suites incrementally — no big-bang rollout
