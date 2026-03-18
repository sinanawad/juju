# Charm Support in Juju

## Overview

Charms are operator packages that define how applications are deployed and managed. Juju supports multiple charm formats with different capabilities.

**Key Locations:**
- Metadata parsing: `internal/charm/meta.go` (~1300 lines)
- Manifest parsing: `internal/charm/manifest.go`
- Domain types: `domain/application/charm/types.go`
- Hooks: `internal/charm/hooks/hooks.go`

---

## Charm Formats

### Format Detection

```go
// internal/charm/charm.go:679-686
const (
    FormatUnknown Format = iota
    FormatV1      Format = iota
    FormatV2      Format = iota
)
```

**Detection Rules:**
| Format | Criteria |
|--------|----------|
| FormatV1 | No manifest, no bases, no containers |
| FormatV2 | Has manifest with bases OR has containers |

**Selection Reasons (bitflags):**
- `SelectionManifest` - manifest.yaml exists
- `SelectionBases` - at least 1 base in manifest
- `SelectionContainers` - at least 1 container defined

---

## Charm Metadata Structure

### Core Meta Struct

```go
// internal/charm/meta.go:219-242
type Meta struct {
    // v1 fields
    Name           string
    Summary        string
    Description    string
    Subordinate    bool
    Provides       map[string]Relation
    Requires       map[string]Relation
    Peers          map[string]Relation
    ExtraBindings  map[string]ExtraBinding
    Categories     []string
    Tags           []string
    Storage        map[string]Storage      // ← KEY: Indicates statefulness
    Devices        map[string]Device
    Resources      map[string]resource.Meta
    Terms          []string
    MinJujuVersion semversion.Number

    // v2 fields (sidecar charms)
    Containers     map[string]Container    // ← KEY: Workload containers
    Assumes        *assumes.ExpressionTree
    CharmUser      RunAs                   // root, sudoer, non-root
}
```

### Storage Definition

```go
type Storage struct {
    Name        string
    Description string
    Type        StorageType  // "block" or "filesystem"
    Shared      bool         // Shared across units
    ReadOnly    bool
    CountMin    int
    CountMax    int
    MinimumSize uint64
    Location    string
    Properties  []string
}
```

**Storage Types:**
- `StorageBlock` - Block device (rare)
- `StorageFilesystem` - Filesystem mount (common)

### Container Definition (Sidecar Charms)

```go
// internal/charm/meta.go:244-256
type Container struct {
    Resource string  // Reference to OCI image resource
    Mounts   []Mount // Storage mounts
    Uid      *int    // User ID for Pebble
    Gid      *int    // Group ID for Pebble
}

type Mount struct {
    Storage  string // Storage name to mount
    Location string // Mount path in container
}
```

---

## Sidecar Charms (v2 / Kubernetes-Native)

### What Is a Sidecar Charm?

A **sidecar charm** is a Kubernetes charm where the charm (operator) runs in its own container alongside workload containers in the same pod. This is the modern pattern for K8s charms.

**Pod Structure:**
```
┌─────────────────────────────────────────────┐
│  Kubernetes Pod (= 1 Juju Unit)             │
├─────────────────────────────────────────────┤
│  Init Container (setup, secrets)            │
├─────────────────────────────────────────────┤
│  Charm Container (sidecar, runs operator)   │
├─────────────────────────────────────────────┤
│  Workload Container 1 (runs Pebble)         │
│  Workload Container 2 (runs Pebble)         │
│  ...                                        │
└─────────────────────────────────────────────┘
```

### Pebble Integration

**Pebble** is a lightweight init system injected into every workload container.

**Workload Hooks** (triggered by Pebble events):
```go
// internal/charm/hooks/hooks.go:100-105
const (
    PebbleReady          Kind = "pebble-ready"
    PebbleCheckFailed    Kind = "pebble-check-failed"
    PebbleCheckRecovered Kind = "pebble-check-recovered"
    PebbleCustomNotice   Kind = "pebble-custom-notice"
)
```

**Health Check Ports:**
- API Server (jujud): 38811
- Charm container: 38812
- Workload containers: 38813+ (incremented)

### Example Sidecar Charm Metadata

```yaml
# metadata.yaml
name: my-webapp
containers:
  nginx:
    resource: nginx-image
    mounts:
      - storage: web-content
        location: /var/www/html
  redis:
    resource: redis-image
resources:
  nginx-image:
    type: oci-image
  redis-image:
    type: oci-image
storage:
  web-content:
    type: filesystem
charm-user: non-root
```

```yaml
# manifest.yaml
bases:
  - name: ubuntu
    channel: 22.04/stable
    architectures: [amd64, arm64]
```

---

## CharmUser (Security Context)

```go
const (
    RunAsDefault RunAs = ""
    RunAsRoot    RunAs = "root"      // UID 0
    RunAsSudoer  RunAs = "sudoer"    // UID 171, has sudo
    RunAsNonRoot RunAs = "non-root"  // UID 170, restricted
)
```

---

## How Charm Metadata Flows to K8s

### Current Flow (Incomplete)

```
Charm metadata.yaml
        ↓
internal/charm/meta.go (parsing)
        ↓
domain/application/charm/types.go (Metadata struct)
        ↓
apiserver facade (CAASApplicationProvisioningInfo)
        ↓  ⚠️ NO DeploymentType field
worker (caasapplicationprovisioner)
        ↓  ⚠️ Hardcodes DeploymentStateful
broker.Application(name, caas.DeploymentStateful)
        ↓
internal/provider/kubernetes/application/
        ↓
K8s StatefulSet (always)
```

