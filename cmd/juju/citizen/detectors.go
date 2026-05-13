// Copyright 2026 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package citizen

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/juju/names/v6"

	corestatus "github.com/juju/juju/core/status"
	"github.com/juju/juju/rpc/params"
)

// Field availability verified against rpc/params/status.go on branch
// 4.0 (2026-05-13):
//
//   - params.FullStatus.Applications        map[string]ApplicationStatus
//   - ApplicationStatus.CanUpgradeTo        string
//   - ApplicationStatus.Units               map[string]UnitStatus
//   - UnitStatus.WorkloadStatus             DetailedStatus
//   - DetailedStatus.Status                 string  (workload state)
//   - DetailedStatus.Info                   string  (status message)
//   - DetailedStatus.Since                  *time.Time
//
// All three v1 detectors operate on these fields. If a field is
// missing (zero value, nil pointer, nil map), the detector emits zero
// findings -- never a runtime panic. See spec FR-009.

// Detector is a pure function that consumes a status snapshot and
// reference time, returning zero or more Findings. Stateless and
// idempotent.
type Detector func(status *params.FullStatus, now time.Time) []Finding

// detectors is the registry of pure (stateless) detectors. Order is
// not meaningful -- findings are sorted by severity then entity at
// the end of the pipeline.
var detectors = []Detector{
	detectActiveWithMessage,
	detectBlockedNoMessage,
	detectCharmRevisionAging,
	detectUnitBlockedStale,
	detectEntityStuckDying,
	detectModelSuspendedCredential,
}

// statusHistoryAPI is the slice of Client used by stateful detectors
// that look at per-unit status history. The real implementation is
// satisfied by *api/client/client.Client.
type statusHistoryAPI interface {
	StatusHistory(
		ctx context.Context,
		kind corestatus.HistoryKind,
		tag names.Tag,
		filter corestatus.StatusHistoryFilter,
	) (corestatus.History, error)
}

// StatefulDetector is a detector that needs the per-unit status
// history. It receives the current snapshot, a context, an API to
// query history with, and the reference time.
//
// Stateful detectors run AFTER the pure detectors in the pipeline.
// If a stateful detector returns an error or its API call fails, the
// command logs a warning to stderr and continues with the findings
// it has -- per constitution Principle VI (graceful degradation).
type StatefulDetector func(
	ctx context.Context,
	api statusHistoryAPI,
	status *params.FullStatus,
	now time.Time,
) ([]Finding, error)

var statefulDetectors = []StatefulDetector{
	detectStatusChurn,
	detectStuckMaintenance,
	detectAgentError,
}

// runDetectors dispatches every pure detector against the supplied
// status and returns the accumulated findings. The status pointer is
// permitted to be nil only if it is also empty; detectors defend
// against nil maps and zero-value substructures internally.
func runDetectors(status *params.FullStatus, now time.Time) []Finding {
	if status == nil {
		return nil
	}
	var out []Finding
	for _, d := range detectors {
		out = append(out, d(status, now)...)
	}
	return out
}

// runStatefulDetectors dispatches every stateful detector. Errors from
// individual detectors are accumulated and returned alongside any
// findings that DID succeed; callers MUST treat error as advisory
// (log to stderr) and proceed with the partial findings.
func runStatefulDetectors(
	ctx context.Context,
	api statusHistoryAPI,
	status *params.FullStatus,
	now time.Time,
) ([]Finding, error) {
	if status == nil || api == nil {
		return nil, nil
	}
	var out []Finding
	var firstErr error
	for _, d := range statefulDetectors {
		findings, err := d(ctx, api, status, now)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		out = append(out, findings...)
	}
	return out, firstErr
}

// ----------------------------------------------------------------------
// Signal 1: active-with-message
// ----------------------------------------------------------------------

const (
	checkActiveWithMessage   = "active-with-message"
	// protocolActiveEmptyMsg cites the brief's §4c.2 Status protocol
	// violation symptom enumerated at citizenship-observatory-brief.md:282
	// ("`active` with a non-empty message").
	protocolActiveEmptyMsg            = "protocol://citizenship/4c.2#active-with-message"
	summaryActiveWithMessage          = "Unit reports 'active' with a non-empty status message."
	recommendActiveWithMessageDefault = "Convention is that 'active' carries no message; the empty string is the visual signal of normal operation."
)

