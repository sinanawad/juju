---

description: "Tasks for `juju advisor` operator CLI command (specs/003-juju-advisor-cli/)"
---

# Tasks: `juju advisor` operator CLI command

**Input**: Design documents from `specs/003-juju-advisor-cli/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli-contract.md, quickstart.md (all present)

**Tests**: Included. The spec's acceptance criterion #8 ("Unit tests cover each detector predicate in isolation against a synthetic `params.FullStatus`") and constitution Gate I (Finding schema completeness) both make tests load-bearing for this feature.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story. The plan's M0-M6 milestones map onto Setup (M0) + Foundational (M1) + US1 (M2-M4) + US2/US3 (verification of foundational) + US4 (M5 filter) + US5 (M5 AI) + Polish (M6).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4, US5)
- Include exact file paths in descriptions

## Path Conventions

All work lives in `cmd/juju/advisor/` (new subpackage) plus one
registration line in `cmd/juju/commands/main.go`. Paths below are
repository-relative.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the subpackage skeleton and the type system that
every other phase reuses. This is M0 of the plan.

- [X] T001 Create directory `cmd/juju/advisor/` and subdirectories `cmd/juju/advisor/testdata/`
- [X] T002 [P] Create `cmd/juju/advisor/doc.go` with three-paragraph package doc per `AGENTS.doc-dot-go-rules.md` (paragraph 1: tl;dr "Package advisor surfaces deployment-level findings"; paragraph 2: defines the Finding concept; paragraph 3: zoom-out to `cmd/juju/status` and zoom-in to detectors)
- [X] T003 [P] Create `cmd/juju/advisor/finding.go` defining `Severity`, `Owner`, `EntityKind` typed-string enums with their constants, the `Finding` struct (8 fields, snake_case yaml+json tags), `newFinding(...)` constructor that panics on empty required fields, and the `(Severity).rank()` method per `data-model.md`
- [X] T004 [P] Create `cmd/juju/advisor/export_test.go` exporting setters for the clock (used by M4 tests) and the fixture loader (used by M5 tests)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Stand up the command shell, output dispatcher, and the
three formatters with one hardcoded synthetic finding. After this phase
`juju advisor -o json` returns a parseable list. This is M1 of the plan.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete. The command must compile, register, and render before detectors are added.

- [X] T005 Create `cmd/juju/advisor/command.go` with `advisorCommand` struct embedding `modelcmd.ModelCommandBase`, plus `NewAdvisorCommand()` wrapped by `modelcmd.Wrap` — mirror the shape of `cmd/juju/block/list.go:28-50` exactly
- [X] T006 Implement `(*advisorCommand).Info()` in `cmd/juju/advisor/command.go` returning `jujucmd.Info(&cmd.Info{Name: "advisor", Purpose: "Surface deployment-level findings.", Doc: ...})`, mirroring `cmd/juju/block/list.go:58-69`
- [X] T007 Implement `(*advisorCommand).Init(args)` in `cmd/juju/advisor/command.go` returning `cmd.CheckEmpty(args)` (no positional args per `contracts/cli-contract.md`)
- [X] T008 Register the command in `cmd/juju/commands/main.go` by adding `r.Register(advisor.NewAdvisorCommand())` alphabetically near the existing `r.Register(block.NewListCommand())` block around line 458-460
- [X] T009 Verify the M0 build: run `go build ./cmd/juju/...` from repo root and then `make juju && ~/go/bin/juju advisor --help` — must show the command in help output and exit 0
- [X] T010 [P] Create `cmd/juju/advisor/formatter.go` with `formatHybrid(writer io.Writer, value any) error` implementing the byte-locked format from `contracts/cli-contract.md` "Hybrid format" section (uppercase severity padded to width 8, three-space-arrow note prefix, blank line between findings, no trailing blank line)
- [X] T011 Implement `(*advisorCommand).SetFlags(f *gnuflag.FlagSet)` in `cmd/juju/advisor/command.go`: call `c.ModelCommandBase.SetFlags(f)`, then `c.out.AddFlags(f, "hybrid", map[string]cmd.Formatter{"hybrid": c.formatHybrid, "yaml": cmd.FormatYaml, "json": cmd.FormatJson})`
- [X] T012 Implement `(*advisorCommand).Run(ctx)` in `cmd/juju/advisor/command.go` returning a hardcoded `[]Finding` with one synthetic Signal-1-style entry; when the format is `"hybrid"` and the slice is empty, print `"No findings."` to ctx.Stdout exactly per FR-011; otherwise call `c.out.Write(ctx, findings)`
- [X] T013 [P] Create `cmd/juju/advisor/command_test.go` with `commandSuite` (using `github.com/juju/tc` per `AGENTS.md`) and `TestCitizenInfo` asserting `Name == "advisor"`, `Purpose` non-empty
- [X] T014 [P] Add `TestRunEmptyHybridPrintsLiteral` to `cmd/juju/advisor/command_test.go` — with Run() forced to return zero findings, hybrid format stdout must be exactly `"No findings.\n"` AND stderr must be empty (covers SC-006's stderr-empty contract)
- [X] T015 [P] Create `cmd/juju/advisor/formatter_test.go` with `TestFormatHybridSyntheticFindings` — golden output diff against the byte-locked example from `contracts/cli-contract.md` (one critical finding, one warning, one info)
- [X] T015a Constitution Principle V gate: grep `advisor-brief.md` §4c for the three `protocol_ref` slugs the detectors will cite (`status/active-empty-msg`, `revision/track-channel`, `status/blocked-bounded`); if any slug is absent from §4c, either rename the slug in `cmd/juju/advisor/detectors.go` to match an existing clause anchor OR amend the brief in this same PR to add the missing anchor — do not leave the citation dangling

**Checkpoint**: `juju advisor` runs; `juju advisor -o json | jq .` returns a JSON array; the hybrid format is byte-locked by golden test; all three `protocol_ref` slugs resolve to clauses in the brief. Foundation ready -- user story implementation can now begin.

---

## Phase 3: User Story 1 - Operator triages an unfamiliar model (Priority: P1) 🎯 MVP

**Goal**: An operator running `juju advisor` against any model sees real, sorted findings for the three v1 signals. This is the MVP described in the spec's User Story 1 and corresponds to milestones M2-M4 in the plan.

**Independent Test**: Deploy a charm whose unit reports `active` with a non-empty message (e.g., `grafana-k8s`). Run `juju advisor`. Output is exactly one info-severity finding for the unit with the correct check_id and entity.

### Tests for User Story 1

> Per `AGENTS.md`: use `github.com/juju/tc` checkers. Detectors are pure functions over `*params.FullStatus`, so tests need no controller.

- [X] T016 [P] [US1] Verify `params.FullStatus` field names against the working tree (4.0 branch): grep `rpc/params/` for `FullStatus`, `ApplicationStatus`, `UnitStatus`, `DetailedStatus` and record the exact field names (`CanUpgradeTo`, `WorkloadStatus.Status`, `WorkloadStatus.Info`, `WorkloadStatus.Since`) in a comment block at the top of `cmd/juju/advisor/detectors.go`; if any field is renamed in 4.0, update the corresponding detector and note the divergence in `specs/003-juju-advisor-cli/research.md`
- [X] T017 [P] [US1] Create `cmd/juju/advisor/detectors_test.go` with helper `cleanStatus()` returning `*params.FullStatus{Applications: map[string]params.ApplicationStatus{}}` and helper `unitStatus(app, unit, workload, msg, since)` per the skeleton in `data-model.md`
- [X] T018 [P] [US1] Add `TestDetectActiveWithMessage` to `cmd/juju/advisor/detectors_test.go` covering three cases: clean status returns zero findings; one matching unit returns one finding with severity=info, owner=charm-author, entity_kind=unit, check_id="active-with-message"; whitespace-only message also counts as non-empty per spec edge case
- [X] T019 [P] [US1] Add `TestDetectCharmRevisionAging` to `cmd/juju/advisor/detectors_test.go`: zero apps → zero findings; one app with non-empty `CanUpgradeTo` → one finding severity=warning owner=operator entity_kind=application check_id="charm-revision-aging"
- [X] T020 [P] [US1] Add `TestDetectUnitBlockedStaleBoundaries` to `cmd/juju/advisor/detectors_test.go` using `testclock.NewClock` (or `clock.NewClock(fixed)` per `CODING.md`): fixed time at 2026-05-13T12:00:00Z; unit blocked at T-23h59m59s → zero findings; T-24h0m1s → one warning; T-7d → one warning (boundary); T-7d-1s → one critical; T+1m (clock skew) → zero findings (per spec edge case)

### Implementation for User Story 1

- [X] T021 [US1] Create `cmd/juju/advisor/detectors.go` with the `Detector` type alias `func(*params.FullStatus, time.Time) []Finding` and the registry `var detectors = []Detector{...}` per `data-model.md`
- [X] T022 [P] [US1] Implement `detectActiveWithMessage(*params.FullStatus, time.Time) []Finding` in `cmd/juju/advisor/detectors.go` with hardcoded constants for `checkID="active-with-message"`, summary, hand-written recommendation, protocol_ref=`protocol://advisor/4c#status/active-empty-msg`, owner=`OwnerCharmAuthor`, severity=`SeverityInfo`, entity_kind=`EntityKindUnit`; defend against nil `Applications` map, nil per-app `Units` map, and zero-value `WorkloadStatus` struct — any missing field MUST yield zero findings, never a panic (FR-009)
- [X] T023 [P] [US1] Implement `detectCharmRevisionAging(*params.FullStatus, time.Time) []Finding` in `cmd/juju/advisor/detectors.go` with `checkID="charm-revision-aging"`, severity=`SeverityWarning`, owner=`OwnerOperator`, entity_kind=`EntityKindApplication`, protocol_ref=`protocol://advisor/4c#revision/track-channel`; defend against nil `Applications` map and missing `CanUpgradeTo` field (empty string is the no-finding signal, never a panic) (FR-009)
- [X] T024 [P] [US1] Implement `detectUnitBlockedStale(*params.FullStatus, time.Time) []Finding` in `cmd/juju/advisor/detectors.go` with the duration-based severity selector (>24h && <=7d → warning; >7d → critical), owner=`OwnerMixed`, entity_kind=`EntityKindUnit`, protocol_ref=`protocol://advisor/4c#status/blocked-bounded`; nil `Since` pointer is treated as not-stale (zero findings)
- [X] T025 [US1] Add the `statusAPI` interface and the `getStatusAPI(ctx)` helper to `cmd/juju/advisor/command.go` mirroring `cmd/juju/status/status.go:33-35,211-228` — interface has `Status(ctx, *client.StatusArgs) (*params.FullStatus, error)` and `Close() error`; helper opens `c.NewAPIRoot(ctx)` and returns `client.NewClient(root, logger)`
- [X] T026 [US1] Replace the hardcoded findings in `(*advisorCommand).Run` (cmd/juju/advisor/command.go) with: open `statusAPI`, defer Close, call `Status(ctx, nil)`, dispatch each detector with `c.clock.Now()`, accumulate `[]Finding`, then `sort.SliceStable` by `severity.rank()` then `entity` then `check_id` per `contracts/cli-contract.md` sorting rules. If `Status(ctx, nil)` returns a non-nil error, return `errors.Annotate(err, "status fetch failed")` so the wrapped error reaches stderr in the exact FR-018 format. Do NOT wrap detector calls in `defer recover()` — FR-009 mandates hard-fail on detector panic.
- [X] T027 [US1] Add `clock` field to `advisorCommand` struct in `cmd/juju/advisor/command.go`, initialize to `clock.WallClock` in `NewAdvisorCommand()`, and expose a setter in `export_test.go` so detector tests can inject `testclock.NewClock(fixed)`
- [X] T028 [US1] Add `TestRunDispatchesAllDetectors` to `cmd/juju/advisor/command_test.go` using a fake `statusAPI` returning a synthetic FullStatus with all three signals; assert three findings emitted, sorted critical→warning→info
- [X] T028a [P] [US1] Add `TestRunStatusFailureHardError` to `cmd/juju/advisor/command_test.go` — inject a fake `statusAPI.Status()` returning `errors.New("controller unreachable")`; assert non-zero exit, stderr matches `ERROR: status fetch failed: controller unreachable` exactly, stdout is empty. Covers FR-018 Status-failure contract.

