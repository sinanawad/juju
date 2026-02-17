# CI Best Practices Research for Juju

> Deep research synthesis from 4 parallel investigations covering large Go projects,
> smart test selection, cost/speed optimization, and orchestration platform CI patterns.
>
> **Purpose**: Inform the design of Juju's predicate-based CI test suite.
> **Date**: 2026-02-15

---

## Executive Summary

Across Kubernetes, CockroachDB, OpenStack, Terraform, Crossplane, and other large infrastructure projects, a consistent CI architecture emerges:

1. **Fast mandatory gate** (lint + compile + affected tests, <15 min) on every PR
2. **Broader post-merge validation** (full unit + regression) on push to main
3. **Comprehensive nightly suites** (all tiers, all providers, stress tests)
4. **Smart test selection** via dependency graph analysis (not just path globs)
5. **Resource cleanup sweepers** for any CI that creates real infrastructure
6. **Flaky test management** with tracking, quarantine, and SLOs

The single highest-impact improvement for Juju is **Go dependency graph analysis** (DigitalOcean's GTA tool), which reduced their CI from 20 minutes to 2-3 minutes. Combined with the existing path-filter infrastructure, this would give Juju precise test selection without expensive ML infrastructure.

---

## 1. Architecture Patterns from Comparable Projects

### Tier Model (Universal)

Every major project uses a 3-4 tier model:

| Tier | Trigger | Time Budget | What Runs | Gate? |
|------|---------|-------------|-----------|-------|
| **Presubmit/PR** | PR push | <15 min | Lint + compile + affected unit tests + smoke | Required |
| **Postsubmit/Merge** | Push to main | <60 min | Full unit suite + regression | Required |
| **Periodic/Nightly** | Cron | Hours | All suites + stress + cross-version | Informational |
| **Release** | Tag/manual | Hours-days | Full E2E + soak + multi-provider | Release gate |

### Project-Specific Patterns

| Project | CI System | Key Innovation | Scale |
|---------|-----------|----------------|-------|
| **Kubernetes** | Prow + Tide | `run_if_changed` + optional contexts + `/test` commands | 10K+ jobs/day |
| **OpenStack** | Zuul | Speculative execution (test N PRs as if all pass) | Cross-project gating |
| **Terraform** | GitHub Actions | Sweepers for cloud resource cleanup + `TF_ACC` env gating | Thousands of acceptance tests |
| **CockroachDB** | Custom | Metamorphic testing + 5-day stress tests + TestEng team | 80K tests/PR |
| **Crossplane** | GitHub Actions | Uptest (manifest-driven lifecycle testing) + comment triggers | Per-provider CI |
| **Vitess** | GitHub Actions | Hermetic Docker-per-test + sharded E2E workflows | Full topology per test |

---

## 2. Smart Test Selection — Approaches Ranked by Applicability

### Tier 1: Go Dependency Graph Analysis (Highest Impact for Juju)

**Tool**: [DigitalOcean GTA](https://github.com/digitalocean/gta) (Go Transitive Analysis)

**How it works**:
1. Compares current branch against merge-base using git
2. Identifies changed Go packages
3. Builds reverse dependency graph using Go's static analysis
4. Outputs all affected package import paths (transitive)

**Real-world result**: DigitalOcean's CI dropped from **20 minutes to 2-3 minutes**.

**Integration pattern for Juju**:
```bash
# In CI: find affected packages and run only their tests
AFFECTED=$(gta -merge-base origin/main)
if [ -z "$AFFECTED" ]; then
  echo "No affected Go packages"
  exit 0
fi
go test -count=1 $AFFECTED
```

**Limitation**: When `core/` packages change, nearly everything is affected. Mitigation: use risk-based escalation — if >80% of packages affected, just run the full suite.

**Alternative tools**:
- `go list -deps` (built-in, requires custom scripting)
- [jharlap/affected](https://github.com/jharlap/affected) (similar to GTA)
- [loov/goda](https://github.com/loov/goda) (dependency visualization)

### Tier 2: Path-Based Filtering (Already Implemented, Needs Refinement)

Juju's `context-tests.yml` already uses `dorny/paths-filter@v3`. Current problem: most Go checks use `**.go` (any Go file), defeating the purpose.

**Recommended refinement** — replace `**.go` with domain-specific groups:
```yaml
filters:
  domain-application:
    - 'domain/application/**'
  domain-machine:
    - 'domain/machine/**'
  apiserver:
    - 'apiserver/**'
  cmd-juju:
    - 'cmd/juju/**'
  core:
    - 'core/**'
  workers:
    - 'internal/worker/**'
  k8s-provider:
    - 'internal/provider/kubernetes/**'
    - 'caas/**'
  infrastructure:
    - 'go.mod'
    - 'go.sum'
    - 'Makefile'
    - 'make_functions.sh'
```

**Escalation rules**:
- `core/**` or `infrastructure` changed → run full unit suite
- Only `cmd/juju/**` changed → run only client tests
- Only `domain/X/**` changed → run domain X tests + its facade tests

### Tier 3: Risk-Based Tiering

Classify changes by blast radius:

| Risk Level | Trigger | Test Scope |
|------------|---------|------------|
| **Low** | Docs, comments, test-only files | Lint only |
| **Medium** | Leaf packages (no dependents) | Affected package tests |
| **High** | Mid-layer packages | Affected + transitively dependent tests |
| **Critical** | `core/`, `go.mod`, DB schema, API schemas | Full unit suite + smoke tests |

### Tier 4: Coverage-Based Test Impact Analysis (Future)

**Datadog TIA** for Go: instruments tests with `orchestrion` + `dd-trace-go` to track which source files each test touches. On subsequent runs, skips tests whose covered files haven't changed.

**DIY approach**: Use Go's `-coverpkg` flag during nightly full runs to collect per-package coverage maps, then use them to filter PR tests.

**When to adopt**: After the dependency graph approach (Tier 1) is in place and you want even more precision.

### Tier 5: ML-Based Predictive Selection (Long-term)

Facebook's approach (ICSE 2019): gradient-boosted decision trees on historical test outcomes → 2x infrastructure cost reduction, 95%+ failure detection.

**Tools**: [Launchable](https://www.launchableinc.com/) (now CloudBees) commercializes this approach.

**When to adopt**: Only after accumulating months of per-commit test outcome data. Best applied to expensive integration/smoke tests, not unit tests.

---

## 3. Cost and Speed Optimization

### Parallelization

| Strategy | Tool | Impact |
|----------|------|--------|
| **Test sharding** (split packages across N runners) | GitHub Actions matrix + `go list` modular split | 8 shards: 45 min → 6-8 min wall-clock |
| **Timing-based sharding** (balance by historical duration) | [gotestsum](https://github.com/gotestyourself/gotestsum) + [go-test-split-action](https://github.com/hashicorp-forge/go-test-split-action) | Even load distribution |
| **Go parallelism tuning** | `-p 16 -parallel 128` (package + intra-package) | 2-3x speedup on multi-core runners |
| **Fail-fast** | `strategy.fail-fast: true` in matrix | Cancel other shards on first failure |

### Caching

| Cache Type | Size | Time Saved | Key Strategy |
|-----------|------|-----------|--------------|
| Go modules (`GOMODCACHE`) | 500MB-2GB | 1-3 min | Key on `go.sum` hash |
| Go build (`GOCACHE`) | 2-10GB | 3-8 min | Key on `go.sum` + `github.sha` with fallback |
| Docker layers | 1-5GB | 2-10 min | BuildKit `type=gha,mode=max` |

**Critical**: `actions/setup-go@v4+` caches modules and build cache automatically. Ensure it's enabled.

### Runner Infrastructure

| Option | Cost/min | Best For |
|--------|----------|----------|
| GitHub-hosted (standard 2-core) | $0.008 | Low volume, public repos |
| GitHub-hosted (xlarge 16-core) | $0.032 | Fast compilation, short wall-clock |
| Self-hosted (AWS on-demand) | $0.003 | Medium volume |
| Self-hosted (AWS spot) | $0.001 | High volume, cost-sensitive |
| [RunsOn](https://runs-on.com/) (AWS spot) | ~$0.001 | Drop-in GitHub-hosted replacement |
| GitHub arm64 runners | $0.005 | 37% cheaper than x64 |

**Right-sizing**:
- Unit tests: 4-8 vCPU (CPU-bound compilation)
- Integration tests: 8-16 vCPU, 32GB RAM (multiple processes)
- Race detector: 8+ vCPU, 32GB+ RAM (5-10x memory overhead)
- Linting: 2-4 vCPU (single-threaded tools)

### Pipeline Architecture

**Concurrency groups** (immediate win, zero cost):
```yaml
concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true
```
Saves 15-25% of CI minutes by canceling stale runs when new pushes arrive.

**Ordered stages** (fail fast):
```
Stage 1: Lint + compile (1-2 min) → fail fast
Stage 2: Affected unit tests (fan-out, 5-10 min) → fail fast
Stage 3: Smoke tests (10-15 min) → only if stage 2 passes
Stage 4: Regression tests (30-60 min) → post-merge only
```

**Merge queues** (GitHub native, GA since 2023):
- Batches PRs and tests them together
- Bisects on failure to find culprit
- 20-40% reduction in total CI runs on active repos

### Projected Cost for Juju (~20 PRs/day)

| Strategy | Wall-clock/PR | Cost/PR | Monthly |
|----------|--------------|---------|---------|
| Naive (run everything, GitHub-hosted) | 45 min | $0.36 | $216 |
| + Caching | 30 min | $0.24 | $144 |
| + Sharding (8x) | 6 min | $0.32 | $192 |
| + GTA affected-only | 3 min | $0.16 | $96 |
| + Self-hosted spot | 3 min | $0.02 | $12 |

---

## 4. Flaky Test Management

### Detection

| Approach | Tool | How It Works |
|----------|------|--------------|
| Historical analysis | [Buildkite Test Analytics](https://buildkite.com/test-analytics), [Datadog CI Visibility](https://docs.datadoghq.com/tests/) | Track pass/fail per test over time; flaky = fails >1% but <100% |
| Repeated runs | `go test -count=5` | Run each test N times; intermittent failure = flaky |
| Deflake on CI | `gotestsum --rerun-fails --rerun-fails-max-failures=3` | Re-run only failed tests |

### Quarantine Strategy

1. **Detect**: Test fails >X% over Y runs → flagged as flaky
2. **Quarantine**: Move to non-blocking CI job; file issue with 30-day fix SLO
3. **Track**: Dashboard showing quarantine size, mean time to fix
4. **Enforce**: Quarantined tests must be fixed or deleted within SLO

### Retry Policy

| Strategy | When |
|----------|------|
| No retry | Unit tests (deterministic) |
| Retry failed only | Integration tests with external deps |
| Retry entire job | Infrastructure flakiness (runner issues) |
| Max 2-3 retries | Never more; if needs more, quarantine it |

---

## 5. Resource Cleanup (Critical Gap in Juju)

Every project that creates real infrastructure has cleanup mechanisms. Juju currently lacks this.

### Terraform's Sweeper Pattern (Recommended for Juju)

```bash
# Scheduled cleanup job (hourly)
# 1. List all CI controllers older than 2 hours
for ctrl in $(juju controllers --format=json | jq -r '.controllers | keys[] | select(startswith("ci-"))'); do
  age=$(calculate_age "$ctrl")
  if [ "$age" -gt 7200 ]; then
    juju destroy-controller --force --no-wait --destroy-all-models "$ctrl"
  fi
done

# 2. Clean up MicroK8s namespaces
for ns in $(microk8s kubectl get ns -o name | grep "^namespace/ci-"); do
  microk8s kubectl delete "$ns" --force
done

# 3. Clean up orphaned LXD containers
for container in $(lxc list --format=json | jq -r '.[] | select(.name | startswith("ci-")) | .name'); do
  lxc delete --force "$container"
done
```

### Naming Convention

All CI-created resources must use a prefix: `ci-{pr-number}-{suite}-{timestamp}`

Example: `ci-1234-smoke-1708012800`

This enables:
- Safe bulk cleanup (only delete `ci-*` resources)
- Age-based cleanup (parse timestamp)
- Debugging (identify which PR/suite created a resource)

---

## 6. Developer Experience Features

### Comment-Triggered Tests (From Kubernetes + Crossplane)

Add `/test <suite>` and `/retest` commands to PRs:

```yaml
# Triggered by PR comment
on:
  issue_comment:
    types: [created]

jobs:
  test-on-demand:
    if: |
      github.event.issue.pull_request &&
      startsWith(github.event.comment.body, '/test ')
    steps:
      - name: Parse suite name
        run: echo "SUITE=$(echo '${{ github.event.comment.body }}' | sed 's|/test ||')" >> $GITHUB_ENV
      - name: Run requested suite
        run: cd tests && ./main.sh "$SUITE"
```

### Required vs Optional Contexts (From Kubernetes Tide)

| Context | On PR | On Push | On Nightly |
|---------|-------|---------|------------|
| lint + compile | Required | Required | Required |
| Affected unit tests | Required | Required | Required |
| Full unit suite | Optional | Required | Required |
| Smoke tests (K8s) | Required | Required | Required |
| Regression tests | Optional (informational) | Required | Required |
| Integration tests | Skip | Skip | Required |
| Race detector | Skip | Skip | Required |
| Cross-provider | Skip | Skip | Required |

**Key insight from Kubernetes**: Optional tests provide signal without blocking merge. Developers can see regression test results on their PR, but a flaky regression test doesn't block the entire team.

### Test Result Dashboard

Minimum viable dashboard tracks:
- Pass/fail history per suite (7-day rolling)
- Flakiness rate per test
- Mean CI time per PR
- Quarantine size over time

Tools: [TestGrid](https://github.com/kubernetes/test-infra/tree/master/testgrid) (K8s), gotestsum JUnit XML → Grafana, or GitHub Pages with simple charts.

---

## 7. Recommended Architecture for Juju

### Phase 1: Foundation (Weeks 1-4)

1. **Refine path filters** — Replace `**.go` with domain-specific groups in `context-tests.yml`
2. **Add concurrency groups** — Cancel stale runs on new pushes
3. **Implement fail-fast staging** — Lint → compile → unit tests → smoke
4. **Add `gotestsum --rerun-fails`** — Handle flaky tests without blocking

### Phase 2: Smart Selection (Weeks 5-8)

5. **Integrate GTA** — Add Go dependency analysis to the changed-files job
6. **Test sharding** — Split unit tests across 8 matrix runners with timing-based distribution
7. **Predicate YAML** — Declare per-suite activation conditions in a declarative file
8. **Risk-based escalation** — `core/` changes → full suite; leaf changes → affected only

### Phase 3: Infrastructure (Weeks 9-12)

9. **Resource cleanup sweepers** — Hourly cleanup of leaked CI controllers/namespaces
10. **CI naming convention** — `ci-{pr}-{suite}-{timestamp}` for all resources
11. **Comment-triggered tests** — `/test <suite>` and `/retest` commands
12. **Required vs optional contexts** — Mark regression tier as optional on PRs

### Phase 4: Optimization (Ongoing)

13. **Flaky test dashboard** — Track and quarantine unreliable tests
14. **Coverage-based TIA** — Nightly coverage maps for precise PR test selection
15. **Self-hosted runners** — Move to spot instances for cost reduction
16. **Merge queue** — Enable GitHub merge queue with batching

---

## 8. Key Tools Reference

| Tool | Purpose | URL |
|------|---------|-----|
| **GTA** | Go dependency-based affected package detection | https://github.com/digitalocean/gta |
| **gotestsum** | Go test runner with JUnit, rerun-fails, timing | https://github.com/gotestyourself/gotestsum |
| **go-test-split-action** | Timing-based test sharding for GH Actions | https://github.com/hashicorp-forge/go-test-split-action |
| **dorny/paths-filter** | Path-based filtering (already used by Juju) | https://github.com/dorny/paths-filter |
| **RunsOn** | Cheap self-hosted runners on AWS | https://runs-on.com |
| **actions-runner-controller** | K8s-native GH runner autoscaling | https://github.com/actions/actions-runner-controller |
| **Buildkite Test Analytics** | Test timing + flaky detection | https://buildkite.com/test-analytics |
| **Codecov Test Analytics** | Flaky test detection (free for OSS) | https://about.codecov.io/product/feature/test-analytics/ |
| **Datadog CI Visibility** | CI analytics + Go TIA | https://docs.datadoghq.com/tests/ |
| **Launchable** | ML-based predictive test selection | https://www.launchableinc.com |

---

## Sources

### Large Go Project CI
- [Prow Jobs Documentation](https://docs.prow.k8s.io/docs/jobs/)
- [CockroachDB Test Engineering](https://www.cockroachlabs.com/blog/test-engineering-team/)
- [Vitess E2E Test Migration](https://planetscale.com/blog/planetscale-migrates-open-source-vitess-test-suite-from-python-to-go)
- [etcd Robustness Tests](https://github.com/etcd-io/etcd/blob/main/tests/robustness/README.md)
- [Mattermost: Cutting Test Runtime by 60%](https://mattermost.com/blog/cutting-test-runtime-by-60-with-selective-parallelism-in-go/)

### Smart Test Selection
- [GTA: Go Transitive Analysis (DigitalOcean)](https://www.digitalocean.com/blog/gta-detecting-affected-dependent-go-packages)
- [The Rise of Test Impact Analysis (Martin Fowler)](https://martinfowler.com/articles/rise-test-impact-analysis.html)
- [Predictive Test Selection (Facebook, ICSE 2019)](https://arxiv.org/abs/1810.05286)
- [Datadog Test Impact Analysis for Go](https://docs.datadoghq.com/tests/test_impact_analysis/setup/go/)
- [Pipeline-Aware Regression Test Optimization (ICST 2025)](https://arxiv.org/abs/2501.11550)

### Cost Optimization
- [Go Test Parallelism (Three Dots Labs)](https://threedots.tech/post/go-test-parallelism/)
- [Better GitHub Actions Caching for Go](https://danp.net/posts/github-actions-go-cache/)
- [GitHub Actions 2026 Pricing Changes](https://resources.github.com/actions/2026-pricing-changes-for-github-actions/)
- [Self-Hosted Runners 54% Faster at 13% Cost](https://zenn.dev/team_soda/articles/b2783d8e009104)

### Orchestrator CI Patterns
- [Prow + Tide for Contributors](https://www.kubernetes.dev/blog/2022/12/12/prow-and-tide-for-kubernetes-contributors/)
- [Zuul Project Gating](https://zuul-ci.org/docs/zuul/latest/gating.html)
- [Terraform Acceptance Testing](https://developer.hashicorp.com/terraform/plugin/testing/acceptance-tests)
- [Terraform Sweepers](https://developer.hashicorp.com/terraform/plugin/sdkv2/testing/acceptance-tests/sweepers)
- [Crossplane Uptest](https://github.com/crossplane/uptest)
- [Testing K8s Operators with envtest](https://www.infracloud.io/blogs/testing-kubernetes-operator-envtest/)