// detectActiveWithMessage emits an info finding per unit whose
// workload status is "active" with a non-empty Info message. Owner is
// charm-author. Defends against nil Applications, nil Units, and
// zero-value WorkloadStatus per FR-009.
func detectActiveWithMessage(status *params.FullStatus, _ time.Time) []Finding {
	if status == nil || status.Applications == nil {
		return nil
	}
	var out []Finding
	for appName, app := range status.Applications {
		_ = appName
		if app.Units == nil {
			continue
		}
		for unitName, u := range app.Units {
			if u.WorkloadStatus.Status != "active" {
				continue
			}
			if u.WorkloadStatus.Info == "" {
				continue
			}
			out = append(out, newFinding(
				checkActiveWithMessage,
				SeverityInfo,
				unitName,
				EntityKindUnit,
				OwnerCharmAuthor,
				summaryActiveWithMessage,
				recommendActiveWithMessageDefault,
				protocolActiveEmptyMsg,
			))
		}
	}
	return out
}

// ----------------------------------------------------------------------
// Signal 4: blocked-no-message
// ----------------------------------------------------------------------

const (
	checkBlockedNoMessage = "blocked-no-message"
	// protocolBlockedNoMessage cites the brief's §4c.2 Status protocol:
	// line 275 establishes that 'blocked' = "explicit human action
	// required" and line 283 enumerates "blocked with empty /
	// unactionable message" as the canonical violation symptom.
	protocolBlockedNoMessage           = "protocol://citizenship/4c.2#blocked-no-message"
	summaryBlockedNoMessage            = "Unit is blocked without an actionable status message."
	recommendBlockedNoMessageDefault   = "Blocked units MUST carry an actionable message identifying what human action is required. Update the charm's status-set on the blocked path to describe the unmet precondition (missing config, unmet relation, exhausted resource)."
)

// detectBlockedNoMessage emits a warning finding per unit whose
// workload status is "blocked" AND whose Info message is empty or
// whitespace-only. Whitespace-only counts as no-message here because
// the operator cannot act on it -- this is the inverse of the Signal-1
// edge case where whitespace counts as a misleading message.
// Owner: charm-author. Defends against nil Applications / Units.
func detectBlockedNoMessage(status *params.FullStatus, _ time.Time) []Finding {
	if status == nil || status.Applications == nil {
		return nil
	}
	var out []Finding
	for _, app := range status.Applications {
		if app.Units == nil {
			continue
		}
		for unitName, u := range app.Units {
			if u.WorkloadStatus.Status != "blocked" {
				continue
			}
			if strings.TrimSpace(u.WorkloadStatus.Info) != "" {
				continue
			}
			out = append(out, newFinding(
				checkBlockedNoMessage,
				SeverityWarning,
				unitName,
				EntityKindUnit,
				OwnerCharmAuthor,
				summaryBlockedNoMessage,
				recommendBlockedNoMessageDefault,
				protocolBlockedNoMessage,
			))
		}
	}
	return out
}

// ----------------------------------------------------------------------
// Signal 2: charm-revision-aging
// ----------------------------------------------------------------------

const (
	checkCharmRevisionAging = "charm-revision-aging"
	// protocolRevisionTrackChannel cites the brief's §4b inventory
	// rather than §4c: charm-revision aging is an operator-hygiene
	// signal in the 33-signal catalogue, not a citizenship-contract
	// clause violation (Principle V uses "existing clause" loosely).
	protocolRevisionTrackChannel       = "protocol://citizenship/4b#charm-revision-aging"
	summaryCharmRevisionAging          = "Application is behind its tracked channel."
	recommendCharmRevisionAgingDefault = "Run 'juju refresh <app>' to pick up the newer revision available on the tracked channel."
)

