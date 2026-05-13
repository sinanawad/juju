# Quickstart: `juju citizen`

This is the operator-facing quickstart for the v1 client command. It
also serves as the demo script for the competition window.

## Prerequisites

- A bootstrapped Juju controller (`juju bootstrap microk8s k8s` works).
- A model with at least one deployed application.
- Local build of `juju` from this branch: `make juju` (puts the
  binary at `~/go/bin/juju`).

## 30-second demo

```bash
# Pin which juju we're running
which juju && juju version

# Inspect the current model
juju citizen

# Same model, structured output
juju citizen -o json | jq .

# Switch to another model without changing context
juju citizen -m otherk8s

# Narrow attention to actionable severities
juju citizen --severity=warning,critical

# Verify AI-enrichment is layered, not load-bearing
diff <(juju citizen -o yaml) <(juju citizen --no-ai -o yaml)
# expected: differences scoped to the `recommendation:` field only
```

## Triggering each signal on demand

For demos where the model is too quiet to produce findings:

### Signal 1 (active-with-message, info)

Deploy any charm that intentionally sets a status message while active.
Most COS charms do; `grafana-k8s` reports `active: ready`.

```bash
juju deploy grafana-k8s
juju citizen --severity=info
```

### Signal 2 (charm-revision-aging, warning)

Refresh down one revision so `CanUpgradeTo` becomes non-empty:

```bash
juju deploy postgresql-k8s
# wait until idle
juju refresh postgresql-k8s --revision <current minus 1>
juju citizen --severity=warning
```

### Signal 3 (unit-blocked-stale, warning/critical)

The natural path is to deploy a charm that refuses to settle without
required config, then leave it for 24h+. For demo purposes, use the
clock-injection seam exposed in tests, or fake by deploying a charm
that requires config and letting it sit during the competition.

## Output format check

```bash
juju citizen -o json | jq 'map(keys) | unique'
# expected (sorted): ["check_id","entity","entity_kind","owner",
#                     "protocol_ref","recommendation","severity","summary"]
```

If `jq` reports any extra or missing key, the build is broken (spec
SC-003).

## Smoke-build loop (developer)

```bash
# Build only the client (no controller, no dqlite). Fast.
make juju
# Or with stdlib go:
go build -o /tmp/juju ./cmd/juju && /tmp/juju citizen --help

# Test just this package
go test ./cmd/juju/citizen/...

# Run a single test
go test -run 'TestActiveWithMessageDetector' ./cmd/juju/citizen/

# With race + stress (per AGENTS.md, for goroutined code -- not needed
# for v1 detectors since they're pure functions)
```

## Where to find each milestone's deliverable

| Milestone | Demoable output                                       |
|-----------|-------------------------------------------------------|
| M0        | `juju citizen` prints "No citizenship findings."      |
| M1        | `juju citizen -o json` returns 1 synthetic finding    |
| M2        | Real Signal 1 findings appear against a real model    |
| M3        | Real Signal 2 findings appear                         |
| M4        | Real Signal 3 findings appear (severity-by-duration)  |
| M5        | `--severity` + `--no-ai` + AI fixture all work        |
| M6        | `tests/suites/citizen/task.sh` passes end-to-end      |
