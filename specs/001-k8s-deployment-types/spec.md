# Feature Specification: Kubernetes Deployment and DaemonSet Workload Types

**Feature Branch**: `001-k8s-deployment-types`
**Created**: 2026-02-06
**Status**: Draft
**Input**: User description: "Add support for Deployment and DaemonSet workload types in the Kubernetes provider"

## Clarifications

### Session 2026-02-06

- Q: What charm storage characteristics trigger StatefulSet vs Deployment default? → A: Any charm declaring persistent storage defaults to StatefulSet; only charms with no storage declarations default to Deployment.
- Q: Should the deployment-type constraint be silently ignored or produce a warning on IAAS models? → A: Silently ignore, consistent with existing K8s-only constraint behavior.
- Q: Where should the workload type appear in the status output? → A: Both: as a column in the applications summary table and in per-application detail output.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Deploy a Stateless Application (Priority: P1)

As an operator deploying a stateless charm (e.g., a web frontend or API gateway) to a Kubernetes model, I want my application to run as a Deployment workload rather than a StatefulSet, so that I get faster rollouts, simpler scaling, and behavior that matches how stateless workloads are conventionally run on Kubernetes.

**Why this priority**: This is the most common use case. Many production Kubernetes workloads are stateless and should use Deployments for proper rolling updates, surge capacity during upgrades, and reduced operational overhead. Currently all charms are forced into StatefulSets regardless of whether they need stable storage or stable network identities.

**Independent Test**: Can be fully tested by deploying any charm without persistent storage requirements to a Kubernetes model and verifying it runs as a Deployment workload with proper scaling and lifecycle behavior.

**Acceptance Scenarios**:

1. **Given** a Kubernetes model and a charm with no storage declarations in its metadata, **When** the operator deploys the charm without specifying a workload type, **Then** the system automatically selects the Deployment workload type.
2. **Given** a Kubernetes model, **When** the operator deploys a charm and explicitly specifies the "stateless" workload type via a constraint, **Then** the application runs as a Deployment workload regardless of its storage requirements.
3. **Given** a running Deployment-based application, **When** the operator scales up or down, **Then** the replica count changes and new instances start without requiring stable ordinal identities.
4. **Given** a running Deployment-based application, **When** the operator views application status, **Then** the displayed workload type indicates "Deployment" and unit counts are accurate.

---

### User Story 2 - Explicit Workload Type Selection via Constraints (Priority: P1)

As an operator, I want to explicitly choose the workload type (stateless, stateful, or daemon) when deploying a charm to a Kubernetes model, so that I can override the automatic selection when I know better what my application needs.

**Why this priority**: Automatic inference is a good default, but operators need explicit control. A charm might declare storage for caching purposes but still be best served by a Deployment. Explicit override is essential for production use.

**Independent Test**: Can be fully tested by deploying the same charm three times with different workload type constraints and verifying each uses the correct underlying workload type.

**Acceptance Scenarios**:

1. **Given** a Kubernetes model, **When** the operator deploys a charm with the constraint `deployment-type=stateless`, **Then** the application runs as a Deployment workload.
2. **Given** a Kubernetes model, **When** the operator deploys a charm with the constraint `deployment-type=stateful`, **Then** the application runs as a StatefulSet workload.
3. **Given** a Kubernetes model, **When** the operator deploys a charm with the constraint `deployment-type=daemon`, **Then** the application runs as a DaemonSet workload.
4. **Given** a Kubernetes model, **When** the operator deploys a charm with an invalid deployment-type value, **Then** the system returns a clear error message listing the valid options.
5. **Given** an IAAS (non-Kubernetes) model, **When** the operator specifies a deployment-type constraint, **Then** the system silently ignores the constraint (no warning or error).

---

### User Story 3 - Deploy a Node-Level Agent as a DaemonSet (Priority: P2)

As an operator deploying a monitoring agent, log collector, or security scanner charm, I want my application to run as a DaemonSet so that exactly one instance runs on every node in the cluster, matching how node-level agents are conventionally operated on Kubernetes.

