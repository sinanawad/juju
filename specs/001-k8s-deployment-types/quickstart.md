# Quickstart: K8s Deployment Type Support

## Prerequisites

- Go development environment (see `go.mod` for version)
- DQLite build dependencies for `make install`
- A K8s cluster (microk8s works) for integration testing

## Build

```bash
make install    # Full build with schema rebuild
make go-build   # Quick build without schema
```

## Test

```bash
# Core constraints
go test ./core/constraints/...

# Domain layer
go test ./domain/application/...
go test ./domain/constraints/...

# Worker
go test ./internal/worker/caasapplicationprovisioner/...
go test ./internal/worker/caasfirewaller/...

# Facade
go test ./apiserver/facades/controller/caasapplicationprovisioner/...
go test ./apiserver/facades/client/client/...

# Status display
go test ./cmd/juju/status/...

# All tests
make test
```

## Lint

```bash
make pre-check
```

## Manual Verification

```bash
# Bootstrap a K8s controller
juju bootstrap microk8s test-ctrl

# Add a K8s model
juju add-model test-model

# Deploy with automatic inference (no storage → Deployment)
juju deploy nginx

# Deploy with explicit constraint
juju deploy mysql --constraints="deployment-type=stateful"

# Deploy as DaemonSet
juju deploy node-exporter --constraints="deployment-type=daemon"

# Verify in status
juju status
# Should show "Type" column with Deployment/StatefulSet/DaemonSet
```

## Key Files to Modify (by story)

### Story 1+2 (Foundation + Constraints)
- `core/constraints/constraints.go`
- `core/constraints/constraints_test.go`
- `domain/constraints/constraints.go`

### Story 1+4 (Inference + Backward Compat)
- `domain/schema/model/sql/NNNN-deployment-type.PATCH.sql` (NEW)
- `domain/schema/model.go`
- `domain/application/types.go`
- `domain/application/state/application.go`
- `domain/application/service/application.go`

### Story 1 (Worker Wiring)
- `internal/worker/caasapplicationprovisioner/application.go`
- `internal/worker/caasapplicationprovisioner/ops.go`
- `internal/worker/caasfirewaller/appfirewaller.go`
- `domain/application/service/provider.go`

### Story 3 (DaemonSet)
- `domain/application/service/application.go` (scale validation)
- `domain/application/errors/errors.go`

### Story 5 (Status)
- `rpc/params/status.go`
- `cmd/juju/status/formatted.go`
- `cmd/juju/status/output_tabular.go`
- `cmd/juju/status/formatter.go`
- `apiserver/facades/client/client/status.go`
- `domain/status/service/types.go`

### API Versioning
- `apiserver/facades/controller/caasapplicationprovisioner/register.go`
- `apiserver/facades/controller/caasapplicationprovisioner/provisioner.go`
- `api/facadeversions.go`
- `api/controller/caasapplicationprovisioner/client.go`
