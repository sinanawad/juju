# Phase 1: Data Model

This feature has no persistent storage. The model below describes the
in-process types that flow from the API response through detection,
enrichment, filtering, and rendering.

## Types

### `Severity`

```go
type Severity string

const (
    SeverityInfo     Severity = "info"
    SeverityWarning  Severity = "warning"
    SeverityCritical Severity = "critical"
)
```

**Invariants**:
- Only these three values are emitted (constitution Principle II).
- `severityRank` provides a stable sort order: critical < warning <
  info (lower rank prints first).

```go
func (s Severity) rank() int {
    switch s {
    case SeverityCritical:
        return 0
    case SeverityWarning:
        return 1
    case SeverityInfo:
        return 2
    default:
        return 99 // unreachable; defensive
    }
}
```

### `Owner`

```go
type Owner string

const (
    OwnerCharmAuthor Owner = "charm-author"
    OwnerOperator    Owner = "operator"
    OwnerMixed       Owner = "mixed"
    OwnerPlatform    Owner = "platform"
)
```

**Invariants**: Assigned by the detector. Never modified downstream.

### `EntityKind`

```go
type EntityKind string

const (
    EntityKindUnit        EntityKind = "unit"
    EntityKindApplication EntityKind = "application"
)
```

**Invariants**: v1 detectors emit only `unit` or `application`. Future
signals may add `relation`, `machine`, `secret`, etc.; that is a v2
schema change handled by adding constants and a corresponding detector.

### `Finding`

The single atomic output of the advisor protocol observatory.

```go
type Finding struct {
    CheckID        string     `yaml:"check_id"        json:"check_id"`
    Severity       Severity   `yaml:"severity"        json:"severity"`
    Entity         string     `yaml:"entity"          json:"entity"`
    EntityKind     EntityKind `yaml:"entity_kind"     json:"entity_kind"`
    Owner          Owner      `yaml:"owner"           json:"owner"`
    Summary        string     `yaml:"summary"         json:"summary"`
    Recommendation string     `yaml:"recommendation"  json:"recommendation"`
    ProtocolRef    string     `yaml:"protocol_ref"    json:"protocol_ref"`
}
```

**Field semantics**:

| Field            | Type        | Source             | Mutability                                              |
|------------------|-------------|--------------------|---------------------------------------------------------|
| `CheckID`        | string      | Detector constant  | Immutable.                                              |
| `Severity`       | enum        | Detector (M2/M3) or selector (M4) | Immutable.                               |
| `Entity`         | string      | Detector (from FullStatus) | Immutable.                                      |
| `EntityKind`     | enum        | Detector constant  | Immutable.                                              |
| `Owner`          | enum        | Detector constant  | Immutable.                                              |
| `Summary`        | string      | Detector constant  | Immutable.                                              |
| `Recommendation` | string      | Detector constant  | Rewritable by `enricher.Enrich` unless `--no-ai`.       |
| `ProtocolRef`    | string      | Detector constant  | Immutable.                                              |

**Validation**: A package-level helper `newFinding(...)` is the only
public emission path. Detectors call it; the formatter never builds
findings. The helper panics on any empty field — this is dev-only
guardrail; runtime can't legitimately produce a zero-value Finding
because detectors hardcode all required strings.

### `Detector`

```go
type Detector func(status *params.FullStatus, now time.Time) []Finding
```

A pure function. Takes the status snapshot and the reference time
(M4's stale-blocked detector is the only one that uses `now`; others
ignore it). Returns zero or more Findings.

**Registry**:

```go
var detectors = []Detector{
    detectActiveWithMessage,    // M2
    detectCharmRevisionAging,   // M3
    detectUnitBlockedStale,     // M4
}
```

The registry is a package-level slice. Adding a future detector means
appending one entry. Order is not meaningful — findings are sorted by
severity then entity at the end of the pipeline.

### `Fixture` (M5)

```go
type fixture map[string]string // CheckID -> AI-enriched recommendation
```

Loaded once from the embedded `testdata/findings.json`. Missing keys
mean the detector's hand-written recommendation is preserved.

## Data flow

```text
                 +----------------------------+
   user types -> | NewCitizenCommand / Run    |
                 +-------------+--------------+
                               |
                               v
                 +----------------------------+
                 | Client.Status(ctx, args)   |
                 | -> *params.FullStatus      |
                 +-------------+--------------+
                               |
                               v
                 +----------------------------+
                 | for each Detector:         |
                 |   d(status, clock.Now())   |
                 |   findings = append(...)   |
                 +-------------+--------------+
                               |
                               v
                 +----------------------------+
                 | enricher.Enrich(findings)  | (skipped if --no-ai)
                 +-------------+--------------+
                               |
                               v
                 +----------------------------+
                 | severityFilter(findings)   | (no-op if flag absent)
                 +-------------+--------------+
                               |
                               v
                 +----------------------------+
                 | sort.SliceStable by        |
                 |   (severity.rank, entity)  |
                 +-------------+--------------+
                               |
                               v
                 +----------------------------+
                 | c.out.Write(ctx, findings) |
                 |  -> formatHybrid / yaml /  |
                 |     json                   |
                 +----------------------------+
```

**No state crosses invocation boundaries.** Each `juju advisor` run is
a fresh read.

## Synthetic test fixtures (informative, not committed in M0/M1)

For detector unit tests, the smallest synthetic FullStatus is
hand-built per test. Skeleton:

```go
func cleanStatus() *params.FullStatus {
    return &params.FullStatus{Applications: map[string]params.ApplicationStatus{}}
}

func activeWithMessage(appName, unitName, msg string) *params.FullStatus {
    return &params.FullStatus{
        Applications: map[string]params.ApplicationStatus{
            appName: {
                Units: map[string]params.UnitStatus{
                    unitName: {
                        WorkloadStatus: params.DetailedStatus{
                            Status: "active",
                            Info:   msg,
                        },
                    },
                },
            },
        },
    }
}
```

Field names will be confirmed against the working tree at M2 start
(see `research.md` "Field availability research" section).