// detectCharmRevisionAging emits a warning finding per application
// whose CanUpgradeTo field is non-empty. Owner is operator. Empty
// CanUpgradeTo (the default zero value) is treated as no finding,
// never a panic.
func detectCharmRevisionAging(status *params.FullStatus, _ time.Time) []Finding {
	if status == nil || status.Applications == nil {
		return nil
	}
	var out []Finding
	for appName, app := range status.Applications {
		if app.CanUpgradeTo == "" {
			continue
		}
		out = append(out, newFinding(
			checkCharmRevisionAging,
			SeverityWarning,
			appName,
			EntityKindApplication,
			OwnerOperator,
			summaryCharmRevisionAging,
			recommendCharmRevisionAgingDefault,
			protocolRevisionTrackChannel,
		))
	}
	return out
}

// ----------------------------------------------------------------------
// Signal 3: unit-blocked-stale
// ----------------------------------------------------------------------

const (
	checkUnitBlockedStale = "unit-blocked-stale"
	// protocolBlockedBounded cites the brief's §4c.2 Status protocol:
	// line 275 establishes that 'blocked' = "explicit human action
	// required". A prolonged blocked state is a §4c.2 derivation --
	// the human action has gone unaddressed.
	protocolBlockedBounded           = "protocol://citizenship/4c.2#blocked-bounded"
	summaryUnitBlockedStaleWarning   = "Unit has been blocked for over 24 hours."
	summaryUnitBlockedStaleCritical  = "Unit has been blocked for over 7 days."
	recommendUnitBlockedStaleDefault = "Investigate blocking condition: read the charm's hook message, check peer state, or determine whether operator intervention is required."

	blockedWarningThreshold  = 24 * time.Hour
	blockedCriticalThreshold = 7 * 24 * time.Hour
)

// detectUnitBlockedStale emits a finding per unit whose workload
// status is "blocked" and whose Since timestamp is more than 24h
// before now. Severity is warning for durations in (24h, 7d] and
// critical for durations beyond 7d. Owner is mixed. nil Since pointer
// or future-dated Since (clock skew) yield zero findings.
func detectUnitBlockedStale(status *params.FullStatus, now time.Time) []Finding {
	if status == nil || status.Applications == nil {
		return nil
	}
	var out []Finding
	for _, app := range status.Applications {
		if app.Units == nil {
			continue
		}
		for unitName, u := range app.Units {
			if u.WorkloadStatus.Status != "blocked" {
				continue
			}
			since := u.WorkloadStatus.Since
			if since == nil {
				continue
			}
			age := now.Sub(*since)
			if age <= blockedWarningThreshold {
				continue
			}
			severity := SeverityWarning
			summary := summaryUnitBlockedStaleWarning
			if age > blockedCriticalThreshold {
				severity = SeverityCritical
				summary = summaryUnitBlockedStaleCritical
			}
			out = append(out, newFinding(
				checkUnitBlockedStale,
				severity,
				unitName,
				EntityKindUnit,
				OwnerMixed,
				summary,
				recommendUnitBlockedStaleDefault,
				protocolBlockedBounded,
			))
		}
	}
	return out
}

// ----------------------------------------------------------------------
// Signal 5: status-churn (stateful)
// ----------------------------------------------------------------------

const (
	checkStatusChurn = "status-churn"
	// protocolStatusChurn cites the brief's §4c.2 Status protocol:
	// status enum semantics imply stability of the declared state.
	// Rapid flips between workload statuses ("status churn") undermine
	// the contract by making every status read potentially stale.
	protocolStatusChurn         = "protocol://citizenship/4c.2#status-churn"
	summaryStatusChurn          = "Unit workload status is churning between values."
	recommendStatusChurnDefault = "A unit that flips workload status repeatedly within minutes signals an indecisive reconciler. Audit the charm's status-set logic: check whether collect_unit_status emits different values across consecutive invocations, whether a precondition oscillates, or whether multiple handlers race to set conflicting states."

	// statusChurnWindow is how far back we look for transitions.
	statusChurnWindow = 10 * time.Minute
	// statusChurnMinTransitions is the threshold for flagging churn.
	// A normal charm typically transitions 1-2 times during install
	// then stays put. 3+ transitions in the window is anomalous.
	statusChurnMinTransitions = 3
)

