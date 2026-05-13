# `juju citizen` live demo script

**Saved**: 2026-05-13. This file is the resumption pointer after a
session compaction.

## Where everything is

### Repos and branches

| Repo | Path | Branch | Fork remote |
|---|---|---|---|
| Juju | `/home/sinan.awad@canonical.com/dev/juju` | `003-juju-citizen-cli` | `github.com/sinanawad/juju` |
| Norma charm | `/home/sinan.awad@canonical.com/dev/juju-norma-k8s` | `001-calibration-charm` | `github.com/sinanawad/juju-norma-k8s` |

Most recent commits at save time:
- Juju: `e51de1e054` feat(citizen): add entity-stuck-dying and model-suspended-credential detectors
- Norma: `5810b5f` feat(test-bed): add stuck-dying bad-citizenship mode

### Built artifacts

| Artifact | Path | Status |
|---|---|---|
| `juju` client | `~/go/bin/juju` | Built via `make juju`. Contains all 9 detectors + dashboard table format + verbose snazzy format. |
| Norma charm | `~/dev/juju-norma-k8s/juju-norma-k8s_amd64.charm` | Built via `charmcraft pack`. Contains 8 bad-citizenship modes including `stuck-dying`. |
| Norma ROCK | `localhost:32000/juju-norma:0.1.0` | Pushed to microk8s registry. No changes needed; existing image is fine. |

### Live controller state

- Controller: `mk8s-controller` on cloud `mk8s` (microk8s).
- Model: `norma-demo` (a CAAS workload model).
- `update-status-hook-interval` set to `30s` (lowered for fast churn demo).
- microk8s registry is up at `localhost:32000`.
- User is now in `lxd` group (rockcraft/charmcraft work without `--destructive-mode`).

### Currently deployed in `norma-demo`

| App | bad-citizenship-mode | Expected citizen finding |
|---|---|---|
| `juju-norma-k8s` | `none` (good citizen) | none |
| `bad-active` | `active-with-message` | INFO active-with-message |
| `bad-blocked` | `blocked-no-message` | WARNING blocked-no-message |
| `bad-churn` | `status-churn` | WARNING status-churn |
| `bad-stuck` | `stuck-maintenance` | WARNING stuck-maintenance |

= **4 findings** in `juju citizen` baseline.

## The 9 detectors

| # | check_id | severity | type | clause | Notes |
|---|---|---|---|---|---|
| 1 | active-with-message | info | pure | §4c.2 | Empty active-msg convention |
| 2 | charm-revision-aging | warning | pure | §4b | CanUpgradeTo populated |
| 3 | unit-blocked-stale | warning/critical | pure | §4c.2 | Blocked >24h |
| 4 | blocked-no-message | warning | pure | §4c.2 | Blocked + empty Info |
| 5 | status-churn | warning | stateful | §4c.2 | 3+ workload transitions in 10m |
| 6 | stuck-maintenance | warning | stateful | §4c.2 | Maintenance run >5m |
| 7 | hook-error | warning | stateful | §4c.1 | Agent error in last 30m |
| 8 | entity-stuck-dying (NEW) | warning/critical | pure | §4c.1 | Life=Dying >5m / >1h |
| 9 | model-suspended-credential (NEW) | critical | pure | §4b | Model.ModelStatus=suspended |

## Demo arc

User runs `juju citizen` in a split terminal while I narrate and dispatch deployments.

### Stage 0 — baseline (already live)

```bash
~/go/bin/juju citizen                       # 4 findings, pretty table
~/go/bin/juju citizen --format=verbose      # same findings, arrow-notes, AI-enriched
~/go/bin/juju citizen --format=json | jq .  # 4 JSON objects
```

### Stage 1 — add `bad-hooky` (hook-error)

```bash
CHARM=/home/sinan.awad@canonical.com/dev/juju-norma-k8s/juju-norma-k8s_amd64.charm
RES="--resource juju-norma-image=localhost:32000/juju-norma:0.1.0"

~/go/bin/juju deploy "$CHARM" bad-hooky $RES --trust \
    --config bad-citizenship-mode=hook-error
```

Expected timing:
- t≈0: deploy issued
- t≈30s–60s: pod scheduled, charm install runs, raises RuntimeError
- t≈90s: agent status = `error`
- t≈90s: `juju citizen` shows 5 findings (4 existing + WARNING hook-error on bad-hooky/0)

### Stage 2 — add `bad-rip` (stuck-dying) and wedge it

```bash
~/go/bin/juju deploy "$CHARM" bad-rip $RES --trust \
    --config bad-citizenship-mode=stuck-dying

# wait until bad-rip/0 is active+idle (~1-2 min)
~/go/bin/juju status --watch 5s

# then trigger the wedge
~/go/bin/juju remove-application bad-rip
```

Expected timing after `remove-application`:
- Within seconds: bad-rip/0 transitions to Life=Dying. The first departure hook raises RuntimeError. Agent=failed.
- t+5m: `entity-stuck-dying` detector fires WARNING on bad-rip/0.
- t+1h: severity escalates to CRITICAL.

