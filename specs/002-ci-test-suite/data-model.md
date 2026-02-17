# Data Model: Predicate-Based CI Test Suite

**Date**: 2026-02-17 | **Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

## Primary Entity: `predicates.yaml`

**Location**: `tests/suites/<suite-name>/predicates.yaml`
**One per suite**, co-located with `task.sh`.

### Schema (v1.0)

```yaml
# Required: Schema version for forward compatibility
schema: "1.0"

# Required: Test tier classification (FR-001, FR-002)
# Determines eligibility based on event type
tier: regression  # enum: sanity | smoke | regression | integration

# Required: Provider filter (FR-016)
# Which infrastructure provider this suite requires
provider: k8s     # enum: all | k8s | iaas | ec2 | gce | azure | maas | manual | unmanaged

# Required: Path predicates (FR-007, FR-012, FR-013)
# Glob patterns matching source paths that should trigger this suite
# At least one must match the changeset for the suite to activate
# Use ["*"] for always-run suites
paths:
  - domain/storage/**
  - internal/worker/caas*/**
  - caas/**

# Required: Charm inventory (FR-037)
# Declares which charms the suite uses, enabling sterility audit
charms:
  - name: norma-k8s           # charm name as used in deploy command
    type: calibration          # enum: calibration | third-party
  - name: postgresql-k8s
    type: third-party          # only if test explicitly validates this integration

# Required: Rationale (FR-003)
# Human-readable explanation of why this suite exists and its tier
rationale: "K8s PV/PVC handling and race conditions under StatefulSet and Deployment types"

# Optional: Intra-suite test dependency DAG (FR-039, FR-040, FR-041)
# If omitted, tests run in the order defined in task.sh (existing behavior)
tests:
  - name: setup_bootstrap        # must match the bash function name
    type: prerequisite           # enum: prerequisite | test
    # No depends_on → runs first (root of DAG)

  - name: deploy_norma
    type: prerequisite
    depends_on: [setup_bootstrap]  # list of test names that must pass first

  - name: test_storage_attach
    type: test
    depends_on: [deploy_norma]

  - name: test_storage_persist_restart
    type: test
    depends_on: [deploy_norma]    # independent of test_storage_attach

  - name: test_storage_detach
    type: test
    depends_on: [test_storage_attach]  # sequential dependency
```

### Field Reference

| Field | Type | Required | Default (if missing file) | Spec Ref |
|-------|------|----------|--------------------------|----------|
| `schema` | string | Yes | — | — |
| `tier` | enum | Yes | `integration` (FR-009) | FR-001, FR-002 |
| `provider` | enum | Yes | `all` (FR-009) | FR-016 |
| `paths` | list[string] | Yes | `["*"]` (FR-009) | FR-007, FR-012, FR-013 |
| `charms` | list[object] | Yes | — | FR-037 |
| `charms[].name` | string | Yes | — | FR-037 |
| `charms[].type` | enum | Yes | — | FR-037 |
| `rationale` | string | Yes | — | FR-003 |
| `tests` | list[object] | No | (run in task.sh order) | FR-039 |
| `tests[].name` | string | Yes* | — | FR-039 |
| `tests[].type` | enum | No | `test` | FR-041 |
| `tests[].depends_on` | list[string] | No | `[]` (no deps) | FR-039, FR-040 |

### Validation Rules

1. `tier` must be one of: `sanity`, `smoke`, `regression`, `integration`
2. `provider` must be one of: `all`, `k8s`, `iaas`, `ec2`, `gce`, `azure`, `maas`, `manual`, `unmanaged`
3. `paths` must contain at least one entry; each entry is a glob pattern
4. `charms[].type` must be one of: `calibration`, `third-party`
5. `tests[].name` must match a bash function name defined in the suite's `.sh` files
6. `tests[].depends_on` must reference names defined in the same `tests` list (no forward-reference to other suites)
7. `tests` DAG must be acyclic (no circular dependencies)
8. If `tests` section is present, every test function called by `task.sh` should be listed (completeness)

### Default Behavior (Missing File)

Per FR-009: A suite without `predicates.yaml` defaults to:
```yaml
tier: integration
provider: all
paths: ["*"]
```
This ensures it only runs on nightly/manual builds until explicitly classified.

## Secondary Entity: Evaluator Output (JSON Matrix)

**Produced by**: `tests/evaluate-predicates.sh`
**Consumed by**: `.github/workflows/integration-tests.yml` `strategy.matrix`

```json
{
  "include": [
    {
      "suite": "smoke",
      "provider": "localhost",
      "tier": "smoke",
      "required": true
    },
    {
      "suite": "smoke_k8s",
      "provider": "microk8s",
      "tier": "smoke",
      "required": true
    },
    {
      "suite": "storage_k8s",
      "provider": "microk8s",
      "tier": "regression",
      "required": false
    }
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `suite` | string | Suite directory name (under `tests/suites/`) |
| `provider` | string | Resolved provider for this run (e.g., `microk8s`, `localhost`) |
| `tier` | string | Tier classification |
| `required` | boolean | `true` = merge-blocking; `false` = informational (regression on PRs) |

### `required` Logic (FR-010, FR-024)

| Event | Tier | `required` |
|-------|------|-----------|
| `pull_request` | sanity, smoke | `true` |
| `pull_request` | regression | `false` |
| `pull_request` | integration | not activated |
| `push` (main) | sanity, smoke, regression | `true` |
| `push` (main) | integration | not activated |
| `schedule` | all | `true` |
| `workflow_dispatch` | per override | `true` |

## Entity: Test Result (Runtime)

**Produced by**: Enhanced `run.sh` during suite execution
**Location**: `$TEST_DIR/test-results.tmp` (ephemeral, per run)

```
setup_bootstrap:pass
deploy_norma:pass
test_storage_attach:pass
test_storage_persist_restart:fail
test_storage_detach:skipped:dependency 'test_storage_attach' OK but 'test_storage_persist_restart' is independent
```

Format: `<test-name>:<status>[:<reason>]`

Status values: `pass`, `fail`, `skipped`

Used by the DAG-aware `skip()` function to determine whether to run or skip each test.

## Entity Relationships

```
predicates.yaml (1 per suite)
  ├── tier ──► event-to-tier mapping (FR-010) ──► eligible/not eligible
  ├── provider ──► provider availability check ──► eligible/not eligible
  ├── paths[] ──► changeset matching ──► eligible/not eligible
  ├── charms[] ──► sterility audit (FR-037, FR-053)
  └── tests[] ──► DAG execution (FR-039-042)
        └── depends_on[] ──► test-results.tmp ──► skip/run decision

evaluate-predicates.sh
  reads: all predicates.yaml files + event context
  outputs: JSON matrix

integration-tests.yml
  reads: JSON matrix
  spawns: parallel jobs per matrix entry
  each job runs: main.sh <suite> --fail-fast
```
