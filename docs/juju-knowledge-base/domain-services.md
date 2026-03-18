# Domain Services Architecture

## Overview

The `domain/` package implements business logic with strict separation:
- **Services**: Business logic, validation, orchestration
- **States**: Data persistence abstraction (Sqlair ORM)

Services depend on State *interfaces*, enabling testing and flexibility.

## Directory Structure

```
domain/{name}/
├── service/
│   ├── service.go      # Core Service struct and constructor
│   ├── {entity}.go     # Entity-specific operations
│   ├── watcher.go      # WatchableService (optional)
│   └── provider.go     # ProviderService (optional)
├── state/
│   ├── state.go        # State struct implementing interfaces
│   ├── types.go        # State-specific types
│   └── {entity}.go     # Entity-specific queries
├── types.go            # Domain types
├── errors/             # Domain-specific errors
└── modelmigration/     # Migration support (optional)
```

## Service Pattern

```go
// domain/application/service/service.go
type Service struct {
    st            State              // State interface (abstraction)
    leaderEnsurer leadership.Ensurer // External dependency
    modelUUID     model.UUID
    logger        logger.Logger
    clock         clock.Clock
    charmStore    CharmStore
    statusHistory StatusHistory
}

func NewService(
    st State,
    leaderEnsurer leadership.Ensurer,
    charmStore CharmStore,
    statusHistory StatusHistory,
    modelUUID model.UUID,
    clock clock.Clock,
    logger logger.Logger,
) *Service {
    return &Service{
        st:            st,
        leaderEnsurer: leaderEnsurer,
        modelUUID:     modelUUID,
        logger:        logger,
        clock:         clock,
        charmStore:    charmStore,
        statusHistory: statusHistory,
    }
}
```

## State Interface Pattern

Services define the State interface they need:

```go
// domain/application/service/application.go
type ApplicationState interface {
    GetApplicationName(context.Context, coreapplication.UUID) (string, error)
    GetApplicationUUIDByName(ctx context.Context, name string) (coreapplication.UUID, error)
    CreateIAASApplication(context.Context, string, application.AddIAASApplicationArg, []application.AddIAASUnitArg) (coreapplication.UUID, []coremachine.Name, error)
    // ... more methods
}
```

## State Implementation

```go
// domain/application/state/state.go
type State struct {
    *domain.StateBase  // Provides DB access, statement caching
    modelUUID model.UUID
    clock     clock.Clock
    logger    logger.Logger
}

func NewState(
    factory database.TxnRunnerFactory,
    modelUUID model.UUID,
    clock clock.Clock,
    logger logger.Logger,
) *State {
    return &State{
        StateBase: domain.NewStateBase(factory),
        modelUUID: modelUUID,
        clock:     clock,
        logger:    logger,
    }
}
```

## StateBase Utilities

All states inherit from `domain.StateBase`:

```go
// domain/state.go
type StateBase struct {
    getDB      database.TxnRunnerFactory
    statements map[string]*sqlair.Statement  // Cached statements
}

// Key methods:
db, err := s.DB(ctx)                           // Get transaction runner
stmt, err := s.Prepare(queryString, types...)  // Cache SQL statement
err := s.RunAtomic(ctx, func(atomic AtomicContext) error { ... })  // Transaction
```

## Service Hierarchy

Services compose into hierarchies:

```
WatchableService
    └── ProviderService (optional, for cloud operations)
        └── Service
            └── Embeds: State interface
                       statusHistory StatusHistory
                       clock clock.Clock
                       logger logger.Logger
```

### WatchableService

Adds change notification capability:

```go
type WatchableService struct {
    Service
    watcherFactory WatcherFactory
}

type WatcherFactory interface {
    NewNamespaceWatcher(ctx context.Context, query eventsource.NamespaceQuery, ...) (watcher.StringsWatcher, error)
    NewNotifyWatcher(ctx context.Context, ...) (watcher.NotifyWatcher, error)
}
```

### ProviderService

Adds cloud provider operations:

```go
type ProviderService struct {
    Service
    providerGetter providertracker.ProviderGetter[Provider]
}
```

## Service Factory

`domain/services/` contains factories that wire everything together:

### ControllerServices

For controller-scoped services:

```go
// domain/services/controller.go
type ControllerServices struct {
    controllerDB changestream.WatchableDBFactory
    logger       logger.Logger
    clock        clock.Clock
}

func (s *ControllerServices) Controller() *controllerservice.Service
func (s *ControllerServices) Cloud() *cloudservice.WatchableService
func (s *ControllerServices) ControllerConfig() *controllerconfigservice.WatchableService
func (s *ControllerServices) Model() *modelservice.WatchableService
// ... 25+ service factory methods
```

### ModelServices

For model-scoped services:

```go
// domain/services/model.go
type ModelServices struct {
    modelDB     changestream.WatchableDBFactory
    modelUUID   model.UUID
    clock       clock.Clock
    // ... other dependencies
}

func (s *ModelServices) Application() *applicationservice.WatchableService
func (s *ModelServices) Machine() *machineservice.WatchableService
func (s *ModelServices) Relation() *relationservice.WatchableService
func (s *ModelServices) Storage() *storageservice.WatchableService
// ... 50+ service factory methods
```

### Factory Method Example

```go
func (s *ModelServices) Application() *applicationservice.WatchableService {
    logger := s.logger.Child("application")

    return applicationservice.NewWatchableService(
        applicationstate.NewState(
            changestream.NewTxnRunnerFactory(s.modelDB),
            s.modelUUID,
            s.clock,
            logger,
        ),
        s.modelWatcherFactory("application"),
        providertracker.ProviderRunner[applicationservice.Provider](
            s.providerFactory,
            s.modelUUID.String(),
        ),
        domain.NewStatusHistory(logger, s.clock),
        s.clock,
        logger,
    )
}
```

## Cross-Domain Dependencies

Common dependencies injected into services:

| Dependency | Purpose |
|------------|---------|
| `StatusHistory` | Records status changes over time |
| `WatcherFactory` | Creates change stream watchers |
| `ProviderFactory` | Creates cloud provider instances |
| `LeaseManager` | Distributed lease coordination |
| `ObjectStore` | Charm/resource storage |
| `clock.Clock` | Time operations (testable) |
| `logger.Logger` | Structured logging |

## Error Handling

Each domain defines specific error types:

```go
// domain/application/errors/
type ApplicationNotFound struct{}
type ApplicationAlreadyExists struct{}
type CharmNotFound struct{}

// Usage: errors satisfy coreerrors interfaces
if errors.Is(err, applicationerrors.ApplicationNotFound) { ... }
```

## Major Domains

| Domain | Purpose |
|--------|---------|
| `application` | Application lifecycle, charms, units, config |
| `machine` | Machine provisioning, instances, containers |
| `model` | Model metadata and lifecycle |
| `relation` | Relations between applications |
| `secret` | Secret management and access |
| `storage` | Storage provisioning |
| `network` | Network config, subnets, spaces |
| `access` | User access control |
| `credential` | Cloud credentials |
| `cloud` | Cloud provider config |
| `controller` | Controller metadata |
| `upgrade` | Upgrade tracking |