The unit will remain in Dying. To clean up after the demo:
```bash
~/go/bin/juju config bad-rip bad-citizenship-mode=none
~/go/bin/juju resolve bad-rip/0           # retries the failed teardown hook
# remove-application should then complete
```

### Stage 3 — multi-issue on `bad-churn` (proposal — pending approval)

Currently the `status-churn` mode alternates `active()` and
`waiting("waiting for nothing in particular")`. To get **two
simultaneous findings on the same entity**, alter to alternate:
- `active("operational")` — fires active-with-message + status-churn
- `blocked("")` — fires blocked-no-message + status-churn

Tweak in `src/charm.py` inside `_bad_citizenship_unit_status()`, the
`status-churn` branch:

```python
if mode == "status-churn":
    if len(self._event_ledger) % 2 == 0:
        return ops.ActiveStatus("operational")
    return ops.BlockedStatus("")
```

Then `charmcraft pack` + `juju refresh bad-churn --path ...` to roll
out. After update-status cycle (≤30s), bad-churn/0 will start showing
2 findings simultaneously in `juju citizen`.

### Stage 4 — model-suspended-credential (skipped live)

Cannot be triggered without breaking the controller's actual cloud
credential. Defer to:
- Unit test demonstration: `go test -v -run TestModelSuspendedCredential ./cmd/juju/citizen/...`
- Talk-track narrative: "ships in this PR; needs real credential breakage to demo"

### Final dashboard prediction (after Stages 1+2+3)

```
findings: 7   ● 0 critical   ▲ 6 warning   ◆ 1 info
owners:       AUTHOR 6  •  OP 0  •  MIX 1  •  PLAT 0

  ▲ warn    bad-rip/0          MIX     entity-stuck-dying      ~5m   Entity has been in Dying state for over 5 minutes.
  ▲ warn    bad-stuck/0        AUTHOR  stuck-maintenance       ~30m  Unit has held maintenance status without transition.
  ▲ warn    bad-churn/0        AUTHOR  status-churn            ~10m  Unit workload status is churning between values.
  ▲ warn    bad-churn/0        AUTHOR  blocked-no-message        —   Unit is blocked without an actionable status message.    ★ multi-issue
  ▲ warn    bad-blocked/0      AUTHOR  blocked-no-message        —   Unit is blocked without an actionable status message.
  ▲ warn    bad-hooky/0        AUTHOR  hook-error              ~3m   Unit hit an uncaught hook error recently.
  ◆ info    bad-active/0       AUTHOR  active-with-message       —   Unit reports 'active' with a non-empty status message.

  Tip: run with --format=verbose for per-finding recommendations.
```

★ = bad-churn/0 carries TWO findings simultaneously (post-Stage 3).

## Useful commands

```bash
# Build + reinstall juju client
cd /home/sinan.awad@canonical.com/dev/juju && make juju

# Run citizen with different formats
~/go/bin/juju citizen
~/go/bin/juju citizen --format=verbose
~/go/bin/juju citizen --format=json | jq .
~/go/bin/juju citizen --no-color
~/go/bin/juju citizen --severity=warning,critical

# Run tests (57 total)
cd /home/sinan.awad@canonical.com/dev/juju && go test -race ./cmd/juju/citizen/... -count=1

# Pack the norma charm
cd /home/sinan.awad@canonical.com/dev/juju-norma-k8s && charmcraft pack

# Inspect a deployed unit
~/go/bin/juju status --format=json | jq '.applications | to_entries[] | {app: .key, life: .value."application-status", units: (.value.units // {} | to_entries | map({u: .key, life: .value."workload-status".life}))}'

# Force a status from outside (for ad-hoc demo)
~/go/bin/juju exec --unit bad-active/0 -- status-set active "demo: serving"

# Clean up a stuck-dying unit
~/go/bin/juju config bad-rip bad-citizenship-mode=none
~/go/bin/juju resolve bad-rip/0
~/go/bin/juju remove-application bad-rip

# Destroy the model entirely (when fully done)
~/go/bin/juju destroy-model norma-demo --no-prompt --destroy-storage
# (do NOT use --force; known K8s destroy-deadlock bug)
```

## Open follow-ups after demo

- Polish: per-entity grouping in the table (when one entity carries multiple findings, indent the second+ row under it).
- Detector v1.x: actually trigger model-suspended-credential live (needs a sandbox controller we don't mind breaking).
- Detector v1.x: relation-suspended-or-broken (Discourse signal from the research pass).
- Detector v1.x: unit-waiting-on-relation-stale (Launchpad signal).
- Constitutional Principle IX migration: copy the detectors into a `Health` facade for v1 server-side detection.

## Quick context for whoever picks this up

- The project constitution lives at `.specify/memory/constitution.md` (v1.0.0, 10 principles).
- The full spec/plan/tasks tree is at `specs/003-juju-citizen-cli/`.
- The design doc for the table format is `docs/superpowers/specs/2026-05-13-citizen-table-format-design.md`.
- Three research agents surveyed Launchpad/GitHub/Discourse — their findings drove the choice of `entity-stuck-dying` and `model-suspended-credential` as the new detectors. Notes in conversation history (and the synthesis in this file's "9 detectors" table is the durable summary).
- The user has a competition deadline — favor velocity over perfection in any further changes.