**Checkpoint**: User Story 1 fully functional. `juju advisor` against a real or staged model emits the three signal types. This is the MVP demoable artifact.

---

## Phase 4: User Story 2 - Operator pipes findings into downstream tooling (Priority: P2)

**Goal**: Structured YAML and JSON output is field-stable; downstream consumers can group by severity/owner without value translation.

**Independent Test**: Run `juju advisor -o json | jq 'map(keys) | unique[0] | sort'` against a model with at least one finding. Output is the sorted 8-key array exactly matching the data-model spec.

### Tests for User Story 2

- [ ] T029 [P] [US2] Add `TestFormatYAMLGoldenThreeFindings` to `cmd/juju/advisor/formatter_test.go` — build a fixed `[]Finding{critical, warning, info}` and diff against a checked-in golden file at `cmd/juju/advisor/testdata/three-findings.yaml`
- [ ] T030 [P] [US2] Add `TestFormatJSONGoldenThreeFindings` to `cmd/juju/advisor/formatter_test.go` — same input, diff against `cmd/juju/advisor/testdata/three-findings.json`
- [X] T031 [P] [US2] Add `TestFindingFieldSetExactlyEight` to `cmd/juju/advisor/formatter_test.go` — marshal one Finding to JSON, unmarshal to `map[string]any`, assert `len(map) == 8` and the key set equals exactly `{check_id, severity, entity, entity_kind, owner, summary, recommendation, protocol_ref}` — this enforces SC-003 (field-set stability) at the contract level

