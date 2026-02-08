# Data Model: Kubernetes Deployment Type Support

**Date**: 2026-02-08
**Feature**: 001-k8s-deployment-types

## New Entities

### deployment_type (Lookup Table)

| Field | Type | Constraints |
|-------|------|-------------|
| id    | INT  | PRIMARY KEY |
| name  | TEXT | NOT NULL    |

**Values**:
| id | name       | K8s Resource |
|----|------------|-------------|
| 0  | stateful   | StatefulSet |
| 1  | stateless  | Deployment  |
| 2  | daemon     | DaemonSet   |

### Modified: application

| Field               | Type | Change   | Default | Notes |
|---------------------|------|----------|---------|-------|
| deployment_type_id  | INT  | ADD      | 0       | FK → deployment_type(id). Default 0 (stateful) preserves backward compat for existing apps |

## Modified Domain Types

### core/constraints.Value

New field:
```
DeploymentType *string  // "stateless", "stateful", "daemon"
```

Follows existing pattern of pointer-to-string for optional constraint fields.

### caas.DeploymentType (Existing - No Change)

Already defined in `caas/broker.go:91-93`:
```
DeploymentStateless  = "stateless"
DeploymentStateful   = "stateful"
DeploymentDaemon     = "daemon"
```

### params.CAASApplicationProvisioningInfo (Wire Type)

New field:
```
DeploymentType string  // Added to carry deployment type to worker
```

### params.ApplicationStatus (Wire Type)

New field (CAAS-only):
```
DeploymentType string  // "Deployment", "StatefulSet", "DaemonSet"
```

### cmd/juju/status applicationStatus (Display Type)

New field:
```
DeploymentType string  // Display value for status output
```

### domain/status/service Application (Domain Type)

New field:
```
DeploymentType *string  // nil for IAAS
```

### domain/application AddCAASApplicationArg

New field:
```
DeploymentType string  // Set at deploy time, persisted
```

## State Transitions

```
Deploy (no constraint, no storage) → deployment_type_id = 1 (stateless)
Deploy (no constraint, has storage) → deployment_type_id = 0 (stateful)
Deploy (constraint=stateless)       → deployment_type_id = 1 (stateless)
Deploy (constraint=stateful)        → deployment_type_id = 0 (stateful)
Deploy (constraint=daemon)          → deployment_type_id = 2 (daemon)
Upgrade (existing app, no type set) → deployment_type_id = 0 (stateful) [via DEFAULT]
```

**Immutability**: Once set at deploy time, `deployment_type_id` MUST NOT change for the lifetime of the application. Attempting to change deployment type returns an error directing the operator to redeploy.

## Relationships

```
application ──FK──> deployment_type
    │
    ├── application_scale (existing, no change)
    │     Note: Scale operations blocked for daemon type
    │
    └── k8s_service (existing, no change)
```
