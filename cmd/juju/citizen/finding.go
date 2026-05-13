// Copyright 2026 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package citizen

import "time"

// Severity is the calibrated impact level of a Finding. The set of
// values is fixed by constitution Principle II; detectors MUST NOT
// introduce additional values.
type Severity string

const (
	// SeverityInfo is a convention violation with no functional impact.
	SeverityInfo Severity = "info"
	// SeverityWarning is a degrading state requiring action within a sprint.
	SeverityWarning Severity = "warning"
	// SeverityCritical is a data-integrity, security, or hard-breakage event.
	SeverityCritical Severity = "critical"
)

// rank returns the sort order for severity (lower prints first).
func (s Severity) rank() int {
	switch s {
	case SeverityCritical:
		return 0
	case SeverityWarning:
		return 1
	case SeverityInfo:
		return 2
	default:
		return 99
	}
}

// Owner classifies who is best placed to act on a Finding. Constitution
// Principle III: assigned by the detector, never by the CLI.
type Owner string

const (
	OwnerCharmAuthor Owner = "charm-author"
	OwnerOperator    Owner = "operator"
	OwnerMixed       Owner = "mixed"
	OwnerPlatform    Owner = "platform"
)

// EntityKind names the type of Juju entity a Finding targets.
type EntityKind string

const (
	EntityKindUnit        EntityKind = "unit"
	EntityKindApplication EntityKind = "application"
	EntityKindModel       EntityKind = "model"
)

// Finding is the atomic output of the citizenship observatory. The
// eight required fields are the schema enforced at the detection-layer
// boundary; missing any of them is a bug. Since is optional and set
// only by stateful detectors to record when the violation began;
// pure detectors leave it nil.
type Finding struct {
	CheckID        string     `yaml:"check_id"        json:"check_id"`
	Severity       Severity   `yaml:"severity"        json:"severity"`
	Entity         string     `yaml:"entity"          json:"entity"`
	EntityKind     EntityKind `yaml:"entity_kind"     json:"entity_kind"`
	Owner          Owner      `yaml:"owner"           json:"owner"`
	Summary        string     `yaml:"summary"         json:"summary"`
	Recommendation string     `yaml:"recommendation"  json:"recommendation"`
	ProtocolRef    string     `yaml:"protocol_ref"    json:"protocol_ref"`
	Since          *time.Time `yaml:"since,omitempty" json:"since,omitempty"`
}

// withSince returns a copy of f with Since set. Stateful detectors
// call this after newFinding to attach the violation start time.
func (f Finding) withSince(t time.Time) Finding {
	f.Since = &t
	return f
}

// newFinding constructs a Finding and panics if any required field is
// empty. This is a development-time guardrail: detectors hardcode
// every field, so the zero value can never legitimately reach here at
// runtime.
func newFinding(
	checkID string,
	severity Severity,
	entity string,
	entityKind EntityKind,
	owner Owner,
	summary, recommendation, protocolRef string,
) Finding {
	switch {
	case checkID == "":
		panic("citizen: Finding has empty check_id")
	case severity == "":
		panic("citizen: Finding has empty severity")
	case entity == "":
		panic("citizen: Finding has empty entity")
	case entityKind == "":
		panic("citizen: Finding has empty entity_kind")
	case owner == "":
		panic("citizen: Finding has empty owner")
	case summary == "":
		panic("citizen: Finding has empty summary")
	case recommendation == "":
		panic("citizen: Finding has empty recommendation")
	case protocolRef == "":
		panic("citizen: Finding has empty protocol_ref")
	}
	return Finding{
		CheckID:        checkID,
		Severity:       severity,
		Entity:         entity,
		EntityKind:     entityKind,
		Owner:          owner,
		Summary:        summary,
		Recommendation: recommendation,
		ProtocolRef:    protocolRef,
	}
}
