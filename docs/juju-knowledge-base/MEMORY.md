# Juju Codebase Knowledge Base

This memory contains architectural knowledge for the Juju orchestration engine.

## Quick Reference

**Build**: `make install` (full) | `make go-build` (no schema)
**Controller binary**: `make jujud-controller` (NOT `go build ./cmd/jujud` - that's the agent binary without domain services!)
**Test**: `go test ./path` | `go test -run 'TestName' ./path`
**Lint**: `make pre-check`

## K8s Controller Binary Replacement

When pushing custom binaries to a microk8s controller:
1. Build with `make jujud-controller` → output at `~/go/bin/jujud`
2. `cat ~/go/bin/jujud | microk8s kubectl exec -i controller-0 -n controller-k8s -c api-server -- sh -c 'cat > /tmp/jujud'`
3. `microk8s kubectl exec controller-0 -n controller-k8s -c api-server -- chmod +x /tmp/jujud`
4. `microk8s kubectl exec controller-0 -n controller-k8s -c api-server -- sh -c 'mv /var/lib/juju/tools/jujud /var/lib/juju/tools/jujud.old && mv /tmp/jujud /var/lib/juju/tools/jujud && rm -f /var/lib/juju/tools/jujud.old'`
5. `microk8s kubectl exec controller-0 -n controller-k8s -c api-server -- kill $(microk8s kubectl exec controller-0 -n controller-k8s -c api-server -- pgrep -f 'jujud machine')`
6. IMPORTANT: Kill only the jujud process (NOT PID 1/pebble) or the startup script will overwrite with `/opt/jujud`
7. `/opt/jujud` is read-only; only `/var/lib/juju/tools/jujud` can be replaced
8. `juju debug-log` does NOT show all controller logs; use `microk8s kubectl -n controller-k8s logs controller-0 -c api-server` for full output

## Architecture Layers (strict import boundaries)

```
cmd/           CLI entry points (juju, jujud, jujuc, jujud-controller)
    ↓
api/           Client-side API consumers (thin wire→model translation)
    ↓
apiserver/     RPC facades (auth + encoding + call domain services)
    ↓
domain/        Business logic: services (logic) + state (persistence)
    ↓
core/          Primitives with minimal deps (types, interfaces, validation)
    ↓
internal/worker/  Background actors (tomb lifecycle, dependency engine)
```

**Import rules**: See [architecture-layers.md](architecture-layers.md)

## Key Patterns

| Pattern | Location | Summary |
|---------|----------|---------|
| Service/State | `domain/*/service/`, `domain/*/state/` | Services hold logic, States abstract DB |
| Worker lifecycle | `internal/worker/*/` | tomb.Tomb manages goroutines |
| Manifolds | `internal/worker/*/manifold.go` | Dependency declarations for workers |
| Facades | `apiserver/facades/` | RPC endpoints with auth + bulk args |
| Factory | `domain/services/` | `ModelServices`/`ControllerServices` create services |

## Domain Concepts

- **Controller**: Manages multiple models, runs on controller machines
- **Model**: Deployment environment (IAAS or CAAS/k8s)
- **Application**: Deployed charm with configuration
- **Unit**: Instance of application (e.g., mysql/0)
- **Machine**: Compute resource (can contain containers: 0/lxd/1)
- **Charm**: Operator package defining application lifecycle
- **Relation**: Integration between applications

## Detailed Documentation

- [architecture-layers.md](architecture-layers.md) - Layer boundaries and import rules
- [core-domain-rules.md](core-domain-rules.md) - Life lifecycle, watchers, relations, secrets domain rules
- [domain-services.md](domain-services.md) - Service/state patterns, factory
- [worker-patterns.md](worker-patterns.md) - Worker lifecycle, manifolds, dependency engine
- [apiserver-facades.md](apiserver-facades.md) - Facade registration, auth, bulk operations
- [core-types.md](core-types.md) - Primitive types and interfaces
- [k8s-provider.md](k8s-provider.md) - Kubernetes/CAAS provider architecture and gaps
- [charm-types.md](charm-types.md) - Charm formats, metadata, sidecar charms
- [controller-ha.md](controller-ha.md) - Controller HA architecture, object store, deterioration risks
- [k8s-deployment-type-plan.md](k8s-deployment-type-plan.md) - **ACTIVE** Implementation plan for Deployment/DaemonSet support

## Operational Rules

1. **Prefer juju CLI over substrate** - Anything achievable with `juju` commands, do it with `juju`. Only go to the substrate directly (e.g., `kubectl`) when testing juju behavior by bypassing it intentionally.
2. **Do not use `--force` with juju** unless specifically required or requested by the user. Use normal removal/destruction commands first.
3. **Keep design artifacts in sync** - After each modification, verify spec, plan, tasks, and constitution still reflect what was done. Update them if needed, or notify the user if principles need changing.
4. **microk8s environment** - We use `microk8s` as a normal user per its documented CLI. Always use `microk8s kubectl` (NOT standalone `kubectl`, which is not installed).
5. **No standalone kubectl** - `kubectl` is always `microk8s kubectl`. Never assume `kubectl` is available.
6. **StatefulSet regression guard** - Every modification to K8s support must verify no regression to the original StatefulSet behavior. Run tests for affected packages AND simulate equivalent scenarios with StatefulSet (default deploy without `deployment-type` constraint) to confirm original Juju K8s behavior is preserved.
7. **Always save Juju knowledge** - When learning anything new about Juju internals (architecture, bugs, patterns, domain concepts), summarize it in memory files for future reference across sessions.

## Critical Principles

1. **EVERYTHING FAILS** - All operations must be idempotent/resumable
2. **Strict layering** - Never import up the layer stack
3. **Worker pattern** - Long-running tasks use tomb.Tomb + dependency.Engine
4. **Life is monotonic** - `Alive → Dying → Dead`, never backwards. Dying starts cleanup; Dead only after all teardown complete. See [core-domain-rules.md](core-domain-rules.md)
5. **Watchers signal change, not state** - Notifications may coalesce; consumers must re-query. `NotifyWatcher` for singles, `StringsWatcher` for collections. See [core-domain-rules.md](core-domain-rules.md)
6. **Never use time.Now()** - Inject clock.Clock for testability
7. **Bulk arguments** - Facades accept/return arrays, process all items

## Code Review Checklist

When asked to review code, ALWAYS:
1. Check alignment with spec/plan requirements
2. **Run the test suite** (`go test ./affected/packages/... -count=1`) and verify no regressions
3. Check documentation drift between spec, plan, tasks, and implementation
4. Verify architectural layer boundaries are respected

## Known Active Bugs

See [controller-ha.md](controller-ha.md) for full details:
- **BUG-1**: Spin-lock in `scopedContext.Done()` — rapid CPU burn per blob fetch (PR #21857 fix)
- **BUG-2**: Object store dedup delete — `remove()` unconditionally deletes physical file even when other paths still reference same hash (under investigation)
- **BUG-3**: Namespace mutation in `BlobRetriever` — `r.namespace` permanently overwritten on first controller-NS retrieve call

## Feedback

- [feedback_build_after_rebase.md](feedback_build_after_rebase.md) - Always `go build` after each conflicted rebase commit, not just at the end

## Key Files

| Purpose | Path |
|---------|------|
| Domain service factory | `domain/services/model.go`, `domain/services/controller.go` |
| Facade registry | `apiserver/facade/registry.go` |
| All facades | `apiserver/allfacades.go` |
| Worker dependency engine | External: `github.com/juju/worker/v4/dependency` |
| Agent manifolds | `cmd/jujud/agent/machine/manifolds.go` |
| Object store (file) | `internal/objectstore/fileobjectstore.go` |
| Object store (remote) | `internal/objectstore/remote/retriever.go` |
| Object store metadata DB | `domain/objectstore/state/state.go` |
