# Juju Controller HA Architecture

Comprehensive research into how Juju controllers achieve High Availability, how they share state, and time-related deterioration risks.

## HA Lifecycle Overview

```
Bootstrap (single node)
  → DQLite binds to 127.0.0.1:17666
  → Singular worker claims "primary controller" lease

Charm writes controller.conf → POSTs to configchange.socket on each controller

Original Node Rebind
  → dbaccessor: rebindAddress() → stop DQLite → rewrite cluster.yaml with cloud-local IP
  → dependency.ErrBounce → worker restarts → DQLite on cloud-local address
  → AddDqliteNode() → controller_node row updated

New Node Joins
  → dbaccessor: joinNodeToCluster() → DQLite Raft join via peer addresses
  → AddDqliteNode() → controller_node row written
  → apiaddresssetter (primary only) → controller_api_address table updated
  → apiremotecaller on all nodes → opens API connections to all peers

Failover
  → DQLite Raft elects new leader
  → Singular lease expires → another node claims primary
  → controllerpresence: broken connection → DeleteMachinePresence/DeleteUnitPresence
```

## Key HA Components

### 1. DQLite Clustering (Port 17666)

- **Single source of truth**: All shared state in distributed DQLite (Raft consensus)
- **Config file**: `<data-dir>/agents/controller-<ID>/controller.conf` (YAML: `db-bind-addresses` map)
- **Signal**: Unix socket `configchange.socket` with `/reload` POST endpoint
- **TLS**: Mutual TLS between IAAS controller nodes; loopback-only for K8s
- **Roles**: Voter (full Raft participant), Standby (replicates, no vote), Spare (connected, inactive)
- **Handover**: On graceful shutdown, `dbApp.Handover(ctx)` transfers leadership (30s timeout)
- **Recovery**: `SetClusterToLocalNode()` reconfigures to single-node if cluster lost

### 2. Primary Controller (Lease-based)

The `singular` worker claims a lease in `SingularControllerNamespace`. Renews at `Duration/2` (30s for 1-min lease).

Workers guarded by `ifPrimaryController`:
- `changestreampruner` — prunes change_log
- `leaseExpiry` — expires stale leases
- `apiAddressSetter` — writes API addresses to DQLite
- `externalControllerUpdater`
- `secretBackendRotate`

### 3. Peer Discovery & Liveness

- **apiremotecaller**: Watches `controller_api_address` table, maintains API connections to all peers
- **controllerpresence**: Monitors peer API connections, cleans up presence on disconnect
- **apiaddresssetter**: Primary-only, publishes each controller's API addresses

### 4. Event Propagation

No pubsub bus. **Change stream** polls local DQLite connection, fans out to local subscribers via event multiplexer. DQLite Raft replication ensures writes on any node appear in all change streams.

### 5. No Request Forwarding

Every controller handles all API requests independently. DQLite internally routes writes to the Raft leader transparently.

### 6. EnableHA Status

`juju enable-ha` → `NotSupportedf("enable HA")`. HA is configured at deployment time via the controller charm writing `controller.conf`.

## The Object Store ("Poor Man's Object Store")

### What's Stored

| Content | Namespace | Store |
|---------|-----------|-------|
| Charm archives | Model UUID | `domain/application/charm/store` |
| Agent binaries | `"controller"` | `domain/agentbinary/service/store` |
| Charm resources | Model UUID | `internal/resource/store` |

### File Backend (Default)

- **Directory**: `<data-dir>/objectstore/<namespace>/` (files named by SHA384 hash)
- **Deduplication**: Content-addressed; multiple paths → one file
- **Locking**: Lease-based (`ObjectStoreNamespace`) during put/remove
- **Pruning**: Every 6 hours, removes unreferenced files

### HA Sharing: Pull-on-Demand

Each controller has its **own local filesystem**. No shared filesystem.

1. Controller A uploads blob → local file + DQLite metadata
2. DQLite replicates metadata to B, C
3. B, C observe metadata change → spawn `fetchWorker` to pull blob from A
4. Fetch via HTTP: `GET /objects?:object=<sha256>` on peer's API server
5. If client requests blob before background fetch: falls back to remote fetch (30s timeout)
6. Retry: up to 10 times, doubling delay, up to 1 minute

### S3 Backend (Optional)

- Enabled via `object-store-type: s3` controller config
- Bucket: `juju-<controller-uuid>`
- All controllers share the same S3 bucket — no remote fallback needed
- Migration: `objectstoredrainer` worker drains file → S3, fortress guard blocks access during drain

## Known Bugs (Active Investigation)

### BUG-1: Spin-Lock in scopedContext.Done() (PR #21857)

**File**: `internal/objectstore/remote/retriever.go:236-262`
**Severity**: Critical -- causes rapid CPU degradation within minutes
**Status**: Fix in PR #21857 by Joseph Phillips (manadart)

The `scopedContext.Done()` method spawns a goroutine that enters a tight spin-lock when:
1. `IgnoreChild()` has been called (after successful blob fetch)
2. The child context (connection context) is cancelled (normal after fetch)
3. The `select` hits the closed child channel, `continue`s, and loops at 100% CPU

