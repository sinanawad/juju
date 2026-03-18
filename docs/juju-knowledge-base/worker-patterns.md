# Worker Architecture and Patterns

## Overview

Workers are long-running background tasks in Juju agents. There are 123+ workers covering:
- Networking & communication (apicaller, apiserver, httpserver)
- Machine & unit management (machiner, uniter, deployer)
- Provisioning (storageprovisioner, computeprovisioner)
- Coordination (fortress, gate, leadership, lease)
- And many more...

## Worker Interface

From `github.com/juju/worker/v4`:

```go
type Worker interface {
    Kill()         // Signal shutdown
    Wait() error   // Block until terminated, return error
}
```

## Lifecycle Management with tomb.Tomb

All workers use `tomb.Tomb` for structured goroutine management:

```go
import "gopkg.in/tomb.v2"

type myWorker struct {
    tomb   tomb.Tomb
    config Config
}

func (w *myWorker) Kill() {
    w.tomb.Kill(nil)
}

func (w *myWorker) Wait() error {
    return w.tomb.Wait()
}

func newWorker(config Config) (*myWorker, error) {
    w := &myWorker{config: config}
    w.tomb.Go(w.loop)  // Start main goroutine
    return w, nil
}

func (w *myWorker) loop() error {
    defer cleanup()

    for {
        select {
        case <-w.tomb.Dying():
            return tomb.ErrDying  // Graceful shutdown
        case event := <-someChannel:
            if err := w.handle(event); err != nil {
                return err  // Worker will be restarted
            }
        }
    }
}
```

## Catacomb Pattern

Alternative to tomb for managing sub-workers:

```go
import "github.com/juju/worker/v4/catacomb"

type Worker struct {
    config   Config
    catacomb catacomb.Catacomb
}

func (w *Worker) Kill() { w.catacomb.Kill(nil) }
func (w *Worker) Wait() error { return w.catacomb.Wait() }

func New(config Config) (*Worker, error) {
    w := &Worker{config: config}
    err := catacomb.Invoke(catacomb.Plan{
        Site: &w.catacomb,
        Work: w.loop,
    })
    return w, err
}

// Add sub-worker (catacomb manages its lifecycle)
func (w *Worker) startSubWorker() error {
    sub := newSubWorker()
    return w.catacomb.Add(sub)  // Catacomb will Kill sub when dying
}
```

## Manifold Pattern

Manifolds declare worker dependencies for the dependency engine:

```go
// internal/worker/logger/manifold.go
func Manifold(config ManifoldConfig) dependency.Manifold {
    return dependency.Manifold{
        Inputs: []string{
            config.AgentName,      // Dependency names
            config.APICallerName,
        },
        Start: func(ctx context.Context, getter dependency.Getter) (worker.Worker, error) {
            // Resolve dependencies
            var agent agent.Agent
            if err := getter.Get(config.AgentName, &agent); err != nil {
                return nil, err
            }
            var apiCaller base.APICaller
            if err := getter.Get(config.APICallerName, &apiCaller); err != nil {
                return nil, err
            }

            // Create worker with resolved dependencies
            return NewLogger(WorkerConfig{
                Agent:     agent,
                APICaller: apiCaller,
            })
        },
        Output: outputFunc,  // Optional: export to other workers
        Filter: filterFunc,  // Optional: control restart behavior
    }
}
```

### Manifold Components

| Field | Purpose |
|-------|---------|
| `Inputs` | Names of required dependencies |
| `Start` | Factory function to create worker |
| `Output` | Export worker capabilities to dependents |
| `Filter` | Transform errors to control restart behavior |

### Output Function

Export typed interfaces to dependent workers:

```go
Output: func(in worker.Worker, out interface{}) error {
    worker, _ := in.(*MyWorker)
    switch ptr := out.(type) {
    case *SomeInterface:
        *ptr = worker
    default:
        return errors.Errorf("unexpected type %T", out)
    }
    return nil
}
```

