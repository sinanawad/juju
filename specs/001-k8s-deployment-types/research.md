# Research: Kubernetes Deployment Type Support

**Date**: 2026-02-08
**Feature**: 001-k8s-deployment-types

## Decision 1: Constraint Mechanism for Deployment Type

**Decision**: Add `deployment-type` as a new constraint field in `core/constraints/constraints.go`.

**Rationale**: Constraints are already the mechanism for expressing deployment preferences that flow from CLI through API to workers. The constraints system:
- Already flows through `ProvisioningInfo` to the CAAS worker
- Supports model-level defaults with per-application overrides
- Is silently ignored for inapplicable model types (IAAS)
- Has established patterns for parsing, validation, and serialization

**Alternatives considered**:
- Charm metadata field: Rejected - requires Charmcraft changes, breaks existing charms
- Application config: Rejected - config is for operator-tunable runtime settings, not deployment topology
- New API parameter: Rejected - constraints already handle this class of requirement

**Key locations**:
- `core/constraints/constraints.go` - `Value` struct, parsing, validation
- `domain/constraints/constraints.go` - domain constraint mapping
- Constraints reach the provisioner via `ProvisioningInfo` assembled in `apiserver/facades/controller/caasapplicationprovisioner/provisioner.go:431-440`

## Decision 2: Provider Layer Already Complete

**Decision**: No changes needed to the K8s provider application layer. It already supports all three deployment types.

**Rationale**: `internal/provider/kubernetes/application/application.go:279-415` has a complete switch statement handling `DeploymentStateful`, `DeploymentStateless`, and `DeploymentDaemon` with correct storage, scaling, deletion, and watch behaviors. The gap is entirely in the layers above the provider.

**Evidence**:
- Lines 280-341: StatefulSet creation with VolumeClaimTemplates
- Lines 342-384: Deployment creation with standalone PVCs
- Lines 385-412: DaemonSet creation with standalone PVCs
- `scale.go:23-43`: Scale supports StatefulSet and Deployment; DaemonSet returns NotSupported

## Decision 3: Hardcoded Locations to Update

**Decision**: Replace hardcoded `caas.DeploymentStateful` at 4 production code locations (plus bootstrap).

**Locations**:
1. `internal/worker/caasapplicationprovisioner/application.go:149` - PRIMARY: worker creates broker app
2. `internal/worker/caasfirewaller/appfirewaller.go:81` - firewaller creates broker app
3. `domain/application/service/provider.go:442` - domain provider service
4. `domain/application/service/provider.go:538` - domain provider service (second call)
5. `internal/provider/kubernetes/bootstrap.go:1338` - bootstrap (keep as StatefulSet)

**Rationale**: Each location currently ignores deployment type. After this feature, they must read it from the application's persisted state or constraints.

## Decision 4: Database Schema Change

**Decision**: Add a `deployment_type` column to the `application` table via a PATCH SQL file, with a lookup table for valid values.

**Rationale**: Follows existing patterns:
- PATCH files in `domain/schema/model/sql/` (naming: `NNNN-name.PATCH.sql`)
- Registration in `domain/schema/model.go:128-150` via `modelPostPatchFilesByVersion`
- Application table at `domain/schema/model/sql/0019-application.sql`
- Lookup tables used for constrained values (e.g., `life`, `workload_status_value`)

**Schema**:
```sql
CREATE TABLE deployment_type (
    id INT PRIMARY KEY,
    name TEXT NOT NULL
);
INSERT INTO deployment_type VALUES (0, 'stateful'), (1, 'stateless'), (2, 'daemon');

ALTER TABLE application ADD COLUMN deployment_type_id INT NOT NULL DEFAULT 0
    REFERENCES deployment_type(id);
```

Default 0 (stateful) ensures existing applications get StatefulSet on upgrade.

## Decision 5: Status Display Changes

**Decision**: Add deployment type to both the applications summary table and per-application detail for CAAS models.

**Key locations**:
- `cmd/juju/status/output_tabular.go:141-145` - CAAS column headers (add "Type" column)
- `cmd/juju/status/formatted.go:120-142` - `applicationStatus` struct (add `DeploymentType` field)
- `rpc/params/status.go:143-167` - `ApplicationStatus` struct (add CAAS field)
- `apiserver/facades/client/client/status.go:940-956` - CAAS-specific assembly
- `domain/status/service/types.go` - Domain status Application struct

## Decision 6: API Facade Versioning

**Decision**: Bump CAASApplicationProvisioner facade from v1 to v2 (add deployment type to ProvisioningInfo). No bump needed for Application facade since deployment type flows through existing constraint parameter.

**Rationale**:
- Application facade (v22): Constraints are already a parameter of Deploy; no new API method needed
- CAASApplicationProvisioner (v1 → v2): `params.CAASApplicationProvisioningInfo` needs a new `DeploymentType` field
- Client facade version list in `api/facadeversions.go:23-122` must be updated

## Decision 7: Inference Heuristic Location

**Decision**: Implement `DetermineDeploymentType()` in the CAAS application provisioner worker's ops package.

**Rationale**: The worker already has access to charm metadata (via ProvisioningInfo) and application constraints. The heuristic:
1. If `deployment-type` constraint is set → use that value
2. If charm declares any storage → `DeploymentStateful`
3. Otherwise → `DeploymentStateless`

This keeps the heuristic close to where it's consumed and avoids leaking it into the domain layer.