Two bugs in one:
- **Spin-lock**: `continue` on a closed channel = tight loop. Fix: set `childDone = nil`
- **Missing `return`**: after `closeDone()` on non-ignored child path, goroutine continues looping

Additionally, `Done()` violates the `context.Context` contract ("Successive calls to Done return the same value") -- it creates a new channel and goroutine on every call. HTTP transport calls `Done()` multiple times per request, multiplying the spin-locked goroutines.

**Reproduction**: `for i in {1..20} ; do juju exec -u <unit> --wait 30s "date" ; done` -- grinds to halt within minutes.

### BUG-2: Object Store Deduplication Delete Bug (Probable Root Cause of Missing Objects)

**File**: `internal/objectstore/fileobjectstore.go:778-792`
**Severity**: Critical -- causes data loss (silent file deletion)
**Status**: Under investigation

The `remove()` method:
1. Calls `RemoveMetadata(ctx, path)` -- deletes path row; conditionally deletes metadata row only if no other paths reference it
2. Calls `deleteObject(ctx, hash)` -- **unconditionally deletes the physical file**

If two paths (A, B) share the same SHA384 hash (content deduplication), removing path A deletes the physical file even though path B still references it. Path B's metadata remains in DQLite but the file is gone. Subsequent `Get` for path B returns "file not found", triggering remote fallback (which triggers BUG-1's spin-lock).

**Fix needed**: `deleteObject` must only delete the file when the metadata row was also deleted (no remaining path references). The DB `RemoveMetadata` already handles the conditional delete -- the file store needs to check whether the metadata row was removed before deleting the file.

### BUG-3: Namespace Mutation in BlobRetriever

**File**: `internal/objectstore/remote/retriever.go:143-146`
**Severity**: Moderate -- corrupts retriever's namespace for all subsequent calls

```go
if r.namespace == database.ControllerNS {
    tag, _ := conn.ModelTag()
    r.namespace = tag.Id()  // MUTATES shared struct field permanently
}
```

The first `retrieve()` call on a controller-namespace BlobRetriever permanently overwrites `r.namespace` with a model UUID. All subsequent retrieval calls use the wrong namespace.

## Time-Related Deterioration Risks

### Critical (Silent Accumulation)

| Risk | Mechanism | Impact |
|------|-----------|--------|
| **Charm secret revisions** | No pruning for charm secrets (only user secrets with `auto_prune=true`) | O(months * rotations/day) rows accumulate unboundedly |
| **Orphaned lease_pin rows** | Entity destroyed without unpinning | Expired leases never cleaned; table grows silently |
| **Change log stalling** | Single slow change stream reader blocks pruner watermark advance | `change_log` table grows until reader recovers; single warning only |

### Moderate (Operational)

| Risk | Mechanism | Impact |
|------|-----------|--------|
| **Reconnection storm** | No jitter on agent reconnect after failover | All agents flood new leader simultaneously |
| **DQLite WAL growth** | No explicit VACUUM; SQLite auto-checkpoint can be blocked by long reads | WAL grows under sustained write load; 250-retry loop for checkpoint contention |
| **Object store blobs** | Charms/binaries stay as long as any model references them; no TTL | Disk grows with deployment history |
| **Clock drift** | Lease expiry uses `datetime('now')` (DB clock); singular renewal uses process clock | Skew >1 min causes lease lapse or false failover |

### Bounded (Low Risk)

| Area | Bound |
|------|-------|
| Audit log | 300 MB/file, 10 backups (~3 GB) |
| Agent logs | 100 MB/file, 2 backups |
| Operation results | 2-week TTL or 5 GiB cap, pruned daily |

## Key Files

| Component | Path |
|-----------|------|
| DQLite HA logic | `internal/worker/dbaccessor/worker.go` |
| Config reload socket | `internal/worker/controlleragentconfig/worker.go` |
| controller.conf reader | `internal/worker/dbaccessor/config.go` |
| Peer API connections | `internal/worker/apiremotecaller/worker.go` |
| Presence tracking | `internal/worker/controllerpresence/worker.go` |
| API address publisher | `internal/worker/apiaddresssetter/worker.go` |
| Primary flag (singular) | `internal/worker/singular/flag.go` |
| Lease manager | `internal/worker/lease/manager.go` |
| Lease state (DQLite) | `domain/lease/state/state.go` |
| Lease expiry worker | `internal/worker/leaseexpiry/worker.go` |
| File object store | `internal/objectstore/fileobjectstore.go` |
| S3 object store | `internal/objectstore/s3objectstore.go` |
| Remote blob retriever | `internal/objectstore/remote/retriever.go` |
| Object store drainer | `internal/worker/objectstoredrainer/worker.go` |
| Change stream pruner | `internal/worker/changestreampruner/pruner.go` |
| Secret revision pruner | `internal/worker/secretspruner/worker.go` |
| Controller node schema | `domain/schema/controller/sql/0010-controller-node.sql` |
| Object store schema | `domain/schema/controller/sql/0014-objectstore-metadata.sql` |
| Manifold wiring | `cmd/jujud-controller/agent/machine/manifolds.go` |
