# Kubernetes Provider Architecture in Juju

## Overview

Juju's Kubernetes support is implemented through the CAAS (Container as a Service) abstraction layer. The provider manages K8s workloads using Deployments, StatefulSets, and DaemonSets, with support for storage, secrets, ingress, and admission webhooks.

**Key Locations:**
- Provider: `/internal/provider/kubernetes/` (~15K lines)
- CAAS abstractions: `/caas/`
- K8s-specific workers: `/internal/worker/caas*/`
- Domain CAAS support: `/domain/*/` (scattered)
- API facades: `/apiserver/facades/*/caas*/`

---

## Architecture Layers

```
┌─────────────────────────────────────────────────────────────┐
│  CLI: juju add-k8s, juju deploy (CAAS model)                │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│  API Facades (7 CAAS-specific)                              │
│  - CAASAgent, CAASApplication, CAASAdmission                │
│  - CAASApplicationProvisioner, CAASModelOperator            │
│  - CAASModelConfigManager, CAASOperatorUpgrader             │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│  Workers (10 CAAS workers)                                  │
│  - caasbroker (broker lifecycle)                            │
│  - caasapplicationprovisioner (pod reconciliation)          │
│  - caasfirewaller (network policies)                        │
│  - caasadmission (webhook handler)                          │
│  - caasrbacmapper (ServiceAccount→App mapping)              │
│  - caasmodelconfigmanager (image credentials)               │
│  - caasmodeloperator (model operator pod)                   │
│  - caasprober/caasprobebinder (health probes)               │
│  - caasupgrader (operator upgrades)                         │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│  Broker Interface (caas.Broker)                             │
│  - Application management                                    │
│  - Service exposure                                          │
│  - Storage provisioning                                      │
│  - Secret management                                         │
│  - Model operator lifecycle                                  │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│  kubernetesClient (internal/provider/kubernetes/k8s.go)     │
│  - Kubernetes API interactions                               │
│  - Resource management (48+ types)                          │
│  - Informer-based watching                                   │
└─────────────────────────────────────────────────────────────┘
```

---

## Provider Structure

### Core Files
| File | Purpose |
|------|---------|
| `k8s.go` | Main `kubernetesClient` (39KB) |
| `provider.go` | `EnvironProvider` implementation |
| `bootstrap.go` | Controller bootstrap to K8s (43KB) |
| `teardown.go` | Model destruction |
| `cloud.go` | Cloud detection (microk8s, GKE, EKS, AKS) |
| `credentials.go` | Credential types and validation |
| `namespaces.go` | Namespace management |
| `storage.go` | Storage validation and provisioning |
| `services.go` | K8s Service management |
| `modeloperator.go` | Model operator deployment |

### Subdirectories
| Directory | Purpose |
|-----------|---------|
| `application/` | Per-app deployment lifecycle (78KB+) |
| `resources/` | K8s resource implementations (48 types) |
| `watcher/` | K8s informer-based watchers |
| `storage/` | Storage provider implementations |
| `constants/` | Labels, versions, constants |
| `proxy/` | Application port forwarding |
| `exec/` | Pod command execution |
| `pebble/` | Pebble service management |

---

## Workload Types

**IMPORTANT: Only StatefulSet is actually used in production.**

The types are defined but not fully wired:
```go
DeploymentStateless  // K8s Deployment - DEFINED BUT NOT USED
DeploymentStateful   // K8s StatefulSet - THE ONLY TYPE USED
DeploymentDaemon     // K8s DaemonSet - DEFINED BUT NOT USED
```

**Evidence:**
- `internal/worker/caasapplicationprovisioner/application.go:148-149`:
  ```go
  // TODO(sidecar): support more than statefulset
  app := a.broker.Application(name, caas.DeploymentStateful)
  ```
- `domain/application/service/provider.go:437-439`:
  ```go
  // We currently only support statefulset.
  caasApp := caasApplicationProvider.Application(appName, caas.DeploymentStateful)
  ```

**What exists vs what's wired:**
| Layer | Deployment/DaemonSet Support |
|-------|------------------------------|
| Provider (`internal/provider/kubernetes/application/`) | ✅ Fully implemented |
| Unit tests | ✅ Tested |
| Workers (`caasapplicationprovisioner`) | ❌ Hardcodes StatefulSet |
| Domain (`domain/application/service/`) | ❌ Hardcodes StatefulSet |
| Charm metadata | ❌ No field for deployment type |
| RPC params | ❌ No DeploymentType parameter |
| CLI/API | ❌ No way to specify type |

**To enable Deployment/DaemonSet would require:**
1. Add `deployment-type` constraint to `core/constraints/constraints.go`
2. Wire through domain layer (`domain/constraints/constraints.go`)
3. Add heuristic function based on Storage field (non-shared = StatefulSet)
4. Update `caasapplicationprovisioner` worker to use dynamic type
5. Handle lifecycle differences (DaemonSet has no replicas concept)

**Constraint approach** (preferred over metadata field):
- No Charmcraft changes needed
- Operators can override at deploy time: `--constraints="deployment-type=stateless"`
- Existing charms work without modification

### Service Exposure Types
```go
ServiceCluster       // ClusterIP (default)
ServiceLoadBalancer  // LoadBalancer
ServiceExternal      // ExternalName
ServiceOmit          // No service created
```

---

## Label Versioning

Three label versions for backward compatibility:

| Version | Format | Example |
|---------|--------|---------|
| Legacy (0) | `juju-*` | `juju-app=mysql` |
| v1 | Domain-based | `model.juju.is/name=mymodel` |
| v2 | UUID-based | `model.juju.is/id=abc123` |

v2 enables multi-controller clusters by using UUIDs instead of names.

