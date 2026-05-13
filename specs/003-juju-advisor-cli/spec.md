# Feature Specification: `juju advisor` operator CLI command

**Feature Branch**: `003-juju-advisor-cli`

**Created**: 2026-05-13

**Status**: Draft

**Input**: User description: "Client-side `juju advisor` CLI command surfacing
three advisor signals (active-with-message, charm-revision aging, unit
blocked >24h) with hybrid/yaml/json output, severity filter, model override,
and fixture-backed AI-enrichment."

## Iteration Scope

This iteration delivers the **prototype** as defined in the project
constitution (Principles VII and IX): client-side detection, no
auto-clearing, no controller-side `Health` facade. A subsequent **v1**
iteration will migrate detection behind a `Health` facade and add the
auto-clearing lifecycle; both are explicitly out of scope here. Where
historical clarification bullets below refer to "v1", read "this
prototype iteration".

## Clarifications

### Session 2026-05-13

- Q: When the underlying `Client.Status` call returns an error
  (controller unreachable, network blip, permission revoked mid-call),
  how should `juju advisor` behave? → A: Hard error -- exit non-zero
  with a stderr line `ERROR: status fetch failed: <wrapped err>` and no
  stdout. Matches FR-018 verbatim and aligns with the behavior of
  `juju status` itself on the same failure.
- Q: How should the command handle a detector that panics or returns
  an internal error (e.g., on a malformed status entry)? → A:
  Hard fail -- the panic propagates and the command exits non-zero
  with stderr naming the failing detector. Each detector is
  responsible for defending against known nullable fields (per
  FR-009); recovery scaffolding is deliberately not added in v1.
- Q: Which Juju controller version is the v1 CLI command targeted to
  run against? → A: Juju 4.0 only. The working branch is `4.0`; T016
  verifies `params.FullStatus` field names against this tree only.
  Multi-version compatibility (3.6.x) is explicitly a v2 concern. The
  "missing field = zero findings" rule (FR-009) already makes a
  future broadening low-risk.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Operator triages an unfamiliar model (Priority: P1)

An operator has been handed a model they did not deploy and have not
recently touched. They want a one-screen answer to the question
"is anything degraded?" without scrolling through `juju status`.

They run `juju advisor` and receive a sorted list of findings -- one
header line per finding, plus 1-3 arrow-prefixed notes summarising the
issue and what to do about it. Critical findings appear first; info-level
findings appear last. If the model is clean, the command prints
"No findings." and exits 0.

**Why this priority**: This is the entire reason the command exists. The
operator-facing read-only view is the MVP; everything else either supports
it or follows it.

**Independent Test**: Bootstrap a controller, deploy a charm whose unit
reports `active` with the message "ready", run `juju advisor`. The
expected outcome is exactly one finding header at info severity for the
unit, with one or more arrow notes that name the convention violated and
the recommended action.

**Acceptance Scenarios**:

1. **Given** a model containing zero degradations, **When** the operator
   runs `juju advisor`, **Then** stdout is exactly `No compliance
   findings.` followed by a newline, and the exit code is 0.
2. **Given** a model with one unit reporting `active` with a non-empty
   workload status message, one application whose `CanUpgradeTo` is
   non-empty, and one unit in `blocked` state for more than 24 hours,
   **When** the operator runs `juju advisor`, **Then** stdout contains
   exactly three finding headers, ordered critical -> warning -> info
   (or warning -> info if no critical), each followed by 1-3 arrow notes.
3. **Given** any successful run, **When** stderr is inspected, **Then**
   no diagnostic noise appears on stderr; user-facing output is on stdout.

---

### User Story 2 - Operator pipes findings into downstream tooling (Priority: P2)

An operator (or a CI script, or a Slack-bot bridge) wants machine-readable
findings so they can be persisted, alerted on, or correlated with other
signals. They run `juju advisor -o yaml` or `juju advisor -o json` and
receive a structured list of records with stable field names.

**Why this priority**: Without this, the command is a human-only tool and
cannot feed the future observatory dashboards. Tooling integration is the
second-largest use case.

**Independent Test**: Run `juju advisor -o json` against a staged model
known to have three findings; pipe to `jq '. | length'`. The expected
outcome is `3`. Run `jq '.[0] | keys | sort'`. The expected outcome is a
sorted array of exactly the eight field names defined under "Key Entities".

**Acceptance Scenarios**:

1. **Given** a staged model with three findings, **When** the operator
   runs `juju advisor -o yaml`, **Then** stdout is a YAML list of three
   mappings, each carrying all eight fields specified for a Finding, with
   no extra fields.
2. **Given** the same model, **When** the operator runs `juju advisor
   -o json`, **Then** stdout is a single JSON array of three objects with
   the same field set as the YAML output.