### Implementation for User Story 2

- [ ] T032 [P] [US2] Create `cmd/juju/advisor/testdata/three-findings.yaml` (golden file) matching the canonical example in `contracts/cli-contract.md` "YAML format" section
- [ ] T033 [P] [US2] Create `cmd/juju/advisor/testdata/three-findings.json` (golden file) matching the canonical example in `contracts/cli-contract.md` "JSON format" section

**Checkpoint**: Structured output is contract-tested. SC-003 is enforced at merge time.

---

## Phase 5: User Story 3 - Operator inspects a model other than the current one (Priority: P2)

**Goal**: `-m <model>` works the same way it works for `juju status`.

**Independent Test**: With current model A, run `juju advisor -m B` against a controller hosting both. Output reflects B's state; the operator's current model context remains A.

### Tests for User Story 3

- [ ] T034 [US3] Add `TestModelFlagPassedToAPI` to `cmd/juju/advisor/command_test.go` — inject a fake API factory that records the model name received; run command with `-m alternate-model`; assert recorded model name matches; this verifies `modelcmd.ModelCommandBase` plumbs `-m` through unchanged
- [ ] T035 [US3] Add `TestModelFlagMissingExitsNonZero` to `cmd/juju/advisor/command_test.go` — inject a fake API factory returning a `NotFound` error; assert non-zero exit and stderr contains the model name (verifies the standard modelcmd error path)

