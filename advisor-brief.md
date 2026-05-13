# Opportunity Brief: Juju Operator Advisor

**Status:** Opportunity research (Stage A). NOT a design or implementation plan.
**Date:** 2026-05-12
**Audience:** Director of Engineering, Juju & Ops framework (Canonical).

---

## 1. Context

A new platform capability for Juju 4.0 is being investigated. The initial
framing was an "invariant-checking system" that detects when Juju's internal
model has diverged from observable reality. That framing was explored at
depth in this session (Stage A investigation) and was determined to be
correct in shape but wrong in primary purpose: the highest-leverage signals
in Juju's controller are about **charm behaviour and external-factor
degradation**, not about Juju's own correctness.

The reframed motivation:

> Measure things that could be degraded by external factors (charms, infra)
> while assuming Juju itself is correct. Preemptive, day-2 ops, focused on
> chronic drift AND charm behaviour that is inconsistent with Juju paradigms
> (e.g., a charm that is not consuming its event queue in a timely manner).

This file is the consolidated opportunity brief and a plan for the *next*
stages of work (research/spec/design/build). It is intentionally
implementation-agnostic.

---

## 2. Chosen direction — Juju Operator Advisor

A continuous evaluation of how well each deployed charm behaves as a Juju
advisor — i.e., conforms to Juju's operational contract — independent of
whether the workload itself is healthy. Surfaces deviation as structured
findings with severity, scope, cause, and suggested remediation.