### Filter Function

Control restart behavior:

```go
Filter: func(err error) error {
    if errors.Is(err, ErrUnlocked) {
        return dependency.ErrBounce  // Restart worker
    }
    return err  // Stop worker
}
```

## Dependency Engine

The dependency engine (from `github.com/juju/worker/v4/dependency`) orchestrates workers:

1. **Resolves dependency graph** from manifold declarations
2. **Starts workers** when all inputs are available
3. **Restarts workers** when dependencies change
4. **Stops dependents** when a dependency fails
5. **Converges** toward stable state

### Key Behaviors

- Workers started in essentially random order
- Dependencies restarting cause dependent restarts
- `dependency.ErrMissing` - dependency not available (engine retries)
- `dependency.ErrBounce` - restart this worker
- Other errors - stop worker

## Housing Decorator

`agent/engine/housing.go` adds cross-cutting concerns:

```go
type Housing struct {
    Flags  []string              // Only run when these flags are set
    Occupy string                // Run inside a fortress.Guest lock
    Filter dependency.FilterFunc // Error handling
}
```

## Synchronization Primitives

### Fortress (mutual exclusion)

```go
// Guard interface - control access
type Guard interface {
    Unlock(context.Context) error
    Lockdown(context.Context) error
}

// Guest interface - execute under lock
type Guest interface {
    Visit(context.Context, func() error) error
}
```

### Gate (simple signaling)

```go
type Lock interface {
    Unlock()                    // Signal gate open
    Unlocked() <-chan struct{}  // Wait for open
    IsUnlocked() bool           // Check state
}
```

### Flag (boolean state)

```go
type Flag interface {
    Check() bool  // Always returns same value for instance
}
```

## Watcher Integration

Workers often watch for changes:

```go
func (w *Worker) loop() error {
    watcher, err := w.facade.Watch()
    if err != nil {
        return err
    }
    if err := w.catacomb.Add(watcher); err != nil {
        return err
    }

    for {
        select {
        case <-w.catacomb.Dying():
            return w.catacomb.ErrDying()
        case _, ok := <-watcher.Changes():
            if !ok {
                return errors.New("watcher closed")
            }
            if err := w.handle(); err != nil {
                return err
            }
        }
    }
}
```

## Common Worker Patterns

### Pattern 1: API-dependent worker

```go
type Config struct {
    Agent     agent.Agent
    APICaller base.APICaller
    Logger    logger.Logger
}

func (c Config) Validate() error {
    if c.Agent == nil {
        return errors.NotValidf("nil Agent")
    }
    // ...
}
```

### Pattern 2: NotifyWorker

For event-driven workers:

```go
type NotifyHandler interface {
    SetUp(ctx context.Context) (NotifyWatcher, error)
    Handle(ctx context.Context) error
    TearDown() error
}

// Use with watcher.NewNotifyWorker(handler)
```

### Pattern 3: Zero-dependency worker

```go
func Manifold(config ManifoldConfig) dependency.Manifold {
    return dependency.Manifold{
        Inputs: nil,  // No dependencies
        Start:  config.Start,
    }
}
```

## Key Worker Locations

| Worker | Path | Purpose |
|--------|------|---------|
| uniter | `internal/worker/uniter/` | Unit agent main worker |
| machiner | `internal/worker/machiner/` | Machine lifecycle |
| apiserver | `internal/worker/apiserver/` | API server |
| dbaccessor | `internal/worker/dbaccessor/` | Database access |
| fortress | `internal/worker/fortress/` | Mutual exclusion |
| gate | `internal/worker/gate/` | Simple signaling |

## Agent Manifold Registration

Workers are registered in agent-specific manifold functions:

- `cmd/jujud/agent/machine/manifolds.go` - Machine agent workers
- `cmd/jujud/agent/unit/manifolds.go` - Unit agent workers (if separate)
- `cmd/jujud-controller/agent/manifolds.go` - Controller workers