**Why this priority**: DaemonSets serve a distinct and important use case for infrastructure-level charms (monitoring, logging, security). While less common than stateless Deployments, this workload type enables an entire class of Kubernetes-native operations that are currently impossible with Juju.

**Independent Test**: Can be fully tested by deploying a charm with the "daemon" workload type constraint and verifying that one instance is scheduled per cluster node.

**Acceptance Scenarios**:

1. **Given** a Kubernetes model with multiple nodes, **When** the operator deploys a charm with the constraint `deployment-type=daemon`, **Then** one instance of the application runs on each node.
2. **Given** a running DaemonSet-based application, **When** a new node is added to the cluster, **Then** an instance of the application is automatically scheduled on the new node.
3. **Given** a running DaemonSet-based application, **When** the operator attempts to manually scale the application, **Then** the system returns a clear error explaining that DaemonSet applications are scaled by adding or removing cluster nodes, not by setting a replica count.
4. **Given** a running DaemonSet-based application, **When** the operator views application status, **Then** the displayed workload type indicates "DaemonSet" and the instance count reflects the number of nodes running the application.

---

### User Story 4 - Backward-Compatible Default Behavior (Priority: P1)

As an operator with existing deployed charms on Kubernetes, I want the system to continue working exactly as it does today for all my existing applications, so that this new feature does not disrupt any running workloads.

**Why this priority**: Backward compatibility is critical. Existing charms that rely on StatefulSet semantics (stable network identity, ordered deployment, persistent volumes) must not break.

**Independent Test**: Can be fully tested by deploying any existing charm that uses persistent storage and verifying it continues to run as a StatefulSet with no behavior changes.

**Acceptance Scenarios**:

1. **Given** a Kubernetes model and a charm with persistent storage requirements, **When** the operator deploys the charm without specifying a workload type, **Then** the system automatically selects the StatefulSet workload type (preserving current behavior).
2. **Given** existing running applications deployed before this feature existed, **When** the system is upgraded, **Then** all existing applications continue running as StatefulSets with no disruption.
3. **Given** a charm that previously deployed as a StatefulSet, **When** the operator redeploys or upgrades it without specifying a workload type, **Then** the application continues as a StatefulSet.

---

### User Story 5 - Workload Type Visibility in Status (Priority: P2)

As an operator managing multiple applications across a Kubernetes model, I want to see the workload type of each application in the status output, so that I can quickly understand how each application is deployed and troubleshoot any issues.

**Why this priority**: Visibility into workload types is important for operational awareness, especially as models now contain a mix of Deployments, StatefulSets, and DaemonSets. Without this, operators would need to inspect the Kubernetes cluster directly.

**Independent Test**: Can be fully tested by deploying applications with different workload types and running the status command to verify workload type information is displayed.

**Acceptance Scenarios**:

1. **Given** a Kubernetes model with applications using different workload types, **When** the operator views the model status summary, **Then** the applications table includes a workload type column showing each application's type (Deployment, StatefulSet, or DaemonSet).
2. **Given** a Kubernetes model, **When** the operator views the detail output for a specific application, **Then** the workload type is included in the application detail.
3. **Given** an IAAS model, **When** the operator views the model status, **Then** no workload type column is displayed (not applicable to non-Kubernetes models).

---

### Edge Cases

