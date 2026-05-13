# Implementation Plan: `juju citizen` operator CLI command

**Branch**: `003-juju-citizen-cli` | **Date**: 2026-05-13 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/003-juju-citizen-cli/spec.md`

## Summary

Add a new client-side CLI subcommand `juju citizen` that fetches the
current model status via the existing `Client.Status` facade, runs three
local detector predicates against the response, optionally enriches each
finding's recommendation from a bundled JSON fixture, and renders the
result in one of three formats (default hybrid, YAML, JSON). No new
server-side facade. No persistence. Output shape and detection layer
follow the constitution: structured Findings with six load-bearing
fields, three-level severity, four-value owner enum, and a citation to
the citizenship contract clause violated.

The plan is **iteration-first**: M0-M6 milestones each end with a
runnable, demoable artifact. M0+M1+M2 alone is a sufficient
competition-window MVP; M3-M6 are additive.

## Technical Context

**Language/Version**: Go 1.22 (matches the rest of the juju repo;
`go.mod` is the source of truth).

**Primary Dependencies**:
- `github.com/juju/juju/cmd/modelcmd` — provides `ModelCommandBase` and
  `Wrap`. Handles `-m`/`--model`, controller selection, API root.
- `github.com/juju/juju/cmd/cmd` — provides `cmd.Command`, `cmd.Output`
  (formatter map), `cmd.Info`, `cmd.CheckEmpty`.
- `github.com/juju/juju/api/client/client` — provides `Client.Status()`
  returning `*params.FullStatus`.
- `github.com/juju/juju/rpc/params` — provides `FullStatus`,
  `ApplicationStatus`, `UnitStatus`, `DetailedStatus`.
- `github.com/juju/gnuflag` — flag parsing within `SetFlags`.
- `github.com/juju/clock` — injected clock for the time-since-blocked
  detector (testability — see constitutional CODING rules: never use
  `time.Now()`).
- `embed` (stdlib) — bundle the AI fixture inside the binary at build
  time.

**Storage**: N/A. The command is a stateless read.

**Testing**:
- Unit tests with `github.com/juju/tc` (per `AGENTS.md`). Detector
  predicates tested in isolation against synthetic `params.FullStatus`
  fixtures.
- Format-layer tests are golden-style: build a known `[]Finding`, render
  each format, diff against checked-in expected output.
- Optional bash integration test under `tests/suites/citizen/`
  (post-M6).

**Target Platform**: Linux/macOS (any platform juju builds on). The
binary ships inside the existing `juju` client.

**Project Type**: New subpackage `cmd/juju/citizen/` under the existing
client CLI. Registered in `cmd/juju/commands/main.go` alongside `block`,
`status`, etc.

**Performance Goals**: No perceptible additional latency over the
underlying `juju status` call (spec SC-001). Detection is O(N) over
units+apps with constant-factor predicates; trivial.

**Constraints**:
- MUST follow the `cmd/juju/block/list.go` command shape (constitution
  Principle VIII).
- MUST NOT modify `params.FullStatus` or any other wire type
  (constitution Principle X; no-new-facade promise).
- MUST work against a Juju 4.0 controller (the working branch). v1
  targets 4.0 only -- see spec Clarification Q3. Fields absent from
  the response are still treated as empty rather than errors (FR-009),
  so a future broadening to 3.6.x is a low-risk extension; it is not
  in v1 scope.

**Scale/Scope**: Single new subpackage. Estimated < 1000 LOC including
tests across all six milestones.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Reference: `.specify/memory/constitution.md` (v1.0.0).

- **Gate I -- Finding schema completeness**: PASS. The `Finding` struct
  in `data-model.md` carries all eight required fields (the six
  constitutional + `entity_kind` + `check_id` since check_id IS the
  contract clause anchor). A constructor `newFinding(...)` is the only
  emission path; missing-field detection happens via Go's zero-value
  semantics at the detector boundary (compile-time enforcement for
  enums via typed constants).
- **Gate II -- Severity calibration**: PASS. The `Severity` type is a
  typed string with three exported constants
  (`SeverityInfo`, `SeverityWarning`, `SeverityCritical`). Detectors
  reference the constants; no detector introduces a fourth value.
- **Gate III -- Owner classification at detection layer**: PASS. Each
  detector hardcodes its `owner` value at the point of finding
  construction. The CLI/renderer never decides ownership.
- **Gate V -- Contract clause citation**: CONDITIONAL PASS. Detectors
  cite clause anchors from `citizenship-observatory-brief.md` §4c. The
  exact clause IDs are recorded in `data-model.md`. If the brief's §4c
  uses a different anchor convention, the planner records the
  divergence under Complexity Tracking and updates the brief in the
  same PR (per constitution Principle V).
- **Gate VIII -- Juju conventions**: PASS. The package layout follows
  `cmd/juju/block/`: `command.go`, `formatter.go`, `detectors.go`,
  `command_test.go`. Command shape mirrors `block.NewListCommand` —
  `modelcmd.Wrap`, `ModelCommandBase`, `cmd.Output.AddFlags`,
  `jujucmd.Info`. No new facade introduced.
- **Gate X -- Backwards compatibility**: PASS. The change is purely
  additive: a new subpackage and one line in
  `cmd/juju/commands/main.go`'s `registerCommands`. No existing
  command output changes. No wire-type changes.

**Design-direction principles** (not pass/fail):
- **Principle IV (Runtime observation)**: All three v1 detectors run
  against live `Client.Status` output. None can be answered by static
  charm analysis: Signal 1 is per-unit message text; Signal 2 is
  resolver-driven (`CanUpgradeTo` is computed); Signal 3 is temporal.
- **Principle VI (AI as optional transformer)**: Fixture lookup runs
  AFTER detection. Each Finding is operator-actionable with its
  hand-written recommendation. `--no-ai` simply skips the fixture
  swap. Missing fixture is non-fatal (degrades, doesn't error).
- **Principle VII (Auto-clearing)**: N/A for v1 — explicitly deferred
  in spec Out of Scope. Each invocation is a fresh read.
- **Principle IX (Detection layer placement)**: Client-side per v1
  scope. Data shape (`Finding`) is identical to what a future
  controller-side `Health` facade would emit. The detector pure
  functions take `*params.FullStatus` only — they are portable to the
  server side without modification.

## Project Structure

### Documentation (this feature)

```text
specs/003-juju-citizen-cli/
├── plan.md              # This file
├── research.md          # Phase 0 output (this command)
├── data-model.md        # Phase 1 output (this command)
├── quickstart.md        # Phase 1 output (this command)
├── contracts/
│   └── cli-contract.md  # CLI argv + output schema
├── checklists/
│   └── requirements.md  # From /speckit-specify
└── tasks.md             # Phase 2 output (/speckit-tasks)
```

### Source Code (repository root)

```text
cmd/juju/citizen/
├── command.go             # NewCitizenCommand, Init, Run, SetFlags, Info
├── doc.go                 # Package doc (per AGENTS.doc-dot-go-rules)
├── detectors.go           # Pure-function detectors + DetectorRegistry
├── finding.go             # Finding type + Severity/Owner/EntityKind enums
├── formatter.go           # hybrid/yaml/json formatters
├── enricher.go            # Fixture loader + recommendation swap
├── fixtures.go            # //go:embed testdata/findings.json
├── testdata/
│   └── findings.json      # 3 entries, one per detector (M5 deliverable)
├── command_test.go        # End-to-end command tests
├── detectors_test.go      # Per-detector predicate tests
├── formatter_test.go      # Golden-output tests
└── export_test.go         # Test-only exports

