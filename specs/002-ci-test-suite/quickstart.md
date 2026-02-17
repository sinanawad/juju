# Quickstart: Working with the Predicate CI System

## Prerequisites

| Tool | Required Version | Check Command | Notes |
|------|-----------------|---------------|-------|
| `yq` | mikefarah/yq v4.x | `yq --version` (must show `mikefarah/yq`) | Used by predicate evaluator and DAG parser. **NOT** `kislyuk/yq` (Python). Pre-installed on GitHub Actions self-hosted runners. |
| `jq` | 1.6+ | `jq --version` | Used by test assertions and evaluator output. Already in test dependencies. |
| `shellcheck` | 0.8+ | `shellcheck --version` | Used by `static_analysis` suite. Already in test dependencies. |

Install `mikefarah/yq` v4 locally (if not already present):

```bash
# Linux (amd64)
sudo wget -qO /usr/local/bin/yq https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64
sudo chmod +x /usr/local/bin/yq

# Verify it is mikefarah/yq (NOT kislyuk/yq)
yq --version
# Expected: yq (https://github.com/mikefarah/yq/) version v4.x.x
```

## Adding Predicates to an Existing Suite

1. Create `tests/suites/<your-suite>/predicates.yaml`:

```yaml
schema: "1.0"
tier: regression
provider: k8s
paths:
  - domain/storage/**
  - internal/worker/caas*/**
charms:
  - name: norma-k8s
    type: calibration
rationale: "K8s storage lifecycle including PVC management"
```

2. That's it. The suite now participates in predicate-based activation. No workflow changes needed.

## Adding a New Test Suite

1. Create `tests/suites/<new-suite>/task.sh` following existing patterns:

```bash
test_new_suite() {
    if [ "$(skip 'test_new_suite')" ]; then return; fi
    set_verbosity
    check_dependencies juju

    bootstrap "test-new-suite" "${TEST_DIR}/test-new-suite.log"
    test_my_feature
    destroy_controller "test-new-suite"
}

test_my_feature() {
    # ... your test logic
}
```

2. Create `tests/suites/<new-suite>/predicates.yaml` with tier, provider, paths, charms.

3. The predicate evaluator will automatically discover and activate the suite on matching PRs.

## Adding a Test DAG (Fail-Fast Dependencies)

Add a `tests` section to your `predicates.yaml`:

```yaml
tests:
  - name: test_new_suite        # matches the entry function in task.sh
    type: prerequisite

  - name: test_my_feature
    type: test
    depends_on: [test_new_suite]

  - name: test_cleanup_verify
    type: test
    depends_on: [test_my_feature]
```

If `test_new_suite` (bootstrap) fails, `test_my_feature` and `test_cleanup_verify` are skipped immediately.

## Adding Substrate Verification to a Test

Source the substrate helpers and call after Juju operations:

```bash
source "${TEST_DIR}/../includes/substrate.sh"

test_deploy_and_verify() {
    juju deploy norma-k8s
    wait_for "norma-k8s" "$(idle_condition "norma-k8s")"

    # Verify on the substrate
    substrate_verify_deploy "norma-k8s"
}

test_destroy_and_verify() {
    juju destroy-model test-model --no-prompt
    wait_for_model_removed "test-model"

    # Verify on the substrate
    substrate_check_namespace_gone "test-model"
}
```

## Running Locally

Nothing changes for local development:

```bash
# Run a specific suite (existing behavior, unchanged)
cd tests && ./main.sh -p microk8s smoke_k8s

# Run with fail-fast (new flag, optional)
cd tests && ./main.sh -p microk8s --fail-fast smoke_k8s

# Run a specific test within a suite
cd tests && ./main.sh -p microk8s smoke_k8s test_deploy
```

## Testing the Predicate Evaluator

```bash
# See what suites would activate for a PR that changes storage code
tests/evaluate-predicates.sh \
  --event-type pull_request \
  --changed-files <(echo "domain/storage/service/service.go") \
  --provider all

# See what runs on nightly (all tiers)
tests/evaluate-predicates.sh \
  --event-type schedule \
  --changed-files /dev/null \
  --provider all

# See what runs when core/ changes (should be everything for the tier)
tests/evaluate-predicates.sh \
  --event-type push \
  --changed-files <(echo "core/network/address.go") \
  --provider all
```

## Provider Mapping

| `provider` in YAML | CI Provider | Substrate Tool |
|-------|-------------|----------------|
| `all` | Both LXD and MicroK8s | Both |
| `k8s` | MicroK8s | `microk8s kubectl` |
| `iaas` | LXD | `lxc` |
| `ec2` | AWS | `aws` CLI |
| `gce` | GCE | `gcloud` CLI |
| `azure` | Azure | `az` CLI |
| `maas` | MAAS | MAAS CLI |

## Tier Time Targets

| Tier | Target | When It Runs |
|------|--------|-------------|
| Sanity | <5 min | Always |
| Smoke | <15 min | PR (blocking), push, nightly |
| Regression | <60 min | PR (informational), push (blocking), nightly |
| Integration | <4 hours | Nightly, manual |