The Operator Advisor is NOT:
- An invariant check for Juju's internal correctness (Juju is assumed correct).
- A workload-health tool (that's the charm's job and/or COS's job).
- A deployment-time linter (we want day-2 signal, not deploy-gate).
- A COS replacement (we ride alongside; COS measures the workload, we measure
  the charm-as-Juju-advisor).
- A generic K8s/IaaS observability tool (no parity goal with kube-state-metrics).

It draws from chronic-drift surfacing as a byproduct (e.g., a charm whose
queue is backing up will also fail to rotate secrets on time — so queue
degradation is a leading indicator for secret-expiry pile-up).

---

## 3. Why this direction

Three reasons. All informed by parallel research in this session
(`AGENTS.md` cross-references at end of file).

### 3.1 Uniquely defensible

No adjacent platform/orchestration tool measures this. Kube-state-metrics
measures Pods, Polaris/kube-score lint manifests, ArgoCD measures Git-vs-cluster
drift, Datadog Watchdog measures workload anomalies, AWS Trusted Advisor flags
best-practice gaps. None of them measure conformance to a curated charm
operational contract — because none of those platforms HAVE such a contract.

Juju does. The charm-Juju interface (hooks, leases, relations, secrets,
status, events) is a unique platform asset. Observability into conformance
with it is the part of the product that justifies Juju existing as a layer
on top of K8s/IaaS.

### 3.2 Existing signal is rich; existing surface is thin

The Juju 4.0 controller already tracks substantial behavioural signal that
never reaches an operator:

- Agent-presence timestamps (`unit_agent_presence`, `machine_agent_presence`
  in the model DQLite DB).
- Lease renewal cadence + holder identity (`lease` table in controller DB).
- Status-update timestamps per unit/agent/workload/k8s-pod
  (`unit_agent_status.updated_at`, `unit_workload_status.updated_at`).
- Change-stream throughput and subscription counts
  (`internal/worker/changestream/metrics.go`).
- Dependency-engine worker restart counters (`agent/engine/metrics.go`).
- Removal-scheduling timestamps (proxy for "Dying entered at T") in the
  `removal` table.
- Uniter per-unit event queue and operation history (currently in-memory in
  the uniter worker; not exposed).
- API server request-duration histograms keyed by facade/method.

The gap matrix has 21 categories of operator-health information. Nine are
full GAP. Eight are PARTIAL. The existing surface (`juju status`,
`debug-log`, status enum, COS) covers liveness and lifecycle phase well, and
charm self-assessment freeform, but covers degradation reason / severity
grading / external-cause taxonomy / cross-entity correlation / trend / drift
detection / fleet rollup very poorly.

### 3.3 Strategic story

The Drift Catalog alone reads as "Juju copying AWS Trusted Advisor."
Commodity functionality. The Operator Advisor reframes Juju's value:

> We tell you when your operators are degrading, not just when your
> workloads are.

That's a load-bearing reason to keep using Juju over rolling-your-own
operators on raw K8s.

---

## 4. Signal landscape (high-level)

This is an initial enumeration. **It is NOT a v1 scope.** A Stage B
investigation would prune and prioritise. Items in *italics* are signals
Juju internally tracks today but does not expose to operators.

### 4.1 Hook execution behaviour
- *Hook completion latency vs the charm's own historical baseline*
- *Hook outcome distribution (success / error / failed / pending) per unit*
- *Hook scheduling drift (queued hooks vs real-time — has the uniter fallen
  behind?)*
- *Per-unit operation queue depth (pending operations in the uniter state
  manager)*
- Idempotency violations (config-changed firing without observable
  downstream effect — requires new instrumentation)

### 4.2 Charm-Juju API interaction
- *Status-set cadence (how often does each charm publish workload status?)*
- *Relation-data-bag write frequency and content size*
- *Secret-get / secret-info-get / secret-add cadence*
- *Leader-elected hook handling latency*
- *Open-port / close-port / rebooted hook tool call patterns*

### 4.3 Leader lease behaviour
- *Renewal margin: time-to-expiry at each successful renewal*
- *Holder flap rate over a moving window (requires either persisted history
  or a worker-owned rolling buffer)*
- *Pinned-but-stuck leases*

### 4.4 Relation health (orthogonal to relation status enum)
- *Relation-settings-hash staleness (when was the data last touched)*
- *Per-side hook-firing parity for CMR (the original invariant #1 — moved
  here; it's a advisor signal, not an invariant)*
- *Departed-but-not-broken-yet relations stuck in lifecycle*

### 4.5 Secret compliance
- *Owned-secret rotation cadence vs declared `rotate-policy`*
- *Expired-but-not-rotated secrets (counts per charm)*
- *Consumer-secret revision lag (how many revisions behind latest?)*

### 4.6 Resource hygiene
- *Per-application leaked K8s RBAC triples (StatefulSet + accumulated
  Role/RoleBinding/ServiceAccount from past consumer-adds)*
- *PVCs orphaned by removed applications*
- *Application charm-revision age in months vs available newer revision in
  same channel*

### 4.7 External-dependency reach (observed through Juju)
- *Vault / secret-backend reachability latency (sampled by the controller)*
- *Charmhub refresh-API reachability*
- *OCI registry pull latency for the charm's resources*
- *Cloud-provider credential validity (currently a binary `Invalid` flag —
  could be enriched with "expiring in N days")*

---

## 4b. Phase B.1 — Signal inventory results

Read-only investigation across 33 candidate signals. Verdict distribution:

| Verdict | Count | Meaning |
|---|---|---|
| ALREADY-VISIBLE | 7 | Operator can see this today (sometimes inadvertently) |
| TRACKED-BUT-HIDDEN | 17 | Data exists in dqlite or worker memory; no operator surface |
| NOT-TRACKED | 9 | Data does not exist anywhere in Juju |

### Headline pattern — persistence asymmetry

Lease, relation, and secret state tables are temporally rich (start,
expiry, `updated_at`, `next_rotate_at`, `current_revision`) — but renewal
margins, holder flap rates, consumer revision lag, and "due-soon"
assessments are computed client-side or never computed. CMR macaroon
expiries are sealed inside opaque `TEXT` blobs and only decoded at use
time. Charmhub call success/failure exists as in-memory Prometheus
counters but is never persisted. Vault backend reachability and dqlite
peer reachability are not tracked at all.

The data structure for most advisor signals already exists; the
queries, views, and operator surface for them don't.

### Full inventory table

| # | Signal | Tracked at | Persisted | Visibility | Verdict |
|---|---|---|---|---|---|
| 4.1.1 | Hook start/end timestamps per unit | `internal/worker/uniter/operation/runhook.go:~154` | `unit_workload_status.updated_at` (side effect, not per-hook) | STATUS (coarse) | TRACKED-BUT-HIDDEN |
| 4.1.2 | Hook outcome per unit | `internal/worker/uniter/op_callbacks.go:102`; `domain/status/` | `unit_agent_status` + freeform message | STATUS | TRACKED-BUT-HIDDEN |
| 4.1.3 | Operation queue depth | `internal/worker/uniter/resolver.go:59-164` | in-memory only | LOG-ONLY | TRACKED-BUT-HIDDEN |
| 4.1.4 | Hook scheduling drift | not tracked | — | NONE | NOT-TRACKED |
| 4.1.5 | Status-set call cadence | `apiserver/facades/agent/uniter/status.go:97-99` | per-call timestamp, no counter | STATUS | TRACKED-BUT-HIDDEN |
| 4.1.6 | Relation-data-bag write frequency | `domain/relation/state/relation.go` | per-write to `relation_unit_setting`, no rate | FACADE | TRACKED-BUT-HIDDEN |
| 4.1.7 | Secret hook-tool call cadence | `apiserver/facades/agent/secretsmanager/secrets.go` | per-call DB write, no counter | FACADE | TRACKED-BUT-HIDDEN |
| 4.1.8 | Per-agent API call rate | `apiserver/observer/metricobserver/metricobserver.go:77-94` | Prometheus, in-memory | PROMETHEUS | ALREADY-VISIBLE (no agent_identity label) |
| 4.1.9 | Last-uniter-activity timestamp | `unit_agent_presence.last_seen` (`0020-unit.sql:219-225`) | dqlite | STATUS | ALREADY-VISIBLE |
| 4.1.10 | Hook-failure pause/retry state | `unit_resolved` table (`0020-unit.sql:336-355`) | dqlite | STATUS (via `juju resolved`) | ALREADY-VISIBLE |
| 4.3.1 | Lease renewal margin | `lease.start`/`lease.expiry` (`0002-lease.sql:13-24`) | dqlite (start of *current* holder; not per-renewal) | LOG-ONLY | TRACKED-BUT-HIDDEN |
| 4.3.2 | Lease holder flap rate | `lease.holder` mutations | dqlite (current value only; no history) | NONE | TRACKED-BUT-HIDDEN |
| 4.3.3 | Pinned-but-stuck leases | `lease_pin` (`0002-lease.sql:32-47`) | dqlite | NONE | NOT-TRACKED (no anomaly detector) |
| 4.4.1 | Relation settings hash staleness | `relation_unit_settings_hash`, `relation_application_settings_hash` (`0024-relation.sql:133-188`) | dqlite hash; no timestamp | LOG-ONLY | TRACKED-BUT-HIDDEN |
| 4.4.2 | CMR hook firing parity | `internal/worker/remoterelationconsumer/worker.go` | in-memory only | LOG-ONLY | NOT-TRACKED |
| 4.4.3 | Stuck transitional relations | `relation_status.updated_at` (`0024-relation.sql:206-219`) | dqlite | STATUS (via `v_relation_status`) | ALREADY-VISIBLE |
| 4.5.1 | Owned-secret rotation cadence vs policy | `secret_rotation.next_rotation_time`, `secret_metadata.rotate_policy_id` (`0012-secret.sql:42-65`) | dqlite | LOG-ONLY | TRACKED-BUT-HIDDEN |
| 4.5.2 | Expired-but-not-rotated secrets count | `domain/secret/state/state.go` (data present, no service method) | dqlite | NONE | TRACKED-BUT-HIDDEN |
| 4.5.3 | Consumer-secret revision lag | `secret_unit_consumer.current_revision` (`0012-secret.sql:186-206`) | dqlite | LOG-ONLY | TRACKED-BUT-HIDDEN |
| 4.5.4 | CMR macaroon expiry | `application_remote_offerer.macaroon` opaque TEXT (`0034-cross-model-relation.sql:23-65`) | opaque blob; expiry parsed only at use | NONE | NOT-TRACKED |
| 4.5.5 | Relation suspension reason | `relation.suspended_reason` (`0024-relation.sql:79-80`) | dqlite freeform TEXT | STATUS | ALREADY-VISIBLE (unstructured) |
| 4.6.1 | Per-app K8s RBAC accumulation | `application_k8s_resources_managed` (`0054-application-k8s-resources.PATCH.sql:1-7`) | dqlite (flag only, no per-consumer count) | LOG-ONLY | TRACKED-BUT-HIDDEN |
| 4.6.2 | Orphan PVCs | `internal/provider/kubernetes/volume.go` | reactive cleanup, no registry | NONE | NOT-TRACKED |
| 4.6.3 | Charm-revision age | `charm.create_time` (`0015-charm.sql:73`) | dqlite | LOG-ONLY | ALREADY-VISIBLE (no last-refresh) |
| 4.6.4 | CanUpgradeTo | `apiserver/facades/client/client/status.go:1062-1063` | per-request in-memory | FACADE (in `ApplicationStatus`) | TRACKED-BUT-HIDDEN (no staleness) |
| 4.6.5 | Agent binary version drift | `agent_version` (`0027-agent-version.sql:17-22`) + `machine_agent_version` (`0018-machine.sql:93-100`) | dqlite | STATUS | ALREADY-VISIBLE |
| 4.7.1 | Vault / secret-backend reachability | `internal/worker/secretbackendrotate/` | not probed | NONE | NOT-TRACKED |
| 4.7.2 | Charmhub refresh-API reachability | `internal/charmhub/refresh.go` + `core/charm/metrics` | in-memory metrics, ephemeral | PROMETHEUS (if scraped) | TRACKED-BUT-HIDDEN |
| 4.7.3 | OCI registry pull latency / failure | `internal/docker/registry/registry.go`; K8s provisioner | not tracked | LOG-ONLY | NOT-TRACKED |
| 4.7.4 | Cloud-provider credential validity | `cloud_credential.invalid` + `invalid_reason` (`0005-cloud.sql:134-153`) | dqlite (no `last_validated_at`) | FACADE | TRACKED-BUT-HIDDEN |
| 4.7.5 | Controller-cluster quorum / dqlite peer | `internal/worker/dbaccessor/worker.go` | in-memory cluster list | NONE | NOT-TRACKED |
| 4.7.6 | Idle FD count on controller | `apiserver/apiservermetrics.go:123-135` | Prometheus | PROMETHEUS | TRACKED-BUT-HIDDEN |
| 4.7.7 | Object-store hash-ref count | `object_store_metadata` + `object_store_metadata_path` (`0006-objectstore-metadata.sql:13-19`) | dqlite (path list, no aggregate) | LOG-ONLY | TRACKED-BUT-HIDDEN |
| 4.7.8 | Provider-tracker restart-loop signal | `internal/worker/providertracker/` | in-memory engine state | NONE | TRACKED-BUT-HIDDEN |

### What this implies for v1

- **v1 is mostly a query-and-surface project**, not an instrumentation project. The 17 TRACKED-BUT-HIDDEN signals plus the 7 ALREADY-VISIBLE signals (re-presented as findings rather than freeform status) cover 24 of 33 candidates with **no new schema migrations and no new agent instrumentation**.
- **The 9 NOT-TRACKED signals are v2+.** Each needs new schema or new instrumentation. Highest-value of these for future cohorts: CMR hook firing parity, pinned-stuck lease detection, Vault reachability probing.
- **Three signals merit special note:**
  1. *Per-agent API call rate* is the only signal already emitted as Prometheus, but labels exclude agent identity. Adding an `agent_identity` label (one place in `apiserver/observer/metricobserver/metricobserver.go`) unlocks per-charm API behavior with near-zero effort.
  2. *`unit_resolved` table* is the cleanest existing precedent for "structured behavioral signal in dqlite, exposed via CLI command" (`juju resolved` reads it). The v1 surface can be modelled after this pattern.
  3. *`agent_version` table* already supports drift detection at the schema level. Surfacing "you have N agents on old binaries" is a service-layer query against existing rows.

---

## 4c. Phase B.A — The implicit Juju advisor protocol

The observatory measures conformance to a contract Juju has *never written down in one place*. The contract is distributed across `docs/reference/hook.md`, `docs/reference/status.md`, the per-tool docs under `docs/reference/hook-command/`, the Operator-framework SDK guidance, and a layer of Discourse-folklore. The articulation below is the synthesis of that contract surface — one half of what compliance checks evaluate against. Each protocol carries three subsections: what Juju *guarantees* to the charm, what Juju *expects* in return, and the *violation symptoms* (these are CANDIDATES for observatory checks; not yet a v1 scope).

### 4c.1 Hook firing protocol

**Juju guarantees:**
- Documented lifecycle ordering. Install: `storage-attached` → `install` → `config-changed` → `start`. Teardown: `stop` → `storage-detaching` ∪ `relation-broken` → `remove`. Operation phase has no ordering guarantee across event classes (`docs/reference/hook.md:58-99`).
- Relation hook sub-ordering is strict per remote unit: `relation-created` → `relation-joined` → `relation-changed` (`docs/reference/hook.md:173-186`).
- At-least-once delivery; the uniter re-fires hooks on retry from error state (`internal/worker/uniter/operation/runhook.go:154-177`).
- Hook-tool effects are transactional: `relation-set`, `open-port`, `status-set` apply only on successful (exit-0) hook completion.

**Juju expects (the contract):**
- Hooks MUST be idempotent and safe to re-run from the start on transient failure (`docs/reference/hook.md:103-113`).
- Charms MUST NOT assume prior hook execution except where ordering is documented.
- Charms MUST handle hook-tool effects as transactional batches at hook-exit, not mid-hook.
- During teardown, charms MUST clean up; persistent state belongs in charm config, relations, secrets, or declared storage — not local disk.

**Violation symptoms:**
- Hook crashes on re-fire (idempotency break).
- Service restart on every `config-changed` regardless of whether config actually changed.
- Hook hangs >5 minutes with no `maintenance` status update (folklore boundary; no formal timeout).
- Charm assumes `pebble-ready` runs once; reconfiguration race on pod churn (`docs/reference/hook.md:687-696`).
- Forced upgrade from error state silently skips `upgrade-charm`; data migrations declared there go unrun (LP#2068500).

### 4c.2 Status protocol

**Juju guarantees:**
- Status enum semantics at `docs/reference/status.md` and `core/status/status.go:47-176`.
- Charm-set workload statuses: `active`, `blocked`, `waiting`, `maintenance`, `unknown` (Juju-default), `error` (Juju-set on hook crash), `terminated` (Juju-set during removal).
- Application status is the highest-priority unit workload status unless leader explicitly sets it.

**Juju expects:**
- Charm MUST set a workload status by end of `install` (`docs/reference/status.md:71`).
- `active` means "all services running"; conventionally NO message (ops issue #126).
- `blocked` MUST carry an actionable message — explicit human action required.
- `waiting` means external dependency not ready; no human action.
- `maintenance` means long-running non-error work in progress.
- Only the leader may call `status-set --application` (`docs/reference/hook-command/list-of-hook-commands/status-set.md:38-49`).

**Violation symptoms:**
- Unit remains `unknown` indefinitely.
- `active` with a non-empty message.
- `blocked` with empty / unactionable message.
- Status churn — rapid flips between states within seconds.
- Non-leader writes application status.

### 4c.3 Lease / leadership protocol

**Juju guarantees:**
- A unit observing `is-leader: True` is leader for **at least 30 seconds** (`docs/reference/hook-command/list-of-hook-commands/is-leader.md:22-23`).
- Renewal is pull-based: the leadership tracker proactively renews on a `duration/2` schedule (`internal/worker/leadership/tracker.go:231-254`).
- `leader-elected` fires on transition.

**Juju expects:**
- Non-leaders MUST defer writes to application-scoped data (application relation data, application-owned secrets).
- Charms SHOULD assume leadership can change at any time; handlers SHOULD be idempotent across transitions.
- Hooks on the leader threaten the lease — if a hook runs longer than `lease-duration`, the lease can be lost mid-hook.

**Violation symptoms:**
- Leadership flap (>1 holder change per 30s window).
- Non-leader writes to application databag (silently ignored).
- Leader's `leader-elected` hook runs >2×lease-duration; next election blocked.

### 4c.4 Relation protocol

**Juju guarantees:**
- Unit databag per unit; application databag per application. Each unit reads all remote databags; only its own unit databag (and, if leader, own application databag) is writable (`docs/reference/relation.md:100-110`).
- `relation-changed` fires on remote databag change; settings-version hashing is internal to Juju and not exposed.
- `relation-broken` is guaranteed AFTER all `relation-departed`, with relation data still readable for cleanup (`docs/reference/hook.md:198-208`).
- In peer relations, the leader does NOT receive `relation-changed` for its own writes.
- CMR follows the same protocol with one twist: remote application names are obfuscated as `remote-<token>` (`docs/reference/relation.md:70-71`).

**Juju expects:**
- Charms MUST gracefully handle missing keys in `relation-get` (other than `private-address`) (`docs/reference/hook-command/list-of-hook-commands/relation-get.md:59-68`).
- `relation-departed` is the cleanup opportunity; relation data is still readable.
- Charm SHOULD NOT overwrite `private-address` unless serving as a proxy.

**Violation symptoms:**
- Non-leader attempts application-databag writes.
- Charm doesn't handle `relation-departed`; stale entries persist.
- Excessive `relation-changed` handler cost — handler runs longer than the relation's update interval; backlog accumulates.
- Cryptographic material in plaintext relation data (anti-pattern — secrets should use the Secret backend).

### 4c.5 Secret protocol

**Juju guarantees:**
- Juju fires `secret-rotate` on the owner at the declared rotation interval. The charm MUST create a new revision via `secret-set` in that hook.
- Juju fires `secret-remove` on the owner once all consumers stop tracking an old revision. The charm MUST remove the revision or `secret-remove` re-fires (`docs/reference/secret.md:207-213`).
- `secret-get` returns the consumer's tracked revision by default; `--refresh` adopts latest; `--peek` reads latest without adopting.
- For CMR consumers, secret access is revoked when the relation breaks (`docs/reference/secret.md:186-189`).
- Backend abstraction: charms see no difference between internal and Vault backends.

**Juju expects:**
- Owners rotate on schedule when `--rotate` is declared.
- Owners remove old revisions on `secret-remove`.
- Consumers refresh via `secret-get --refresh` on `secret-changed`; SHOULD NOT pin to an old revision.

**Violation symptoms:**
- Unbounded secret revision growth (owner ignores `secret-remove`).
- Rotation declared but `secret-rotate` hook errors or takes no action; old credentials never retired.
- Consumer caches a secret value past `secret-changed`; reads stale data.
- CMR consumer accesses a secret after `relation-broken`; access denied at runtime.

### 4c.6 Storage protocol

**Juju guarantees:**
- IaaS machines: `storage-attached` fires BEFORE `install`. CAAS: fires AFTER `start` (`docs/reference/hook.md:62-69, 680-681`).
- Volume is available at `JUJU_STORAGE_LOCATION` when the hook fires.
- `storage-detached` / `storage-detaching` fires before `remove`.
- Persistence model is pool-dependent.

**Juju expects:**
- Charms MUST mount/format/use storage only after `storage-attached`.
- Charms MUST flush/unmount cleanly in `storage-detaching`.

**Violation symptoms:**
- Charm accesses storage before attach hook completes.
- No flush on detach; data loss on reattach.
- Charm assumes persistence in a non-durable pool; data lost on unit destruction.

### 4c.7 Action protocol

**Juju guarantees:**
- Action handler runs with `JUJU_ACTION_NAME` / `UUID` / `REVISION` set; results captured via `action-set` / `action-fail` / `action-log`.
- Synchronous default since Juju 2.8.

**Juju expects:**
- Long-running actions (>30s) SHOULD adopt an async pattern (return job-ID, allow polling).
- Action handlers SHOULD honour user-supplied parameters in metadata.

**Violation symptoms:**
- Action hangs indefinitely; no `action-fail` / no exit.
- Action result missing expected keys.
- Action mutates shared state concurrently with hooks (race).

### 4c.8 SDK-recommended-but-not-enforced layer

Layered on top of the controller-side contract; the prime "implicit compliance" targets because Juju has the vantage to observe violations the SDK alone can't enforce.

| Recommendation | Source | Observable symptom |
|---|---|---|
| Hooks must be idempotent / re-entrant | Ops framework docs, `docs/reference/hook.md:103-113` | Same write twice on re-run; duplicate side effects |
| `ActiveStatus` must have no message | ops issue #126 (folklore-canonised) | `active` + non-empty message |
| Hooks should complete within ~5 minutes | FOLKLORE (no formal timeout) | Hook duration >5m with no `maintenance` status update |
| Rapid status churn discouraged | FOLKLORE | >1 status change per 5 seconds |
| Don't put secrets in relation data | FOLKLORE, Discourse | High-entropy strings in relation databag |
| Relation data should be small | FOLKLORE | Per-unit relation databag >1MB |
| Pebble probes drive workload readiness | Discourse "Health checks in Pebble" | `update-status` used as workload health poll |
| Structured JSON logs with topology labels | Charmcraft extension docs | Plaintext logs / missing topology labels |
| COS observability relations as baseline | Ops tutorial "Observe with COS Lite" | No `prometheus_scrape` / `grafana_dashboard` / `loki_push_api` |
| Long-running actions should be async | Discourse "Juju Actions UX 2.8" | Sync action running >10 minutes |
| Config-changed must handle deprecated keys | FOLKLORE | Hook fails on removed config key |

### 4c.9 Contract → Inventory cross-walk

Mapping the protocol surfaces above onto the 33-signal inventory in Section 4b:

| Protocol | Inventory signals (Section 4b numbering) |
|---|---|
| Hook firing | 4.1.1, 4.1.2, 4.1.3, 4.1.4, 4.1.5, 4.1.10 |
| Status | 4.1.5, 4.4.3 |
| Lease/leadership | 4.3.1, 4.3.2, 4.3.3 |
| Relations | 4.1.6, 4.4.1, 4.4.2, 4.4.3 |
| Secrets | 4.1.7, 4.5.1, 4.5.2, 4.5.3, 4.5.4 |
| Storage | (mostly not in inventory; 4.6.2 orphan-PVC adjacent) |
| Actions | (not in inventory — gap) |
| SDK recommendations | observable via 4.1.8 (per-agent API rate with new agent_identity label) and the 4.2.x cadence signals |

Storage and Action protocols are under-instrumented today. They're candidates for v2+ instrumentation, not v1.

### 4c.10 Two foundational takeaways

1. **The contract is articulable but distributed.** No canonical "Juju compliance spec" exists; the SDK side is half-written-down and half-folklore. Writing this synthesis up as a first-class document (separate from the observatory) is itself a value-add for the charm-author community and reduces the observatory's accusation surface — operators and authors can't be flagged for a rule they never read about.
2. **The hardest-to-detect violations are timing and idempotency.** Status protocol violations and secret rotation violations are quasi-static and dqlite-queryable. Hook idempotency and rapid status churn are temporal and need windowed analysis — the advisor protocol-vs-invariant pivot we made earlier. Compliance observation MUST handle windowed signals natively, not as an afterthought.

---

## 4d. Static vs runtime — why a runtime observatory is necessary

A natural challenge to this work: if symptoms can be detected by static analysis of charm code, why build runtime infrastructure? The honest answer required classifying each symptom by detectability.

### What static analysis catches in the Juju ecosystem today (close to nothing)

- **`charmcraft analyze`**: validates metadata (`metadata.yaml`, `charmcraft.yaml`) — schema, relation/storage/device structure, manifest, entry-point file existence. **Zero Python code analysis.** No AST, no `pylint`, no `mypy`, no type-hint enforcement.
- **Ops framework** (`ops.testing`): state-transition test harness, not lint. Built-in consistency checker validates event plausibility, not code patterns.
- **Charmhub publish-time validation**: metadata + artifact integrity. No code review, no security scan in the public pipeline.
- **Juju server-side validation at deploy time** (`domain/deployment/charm/meta.go:727-818`): validates metadata schema only. Does NOT verify hook handlers exist or that the code is correct.
- The legacy `charm proof` tool (deprecated) did do hook-implementation-presence checks. Nothing replaced it.

**Net: zero static analysis of charm Python code in the ecosystem today.** This is itself a gap — and a distinct opportunity from the observatory.

### Per-symptom classification (Section 4c symptoms)

| Tier | Count | Examples |
|---|---|---|
| EASILY STATIC | ~5 (10%) | `ActiveStatus("non-empty")`, missing COS libs in `charmcraft.yaml`, missing Pebble plan when CAAS charm declares workload, action handler ↔ metadata schema mismatch, missing `relation-departed` handler |
| PARTIALLY STATIC (false-positive prone) | ~10 (20%) | Service restart without diff-check; non-leader app-databag write; missing `secret-remove` handler; consumer caches secret across `secret-changed`; `relation-get` access without `.get()` fallback; high-entropy strings in `relation-set` |
| HARDLY STATIC | ~15 (30%) | Hook idempotency in general (data-flow analysis intractable); CMR consumer accesses secret post-`relation-broken`; storage access before attach completes; action concurrency races |
| IMPOSSIBLE STATIC | ~20 (40%) | Anything temporal (flap rate, hook duration vs baseline, churn, scheduling drift); anything scale-dependent (latency at N units, queue depth, relation-changed cost); anything environment-dependent (Vault/OCI/Charmhub reachability, credential expiry observed); anything peer/CMR-dependent (revision lag, freshness across sides) |

**Roughly 70% of Section 4c symptoms are not statically detectable.** They emerge from the deployment context — they don't exist in the charm artifact.

### The complementarity case

Adjacent-ecosystem precedent confirms layering, not replacement. K8s ships kube-score (static manifest analysis) AND OPA/Gatekeeper (admission-time policy) AND Trivy-operator (runtime continuous scanning) — all three coexist. ArgoCD ships pre-sync hooks AND continuous health checks. The pattern is universal: static catches what static can; runtime catches the rest.

For Juju, three distinct products serve three distinct audiences and lifecycles:

| Surface | Audience | When | What it catches |
|---|---|---|---|
| **Juju Advisor Lint** (complementary product, not this brief) | Charm authors | Pre-publish, CI | The 5–15 statically-detectable symptoms; catches issues before they ship |
| **Charmhub publish gate** (lint extension) | Charm authors | At upload | Same lints, server-side enforcement |
| **Juju Operator Advisor** (this brief) | Operators | Day-2, post-deploy | The 35–40 temporal / scale / env / peer-dependent symptoms — what only runtime can see |

**Why runtime is strictly necessary, restated:** every signal in Section 4b's inventory marked TRACKED-BUT-HIDDEN is by definition a runtime fact. Hook latency, queue depth, lease renewal margin, secret revision lag, agent presence freshness, RBAC accumulation — none of these exist in the charm artifact. Static analysis cannot derive them in principle.

A charm can pass every lint and still degrade in production because:
- its hooks happen to take 3× longer on the operator's cloud,
- its peer charm rotates secrets faster than this charm refreshes,
- the operator runs 50 units when the charm was authored against 5,
- the Vault backend takes 8s to respond instead of 80ms,
- the offerer model in a CMR is itself degraded.

None of those are visible to a code reader. All are visible to a runtime observer.

### Implication for this work

The observatory is uniquely positioned to catch the 70% no other tool can reach. The static-analysis 10–15% should not block this work — they're a complementary product. A reasonable team structure is one for the lint (small, charm-author-facing), one for the observatory (this brief). They cross-pollinate but don't depend on each other for value.

---

## 4e. Operator action loop — symptom → action mapping

The observatory will produce findings; each finding has to lead the operator to a specific next move. Without this mapping, severity grading is hand-wavy and the surface becomes noise. This section maps every Section 4c violation symptom onto:

- **severity** — info / warning / critical (calibrated for operational impact; some are judgement calls flagged in the table)
- **owner** — who can actually fix the underlying cause
- **action** — what the operator is supposed to do when they see this finding
- **resolution** — how the finding clears (most auto-clear when the underlying condition resolves)

### Aggregate distributions

| Dimension | Distribution |
|---|---|
| **Owner** | ~50% charm-author, ~20% operator, ~10% Juju/Canonical, ~20% mixed |
| **Severity** | ~20% info, ~50% warning, ~30% critical |
| **Resolution** | ~60% auto-clear on next observation, ~20% requires operator action, ~40% requires charm-author release (overlaps with auto-clear) |

### Hook firing protocol (Section 4c.1)

| Symptom | Severity | Owner | Operator action | Resolution |
|---|---|---|---|---|
| Hook crashes on re-fire (idempotency break) | critical | charm-author | `juju resolved` to retry; report to author | auto-clear when next re-fire succeeds with fixed charm |
| Service restart on every config-changed regardless of diff | info | charm-author | Report to author | charm-author fix |
| Hook hangs >5 minutes with no maintenance status update | warning | mixed (charm-author / operator scale) | Investigate via `juju debug-log`; consider scale | auto-clear when hook completes |
| Charm assumes pebble-ready runs once; race on pod churn | warning | charm-author | Report to author; consider pinning revision | charm-author fix |
| Forced upgrade skipped upgrade-charm; migrations skipped | critical | Juju (platform fix) + operator (mitigation) | Run migration manually via action or upgrade-charm | requires platform fix (LP#2068500) |

### Status protocol (Section 4c.2)

| Symptom | Severity | Owner | Operator action | Resolution |
|---|---|---|---|---|
| Unit remains 'unknown' indefinitely | warning | charm-author | Inspect debug-log; report to author | charm-author fix |
| `active` with non-empty message | info ⚖️ | charm-author | Report to author | charm-author fix |
| `blocked` with empty/unactionable message | warning | charm-author | Report to author with reproduction | charm-author fix |
| Status churn — rapid flips within seconds | warning ⚖️ | charm-author | Inspect to identify event-loop cause | charm-author fix |
| Non-leader writes application status | warning | charm-author | Report to author | charm-author fix |

### Lease / leadership protocol (Section 4c.3)

| Symptom | Severity | Owner | Operator action | Resolution |
|---|---|---|---|---|
| Leadership flap (>1 holder change per 30s) | warning ⚖️ | mixed (charm-author hook duration / infra) | Check hook duration on leader; check dqlite cluster health | auto-clear when underlying cause resolves |
| Non-leader writes to application databag (silently ignored) | warning | charm-author | Report to author | charm-author fix |
| Leader's leader-elected runs >2×lease-duration | critical | charm-author | Report to author; consider unit failover | charm-author fix |

### Relation protocol (Section 4c.4)

| Symptom | Severity | Owner | Operator action | Resolution |
|---|---|---|---|---|
| Charm doesn't handle relation-departed; stale entries | warning | charm-author | Report to author | charm-author fix |
| Excessive relation-changed handler cost (backlog) | warning ⇄ critical at scale | mixed (charm-author / operator scale) | Reduce scale OR report to author | charm-author fix or operator-scale |
| Cryptographic material in plaintext relation data | critical (security) | charm-author | Report; consider rotating leaked credentials | charm-author fix + credential rotation |

### Secret protocol (Section 4c.5)

| Symptom | Severity | Owner | Operator action | Resolution |
|---|---|---|---|---|
| Unbounded secret revision growth (owner ignores secret-remove) | warning | charm-author | Report to author | charm-author fix |
| Rotation declared but secret-rotate produces no new revision | warning ⇄ critical (if downstream auth fails) | charm-author | Report to author; consider manual rotation | charm-author fix |
| Consumer caches secret past secret-changed; stale data | warning | charm-author | Report; consider unit restart as workaround | charm-author fix |
| CMR consumer accesses secret after relation-broken | warning | charm-author | Report to author | charm-author fix |

### Storage protocol (Section 4c.6)

| Symptom | Severity | Owner | Operator action | Resolution |
|---|---|---|---|---|
| Charm accesses storage before attach completes | critical | charm-author | Report to author | charm-author fix |
| No flush on detach; data loss on reattach | critical (data loss) | charm-author | Report; pin revision until fixed | charm-author fix |
| Charm assumes persistence in non-durable pool | warning | mixed (operator pool / charm docs) | Move to durable pool OR document requirement | operator-action or charm-author fix |

### Action protocol (Section 4c.7)

| Symptom | Severity | Owner | Operator action | Resolution |
|---|---|---|---|---|
| Action hangs indefinitely | warning ⇄ critical (if blocking automation) | charm-author | Kill via task; report to author | charm-author fix |
| Action result missing expected keys | warning | charm-author | Report to author | charm-author fix |
| Action mutates shared state concurrently with hooks | warning ⇄ critical (if corruption) | charm-author | Report; avoid concurrent action invocation | charm-author fix |

### SDK-recommended layer (Section 4c.8)

| Symptom | Severity | Owner | Operator action | Resolution |
|---|---|---|---|---|
| Per-unit relation databag >1MB | warning | charm-author | Report to author | charm-author fix |
| update-status used for workload health | info | charm-author | Report to author | charm-author fix |
| Plaintext logs / missing topology labels | info | charm-author | Report to author | charm-author fix |
| No COS observability integrations | info | mixed (charm-author / operator) | Deploy COS OR consume the charm's lib OR report missing | depends |
| Sync action running >10 minutes | warning | charm-author | Report; switch to async pattern | charm-author fix |
| Hook fails on removed config key | warning | charm-author | Report; or restore old config key as workaround | charm-author fix |

### Operational-hygiene findings (Section 4b inventory items that didn't appear in 4c)

| Symptom | Severity | Owner | Operator action | Resolution |
|---|---|---|---|---|
| K8s RBAC triples accumulating per app | warning | Juju (#22384) | Report; await fix; manual kubectl cleanup as last resort | requires platform fix |
| Orphan PVCs after application removal | warning | Juju + operator | Manual kubectl cleanup | operator-action |
| Charm-revision age (months behind same-channel latest) | info | operator | `juju refresh` after change-review | operator-action |
| Agent binary version drift | warning | operator | `juju upgrade-model` in maintenance window | operator-action |
| Vault / secret-backend unreachable | critical (blocks secret ops) | operator (infra) | Restore backend connectivity | operator-action; auto-clear on next probe |
| Charmhub refresh-API unreachable | warning | operator (proxy/network) | Verify proxy config; restore connectivity | operator-action; auto-clear |
| Cloud-provider credential validity expiring | warning ⇄ critical at expiry | operator | Rotate credential | operator-action |
| Controller-cluster quorum / dqlite peer unreachable | critical | Juju + operator | Investigate; restore peer; restart controller | requires investigation |
| Idle FD count high on controller | warning | Juju (#21081 ongoing) | Monitor; restart if necessary | requires platform fix |
| Provider-tracker in restart-loop | warning ⇄ critical (if blocks model destroy) | Juju | Investigate via depengine introspection; report | requires platform fix |

⚖️ marks judgement-call severity assignments — see Section 6 question #2 for the deeper question.

### Three design implications

1. **Every finding needs a structured "owner" field.** Severity alone isn't enough. An operator seeing 47 warnings, of which 40 are charm-author fixes, needs the surface to make that immediately legible — otherwise they stop reading. Remediation field shape is at minimum `{ severity, owner, action_text }`; preferably `{ severity, owner, action_kind, action_target, action_text }` where `action_kind` ∈ `{ self-fix, report-to-author, await-platform, switch-revision, suppress, configure-tolerance, investigate }`.

2. **The observatory has natural symmetry as a charm-author quality feedback loop.** If 50% of findings are charm-author-fix, the same data has value to the charm author — anonymised, aggregated, scoped to their charm. Section 6 question #8 (charm-author feedback loop) goes from "nice to have" to "load-bearing for the strategic story." This affects privacy posture (what leaves the operator's controller?), data plane (Charmhub-side aggregator?), and competitive positioning ("Juju's charms get better in the field").

3. **For charm-author-fix findings, the operator's action surface is finite and prescribable.** When the operator sees a finding they can't directly fix, what they CAN do is bounded — `report-to-author` (with structured data, much better than a Discourse thread), `switch-revision`, `suppress` (acknowledge as known issue), `configure-tolerance` ("I know this charm is slow; raise the threshold"). The observatory should encode these as first-class actions. Without them, charm-author-fix findings become noise.

### Open severity questions

These are calibration questions for Stage C spec (Section 6 question #2):
- Is "ActiveStatus with message" info or warning? (Functional impact: zero. Convention violation: clear.)
- Is "leadership flap" warning or critical, and at what threshold (1 flip/30s? 3 flips/5min?)
- Is "COS observability not wired up" a finding at all? (Operator chose; it's their deployment.)
- Is "hook duration >5 minutes" still a folklore boundary, or do we promote it to a formal threshold?

These are not blocking pre-design; they're decisions for the spec.

---

## 5. Affordances to adopt from adjacent ecosystems

Observed as load-bearing design choices across 25+ surveyed tools (K8s,
IaC, cloud-vendor, observability, PaaS). Each one should be a deliberate
yes/no for the Operator Advisor.

1. **Severity laddering** with explicit finite categories (e.g.,
   info/warning/critical), not freeform strings.
2. **Findings are first-class queryable objects**, not just notifications
   (operators can `get` / `watch` / RBAC them).
3. **Orthogonal axes** for findings — e.g., "is conformance to declared
   intent OK" vs "is observed behaviour OK." Avoids overloading a single
   status.
4. **Auto-clearing lifecycle.** Findings open, update, and close
   automatically; operators trust the dashboard.
5. **Suggested remediation attached.** Every finding carries a copy-pasteable
   next action (juju command, doc link, structured hint).
6. **Fleet rollup that decomposes** to a single object. Both the heatmap and
   the per-thing view.
7. **Personalisation.** Show me only what affects my deployment.
8. **Dependency-graph explainer.** Show *which* neighbour or dependency is
   the proximate cause.

---

## 6. Open product questions (require human input)

The codebase cannot answer these. They are the gates for Stage B.

1. **Audience priority.** Cloud admin running models day-to-day? SRE
   running JAAS for many tenants? Charm authors as a feedback loop on field
   behaviour? Each yields a different shape.
2. **Severity model.** Adopt three levels (info/warning/critical), align
   with the existing `core/status.Status` enum, or design a richer
   taxonomy (per-dimension severity: liveness, performance, drift,
   conformance)?
3. **Threshold ownership.** Who decides "queue depth >50 is degraded"?
   Per-check defaults baked in code, controller config, model config,
   per-application overrides, or learned baselines per charm?
4. **Privacy / scope.** Is per-charm compliance data visible only to the
   model owner, or also reportable upstream (Canonical, charm author) to
   improve charms in the field? If the latter, where do consent and
   anonymisation live?
5. **Personalisation.** Per-model only (like `juju status`), or
   controller-wide rollup (like the controller facade)? Cross-controller
   via JIMM?
6. **Strategic time horizon.** Is this a Juju 4.1 feature, a Juju 5
   foundation, or a JAAS-only differentiator that doesn't ship in
   open-source Juju?
7. **Relationship to COS.** Co-exist (Juju does paradigm, COS does
   workload), or absorb (some COS signals like hook-latency move into
   Juju)? Implications for charm libraries.
8. **Charm-author feedback loop.** If we observe that the postgresql charm
   has slow hooks at scale, can we feed that back to the charm author
   without operator-confidential data leakage?

---

## 7. Plan — phases beyond this brief

### Phase A — This session (DONE)
Stage A opportunity research. Pivot from invariant-checking to compliance
observatory. Direction chosen.

### Phase B.1 — Deep signal inventory (DONE)
Read-only investigation across 33 candidate behavioural signals. Results
captured in Section 4b. Verdict distribution: 7 ALREADY-VISIBLE, 17
TRACKED-BUT-HIDDEN, 9 NOT-TRACKED. Headline takeaway: v1 scope is
bounded by the 24 already-tracked signals, which can be surfaced without
new schema or new agent instrumentation.

### Phase B.2 — Operator validation (NEXT)
2-4 interviews with operators running real Juju estates. Show them the
24-signal "reachable in v1" list and the 9 "future cohort" list. Ask:
- "If you could see only three of these on day one, which three?"
- "Which of these would have prevented an incident in the last 6 months?"
- "What's missing from the list?"
- "Per-signal severity threshold — should it be vendor-default, your
  default, or per-application override?"

Output: prioritised v1 signal list (3-5 signals), anchored in real
operator pain. Resolves Question 1 (audience priority) and Question 3
(threshold ownership) from Section 6.

### Phase C — Stage C spec (after Phase B)
Written spec for v1. Picks 3-5 signals from B.2. Defines the finding
record shape, the severity model, the auto-clearing lifecycle, the
operator surface (CLI + facade + integration with `juju status`), and the
charm-author surface (if any). Resolves the open questions in §6.

### Phase D — Design (after Phase C)
Architecture spec. Implementation plan. Worker placement, facade
registration, schema additions, back-compat strategy. This is where the
Stage A investigation findings in this session become directly load-bearing.

### Phase E — Build (after Phase D)
Implementation. Ship v1 to Juju 4.x. Iterate on operator feedback.

---

## 8. What we DELIBERATELY did not pick

For traceability:

- **Operational Drift Catalog (Shape B).** Strong second-wave candidate.
  Folded in partially as Section 4.6 (Resource hygiene) and parts of 4.5
  (Secret compliance), but not the primary frame. Reason: lower
  differentiation, commodity surface area.
- **Anti-Pattern Detector (Shape C).** Deferred to a separate effort,
  ideally as a deploy-time / refresh-time linter (`juju deploy --dry-run`
  / `juju lint`). Reason: operates at deploy-time, user explicitly
  de-prioritised deploy-time.
- **Original invariant framing.** Stage A research confirmed feasibility
  for 2 of 8 candidate invariants but the framing as a whole solved a
  problem (Juju internal correctness) that does not match the value the
  user wants to deliver. Some specific invariants (e.g., hook firing
  completeness across CMR) re-emerged here as advisor signals
  (Section 4.4).

---

## 9. References (from this session's research)

All paths relative to `/home/sinan.awad@canonical.com/dev/juju` at branch 4.0.

### Existing operator-health surfaces (Agent B findings)
- `cmd/juju/status/formatted.go:18-27` — top-level `formattedStatus` struct.
- `rpc/params/status.go:24-35` — wire `FullStatus` type with omitempty fields.
- `core/status/status.go:47-233` — the unified status enum.
- `docs/reference/juju-dashboard.md` — JAAS-hosted Juju Dashboard
  ("at-a-glance health checks" via aggregating per-entity status).

### Internal behavioural signals already tracked (Stage A findings)
- `unit_agent_presence.last_seen` — `domain/schema/model/sql/0020-unit.sql:219-225`
- `unit_agent_status.updated_at` — `0020-unit.sql:177-189`
- `unit_workload_status.updated_at` — `0020-unit.sql:191-203`
- `lease.start / expiry` — `domain/schema/controller/sql/0002-lease.sql:13-24`
- `removal.scheduled_for` — `domain/schema/model/sql/0028-cleanup.sql:33`
- Engine restart counters — `agent/engine/metrics.go:42-58`
- Changestream throughput metrics — `internal/worker/changestream/metrics.go:136-167`
- API request duration histograms — `apiserver/apiservermetrics.go:106-198`
- Uniter event queue and operation history — in-memory in
  `internal/worker/uniter/operation/*` and `internal/worker/uniter/remotestate/*`
  (not persisted; not exposed).

### Adjacent-ecosystem affordances (Agent C findings)
- ArgoCD sync × health decomposition.
- Kyverno `PolicyReport` / Trivy-operator `*Report` CRD pattern
  (findings as first-class queryable objects).
- AWS Trusted Advisor severity-laddering across six fixed categories.
- Cloudflare Health Checks multi-region quorum (anti-flap design).
- Datadog Watchdog deployment-correlated anomaly stories.
- Headlamp / Octant resource-map dependency-graph rendering.

### Field-issue catalog (Agent A findings)
36 distinct examples across ACUTE (12), CHRONIC (8), ANTI-PATTERN (5),
EXTERNAL (11). Highest-recurrence chronic-drift cohort: secret-backend
desync, charm-revision aging, leaked K8s RBAC, orphan PVCs, idle FD
accumulation. See agent output for citations.

---

## Appendix A — Hackathon prototype plan (2026-05-13)

A one-day prototype of the Juju Operator Advisor as a native `juju
advisor` command. Demonstrates the core value proposition against a staged
microk8s+juju deployment with three detectable signals. AI enrichment is
architecturally accommodated but stubbed for the demo (the user does not
have Claude API access yet; AI integration is deferred to a separate
discussion).

### A.1 Scope

**In:**
- Native `juju advisor` CLI command in `cmd/juju/advisor/`
- Client-side detection (calls existing facades; no new facade work)
- Three signal detectors with structured `Finding` output
- Terminal-rendered output (hybrid format: compact header + arrow notes)
- `-o yaml` / `-o json` for structured output
- Microk8s+juju 4.0 staging environment

**Out (intentionally, for v1 later):**
- Controller-side facade
- Persistence of findings to dqlite
- Auto-clear lifecycle
- Watcher / streaming
- Suppression / acknowledgement
- Integration with `juju status` output
- AI integration (deferred to its own discussion)

### A.2 The three signals

| # | Signal | Owner | Severity | Data source | Predicate |
|---|---|---|---|---|---|
| 1 | ActiveStatus carrying message | charm-author | info | `client.FullStatus` | `unit.workload-status.current == "active" && message != ""` |
| 2 | Charm-revision aging | operator | warning | `client.FullStatus` | `app.can-upgrade-to != ""` |
| 3 | Unit blocked for >24h | mixed | warning (critical >7d) | `client.FullStatus` | `unit.workload-status.current == "blocked" && now - since > 24h` |

All three detectable from a single `Client.FullStatus` RPC call — one round-trip to the controller.

### A.3 CLI shape

```
$ juju advisor                 # default: hybrid output, all findings
$ juju advisor --details       # placeholder — hybrid already shows them
$ juju advisor -o yaml         # structured output (Finding records)
$ juju advisor -o json         # JSON for piping
$ juju advisor -m <model>      # scope to another model
```

Default output (from the AskUserQuestion preview, locked in):

```
Compliance findings for model my-app

postgresql/0   CRITICAL  secret-rotation-stuck
  → 'postgres-creds' has no new revision in 14 days (rotate-policy: weekly)
  → Charm author: implement _on_secret_rotate handler

postgresql     WARNING   charm-revision-aging
  → 47 revisions behind latest/stable
  → Operator: juju refresh postgresql --channel=latest/stable

hello-world/0  INFO      active-status-with-message
  → ActiveStatus carrying message "Started" (convention violation)
  → Charm author: clear status message on transition to active

3 findings.
```

### A.4 Code structure

```
cmd/juju/advisor/
  advisor.go          # cmd.Command implementation; CLI entry
  advisor_test.go     # unit tests using cmd.Tester
  format.go           # output rendering (hybrid + yaml + json)
  detector/
    detector.go       # Finding struct, Detector interface
    activestatus.go   # detector 1
    revision.go       # detector 2
    blocked.go        # detector 3
```

Approximate code size: ~600 lines including tests.

### A.5 Wiring into juju

- Register the command in the top-level command registry alongside other
  `cmd/juju/*` commands. Pattern reference: `cmd/juju/block/list.go`
  (124 lines, similar shape — embed `modelcmd.ModelCommandBase` to
  inherit `-m`/`--model` handling).
- Call the Client facade for `FullStatus` via `api/client/client.go`.
  No new facade, no `allfacades.go` change.

### A.6 Finding record (designed for AI enrichment later)

```go
type Finding struct {
    CheckID        string
    Severity       Severity      // info | warning | critical
    Entity         string        // e.g., "postgresql/0"
    EntityKind     string        // "unit" | "application"
    Owner          string        // "charm-author" | "operator" | "mixed"
    RawData        any           // signal-specific values
    Summary        string        // one-line scan-friendly
    Recommendation string        // multiline guidance (arrow lines)
    ProtocolRef    string        // citation in Section 4c (e.g., "4c.2")
}
```

For the prototype the `Recommendation` field is hand-written per signal.
The AI integration (deferred) is a function `Enrich(Finding) Finding`
that can replace or augment `Recommendation`. The CLI doesn't know
whether AI ran — same record type either way.

### A.7 Staging on microk8s

1. Bootstrap: `juju bootstrap microk8s advisor-demo` (~3 min)
2. `juju add-model demo`
3. Stage the three signals:
   - **Signal 1:** deploy any charm that emits an ActiveStatus message —
     many production charms do. Verify in `juju status` before demo.
   - **Signal 2:** deploy a deliberately old revision
     (`juju deploy <charm> --revision=<old>`); `juju status` will show
     `can-upgrade-to`.
   - **Signal 3:** the cleanest staging is to mock the since-timestamp
     in the prototype's detector logic so a blocked unit reports as
     >24h regardless. Direct dqlite manipulation of `since` is not
     recommended — too brittle. This is honest enough for the demo;
     real implementation would just read the actual `since`.

### A.8 Day plan (≈ 8h)

**Morning (4h)**
- 30 min: clone juju, build (`make juju` for the client only — faster than full `make install`)
- 1h: scaffold `cmd/juju/advisor/advisor.go` following `cmd/juju/block/list.go`
- 1h: implement `Detector` interface + 3 detectors against `params.FullStatus`
- 1.5h: output formatter (hybrid + yaml + json)

**Lunch + (2h)**
- 1h: end-to-end test against staged microk8s
- 1h: AI integration stub (a function that returns the hand-written
  Recommendation today; structured to swap in an LLM call tomorrow)

**Afternoon (2h)**
- 1h: dry-run the demo arc twice
- 1h: 3-slide deck or 90-second narrative

### A.9 Demo arc (90 seconds)

1. `juju status` shows everything green. "Looks healthy."
2. `juju advisor` shows three findings, severity-coloured.
3. Read out: "ActiveStatus with message — charm-author convention
   violation. Charm-revision aging — operator can refresh. Unit blocked
   for 38 hours — needs investigation."
4. `juju advisor -o yaml` — point at the structured fields (Owner,
   Recommendation, ProtocolRef).
5. Strategic frame: "This demo shows 3 of the 33 signals catalogued in
   the brief; the same code path scales. The Recommendation field today
   is hand-written; the design is ready for AI enrichment as soon as
   the integration story is settled."

### A.10 What's NOT in the prototype (be honest in the demo)

- No persistence — findings recomputed every command invocation.
- No auto-clear — same.
- No watcher — `juju advisor --watch` would be ~30 minutes more code, skip for the day.
- No real AI — the Recommendation is hand-written but designed to be
  replaced by AI output.
- Detection in the client, not the controller — fine for v1-design
  validation; the v1 implementation should move it controller-side
  behind a Health facade (see Section 4d, Q3 in earlier brief).

### A.11 If time remains

In priority order:
1. **Coloured severity** in the hybrid output (red/yellow/blue ANSI). 10
   minutes.
2. **`--severity` filter** (`juju advisor --severity=warning,critical`). 15
   minutes.
3. **Sort by severity** in default output (most severe first). 10 minutes.
4. **A fourth signal** if staging time was generous — `agent-lost` is
   trivial to detect from `juju status`.

---

## Appendix B — Pre-canned AI-enriched fixtures

The prototype's "AI" layer is a function `Enrich(Finding) Finding`. For
the demo, that function reads from the JSON fixture below — generated in
this Claude Code session against the advisor protocol in Section 4c.
The demo can honestly say: "These recommendations were produced by Claude
using the advisor protocol spec. Live integration is the next step;
the data shape is identical." Live LLM integration is a one-function
swap when AI access is settled.

The terse hand-written `Recommendation` is shipped alongside, so the
demo can show "before AI" vs "after AI" side by side — the strongest
exhibit for the value proposition.

### B.1 Fixture file (`cmd/juju/advisor/testdata/findings.json`)

```json
[
  {
    "check_id": "active-status-with-message",
    "severity": "info",
    "entity": "hello-world/0",
    "entity_kind": "unit",
    "owner": "charm-author",
    "raw_data": {
      "workload_status_current": "active",
      "workload_status_message": "Started",
      "since": "2026-05-12T09:14:00Z"
    },
    "summary": "ActiveStatus carrying message 'Started'",
    "recommendation_terse": "ActiveStatus carrying message 'Started' — convention violation. Charm author: clear status message on transition to active.",
    "recommendation_enriched": "The hello-world/0 unit reports `active` workload status while carrying the message 'Started'. By the Juju Operator framework convention (docs/reference/status.md), `ActiveStatus` should have no message — the empty message is itself the visual signal of normal operation, and any text in `active` reads to operators as a stale 'maintenance' or 'waiting' indicator that doesn't reflect actual unit health.\n\nCharm-author fix: in the relevant event handler (likely `_on_install`, `_on_start`, or a relation handler), replace `self.unit.status = ActiveStatus('Started')` with `self.unit.status = ActiveStatus()` once initialisation completes. If the 'Started' string was intended as a state-transition marker, prefer a structured log entry or an action result rather than encoding it in workload status.",
    "protocol_ref": "Section 4c.2 — Status protocol"
  },
  {
    "check_id": "charm-revision-aging",
    "severity": "warning",
    "entity": "postgresql",
    "entity_kind": "application",
    "owner": "operator",
    "raw_data": {
      "current_revision": 138,
      "available_revision": 185,
      "channel": "latest/stable",
      "last_refresh": "2025-04-02T11:30:00Z"
    },
    "summary": "47 revisions behind latest/stable",
    "recommendation_terse": "47 revisions behind latest/stable. Operator: juju refresh postgresql --channel=latest/stable.",
    "recommendation_enriched": "The `postgresql` application is running revision 138 from the `latest/stable` channel, while revision 185 (47 revisions newer) is available in the same channel. The application has not been refreshed since 2025-04-02 — over a year ago. While newer revisions within the same risk track are intended to be backward-compatible, a gap this large is likely to contain meaningful security and feature improvements.\n\nOperator action: (1) review the charm's release notes on Charmhub or its Discourse topic for revision 185 to identify breaking changes or deprecated config keys. (2) Stage the refresh in a non-production model first with `juju refresh postgresql --channel=latest/stable`. (3) Verify the `upgrade-charm` and `config-changed` hooks complete cleanly via `juju debug-log --include postgresql --replay --tail 200`. (4) Repeat in production during a maintenance window.",
    "protocol_ref": "Section 4b inventory item 4.6.3 — charm-revision age"
  },
  {
    "check_id": "unit-blocked-prolonged",
    "severity": "warning",
    "entity": "worker/0",
    "entity_kind": "unit",
    "owner": "mixed",
    "raw_data": {
      "workload_status_current": "blocked",
      "workload_status_message": "waiting for database relation",
      "since": "2026-05-11T03:14:00Z",
      "duration_hours": 38
    },
    "summary": "Blocked for 38 hours: 'waiting for database relation'",
    "recommendation_terse": "Blocked for 38 hours. Operator: inspect 'juju status' and 'juju debug-log'; consider reporting to charm author if recurring.",
    "recommendation_enriched": "The unit `worker/0` has been in `blocked` workload status for 38 hours (since 2026-05-11T03:14Z), with the message 'waiting for database relation'. The Juju advisor protocol (Section 4c.2) treats `blocked` as 'human action required' — but a unit blocked for >24h indicates either an operator who hasn't acted on the request, or a charm-author bug where `blocked` was set for a condition the charm should resolve by reaching `waiting` instead.\n\nInvestigation path:\n  1. `juju show-unit worker/0` — is the database relation present? What keys are in the data bag?\n  2. `juju status` — is the database application healthy and providing the expected interface?\n  3. `juju debug-log --include worker/0 --replay --tail 100` — review recent events on this unit.\n\nIf the database relation is missing entirely, the operator forgot to integrate; run `juju integrate worker postgresql`. If the relation is present but data is missing, the database side is the proximate cause. If neither applies, the recurring `blocked` should be reported to the charm author with the captured relation state — the charm is likely setting `blocked` for a condition that warrants `waiting`.",
    "protocol_ref": "Section 4c.2 — Status protocol; Section 4e operator action loop"
  }
]
```

### B.2 Rendering with the hybrid format

With `recommendation_terse` (no AI):

```
hello-world/0  INFO      active-status-with-message
  → ActiveStatus carrying message 'Started'
  → Charm author: clear status message on transition to active

postgresql     WARNING   charm-revision-aging
  → 47 revisions behind latest/stable
  → Operator: juju refresh postgresql --channel=latest/stable

worker/0       WARNING   unit-blocked-prolonged
  → Blocked for 38 hours: 'waiting for database relation'
  → Operator: inspect 'juju status' and 'juju debug-log'

3 findings.
```

With `recommendation_enriched` (AI), `juju advisor --details` or `-o yaml`
expands the multi-paragraph recommendations. The default hybrid output
shows only the first two lines of the enriched recommendation as the
arrow notes, with a "→ ... (full guidance: juju advisor --details
worker/0)" hint when the recommendation has more content.

### B.3 Code shape for the Enrich function

```go
// Enrich applies AI-generated guidance to a Finding. The prototype
// reads from testdata/findings.json keyed by check_id+entity; the
// real implementation would call an LLM with the Finding's raw_data
// and the protocol_ref context.
func Enrich(f Finding) Finding {
    if fixture, ok := loadFixture(f.CheckID, f.Entity); ok {
        f.Recommendation = fixture.RecommendationEnriched
    }
    return f
}
```

A `--no-ai` flag can use `recommendation_terse` instead, supporting the
side-by-side comparison demo.

### B.4 Demo value of this exhibit

The side-by-side comparison is the strongest part of the demo:

- Without AI: an operator sees a one-line warning and is left to fill
  in the context, action, and reasoning themselves.
- With AI: the same finding becomes a triage paragraph — naming the
  contract clause being violated, explaining the underlying convention,
  and giving operator/charm-author-specific action paths.

For the 90-second pitch, the closer is: *"The detection layer is one
day's work; the AI enrichment is what makes the output operator-grade.
The advisor protocol in Section 4c is what gives the AI enough
context to be useful."*

---

## Appendix C — Spec-Kit input artifact

GitHub Spec Kit (`github.com/github/spec-kit`) is the implementation
workflow chosen for tomorrow. Its commands are: `specify init` to
bootstrap, `/speckit.constitution` for principles, `/speckit.specify`
for narrative requirements, `/speckit.plan` for technical constraints,
`/speckit.tasks` to break down, `/speckit.implement` to execute.

This appendix is the three inputs ready to paste:
- **C.1** — `.specify/memory/constitution.md` (principles)
- **C.2** — the `/speckit.specify` prompt (what/why; narrative)
- **C.3** — the `/speckit.plan` prompt (technical stack + juju conventions)

Tomorrow's workflow:
```bash
specify init juju-advisor
cd juju-advisor
# paste C.1 into .specify/memory/constitution.md
/speckit.constitution                # confirms principles
/speckit.specify "<paste C.2>"
/speckit.plan "<paste C.3>"
/speckit.tasks
/speckit.implement
```

### C.1 Constitution — paste into `.specify/memory/constitution.md`

```markdown
# Juju Operator Advisor — Project Constitution

## Mission

The Juju Operator Advisor surfaces violations of the implicit
charm-Juju operational contract — degradations caused by external
factors (charms, infrastructure) that are invisible to today's
`juju status` surface. The platform itself is assumed correct; the
observatory measures everything else.

## Foundational principles

1. **Findings are first-class queryable data**, not freeform strings.
   Every finding has: severity (info|warning|critical), owner
   (charm-author|operator|mixed|platform), entity, summary, structured
   recommendation, and a citation to the contract clause it violates.

2. **Severity is calibrated for operational impact**, not technical
   detail. Info = convention violation with no functional impact.
   Warning = degrading state requiring action within a sprint.
   Critical = data integrity, security, or hard breakage.

3. **Owner classification is load-bearing.** ~50% of compliance
   violations are charm-author-fix; the rest are operator-fix or
   platform-fix. Without surfacing this, operators see noise.

4. **Runtime observation is uniquely necessary.** ~70% of compliance
   symptoms cannot be derived from charm code (they are temporal,
   scale-dependent, environment-dependent, or peer-dependent).
   Static analysis is a complementary product, not a substitute.

5. **The advisor protocol is articulated separately** and lives
   in the codebase as a referenceable document. Findings cite the
   clause they violate. Operators and charm authors are never flagged
   for a rule they cannot read.

6. **AI enrichment is a layered, optional transformer** on the
   Finding record. The detection layer produces structured Findings;
   the enricher (optionally) rewrites the recommendation with richer
   prose. The CLI does not care whether enrichment ran.

7. **Auto-clearing lifecycle** is the universal resolution mechanism
   when full v1 ships. Findings open, update, and close automatically
   based on next observation. Operators trust the dashboard.

8. **Follow juju conventions ruthlessly.** New facades, commands,
   schema follow existing patterns (e.g., `cmd/juju/block/list.go`
   for command shape). No novel abstractions in v1.

9. **Detection layer placement is an implementation detail**, not a
   product commitment. Client-side for the prototype; controller-side
   behind a `Health` facade for v1. The data contract is identical.

10. **Backwards compatibility is preserved.** Adding the observatory
    must not break existing CLI or facade contracts. New fields on
    `params.FullStatus` are `omitempty`.

## Non-goals (explicit)

- The observatory does NOT measure workload health (that's the charm's
  job, surfaced via COS or pebble probes).
- The observatory does NOT replace `juju status` (it complements it).
- The observatory does NOT lint charm source (that's a separate Juju
  Advisor Lint product).
- The observatory does NOT modify charm code or relations on the
  operator's behalf (read-only).

## Reference documents

The companion brief at `advisor-brief.md` is the
authoritative product context:
- Sections 1-3: opportunity framing
- Section 4b: 33-signal inventory with verdicts
- Section 4c: 8-protocol advisor protocol
- Section 4e: ~50 violation symptoms with severity/owner/action mapping
- Appendix A: hackathon prototype scope
- Appendix B: pre-canned AI fixtures
```

### C.2 `/speckit.specify` prompt — paste as the command argument

```
We are building a new juju client-side CLI command called `juju advisor`
that surfaces deployment-level compliance degradations to operators.
The compliance observatory is an internal Canonical concept: it
measures how well each deployed charm behaves as a Juju advisor —
i.e., conforms to the operational contract Juju expects (hooks, status,
leases, relations, secrets). It does NOT measure workload health (the
charm's job) and does NOT replace `juju status`; it complements both.

For this iteration we build only the operator-facing client command.
Detection runs in the client (calls existing Juju facades, no new
facade). Three signals are detected; future iterations will add more.

WHAT THE OPERATOR EXPERIENCES

The operator types `juju advisor` against any model they have read
access to. The command queries the controller via existing facades,
runs three detector predicates locally, and prints a list of findings.
Each finding shows severity, the affected entity (unit or application),
the check identifier, and 1-3 arrow-prefixed lines summarising the
issue and the recommended action. Output is sorted by severity
(critical first).

By default findings render in a hybrid format (compact header + arrow
notes). With `-o yaml` or `-o json`, findings are emitted as
structured records suitable for piping into downstream tooling.

The user invokes `juju advisor -m <model>` to inspect a model other
than their current one. `juju advisor --severity=warning,critical`
filters output. `juju advisor --no-ai` disables AI-enriched
recommendations and falls back to the terse hand-written text.

THE THREE SIGNALS THIS COMMAND DETECTS

Signal 1 — ActiveStatus carrying message.
For each unit whose workload status is "active" AND whose workload
status message is non-empty, emit an info-severity finding. The
Juju Operator framework convention is that `active` status carries no
message; the empty message is the visual signal of normal operation.
Owner: charm-author.

Signal 2 — Charm-revision aging.
For each application whose `CanUpgradeTo` field is non-empty, emit
a warning-severity finding. The application is behind the current
revision in its tracked channel. Owner: operator.

Signal 3 — Unit blocked for >24 hours.
For each unit whose workload status is "blocked" AND whose `since`
timestamp is more than 24 hours in the past, emit a warning-severity
finding (critical if more than 7 days). The Juju advisor protocol
treats `blocked` as "human action required"; prolonged blocked state
indicates either operator inaction or a charm-author bug. Owner: mixed.

THE FINDING DATA SHAPE

Each finding is a structured record with: check_id (string), severity
(enum: info|warning|critical), entity (string, e.g., "postgresql/0"),
entity_kind (enum: unit|application), owner (enum: charm-author|operator|
mixed|platform), summary (one-line string), recommendation (multi-line
string), protocol_ref (citation to the advisor protocol).

The recommendation is hand-written by default. When AI enrichment is
enabled (the default), the recommendation is replaced by a richer
LLM-generated explanation loaded from a JSON fixture file at
`cmd/juju/advisor/testdata/findings.json`. The fixture file is
populated ahead of demo with three entries — one per signal. The
production design (out of scope for this iteration) replaces the
fixture lookup with a live LLM call.

ACCEPTANCE CRITERIA

1. `juju advisor` against a model with no degradations prints
   "No findings." and exits 0.
2. `juju advisor` against a staged model with all three signals
   prints exactly three findings in the hybrid format, sorted by
   severity.
3. `juju advisor -o yaml` emits a YAML list of finding records with
   all eight fields populated for each finding.
4. `juju advisor -o json` does the same in JSON.
5. `juju advisor --no-ai` substitutes the terse recommendation for
   each finding; the rest of the output is identical.
6. `juju advisor -m <other-model>` operates on the named model.
7. `juju advisor --severity=critical` filters out info and warning
   findings.
8. Unit tests cover each detector predicate in isolation against a
   synthetic `params.FullStatus`.

OUT OF SCOPE FOR THIS ITERATION

- Persistence of findings.
- Auto-clearing lifecycle.
- Watcher (`--watch`) mode.
- A controller-side facade (detection is client-side only).
- Live LLM integration (fixtures stand in).
- Integration with `juju status` output.
- Suppression / acknowledgement of findings.
- The other 30+ signals catalogued in the companion brief.
```

### C.3 `/speckit.plan` prompt — paste as the command argument

```
TECHNICAL STACK

- Implementation language: Go (juju is Go 1.26+; see go.mod).
- The command lives in /home/sinan.awad@canonical.com/dev/juju/cmd/juju/advisor/.
- We are working on branch 4.0 of the juju repository.

CODE STRUCTURE

cmd/juju/advisor/
  advisor.go              # cmd.Command implementation
  advisor_test.go         # cmd.Tester-driven unit tests
  format.go               # output rendering (hybrid + yaml + json)
  detector/
    detector.go           # Finding struct, Detector interface
    activestatus.go       # detector 1
    revision.go           # detector 2
    blocked.go            # detector 3
    detector_test.go      # detector unit tests against synthetic FullStatus
  enrich/
    enrich.go             # Enrich(Finding) Finding; reads fixtures
  testdata/
    findings.json         # AI-enriched fixtures (committed)

JUJU CONVENTIONS TO FOLLOW

- The command embeds `modelcmd.ModelCommandBase` for -m/--model
  handling. Reference: `cmd/juju/block/list.go` (124 lines, similar
  shape — read it as the template).
- The command implements `cmd.Command` interface (Info, SetFlags, Init,
  Run).
- Output formatters are registered via `cmd.NewSuperCommand` /
  `cmd.WriteSet` pattern. Reference: `cmd/juju/status/formatter.go`.
- Unit tests use `cmd.Tester` from github.com/juju/cmd/v4/cmdtesting.
  Reference: `cmd/juju/block/list_test.go`.
- Use the `tc` test framework (not standard `testing`). Reference:
  AGENTS.md "Unit Test Conventions" section.
- Imports follow gci three-stanza ordering: stdlib, external,
  github.com/juju/juju. Run `gci write --section standard --section
  default --section "Prefix(github.com/juju/juju)" <file>` after every
  edit. Reference: AGENTS.md "Code Formatting" section.

API CALLS

- Use the existing Client facade for `FullStatus`. Reference:
  api/client/client.go.
- No new facade is created. No allfacades.go change. No schema
  migration. No domain service.

REGISTRATION

- Register `juju advisor` in the top-level command registry
  alongside other `cmd/juju/*` commands. Reference: the registration
  pattern in `cmd/juju/commands/main.go`.

DATA SHAPES

```go
type Severity string
const (
    SeverityInfo     Severity = "info"
    SeverityWarning  Severity = "warning"
    SeverityCritical Severity = "critical"
)

type Owner string
const (
    OwnerCharmAuthor Owner = "charm-author"
    OwnerOperator    Owner = "operator"
    OwnerMixed       Owner = "mixed"
    OwnerPlatform    Owner = "platform"
)

type Finding struct {
    CheckID        string    `json:"check_id" yaml:"check-id"`
    Severity       Severity  `json:"severity" yaml:"severity"`
    Entity         string    `json:"entity" yaml:"entity"`
    EntityKind     string    `json:"entity_kind" yaml:"entity-kind"`
    Owner          Owner     `json:"owner" yaml:"owner"`
    Summary        string    `json:"summary" yaml:"summary"`
    Recommendation string    `json:"recommendation" yaml:"recommendation"`
    ProtocolRef    string    `json:"protocol_ref" yaml:"protocol-ref"`
}

type Detector interface {
    ID() string
    Detect(status *params.FullStatus) []Finding
}
```

ENRICH FUNCTION

```go
// Enrich substitutes Finding.Recommendation with AI-generated text
// loaded from testdata/findings.json. The fixture is keyed by
// (CheckID, Entity). When `--no-ai` is passed, Enrich is a no-op.
func Enrich(f Finding) Finding
```

OUTPUT FORMATTING

- Default (hybrid): one severity-colored line per finding ("entity
  severity check_id\n  → summary\n  → recommendation first line"),
  followed by a trailing "N findings (X critical, Y warning, Z info)."
- `-o yaml`: yaml.Marshal of `[]Finding`.
- `-o json`: json.Marshal of `[]Finding`.
- ANSI color: red for critical, yellow for warning, blue for info.
  Suppress colors when stdout is not a TTY (use the existing helper
  pattern in cmd/juju/status/).

ERROR HANDLING

- Unreachable controller: surface the underlying RPC error wrapped
  with "fetching FullStatus".
- Empty findings: print "No findings." and exit 0.
- Malformed fixture file: log a warning, fall back to terse
  recommendations, continue.

TESTING

- Unit tests for each detector against synthetic `params.FullStatus`
  inputs covering: zero matches, one match, many matches, edge cases
  (e.g., `since` is exactly 24h, status is `unset`).
- Integration test (skip-on-short-mode) that mocks the API client
  and verifies end-to-end command output.

OUT OF SCOPE FOR THIS PLAN (do NOT generate tasks for these)

- Schema migrations.
- New facades.
- Controller-side workers.
- Live LLM integration.
- Persistence layer.
- Watcher streaming.
- Insertion into `juju status` output.
- Other detectors beyond the three named.
```

### C.4 Priming Spec Kit tomorrow — step-by-step

Spec Kit is initialised **in place** inside the juju repo so the
compile-iterate loop is immediate: generated code lands directly under
`cmd/juju/advisor/`, you `make juju` and run, no porting step. The
`.specify/` directory stays local-only via `.git/info/exclude` (no
upstream commit, no `.gitignore` change).

**Steps:**

```bash
# 1. Start on a feature branch from the 4.0 branch
cd ~/dev/juju
git checkout 4.0
git pull
git checkout -b feature/juju-advisor-prototype

# 2. Init spec-kit in place. juju is a non-empty repo, hence --force.
specify init . --here --force

# 3. Keep .specify/ out of the working tree locally — no upstream pollution.
echo ".specify/" >> .git/info/exclude

# 4. Drop the constitution (Appendix C.1) into memory.
# Open .specify/memory/constitution.md in your editor and paste the
# entire C.1 block verbatim. Replace any default content created by init.

# 5. Give the LLM the full brief as persistent context across phases.
cp advisor-brief.md \
   .specify/memory/compliance-context.md

# 6. Validate the constitution in your AI agent.
/speckit.constitution

# 7. Specify phase — paste C.2 as the argument.
/speckit.specify "<paste C.2>"
# Review .specify/specs/001-*/spec.md before continuing. If it
# contradicts the constitution non-goals (persistence, new facades,
# integration with juju status), redirect before /speckit.plan.

# 8. Plan phase — paste C.3.
/speckit.plan "<paste C.3>"
# Review plan.md. The plan should land code in cmd/juju/advisor/.
# If it proposes any other path, redirect.

# 9. Tasks (auto-generated from plan).
/speckit.tasks
# Expect 15-30 small tasks. If 100+, simplify spec/plan first.

# 10. Implement.
/speckit.implement
```

**Compile-iterate loop after generation:**
```bash
make juju                       # builds the client only; faster than make install
~/go/bin/juju advisor           # against your current model
```

**Mitigating LLM confusion from the juju tree size.** The juju repo is
large and the LLM might wander. Three mitigations are baked into the
prompts:

- **C.1 constitution** principle #8 ("follow juju conventions ruthlessly")
  + Section 4d's reference patterns scope the LLM's mental model.
- **C.3 plan prompt** names exact file paths to reference
  (`cmd/juju/block/list.go`, `api/client/client.go`, `cmd/juju/status/`)
  and exact paths to write to (`cmd/juju/advisor/`). The LLM should not
  need to crawl elsewhere.
- The `.specify/memory/compliance-context.md` (full brief) is the only
  external reference the LLM needs. If it tries to read other parts of
  the juju tree without a specific reason, redirect: "Reference only
  what's named in the plan."

**Five redirects to be ready with during generation:**

1. **If the LLM proposes persistence:** "Out of scope per Appendix A.1
   non-goals and constitution principle #7."
2. **If the LLM proposes a new facade or controller-side worker:**
   "No new facade. Detection is client-side only — see plan prompt
   'API CALLS'. Use the existing Client facade's FullStatus method."
3. **If the LLM uses standard `testing`:** "Juju uses
   `github.com/juju/tc`. Adapt tests per AGENTS.md 'Unit Test
   Conventions' and the reference in `cmd/juju/block/list_test.go`."
4. **If the LLM hand-writes recommendation strings in Go:** "Point at
   Appendix B fixtures and C.3 'ENRICH FUNCTION'. Recommendation
   content loads from `testdata/findings.json`, not Go."
5. **If the LLM forgets gci import ordering:** "Run
   `gci write --section standard --section default --section
   'Prefix(github.com/juju/juju)' <file>` on every edited Go file.
   See AGENTS.md 'Code Formatting'."

**Sanity-checking commits.** Every time the LLM writes a file, the
juju build hooks may run pre-commit lint (golangci-lint). Don't
disable them — fix the issue. If `make pre-check` fails, that's
generally what CI will fail on too.