cmd/juju/commands/main.go  # MODIFIED: one line to register the command
```

**Structure Decision**: A self-contained subpackage `cmd/juju/citizen/`
mirroring `cmd/juju/block/`. Detection lives in `detectors.go` as pure
functions (`func detectActiveWithMessage(*params.FullStatus) []Finding`),
deliberately decoupled from the command struct so future migration to a
controller-side `Health` facade (Principle IX) is a copy-paste, not a
refactor.

## Iteration Plan (the user-facing roadmap)

Each milestone ends with a working `juju citizen` binary that
demonstrates exactly one capability the previous milestone lacked.
Each milestone targets ≤ 3 files changed (preferred: 2) and ≤ 30
minutes of focused work. Drop-out points: M2, M4.

### M0 -- Scaffold (target: 20 min, drop-out: NO)

**Deliverable**: `juju citizen` runs against any model, prints
`No citizenship findings.` and exits 0. `juju help citizen` prints
purpose + flags.

**Files touched**:
1. `cmd/juju/citizen/command.go` — minimal `citizenCommand` struct,
   Info/Init/Run/SetFlags. Run() prints the no-findings literal.
2. `cmd/juju/citizen/doc.go` — package doc per Juju doc.go rules.
3. `cmd/juju/commands/main.go` — `r.Register(citizen.NewCitizenCommand())`
   inserted alphabetically near `block.NewListCommand()`.

**Test**: `command_test.go` — assert Info().Name == "citizen", Run()
prints the literal, exit code 0.

**Demo line**: `~/go/bin/juju citizen` against a fresh microk8s model.

### M1 -- Lock the output shape (target: 30 min, drop-out: NO)

**Deliverable**: `juju citizen` (no flags) returns one hardcoded
synthetic finding in hybrid format. `-o yaml` and `-o json` work.

**Files touched**:
1. `cmd/juju/citizen/finding.go` — `Finding` struct, three enums,
   `serializableFinding` shadow (the YAML/JSON view), `toSerializable()`.
2. `cmd/juju/citizen/formatter.go` — `formatHybrid(io.Writer, []Finding)`;
   wire `c.out.AddFlags(f, "hybrid", map[string]cmd.Formatter{"hybrid":
   formatHybrid, "yaml": cmd.FormatYaml, "json": cmd.FormatJson})`.
3. `cmd/juju/citizen/command.go` — Run() returns a hardcoded
   `[]Finding{synthetic}`.

**Test**: `formatter_test.go` — golden outputs for hybrid/yaml/json
against a fixed two-finding `[]Finding`. Lock the byte-exact form here
so M2+ never re-tests rendering.

**Risk pin**: This is the milestone where output bikeshedding could
eat the clock. The hybrid format is fixed in `contracts/cli-contract.md`
before M1 begins — see that file. Do NOT iterate on visual design
during M1.

**Demo line**: `~/go/bin/juju citizen -o json | jq .`

### M2 -- First real detector: active-with-message (target: 30 min, drop-out: YES)

**Deliverable**: `juju citizen` actually walks the model. For any unit
in `active` state with a non-empty workload message, an info-severity
finding is emitted. Otherwise the no-findings message is printed.

**Files touched**:
1. `cmd/juju/citizen/detectors.go` — `detectActiveWithMessage(
   status *params.FullStatus) []Finding`. Hand-written summary +
   recommendation + protocol_ref baked into the detector constants.
2. `cmd/juju/citizen/command.go` — `getStatusAPI()` interface +
   `NewAPIRoot(ctx)` wiring, mirroring `block/list.go` lines 110-117.
   Run() calls Status, dispatches to detectors, renders.

**Test**: `detectors_test.go` — feed three synthetic FullStatus
fixtures (clean, one match, three matches) directly into
`detectActiveWithMessage`; assert finding count and field values.

**Drop-out point**: If the competition window closes here, this is a
complete, defensible demo of the "Juju citizenship" concept.

**Demo line**: Deploy a charm whose unit reports `active` with a
non-trivial message (most COS charms do). Run `juju citizen`.

### M3 -- Second detector: charm-revision-aging (target: 20 min, drop-out: YES)

**Deliverable**: Each application whose `ApplicationStatus.CanUpgradeTo`
is non-empty produces a warning-severity finding.

**Files touched**:
1. `cmd/juju/citizen/detectors.go` — `detectCharmRevisionAging`.
2. `cmd/juju/citizen/command.go` — append to detector dispatch chain.

**Test**: `detectors_test.go` — one new test function.

**Demo line**: `juju refresh --revision N-1` an app, then
`juju citizen`.

### M4 -- Third detector: unit-blocked-stale (target: 30 min, drop-out: YES)

**Deliverable**: Each unit in `blocked` for >24h emits a warning;
>7d emits a critical. Uses injected clock.

**Files touched**:
1. `cmd/juju/citizen/detectors.go` — `detectUnitBlockedStale(
   *params.FullStatus, clock.Clock) []Finding`. Severity by duration.
2. `cmd/juju/citizen/command.go` — clock field on the command struct
   (defaults to `clock.WallClock`); plumb to detector.
3. `cmd/juju/citizen/export_test.go` — expose a setter for the clock
   in tests.

**Test**: `detectors_test.go` — boundary cases at 24h-1s, 24h+1s,
7d-1s, 7d+1s. Uses `clock.NewClock(fixedTime)`.

**Demo line**: Set a charm to error+resolve into blocked, wait (or
fake the since), `juju citizen`.

### M5 -- AI enrichment + severity filter + --no-ai (target: 40 min, drop-out: YES)

**Deliverable**: `--severity`, `--no-ai`, and the fixture-driven
recommendation rewrite all work. The hybrid output's note lines come
from the AI-enriched recommendation by default.

**Files touched**:
1. `cmd/juju/citizen/fixtures.go` — `//go:embed testdata/findings.json`;
   `Enrich(findings []Finding) []Finding`.
