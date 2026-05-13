<p align="center">
  <img alt="Canonical Juju advisor — v1, good standing" src="docs/.sphinx/_static/logos/advisor.png?raw=true" width="35%">
</p>

# `juju advisor` — proof of concept

> **This is a fork-branch README.** The upstream Juju README has been replaced
> on this branch with a focused write-up of the `juju advisor` proof of
> concept. The upstream README is preserved in `main`.

`juju advisor` is a new client-side subcommand that surfaces deployment-level
issues — convention violations, temporal anomalies, and structural wedges —
that aren't visible in `juju status`'s state-snapshot output. It tells you
*what's wrong*, *how urgent it is*, *who should fix it*, and *what to do about
it*.

This branch is a time-boxed proof of concept. The code is functional and
tested against real Juju 4.0 controllers, but it is not production-graded and
is not in shape for upstream merge as-is.

---

## Why this exists

`juju status` shows model state. State alone is ambiguous to an operator:

- A unit reporting `active, "serving on port 8080"` looks healthy — but the
  active-status convention is to carry no message. The message itself is a
  contract violation; nothing in `juju status` flags it.
- A unit reporting `maintenance, "preparing calibration suite"` looks like
  work in progress — until you notice it's been saying that for six hours.
  `juju status` is a snapshot; it has no concept of "stuck."
- A unit reporting `blocked` with an empty message gives the operator nothing
  to act on.
- A unit wedged in `Life=Dying` after a removal request is invisible in
  `juju status` headlines — it just looks like another `error`.

Operators are left to triage by hand: read state, infer meaning, guess
priority, find owner, derive fix. `juju advisor` does that triage and
surfaces it as a sorted, severity-tagged, ownership-attributed report with
mitigation recommendations.

The pitch in one line: **`juju status` shows what is. `juju advisor` shows
what's wrong and what to do.**

---

## What's implemented in v0.1

A client-side detector framework with **9 detectors** covering both static
(single-snapshot) and temporal (StatusHistory-based) issues:

| # | check_id                     | severity      | type     | trigger                                          |
|---|------------------------------|---------------|----------|--------------------------------------------------|
| 1 | `active-with-message`        | INFO          | static   | active state with non-empty message              |
| 2 | `blocked-no-message`         | WARNING       | static   | blocked state with empty message                 |
| 3 | `charm-revision-aging`       | WARNING       | static   | unit lags its tracked channel                    |
| 4 | `unit-blocked-stale`         | WARN/CRITICAL | static   | unit blocked >24h                                |
| 5 | `entity-stuck-dying`         | WARN/CRITICAL | static   | application or unit in `Life=Dying` >5m         |
| 6 | `model-suspended-credential` | CRITICAL      | static   | model credential suspended                       |
| 7 | `status-churn`               | WARNING       | temporal | 3+ workload-status transitions in 10 minutes     |
| 8 | `stuck-maintenance`          | WARNING       | temporal | maintenance state held >5 minutes, no progression|
| 9 | `hook-error`                 | WARNING       | temporal | uncaught hook exception in last 30 minutes       |

Three output formats:

- **`table`** (default) — dashboard panel + 6-column severity-sorted table
  with severity glyphs, owner classification, age, and one-line summaries.
- **`verbose`** — per-finding multi-line output with mitigation
  recommendations.
- **`json` / `yaml`** — structured output for tooling.

Each finding carries an **owner classification**:

- `AUTHOR` — the charm author should investigate.
- `OP` — the operator running the deployment should investigate.
- `PLAT` — the underlying platform (substrate, network, storage) is
  implicated.
- `MIX` — symptom crosses team boundaries.

Each finding also carries a **recommendation**: a short, plain-English
suggestion for what to do next. Recommendations are loaded from a static
fixture file shipped with the binary — the structure is ready for a live
LLM integration, but no LLM is called today.