// detectStatusChurn queries each unit's workload-status history and
// flags units whose workload status changes 3+ times in the last
// 10 minutes. The 5+ minute defaults of update-status-hook-interval
// mean a steady charm contributes at most 1-2 history entries in
// this window; anything above 3 distinct transitions indicates the
// charm is indecisive.
//
// API errors from StatusHistory for a single unit are skipped (we
// just don't flag that unit). A model-wide history outage returns
// the first error so the caller can log a warning.
func detectStatusChurn(
	ctx context.Context,
	api statusHistoryAPI,
	status *params.FullStatus,
	now time.Time,
) ([]Finding, error) {
	if status == nil || status.Applications == nil || api == nil {
		return nil, nil
	}
	var out []Finding
	var firstErr error
	for _, app := range status.Applications {
		if app.Units == nil {
			continue
		}
		for unitName := range app.Units {
			tag := names.NewUnitTag(unitName)
			history, err := api.StatusHistory(
				ctx,
				corestatus.KindWorkload,
				tag,
				corestatus.StatusHistoryFilter{Size: 30},
			)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			if !isChurning(history, statusChurnWindow, now, statusChurnMinTransitions) {
				continue
			}
			f := newFinding(
				checkStatusChurn,
				SeverityWarning,
				unitName,
				EntityKindUnit,
				OwnerCharmAuthor,
				summaryStatusChurn,
				recommendStatusChurnDefault,
				protocolStatusChurn,
			)
			if t := oldestInWindow(history, now.Add(-statusChurnWindow)); !t.IsZero() {
				f = f.withSince(t)
			}
			out = append(out, f)
		}
	}
	return out, firstErr
}

// isChurning returns true if `history` contains at least
// `minTransitions` distinct status values in entries whose Since is
// within `window` of `now`. Entries with nil Since are skipped.
func isChurning(history corestatus.History, window time.Duration, now time.Time, minTransitions int) bool {
	cutoff := now.Add(-window)
	var prev corestatus.Status
	transitions := 0
	for i := len(history) - 1; i >= 0; i-- {
		h := history[i]
		if h.Since == nil || h.Since.Before(cutoff) {
			continue
		}
		if h.Status != prev && prev != "" {
			transitions++
		}
		prev = h.Status
	}
	return transitions >= minTransitions
}

// ----------------------------------------------------------------------
// Signal 6: stuck-maintenance (stateful)
// ----------------------------------------------------------------------

const (
	checkStuckMaintenance = "stuck-maintenance"
	// protocolStuckMaintenance cites the brief's §4c.2 Status protocol:
	// "maintenance means long-running non-error work in progress"
	// (brief line 277). Holding maintenance indefinitely violates the
	// implicit "bounded" contract -- operators have no signal that the
	// charm is actually idle vs actively working.
	protocolStuckMaintenance         = "protocol://citizenship/4c.2#stuck-maintenance"
	summaryStuckMaintenance          = "Unit has held maintenance status without transition."
	recommendStuckMaintenanceDefault = "Maintenance is for long-running but bounded work. A unit holding maintenance status without ever transitioning out signals either a stuck task, a charm that never sets a terminal status, or a misclassified state. Audit the reconciler: identify what should mark the work complete and transition out of maintenance, or move the message into the active/waiting state if the work was always idle."

	// stuckMaintenanceThreshold flags units that have been continuously
	// in maintenance for longer than this. Production default is 30
	// minutes; demo deployments may lower it via a future config knob.
	stuckMaintenanceThreshold = 5 * time.Minute
)