### Implementation for User Story 3

No new production code: `-m`/`--model` comes free from `modelcmd.Wrap` (already in Phase 2). The two tasks above are pure verification.

**Checkpoint**: Cross-model inspection verified.

---

## Phase 6: User Story 4 - Operator narrows attention to actionable severities (Priority: P3)

**Goal**: `--severity=<csv>` filters findings post-detection.

**Independent Test**: Against a model with all three signals, `juju advisor --severity=critical` emits only the critical finding (if any) and exits 0. `--severity=bogus` exits non-zero with a clear stderr message.

### Tests for User Story 4

- [X] T036 [P] [US4] Add `TestSeverityFilterSingleValue` to `cmd/juju/advisor/command_test.go` — fixture with one finding per severity; `--severity=critical` produces one finding; `--severity=info` produces one finding; `--severity=warning,critical` produces two
- [X] T037 [P] [US4] Add `TestSeverityFilterRejectsInvalidValue` to `cmd/juju/advisor/command_test.go` — `--severity=bogus` returns non-zero, stderr message matches the spec format `ERROR invalid --severity value "bogus": must be one of info, warning, critical`
- [X] T038 [P] [US4] Add `TestSeverityFilterWhitespaceTolerance` to `cmd/juju/advisor/command_test.go` — `--severity=critical, warning` and `--severity= info ` both parse correctly

