# Contract: Status Display Changes

**Change**: Add deployment type to `juju status` output for CAAS models.

## Applications Summary Table (CAAS)

### Current
```
App       Version  Status  Scale  Charm   Channel  Rev  Address      Exposed  Message
nginx     1.19     active  3      nginx   stable   42   10.1.2.3     no
mysql     8.0      active  1      mysql   stable   15   10.1.2.4     no
```

### Proposed
```
App       Version  Status  Scale  Charm   Channel  Rev  Address      Exposed  Type         Message
nginx     1.19     active  3      nginx   stable   42   10.1.2.3     no       Deployment
mysql     8.0      active  1      mysql   stable   15   10.1.2.4     no       StatefulSet
monitor   0.1      active  3/3    monitor stable   7    10.1.2.5     no       DaemonSet
```

## Per-Application Detail

The deployment type is also included in per-application status detail output, available via `juju status --format=yaml` and `juju status --format=json`.

## Wire Type Change

`rpc/params.ApplicationStatus` gains:
```go
DeploymentType string `json:"deployment-type,omitempty"`
```

## IAAS Models

No change. The "Type" column is not displayed for IAAS models, consistent with how "Address" is already CAAS-only.