---

## CAAS vs IAAS Differences

| Aspect | IAAS | CAAS/K8s |
|--------|------|----------|
| Cloud Type | MAAS, AWS, Azure, etc. | Kubernetes only |
| Unit Placement | Machines + containers | Pods (no machines) |
| Network Endpoint | Machine/container IP | CloudService (LB) |
| Secrets Storage | Juju internal DB | K8s Secrets API |
| Scaling | Manual unit add | Replica count |
| Domain Types | `AddIAASUnitArg` | `AddCAASUnitArg` |

---

## Key Interfaces

### Broker (`caas/broker.go`)
```go
type Broker interface {
    environs.InstancePrechecker
    environs.BootstrapEnviron
    environs.Networking
    StorageValidator
    ApplicationBroker
    ServiceManager
    ModelOperatorManager
    ProxyManager
    // ...
}
```

### Application (`internal/provider/kubernetes/application/`)
```go
type Application interface {
    Ensure(config ApplicationConfig) error
    Exists() (DeploymentState, error)
    Delete() error
    Watch(context.Context) (watcher.NotifyWatcher, error)
    WatchReplicas() (watcher.NotifyWatcher, error)
    Scale(int) error
    Trust(bool) error
    State() (ApplicationState, error)
    Units() ([]Unit, error)
    Service() (*Service, error)
    UpdatePorts([]ServicePort, bool) error
    EnsurePVCs(...) error
}
```

---

## Worker Reconciliation Flow

```
1. caasbroker loads K8s connection
          ↓
2. caasapplicationprovisioner watches:
   - WatchApplications() (which apps exist)
   - WatchApplicationScale() (desired replicas)
   - WatchApplicationUnitLife() (unit changes)
          ↓
3. Per-app worker:
   - Watches WatchProvisioningInfo()
   - Calls broker.Application().Ensure()
   - Updates unit status via UpdateCAASUnit()
          ↓
4. caasadmission webhook:
   - Intercepts K8s resource creates/updates
   - Annotates with Juju labels
   - caasrbacmapper maps ServiceAccount→App
```

---

## API Facades Summary

| Facade | Side | Purpose |
|--------|------|---------|
| `CAASAgent` | Agent | Cloud spec access |
| `CAASApplication` | Agent | Unit introduction/termination |
| `CAASAdmission` | Agent | Admission webhook config |
| `CAASApplicationProvisioner` | Controller | Provisioning info, scaling |
| `CAASModelOperator` | Controller | Model operator deployment |
| `CAASModelConfigManager` | Controller | Controller config |
| `CAASOperatorUpgrader` | Controller | Operator version upgrades |

---

## Potential Gaps & Incomplete Work

### 1. **Deployment/DaemonSet Support (Incomplete)**
- Provider layer has full implementation for all three types
- Production code hardcodes `DeploymentStateful` everywhere
- TODO comments reference "sidecar" work that was never completed
- No charm metadata or API path to specify deployment type
- **Impact**: All K8s charms run as StatefulSets even when Deployment would be more appropriate

### 2. **Limited Storage Features**
- `FilesystemProvisioningInfo()` marked TODO in provisioner facade
- No CSI driver integration beyond basic StorageClass
- Volume snapshot support unclear

### 3. **Network Policy Gaps**
- `caasfirewaller` worker exists but NetworkPolicy support appears basic
- No native Ingress class selection
- No service mesh integration (Istio, Linkerd)

### 4. **Multi-Cluster Support**
- Single cluster per model assumption
- No federation or multi-cluster deployment
- Cross-cluster relations not supported

### 5. **Resource Quotas & Limits**
- No ResourceQuota management per model/namespace
- LimitRange support unclear
- No PodDisruptionBudget management

### 6. **Observability Gaps**
- No native Prometheus/metrics integration
- No tracing support
- Pod logs accessible but no aggregation

### 7. **Advanced K8s Features Missing**
- No HorizontalPodAutoscaler (HPA) integration
- No VerticalPodAutoscaler (VPA)
- No PodTopologySpread constraints
- No pod priority/preemption
- No init container support for charms
- No sidecar container injection (except admission webhook)

### 8. **Security Features**
- Pod Security Standards/Policies not managed
- No OPA/Gatekeeper integration
- Network policies basic
- No Secrets encryption-at-rest configuration

### 9. **Operator Pattern**
- Single model operator per model
- No per-application operator pods (uses single deployment)
- Charm containers run with single operator deployment

### 10. **CRD/Operator Support**
- CRD management exists but not well documented
- No Operator Lifecycle Manager (OLM) integration
- Custom resources tracked but lifecycle unclear

### 11. **Cloud Provider Integration**
- GKE/EKS/AKS detection works
- No cloud-specific features (IAM roles, managed identities)
- No workload identity integration

---

## Files to Explore for Expansion

| Area | Key Files |
|------|-----------|
| Provider extension | `internal/provider/kubernetes/provider.go` |
| New resource types | `internal/provider/kubernetes/resources/` |
| New workers | `internal/worker/caas*/` patterns |
| Domain CAAS | `domain/application/service/provider.go` |
| API expansion | `apiserver/facades/*/caas*/` |
| Storage | `internal/provider/kubernetes/storage.go` |
| Networking | `internal/provider/kubernetes/services.go` |

---

## References

- [Main k8s provider](file:///data/dev/juju/internal/provider/kubernetes/)
- [CAAS broker interface](file:///data/dev/juju/caas/broker.go)
- [Application provisioner worker](file:///data/dev/juju/internal/worker/caasapplicationprovisioner/)
- [Model type determination](file:///data/dev/juju/domain/model/service/service.go) lines 529-546
