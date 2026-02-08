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