3. **Given** structured output, **When** `severity` is inspected, **Then**
   every value is one of `info`, `warning`, `critical` -- no other strings.
4. **Given** structured output, **When** `owner` is inspected, **Then**
   every value is one of `charm-author`, `operator`, `mixed`, `platform`.

---

### User Story 3 - Operator inspects a model other than the current one (Priority: P2)

An operator has multiple models on a controller. They want to check the
compliance state of a model without first switching their working model
context, because they are mid-investigation in another model and do not
want to lose state.

**Why this priority**: Standard juju-CLI ergonomics. Every comparable
command (`juju status`, `juju config`, `juju show-application`) accepts
`-m`/`--model`. Omitting it would feel broken on day one.

**Independent Test**: With current model A, run `juju advisor -m B`
against a controller hosting models A and B with known different finding
sets. The output reflects model B's state, not A's.

**Acceptance Scenarios**:

1. **Given** the operator's current model is A and model B has one
   finding, **When** they run `juju advisor -m B`, **Then** the output
   reflects model B's single finding and the operator's current model
   context remains A.
2. **Given** the operator names a model the controller does not host,
   **When** they run `juju advisor -m nonexistent`, **Then** the command
   exits non-zero with a stderr message that names the missing model.

---

### User Story 4 - Operator narrows attention to actionable severities (Priority: P3)

An operator running a multi-charm production deployment expects info-level
convention noise. They want to focus on findings that demand action this
sprint or sooner.

**Why this priority**: Useful but not load-bearing for the demo. The MVP
output is already severity-sorted; filtering is a refinement.

**Independent Test**: Against a model known to have all three signals,
run `juju advisor --severity=warning,critical`. The info-severity finding
is omitted; the warning and (if present) critical findings remain.

**Acceptance Scenarios**:

1. **Given** a model with findings at info, warning, and critical
   severity, **When** the operator runs `juju advisor
   --severity=critical`, **Then** only critical findings appear and the
   exit code is 0 (the command is not a check, so absence of matching
   findings is not an error).
2. **Given** the same model, **When** the operator runs `juju advisor
   --severity=warning,critical`, **Then** only warning and critical
   findings appear.
3. **Given** an invalid severity string, **When** the operator runs
   `juju advisor --severity=bogus`, **Then** the command exits non-zero
   with a stderr message naming the valid values.

---

### User Story 5 - Operator works offline or AI-skeptically (Priority: P3)

An operator is on a restricted network, or distrusts AI-generated text,
or simply wants to verify what the underlying detector said before any
enrichment was applied. They run `juju advisor --no-ai` to suppress AI
enrichment of recommendation text.

**Why this priority**: Constitutionally required (AI must be optional, per
Principle VI). Functionally, the hand-written recommendation is shorter
and considered authoritative; the AI version is a presentation layer.

**Independent Test**: Run `juju advisor` and `juju advisor --no-ai`
against the same staged model. The header line, severity, owner, entity,
check_id, and protocol_ref of each finding are byte-identical; only the
recommendation differs.

**Acceptance Scenarios**:

1. **Given** any staged model, **When** the operator runs the command
   twice -- once with `--no-ai` and once without -- **Then** the set of
   finding identities (check_id + entity) is identical between the two
   runs, and within each finding only the recommendation text differs.
2. **Given** the AI fixture file is missing or unreadable, **When** the
   operator runs `juju advisor` (no `--no-ai`), **Then** the command
   completes successfully using the terse hand-written recommendations
   and emits a single stderr warning indicating enrichment was skipped.

---

### Edge Cases

- A model containing zero applications and zero units returns "No
  findings." and exits 0.
- A unit whose status is `active` with whitespace-only message (`"   "`)
  is treated as a non-empty message and triggers Signal 1.
- A unit whose `since` timestamp lies in the future (clock skew between
  controller and client) is treated as `since = now` for Signal 3 and
  therefore does not trigger.
- A controller running a Juju version whose status response does not
  populate `CanUpgradeTo` results in zero Signal 2 findings, not an
  error.
- An operator with read access to the model but no internet access to
  fetch fixtures sees the same `--no-ai`-equivalent recommendation
  text plus a stderr warning. Fixtures live inside the binary's
  testdata; this case is reserved for production AI integration.
- A finding whose entity has since been removed between the status read
  and rendering is still emitted; the entity name is whatever the status
  response named it.
- A user with no read permission on the named model receives the standard
  juju permission-denied error and a non-zero exit.
- If `Client.Status` returns an error mid-call (controller unreachable,
  network failure, permission revoked, partial response), the command
  exits non-zero with stderr `ERROR: status fetch failed: <wrapped err>`
  and writes nothing to stdout. No partial findings are emitted. This is
  distinct from the fixture-loader failure path (FR-016), which is
  intentionally graceful.