- What happens when a charm declares persistent storage but the operator deploys with `deployment-type=stateless`? The system should allow this (the operator may intend to use ephemeral storage or shared volumes) but issue a warning that persistent storage may not behave as expected with a stateless workload type.
- What happens when a DaemonSet application charm tries to use storage with non-shared access mode? The system should reject this with a clear error, as DaemonSets cannot use storage that requires stable identity.
- What happens when an operator tries to change the workload type of an already-running application? The system should reject this with a clear error, as changing workload types requires a full redeployment (this is a destructive operation).
- What happens when constraints are set at the model level for deployment-type? The system should apply the model-level constraint as a default for new deployments within that model, overridable per-application.
- What happens during a controller upgrade when existing applications have no workload type recorded? The system should default existing applications to StatefulSet to preserve current behavior.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST support three workload types for Kubernetes applications: stateless (Deployment), stateful (StatefulSet), and daemon (DaemonSet).
- **FR-002**: System MUST allow operators to specify workload type via a `deployment-type` constraint with values `stateless`, `stateful`, or `daemon`.
- **FR-003**: System MUST automatically infer the workload type when not explicitly specified: any charm that declares persistent storage in its metadata defaults to stateful (StatefulSet); only charms with no storage declarations default to stateless (Deployment).
- **FR-004**: System MUST continue to deploy all existing charms as StatefulSets when no workload type is specified and the charm has persistent storage, preserving backward compatibility.
- **FR-005**: System MUST reject manual scaling operations (add-unit, remove-unit, scale) for DaemonSet applications with a clear error message explaining that DaemonSet scaling is determined by the number of cluster nodes.
- **FR-006**: System MUST reject attempts to change the workload type of a running application, returning an error indicating that redeployment is required.
- **FR-007**: System MUST display the workload type of each application in the status output for Kubernetes models, both as a column in the applications summary table and in per-application detail output.
- **FR-008**: System MUST validate the `deployment-type` constraint value at deploy time and return a clear error for invalid values.
- **FR-009**: System MUST silently ignore the `deployment-type` constraint when used with non-Kubernetes (IAAS) models, consistent with how other Kubernetes-specific constraints behave.
- **FR-010**: System MUST persist the workload type for each application so that it survives controller restarts and upgrades.
- **FR-011**: System MUST default existing applications (deployed before this feature) to the StatefulSet workload type during upgrades, with no disruption to running workloads.
- **FR-012**: System MUST issue a warning when an operator deploys with `deployment-type=stateless` but the charm declares persistent storage requirements.

### Key Entities

- **Workload Type**: The kind of Kubernetes workload controller used for an application. One of: stateless (Deployment), stateful (StatefulSet), or daemon (DaemonSet). Determined at deploy time either automatically or by explicit operator constraint, and immutable for the lifetime of the application.
- **Deployment-Type Constraint**: A new constraint field that allows operators to explicitly select the workload type. Follows existing constraint semantics (can be set at model or application level, application-level overrides model-level).

### Assumptions

- The underlying Kubernetes provider already supports creating and managing Deployment, StatefulSet, and DaemonSet resources. This feature is about wiring the selection mechanism through the system, not implementing new Kubernetes resource management.
- The automatic inference heuristic is simple: any charm declaring persistent storage in its metadata gets StatefulSet; charms with no storage declarations get Deployment. This is deliberately conservative to preserve backward compatibility. Operators who need different behavior can use the explicit constraint.
- Changing workload type on a running application is out of scope. This would require destroying and recreating the Kubernetes resources, which is a complex and potentially data-losing operation best handled by explicit redeployment.
- DaemonSet unit representation in status will use the same ordinal naming as other workload types for simplicity. Mapping pods to specific nodes is deferred to future work.
- The `deployment-type` constraint is only meaningful for Kubernetes (CAAS) models and is silently ignored (no warning or error) for IAAS models, consistent with how other Kubernetes-specific constraints behave.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Operators can deploy stateless charms that run as Deployments, completing the deploy workflow in the same amount of time as current StatefulSet deployments (no performance regression).
- **SC-002**: Operators can deploy DaemonSet-based charms that automatically scale to all cluster nodes, with the instance count matching the node count within 60 seconds of a node joining.
- **SC-003**: 100% of existing charms deployed before this feature continue to operate as StatefulSets after a controller upgrade, with zero disruption.
- **SC-004**: Operators can determine the workload type of any Kubernetes application from the status output without needing to inspect the Kubernetes cluster directly.
- **SC-005**: Invalid workload type values are caught at deploy time with a clear, actionable error message, preventing any partial or failed deployments.
- **SC-006**: Manual scaling operations on DaemonSet applications are blocked with a clear error, preventing operator confusion about how DaemonSets scale.
- **SC-007**: Operators using the constraint override can deploy any charm as any supported workload type, with 100% of valid constraint values being accepted and applied correctly.
