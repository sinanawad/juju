# Juju GitHub Actions CI Reference

> Generated for spec authoring reference. Describes the GitHub Actions
> workflows under `.github/workflows/` and their relationship to the
> bash integration test suite (`tests/`).

## Architecture at a Glance

```
                        ┌─────────────────────────┐
                        │   context-tests.yml      │  ← orchestrator
                        │   (path-based gating)    │
                        └────┬──┬──┬──┬──┬──┬──┬──┘
                             │  │  │  │  │  │  │  │
              ┌──────────────┘  │  │  │  │  │  │  └──────────────┐
              ▼                 ▼  ▼  ▼  ▼  ▼  ▼                 ▼
          build.yml        gen  docs snap ddl upgrade     terraform-smoke
          client-tests                        migrate(off)

    ┌─────────────┐    ┌──────────────┐    ┌───────────────┐
    │ smoke.yml   │    │ upgrade.yml  │    │ static-       │  ← independent
    │ [lxd,mk8s] │    │ [lxd,mk8s]  │    │ analysis.yml  │
    └─────────────┘    └──────────────┘    └───────────────┘

    ┌─────────────────┐  ┌──────────────┐  ┌─────────┐
    │ postgresql-k8s  │  │ jaas-smoke   │  │ cla.yml │  ← standalone
    └─────────────────┘  └──────────────┘  └─────────┘
```

---

## Workflow Inventory (19 files)

### Orchestrator

#### `context-tests.yml` — Conditional Test Router
- **Trigger**: Push to main/release branches, PRs (non-draft)
- **How it works**: Uses `dorny/paths-filter@v3` to detect changed files, then calls reusable workflows only when relevant paths changed
- **Path filters** (18 total):
  - `check-build`: `**.go`, `go.mod`, `Makefile`, `scripts/dqlite/**`
  - `check-client`: Same + `.github/workflows/client-tests.yml`
  - `check-generate`: `**.go`, `go.mod`
  - `check-docs`: `**.go`, `go.mod`
  - `check-snap`: `**.go`, `go.mod`, `snap/*`
  - `check-upgrade`: Same + `.github/setup-lxd/**`
  - `check-terraform`: `**.go`, `go.mod`
  - `check-ddl`: `go.mod`, `domain/schema/**`, `core/database/**`, `internal/database/**`
- **Final gate**: `result-checker` job aggregates all conditional results

---

### Build & Compilation

#### `build.yml` — Cross-Platform Build
- **Trigger**: Reusable (`workflow_call`) + manual
- **Runner**: Self-hosted, xxlarge
- **Matrix**: `[linux/amd64, linux/arm64, linux/s390x, darwin/amd64, darwin/arm64, windows/amd64]`
- **Target**: `make go-build` with `GOOS`/`GOARCH`
- **Deps**: DQLite dev, libsqlite3, musl tools

#### `client-tests.yml` — macOS Client Tests
- **Trigger**: Reusable + manual
- **Runner**: macOS-latest
- **Target**: `make run-go-tests` on `cmd/juju/**` packages
- **Timeout**: 15 minutes per test

#### `snap.yml` — Snapcraft Build
- **Trigger**: Reusable + manual
- **Runner**: Self-hosted, quad-xlarge
- **What it does**: Builds snap package, installs, verifies `juju version`

---

### Code Quality (Always Required)

#### `static-analysis.yml` — Linting & Security
- **Trigger**: Push + PR (all branches, no path filter — always runs)
- **Jobs**:
  1. **checks** (arm64/quad-xlarge): `make static-analysis` (golangci-lint v2.6.1, govulncheck, shfmt)
  2. **sql** (x64/large): SQLFluff v3.0.7 on `domain/schema/**/sql/*.sql`
  3. **conventional-commits** (ubuntu-latest): commitlint on PR titles

#### `ddl.yml` — Database Schema Validation
- **Trigger**: Reusable + manual
- **Runner**: ubuntu-latest
- **What it does**: Validates DDL changes don't mutate released patch versions
- **Scope**: `domain/schema/**`, `core/database/**`, `internal/database/**`

