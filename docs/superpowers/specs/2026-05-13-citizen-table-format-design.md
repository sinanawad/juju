# `juju citizen` pretty default table format — design

**Date**: 2026-05-13
**Status**: Approved via brainstorming dialogue. Implementation: today.
**Scope**: Replace `juju citizen`'s default output with a dashboard-panel + 6-column table. Rename the existing arrow-notes format to `--format=verbose`.

## Goal

A glanceable, demo-grade default for `juju citizen`. Operators see at-a-glance: model context, finding counts by severity and owner, and one row per finding with severity color, entity, owner, check, age, and human summary.

## Visual contract

```
┌─ juju citizen ──────────────────────────────────────────────────────┐
│  model: norma-demo                              scanned: 13:54:21  │
│  findings: 5   ● 1 critical   ▲ 3 warning   ◆ 1 info               │
│  owners:       charm-author 5  •  operator 0  •  platform 0        │
└────────────────────────────────────────────────────────────────────┘

  SEV       ENTITY          OWNER         CHECK                  AGE     SUMMARY
  ● crit    db/0            charm-author  hook-error             5m      Recent uncaught hook exception
  ▲ warn    bad-blocked/0   charm-author  blocked-no-message     —       Blocked w/o actionable msg
  ▲ warn    bad-churn/0     charm-author  status-churn           10m     Workload status churning
  ▲ warn    bad-stuck/0     charm-author  stuck-maintenance      18m     Held maintenance w/o transition
  ◆ info    bad-active/0    charm-author  active-with-message    —       Active unit carries non-empty msg
```

**Severity glyphs**: `●` (red) crit, `▲` (yellow) warn, `◆` (cyan) info. Glyph and tag colored together.

**Columns** (fixed order, dynamic widths): SEV(8) / ENTITY(grow, cap 24) / OWNER(13) / CHECK(grow, cap 22) / AGE(7) / SUMMARY(grow, no cap). Two-space margin and two-space column separator. No pipes.

**Box drawing**: hardcoded 80-char width for v0. Box-drawing characters `─│┌┐└┘`.

**Sort**: severity rank asc → age desc within severity → entity asc → check_id asc.

**Empty state**: dashboard panel renders with `findings: 0` and a `✓ all units are good citizens` line; no table area.

**Color toggle**: color ON by default; `--no-color` flag disables ANSI escapes (glyphs remain). TTY autodetect deferred to v0.1.

## Format flag

```
juju citizen                    → table (new default)
juju citizen --format=table     → table (explicit)
juju citizen --format=verbose   → existing arrow-notes hybrid (renamed)
juju citizen --format=yaml      → YAML list (unchanged)
juju citizen --format=json      → JSON array (unchanged)
```

`--format=hybrid` is removed. Single-release breaking change; release notes call it out.

## Data model change

`Finding` gets one new field:

```go
Since *time.Time `yaml:"since,omitempty" json:"since,omitempty"`
```

Set by stateful detectors to the violation start timestamp:
- `detectStatusChurn` → oldest in-window transition's `Since`
- `detectStuckMaintenance` → the value already computed by `continuousRunStart`
- `detectAgentError` → oldest in-window error entry's `Since`

Pure detectors leave it nil; AGE column renders `—`.

## File scope

| File | Change |
|---|---|
| `finding.go` | Add `Since *time.Time` field with `omitempty`. |
| `formatter.go` | New `formatTable` + helpers (severity styling, age fmt, dashboard panel). Existing `formatHybrid` function unchanged in body. |
| `command.go` | `AddFlags` default `"hybrid"` → `"table"`. Map: drop `hybrid`, add `table` and `verbose`. Add sort tertiary by age-desc. New `--no-color` flag. |
| `detectors.go` | 3 stateful detectors set `Since` on emitted findings. |
| `testdata/three-findings.{yaml,json}` | Add `since: null` to expected output (verifies omitempty). |

## Tests (v0 essentials)

| Test | What it pins |
|---|---|
| `TestFormatTableGolden` | Byte-diff against a fixed 4-finding fixture with `--no-color`. Validates dashboard, columns, sort. |
| `TestFormatTableEmpty` | Empty slice → dashboard with `findings: 0` and checkmark line. No table rows. |
| `TestFormatTableSortAgeDesc` | Two warnings, ages 5m and 60m → 60m sorts first within warning severity. |
| `TestFormatVerboseBackcompat` | `--format=verbose` produces the existing hybrid golden byte-for-byte. |

Deferred to v0.1: TTY/NO_COLOR auto-detection, narrow-terminal collapse, age-column unit tests, `hybrid` deprecation alias.

## Out of scope

Themes, colorblind-safe palettes, 24-bit color, `--watch` mode, SUMMARY wrapping, `--no-dashboard` flag.

## Acceleration notes

- Color is unconditional ON unless `--no-color`. TTY auto-detect deferred.
- Width hardcoded at 80. Narrow-terminal collapse deferred.
- Implementation parallelized: agent writes `formatter.go`; main thread updates `finding.go`/`command.go`/`detectors.go`.
