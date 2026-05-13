# Phase 0: Research

## Open questions from Technical Context

The spec was concrete; the only open implementation questions were
around Juju conventions, which the constitution Principle VIII pins to
established patterns. Each decision below cites the source.

### Decision: Command shape

**Decision**: Mirror `cmd/juju/block/list.go`.
- `NewAdvisorCommand()` returns `modelcmd.Wrap(&advisorCommand{...})`.
- `advisorCommand` embeds `modelcmd.ModelCommandBase` (gives `-m`, API
  root resolution, controller selection, login).
- `Info() *cmd.Info` returns `jujucmd.Info(...)`.
- `SetFlags(f *gnuflag.FlagSet)` calls `c.ModelCommandBase.SetFlags(f)`,
  then `c.out.AddFlags(f, "hybrid", map[string]cmd.Formatter{...})`,
  then registers custom flags (`--severity`, `--no-ai`).
- `Init(args []string)` returns `cmd.CheckEmpty(args)` (no positional
  args).
- `Run(ctx *cmd.Context)` opens API root, calls Status, runs detectors,
  enriches, filters, writes via `c.out.Write(ctx, value)`.

**Rationale**: Constitution Principle VIII mandates the
`block/list.go` shape verbatim. Every detail above is copied from that
file (lines 28-138).

**Alternatives considered**: None. Constitutional gate.

### Decision: How to call Status

**Decision**: Use the existing `Client.Status(ctx, *client.StatusArgs)`
facade method, accessed via:

```go
import "github.com/juju/juju/api/client/client"

func (c *advisorCommand) getStatusAPI(ctx context.Context) (statusAPI, error) {
    root, err := c.NewAPIRoot(ctx)
    if err != nil {
        return nil, errors.Trace(err)
    }
    return client.NewClient(root, logger), nil
}

type statusAPI interface {
    Status(context.Context, *client.StatusArgs) (*params.FullStatus, error)
    Close() error
}
```

This is exactly the `statusAPI` interface defined in
`cmd/juju/status/status.go:33-35` of the current tree.

**Rationale**: Already used by `juju status`. No new facade required
(constitution Principle X). Verified via:
`grep -n "Status(" cmd/juju/status/status.go` and
`grep -n "statusAPI" cmd/juju/status/status.go`.

**Alternatives considered**:
- Calling `controller.Client.AllModels` then `Client.Status` per model.
  Rejected: scope is single-model per invocation.
- A new `Health` facade. Rejected: constitution Principle IX defers
  this to a later iteration; v1 is client-side.

### Decision: Output format set and flag wiring

**Decision**: Register three formats via `cmd.Output.AddFlags`:

```go
c.out.AddFlags(f, "hybrid", map[string]cmd.Formatter{
    "hybrid": c.formatHybrid,
    "yaml":   cmd.FormatYaml,
    "json":   cmd.FormatJson,
})
```

The default is `"hybrid"`; users can request YAML or JSON via `-o yaml`
or `-o json` (or `--format=yaml/json`). This is the same wiring
`block.listCommand.SetFlags` uses (`list.go:74-79`), except the default
there is `"tabular"`.

**Rationale**: `cmd.Output` is the canonical juju formatter dispatcher
(see `CODING.md` "CLI Implementation Thoughts" — never write your own,
use `cmd.Output`). Reusing it gives us free `-o` and `--format` parsing
plus stdout discipline.

**Alternatives considered**: Hand-rolling format dispatch. Rejected per
`CODING.md`.

### Decision: AI fixture loading

**Decision**: Bundle the fixture at build time via `//go:embed`.

```go
//go:embed testdata/findings.json
var fixtureBytes []byte

type fixture map[string]string // check_id -> recommendation
```

The file path is exactly the spec-mandated
`cmd/juju/advisor/testdata/findings.json`. The map is loaded once per
invocation in a package-level `sync.Once` or simply on first Run() call.

**Rationale**: Spec Out of Scope excludes live LLM. Embed avoids any
runtime filesystem dependency (a `--no-ai`-equivalent fallback is
already specified for missing fixtures, but with embed the fixture is
present unless the JSON itself is malformed).

**Alternatives considered**:
- Reading from `~/.local/share/juju/advisor-fixtures.json`. Rejected:
  extra user setup.
- Calling a real LLM endpoint. Rejected: explicitly out of scope.

