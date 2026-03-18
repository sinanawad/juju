# API Server Facades

## Overview

Facades are RPC endpoints that expose Juju functionality. They follow strict patterns:
- Thin orchestration (not business logic)
- Authorization at facade level
- Bulk argument handling
- Domain service integration

## Directory Structure

```
apiserver/facades/
├── agent/              # Agent-facing (unit agents, machine agents)
│   ├── uniter/
│   ├── provisioner/
│   └── ...
├── client/             # Client-facing (CLI, user tools)
│   ├── application/
│   ├── client/
│   └── ...
└── controller/         # Controller-specific (cross-model, migrations)
    ├── firewaller/
    ├── migrationmaster/
    └── ...
```

## Facade Registration

### Registry

`apiserver/facade/registry.go`:

```go
type Registry struct {
    facades map[string]versions  // name -> version map
}

type versions map[int]record     // version -> factory
```

### Register Function

Each facade has a `register.go`:

```go
// apiserver/facades/client/action/register.go
func Register(registry facade.FacadeRegistry) {
    registry.MustRegister("Action", 7, func(stdCtx context.Context, ctx facade.ModelContext) (facade.Facade, error) {
        return newActionAPIV7(ctx)
    }, reflect.TypeOf((*APIv7)(nil)))
}
```

### Central Registration

`apiserver/allfacades.go`:

```go
func AllFacades() *facade.Registry {
    registry := new(facade.Registry)

    action.Register(registry)
    application.Register(registry)
    client.Register(registry)
    // ... 50+ facades

    return registry
}
```

## ModelContext Interface

Facades receive a `ModelContext` with all dependencies:

```go
type ModelContext interface {
    Auth() Authorizer                           // Authorization
    DomainServices() services.DomainServices    // Domain services
    ModelUUID() model.UUID                      // Current model
    WatcherRegistry() WatcherRegistry           // Watcher management
    Logger() logger.Logger                      // Logging
    // ...
}
```

## Facade Factory Pattern

```go
func newActionAPIV7(ctx facade.ModelContext) (*APIv7, error) {
    // Authorization check
    if !ctx.Auth().AuthClient() {
        return nil, apiservererrors.ErrPerm
    }

    // Extract services
    services := ctx.DomainServices()

    api := &ActionAPI{
        authorizer:         ctx.Auth(),
        applicationService: services.Application(),
        operationService:   services.Operation(),
        modelInfoService:   services.ModelInfo(),
        watcherRegistry:    ctx.WatcherRegistry(),
    }

    return &APIv7{ActionAPI: api}, nil
}
```

## Authorization

### Authorizer Interface

```go
type Authorizer interface {
    GetAuthTag() names.Tag
    AuthController() bool
    AuthMachineAgent() bool
    AuthUnitAgent() bool
    AuthClient() bool
    AuthOwner(tag names.Tag) bool
    HasPermission(ctx context.Context, op permission.Access, target names.Tag) error
}
```

### Authorization Patterns

**Check entity type:**
```go
if !ctx.Auth().AuthMachineAgent() && !ctx.Auth().AuthUnitAgent() {
    return nil, apiservererrors.ErrPerm
}
```

**Check permission:**
```go
func (a *ActionAPI) checkCanWrite(ctx context.Context) error {
    return a.authorizer.HasPermission(ctx, permission.WriteAccess, a.modelTag)
}

func (a *ActionAPI) RunAction(ctx context.Context, args params.Actions) (params.ActionResults, error) {
    if err := a.checkCanWrite(ctx); err != nil {
        return params.ActionResults{}, err
    }
    // ...
}
```

**Tag-based auth helpers:**
```go
accessUnit := common.AuthFuncForTagKind(names.UnitTagKind)
accessMachine := common.AuthFuncForTagKind(names.MachineTagKind)
accessAny := common.AuthAny(accessUnit, accessMachine)

canAccess, _ := accessAny(ctx)
if !canAccess(entityTag) {
    return apiservererrors.ErrPerm
}
```

## Bulk Argument Pattern