#### `gen.yml` — Code Generation Verification
- **Trigger**: Reusable + manual
- **Runner**: Self-hosted, arm64/xxlarge
- **What it does**: Runs `go generate` (8 parallel workers), fails if output differs from committed code

---

### Integration Tests

#### `smoke.yml` — Smoke Tests (LXD + MicroK8s)
- **Trigger**: Push to main/release, PRs (non-draft), manual
- **Runner**: Self-hosted, x64/quad-xlarge
- **Matrix**: `cloud: ["localhost", "microk8s"]`
- **Timeout**: 60 minutes
- **Jobs**:
  - **smoke**: Runs `tests/main.sh` smoke suite
    - LXD: `./main.sh smoke`
    - MicroK8s: `./main.sh -c microk8s smoke_k8s`
  - **deploy**: Runs `tests/main.sh` deploy suite
    - LXD only: `./main.sh deploy`
    - MicroK8s: skipped (`SKIP_DEPLOY_microk8s: "true"`)
- **K8s Setup**: `balchua/microk8s-actions` (channel `1.34-strict/stable`), addons: dns, hostpath-storage, rbac
- **Key envs**: `SKIP_DESTROY=true`, `BOOTSTRAP_ADDITIONAL_ARGS=--model-default enable-os-upgrade=false`
- **Failure artifacts**: `juju-debug.log`, `microk8s inspect`

#### `upgrade.yml` — Upgrade Tests
- **Trigger**: Reusable + manual
- **Runner**: Self-hosted, x64/quad-xlarge
- **Matrix**: `cloud: ["localhost", "microk8s"]`
- **Timeout**: 30 minutes
- **What it does**: Tests controller + model upgrade paths
  - LXD: apache2 charm (IaaS upgrade)
  - MicroK8s: snappass-test charm (CaaS upgrade) with custom OCI registry
- **K8s extras**: Self-signed CA, local registry (`10.152.183.69`), podman for image builds, `make microk8s-operator-update`
- **Condition**: Only on PRs not targeting main (`github.base_ref != 'main'`)

#### `postgresql-k8s.yml` — PostgreSQL K8s Smoke
- **Trigger**: Push to main/release, manual
- **Runner**: Self-hosted, x64/quad-xlarge
- **Matrix**: `cloud: ["microk8s"]` only
- **What it does**: `tests/main.sh -c microk8s smoke_k8s_psql`

#### `microk8s-tests.yml` — Kubeflow (Manual Only)
- **Trigger**: `workflow_dispatch` only (not automated)
- **Runner**: Self-hosted, arm64/large
- **What it does**: Full Kubeflow bundle deployment on MicroK8s
- **Notes**: Aggressive disk cleanup (14GB runner limit), 90-minute deploy timeout

#### `terraform-smoke.yml` — Terraform Provider
- **Trigger**: Reusable + manual
- **Runner**: Self-hosted, x64/quad-xlarge
- **Status**: Disabled in context-tests (`if: false`)

#### `migrate.yml` — Model Migration
- **Trigger**: Reusable + manual
- **Runner**: Self-hosted, arm64/xlarge
- **Status**: Disabled (`if: false` — migration infra broken for Juju 4)

---

### Other Workflows

#### `jaas-smoke.yml` — JAAS Integration
- **Trigger**: Push to release branches + PRs, manual
- **Runner**: Self-hosted, x64/large
- **What it does**: Tests Juju against JIMM (multi-cloud management)
- **External dep**: `canonical/jimm/.github/actions/test-server@v3`

#### `docs.yml` — Sphinx Documentation
- **Trigger**: Reusable + manual
- **What it does**: Builds Sphinx docs, verifies rendering

#### `cla.yml` — Contributor License Agreement
- **Trigger**: PR events
- **What it does**: Verifies canonical CLA signature

#### `merge.yml` — Branch Merge Monitor
- **Trigger**: Push to release branches (2.9, 3.x, 4.0)
- **What it does**: Detects merge conflicts, notifies via Mattermost