### Implementation for User Story 4

- [X] T039 [US4] Implement `severitySet` type in `cmd/juju/advisor/command.go` as a `map[Severity]bool` with `String()` (for gnuflag) and `Set(value string)` methods that split on comma, trim whitespace per element, validate each against the three constants, and return a typed error naming the invalid input
- [X] T040 [US4] Wire `severitySet` into `SetFlags` in `cmd/juju/advisor/command.go` via `f.Var(&c.severityFilter, "severity", "Filter findings by severity (comma-separated)")`
- [X] T041 [US4] Apply the filter inside `Run()` in `cmd/juju/advisor/command.go` immediately after enrichment and immediately before the sort step; when the filter set is empty (flag omitted), all findings pass through

**Checkpoint**: Severity filter works and is tested.

---

## Phase 7: User Story 5 - Operator works offline or AI-skeptically (Priority: P3)

**Goal**: `--no-ai` skips fixture enrichment. With the flag absent, the fixture rewrites the `recommendation` field; with it present, the hand-written recommendation from each detector is used unchanged. Missing fixture degrades gracefully.

**Independent Test**: Run `juju advisor -o yaml` and `juju advisor --no-ai -o yaml` against the same model. Diff: only the `recommendation` field differs across findings; all other fields are byte-identical.

### Tests for User Story 5

- [X] T042 [P] [US5] Add `TestEnrichApplies` to `cmd/juju/advisor/enricher_test.go` (new file) — given `[]Finding` with three check_ids and a fixture map covering all three, assert each recommendation is replaced with the fixture value
- [X] T043 [P] [US5] Add `TestEnrichLeavesUnmatched` to `cmd/juju/advisor/enricher_test.go` — given a Finding whose check_id is not in the fixture, the recommendation is unchanged
- [X] T044 [P] [US5] Add `TestEnrichGracefulFallbackOnMalformedFixture` to `cmd/juju/advisor/enricher_test.go` — given an invalid JSON byte slice as the fixture source, `Enrich` returns the input findings unchanged AND records a warning that the caller can route to stderr
- [X] T045 [P] [US5] Add `TestNoAIPreservesHandwritten` to `cmd/juju/advisor/command_test.go` — run command twice with the same fixture model (once with `--no-ai`, once without); the only field that differs is `recommendation`; all eight findings of each side have matching `check_id`, `severity`, `entity`, `entity_kind`, `owner`, `summary`, `protocol_ref`

### Implementation for User Story 5

- [X] T046 [P] [US5] Create `cmd/juju/advisor/testdata/findings.json` containing three entries keyed by `check_id` (`active-with-message`, `charm-revision-aging`, `unit-blocked-stale`), each value a paragraph-length AI-style recommendation (max 3 sentences per spec assumption)
- [X] T047 [P] [US5] Create `cmd/juju/advisor/enricher.go` (a.k.a. `fixtures.go` per the plan) with `//go:embed testdata/findings.json` for `fixtureBytes []byte`, a package-level `sync.Once`-guarded `loadFixture() (map[string]string, error)`, and `Enrich(findings []Finding, fixture map[string]string) []Finding` returning a new slice with rewritten recommendations
- [X] T048 [US5] Add `noAI bool` field to `advisorCommand` and register `--no-ai` in `SetFlags` with `f.BoolVar(&c.noAI, "no-ai", false, "Disable AI-enriched recommendations")`
- [X] T049 [US5] In `(*advisorCommand).Run` in `cmd/juju/advisor/command.go`: after detection and before severity filtering, when `!c.noAI`, call `loadFixture()`; on success call `Enrich(findings, fixture)`; on failure emit one stderr line `WARNING compliance: AI enrichment skipped: <err>` and continue with un-enriched findings

