# Contract: deployment-type Constraint

**Change**: Add `deployment-type` to the constraints vocabulary.

## Constraint Definition

- **Name**: `deployment-type`
- **Type**: string (optional)
- **Valid values**: `stateless`, `stateful`, `daemon`
- **Default**: nil (inferred from charm storage)
- **Scope**: CAAS models only (silently ignored on IAAS)

## CLI Usage

```bash
# Explicit selection
juju deploy nginx --constraints="deployment-type=stateless"
juju deploy mysql --constraints="deployment-type=stateful"
juju deploy node-exporter --constraints="deployment-type=daemon"

# Combined with other constraints
juju deploy nginx --constraints="deployment-type=stateless cores=2 mem=4G"

# Model-level default
juju set-model-constraints deployment-type=stateless

# Invalid value
juju deploy nginx --constraints="deployment-type=invalid"
# Error: invalid deployment-type "invalid": valid values are stateless, stateful, daemon
```

## Wire Format

Part of existing `constraints.Value` struct. Serialized as part of standard constraint string parsing. No new API endpoint needed.

## Validation Rules

1. Value MUST be one of: `stateless`, `stateful`, `daemon` (case-sensitive)
2. On IAAS models, value is silently ignored during constraint resolution
3. On CAAS models, value is persisted with the application
4. Once set at deploy time, value cannot be changed (immutable)
