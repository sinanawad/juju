# Research: Predicate-Based CI Test Suite

**Date**: 2026-02-17 | **Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

## R1: Existing Test Framework Architecture

**Decision**: Enhance the existing bash test framework; do not replace it.

**Rationale**: The framework (`main.sh` → `run.sh` → `suites/*/task.sh`) is well-structured, understood by the team, and covers dispatch, filtering, bootstrap, and assertion lifecycle. The `skip()` function in `run.sh` already provides whitelist/blacklist filtering — extending it for DAG-aware skipping is straightforward.

**Key findings from codebase analysis**:
- `tests/main.sh` (407 lines): Dispatches to suites via `TEST_NAMES` registry, supports `-p` provider, `-s` skip list, `run_test()` wraps each suite execution
- `tests/includes/run.sh` (105 lines): `run()` executes subtests with timing; `skip()` uses grep against `RUN_LIST`/`SKIP_LIST` — flat, no DAG
- `tests/includes/juju.sh` (662 lines): `bootstrap()`, `ensure()`, `destroy_controller()` — provider-to-cloud mapping, reuse support
- `tests/includes/wait-for.sh` (415 lines): `wait_for()` polls `juju status` with jq queries
- `tests/includes/check.sh` (129 lines): Assertion helpers (`check_contains`, `check_gt`, etc.)
- Suite pattern: `test_SUITE() { bootstrap → test_X → test_Y → destroy_controller }`

**Alternatives considered**:
- Go test binary → rejected: would require rewriting all 50 suites in Go
- Python pytest → rejected: adds dependency, team uses bash
- No changes → rejected: misses fail-fast and predicate goals

## R2: Predicate Evaluation Strategy

**Decision**: Bash script (`evaluate-predicates.sh`) using `yq` for YAML parsing, outputs JSON matrix.

**Rationale**: Bash is consistent with the test framework. The evaluator is a pure function (inputs → outputs). `yq` is already available on GitHub Actions runners. Output format matches GitHub Actions `matrix` strategy natively.

**Implementation approach**:
```bash
# Pseudocode for evaluate-predicates.sh
for suite_dir in tests/suites/*/; do
  pred_file="$suite_dir/predicates.yaml"
  if [[ ! -f "$pred_file" ]]; then
    # Default: integration tier, all providers, wildcard paths (FR-009)
    tier="integration"; provider="all"; paths=("*")
  else
    tier=$(yq '.tier' "$pred_file")
    provider=$(yq '.provider' "$pred_file")
    paths=($(yq '.paths[]' "$pred_file"))
  fi

  if tier_eligible "$tier" "$event_type" && \
     provider_matches "$provider" "$target_provider" && \
     paths_match "${paths[@]}" "${changed_files[@]}"; then
    add_to_matrix "$suite" "$provider" "$tier"
  fi
done
echo "$matrix_json"
```

**Alternatives considered**:
- Go binary → rejected: adds build step, overkill for YAML parsing + glob matching
- GitHub Actions composite action → rejected: locks logic into GitHub-specific construct
- Extend `dorny/paths-filter` → rejected: doesn't support per-suite YAML metadata

## R3: Fail-Fast DAG Implementation

**Decision**: Extend `run.sh` to read `tests` section of `predicates.yaml`, topologically sort tests, and skip dependents on prerequisite failure.

**Rationale**: DAG per suite is small (3–10 tests). Topological sort in bash is simple for small graphs. The `skip()` function already exists — extending it to check a results file is minimal.

**Implementation approach**:
- On suite start: parse `predicates.yaml` `tests` section into bash associative arrays
- `DEPENDS_ON[test_name]="dep1 dep2"` and `TEST_RESULT[test_name]="pending|pass|fail|skipped"`
- Before each test: check all deps in `DEPENDS_ON[$test]` — if any have `TEST_RESULT=fail`, set this test to `skipped`
- After each test: record result in `TEST_RESULT[$test]`
- Print DAG summary at start: "Test execution order: setup_bootstrap → deploy_norma → {test_a, test_b} → test_c"
- Print skip reasons: "SKIPPED test_storage_attach: dependency 'deploy_norma' failed"

**Alternatives considered**:
- `make` with dependencies → rejected: doesn't integrate with `run()` function, adds complexity
- `set -e` (stop on first error) → rejected: too coarse, kills entire suite instead of selectively skipping dependents
- External orchestrator (e.g., `parallel --dag`) → rejected: adds dependency

## R4: Substrate Verification Approach

**Decision**: New `tests/includes/substrate.sh` with provider-aware verification functions.

