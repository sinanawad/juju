# Contract: CAASApplicationProvisioningInfo

**Change**: Add `DeploymentType` field to provisioning info returned by CAASApplicationProvisioner facade.

## Current (v1)

```go
type CAASApplicationProvisioningInfo struct {
    ImageRepo            DockerImageInfo
    Version              semversion.Number
    APIAddresses         []string
    CACert               string
    Tags                 map[string]string
    Constraints          constraints.Value
    Devices              []KubernetesDeviceParams
    Base                 Base
    CharmModifiedVersion int
    Scale                int
    Trust                bool
    Error                *Error
}
```

## Proposed (v2)

```go
type CAASApplicationProvisioningInfo struct {
    ImageRepo            DockerImageInfo
    Version              semversion.Number
    APIAddresses         []string
    CACert               string
    Tags                 map[string]string
    Constraints          constraints.Value
    DeploymentType       string             // NEW: "stateless", "stateful", "daemon"
    Devices              []KubernetesDeviceParams
    Base                 Base
    CharmModifiedVersion int
    Scale                int
    Trust                bool
    Error                *Error
}
```

## Backward Compatibility

- Facade version bumped from 1 → 2
- v1 clients continue to work (field is additive, zero value defaults to stateful behavior)
- Worker must negotiate for v2 to receive deployment type