#### `docs-sphinx-python-dependency-build-checks.yml`
- **Trigger**: Scheduled (Mon + Fri 02:00 UTC), manual
- **What it does**: Python dependency version verification for docs toolchain

---

## Relationship: GitHub Actions ↔ Test Suites

### Which suites run in CI?

| Workflow | Suite(s) Run | Provider | Automated? |
|----------|-------------|----------|------------|
| smoke.yml | `smoke` | LXD | Yes (push/PR) |
| smoke.yml | `smoke_k8s` | MicroK8s | Yes (push/PR) |
| smoke.yml | `deploy` | LXD | Yes (push/PR) |
| postgresql-k8s.yml | `smoke_k8s_psql` | MicroK8s | Yes (push to main) |
| upgrade.yml | Custom (not suite-based) | LXD + MicroK8s | Yes (PR, not-main) |
| microk8s-tests.yml | `kubeflow` | MicroK8s | Manual only |

### Which suites are NOT in CI?

These 44+ suites exist in `tests/suites/` but have **no automated GitHub Actions trigger**:

| Suite | Category | Notes |
|-------|----------|-------|
| `deploy_caas` | K8s deployment | Not in any automated workflow |
| `storage_k8s` | K8s storage | Not automated |
| `sidecar` | K8s sidecar charms | Not automated |
| `caasadmission` | K8s admission | Not automated |
| `secrets_k8s` | K8s secrets | Not automated |
| `coslite` | K8s COS Lite | Not automated |
| `ck` | Charmed K8s | Not automated |
| `deploy_aks` | AKS | Disabled (TODO in code) |
| `constraints` | Resource constraints | Not automated |
| `storage` | IaaS storage | Not automated |
| `resources` | Charm resources | Not automated |
| `relations` | Charm relations | Not automated |
| `cmr` | Cross-model relations | Not automated |
| `secrets_iaas` | IaaS secrets | Not automated |
| `hooks` | Hook dispatch | Not automated |
| `hooktools` | Hook tools | Not automated |
| `actions` | Charm actions | Not automated |
| `model` | Model lifecycle | Not automated |
| `controller` | Controller ops | Not automated |
| `bootstrap` | Bootstrap | Not automated |
| `agents` | Agent ops | Not automated |
| `network` | Network health | Not automated |
| `spaces_ec2` | EC2 spaces | Not automated |
| `spaces_gce` | GCE spaces | Not automated |
| `cloud_azure` | Azure features | Not automated |
| `cloud_gce` | GCE features | Not automated |
| `firewall` | Firewall rules | Not automated |
| `user` | User management | Not automated |
| `authorized_keys` | SSH keys | Not automated |
| `credential` | Credentials | Not automated |
| `cli` | CLI commands | Not automated |
| `charmhub` | Charmhub | Not automated |
| `refresh` | Charm refresh | Not automated |
| `dashboard` | Dashboard | Not automated |
| `appdata` | Application data | Not automated |
| `examples` | Examples | Not automated |
| `machine` | Machine ops | Not automated |
| `manual` | Manual provider | Not automated |
| `unmanaged` | Unmanaged provider | Not automated |

> These suites are likely run in a separate CI system (e.g., Jenkins, internal
> Canonical CI) or manually before releases. The GitHub Actions cover only
> the fast-feedback loop for PRs.

---

## Runner Infrastructure

| Label | Size | Used By |
|-------|------|---------|
| `quad-xlarge` | 4x large | smoke, upgrade, snap, terraform-smoke, static-analysis |
| `xxlarge` | 2x large | build, gen |
| `xlarge` | Extra large | migrate |
| `large` | Large | docs, jaas-smoke, microk8s-tests, SQL lint |
| `ubuntu-latest` | GitHub-hosted | cla, merge, ddl, conventional-commits |
| `macOS-latest` | GitHub-hosted | client-tests |

Architecture: `x64` (default), `arm64` (static-analysis, gen, microk8s-tests, migrate)