**Rationale**: Centralizing substrate checks in a shared helper ensures consistency and reuse. Provider detection is already done in `juju.sh`.

**Function catalog**:
```bash
# K8s (MicroK8s)
substrate_check_pod_exists <app> [namespace]
substrate_check_pod_count <app> <expected> [namespace]
substrate_check_namespace_exists <namespace>
substrate_check_namespace_gone <namespace>
substrate_check_pvc_exists <name> [namespace]
substrate_check_pvc_count <expected> [namespace]
substrate_check_service_exists <name> [namespace]

# LXD
substrate_check_container_exists <name>
substrate_check_container_gone <name>
substrate_check_container_count <expected>

# Generic (provider-aware dispatch)
substrate_verify_deploy <app> [model]
substrate_verify_destroy_model <model>
substrate_verify_scale <app> <expected> [model]
```

Each function:
- Detects current provider from `BOOTSTRAP_PROVIDER` env var
- Calls appropriate substrate tool (`microk8s kubectl`, `lxc`)
- Returns 0 on match, 1 on mismatch with clear error message
- Includes configurable timeout (default 60s) for eventual consistency

**Alternatives considered**:
- Inline per-test → rejected: duplication, inconsistency
- Charm-only verification via actions → rejected: misses cases where Juju and substrate disagree
- Both (charm + substrate) → accepted as enhancement: charm actions validate Juju's view, substrate checks validate reality

## R5: GitHub Actions Workflow Architecture

**Decision**: Single `integration-tests.yml` with matrix strategy from evaluator output.

**Rationale**: GitHub Actions native `matrix` strategy handles parallelism natively. `continue-on-error` achieves non-blocking regression. Keeping existing workflows during transition honors "CI never goes dark."

**Workflow structure**:
```yaml
jobs:
  evaluate:
    outputs:
      matrix: ${{ steps.eval.outputs.matrix }}
    steps:
      - uses: actions/checkout@v4
      - uses: dorny/paths-filter@v3  # get changed files
      - id: eval
        run: |
          tests/evaluate-predicates.sh \
            --event-type "${{ github.event_name }}" \
            --changed-files "${{ steps.paths.outputs.changes_file }}" \
            --provider "all"

  test:
    needs: evaluate
    strategy:
      fail-fast: false
      matrix: ${{ fromJson(needs.evaluate.outputs.matrix) }}
    steps:
      - # setup provider (LXD or MicroK8s based on matrix.provider)
      - # build juju
      - run: cd tests && ./main.sh -p ${{ matrix.provider }} --fail-fast ${{ matrix.suite }}
    continue-on-error: ${{ matrix.required == false }}  # regression on PRs
```

**Transition plan**:
1. Deploy `integration-tests.yml` alongside `smoke.yml`
2. Both run on PRs for 2 weeks
3. Verify `integration-tests.yml` produces equivalent or better results
4. Remove `smoke.yml`, `postgresql-k8s.yml`, `upgrade.yml`, `microk8s-tests.yml`

## R6: Resource Sweeper Design

**Decision**: `ci-sweeper.sh` + hourly `ci-sweeper.yml` workflow.

**Rationale**: Simple, idempotent, no external dependencies. Proven pattern from Terraform CI.

**Naming convention**: `ci-<GITHUB_RUN_ID>-<suite>` for controllers/models created in CI.

**Sweep logic**:
```bash
# List all controllers matching ci-* pattern
for controller in $(juju controllers --format=json | jq -r '.controllers | keys[] | select(startswith("ci-"))'); do
  age=$(controller_age_hours "$controller")
  if (( age > TTL_HOURS )); then
    juju destroy-controller "$controller" --destroy-all-models --destroy-storage -y
  fi
done
# Also sweep orphaned LXD containers and K8s namespaces
```

## Industry Best Practices Applied

See [ci-best-practices-research.md](ci-best-practices-research.md) for full industry analysis. Key patterns adopted:

| Practice | Source | How Applied |
|----------|--------|-------------|
| Path-based test selection | DigitalOcean (GTA), Kubernetes (Prow) | Curated glob patterns in `predicates.yaml` |
| Tiered test execution | CockroachDB, Kubernetes | 4-tier taxonomy: sanity/smoke/regression/integration |
| Informational checks on PRs | Kubernetes Tide | Regression tier as non-blocking on PRs |
| Resource sweepers | Terraform CI | `ci-*` naming + hourly TTL-based cleanup |
| Matrix-based parallelism | GitHub Actions native | Evaluator outputs JSON matrix for `strategy.matrix` |
| Calibration/mock charms | Internal (norma-k8s) | Contract-driven test sterility |