## Requirements *(mandatory)*

### Functional Requirements

**Command surface**

- **FR-001**: The `juju advisor` command MUST be invocable with no
  arguments and operate against the operator's current model context.
- **FR-002**: The command MUST accept `-m <model>` / `--model <model>` to
  override the model context for this invocation.
- **FR-003**: The command MUST accept `-o yaml` / `--format=yaml` and
  `-o json` / `--format=json` flags, following the same format-flag
  conventions as `juju status`. The default format is the hybrid
  human-readable format defined in FR-010.
- **FR-004**: The command MUST accept `--severity=<csv>` accepting any
  non-empty comma-separated subset of `{info, warning, critical}`. An
  unrecognised value MUST cause non-zero exit with a stderr message.
- **FR-005**: The command MUST accept `--no-ai` to disable AI enrichment
  of finding recommendations. AI enrichment is enabled by default.

**Detection**

- **FR-006**: The command MUST emit one Signal 1 (`active-with-message`)
  Finding per unit whose workload status equals `active` and whose
  workload status message is non-empty (after trimming, whitespace-only
  messages count as non-empty). Severity `info`. Owner `charm-author`.
- **FR-007**: The command MUST emit one Signal 2 (`charm-revision-aging`)
  Finding per application whose `CanUpgradeTo` field is a non-empty
  string. Severity `warning`. Owner `operator`.
- **FR-008**: The command MUST emit one Signal 3 (`unit-blocked-stale`)
  Finding per unit whose workload status equals `blocked` and whose
  `since` timestamp is more than 24 hours in the past. Severity is
  `warning` for durations in (24h, 7d] and `critical` for durations
  beyond 7d. Owner `mixed`.
- **FR-009**: Detection MUST run client-side; the command MUST NOT
  require any new server-side facade. It MUST be implementable using
  whatever facades `juju status` already calls. Each detector is a
  pure function and MUST defend against known nullable fields in the
  status response (a missing or zero-value field MUST result in zero
  findings from that detector, not a runtime panic). The command does
  NOT wrap detectors in a recover boundary in v1 -- a detector that
  panics causes the command to exit non-zero, surfacing the bug
  directly rather than degrading silently.

**Output**

- **FR-010**: In hybrid (default) format, each Finding MUST render as one
  header line followed by 1-3 indented lines each prefixed `   -> `
  (three spaces, arrow, space). The header line MUST contain, in this
  order: severity tag (uppercase), entity, check_id. Findings MUST be
  sorted by severity (critical, warning, info) and then by entity name
  within a severity.
- **FR-011**: When no findings remain after filtering, stdout MUST be
  exactly `No findings.\n` and the exit code MUST be 0.
- **FR-012**: In yaml and json formats, output MUST be a single
  list/array whose items are records carrying all eight fields defined
  under Key Entities. Field names MUST be byte-identical across both
  formats.
- **FR-013**: All human messaging not intended for parsing (warnings,
  errors, hints) MUST go to stderr; stdout is reserved for finding
  output exclusively.

**AI enrichment**

- **FR-014**: With AI enrichment enabled, the `recommendation` field MUST
  be populated from the fixture file
  `cmd/juju/advisor/testdata/findings.json` keyed by check_id. With
  `--no-ai` the field MUST carry the terse hand-written recommendation
  that lives in the detector definition.
- **FR-015**: AI enrichment MUST modify only the `recommendation` field.
  Any difference between the AI-enriched and non-AI outputs in any other
  field is a bug.
- **FR-016**: A missing, unparseable, or check_id-incomplete fixture file
  MUST NOT cause the command to fail. The command MUST fall back to
  per-Finding hand-written text and emit one stderr line noting that
  enrichment was skipped.

**Filtering and exit code**

- **FR-017**: Severity filtering MUST be applied after detection and
  enrichment. A Finding whose severity is not in the filter set is
  omitted from output but does not affect exit code.
- **FR-018**: The exit code MUST be 0 whenever detection completed,
  irrespective of how many Findings were emitted. The command is a
  read-only inspection, not an assertion: presence of findings is not
  an error condition. Non-zero exit is reserved for command-invocation
  failures: unknown model, no read permission, malformed flags,
  controller unreachable, AND any error returned by the underlying
  `Client.Status` call (network failure, permission revoked mid-call,
  partial response). In the Status-failure case, stderr MUST contain
  `ERROR: status fetch failed: <wrapped err>` and stdout MUST be empty
  (no partial findings emitted). This is intentionally distinct from
  FR-016 (fixture-loader failure), which is graceful.

### Key Entities *(include if feature involves data)*

