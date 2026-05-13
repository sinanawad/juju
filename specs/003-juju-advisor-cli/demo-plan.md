# Demo plan: `juju advisor` against `juju-norma-k8s`

**Purpose**: live, code-driven demonstration of the advisor protocol
observatory. We start from a textbook-compliant-charm charm and iterate
through one *deliberate* bad-compliance regression at a time. Each
iteration is a single-line (or near-single-line) edit to `src/charm.py`,
followed by a rebuild + refresh, followed by a `juju advisor` run that
detects exactly the regression we introduced.

## Why this charm

`juju-norma-k8s` is a Canonical-authored calibration charm that
*intentionally* follows every well-known compliance convention:

- `collect_unit_status` event handler (canonical ops pattern).
- `event.add_status(ops.ActiveStatus())` with empty message (textbook
  compliant charm, §4c.2 of the protocol brief).
- `BlockedStatus` is set only with an actionable message
  (`BlockedStatus(error_msg)` on config-validation failure, etc.) —
  matches §4c.2 "blocked MUST carry actionable message".
- `WaitingStatus("Waiting for Pebble")` — non-empty message on a
  non-`active` state is convention-compliant.

So when we run `juju advisor` against the unmodified charm, the expected
output is `No findings.` That is the **baseline frame** for
every iteration: if advisor ever fires while the charm is unmodified,
either advisor has a false-positive or the charm itself drifted.

## Tooling state (verified 2026-05-13)

| Tool | Path | Status |
|---|---|---|
| `juju` (with `advisor`) | `~/go/bin/juju` | ✅ rebuilt via `make juju` |
| `uv` | `~/.local/bin/uv` | ✅ |
| `charmcraft` | — | ❌ needs `sudo snap install charmcraft --classic` |
| `rockcraft` | — | ❌ needs `sudo snap install rockcraft --classic` |
| `skopeo` | `rockcraft.skopeo` | ✅ ships inside rockcraft snap (no separate install) |
| microk8s registry on `localhost:32000` | — | ⚠ check `microk8s enable registry` |

Install gate before iteration 0: snap-install `charmcraft` and
`rockcraft`. Skopeo is included.

## Iteration sequence

We iterate **one violation at a time**. Each iteration follows the same
five steps so the demo cadence stays recognizable:

1. **Show the diff** (`git diff src/charm.py` — one-liner each time).
2. **Build** (`charmcraft pack` — incremental, ~30s after the first
   build because uv caches dependencies).
3. **Refresh** (`juju refresh juju-norma-k8s --path
   ./juju-norma-k8s_ubuntu-24.04-amd64.charm`).
4. **Run `juju advisor`** — expect *exactly* the finding the edit
   targets.
5. **Revert** (`git checkout src/charm.py`) — advisor returns to clean,
   showing the symptom is repairable.

### Iteration 0 — clean baseline

**No edit.** Build and deploy the charm as-is. Confirm:

```text
$ juju status
juju-norma-k8s/0   active   idle   ...
$ juju advisor
No findings.
```

This proves the charm is a compliant charm and advisor is not flagging it
spuriously. **Drop-out point**: if iteration 0 doesn't pass, we stop
and diagnose before introducing any violations.

### Iteration 1 — Signal 1: active-with-message

**Edit point**: `src/charm.py:400`.

**The mistake we're simulating**: the charm author thinks "I'll surface
a useful runtime value as the active-status message so operators can
see what's happening." This is the most common §4c.2 violation in the
wild — captured by the brief at line 282.

**Diff**:

```python
# BEFORE (charm.py:400) — compliant charm, empty message
event.add_status(ops.ActiveStatus())

# AFTER — misbehaving charm, helpful-looking but convention-breaking
port = int(self.config.get("calibration-int", norma.DEFAULT_PORT))
event.add_status(ops.ActiveStatus(f"serving on port {port}"))
```

**Expected `juju advisor` output**:

```text
INFO     juju-norma-k8s/0 [active-with-message]
   -> Unit reports 'active' with a non-empty status message.
   -> Active charms should carry an empty status message...
```

**Why this is the headline iteration**: `juju status` will show
`active: serving on port 8080`, which to an unfamiliar operator looks
**helpful**. Operators routinely write status parsers that latch onto
the message field. The convention violation is invisible without
advisor. This is the strongest single-shot demonstration of the
observatory's value.

### Iteration 2 — Signal 1 stacked: also fire on app-level status

**Edit point**: `src/charm.py:408` (in `_on_collect_app_status`).