**Checkpoint**: AI enrichment works; `--no-ai` works; missing fixture is non-fatal.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Final smoothing — integration suite, doc.go content, lint clean.

- [ ] T050 [P] Run `gci write --section standard --section default --section "Prefix(github.com/juju/juju)" cmd/juju/advisor/*.go cmd/juju/commands/main.go` per `AGENTS.md`
- [ ] T051 [P] Run `make pre-check` from repo root; resolve any golangci-lint findings in `cmd/juju/advisor/`
- [X] T052 [P] Run `go test -race ./cmd/juju/advisor/...` — must pass with `-race` per `AGENTS.md`
- [X] T053 [P] Flesh out `cmd/juju/advisor/doc.go` paragraphs 2-3 with the data-flow diagram from `data-model.md` (ASCII art is permitted per `AGENTS.doc-dot-go-rules.md`)
- [ ] T054 Create `tests/suites/advisor/task.sh` that bootstraps microk8s, deploys `grafana-k8s` (predictable active-with-message charm), runs `juju advisor -o json`, and asserts `jq 'length >= 1'` and `jq '.[] | .check_id' | grep active-with-message`; register suite in `tests/main.sh`
- [ ] T055 Run `quickstart.md`'s "30-second demo" end-to-end against a fresh microk8s controller and verify each step produces the documented output

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 Setup**: No dependencies. Tasks T001-T004 can start immediately.
- **Phase 2 Foundational (M0+M1)**: Depends on Phase 1 completion. **BLOCKS all user stories.** Tasks T005-T015a (T015a is the Principle V slug-verification gate).
- **Phase 3 User Story 1 (P1, MVP)**: Depends on Phase 2 completion.
- **Phase 4 User Story 2 (P2)**: Depends on Phase 2 completion (uses foundational format dispatcher). Independent of US1's detectors but more meaningful with them.
- **Phase 5 User Story 3 (P2)**: Depends on Phase 2 completion. Independent of US1, US2.
- **Phase 6 User Story 4 (P3)**: Depends on Phase 2 completion. The filter operates on any `[]Finding`, so independent of US1 detectors for unit tests.
- **Phase 7 User Story 5 (P3)**: Depends on Phase 2 completion. Independent for unit tests; meaningful end-to-end after US1 detectors land.
- **Phase 8 Polish**: Depends on whichever user stories were completed.

### User Story Dependencies

- **US1**: No story dependencies; pure addition.
- **US2**: No story dependencies. Most meaningful demo after US1 produces real findings, but tests use synthetic fixtures so unit tests run standalone.
- **US3**: No story dependencies. Verification of inherited `modelcmd` behavior.
- **US4**: No story dependencies. Filter logic is independent of detectors at the unit-test layer.
- **US5**: No story dependencies. Enricher is a pure transform over `[]Finding`.

### Within Each User Story

- Tests are written FIRST and must FAIL before implementation (per `AGENTS.md` discipline, where tests exist).
- Detector predicate tests (T018, T019, T020) can run in parallel — different test functions, same file → parallel-safe in Go's testing model.
- Detector implementations (T022, T023, T024) can run in parallel — they're separate function bodies appended to the same file; resolve append order by file diff merge.

### Parallel Opportunities

