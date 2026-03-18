# Core Types and Primitives

## Overview

The `core/` package contains 69 subdirectories of primitive types with minimal dependencies. These are used throughout Juju and form the shared vocabulary.

## Identity Types

String wrappers with validation for type safety:

### Unit
```go
// core/unit/
type UUID string   // e.g., "550e8400-e29b-41d4-a716-446655440000"
type Name string   // e.g., "mysql/0", "postgresql/1"

func (n Name) Validate() error  // Checks format "application/number"
```

### Machine
```go
// core/machine/
type UUID string
type Name string   // e.g., "0", "0/lxd/1", "1/kvm/0/lxd/2"

// Supports container hierarchy
func (n Name) Validate() error  // Checks hierarchical format
```

### Application
```go
// core/application/
type UUID string
```

### Model
```go
// core/model/
type UUID string
type Qualifier struct { ... }  // Model qualifier
type ModelType string          // "iaas" or "caas"
```

### User
```go
// core/user/
type UUID string
type Name string  // Supports local and domain users

func (n Name) IsLocal() bool
func (n Name) Domain() string
```

### Storage
```go
// core/storage/
type ID string    // e.g., "data/0"
type Name string  // e.g., "data"
```

## Status Types

```go
// core/status/
type Status string

// Common statuses
const (
    Error      Status = "error"
    Active     Status = "active"
    Idle       Status = "idle"
    Pending    Status = "pending"
    Blocked    Status = "blocked"
    Waiting    Status = "waiting"
    Maintenance Status = "maintenance"
    Terminated Status = "terminated"
)

type StatusInfo struct {
    Status  Status
    Message string
    Data    map[string]interface{}
    Since   *time.Time
}

// Predicates
func KnownAgentStatus(s Status) bool
func KnownWorkloadStatus(s Status) bool
func ValidWorkloadStatus(s Status) bool
```

## Lifecycle

```go
// core/life/
type Value string

const (
    Alive Value = "alive"
    Dying Value = "dying"
    Dead  Value = "dead"
)

func (v Value) Validate() error
```

## Network Types

```go
// core/network/

// Address interface
type Address interface {
    Host() string
    AddressType() AddressType
    AddressScope() Scope
    AddressCIDR() string
    AddressConfigType() AddressConfigType
    AddressIsSecondary() bool
}

// Scope levels
type Scope string
const (
    ScopePublic      Scope = "public"
    ScopeCloudLocal  Scope = "local-cloud"
    ScopeFanLocal    Scope = "local-fan"
    ScopeMachineLocal Scope = "local-machine"
    ScopeUnknown     Scope = "unknown"
)

// Concrete types
type MachineAddress struct { ... }
type ProviderAddress struct { ... }  // Adds space, provider metadata
type SpaceAddress struct { ... }     // Adds space ID
```

## Constraints

```go
// core/constraints/
type Value struct {
    Arch             *string
    Container        *instance.ContainerType
    CpuCores         *uint64
    CpuPower         *uint64
    Mem              *uint64  // MB
    RootDisk         *uint64  // MB
    RootDiskSource   *string
    Tags             *[]string
    InstanceRole     *string
    InstanceType     *string
    Spaces           *[]string  // "space" or "^space" (exclude)
    VirtType         *string
    Zones            *[]string
    AllocatePublicIP *bool
    ImageID          *string
}

func (v Value) Validate() error
func Parse(args ...string) (Value, error)
```

## Permissions

```go
// core/permission/

type Access string

// Model permissions (hierarchical)
const (
    NoAccess    Access = ""
    ReadAccess  Access = "read"
    WriteAccess Access = "write"
    AdminAccess Access = "admin"
)

// Controller permissions
const (
    LoginAccess     Access = "login"
    SuperuserAccess Access = "superuser"
)

// Offer permissions
const (
    ConsumeAccess Access = "consume"
)

// Cloud permissions
const (
    AddModelAccess Access = "add-model"
)

type ObjectType string
const (
    Cloud      ObjectType = "cloud"
    Controller ObjectType = "controller"
    Model      ObjectType = "model"
    Offer      ObjectType = "offer"
)
```

## Instance

```go
// core/instance/
type Id string  // Provider-specific instance ID

type Status string
const (
    StatusEmpty        Status = ""
    StatusAllocating   Status = "allocating"
    StatusRunning      Status = "running"
    StatusProvisioning Status = "provisioning"
    StatusError        Status = "error"
)
```

## Charm

```go
// core/charm/

type ID string

type Origin struct {
    Source   OriginSource
    ID       string
    Hash     string
    Revision int
    Channel  *charm.Channel
    Platform Platform
}

type Platform struct {
    Architecture string
    OS           string
    Channel      string
}

// Repository interface
type Repository interface {
    GetDownloadURL(ctx, url, origin) (*url.URL, Origin, error)
    Download(ctx, name, origin, path) (Origin, *Digest, error)
    ResolveWithPreferredChannel(ctx, url, origin) (ResolvedData, error)
    ListResources(ctx, url, origin) ([]Resource, error)
}
```

## Watcher Interface

```go
// core/watcher/
type Watcher[T any] interface {
    worker.Worker
    Changes() <-chan T
}

// Common watcher types
type NotifyWatcher = Watcher[struct{}]
type StringsWatcher = Watcher[[]string]
```

## Secrets

```go
// core/secrets/
type SecretData map[string]string

func (d SecretData) Validate() error  // Validates keys, base64 values
```

## Version

```go
// core/version/
var Current = semversion.MustParse("4.0.2")

func GitCommit() string
func GitTreeState() string
```

## Key Interfaces

### StatusSetter/Getter
```go
type StatusSetter interface {
    SetStatus(StatusInfo) error
}

type StatusGetter interface {
    Status() (StatusInfo, error)
}
```

### Validation Pattern

All identity types follow this pattern:
```go
func (t Type) Validate() error {
    if !validRegex.MatchString(string(t)) {
        return errors.NotValidf("type %q", t)
    }
    return nil
}
```

## Error Types

```go
// core/errors/
// Wraps github.com/juju/errors with constants

// Type checking
errors.Is(err, coreerrors.NotFound)
errors.Is(err, coreerrors.NotValid)
errors.Is(err, coreerrors.Unauthorized)
errors.Is(err, coreerrors.AlreadyExists)
```

## Design Principles

1. **Minimal dependencies** - Core types import almost nothing from Juju
2. **Validation built-in** - All types have `Validate()` methods
3. **Immutable by design** - Small, copyable values
4. **Type safety** - String wrappers prevent mixing identities
5. **No persistence logic** - Pure in-memory types
6. **Interface definitions** - Contracts for major subsystems