// detectStuckMaintenance queries each unit currently in `maintenance`
// status and computes the duration of the unit's current "continuous
// run" of maintenance entries in workload-status history. If that
// run exceeds the threshold, the unit is flagged.
//
// The Since field on the live WorkloadStatus snapshot is unreliable
// for this signal because Juju updates it on every re-emission of
// the same status value -- so a charm that re-emits the same
// maintenance message on every reconcile keeps a perpetually-fresh
// Since. Walking the history backwards from the most recent entry
// gives us the true transition timestamp.
func detectStuckMaintenance(
	ctx context.Context,
	api statusHistoryAPI,
	status *params.FullStatus,
	now time.Time,
) ([]Finding, error) {
	if status == nil || status.Applications == nil || api == nil {
		return nil, nil
	}
	var out []Finding
	var firstErr error
	for _, app := range status.Applications {
		if app.Units == nil {
			continue
		}
		for unitName, u := range app.Units {
			if u.WorkloadStatus.Status != "maintenance" {
				continue
			}
			tag := names.NewUnitTag(unitName)
			history, err := api.StatusHistory(
				ctx,
				corestatus.KindWorkload,
				tag,
				corestatus.StatusHistoryFilter{Size: 50},
			)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			run := continuousRunStart(history, corestatus.Maintenance, now)
			if run.IsZero() {
				continue
			}
			if now.Sub(run) <= stuckMaintenanceThreshold {
				continue
			}
			out = append(out, newFinding(
				checkStuckMaintenance,
				SeverityWarning,
				unitName,
				EntityKindUnit,
				OwnerCharmAuthor,
				summaryStuckMaintenance,
				recommendStuckMaintenanceDefault,
				protocolStuckMaintenance,
			).withSince(run))
		}
	}
	return out, firstErr
}

// ----------------------------------------------------------------------
// Signal 7: hook-error (stateful)
// ----------------------------------------------------------------------

const (
	checkHookError = "hook-error"
	// protocolHookError cites the brief's §4c.1 Hook firing protocol:
	// hooks MUST complete; uncaught exceptions interrupt the firing
	// protocol and drive the unit to error status. A unit observed in
	// agent-status error within the recent window is a direct violation
	// of that contract.
	protocolHookError         = "protocol://citizenship/4c.1#hook-error"
	summaryHookError          = "Unit hit an uncaught hook error recently."
	recommendHookErrorDefault = "Hook failures interrupt the firing protocol (§4c.1). Inspect 'juju debug-log --include <unit>' and 'juju show-status-log --type=juju-unit <unit>' for the failing hook. Recover with 'juju resolve <unit>' after fixing the underlying charm bug."

	// hookErrorWindow is how far back we look for agent-status error
	// entries. A single error inside the window is enough to flag.
	hookErrorWindow = 30 * time.Minute
)

// detectAgentError queries each unit's agent-status history (NOT
// workload-status) and flags units that have any entry with
// Status == corestatus.Error whose Since falls within the last
// hookErrorWindow. Severity is warning because the unit is recoverable
// via 'juju resolve'; owner is charm-author because uncaught
// exceptions are charm-code bugs.
//
// One finding is emitted per unit regardless of how many error entries
// fall in the window -- the existence of any recent error is the
// signal, not the count.
//
// API errors from StatusHistory for a single unit are skipped (we
// just don't flag that unit). A model-wide history outage returns
// the first error so the caller can log a warning.
func detectAgentError(
	ctx context.Context,
	api statusHistoryAPI,
	status *params.FullStatus,
	now time.Time,
) ([]Finding, error) {
	if status == nil || status.Applications == nil || api == nil {
		return nil, nil
	}
	cutoff := now.Add(-hookErrorWindow)
	var out []Finding
	var firstErr error
	for _, app := range status.Applications {
		if app.Units == nil {
			continue
		}
		for unitName := range app.Units {
			tag := names.NewUnitTag(unitName)
			history, err := api.StatusHistory(
				ctx,
				corestatus.KindUnitAgent,
				tag,
				corestatus.StatusHistoryFilter{Size: 50},
			)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			errSince := oldestErrorInWindow(history, cutoff)
			if errSince.IsZero() {
				continue
			}
			out = append(out, newFinding(
				checkHookError,
				SeverityWarning,
				unitName,
				EntityKindUnit,
				OwnerCharmAuthor,
				summaryHookError,
				recommendHookErrorDefault,
				protocolHookError,
			).withSince(errSince))
		}
	}
	return out, firstErr
}