### Decision: Clock injection for the time-based detector

**Decision**: Use `github.com/juju/clock` per `CODING.md` "time.Now Is
The Winter Of Our Discontent."

```go
type advisorCommand struct {
    modelcmd.ModelCommandBase
    out      cmd.Output
    clock    clock.Clock // defaults to clock.WallClock in NewAdvisorCommand
    // ...
}
```

`export_test.go` exposes a setter so tests can substitute
`clock.NewClock(fixedTime)` or `testclock.NewClock(fixed)`.

**Rationale**: `CODING.md` mandates this. Without it, the M4 detector
test is timing-dependent and flaky.

**Alternatives considered**: Direct `time.Now()`. Rejected — explicitly
forbidden by repo coding rules.

### Decision: Contract clause IDs

**Decision**: Each detector cites a stable URL-like string of the form
`protocol://advisor/4c#<slug>`.

| Detector              | check_id                | protocol_ref slug          |
|-----------------------|-------------------------|----------------------------|
| Signal 1 (M2)         | `active-with-message`   | `status/active-empty-msg`  |
| Signal 2 (M3)         | `charm-revision-aging`  | `revision/track-channel`   |
| Signal 3 (M4)         | `unit-blocked-stale`    | `status/blocked-bounded`   |

If the brief's §4c uses different anchor names, the planner reconciles
in the same commit per constitution Principle V.

**Rationale**: Constitution Principle V mandates that every Finding
cites a contract clause. The slugs above are derived from the topic of
each signal and are stable across iterations.

**Alternatives considered**: Numeric clause IDs (e.g. `4c.3`).
Rejected: brittle to reordering.

### Decision: Severity filter parsing

**Decision**: A `gnuflag.Var` implementation that splits on comma,
trims whitespace per element, validates against the three constants,
and stores `map[Severity]bool`. Empty filter (flag omitted) means
"include all". Filtering is applied after enrichment so that AI text
is never wasted on a finding that gets dropped.

**Rationale**: Spec FR-004, FR-017. Standard juju pattern (cf.
`cmd/juju/status/status.go` `patterns` field).

**Alternatives considered**: Multiple `--severity` invocations
(`--severity=warning --severity=critical`). Rejected: spec wants CSV.

### Decision: Where to register the command

**Decision**: `cmd/juju/commands/main.go`, in `registerCommands`,
alphabetically near the existing `block.New*Command()` entries (line
458-460 of the current file). Single line:

```go
r.Register(advisor.NewAdvisorCommand())
```

**Rationale**: Verified by `grep registerCommands cmd/juju/commands/main.go`
— the function is the central registry for all client-CLI commands.

**Alternatives considered**: None.

### Decision: Field names on the wire (Finding serialisation)

**Decision**: Snake_case JSON tags matching the spec's Key Entities
section verbatim: `check_id`, `severity`, `entity`, `entity_kind`,
`owner`, `summary`, `recommendation`, `protocol_ref`. YAML uses the
same names via the `yaml:"…"` tag.

**Rationale**: Snake_case is the standard for structured CLI output in
Juju (see `params.Block` -> `command-set`, `params.ModelBlockInfo` ->
`model-uuid`). The spec's Key Entities section is authoritative.

**Alternatives considered**: CamelCase. Rejected: inconsistent with
Juju CLI norms.

## Field availability research (deferred to M2 start)

The spec assumes the following `params.FullStatus` shape:

- `FullStatus.Applications map[string]ApplicationStatus`.
- `ApplicationStatus.CanUpgradeTo string` — used by Signal 2.
- `FullStatus.Applications[...].Units map[string]UnitStatus`.
- `UnitStatus.WorkloadStatus DetailedStatus` — `.Status` and
  `.Info` (message) and `.Since *time.Time` (when status was set).

The actual field names on branch `4.0` MUST be verified before the
first detector is written. The verification is logged as the first
action of M2, not here, so this research doc stays implementation-
agnostic. If a field is renamed or missing, the affected detector
short-circuits (FR Assumption — "absence of field = zero findings, not
error").

## Out of Scope (research level)

The following are deliberately not researched in Phase 0 because the
spec's Out of Scope section excludes them from v1:

- Live LLM provider integration.
- Watcher API for `--watch` mode.
- Persistence schema.
- Controller-side `Health` facade design.
- Cross-model integration.
- The remaining ~30 brief-catalogued signals.