**The mistake we're simulating**: charm author "rolls up" the unit
message into the application status for at-a-glance leader visibility.
Equally common, equally wrong.

**Diff**:

```python
# BEFORE (charm.py:408)
event.add_status(ops.ActiveStatus())

# AFTER
version = self._get_charm_version()
event.add_status(ops.ActiveStatus(f"v{version}"))
```

**Expected behavior**: this surfaces on the application's workload
status, not a unit's. The v1 detector targets units only
(`entity_kind: unit`). So advisor WILL NOT fire on this alone.

**Demo move**: use this iteration to **identify a known gap** in v1.
The audience sees `juju status` show `juju-norma-k8s   active   v1.2.3`
on the application row and asks "shouldn't advisor flag this?" — yes,
in a future iteration where the detector covers `EntityKind.application`
for the active-with-message signal. We add a follow-up TODO.

This iteration doubles as a **forcing function** for the next round of
detector work, not just a demo. (Optional: stage it if time permits.)

### Iteration 3 — Signal 3 lite: deliberate misconfig → blocked

**Edit point**: `juju config juju-norma-k8s calibration-int=70000` (no
charm code edit — pure config).

**The mistake we're simulating**: an operator pushes a config that the
charm correctly rejects with a blocked + actionable message:
`calibration-int must be between 1 and 65535, got 70000`.

**Expected behavior immediately**: the charm goes blocked with the
actionable message. **advisor DOES NOT fire** — because blocked with an
actionable message is exemplary compliance by §4c.2 (line 275 of the
brief).

**Demo move**: this iteration **proves the converse** — advisor does
not whine about blocked states that are correctly bounded and
actionable. The audience learns that advisor is specifically
calibrated, not just "anything yellow is a finding".

To trigger the actual Signal 3 (`unit-blocked-stale`, >24h), we'd need
to backdate `WorkloadStatus.Since` in the controller DQLite — beyond
the demo window. We acknowledge and skip.

### Iteration 4 — clean-up

Revert all charm.py edits, set `calibration-int` back to default, run
`juju advisor` one last time. Output is `No findings.` —
the charm is again a textbook compliant charm. The demo closes by showing
the cycle is symmetric: detect → fix → re-verify.

## Tooling steps (per iteration after iteration 0)

```bash
# 1. Inspect the edit
git diff src/charm.py

# 2. Repack the charm (fast after the first run; uv cache + charm cache)
charmcraft pack

# 3. Refresh in place
juju refresh juju-norma-k8s \
    --path ./juju-norma-k8s_ubuntu-24.04-amd64.charm

# 4. Wait for idle
juju status --watch 2s   # ctrl-c once the unit settles

# 5. Demo
juju status
juju advisor
juju advisor --format=json | jq .

# 6. Revert
git checkout src/charm.py
charmcraft pack
juju refresh juju-norma-k8s \
    --path ./juju-norma-k8s_ubuntu-24.04-amd64.charm
```

The ROCK image is only rebuilt for iteration 0 (the workload binary
doesn't change between iterations). After iteration 0, only the charm
`.charm` is repacked, which takes ~30 seconds.

## Model lifecycle

- Iteration 0 deploys into a fresh model `norma-demo`.
- All subsequent iterations refresh the same application in the same
  model.
- After iteration 4, optionally `juju destroy-model norma-demo
  --no-prompt --destroy-storage` (no `--force` — known K8s
  destroy-deadlock bug).

## What we are NOT doing

- We are not editing `juju advisor` between iterations. The same
  detector code runs against every state.
- We are not editing `src/norma.py` or `workload/main.go`. The
  workload's Pebble layer and HTTP behavior are unchanged. Every
  advisor signal is a Juju-protocol-level artifact, not a
  workload-internal one.
- We are not exercising the charm's `set-status` action to fake
  violations. The whole point is that the **code itself** is the
  advisor — actions could mask the simulation.
- We are not stacking violations in iteration 1 — keep the cadence
  one-violation-per-iteration so each detector's signal is
  cleanly attributable.

## Open question for v1.x iteration

Iteration 2 is a forcing function for an **`EntityKind.application`
variant of the active-with-message detector**. Current v1 only covers
`EntityKind.unit`. The brief's §4c.2 violation symptom applies to
both. If you want to take the v1.x detector pass before the demo, we
add `detectAppActiveWithMessage` in ~15 minutes and iteration 2
becomes a real finding rather than a known gap.