// hasRecentError returns true if `history` contains at least one entry
// whose Status is corestatus.Error and whose Since is non-nil and
// not before `cutoff`. Entries with nil Since are skipped defensively.
func hasRecentError(history corestatus.History, cutoff time.Time) bool {
	return !oldestErrorInWindow(history, cutoff).IsZero()
}

// oldestErrorInWindow returns the Since timestamp of the OLDEST
// corestatus.Error entry in `history` whose Since is non-nil and at
// or after `cutoff`. Returns zero time if no such entry exists.
func oldestErrorInWindow(history corestatus.History, cutoff time.Time) time.Time {
	var oldest time.Time
	for _, h := range history {
		if h.Status != corestatus.Error {
			continue
		}
		if h.Since == nil {
			continue
		}
		if h.Since.Before(cutoff) {
			continue
		}
		if oldest.IsZero() || h.Since.Before(oldest) {
			oldest = *h.Since
		}
	}
	return oldest
}

// oldestInWindow returns the Since timestamp of the OLDEST entry in
// `history` whose Since is non-nil and at or after `cutoff`. Returns
// zero time if no such entry exists. Used by stateful detectors that
// need a "violation began at" timestamp.
func oldestInWindow(history corestatus.History, cutoff time.Time) time.Time {
	var oldest time.Time
	for _, h := range history {
		if h.Since == nil {
			continue
		}
		if h.Since.Before(cutoff) {
			continue
		}
		if oldest.IsZero() || h.Since.Before(oldest) {
			oldest = *h.Since
		}
	}
	return oldest
}

// ----------------------------------------------------------------------
// Signal 8: entity-stuck-dying
// ----------------------------------------------------------------------

const (
	checkEntityStuckDying = "entity-stuck-dying"
	// protocolEntityStuckDying cites the brief's §4c.1 Hook firing
	// protocol: teardown completeness is part of the firing-protocol
	// contract. An entity stuck in Dying signals a teardown hook that
	// never returns success.
	protocolEntityStuckDying        = "protocol://citizenship/4c.1#entity-stuck-dying"
	summaryEntityStuckDyingWarning  = "Entity has been in Dying state for over 5 minutes."
	summaryEntityStuckDyingCritical = "Entity has been in Dying state for over 1 hour."
	recommendEntityStuckDyingDefault = "An entity that lingers in Dying typically signals a failing teardown hook (relation-departed/relation-broken/stop). Inspect the unit's debug-log and recent agent-status history. Avoid 'juju resolve --force' on the broken hook -- it can leave substrate resources orphaned. Fix the teardown path in charm code, then 'juju resolve' to retry."

	entityDyingWarningThreshold  = 5 * time.Minute
	entityDyingCriticalThreshold = 1 * time.Hour
)

// detectEntityStuckDying emits a finding per application or unit whose
// Life is "dying" and whose transition timestamp is older than the
// warning threshold. Severity is warning for ages in (5m, 1h] and
// critical for ages beyond 1h. Owner is mixed -- the underlying defect
// is usually a charm teardown bug, but operators must intervene via
// 'juju resolve' or '--force' to actually unstick the entity. nil or
// future-dated transition timestamps yield zero findings.
//
// Application transition timestamp is taken from app.Status.Since.
// Unit transition timestamp prefers WorkloadStatus.Since and falls back
// to AgentStatus.Since when the former is nil. The Since field on the
// resulting Finding records the transition time so the AGE column
// populates correctly.
func detectEntityStuckDying(status *params.FullStatus, now time.Time) []Finding {
	if status == nil || status.Applications == nil {
		return nil
	}
	var out []Finding
	for appName, app := range status.Applications {
		if string(app.Life) == "dying" {
			if f, ok := dyingFinding(appName, EntityKindApplication, app.Status.Since, now); ok {
				out = append(out, f)
			}
		}
		if app.Units == nil {
			continue
		}
		for unitName, u := range app.Units {
			if string(u.WorkloadStatus.Life) != "dying" {
				continue
			}
			since := u.WorkloadStatus.Since
			if since == nil {
				since = u.AgentStatus.Since
			}
			if f, ok := dyingFinding(unitName, EntityKindUnit, since, now); ok {
				out = append(out, f)
			}
		}
	}
	return out
}

