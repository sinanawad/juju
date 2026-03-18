# K8s Deployment Type Support - Implementation Status

**Status**: Phases 1-3 complete (MVP), Phase 6 partial (migration fixes), Phases 4-5 pending
**Branch**: `001-k8s-deployment-types`
**Spec**: `/data/dev/juju/specs/001-k8s-deployment-types/spec.md`

## Completed Work

### MVP (Phases 1-3, T001-T020)
- Schema: `deployment_type` lookup table + column on `application` (PATCH 0046)
- Core constraint: `deployment-type` field with validation (stateless|stateful|daemon)
- Domain persistence and retrieval via `GetApplicationDeploymentType`
- API: facade v2, `DeploymentType` in `CAASApplicationProvisioningInfo`
- Worker: `DetermineDeploymentType()` replaces hardcoded StatefulSet
- Provider service: dynamic deployment type in both provider.go locations
- Immutability enforcement (FR-006)
- Storage mismatch warning (FR-012)

### Bug Fixes (Phase 6 partial, T036-T039)
- YAML constraint validation bypass fixed (routed through `setDeploymentType()`)
- Migration: `InsertApplicationArgs.DeploymentType` threaded to state layer
- Migration: Re-inference from charm metadata for inferred types

## Remaining Work

### Phase 4: DaemonSet Scale Blocking (T021-T022)
- `DaemonSetScaleNotSupported` error constant
- Check in `SetApplicationScale()` / `ChangeApplicationScale()`

### Phase 5: Status Visibility (T023-T029)
- Wire types, facade, formatter, tabular output

### Phase 6: Migration External Dep (T040-T043) — RELEASE BLOCKER
- `description/v11` lacks `DeploymentType` in `ConstraintsArgs`/`Constraints`
- **Without this**: daemon apps and explicit constraint overrides lost on migration
- **With re-inference**: inferred stateless/stateful types survive
- See plan.md "Pre-Merge Dependencies" section

### Phase 7: Polish (T030-T035)
- Full test suite runs + build verification

## Lifecycle Analysis (2026-02-08)

Full lifecycle audit confirmed deployment type correctness across all phases:

| Phase | Status | Notes |
|-------|--------|-------|
| **Deploy** | Correct | Full flow: constraints → persistence → facade → worker → K8s provider |
| **Charm Upgrade** | Correct | `SetApplicationCharm` doesn't touch `deployment_type_id` (immutable by design) |
| **Scale** | Correct at provider | Broker rejects DaemonSet scale; service-level guard (T022) still needed |
| **Destroy** | Correct | `Delete()` switches on type for correct K8s cleanup |
| **Restart** | Correct | Worker re-reads deployment type from state on every restart |
| **Config/Trust** | Correct | `EnsureTrust` doesn't re-provision; deployment type preserved |
| **Expose** | Correct | Firewaller fetches type fresh each time |
| **Relations** | N/A | Don't trigger K8s provisioning |
| **Migration** | Partial | Inferred types preserved; explicit constraints need description lib fix |

### Key Edge Case
Charm upgrade that adds storage to a previously storage-less charm does NOT change deployment type (immutability). App stays stateless, FR-012 warning fires. Operator must redeploy to change workload type.

### Worker Architecture Note
The broker `app` object is created ONCE at worker startup with deployment type baked in. Since deployment type is immutable (FR-006), this is safe. On worker restart, type is re-read from state.

## K8s Substrate Analysis (2026-02-08)

Analyzed `internal/provider/kubernetes/application/` to verify K8s-level correctness.

### Correct Behavior (no changes needed)
- **Ensure()**: Creates correct resource type (StatefulSet/Deployment/DaemonSet) based on deployment type
- **Delete()**: Type-specific deletion + bulk label-based cleanup catches all resource types
- **Scale()**: Supports StatefulSet/Deployment via `Spec.Replicas`; returns `NotSupported` for DaemonSet (correct)
- **State()**: DaemonSet uses `Status.DesiredNumberScheduled`; others use `Spec.Replicas`
- **Watch()**: Type-specific informer + Service watcher
- **Units()**: Lists pods by label selector; marks `Stateful: true` only for StatefulSet
- **Storage**: StatefulSet uses VolumeClaimTemplates; Deployment/DaemonSet use standalone PVCs via `handlePVCForStatelessResource`
- **Resource applier**: Strategic merge patch with create-or-update pattern, conflict retry (5 attempts)

### Known Gaps (pre-existing, not introduced by this feature)
| Gap | Severity | Details |
|-----|----------|---------|
| `computeStatus()` incomplete | Medium | Only fully implemented for StatefulSet; returns `NotSupported` for Deployment/DaemonSet |
| `Exists()` single-type check | Low | Only checks the resource matching stored deployment type; won't detect stray resources of other types with same name |
| No drift detection | Low | Manual `kubectl` edits to deployment type aren't detected by Juju |
| No cross-type validation | Low | If wrong resource type exists with same name, `Exists()` returns false rather than error |

### Assessment
These gaps are **pre-existing** in the K8s provider and affect all deployment types equally. The `computeStatus()` gap for Deployment/DaemonSet is the most operationally significant — it means `juju status` may show less detailed status information for non-StatefulSet workloads. This is a separate improvement, not a blocker for this feature.

## Key Code Locations

| Purpose | Path |
|---------|------|
| Constraint parsing | `core/constraints/constraints.go` |
| Domain types | `domain/application/types.go` |
| Persistence | `domain/application/state/application.go` |
| Service | `domain/application/service/application.go`, `provider.go` |
| Migration | `domain/application/service/migration.go`, `state/migration.go` |
| Facade | `apiserver/facades/controller/caasapplicationprovisioner/provisioner.go` |
| Worker | `internal/worker/caasapplicationprovisioner/application.go`, `ops.go` |
| Firewaller | `internal/worker/caasfirewaller/appfirewaller.go` |
| K8s provider | `internal/provider/kubernetes/application/application.go` |
| Schema PATCH | `domain/schema/model/sql/0046-deployment-type.PATCH.sql` |
