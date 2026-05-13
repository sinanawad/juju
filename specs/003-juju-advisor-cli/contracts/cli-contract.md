# CLI Contract: `juju advisor`

This document fixes the user-facing contract of the command before any
code is written. Locking the contract here is what enables the M1
milestone to be 30 minutes -- there is no bikeshedding window during
implementation.

## Synopsis

```text
juju advisor [-m <model>] [-o yaml|json|hybrid] [--format=<fmt>] \
             [--severity=<csv>] [--no-ai]
```

- No positional arguments.
- All flags are optional.

## Flags

| Flag                  | Type        | Default    | Source        |
|-----------------------|-------------|------------|---------------|
| `-m`, `--model`       | string      | current    | modelcmd      |
| `-o`, `--format`      | enum        | `hybrid`   | cmd.Output    |
| `--severity`          | CSV enum    | unset (=all) | this command|
| `--no-ai`             | bool        | `false`    | this command  |

The first two are inherited from existing Juju CLI infrastructure and
behave exactly as in `juju status`. The last two are unique to this
command.

## Exit codes

| Code | Meaning                                                           |
|------|-------------------------------------------------------------------|
| 0    | Detection completed. Zero or more findings emitted.               |
| !=0  | Command-invocation failure (unknown model, no permission, malformed flag, controller unreachable). Standard cobra/juju exit. |

Presence of findings is NOT an error. The command is read-only
inspection (spec FR-018).

## Streams

- **stdout**: machine-consumable findings output (the chosen format) or
  the literal `No findings.\n` when zero findings remain
  after filtering, AND the format is `hybrid`.
- **stderr**: warnings and errors only (e.g. fixture file missing, AI
  enrichment skipped). Never findings.

For non-hybrid formats with zero findings, stdout contains the format's
empty-list representation:
- YAML: empty document (`null\n` or `[]\n` depending on encoder; spec
  FR-011 only mandates the literal for hybrid).
- JSON: `[]\n`.

This deviation from FR-011 is intentional and consistent with
`block.formatBlocks` which also gates the "no items" prose behind
`c.out.Name() == "tabular"`.

## Hybrid format (M1 contract -- byte-locked)

Each finding renders as exactly:

```text
<SEVERITY> <entity> [<check_id>]
   -> <line 1 of summary>
   -> <line 2 of recommendation if non-empty>
   -> <line 3 of recommendation if non-empty>
```

- `<SEVERITY>` is the severity value uppercased, padded to width 8 on
  the right (so the columns align): `INFO    `, `WARNING `, `CRITICAL`.
- `<entity>` is the entity string exactly as the detector emitted it.
- `<check_id>` is enclosed in square brackets.
- Each note line starts with exactly three spaces, then `-> `, then the
  note text.
- Note text is whichever of summary+recommendation the renderer chose:
  - Line 1: always `summary`.
  - Lines 2-3: `recommendation` split on `\n`, capped at the first two
    non-empty lines.
- Findings are separated by a blank line (one `\n` between them).
- After the last finding, no trailing blank line; one terminating `\n`.

**Example** (three findings, hybrid):

```text
CRITICAL postgresql/0 [unit-blocked-stale]
   -> Unit has been blocked for 9 days.
   -> Investigate blocking condition: charm hook message, peer state,
   -> or operator intervention required.

WARNING  nginx-ingress [charm-revision-aging]
   -> Application is behind its tracked channel.
   -> Run 'juju refresh nginx-ingress' to pick up newer revision.

INFO     postgresql/0 [active-with-message]
   -> Unit reports 'active' with a non-empty status message.
   -> Convention is that 'active' carries no message; the empty string
   -> is the visual signal of normal operation.
```

(Note: in this example postgresql/0 appears twice because two
distinct findings target the same entity. Entities are not deduplicated
across detectors.)

## YAML format

A single top-level list of finding mappings. Field order within each
mapping follows the struct declaration:

```yaml
- check_id: unit-blocked-stale
  severity: critical
  entity: postgresql/0
  entity_kind: unit
  owner: mixed
  summary: Unit has been blocked for 9 days.
  recommendation: |
    Investigate blocking condition: charm hook message, peer state,
    or operator intervention required.
  protocol_ref: protocol://advisor/4c#status/blocked-bounded
- check_id: charm-revision-aging
  severity: warning
  ...
```

Zero findings → `[]\n` or empty document, whatever the YAML encoder
emits for `[]Finding{}`.

## JSON format

A single JSON array of objects. Encoder defaults (no indentation other
than what `cmd.FormatJson` provides):

```json
[
  {
    "check_id": "unit-blocked-stale",
    "severity": "critical",
    "entity": "postgresql/0",
    "entity_kind": "unit",
    "owner": "mixed",
    "summary": "Unit has been blocked for 9 days.",
    "recommendation": "Investigate blocking condition...",
    "protocol_ref": "protocol://advisor/4c#status/blocked-bounded"
  }
]
```

Zero findings → `[]\n`.

## Sorting

Findings are sorted by:
1. `severity.rank()` ascending (critical first, info last).
2. Within a severity, `entity` ascending lexicographically.
3. Ties broken by `check_id` ascending (deterministic for tests).

The sort is stable; in the (impossible in v1) case of two findings
identical on all three keys, original detector order is preserved.

## Filtering

`--severity=<csv>` accepts a comma-separated list of `info`, `warning`,
`critical`. Whitespace around items is trimmed. Unknown values cause
non-zero exit with a stderr line of the form:

```text
ERROR invalid --severity value "bogus": must be one of info, warning, critical
```

When the flag is set, findings whose severity is not in the set are
dropped before rendering. The "no findings" message rules apply to the
post-filter set.

## --no-ai

Boolean. When set, skips the fixture lookup step. The hand-written
`recommendation` baked into each detector is rendered as-is.

## Locking statement

This contract is fixed at the start of M1. Changes between M1 and M6
require updating this document AND re-running the M1 golden-output
tests.