---

## Storage Adaptation Research (2026-02-10)

### Decision 8: Storage Attachment Cleanup Strategy

**Decision**: Extend `ClearCAASUnitCloudContainer()` in `domain/application/state/unit.go` to cascade-delete `storage_filesystem_attachment` and `storage_volume_attachment` rows within the same transaction.

**Rationale**: The function already runs within a `db.Txn()` block and deletes `k8s_pod`/`k8s_pod_port` rows keyed by the unit's `net_node_uuid`. Storage attachment tables (`storage_filesystem_attachment`, `storage_volume_attachment`) are also keyed by `net_node_uuid`, making the cascade a natural extension. All DELETE statements are idempotent (no error if rows don't exist).

**Alternatives considered**:
- UPSERT in `setFilesystemProviderIDs`/`setFilesystemAttachmentProviderIDs`: Rejected — these functions UPDATE existing records (not INSERT), so the duplicate key comes from the attachment table, not the provider ID columns. UPSERT doesn't address the root cause.
- Separate `ClearCAASUnitStorage()` method: Rejected — the cleanup must be atomic with `k8s_pod` cleanup to prevent race windows. A single transaction is simpler and safer.

**Key tables**:
- `storage_filesystem_attachment` (schema 0011-storage.sql:418) — keyed by `net_node_uuid`
- `storage_volume_attachment` (schema 0011-storage.sql:304) — keyed by `net_node_uuid`

### Decision 9: PVC Cleanup in Delete()

**Decision**: Add PVC listing and deletion to `Delete()` in the K8s provider for non-StatefulSet workloads, using the existing `resourceLabels` selector pattern.

**Rationale**: The `Delete()` method already lists and deletes 14 resource types (StatefulSets, Services, Secrets, ConfigMaps, Roles, etc.) using the same label selector pattern. PVCs carry the same Juju labels (`app.kubernetes.io/managed-by=juju`, `app.kubernetes.io/name={appName}`). Adding PVCs follows the identical pattern — `resources.ListPersistentVolumeClaims()` is already available in the codebase.

**Guard**: Only delete PVCs for non-StatefulSet applications. StatefulSet PVCs are managed by K8s VolumeClaimTemplates retention policy (`persistentVolumeClaimRetentionPolicy`).

**Alternatives considered**:
- Delete PVCs for all workload types: Rejected — would change StatefulSet behavior and break backward compatibility.
- Use `pvcNames()` regex matching: Rejected — `pvcNames()` is a complex function for PVC reuse during scaling. For deletion, the simpler label selector approach matches all other resource cleanup.

**Key locations**:
- `Delete()`: `internal/provider/kubernetes/application/application.go:986-1215`
- PVC labels: `utils.LabelsForStorage()` in `internal/provider/kubernetes/utils/labels.go`
- PVC resource helper: `resources.ListPersistentVolumeClaims()`

### Decision 10: Access Mode Validation Approach

**Decision**: Emit a non-blocking warning at deploy time when storage access modes are incompatible with the workload type. No blocking errors.

**Rationale**: Per clarification session Q2, the operator explicitly overrode inference by setting `deployment-type=stateless` or `deployment-type=daemon`. The system should warn but not block, since:
1. The operator may have a ReadWriteMany-capable storage class configured
2. K8s `StorageClass` objects do not expose supported access modes in their spec — the system cannot definitively determine incompatibility
3. Single-replica Deployments and single-node DaemonSets work fine with RWO

**Validation location**: `internal/worker/caasapplicationprovisioner/ops.go` (alongside existing stateless+storage warning at line 260). Access mode is available via `ParseStorageMode()` in `storage/volumes.go:127`.

**Limitation**: The system checks the access mode from the Juju storage pool config, not from the K8s StorageClass. If the operator doesn't set an explicit access mode, the default is ReadWriteOnce. The warning fires based on this configured value.

### Decision 11: Ephemeral Storage Requires No New Code

**Decision**: The existing `VolumeSourceForFilesystem()` code path already handles EmptyDir/tmpfs for all deployment types. Phase 12 is primarily validation and test.

**Rationale**: `VolumeSourceForFilesystem()` in `storage/storage.go:57-86` returns a non-nil `VolumeSource` for `rootfs` (EmptyDir) and `tmpfs` (EmptyDir with Memory medium). When the VolumeSource is non-nil, `filesystemToVolumeInfo()` (application.go:2302) bypasses PVC creation entirely. This code path is deployment-type-agnostic — it works the same for StatefulSet, Deployment, and DaemonSet.

**What needs validation**:
- Pod replacement with ephemeral storage creates a fresh volume (expected K8s behavior)
- No storage attachment records are created for ephemeral volumes (needs verification)
- Mixed persistent+ephemeral storage works (persistent gets PVC, ephemeral gets EmptyDir)

### Decision 12: PVC Stability During Scaling

**Decision**: Verify (not implement) that scale operations don't create or delete PVCs for non-StatefulSet workloads.

**Rationale**: The `Ensure()` method's Deployment path (application.go:342-384) calls `handlePVCForStatelessResource` only when the PVC doesn't exist. The function uses `resources.NewPersistentVolumeClaim` with `Apply()` semantics (create-or-update), so re-running on an existing PVC is a no-op. Scale changes only modify the Deployment's `replicas` field. Phase 13 confirms this behavior with tests.