- **Finding**: The atomic unit of advisor output. Eight
  fields, all mandatory:
  - `check_id` (string): stable detector identifier, e.g.,
    `active-with-message`, `charm-revision-aging`, `unit-blocked-stale`.
  - `severity` (enum: `info` | `warning` | `critical`).
  - `entity` (string): Juju entity identifier, e.g., `postgresql/0` for
    a unit or `postgresql` for an application.
  - `entity_kind` (enum: `unit` | `application`).
  - `owner` (enum: `charm-author` | `operator` | `mixed` | `platform`):
    who is best placed to act on the finding.
  - `summary` (string, one line): short human-readable label of the
    violation.
  - `recommendation` (string, multi-line): action to take. Hand-written
    text under `--no-ai`; fixture-loaded text by default.
  - `protocol_ref` (string): citation of the advisor protocol-contract clause
    being violated, e.g., `protocol://advisor/4c#hook-execution`.
- **Detector**: A predicate that consumes the controller's status
  response and emits zero or more Findings. v1 ships exactly three
  detectors corresponding to Signals 1-3. Each detector owns its
  `check_id`, `severity` (or severity selector for Signal 3), `owner`,
  hand-written `summary`, hand-written `recommendation`, and
  `protocol_ref`.
- **Fixture**: A JSON file at `cmd/juju/advisor/testdata/findings.json`
  mapping `check_id` to an AI-style recommendation string. v1 ships
  with three entries -- one per detector. The fixture stands in for a
  live LLM call; the production design is out of scope for this
  iteration.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An operator unfamiliar with a model can identify all
  current findings, ordered by severity, in a single
  command invocation that completes within the time `juju status`
  completes against the same model (no perceptible additional latency).
- **SC-002**: Against a model staged to contain exactly one instance of
  each of the three signals, the command produces exactly three findings
  in every output format -- no more, no fewer.
- **SC-003**: The structured-output field set is stable: any addition,
  removal, or rename of a Finding field between the YAML and JSON
  emitters MUST fail a contract test before merge.
- **SC-004**: Downstream tooling consuming `juju advisor -o json` can
  group findings by `severity` and by `owner` with no transformation of
  field values (both enums use the exact strings the constitution
  defines).
- **SC-005**: Disabling AI enrichment via `--no-ai` changes only the
  `recommendation` field across every other field of every finding --
  verifiable by diffing the two outputs and observing diffs scoped to
  `recommendation` only.
- **SC-006**: A model with no degradations produces exactly the literal
  string `No findings.` followed by a newline on stdout and
  nothing on stderr.
- **SC-007**: Severity filter narrows the output deterministically:
  `--severity=critical` against a model with no critical findings
  produces the no-findings message and exits 0; the same flag against a
  model containing critical findings emits exactly those findings.

## Assumptions

These are the reasonable defaults this spec adopts. If any prove wrong,
the spec needs revision before plan.

- The Juju 4.0 `Client.Status` facade response carries enough
  information to evaluate all three signals (unit workload status +
  message + since timestamp; application `CanUpgradeTo` field). v1
  targets Juju 4.0 controllers only (see Clarification Q3). If any
  required field is absent on a given controller version, the detector
  for that signal returns zero findings rather than erroring.
- "Workload status" refers to the charm-reported workload status, not
  the agent status. Signals 1 and 3 are evaluated against workload
  status only.
- The fixture file ships inside the binary (bundled at build time, not
  fetched at runtime). It is not user-configurable in v1.
- The protocol reference URIs are stable strings derived from the
  advisor protocol document; they are not resolved or fetched by
  the CLI in v1.
- The command is invocable by any user with model read permission. No
  new permission gate is introduced.
- Default output format is hybrid because operators are the primary
  audience and the YAML/JSON forms are opt-in for pipelines.
- Severity-filter parsing is permissive on whitespace
  (`--severity=critical, warning` works) but strict on values.
- The command name `juju advisor` is final for v1; renaming is a v2
  concern.
- "Advisor protocol" refers to the 8-protocol contract documented in
  Section 4c of `advisor-brief.md`. Each detector's
  `protocol_ref` cites a clause from that document.

## Out of Scope

These items are explicitly excluded from this iteration. They are
captured for traceability and to prevent scope creep, not as commitments.

- Persistence of findings (no database, no file cache).
- Auto-clearing lifecycle (every invocation is a fresh read).
- Watcher mode (`--watch` flag).
- A controller-side `Health` facade -- detection is client-side only.
- Live LLM integration -- fixtures stand in.
- Integration with or modification of `juju status` output.
- Operator suppression / acknowledgement of findings.
- The remaining ~30 signals catalogued in
  `advisor-brief.md` §4b.
- Cross-model relation findings.
- K8s-specific signals (the three v1 detectors are substrate-agnostic).