// dyingFinding builds the warning/critical Finding for a Dying entity,
// or returns ok=false when the transition timestamp is nil, future, or
// not yet old enough to flag.
func dyingFinding(entity string, kind EntityKind, since *time.Time, now time.Time) (Finding, bool) {
	if since == nil {
		return Finding{}, false
	}
	age := now.Sub(*since)
	if age <= entityDyingWarningThreshold {
		return Finding{}, false
	}
	severity := SeverityWarning
	summary := summaryEntityStuckDyingWarning
	if age > entityDyingCriticalThreshold {
		severity = SeverityCritical
		summary = summaryEntityStuckDyingCritical
	}
	return newFinding(
		checkEntityStuckDying,
		severity,
		entity,
		kind,
		OwnerMixed,
		summary,
		recommendEntityStuckDyingDefault,
		protocolEntityStuckDying,
	).withSince(*since), true
}

// ----------------------------------------------------------------------
// Signal 9: model-suspended-credential
// ----------------------------------------------------------------------

const (
	checkModelSuspendedCredential = "model-suspended-credential"
	// protocolModelSuspendedCredential cites the brief's §4b operational
	// hygiene inventory: a suspended model represents a model-wide
	// outage that silently halts all workload commands.
	protocolModelSuspendedCredential         = "protocol://citizenship/4b#model-suspended-credential"
	summaryModelSuspendedCredential          = "Model is suspended -- cloud credential validation failed."
	recommendModelSuspendedCredentialDefault = "All workload commands will hang indefinitely until the cloud credential is restored. Rotate the credential and re-attach: 'juju add-credential', 'juju update-credential', or contact the cloud admin if a service principal expired. The --force flag does NOT bypass model suspension."
)

// detectModelSuspendedCredential emits at most one critical finding
// when the model's ModelStatus.Status is "suspended" -- almost always
// because cloud credential validation failed. Owner is operator
// (credential rotation is operator-side). EntityKind is "model" and
// the entity name is the model name from status.Model.Name; if empty
// for any reason, "<model>" is used defensively.
func detectModelSuspendedCredential(status *params.FullStatus, _ time.Time) []Finding {
	if status == nil {
		return nil
	}
	if status.Model.ModelStatus.Status != "suspended" {
		return nil
	}
	entity := status.Model.Name
	if entity == "" {
		entity = "<model>"
	}
	f := newFinding(
		checkModelSuspendedCredential,
		SeverityCritical,
		entity,
		EntityKindModel,
		OwnerOperator,
		summaryModelSuspendedCredential,
		recommendModelSuspendedCredentialDefault,
		protocolModelSuspendedCredential,
	)
	if since := status.Model.ModelStatus.Since; since != nil {
		f = f.withSince(*since)
	}
	return []Finding{f}
}

// continuousRunStart returns the Since timestamp of the OLDEST entry
// in the contiguous run of `target` status that ends at the most
// recent entry. Returns the zero time if the most recent entry isn't
// `target` or if history is empty.
//
// Status history ordering from Juju is not contractually specified
// (live observation: oldest-first; tests historically: newest-first),
// so we sort newest-first explicitly before walking. Then we walk
// from newest to older, tracking the timestamp of each contiguous
// `target` entry. We stop at the first non-target entry; the last
// `oldest` value at that point is the start of the current run.
func continuousRunStart(history corestatus.History, target corestatus.Status, now time.Time) time.Time {
	if len(history) == 0 {
		return time.Time{}
	}
	sorted := make(corestatus.History, len(history))
	copy(sorted, history)
	sort.SliceStable(sorted, func(i, j int) bool {
		var ti, tj time.Time
		if sorted[i].Since != nil {
			ti = *sorted[i].Since
		}
		if sorted[j].Since != nil {
			tj = *sorted[j].Since
		}
		return ti.After(tj)
	})
	var oldest time.Time
	for _, h := range sorted {
		if h.Status != target {
			break
		}
		if h.Since != nil {
			oldest = *h.Since
		}
	}
	return oldest
}
