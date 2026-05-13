<!--
SYNC IMPACT REPORT
==================
Version change: 0.0.0 (template, placeholders only) -> 1.0.0
Bump rationale: MAJOR. First concrete ratification. The prior file was the
unfilled template; this commit replaces every placeholder with binding
content and adopts a 10-principle structure plus Mission and Non-goals
sections, which is a governance establishment rather than an amendment.

Modified principles: n/a (no prior named principles existed)
Added principles:
  I. Findings Are First-Class Queryable Data
  II. Severity Is Calibrated For Operational Impact
  III. Owner Classification Is Load-Bearing
  IV. Runtime Observation Is Uniquely Necessary
  V. The Advisor Protocol Is Articulated Separately
  VI. AI Enrichment Is A Layered, Optional Transformer
  VII. Auto-Clearing Lifecycle
  VIII. Follow Juju Conventions Ruthlessly
  IX. Detection Layer Placement Is An Implementation Detail
  X. Backwards Compatibility Is Preserved

Added sections:
  Mission
  Non-Goals
  Reference Documents
  Governance

Removed sections: n/a

Templates requiring updates:
  .specify/templates/plan-template.md  - Constitution Check section updated
                                          with concrete gates derived from
                                          Principles I, II, III, V, VIII, X (next step).
  .specify/templates/spec-template.md  - No structural change required;
                                          spec authors must cite contract
                                          clauses in functional requirements
                                          when the feature emits Findings
                                          (enforced at plan/checklist time).
  .specify/templates/tasks-template.md - No structural change required.
  .specify/templates/checklist-template.md - No structural change required.

Follow-up TODOs:
  None. RATIFICATION_DATE set to today (initial adoption).
-->

# Juju Operator Advisor Constitution

## Mission

The Juju Operator Advisor surfaces violations of the implicit
charm-Juju operational contract -- degradations caused by external factors
(charms, infrastructure) that are invisible to today's `juju status`
surface. The platform itself is assumed correct; the observatory measures
everything else.

## Core Principles

### I. Findings Are First-Class Queryable Data

Findings MUST be structured records, not freeform strings. Every Finding
MUST carry, at minimum: `severity` (one of `info`, `warning`, `critical`),
`owner` (one of `charm-author`, `operator`, `mixed`, `platform`), `entity`
(the targeted application/unit/relation/machine identifier), `summary` (a
short human-readable label), `recommendation` (a structured action), and
`contract_clause` (the citation of the contract clause violated). Code that
emits a Finding without all six fields MUST fail validation at the
detection-layer boundary. CLI, API, and storage layers MUST be able to
filter and sort on every field.

**Rationale**: Freeform diagnostic strings cannot be triaged, aggregated,
or auto-cleared. Structure is what turns an alert into a workflow.

### II. Severity Is Calibrated For Operational Impact

Severity MUST reflect operational impact, not technical detail. The three
levels are defined as follows and MUST NOT be redefined per-detector:
`info` = convention violation with no functional impact; `warning` =
degrading state requiring action within a sprint; `critical` = data
integrity, security, or hard breakage. New detectors MUST map their output
to one of these three levels; detectors MUST NOT introduce new severity
values.

**Rationale**: A flat, operator-meaningful severity scale is the only way
the dashboard remains actionable as detectors multiply.

### III. Owner Classification Is Load-Bearing

Every Finding MUST be classified to an `owner` chosen from
`charm-author | operator | mixed | platform`. The classifier MUST run at
the detection layer, not the CLI. Detectors that cannot confidently
classify ownership MUST emit `mixed` rather than guessing. CLI views MUST
allow filtering by owner. Operators MUST be able to suppress
`charm-author`-owned Findings from their own attention surface.

**Rationale**: Roughly half of violations are charm-author
fixes; surfacing them indiscriminately to operators turns the observatory
into noise.

### IV. Runtime Observation Is Uniquely Necessary

The detection layer MUST operate against live runtime state (status, model
config, relation data, observed events). Static-only analysis MUST NOT be
the sole signal for any v1 detector that targets temporal,
scale-dependent, environment-dependent, or peer-dependent symptoms. Static
analysis remains a valid complementary product; it does NOT belong inside
this observatory.

**Rationale**: Approximately 70% of symptoms are not derivable
from charm code. The observatory's value proposition is exactly the
symptoms a linter cannot see.

### V. The Advisor Protocol Is Articulated Separately

The advisor protocol MUST live in the codebase as a referenceable
document and MUST be addressable by stable clause IDs. Every Finding MUST
cite at least one clause ID. Pull requests that add a new detector MUST
either cite an existing clause or extend the contract document in the same
change. Operators and charm authors MUST never be flagged for a rule they
cannot read.

**Rationale**: Citation is what makes findings disputable, learnable, and
versionable. A rule without a published clause is not a rule.

### VI. AI Enrichment Is A Layered, Optional Transformer

AI enrichment MUST be a transformer applied AFTER Finding emission, not a
producer of Findings. The detection layer MUST produce complete,
operator-actionable Findings without any enrichment. The enricher MAY
rewrite the `recommendation` field with richer prose; it MUST NOT modify
`severity`, `owner`, `entity`, `summary`, or `contract_clause`. The CLI,
API, and storage layers MUST behave identically whether or not enrichment
ran. Enrichment failures MUST NOT block Finding emission.

