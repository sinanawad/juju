# Feature Specification: Kubernetes Deployment and DaemonSet Workload Types

**Feature Branch**: `001-k8s-deployment-types`
**Created**: 2026-02-06
**Status**: Draft
**Input**: User description: "Add support for Deployment and DaemonSet workload types in the Kubernetes provider"

## Clarifications

### Session 2026-02-06

- Q: What charm storage characteristics trigger StatefulSet vs Deployment default? → A: Any charm declaring persistent storage defaults to StatefulSet; only charms with no storage declarations default to Deployment.
- Q: Should the deployment-type constraint be silently ignored or produce a warning on IAAS models? → A: Silently ignore, consistent with existing K8s-only constraint behavior.
- Q: Where should the workload type appear in the status output? → A: Both: as a column in the applications summary table and in per-application detail output.

### Session 2026-02-10 — Storage Adaptation Research

- Q: How does the K8s provider create PVCs for Deployment/DaemonSet? → A: Via `handlePVCForStatelessResource` which creates standalone PVCs that all pods share (unlike StatefulSet's `VolumeClaimTemplates` which create per-pod PVCs).
- Q: Does `Delete()` clean up PVCs? → A: No. PVCs are never deleted for any workload type. StatefulSet PVCs follow K8s retention policy; Deployment/DaemonSet PVCs are fully orphaned.
- Q: Does `ClearCAASUnitCloudContainer()` clean up storage attachments? → A: No. It only deletes `k8s_pod` and `k8s_pod_port` rows. Filesystem/volume attachment records are left stale.
- Q: Is `FilesystemProvisioningInfo()` implemented? → A: No. The facade method returns empty (TODO). This blocks `EnsurePVCs` for scaling operations.
- Q: What happens when a DaemonSet uses ReadWriteOnce storage on a multi-node cluster? → A: Only the pod on the node where the PV is bound can mount it. Other nodes' pods fail with `Multi-Attach error`. No deploy-time validation exists.
- Q: Is implementing the `FilesystemProvisioningInfo()` facade stub in scope for the storage stories? → A: No. It is a separate prerequisite. These stories assume the facade works or note where it blocks functionality.
- Q: Should RWO storage + Deployment/DaemonSet at scale > 1 be a blocking error or non-blocking warning? → A: Non-blocking warning. The operator explicitly overrode inference, so the system should warn (in both controller logs and CLI output) but not block deployment.
- Q: Is the `remove-application` race fix in scope for the storage stories? → A: No. US8 adds PVC deletion to `Delete()` so it works when called, but the pre-existing race that prevents `Delete()` from running is a separate fix.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Deploy a Stateless Application (Priority: P1)

As an operator deploying a stateless charm (e.g., a web frontend or API gateway) to a Kubernetes model, I want my application to run as a Deployment workload rather than a StatefulSet, so that I get faster rollouts, simpler scaling, and behavior that matches how stateless workloads are conventionally run on Kubernetes.

**Why this priority**: This is the most common use case. Many production Kubernetes workloads are stateless and should use Deployments for proper rolling updates, surge capacity during upgrades, and reduced operational overhead. Currently all charms are forced into StatefulSets regardless of whether they need stable storage or stable network identities.

**Independent Test**: Can be fully tested by deploying any charm without persistent storage requirements to a Kubernetes model and verifying it runs as a Deployment workload with proper scaling and lifecycle behavior.

**Acceptance Scenarios**:

1. **Given** a Kubernetes model and a charm with no storage declarations in its metadata, **When** the operator deploys the charm without specifying a workload type, **Then** the system automatically selects the Deployment workload type.
2. **Given** a Kubernetes model, **When** the operator deploys a charm and explicitly specifies the "stateless" workload type via a constraint, **Then** the application runs as a Deployment workload regardless of its storage requirements.
3. **Given** a running Deployment-based application, **When** the operator scales up or down, **Then** the replica count changes and new instances start without requiring stable ordinal identities.
4. **Given** a running Deployment-based application, **When** the operator views application status, **Then** the displayed workload type indicates "Deployment" and unit counts are accurate.

---

### User Story 2 - Explicit Workload Type Selection via Constraints (Priority: P1)

As an operator, I want to explicitly choose the workload type (stateless, stateful, or daemon) when deploying a charm to a Kubernetes model, so that I can override the automatic selection when I know better what my application needs.

**Why this priority**: Automatic inference is a good default, but operators need explicit control. A charm might declare storage for caching purposes but still be best served by a Deployment. Explicit override is essential for production use.

**Independent Test**: Can be fully tested by deploying the same charm three times with different workload type constraints and verifying each uses the correct underlying workload type.

**Acceptance Scenarios**:

1. **Given** a Kubernetes model, **When** the operator deploys a charm with the constraint `deployment-type=stateless`, **Then** the application runs as a Deployment workload.
2. **Given** a Kubernetes model, **When** the operator deploys a charm with the constraint `deployment-type=stateful`, **Then** the application runs as a StatefulSet workload.
3. **Given** a Kubernetes model, **When** the operator deploys a charm with the constraint `deployment-type=daemon`, **Then** the application runs as a DaemonSet workload.
4. **Given** a Kubernetes model, **When** the operator deploys a charm with an invalid deployment-type value, **Then** the system returns a clear error message listing the valid options.
5. **Given** an IAAS (non-Kubernetes) model, **When** the operator specifies a deployment-type constraint, **Then** the system silently ignores the constraint (no warning or error).

---

### User Story 3 - Deploy a Node-Level Agent as a DaemonSet (Priority: P2)

As an operator deploying a monitoring agent, log collector, or security scanner charm, I want my application to run as a DaemonSet so that exactly one instance runs on every node in the cluster, matching how node-level agents are conventionally operated on Kubernetes.

**Why this priority**: DaemonSets serve a distinct and important use case for infrastructure-level charms (monitoring, logging, security). While less common than stateless Deployments, this workload type enables an entire class of Kubernetes-native operations that are currently impossible with Juju.

**Independent Test**: Can be fully tested by deploying a charm with the "daemon" workload type constraint and verifying that one instance is scheduled per cluster node.

**Acceptance Scenarios**:

1. **Given** a Kubernetes model with multiple nodes, **When** the operator deploys a charm with the constraint `deployment-type=daemon`, **Then** one instance of the application runs on each node.
2. **Given** a running DaemonSet-based application, **When** a new node is added to the cluster, **Then** an instance of the application is automatically scheduled on the new node.
3. **Given** a running DaemonSet-based application, **When** the operator attempts to manually scale the application, **Then** the system returns a clear error explaining that DaemonSet applications are scaled by adding or removing cluster nodes, not by setting a replica count.
4. **Given** a running DaemonSet-based application, **When** the operator views application status, **Then** the displayed workload type indicates "DaemonSet" and the instance count reflects the number of nodes running the application.

---

### User Story 4 - Backward-Compatible Default Behavior (Priority: P1)

As an operator with existing deployed charms on Kubernetes, I want the system to continue working exactly as it does today for all my existing applications, so that this new feature does not disrupt any running workloads.

**Why this priority**: Backward compatibility is critical. Existing charms that rely on StatefulSet semantics (stable network identity, ordered deployment, persistent volumes) must not break.

**Independent Test**: Can be fully tested by deploying any existing charm that uses persistent storage and verifying it continues to run as a StatefulSet with no behavior changes.

**Acceptance Scenarios**:

1. **Given** a Kubernetes model and a charm with persistent storage requirements, **When** the operator deploys the charm without specifying a workload type, **Then** the system automatically selects the StatefulSet workload type (preserving current behavior).
2. **Given** existing running applications deployed before this feature existed, **When** the system is upgraded, **Then** all existing applications continue running as StatefulSets with no disruption.
3. **Given** a charm that previously deployed as a StatefulSet, **When** the operator redeploys or upgrades it without specifying a workload type, **Then** the application continues as a StatefulSet.

---

### User Story 5 - Workload Type Visibility in Status (Priority: P2)

As an operator managing multiple applications across a Kubernetes model, I want to see the workload type of each application in the status output, so that I can quickly understand how each application is deployed and troubleshoot any issues.

**Why this priority**: Visibility into workload types is important for operational awareness, especially as models now contain a mix of Deployments, StatefulSets, and DaemonSets. Without this, operators would need to inspect the Kubernetes cluster directly.

**Independent Test**: Can be fully tested by deploying applications with different workload types and running the status command to verify workload type information is displayed.

**Acceptance Scenarios**:

1. **Given** a Kubernetes model with applications using different workload types, **When** the operator views the model status summary, **Then** the applications table includes a workload type column showing each application's type (Deployment, StatefulSet, or DaemonSet).
2. **Given** a Kubernetes model, **When** the operator views the detail output for a specific application, **Then** the workload type is included in the application detail.
3. **Given** an IAAS model, **When** the operator views the model status, **Then** no workload type column is displayed (not applicable to non-Kubernetes models).

---

### User Story 6 - Pod Recovery and Resilience (Priority: P1)

As an operator running a Deployment or DaemonSet application on Kubernetes, I want pods that are deleted (due to node failure, eviction, manual deletion, or rolling updates) to be automatically replaced by Kubernetes and seamlessly re-registered in Juju, so that my application self-heals without manual intervention.

**Why this priority**: Without this fix, Deployment and DaemonSet pods that are replaced by Kubernetes (a normal operational event) fail to re-register, leaving units permanently stuck. This is a critical gap for any non-StatefulSet workload.

**Independent Test**: Can be tested by deploying a Deployment application, deleting a pod, and verifying the replacement pod re-registers the existing unit.

**Acceptance Scenarios**:

1. **Given** a running Deployment app, **When** a pod is deleted, **Then** K8s creates a replacement pod and Juju reassigns the existing unit to it within one worker cycle (~10s).
2. **Given** a running DaemonSet app, **When** a node is drained/removed, **Then** pods on the removed node have their stale k8s_pod entries cleared and replacement pods on other nodes register correctly.
3. **Given** a running StatefulSet app, **When** a pod is deleted, **Then** K8s recreates it with the same name and existing registration succeeds (no regression).
4. **Given** a multi-replica Deployment, **When** multiple pods are replaced simultaneously, **Then** each replacement pod is correctly matched to a distinct existing unit.

**Edge case**: If the worker hasn't run yet when the new pod registers, registration fails with a retryable error. The agent retries, and after the worker clears the stale entry (next cycle), registration succeeds.

---

### User Story 7 - Storage Attachment Cleanup on Pod Replacement (Priority: P1)

As an operator running a Deployment or DaemonSet application that uses persistent storage, I want the system to automatically clean up stale storage attachment records when pods are replaced, so that replacement pods can re-register without errors and my application continues operating with its storage intact.

**Why this priority**: This directly extends pod recovery (US6). Without cleaning up stale filesystem and volume attachment records alongside the stale k8s_pod entries, replaced pods encounter duplicate key errors during re-registration. This makes any storage-bearing Deployment/DaemonSet unable to self-heal after pod replacement — a critical operational gap.

**Independent Test**: Can be tested by deploying a Deployment with persistent storage, deleting a pod, and verifying the replacement pod re-registers and remounts storage without errors.

**Acceptance Scenarios**:

1. **Given** a running Deployment with persistent storage and a pod is deleted, **When** the worker detects the stale pod entry and clears it, **Then** the associated filesystem attachment and volume attachment records for that unit are also cleared, allowing the replacement pod to re-register with fresh storage bindings.
2. **Given** a running Deployment with persistent storage at scale=3, **When** a single pod is replaced, **Then** the replacement pod remounts the shared PVC and the unit returns to active/idle without duplicate key errors in the controller logs.
3. **Given** a running StatefulSet with persistent storage, **When** a pod is replaced (same name), **Then** storage attachment records remain unchanged and the unit re-registers normally (no regression).
4. **Given** a running Deployment with persistent storage at scale=3, **When** all pods are replaced simultaneously, **Then** stale storage attachment records for all 3 units are cleared and at least 2 of 3 replacement pods successfully re-register with storage (consistent with the S2.3 known limitation for k8s_pod recovery).

---

### User Story 8 - PVC Cleanup on Non-StatefulSet Application Removal (Priority: P1)

As an operator removing a Deployment or DaemonSet application that uses persistent storage, I want all PersistentVolumeClaims created by Juju to be deleted when the application is removed, so that storage resources are not leaked in the Kubernetes namespace.

**Why this priority**: When Juju creates standalone PVCs for Deployment/DaemonSet workloads (as opposed to StatefulSet VolumeClaimTemplates), those PVCs are never cleaned up. Over time, this causes storage capacity to be consumed by orphaned PVCs that operators may not notice. This is a resource leak that worsens with each deploy/remove cycle.

**Independent Test**: Can be tested by deploying a Deployment with storage, verifying PVCs exist in the namespace, removing the application, and confirming the PVCs are deleted.

**Acceptance Scenarios**:

1. **Given** a running Deployment with persistent storage, **When** the operator removes the application, **Then** all PersistentVolumeClaims created by Juju for that application are deleted from the Kubernetes namespace.
2. **Given** a running DaemonSet with persistent storage, **When** the operator removes the application, **Then** all PersistentVolumeClaims created by Juju for that application are deleted from the Kubernetes namespace.
3. **Given** a running StatefulSet with persistent storage, **When** the operator removes the application, **Then** PVC behavior is unchanged from current behavior (PVCs created by VolumeClaimTemplates follow the existing retention policy — no regression).
4. **Given** a Deployment with persistent storage that was forcefully removed (model destroy), **When** the operator inspects the namespace, **Then** PVCs bearing Juju labels for the removed application are identifiable for manual cleanup.

---

### User Story 9 - Deployment with Shared Persistent Storage (Priority: P2)

As an operator deploying a charm with persistent storage as a Deployment workload (via explicit `deployment-type=stateless` constraint override), I want all replicas to share a single persistent volume using ReadWriteMany access, so that I can use Deployments for workloads that need shared caches, log directories, or configuration stores without data isolation between replicas.

**Why this priority**: The current provider creates a single standalone PVC for Deployments that all pods reference. This works correctly with ReadWriteMany (RWX) storage classes, but silently fails with ReadWriteOnce (RWO) when scale > 1 — only one pod can mount the volume, others hang. Operators need clear feedback about access mode requirements.

**Independent Test**: Can be tested by deploying a storage-bearing charm as a Deployment with a ReadWriteMany storage class, scaling to 3, and verifying all pods mount the shared volume.

**Acceptance Scenarios**:

1. **Given** a Kubernetes model with a ReadWriteMany-capable storage class, **When** the operator deploys a storage-bearing charm with `deployment-type=stateless`, **Then** a single shared PVC is created and all pods mount it concurrently.
2. **Given** a running Deployment with shared storage at scale=1, **When** the operator scales to 3, **Then** all 3 pods mount the existing shared PVC without creating new PVCs.
3. **Given** a Kubernetes model where the default storage class only supports ReadWriteOnce, **When** the operator deploys a storage-bearing charm with `deployment-type=stateless` and scale > 1, **Then** the system issues a non-blocking warning (in CLI output and controller logs) explaining that the storage class should support ReadWriteMany for shared Deployment storage, but proceeds with deployment.
4. **Given** a running Deployment with shared storage, **When** a pod is replaced, **Then** the replacement pod remounts the same shared PVC seamlessly.

---

### User Story 10 - DaemonSet Storage Access Mode Validation (Priority: P2)

As an operator deploying a DaemonSet charm that declares persistent storage, I want the system to warn me at deploy time when the storage access mode is incompatible with DaemonSet semantics (one pod per node, all sharing a single PVC), so that I am informed before discovering at runtime that pods on other nodes cannot mount the volume.

**Why this priority**: DaemonSets run one pod per node. When a DaemonSet charm declares persistent storage, Juju creates a single standalone PVC that all pods reference. With ReadWriteOnce storage (the most common default), only the pod on the node where the PV is physically bound can mount it — all other nodes' pods fail with `Multi-Attach error`. This is a confusing runtime failure that should be caught at deploy time.

**Independent Test**: Can be tested by attempting to deploy a storage-bearing charm as a DaemonSet on a multi-node cluster and verifying the system rejects it (or warns) when the storage class doesn't support ReadWriteMany.

**Acceptance Scenarios**:

1. **Given** a multi-node Kubernetes cluster, **When** the operator deploys a charm with persistent storage using `deployment-type=daemon` and the storage class only supports ReadWriteOnce, **Then** the system issues a non-blocking warning (in CLI output and controller logs) explaining that DaemonSet storage should use ReadWriteMany access mode or ephemeral storage, but proceeds with deployment.
2. **Given** a multi-node Kubernetes cluster with a ReadWriteMany storage class, **When** the operator deploys a charm with persistent storage using `deployment-type=daemon`, **Then** the deployment succeeds and one pod per node mounts the shared PVC.
3. **Given** a charm that declares only ephemeral storage (tmpfs or rootfs provider), **When** the operator deploys it with `deployment-type=daemon`, **Then** the deployment succeeds without storage access mode validation (ephemeral storage is always per-pod).
4. **Given** a single-node cluster, **When** the operator deploys a charm with persistent RWO storage using `deployment-type=daemon`, **Then** the deployment succeeds (only one node → only one pod → RWO is sufficient).

---

### User Story 11 - Ephemeral Storage for Stateless Workloads (Priority: P2)

As an operator deploying a Deployment or DaemonSet charm that needs temporary scratch space, caching, or write-ahead logs, I want to use ephemeral storage (EmptyDir or tmpfs) that is local to each pod and doesn't require persistent volumes, so that my stateless workloads can use fast local storage without the complexity of PVC management.

**Why this priority**: Many Kubernetes workloads need temporary storage (caches, buffers, sockets) but not persistence across pod replacements. Ephemeral storage (EmptyDir/tmpfs) is the standard Kubernetes answer — it's created with the pod, destroyed with the pod, and requires no PVC provisioning. Supporting this cleanly for Deployments and DaemonSets removes the main reason operators might force a StatefulSet when they don't need one.

**Independent Test**: Can be tested by deploying a charm that declares tmpfs or rootfs storage as a Deployment, verifying the pod has an EmptyDir volume mount, and confirming no PVCs are created.

**Acceptance Scenarios**:

1. **Given** a charm declaring storage with tmpfs provider type, **When** deployed as a Deployment, **Then** each pod gets its own EmptyDir volume with Memory medium and no PVCs are created in the namespace.
2. **Given** a charm declaring storage with rootfs provider type, **When** deployed as a DaemonSet, **Then** each pod gets its own EmptyDir volume backed by node disk and no PVCs are created.
3. **Given** a Deployment with ephemeral storage at scale=3, **When** a pod is replaced, **Then** the replacement pod gets a fresh empty volume (no stale data) and the unit re-registers without storage attachment errors.
4. **Given** a charm declaring both persistent and ephemeral storage, **When** deployed as a Deployment, **Then** persistent storage uses a shared PVC and ephemeral storage uses per-pod EmptyDir volumes.

---

### Edge Cases

- What happens when a charm declares persistent storage but the operator deploys with `deployment-type=stateless`? The system should allow this (the operator may intend to use ephemeral storage or shared volumes) but issue a warning that persistent storage may not behave as expected with a stateless workload type.
- What happens when a DaemonSet application charm tries to use storage with non-shared access mode? The system should issue a non-blocking warning (consistent with FR-016), as the operator explicitly chose the workload type override. The deployment proceeds but warns that ReadWriteOnce storage may cause Multi-Attach errors on multi-node clusters.
- What happens when an operator tries to change the workload type of an already-running application? The system should reject this with a clear error, as changing workload types requires a full redeployment (this is a destructive operation).
- What happens when constraints are set at the model level for deployment-type? The system should apply the model-level constraint as a default for new deployments within that model, overridable per-application.
- What happens during a controller upgrade when existing applications have no workload type recorded? The system should default existing applications to StatefulSet to preserve current behavior.
- What happens when a CAAS model is migrated between controllers? The deployment type must survive the migration. For inferred types (no explicit constraint), the import side must re-infer from charm metadata. For explicit constraints (including `daemon`), the serialization layer (`description/v11`) must carry the `deployment-type` field through the export/import round-trip.
- What happens when a Deployment with shared persistent storage scales down from 3 to 1 and back to 3? The shared PVC should remain intact across scaling operations. The surviving pod keeps its mount, new pods mount the same PVC. No PVC deletion or recreation should occur on scale changes.
- What happens when a Deployment pod with persistent storage is deleted and the replacement pod starts before the worker clears stale storage attachments? The registration should fail with a retryable error (duplicate key or filesystem attachment conflict). After the worker clears stale entries on its next cycle, the retry succeeds. The system should not create duplicate storage instances.
- What happens when a DaemonSet with ephemeral storage loses a node? Pods on the removed node are naturally terminated by K8s. The stale unit's cloud container and storage attachments should be cleaned up by the worker. If the node returns, the DaemonSet schedules a new pod with fresh ephemeral storage.
- What happens when the operator specifies a ReadWriteOnce storage class explicitly for a Deployment at scale=1, then later scales to 3? The deploy-time warning (FR-016) already informed the operator about the access mode incompatibility. At scale-up time, the system does not re-validate (per assumption: "Storage access mode validation is a deploy-time check, not a runtime check"). The first pod continues running; additional pods will fail to mount with a Multi-Attach error from Kubernetes. Scale-time re-validation is deferred to future work.
- What happens when a charm declares multiple storage instances (e.g., `data` and `cache`), one persistent and one ephemeral, and is deployed as a Deployment? Each storage type should be handled independently: persistent gets a shared PVC, ephemeral gets per-pod EmptyDir. The two storage types should not interfere with each other during pod replacement or scaling.

### Known Bugs (Pre-existing, Not Introduced by This Feature)

#### BUG: `remove-application` Race Leaves Orphaned K8s Resources

**Severity**: High — affects ALL CAAS workload types (StatefulSet, Deployment, DaemonSet).

**Summary**: When `juju remove-application` is called, the removal service and the caasapplicationprovisioner worker race against each other. The removal service deletes the application's DB record before the worker reaches the `appDead()` path where `app.Delete()` cleans up Kubernetes resources. This leaves orphaned K8s Deployments, Services, Roles, RoleBindings, Secrets, and ServiceAccounts in the model namespace.

**Reproduction sequence** (observed with nginx Deployment on `001-k8s-deployment-types` branch):
```
1. juju remove-application nginx
2. Removal service: "cannot delete, still has 1 unit" → waits
3. appWorker: detects Dying → calls ensureScale(0) → units removed
4. Removal service: sees 0 units → deletes DB record
5. appWorker: ensureScale() tries to update scaling state → "application not found" → CRASH
6. appWorker: restarts, finds app gone → exits cleanly
7. appDead() / app.Delete() was NEVER called → K8s resources orphaned
```

**Root cause**: The lifecycle assumes Dying → Dead → DB removal, with K8s cleanup in the Dead phase. But the removal service deletes the DB record as soon as units reach 0, which races with the worker's Dying phase. The existing TODO in `ops.go:443-447` acknowledges this:
```go
// TODO(k8s): re-implement this to prevent a dead app from going away through
// creating a new domain concept that holds the application until this worker
// has destroyed all the k8s resources.
```

**Secondary issue**: The `Delete()` method in `internal/provider/kubernetes/application/application.go` uses a label selector for `app.juju.is/created-by` (a label added by the mutating admission webhook). But when the controller's service account creates resources, the webhook's RBAC mapper doesn't map it to the app name (`FailurePolicy: Ignore`), so the label is silently not added. Even if `Delete()` were called, it would find zero matching resources. The only resources that would be cleaned up are the 3 direct-by-name deletions (ClusterRoleBinding, ClusterRole, ServiceAccount at lines 1005-1007).

**Files involved**:
- `internal/worker/caasapplicationprovisioner/ops.go` — `appDying()` (line 401), `appDead()` (line 422)
- `domain/removal/service/service.go` — removal job (line ~204)
- `internal/provider/kubernetes/application/application.go` — `Delete()` (line ~1009)
- `internal/worker/caasadmission/handler.go` — webhook label patching (line 133)

**Suggested fix direction**: Either (a) introduce a "has-resources" flag that prevents DB deletion until the worker confirms K8s cleanup is complete, or (b) make the `appDying()` path tolerant of "not found" errors so it can fall through to K8s cleanup even when the DB record is already gone.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST support three workload types for Kubernetes applications: stateless (Deployment), stateful (StatefulSet), and daemon (DaemonSet).
- **FR-002**: System MUST allow operators to specify workload type via a `deployment-type` constraint with values `stateless`, `stateful`, or `daemon`.
- **FR-003**: System MUST automatically infer the workload type when not explicitly specified: any charm that declares persistent storage in its metadata defaults to stateful (StatefulSet); only charms with no storage declarations default to stateless (Deployment).
- **FR-004**: System MUST continue to deploy all existing charms as StatefulSets when no workload type is specified and the charm has persistent storage, preserving backward compatibility.
- **FR-005**: System MUST reject manual scaling operations (add-unit, remove-unit, scale) for DaemonSet applications with a clear error message explaining that DaemonSet scaling is determined by the number of cluster nodes.
- **FR-006**: System MUST reject attempts to change the workload type of a running application, returning an error indicating that redeployment is required.
- **FR-007**: System MUST display the workload type of each application in the status output for Kubernetes models, both as a column in the applications summary table and in per-application detail output.
- **FR-008**: System MUST validate the `deployment-type` constraint value at deploy time and return a clear error for invalid values.
- **FR-009**: System MUST silently ignore the `deployment-type` constraint when used with non-Kubernetes (IAAS) models, consistent with how other Kubernetes-specific constraints behave.
- **FR-010**: System MUST persist the workload type for each application so that it survives controller restarts and upgrades.
- **FR-011**: System MUST default existing applications (deployed before this feature) to the StatefulSet workload type during upgrades, with no disruption to running workloads.
- **FR-012**: System MUST issue a warning when an operator deploys with `deployment-type=stateless` but the charm declares persistent storage requirements.
- **FR-013**: System MUST preserve the deployment type of CAAS applications during model migration. Inferred types must be re-derived from charm metadata on import. Explicitly constrained types (including `daemon`) require the `description` serialization library to support the `deployment-type` constraint field.
- **FR-014**: System MUST clear stale filesystem attachment and volume attachment records when clearing stale cloud container entries for replaced Deployment/DaemonSet pods, preventing duplicate key errors during re-registration.
- **FR-015**: System MUST delete all Juju-created standalone PersistentVolumeClaims when a Deployment or DaemonSet application is removed, preventing orphaned storage resources in the Kubernetes namespace.
- **FR-016**: System MUST warn (via both CLI output and controller logs) when the storage access mode is incompatible with the workload type at deploy time: Deployment with persistent storage at scale > 1 with ReadWriteOnce, or DaemonSet with persistent storage on a multi-node cluster with ReadWriteOnce. The warning is non-blocking — the deployment proceeds because the operator explicitly chose the workload type override.
- **FR-017**: System MUST support ephemeral storage (EmptyDir and tmpfs provider types) for Deployment and DaemonSet workloads, creating per-pod volumes that do not require PersistentVolumeClaims.
- **FR-018**: System MUST NOT create or delete PersistentVolumeClaims during Deployment or DaemonSet scale operations. The shared PVC remains stable regardless of replica count changes.

### Key Entities

- **Workload Type**: The kind of Kubernetes workload controller used for an application. One of: stateless (Deployment), stateful (StatefulSet), or daemon (DaemonSet). Determined at deploy time either automatically or by explicit operator constraint, and immutable for the lifetime of the application.
- **Deployment-Type Constraint**: A new constraint field that allows operators to explicitly select the workload type. Follows existing constraint semantics (can be set at model or application level, application-level overrides model-level).
- **Standalone PVC**: A PersistentVolumeClaim created directly by Juju (not via K8s VolumeClaimTemplates) for Deployment and DaemonSet workloads. Shared by all pods of the workload. Juju is responsible for its full lifecycle (creation and deletion).
- **VolumeClaimTemplate PVC**: A PersistentVolumeClaim created by Kubernetes as part of a StatefulSet's VolumeClaimTemplates spec. One PVC per replica, named `{template}-{pod-ordinal}`. K8s manages creation; retention policy governs deletion.
- **Ephemeral Storage**: Pod-local storage (EmptyDir or tmpfs) that exists only for the lifetime of a pod. Created and destroyed with the pod. Does not require PersistentVolumeClaims.
- **Storage Access Mode**: Kubernetes PVC access mode (ReadWriteOnce, ReadWriteMany, ReadOnlyMany). Determines whether a volume can be mounted by pods on one or multiple nodes simultaneously.

### Terminology

- **Workload type** is the user-facing term used in documentation, error messages, and status output (e.g., "Deployment", "StatefulSet", "DaemonSet").
- **Deployment type** is the implementation-level term used in code, constraint names (`deployment-type`), and database columns (`deployment_type_id`).
- Both terms refer to the same concept. Use "workload type" when addressing operators; use "deployment type" in code and technical artifacts.

### Assumptions

- The underlying Kubernetes provider already supports creating and managing Deployment, StatefulSet, and DaemonSet resources. However, `currentScale()` and `computeStatus()` only had full implementations for StatefulSet — Deployment/DaemonSet cases returned `NotSupported`. These were implemented as part of this feature (see tasks T044-T045).
- The automatic inference heuristic is simple: any charm declaring persistent storage in its metadata gets StatefulSet; charms with no storage declarations get Deployment. This is deliberately conservative to preserve backward compatibility. Operators who need different behavior can use the explicit constraint.
- Changing workload type on a running application is out of scope. This would require destroying and recreating the Kubernetes resources, which is a complex and potentially data-losing operation best handled by explicit redeployment.
- DaemonSet unit representation in status will use the same ordinal naming as other workload types for simplicity. Mapping pods to specific nodes is deferred to future work.
- StatefulSet pod names follow `<app>-<ordinal>` convention, so unit registration can derive the ordinal from the pod name. Deployment/DaemonSet pod names have random suffixes (e.g., `nginx-759b4f4b68-5mk8l`), so unit registration uses a 3-step strategy: (1) look up existing unit by provider ID (idempotent re-registration), (2) find an unassigned unit with no `k8s_pod` row (pre-created at deploy time), (3) allocate the next available ordinal. Additionally, the unit registration scaling gate is relaxed for non-StatefulSet deployments: StatefulSet requires `Scaling=true` AND `ordinal < ScaleTarget` (strict gate set by `EnsureScale`), while Deployment/DaemonSet counts alive units and only requires `aliveCount < appScale.Scale` since pods may start before `EnsureScale` runs.
- The `deployment-type` constraint is only meaningful for Kubernetes (CAAS) models and is silently ignored (no warning or error) for IAAS models, consistent with how other Kubernetes-specific constraints behave.
- Deployment and DaemonSet workloads with persistent storage share a single PVC across all pods (standalone PVC model). This is fundamentally different from StatefulSet, where each pod gets its own PVC via VolumeClaimTemplates. Operators who need per-pod persistent storage must use StatefulSet.
- The K8s provider already handles EmptyDir/tmpfs volumes via `VolumeSourceForFilesystem()` — these return a non-nil VolumeSource and bypass PVC creation entirely. This path should work for Deployment/DaemonSet today but requires validation.
- Storage access mode validation is a deploy-time check, not a runtime check. Once deployed, the system does not re-validate access modes on scale changes. Scale-time validation is limited to warning (not blocking) because the operator may have set up a ReadWriteMany-capable StorageClass after initial deployment.
- PVC cleanup on application removal depends on the `Delete()` method in the K8s provider being called. The pre-existing `remove-application` race condition (documented in Known Bugs) can prevent `Delete()` from running, in which case PVCs are orphaned. **The removal race fix is out of scope for these storage stories.** US8 adds PVC deletion to `Delete()` so cleanup works when the method is called. Reliable execution depends on the race being fixed separately.
- The `FilesystemProvisioningInfo()` facade method is currently a stub (returns empty). Full storage support for scaling operations (`EnsurePVCs`) depends on implementing this facade. **This facade fix is out of scope for the storage adaptation stories** — it is a separate prerequisite. Stories that depend on it (US9 scaling with storage, US11 scaling with ephemeral storage) note this dependency but do not include its implementation.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Operators can deploy stateless charms that run as Deployments, completing the deploy workflow in the same amount of time as current StatefulSet deployments (no performance regression).
- **SC-002**: Operators can deploy DaemonSet-based charms that automatically scale to all cluster nodes, with the instance count matching the node count within 60 seconds of a node joining.
- **SC-003**: 100% of existing charms deployed before this feature continue to operate as StatefulSets after a controller upgrade, with zero disruption.
- **SC-004**: Operators can determine the workload type of any Kubernetes application from the status output without needing to inspect the Kubernetes cluster directly.
- **SC-005**: Invalid workload type values are caught at deploy time with a clear, actionable error message, preventing any partial or failed deployments.
- **SC-006**: Manual scaling operations on DaemonSet applications are blocked with a clear error, preventing operator confusion about how DaemonSets scale.
- **SC-007**: Operators using the constraint override can deploy any charm as any supported workload type, with 100% of valid constraint values being accepted and applied correctly.
- **SC-008**: When a Deployment pod with persistent storage is replaced, the replacement pod re-registers and remounts storage within 30 seconds without duplicate key errors in the controller logs.
- **SC-009**: After removing a Deployment or DaemonSet application with persistent storage, zero Juju-created PVCs remain in the Kubernetes namespace within 60 seconds of removal completing.
- **SC-010**: Deploying a Deployment or DaemonSet with persistent ReadWriteOnce storage in an incompatible configuration produces a non-blocking warning in both CLI output and controller logs within 5 seconds of the deploy command.
- **SC-011**: Deployment and DaemonSet workloads with ephemeral storage (EmptyDir/tmpfs) deploy, scale, and recover from pod replacement without creating any PersistentVolumeClaims.