2. `cmd/juju/citizen/testdata/findings.json` — three entries keyed by
   check_id, each with a paragraph-length recommendation.
3. `cmd/juju/citizen/command.go` — `--severity` (`cmd.StringValue` or a
   manual gnuflag var; comma-split + validate) and `--no-ai` flags;
   filter pass after enrichment.

**Test**: `command_test.go` — adds: (a) `--severity=critical` filters
to zero against the standard fixture; (b) `--no-ai` recommendation
differs from default; (c) missing fixture file degrades gracefully
(test via a build-tag-disabled embed or by injecting a broken loader).

**Demo line**: `juju citizen --severity=warning,critical`,
`juju citizen --no-ai`.

### M6 -- Polish + integration smoke (target: 30-60 min, optional)

**Deliverable**: `tests/suites/citizen/task.sh` bootstraps microk8s,
deploys a known-degraded charm, runs `juju citizen -o json`, asserts
finding count via `jq`. Adds a `SeeAlso` link from `juju status` doc
(if owners approve — otherwise skip).

**Files touched**:
1. `tests/suites/citizen/task.sh` (new).
2. `tests/main.sh` — register suite (one line).
3. `cmd/juju/citizen/doc.go` — flesh out, add usage examples.

## Complexity Tracking

> No constitutional violations identified during gate evaluation. This
> table is intentionally empty.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| (none)    | (n/a)      | (n/a)                               |

