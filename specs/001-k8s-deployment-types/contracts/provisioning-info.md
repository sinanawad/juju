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

- Facade registers only v2 (no v1 → v2 negotiation needed)
- v1 registration is dropped (controller-internal facade; client and server ship in the same binary)
- `DeploymentType` defaults to the zero value (stateful behavior) if unset