**Rationale**: AI quality varies and AI access is not universal. The
contract guarantees a working product when enrichment is absent.

### VII. Auto-Clearing Lifecycle

When full v1 ships, Findings MUST open, update, and close automatically
based on the next observation that confirms or refutes the violating
state. Manual close MUST NOT be the primary resolution path. Findings MUST
expose `first_seen` and `last_seen` timestamps to allow operators to
distinguish flapping from durable conditions. The prototype MAY skip
auto-clearing; v1 MUST NOT.

**Rationale**: An observatory that requires manual housekeeping decays
into staleness within a release.

### VIII. Follow Juju Conventions Ruthlessly

New facades, commands, and schema MUST follow existing Juju patterns. New
client commands MUST use the shape established by `cmd/juju/block/list.go`
(positional args, `--format` flag accepting yaml/json, `cmd.Output` for
machine-readable output, separate stdout/stderr discipline). New facades
MUST be implemented in `apiserver/facades/` with thin orchestration only;
business logic MUST live in `domain/`. Bulk arguments are MANDATORY for
facade methods that operate on entity sets. v1 MUST NOT introduce novel
abstractions; deviation from established patterns requires an explicit
entry in the plan's Complexity Tracking table.

**Rationale**: The observatory ships inside Juju. Re-litigating Juju's
architecture inside this project is out of scope and will not pass review.

### IX. Detection Layer Placement Is An Implementation Detail

The data contract -- the Finding record schema and its semantics -- is
load-bearing. The location of the code that produces Findings is NOT.
Prototype detectors MAY live client-side. v1 detectors MUST live
controller-side behind a `Health` facade. The schema, severity scale,
owner enumeration, and contract-clause citation requirement MUST be
byte-identical across both implementations. Migration from client-side to
controller-side MUST NOT require any change to consuming CLI or API
clients beyond endpoint configuration.

**Rationale**: Locking placement is what makes the prototype-to-v1
migration a refactor rather than a rewrite.

### X. Backwards Compatibility Is Preserved

Existing Juju CLI commands and existing facade contracts MUST NOT break.
New fields added to `params.FullStatus` (or any other existing wire type)
MUST be tagged `omitempty` and MUST have a sensible zero-value semantic.
The introduction of the observatory MUST NOT change the output of
`juju status` for users who do not opt in to observatory features. Removal
of any existing field requires a facade version bump and a migration
strategy documented in the plan.

**Rationale**: Juju is a long-lived distributed system with strict
compatibility expectations. A new feature that breaks existing clients is
not a feature; it is an incident.

## Non-Goals

These boundaries are binding. Pull requests proposing work in any of these
areas MUST be rejected as out of scope.

- The observatory does NOT measure workload health. Workload health is the
  charm's responsibility and is surfaced via COS or Pebble probes.
- The observatory does NOT replace `juju status`. It complements it.
- The observatory does NOT lint charm source. That is the scope of a
  separate Juju Advisor Lint product.
- The observatory does NOT modify charm code, relations, configuration,
  or any other model state on the operator's behalf. It is read-only.

## Reference Documents

The companion brief at `advisor-brief.md` (repository
root) is the authoritative product context. Specifications, plans, and
checklists MUST cite this brief when introducing requirements derived
from it. Authoritative sections:

- Sections 1-3: opportunity framing.
- Section 4b: 33-signal inventory with verdicts.
- Section 4c: 8-protocol advisor protocol (the contract referenced by
  Principle V; clause IDs originate here).
- Section 4e: ~50 violation symptoms with severity/owner/action mapping
  (the seed corpus for detector specification).
- Appendix A: hackathon prototype scope.
- Appendix B: pre-canned AI fixtures (the substitute for live AI during
  prototype demos; see Principle VI).

The Juju repository's `AGENTS.md`, `AGENTS.architecture-rules.md`,
`AGENTS.core-domain-rules.md`, and the `~/dev/juju-brain/JUJU.md`
knowledge base remain the authoritative source for Juju coding
conventions referenced by Principle VIII. Where this constitution and a
Juju AGENTS rule both apply, Juju's rules govern implementation detail
and this constitution governs product scope.

## Governance

This constitution supersedes all other project practices within the scope
of the Juju Operator Advisor. It does NOT supersede Juju's own
architectural and coding rules; where those rules apply (per Principle
VIII), they take precedence over any conflicting detail in this document.

**Amendment procedure**: amendments MUST be proposed as a pull request
modifying this file. The PR description MUST state the bump type (MAJOR /
MINOR / PATCH) and justify it. Amendments MUST update the Sync Impact
Report comment at the top of this file and propagate any required changes
to templates under `.specify/templates/`.

**Versioning policy**: MAJOR for backward-incompatible governance changes
or principle removals or redefinitions. MINOR for new principles or
materially expanded guidance. PATCH for clarifications, wording fixes, or
non-semantic refinements.

**Compliance review**: every `/speckit-plan` Constitution Check MUST gate
on Principles I, II, III, V, VIII, and X at minimum. Specs that emit
Findings MUST cite the contract clause they target before the plan can
proceed past Phase 0. The `advisor-brief.md` reference is
load-bearing and MUST exist in the repository root; CI MUST fail if it is
removed without amending this constitution.

**Version**: 1.0.0 | **Ratified**: 2026-05-13 | **Last Amended**: 2026-05-13
