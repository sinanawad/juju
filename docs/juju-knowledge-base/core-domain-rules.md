# Core Domain Rules

Rules governing Juju's domain model behavior. Source: AGENTS.core-domain-rules.md (PR #21800).

## Entity Lifecycle (Life)

Life is a first-class value for lifecycle-managed entities: machines, units, applications, relations.

### Types
- Canonical types: `core/life.Value` or `domain/life.Life`
- Canonical values: `Alive`, `Dying`, `Dead`
- Never introduce extra states, magic numbers, or ad-hoc string comparisons

### Progression: Alive → Dying → Dead (monotonic, never backwards)

| State | Meaning |
|-------|---------|
| **Alive** | Entity participates normally in model workflows |
| **Dying** | Departure work in progress: relation departure, storage cleanup, cloud instance cleanup |
| **Dead** | All teardown complete. Entity can be deleted. |

### Rules
- `Alive → Dying` must be **idempotent**: repeated removal requests for non-Alive entities are no-ops
- `Dying → Dead` only after **all** dependent cleanup is complete; otherwise return blocking errors
- Delete entities when Dead. Exception: some entities become Dead and are removed in one operation (Dead may not be observed externally)
- **Forced removal** (`--force`) bypasses parts of normal Dying workflow — only use when explicitly required
- Keep lifecycle cascades **transactional** so related entities don't split into inconsistent states

### Implications for Testing
- When tests call `destroy-model` or `remove-application`, they trigger Alive → Dying → Dead
- If cleanup is incomplete, the destroy will block (not silently succeed)
- `--force` exists as an escape hatch but skips departure hooks and cleanup verification
- Substrate verification (K8s namespace gone, LXD containers removed) confirms Dead cleanup actually happened

## Watchers and Notifications

### Watcher Types
- **NotifyWatcher** (`struct{}` notifications) — watch a single thing
- **StringsWatcher** (`[]string` changed identifiers) — watch a collection

### Producer Rules
- Notifications indicate **change, not state**. Do not emit full entity state in watcher payloads.
- For collection watchers: emit initial collection members, then deltas
- Emit **changed identifiers only**; include only entities relevant to the watcher concern

### Consumer Rules
- Consumers must **re-query current state** for changed entities before acting
- Never assume the notification payload contains the current state

### Coalescing and Filtering
- Multiple changes between reads may collapse to one notification or one changed identifier — this is expected
- Mapper/filter logic may drop events; emitting no identifiers is valid, not an error
- Do not treat empty notification as failure

## Relations

- A relation connects two application endpoints with compatible interface types
- Endpoint roles: `provides`, `requires`, `peer`
- Peer relations connect units of the same application (e.g., for clustering)
- Relation lifecycle follows entity Life rules: Alive → Dying → Dead
- Departure hooks (`relation-departed`, `relation-broken`) fire during Dying
- Cross-model relations (CMR) use offers and consumers across model boundaries; the relation data exchange is the same, the transport differs

## Secrets

- Secrets have owners (unit or application) and access grants via relations
- Secret lifecycle: create → grant → get → rotate → expire → revoke → remove
- Secrets are backend-agnostic: internal backend (controller DB) is default; external backends (e.g., Vault) configurable per model
- Secret content is versioned by revision; consumers track their current revision and must explicitly refresh to see newer revisions
- Secret drain: when switching backends, secrets migrate from old to new backend