- **Phase 1**: T002, T003, T004 in parallel (different files).
- **Phase 2**: T010, T013, T014, T015 in parallel (different files); T005-T012 are sequential because they share `command.go`.
- **Phase 3 tests** (T017-T020): all in parallel (different test functions; same file's append order is mechanical).
- **Phase 3 implementations** (T022-T024): all in parallel as long as merge order is coordinated.
- **Phase 4 golden files** (T032, T033): parallel.
- **Phase 4 tests** (T029, T030, T031): parallel.
- **Phase 6 tests** (T036, T037, T038): parallel.
- **Phase 7 tests** (T042-T045): parallel.
- **Phase 7 implementations** (T046, T047): parallel; T048, T049 sequential (share command.go).
- **Phase 8** (T050-T053): all parallel.

---

## Parallel Example: User Story 1 Detector Tests

```bash
# After T016 (field verification) and T017 (test helpers) land,
# launch all three detector test functions in parallel:
Task: "Add TestDetectActiveWithMessage in detectors_test.go"
Task: "Add TestDetectCharmRevisionAging in detectors_test.go"
Task: "Add TestDetectUnitBlockedStaleBoundaries in detectors_test.go"

# And in parallel with those, the three implementations:
Task: "Implement detectActiveWithMessage in detectors.go"
Task: "Implement detectCharmRevisionAging in detectors.go"
Task: "Implement detectUnitBlockedStale in detectors.go"
```

---

## Implementation Strategy

### MVP First (M0 → M1 → M2)

The competition-defensible minimum is Phase 1 + Phase 2 + the first
detector slice of Phase 3:

1. **Phase 1 Setup** (T001-T004) — 20 min.
2. **Phase 2 Foundational** (T005-T015a) — 35 min (T015a adds a brief Principle V slug-verification gate).
3. **Phase 3 partial — Signal 1 only** (T016, T017, T018, T021, T022, T025, T026, T027) — 30 min.
4. **STOP AND VALIDATE**: `juju advisor` against `grafana-k8s` emits one info-severity finding for `grafana-k8s/0`. **Demo this.**

Total time to MVP: ~80 min.

### Incremental Delivery

After the MVP, add detectors and presentation features one at a time:

1. MVP → demo Signal 1.
2. Add Signal 2 (T019, T023) → demo charm-revision-aging.
3. Add Signal 3 (T020, T024) → demo blocked-stale with boundary tests.
4. Add structured-output contract tests (Phase 4) → SC-003 hardening.
5. Add `--severity` (Phase 6) → operator filter.
6. Add `--no-ai` + fixture (Phase 7) → constitutional AI-optional path.
7. Polish (Phase 8) → integration suite + lint.

Each step ends with a demoable binary. No checkpoint requires re-doing earlier work.

### Drop-Out Points

Per the plan's milestone discipline:
- **After T026 (M2 done)**: MVP shipped. Can stop here and present.
- **After T028 (US1 complete)**: All three detectors live. Stronger demo.
- **After T041 (US4 complete)**: Filter UX added. Operator story complete.
- **After T049 (US5 complete)**: AI enrichment + constitutional `--no-ai`. Full v1.
- **After T055 (Polish complete)**: Production-ready.

### Parallel Team Strategy

If multiple agents/developers are available after Phase 2 completes:

- Developer A: User Story 1 (P1, the detectors — highest value).
- Developer B: User Story 4 (severity filter — independent codepath).
- Developer C: User Story 5 (AI enricher — independent codepath).

US2 and US3 are verification-only and best done by whoever finishes their primary story first.

---

## Notes

- [P] tasks = different files OR different test functions, no dependencies on incomplete tasks.
- [Story] label maps task to specific user story for traceability.
- Each user story is independently completable and testable in isolation.
- Tests are explicitly requested (spec AC #8 + constitution Gate I); verify they fail before implementing.
- After every Go file edit, run `gci` per `AGENTS.md` (consolidated in T050 at the end).
- Stop at any checkpoint to validate the current increment.
- Avoid: novel abstractions (constitution Principle VIII), new facades (constitution Principle X), modifying `params.FullStatus` (constitution Principle X).