### What's Missing for Deployment/DaemonSet Support

**1. No deployment type field in metadata:**
```yaml
# metadata.yaml - DOES NOT EXIST
deployment:
  type: stateless  # or daemon
```

**2. No DeploymentType in RPC params:**
```go
// rpc/params/caas.go - MISSING
type CAASApplicationProvisioningInfo struct {
    // ... existing fields ...
    // DeploymentType string `json:"deployment-type"` // MISSING
}
```

**3. Worker hardcodes deployment type:**
```go
// internal/worker/caasapplicationprovisioner/application.go:148-149
// TODO(sidecar): support more than statefulset
app := a.broker.Application(name, caas.DeploymentStateful)  // HARDCODED
```

---

## Determining Deployment Type from Metadata

### Possible Heuristics (If No Explicit Field)

| Charm Characteristic | Suggested K8s Type |
|---------------------|-------------------|
| Has `Storage` with `Shared: false` | StatefulSet (needs stable PVCs) |
| Has `Storage` with `Shared: true` | Deployment (shared storage OK) |
| No `Storage` at all | Deployment (stateless) |
| Subordinate charm | Depends on principal |
| Explicit `deployment: daemon` | DaemonSet |

### Storage Field Significance

```go
type Storage struct {
    Shared bool  // KEY: If false, each unit needs its own PVC
}
```

- **`Shared: false`** (default) → StatefulSet (stable pod identity + PVC per replica)
- **`Shared: true`** → Deployment could work (all pods share storage)
- **No storage** → Deployment is fine (truly stateless)

### Code Location for Potential Logic

```go
// Could be added to: internal/worker/caasapplicationprovisioner/ops.go
func determineDeploymentType(meta *charm.Meta, explicitType caas.DeploymentType) caas.DeploymentType {
    // If explicit type specified in metadata, use it
    if explicitType != "" {
        return explicitType
    }

    // Check for per-unit storage (needs StatefulSet)
    for _, storage := range meta.Storage {
        if !storage.Shared {
            return caas.DeploymentStateful
        }
    }

    // No per-unit storage requirement, can use Deployment
    return caas.DeploymentStateless
}
```

---

## Machine vs Kubernetes Charms

| Aspect | Machine Charm | K8s Sidecar Charm |
|--------|--------------|-------------------|
| Format | v1 or v2 | v2 only |
| Target | Ubuntu machines | K8s pods |
| Hooks | Shell scripts | Python/ops framework |
| Workload | Native processes | Containers via Pebble |
| Storage | Machine disks | PVCs |
| Networking | Machine IP | ClusterIP/LoadBalancer |
| Init system | systemd | Pebble |

---

## Key Files Reference

| Purpose | Path |
|---------|------|
| Metadata parsing | `internal/charm/meta.go` |
| Manifest parsing | `internal/charm/manifest.go` |
| Hook definitions | `internal/charm/hooks/hooks.go` |
| Domain types | `domain/application/charm/types.go` |
| Provisioning info | `rpc/params/caas.go` |
| Worker hardcode | `internal/worker/caasapplicationprovisioner/application.go:149` |
| K8s pod spec | `internal/provider/kubernetes/application/application.go` |
| Pebble health | `internal/provider/kubernetes/pebble/pebble.go` |

---

## Historical Context: The `deployment` Field

**CRITICAL: A `deployment` field EXISTED and was REMOVED in May 2024 (commit 327acab6b30).**

The old field was for **podspec charms** (legacy K8s charm format, now deprecated):
```go
// REMOVED - was in internal/charm/meta.go
type Deployment struct {
    DeploymentType string  // stateless, stateful, daemon
    DeploymentMode string
    ServiceType    string
    MinVersion     string
}
```

**Why removed:**
- Podspec charms are no longer supported in Juju 4.0+
- All K8s charms now use sidecar pattern (containers + Pebble)
- The field was a v1 marker, preventing use with v2 charms

**Format detection** (`internal/charm/meta.go:1276`):
```go
// v1 markers (cannot coexist with v2)
v1Markers := []string{"series", "deployment", "min-juju-version"}
// v2 markers
v2Markers := []string{"containers", "assumes", "charm-user"}
```

This means **adding a new deployment field for sidecar charms would be a NEW feature**, not restoring old functionality.

---

## Adding New Metadata Fields (Pattern from containers/charm-user)

To add a new field requires changes across 5+ layers:

| Layer | File | Change |
|-------|------|--------|
| Schema | `internal/charm/meta.go` | Add struct, schema, parser |
| Domain types | `domain/application/charm/types.go` | Add to Metadata struct |
| Domain service | `domain/application/service/metadata.go` | encode/decode functions |
| State | `domain/application/state/metadata.go` | DB persistence |
| RPC params | `rpc/params/charms.go` | Wire format |
| API client | `api/common/charms/common.go` | Conversion |
| API server | `apiserver/common/charms/common.go` | Conversion |

**Charmcraft impact:** Would need corresponding update to Charmcraft's metadata.yaml validation.

---

## TODO/Sidecar Work References

Multiple TODOs reference incomplete "sidecar" work:
- `internal/worker/caasapplicationprovisioner/application.go:148` - "support more than statefulset"
- `internal/provider/kubernetes/application/application.go:522` - "Unify this with Ensure"
- `internal/provider/kubernetes/application/application.go:705` - "juju expose for sidecar charms"
- `domain/application/service/provider.go:437` - "We currently only support statefulset"

These indicate the deployment type flexibility was planned but never completed.