## Risks and Mitigations

- **Risk**: `params.FullStatus` field names differ between the version
  the spec assumed and the version on the working branch (4.0). The
  spec referenced `CanUpgradeTo` and per-unit `since`; the brief on
  3.6 uses these names but 4.0 may have renamed under the DQLite
  rewrite.
  **Mitigation**: M2's first task is to grep `params.FullStatus`,
  `ApplicationStatus`, `DetailedStatus`, and `UnitStatus` in the
  current tree and record actual field names in `research.md` before
  writing the detector. If a field is missing, that detector
  short-circuits to zero findings (per FR Assumptions).

- **Risk**: The hybrid format spends time getting "right." This is the
  classic bikeshed hazard.
  **Mitigation**: The format is fixed byte-for-byte in
  `contracts/cli-contract.md` before M1. M1 implements that; nothing
  else.

- **Risk**: The AI fixture file becomes a writing exercise.
  **Mitigation**: Each fixture entry is capped at three sentences. If
  the demo clock pressures, leave the hand-written terse text and ship
  with the fixture skipped at runtime — the constitution permits this
  (Principle VI).

- **Risk**: Command registration line is added but `make juju` doesn't
  pick up the new package (stale import cache).
  **Mitigation**: After M0, run `go build ./cmd/juju/...` directly to
  verify before invoking `make juju`. (Per `JUJU.md`: `make juju`
  builds the client.)

## Build Loop

For tight iteration during the competition:

```bash
# Build just the client (fast, no controller binary)
make juju             # produces ~/go/bin/juju

# Or even faster -- skip the make wrapper
go build -o /tmp/juju ./cmd/juju && /tmp/juju citizen

# Test a single package
go test ./cmd/juju/citizen/...

# Run a single test
go test -run 'TestActiveWithMessageDetector' ./cmd/juju/citizen/
```

The detector pure-function pattern means most iteration cycles never
need a live controller — `go test ./cmd/juju/citizen/` against
synthetic `params.FullStatus` is enough.