### Request/Response Types

```go
// rpc/params/params.go

// Input wrapper
type Entities struct {
    Entities []Entity `json:"entities"`
}

// Output wrapper - results match input order
type ActionResults struct {
    Results []ActionResult `json:"results"`
}
```

### Bulk Operation Implementation

```go
func (a *ActionAPI) Actions(ctx context.Context, args params.Entities) (params.ActionResults, error) {
    if err := a.checkCanRead(ctx); err != nil {
        return params.ActionResults{}, err
    }

    // Pre-allocate results
    results := params.ActionResults{
        Results: make([]params.ActionResult, len(args.Entities)),
    }

    // Process each item independently
    for i, entity := range args.Entities {
        tag, err := names.ParseActionTag(entity.Tag)
        if err != nil {
            results.Results[i].Error = apiservererrors.ServerError(err)
            continue  // Don't stop on individual errors
        }

        task, err := a.operationService.GetTask(ctx, tag.Id())
        if err != nil {
            results.Results[i].Error = apiservererrors.ServerError(err)
            continue
        }

        results.Results[i] = toActionResult(task)
    }

    return results, nil  // Return all results
}
```

## Domain Service Integration

Facades call domain services, not state directly:

```go
type ActionAPI struct {
    authorizer         facade.Authorizer
    applicationService ApplicationService  // Interface
    operationService   OperationService    // Interface
    // ...
}

// Define minimal interface needed
type ApplicationService interface {
    GetCharmLocatorByApplicationName(ctx context.Context, name string) (CharmLocator, error)
}

type OperationService interface {
    GetTask(ctx context.Context, id string) (operation.Task, error)
    AddActionOperation(ctx context.Context, ...) (operation.RunResult, error)
}
```

## Common Utilities

`apiserver/common/` provides shared functionality:

### BlockChecker

```go
check := common.NewBlockChecker(domainServices.BlockCommand())

if err := check.ChangeAllowed(ctx); err != nil {
    return handleBlockedOperation(err)
}
```

### LifeGetter

```go
lifeGetter := common.NewLifeGetter(
    applicationService,
    machineService,
    authFunc,
    logger,
)
```

## Method Exposure

Exported methods on facade structs become RPC methods:

```go
// Exposed as RPC method "Actions"
func (a *ActionAPI) Actions(ctx context.Context, args params.Entities) (params.ActionResults, error)

// NOT exposed (lowercase)
func (a *ActionAPI) checkCanRead(ctx context.Context) error
```

Required signature:
```go
func (f *Facade) Method(ctx context.Context, args ParamType) (ResultType, error)
```

## Versioning

Multiple versions of same facade:

```go
type APIv7 struct {
    *ActionAPI
}

type APIv8 struct {
    *ActionAPI
}

// v8 can override or add methods
func (a *APIv8) NewMethod(ctx context.Context, args params.NewArgs) (params.NewResults, error) {
    // ...
}
```

## Adding a New Facade

1. Create directory `apiserver/facades/{category}/{name}/`

2. Create `facade.go`:
```go
type MyAPI struct {
    authorizer facade.Authorizer
    service    MyService
}
```

3. Create `register.go`:
```go
func Register(registry facade.FacadeRegistry) {
    registry.MustRegister("MyFacade", 1, newMyAPIV1, reflect.TypeOf((*MyAPI)(nil)))
}

func newMyAPIV1(stdCtx context.Context, ctx facade.ModelContext) (facade.Facade, error) {
    return &MyAPI{
        authorizer: ctx.Auth(),
        service:    ctx.DomainServices().MyService(),
    }, nil
}
```

4. Add to `apiserver/allfacades.go`:
```go
myfacade.Register(registry)
```

## Key Files

| File | Purpose |
|------|---------|
| `apiserver/facade/registry.go` | Registration mechanism |
| `apiserver/facade/interface.go` | ModelContext, Authorizer |
| `apiserver/allfacades.go` | Central registration |
| `apiserver/common/` | Shared utilities |
| `rpc/params/` | Wire format types |