---

## K8s-Specific CI Setup

### MicroK8s Bootstrap in Workflows
```yaml
# Action setup
- uses: balchua/microk8s-actions@13f...
  with:
    channel: '1.34-strict/stable'
    addons: '["dns", "hostpath-storage", "rbac"]'

# Docker cache mirror (AWS runners)
- name: Setup Docker Mirror
  env:
    GH_MIRROR: docker-cache.us-west-2.aws.jujuqa.com:443
```

### OCI Registry for Upgrade Tests
The `upgrade.yml` K8s path creates a local OCI registry for operator images:
1. Generate self-signed CA + certificates
2. Deploy registry as K8s pod (`ghcr.io/distribution/distribution:edge`)
3. Configure MicroK8s containerd to trust the CA
4. Build + push jujud-operator image via `make microk8s-operator-update`
5. Bootstrap with `--config caas-image-repo=${OCI_REGISTRY}/test-repo`

### Test Execution Pattern
```bash
# Install juju from source
make go-install

# Run K8s test suite
cd tests
sg snap_microk8s './main.sh -c microk8s -v smoke_k8s'
#  └─ sg = run as snap_microk8s group (for microk8s socket access)
#      -c microk8s = set BOOTSTRAP_CLOUD=microk8s
#      -v = verbose
```

---

## Disabled / Gated Workflows

| Workflow | Status | Reason |
|----------|--------|--------|
| `migrate.yml` | `if: false` | Migration infra broken for Juju 4 |
| `terraform-smoke.yml` | `if: false` in context-tests | Reliability issues on 3.x branches |
| `deploy_aks` (suite) | Skipped in code | Pending k8s tooling in strict snap |
| `microk8s-tests.yml` | Manual only | Too heavy for automated PR checks |

---

## Relevance for 001-k8s-deployment-types

### What runs automatically on our PRs

1. **`static-analysis.yml`** — Always. Our Go changes must pass golangci-lint, SQLFluff (if we add schema SQL), and conventional commits.
2. **`context-tests.yml`** → **`build.yml`** — Triggered by `.go`/`go.mod` changes. Cross-platform compilation must succeed.
3. **`context-tests.yml`** → **`ddl.yml`** — Triggered by `domain/schema/**` changes. Our new `deployment_type` table/column must not mutate released patches.
4. **`context-tests.yml`** → **`gen.yml`** — If we add `go generate` directives.
5. **`smoke.yml`** — Both LXD and MicroK8s. The `smoke_k8s` suite runs on every PR — our changes must not break the default StatefulSet path.
6. **`upgrade.yml`** — Both LXD and MicroK8s (for PRs not targeting main). Controller/model upgrade with our schema changes.
7. **`postgresql-k8s.yml`** — On push to main. PostgreSQL charm on K8s must keep working.

### Gaps in automated coverage

| Gap | Impact | Mitigation |
|-----|--------|------------|
| `deploy_caas` not in CI | Core K8s deploy tests not automated on PRs | Must run manually |
| `storage_k8s` not in CI | K8s storage regression risk | Must run manually |
| `sidecar` not in CI | Sidecar charm regression risk | Must run manually |
| No deployment-type test suite | New functionality entirely untested in CI | Create new suite + add to smoke.yml |
| `constraints` skips K8s | No K8s constraint testing exists | Add K8s constraint tests |

### Recommendations for our feature

1. **Add `deployment-type` tests to `smoke_k8s`** — This suite runs on every PR via `smoke.yml`. Adding a basic Deployment/DaemonSet deploy-and-verify test here gives us automated regression coverage.
2. **Create `deploy_caas_deployment_type` suite** — For comprehensive testing (scale, storage, lifecycle). Run manually or add to smoke.yml matrix.
3. **DDL compliance** — Our new schema migration in `domain/schema/` will be validated by `ddl.yml` automatically.
4. **Schema SQL linting** — Any `.sql` files we add under `domain/schema/` will be linted by SQLFluff.