Approximately 50 unit tests cover the detectors, the formatters, and the
command itself. Golden-test discipline pins byte-locked CLI contracts (the
table format, the verbose format, and the JSON schema).

A companion charm at
[`github.com/sinanawad/juju-norma-k8s`](https://github.com/sinanawad/juju-norma-k8s)
(branch `001-calibration-charm`) provides 8 deliberately-bad-behavior modes
selectable via a `bad-behavior-mode` config option — useful for staged
demos and for regression testing detectors against real charm behavior.
See `docs/BEHAVIOR-MODES.md` in that repo.

---

## Try it

Requires a Juju 4.0 controller (CAAS or IAAS).

```bash
# Build
make juju

# Default — dashboard + table
~/go/bin/juju advisor

# Per-finding mitigation recommendations
~/go/bin/juju advisor --format=verbose

# Structured output for tooling
~/go/bin/juju advisor --format=json | jq .

# Filter on severity
~/go/bin/juju advisor --severity=warning,critical

# Skip the recommendation enrichment step
~/go/bin/juju advisor --no-ai

# Different model
~/go/bin/juju advisor -m other-controller:other-model
```

---

## How it works

```
juju advisor
    └─> client-side command (cmd/juju/advisor/)
        ├─> calls Client.Status() and Client.StatusHistory() over the wire
        ├─> runs the 6 pure detectors against the snapshot
        ├─> runs the 3 stateful detectors against StatusHistory
        ├─> enriches recommendations from the bundled fixture
        └─> renders one of {table, verbose, json, yaml}
```

All detection runs **client-side**. No new facades. No new schema. No new
controller workers. Anyone running a Juju 4.0 build of this branch can point
it at any controller they have access to.

This is the right shape for a v0.1 — it ships without a controller-side
migration and is easy to iterate on — but it has clear limits, which is
what the roadmap below addresses.

---

## What's NOT in v0.1

Calling these out explicitly so it's clear what the demo is and isn't
showing:

- **Server-side detection.** Every `juju advisor` invocation is a fresh
  round-trip to the controller. Fine for ad-hoc use, not fine for continuous
  observation.
- **Persistent history of findings.** You only see what's true at the moment
  you ran the command. No "yesterday, this entity was stuck for 4 hours."
- **Trend lines / metrics.** No way to ask "how often does `worker` churn its
  status this week?"
- **Alerting / notifications.** No long-running watcher; no `juju advisor
  watch`.
- **Multi-model / controller-wide view.** One model per invocation.
- **Detector authoring SDK.** The 9 detectors are hard-coded in the advisor
  binary. Charm authors can't ship their own.
- **Real LLM integration.** Recommendations come from a static JSON fixture
  baked into the binary. The wiring is in place, but no LLM is called.
- **Audit log.** No record of which recommendations operators followed or
  ignored.

---

## Roadmap

A path from "useful CLI gadget" to "first-class part of Juju's operational
story." Each step is independently shippable.

### v0.2 — server-side detection

Migrate the detector engine into an apiserver facade (provisionally `Health`,
per the constitutional Principle IX of this project). Findings are computed
inside the controller, served via one facade call. The CLI output stays
identical; the protocol moves under the hood.

Why this matters: opens the door to caching, to detectors that need
controller-internal state (e.g., dqlite invariants), and to operators who
can't run a custom CLI build.

### v0.3 — metric collection

Findings are emitted as events into a controller-local time-series. The
controller charm exposes a `prometheus_scrape` endpoint by default, so
existing Grafana dashboards can ingest advisor signals. Operators can ask:

- "Show me the units that churned most often this week."
- "Trend the count of `stuck-maintenance` entities over the last month."
- "Which charms generated the most CRITICAL findings since the last
  release?"

This is where `juju advisor` stops being a CLI moment and starts being a
data source.

### v0.4 — `juju advisor watch`

A long-running session that streams findings as state evolves. Tied into
Juju 4.0's changestream infrastructure, so the watcher doesn't poll — it
re-evaluates affected detectors when the underlying state changes. This is
the foundation for alerting.

### v0.5 — detector marketplace

Charm authors ship detectors as part of the charm. The controller picks
them up at deploy time and runs them alongside the built-in catalog.
Authors can encode domain knowledge ("for *this* charm, `stuck-maintenance`
is normal during the initial 30 minutes" or "this charm has a known
crash-loop signature that warrants its own severity") without waiting on a
Juju release.

This unblocks charm authors as a source of operational signal — today the
advisor's intelligence comes from a tiny team; tomorrow it comes from the
ecosystem.

### v0.6 — live AI recommendations

Replace the static recommendation fixture with calls to a configurable LLM
backend. Operators get context-aware mitigation suggestions:

> "You have `stuck-maintenance` on `worker` and `install` is failing on
> `payments`. These charms share a database peer relation; check the shared
> dependencies first."

Local-only or cloud-API LLM, operator's choice. Recommendations remain
attributable: the operator sees which fixture or which prompt produced
each suggestion.

### v1.0 — controller-side advisor as a service

The advisor becomes a long-running worker in the controller, continuously
evaluating model state, persisting findings, and exposing them via API +
CLI + a permanent Grafana panel. Three audiences see a feedback loop:

- **Operators** see a "model health" view that updates in real time and
  remembers history.
- **Charm authors** see how their charms actually behave in production —
  not just CI — and can iterate against that signal.
- **Platform teams** see compliance posture across all the controllers
  they're responsible for.

At that point `juju advisor` stops being a separate command and becomes
part of how Juju is operated day-to-day.

---

## What's honest about this

- The 9 detectors are real and ship in this branch. You can run them against
  any Juju 4.0 controller right now.
- The detector taxonomy is informed by — but not limited to — a synthesis
  of Juju's distributed-system contract in `advisor-brief.md`. That document
  is itself a contribution: a one-place summary of guarantees and
  expectations that today are scattered across charm SDK docs, Juju
  reference docs, and Discourse folklore.
- The output design is byte-locked and golden-tested. The CLI contract in
  `specs/003-juju-advisor-cli/contracts/cli-contract.md` is the source of
  truth.
- The "AI-enriched recommendations" today are a static fixture. The fixture
  exists, the wiring exists, but no LLM is called. The roadmap is explicit
  about closing that gap.
- This branch is a fork-PoC. It is not an upstream proposal in its current
  shape. The roadmap above is a sketch of *if this concept is endorsed,
  what would full delivery look like*.

---

## Repo context

- **Branch**: `003-juju-citizen-cli` (historical name — the project's
  earlier framing was "citizenship observatory"; the public-facing concept
  is now `juju advisor`).
- **Upstream**: `github.com/juju/juju` — this is a fork. The 4.0 line is in
  `main` upstream (DQLite-based); upstream's `main` here also tracks 4.0.
- **Companion charm**:
  [`github.com/sinanawad/juju-norma-k8s`](https://github.com/sinanawad/juju-norma-k8s),
  branch `001-calibration-charm`. Provides the test-bed for staged-failure
  demos.
- **Project documents** (in `specs/003-juju-advisor-cli/`):
  - `spec.md` — what the v0.1 advisor is and isn't.
  - `plan.md` — implementation plan with M0–M6 milestones.
  - `data-model.md` — the Finding type and its eight-field contract.
  - `contracts/cli-contract.md` — byte-locked CLI output specification.
  - `research.md`, `tasks.md`, `quickstart.md` — development artifacts.
- **Constitution** at `.specify/memory/constitution.md` — the ten principles
  this PoC was built under.
- **Original protocol brief** at `advisor-brief.md` — the
  distributed-systems contract synthesis that informed the detector
  taxonomy.

---

## License

Same as upstream Juju: AGPLv3. See `LICENCE`.
