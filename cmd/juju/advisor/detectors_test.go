// Copyright 2026 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package advisor_test

import (
	"context"
	stdtesting "testing"
	"time"

	"github.com/juju/names/v6"
	"github.com/juju/tc"

	"github.com/juju/juju/cmd/juju/advisor"
	"github.com/juju/juju/core/life"
	corestatus "github.com/juju/juju/core/status"
	"github.com/juju/juju/rpc/params"
)

func TestDetectorsSuite(t *stdtesting.T) {
	tc.Run(t, &detectorsSuite{})
}

type detectorsSuite struct{}

// referenceTime pins all blocked-stale calculations to a known
// instant. Tests dial in the unit's Since timestamp relative to this.
var referenceTime = time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC)

func cleanStatus() *params.FullStatus {
	return &params.FullStatus{
		Applications: map[string]params.ApplicationStatus{},
	}
}

// statusWithUnit returns a status containing a single unit with the
// given workload status fields.
func statusWithUnit(app, unit, workloadState, info string, since *time.Time) *params.FullStatus {
	return &params.FullStatus{
		Applications: map[string]params.ApplicationStatus{
			app: {
				Units: map[string]params.UnitStatus{
					unit: {
						WorkloadStatus: params.DetailedStatus{
							Status: workloadState,
							Info:   info,
							Since:  since,
						},
					},
				},
			},
		},
	}
}

// statusWithUpgradeable returns a status containing a single
// application with a non-empty CanUpgradeTo.
func statusWithUpgradeable(app, target string) *params.FullStatus {
	return &params.FullStatus{
		Applications: map[string]params.ApplicationStatus{
			app: {CanUpgradeTo: target},
		},
	}
}

// ------------------------------------------------------------------
// Signal 1: active-with-message
// ------------------------------------------------------------------

func (s *detectorsSuite) TestActiveWithMessageClean(c *tc.C) {
	findings := advisor.RunDetectors(cleanStatus(), referenceTime)
	c.Check(findings, tc.HasLen, 0)
}

func (s *detectorsSuite) TestActiveWithMessageMatches(c *tc.C) {
	st := statusWithUnit("grafana-k8s", "grafana-k8s/0", "active", "ready", nil)
	findings := advisor.RunDetectors(st, referenceTime)
	// Exactly one info finding from the active-with-message detector;
	// charm-revision-aging emits zero (no CanUpgradeTo);
	// unit-blocked-stale emits zero (not blocked).
	c.Assert(findings, tc.HasLen, 1)
	f := findings[0]
	c.Check(f.CheckID, tc.Equals, "active-with-message")
	c.Check(f.Severity, tc.Equals, advisor.SeverityInfo)
	c.Check(f.Owner, tc.Equals, advisor.OwnerCharmAuthor)
	c.Check(f.EntityKind, tc.Equals, advisor.EntityKindUnit)
	c.Check(f.Entity, tc.Equals, "grafana-k8s/0")
}

func (s *detectorsSuite) TestActiveWithMessageWhitespaceCountsAsMessage(c *tc.C) {
	// Edge case from spec: whitespace-only message is not "empty".
	st := statusWithUnit("noisy", "noisy/0", "active", "   ", nil)
	findings := advisor.RunDetectors(st, referenceTime)
	c.Check(findings, tc.HasLen, 1)
	c.Check(findings[0].CheckID, tc.Equals, "active-with-message")
}

func (s *detectorsSuite) TestActiveWithEmptyMessageNoFinding(c *tc.C) {
	st := statusWithUnit("quiet", "quiet/0", "active", "", nil)
	findings := advisor.RunDetectors(st, referenceTime)
	c.Check(findings, tc.HasLen, 0)
}

// ------------------------------------------------------------------
// Signal 4: blocked-no-message
// ------------------------------------------------------------------

func (s *detectorsSuite) TestBlockedNoMessageEmpty(c *tc.C) {
	st := statusWithUnit("a", "a/0", "blocked", "", nil)
	findings := advisor.RunDetectors(st, referenceTime)
	c.Assert(findings, tc.HasLen, 1)
	f := findings[0]
	c.Check(f.CheckID, tc.Equals, "blocked-no-message")
	c.Check(f.Severity, tc.Equals, advisor.SeverityWarning)
	c.Check(f.Owner, tc.Equals, advisor.OwnerCharmAuthor)
	c.Check(f.EntityKind, tc.Equals, advisor.EntityKindUnit)
	c.Check(f.Entity, tc.Equals, "a/0")
}

func (s *detectorsSuite) TestBlockedNoMessageWhitespaceCountsAsEmpty(c *tc.C) {
	// Symmetric edge case: whitespace-only counts as "no actionable
	// message" because the operator cannot act on it.
	st := statusWithUnit("a", "a/0", "blocked", "   \t  ", nil)
	findings := advisor.RunDetectors(st, referenceTime)
	c.Assert(findings, tc.HasLen, 1)
	c.Check(findings[0].CheckID, tc.Equals, "blocked-no-message")
}

func (s *detectorsSuite) TestBlockedWithMessageNoFinding(c *tc.C) {
	st := statusWithUnit("a", "a/0", "blocked", "missing required config", nil)
	findings := advisor.RunDetectors(st, referenceTime)
	// Blocked with an actionable message is GOOD behavior -- the
	// detector must NOT fire. Since is nil so unit-blocked-stale also
	// emits zero. Total: 0 findings.
	c.Check(findings, tc.HasLen, 0)
}

// ------------------------------------------------------------------
// Signal 2: charm-revision-aging
// ------------------------------------------------------------------

func (s *detectorsSuite) TestCharmRevisionAgingClean(c *tc.C) {
	findings := advisor.RunDetectors(cleanStatus(), referenceTime)
	c.Check(findings, tc.HasLen, 0)
}

func (s *detectorsSuite) TestCharmRevisionAgingMatches(c *tc.C) {
	st := statusWithUpgradeable("postgresql-k8s", "ch:postgresql-k8s-42")
	findings := advisor.RunDetectors(st, referenceTime)
	c.Assert(findings, tc.HasLen, 1)
	f := findings[0]
	c.Check(f.CheckID, tc.Equals, "charm-revision-aging")
	c.Check(f.Severity, tc.Equals, advisor.SeverityWarning)
	c.Check(f.Owner, tc.Equals, advisor.OwnerOperator)
	c.Check(f.EntityKind, tc.Equals, advisor.EntityKindApplication)
	c.Check(f.Entity, tc.Equals, "postgresql-k8s")
}

// ------------------------------------------------------------------
// Signal 3: unit-blocked-stale (boundary tests)
// ------------------------------------------------------------------

func (s *detectorsSuite) TestUnitBlockedStaleUnderThreshold(c *tc.C) {
	// 23h59m59s blocked -> no finding.
	since := referenceTime.Add(-(23*time.Hour + 59*time.Minute + 59*time.Second))
	st := statusWithUnit("a", "a/0", "blocked", "waiting", &since)
	findings := advisor.RunDetectors(st, referenceTime)
	c.Check(findings, tc.HasLen, 0)
}

func (s *detectorsSuite) TestUnitBlockedStaleJustOverWarning(c *tc.C) {
	// 24h0m1s blocked -> warning.
	since := referenceTime.Add(-(24*time.Hour + time.Second))
	st := statusWithUnit("a", "a/0", "blocked", "waiting", &since)
	findings := advisor.RunDetectors(st, referenceTime)
	c.Assert(findings, tc.HasLen, 1)
	c.Check(findings[0].Severity, tc.Equals, advisor.SeverityWarning)
	c.Check(findings[0].CheckID, tc.Equals, "unit-blocked-stale")
}

func (s *detectorsSuite) TestUnitBlockedStaleAtSevenDayBoundary(c *tc.C) {
	// Exactly 7d blocked -> still warning (per cli-contract: (24h, 7d] = warning).
	since := referenceTime.Add(-(7 * 24 * time.Hour))
	st := statusWithUnit("a", "a/0", "blocked", "waiting", &since)
	findings := advisor.RunDetectors(st, referenceTime)
	c.Assert(findings, tc.HasLen, 1)
	c.Check(findings[0].Severity, tc.Equals, advisor.SeverityWarning)
}

func (s *detectorsSuite) TestUnitBlockedStaleJustOverCritical(c *tc.C) {
	// 7d+1s blocked -> critical.
	since := referenceTime.Add(-(7*24*time.Hour + time.Second))
	st := statusWithUnit("a", "a/0", "blocked", "waiting", &since)
	findings := advisor.RunDetectors(st, referenceTime)
	c.Assert(findings, tc.HasLen, 1)
	c.Check(findings[0].Severity, tc.Equals, advisor.SeverityCritical)
}

func (s *detectorsSuite) TestUnitBlockedStaleNilSince(c *tc.C) {
	// Defensive: nil Since must not panic, must yield zero findings.
	st := statusWithUnit("a", "a/0", "blocked", "waiting", nil)
	findings := advisor.RunDetectors(st, referenceTime)
	c.Check(findings, tc.HasLen, 0)
}

func (s *detectorsSuite) TestUnitBlockedStaleFutureSince(c *tc.C) {
	// Clock skew: Since in the future -> no finding (per spec edge case).
	since := referenceTime.Add(1 * time.Minute)
	st := statusWithUnit("a", "a/0", "blocked", "waiting", &since)
	findings := advisor.RunDetectors(st, referenceTime)
	c.Check(findings, tc.HasLen, 0)
}

// ------------------------------------------------------------------
// Defensive: nil inputs
// ------------------------------------------------------------------

func (s *detectorsSuite) TestNilStatus(c *tc.C) {
	findings := advisor.RunDetectors(nil, referenceTime)
	c.Check(findings, tc.HasLen, 0)
}

func (s *detectorsSuite) TestNilApplicationsMap(c *tc.C) {
	st := &params.FullStatus{Applications: nil}
	findings := advisor.RunDetectors(st, referenceTime)
	c.Check(findings, tc.HasLen, 0)
}

func (s *detectorsSuite) TestNilUnitsMap(c *tc.C) {
	st := &params.FullStatus{
		Applications: map[string]params.ApplicationStatus{
			"a": {Units: nil},
		},
	}
	findings := advisor.RunDetectors(st, referenceTime)
	c.Check(findings, tc.HasLen, 0)
}

// ------------------------------------------------------------------
// Signal 5: status-churn (stateful)
// ------------------------------------------------------------------

// fakeHistoryAPI returns a canned history per unit tag.
type fakeHistoryAPI struct {
	byUnit map[string]corestatus.History
	err    error
}

func (f *fakeHistoryAPI) StatusHistory(
	ctx context.Context,
	kind corestatus.HistoryKind,
	tag names.Tag,
	filter corestatus.StatusHistoryFilter,
) (corestatus.History, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.byUnit[tag.Id()], nil
}

// historyEntry helper builds a corestatus.DetailedStatus with a
// Since timestamp relative to referenceTime.
func historyEntry(s corestatus.Status, agoSec int) corestatus.DetailedStatus {
	since := referenceTime.Add(-time.Duration(agoSec) * time.Second)
	return corestatus.DetailedStatus{
		Status: s,
		Since:  &since,
		Kind:   corestatus.KindWorkload,
	}
}

func (s *detectorsSuite) TestStatusChurnFlagsChurningUnit(c *tc.C) {
	st := statusWithUnit("churn", "churn/0", "active", "", nil)
	api := &fakeHistoryAPI{byUnit: map[string]corestatus.History{
		"churn/0": {
			// 5 transitions in the last 5 minutes -> churn
			historyEntry(corestatus.Active, 30),
			historyEntry(corestatus.Waiting, 60),
			historyEntry(corestatus.Active, 90),
			historyEntry(corestatus.Waiting, 120),
			historyEntry(corestatus.Active, 150),
			historyEntry(corestatus.Waiting, 180),
		},
	}}
	findings, err := advisor.RunStatefulDetectors(context.Background(), api, st, referenceTime)
	c.Assert(err, tc.ErrorIsNil)
	c.Assert(findings, tc.HasLen, 1)
	c.Check(findings[0].CheckID, tc.Equals, "status-churn")
	c.Check(findings[0].Severity, tc.Equals, advisor.SeverityWarning)
	c.Check(findings[0].Owner, tc.Equals, advisor.OwnerCharmAuthor)
	c.Check(findings[0].Entity, tc.Equals, "churn/0")
}

func (s *detectorsSuite) TestStatusChurnNoFlagOnStableUnit(c *tc.C) {
	st := statusWithUnit("stable", "stable/0", "active", "", nil)
	api := &fakeHistoryAPI{byUnit: map[string]corestatus.History{
		"stable/0": {
			// One transition during install, then stable for hours
			historyEntry(corestatus.Maintenance, 7200),
			historyEntry(corestatus.Active, 7000),
		},
	}}
	findings, err := advisor.RunStatefulDetectors(context.Background(), api, st, referenceTime)
	c.Assert(err, tc.ErrorIsNil)
	c.Check(findings, tc.HasLen, 0)
}

func (s *detectorsSuite) TestStatusChurnIgnoresEntriesOutsideWindow(c *tc.C) {
	st := statusWithUnit("oldchurn", "oldchurn/0", "active", "", nil)
	api := &fakeHistoryAPI{byUnit: map[string]corestatus.History{
		"oldchurn/0": {
			// 5 transitions, but all >15 minutes ago (outside 10-min window)
			historyEntry(corestatus.Active, 1000),
			historyEntry(corestatus.Waiting, 1100),
			historyEntry(corestatus.Active, 1200),
			historyEntry(corestatus.Waiting, 1300),
			historyEntry(corestatus.Active, 1400),
			historyEntry(corestatus.Waiting, 1500),
		},
	}}
	findings, err := advisor.RunStatefulDetectors(context.Background(), api, st, referenceTime)
	c.Assert(err, tc.ErrorIsNil)
	c.Check(findings, tc.HasLen, 0)
}

func (s *detectorsSuite) TestStatusChurnSurfacesAPIErrorAsAdvisory(c *tc.C) {
	st := statusWithUnit("u", "u/0", "active", "", nil)
	api := &fakeHistoryAPI{err: errAPI("history disabled")}
	findings, err := advisor.RunStatefulDetectors(context.Background(), api, st, referenceTime)
	// Advisory: err returned, findings empty, callers continue.
	c.Check(err, tc.ErrorMatches, "history disabled")
	c.Check(findings, tc.HasLen, 0)
}

// errAPI is a trivial error helper.
func errAPI(msg string) error { return &apiErr{msg} }

type apiErr struct{ s string }

func (e *apiErr) Error() string { return e.s }

// ------------------------------------------------------------------
// Signal 6: stuck-maintenance (stateful)
// ------------------------------------------------------------------

func (s *detectorsSuite) TestStuckMaintenanceFlagsLongRun(c *tc.C) {
	st := statusWithUnit("stuck", "stuck/0", "maintenance", "preparing", nil)
	api := &fakeHistoryAPI{byUnit: map[string]corestatus.History{
		// History returned newest-first by Juju. All entries are
		// maintenance, oldest is 10 min ago -> longer than 5 min
		// threshold -> finding.
		"stuck/0": {
			historyEntry(corestatus.Maintenance, 30),
			historyEntry(corestatus.Maintenance, 90),
			historyEntry(corestatus.Maintenance, 300),
			historyEntry(corestatus.Maintenance, 600),
		},
	}}
	findings, err := advisor.RunStatefulDetectors(context.Background(), api, st, referenceTime)
	c.Assert(err, tc.ErrorIsNil)
	c.Assert(findings, tc.HasLen, 1)
	c.Check(findings[0].CheckID, tc.Equals, "stuck-maintenance")
	c.Check(findings[0].Severity, tc.Equals, advisor.SeverityWarning)
	c.Check(findings[0].Owner, tc.Equals, advisor.OwnerCharmAuthor)
	c.Check(findings[0].Entity, tc.Equals, "stuck/0")
}

func (s *detectorsSuite) TestStuckMaintenanceNoFlagOnShortRun(c *tc.C) {
	st := statusWithUnit("fresh", "fresh/0", "maintenance", "preparing", nil)
	api := &fakeHistoryAPI{byUnit: map[string]corestatus.History{
		// All maintenance, but oldest entry is only 2 min ago -> below
		// the 5 min threshold -> no finding.
		"fresh/0": {
			historyEntry(corestatus.Maintenance, 30),
			historyEntry(corestatus.Maintenance, 90),
			historyEntry(corestatus.Maintenance, 120),
		},
	}}
	findings, err := advisor.RunStatefulDetectors(context.Background(), api, st, referenceTime)
	c.Assert(err, tc.ErrorIsNil)
	c.Check(findings, tc.HasLen, 0)
}

func (s *detectorsSuite) TestStuckMaintenanceIgnoresNonMaintenance(c *tc.C) {
	// Unit is currently active -- detector skips it entirely.
	st := statusWithUnit("ok", "ok/0", "active", "", nil)
	api := &fakeHistoryAPI{byUnit: map[string]corestatus.History{
		"ok/0": {historyEntry(corestatus.Maintenance, 9999)},
	}}
	findings, err := advisor.RunStatefulDetectors(context.Background(), api, st, referenceTime)
	c.Assert(err, tc.ErrorIsNil)
	c.Check(findings, tc.HasLen, 0)
}

func (s *detectorsSuite) TestStuckMaintenanceHandlesOldestFirstHistory(c *tc.C) {
	// Regression test: live Juju returns status history oldest-first.
	// The detector MUST sort explicitly and not be order-dependent.
	st := statusWithUnit("stuck", "stuck/0", "maintenance", "preparing", nil)
	api := &fakeHistoryAPI{byUnit: map[string]corestatus.History{
		"stuck/0": {
			// Oldest first: waiting transient during install,
			// then maintenance for >5 min.
			historyEntry(corestatus.Waiting, 1200),
			historyEntry(corestatus.Maintenance, 600),
			historyEntry(corestatus.Maintenance, 300),
			historyEntry(corestatus.Maintenance, 90),
			historyEntry(corestatus.Maintenance, 30),
		},
	}}
	findings, err := advisor.RunStatefulDetectors(context.Background(), api, st, referenceTime)
	c.Assert(err, tc.ErrorIsNil)
	c.Assert(findings, tc.HasLen, 1)
	c.Check(findings[0].CheckID, tc.Equals, "stuck-maintenance")
}

func (s *detectorsSuite) TestStuckMaintenanceStopsAtRecentTransition(c *tc.C) {
	st := statusWithUnit("recovered", "recovered/0", "maintenance", "preparing", nil)
	api := &fakeHistoryAPI{byUnit: map[string]corestatus.History{
		// Currently in maintenance for only 2 min, but BEFORE that was
		// active for hours. The continuous-maintenance run is just the
		// last 2 min -> no finding.
		"recovered/0": {
			historyEntry(corestatus.Maintenance, 30),
			historyEntry(corestatus.Maintenance, 90),
			historyEntry(corestatus.Active, 1000),
			historyEntry(corestatus.Active, 2000),
		},
	}}
	findings, err := advisor.RunStatefulDetectors(context.Background(), api, st, referenceTime)
	c.Assert(err, tc.ErrorIsNil)
	c.Check(findings, tc.HasLen, 0)
}

// ------------------------------------------------------------------
// Signal 7: hook-error (stateful)
// ------------------------------------------------------------------

func (s *detectorsSuite) TestHookErrorFlagsRecentError(c *tc.C) {
	st := statusWithUnit("crashy", "crashy/0", "active", "", nil)
	api := &fakeHistoryAPI{byUnit: map[string]corestatus.History{
		// A single agent-status error 5 minutes ago, inside the 30-min
		// window -> finding.
		"crashy/0": {
			historyEntry(corestatus.Error, 300),
			historyEntry(corestatus.Idle, 600),
		},
	}}
	findings, err := advisor.RunStatefulDetectors(context.Background(), api, st, referenceTime)
	c.Assert(err, tc.ErrorIsNil)
	c.Assert(findings, tc.HasLen, 1)
	c.Check(findings[0].CheckID, tc.Equals, "hook-error")
	c.Check(findings[0].Severity, tc.Equals, advisor.SeverityWarning)
	c.Check(findings[0].Owner, tc.Equals, advisor.OwnerCharmAuthor)
	c.Check(findings[0].EntityKind, tc.Equals, advisor.EntityKindUnit)
	c.Check(findings[0].Entity, tc.Equals, "crashy/0")
}

func (s *detectorsSuite) TestHookErrorNoFlagOnOldError(c *tc.C) {
	st := statusWithUnit("recovered", "recovered/0", "active", "", nil)
	api := &fakeHistoryAPI{byUnit: map[string]corestatus.History{
		// Single error entry 60 min ago -> outside the 30-min window
		// -> no finding.
		"recovered/0": {
			historyEntry(corestatus.Error, 60*60),
			historyEntry(corestatus.Idle, 60*60+10),
		},
	}}
	findings, err := advisor.RunStatefulDetectors(context.Background(), api, st, referenceTime)
	c.Assert(err, tc.ErrorIsNil)
	c.Check(findings, tc.HasLen, 0)
}

func (s *detectorsSuite) TestHookErrorIgnoresNonError(c *tc.C) {
	st := statusWithUnit("healthy", "healthy/0", "active", "", nil)
	api := &fakeHistoryAPI{byUnit: map[string]corestatus.History{
		// Only idle and executing entries, no error -> no finding.
		// Entries are placed well outside the 10-minute status-churn
		// window so the unrelated status-churn detector does not fire
		// on the same fake history.
		"healthy/0": {
			historyEntry(corestatus.Idle, 1200),
			historyEntry(corestatus.Executing, 1500),
			historyEntry(corestatus.Idle, 1800),
		},
	}}
	findings, err := advisor.RunStatefulDetectors(context.Background(), api, st, referenceTime)
	c.Assert(err, tc.ErrorIsNil)
	c.Check(findings, tc.HasLen, 0)
}

// ------------------------------------------------------------------
// Signal 8: entity-stuck-dying
// ------------------------------------------------------------------

func (s *detectorsSuite) TestEntityStuckDyingWarningThreshold(c *tc.C) {
	// Unit Life=dying, transitioned 6 minutes ago -> warning finding.
	since := referenceTime.Add(-6 * time.Minute)
	st := statusWithDyingUnit("a", "a/0", &since)
	findings := advisor.RunDetectors(st, referenceTime)
	c.Assert(findings, tc.HasLen, 1)
	f := findings[0]
	c.Check(f.CheckID, tc.Equals, "entity-stuck-dying")
	c.Check(f.Severity, tc.Equals, advisor.SeverityWarning)
	c.Check(f.Owner, tc.Equals, advisor.OwnerMixed)
	c.Check(f.EntityKind, tc.Equals, advisor.EntityKindUnit)
	c.Check(f.Entity, tc.Equals, "a/0")
	c.Assert(f.Since, tc.NotNil)
	c.Check(f.Since.Equal(since), tc.IsTrue)
}

func (s *detectorsSuite) TestEntityStuckDyingCriticalThreshold(c *tc.C) {
	// Unit Life=dying, transitioned 90 minutes ago -> critical finding.
	since := referenceTime.Add(-90 * time.Minute)
	st := statusWithDyingUnit("a", "a/0", &since)
	findings := advisor.RunDetectors(st, referenceTime)
	c.Assert(findings, tc.HasLen, 1)
	c.Check(findings[0].Severity, tc.Equals, advisor.SeverityCritical)
	c.Check(findings[0].CheckID, tc.Equals, "entity-stuck-dying")
}

func (s *detectorsSuite) TestEntityStuckDyingApplicationLevel(c *tc.C) {
	// Application Life=dying for 10 min -> warning, EntityKindApplication.
	since := referenceTime.Add(-10 * time.Minute)
	st := &params.FullStatus{
		Applications: map[string]params.ApplicationStatus{
			"a": {
				Life: life.Dying,
				Status: params.DetailedStatus{
					Status: "active",
					Since:  &since,
				},
			},
		},
	}
	findings := advisor.RunDetectors(st, referenceTime)
	c.Assert(findings, tc.HasLen, 1)
	f := findings[0]
	c.Check(f.CheckID, tc.Equals, "entity-stuck-dying")
	c.Check(f.EntityKind, tc.Equals, advisor.EntityKindApplication)
	c.Check(f.Entity, tc.Equals, "a")
	c.Check(f.Severity, tc.Equals, advisor.SeverityWarning)
}

func (s *detectorsSuite) TestEntityStuckDyingNotDyingNoFinding(c *tc.C) {
	// App/unit Life=alive -> zero findings even with old Since.
	since := referenceTime.Add(-90 * time.Minute)
	st := &params.FullStatus{
		Applications: map[string]params.ApplicationStatus{
			"a": {
				Life: life.Alive,
				Status: params.DetailedStatus{
					Status: "active",
					Since:  &since,
				},
				Units: map[string]params.UnitStatus{
					"a/0": {
						WorkloadStatus: params.DetailedStatus{
							Status: "active",
							Life:   life.Alive,
							Since:  &since,
						},
					},
				},
			},
		},
	}
	findings := advisor.RunDetectors(st, referenceTime)
	c.Check(findings, tc.HasLen, 0)
}

func (s *detectorsSuite) TestEntityStuckDyingNilSinceNoFinding(c *tc.C) {
	// Unit Life=dying but Since==nil on both workload AND agent ->
	// defensive zero findings rather than a panic.
	st := statusWithDyingUnit("a", "a/0", nil)
	findings := advisor.RunDetectors(st, referenceTime)
	c.Check(findings, tc.HasLen, 0)
}

// ------------------------------------------------------------------
// Signal 9: model-suspended-credential
// ------------------------------------------------------------------

func (s *detectorsSuite) TestModelSuspendedCredentialFires(c *tc.C) {
	since := referenceTime.Add(-1 * time.Hour)
	st := statusWithSuspendedModel("prod", &since)
	findings := advisor.RunDetectors(st, referenceTime)
	c.Assert(findings, tc.HasLen, 1)
	f := findings[0]
	c.Check(f.CheckID, tc.Equals, "model-suspended-credential")
	c.Check(f.Severity, tc.Equals, advisor.SeverityCritical)
	c.Check(f.Owner, tc.Equals, advisor.OwnerOperator)
	c.Check(f.EntityKind, tc.Equals, advisor.EntityKindModel)
	c.Check(f.Entity, tc.Equals, "prod")
	c.Assert(f.Since, tc.NotNil)
	c.Check(f.Since.Equal(since), tc.IsTrue)
}

func (s *detectorsSuite) TestModelSuspendedCredentialNotSuspendedNoFinding(c *tc.C) {
	st := &params.FullStatus{
		Model: params.ModelStatusInfo{
			Name: "prod",
			ModelStatus: params.DetailedStatus{
				Status: "available",
			},
		},
		Applications: map[string]params.ApplicationStatus{},
	}
	findings := advisor.RunDetectors(st, referenceTime)
	c.Check(findings, tc.HasLen, 0)
}

// statusWithDyingUnit returns a status containing a single unit in
// Life=dying state with the given WorkloadStatus.Since.
func statusWithDyingUnit(app, unit string, since *time.Time) *params.FullStatus {
	return &params.FullStatus{
		Applications: map[string]params.ApplicationStatus{
			app: {
				Life: life.Alive,
				Units: map[string]params.UnitStatus{
					unit: {
						WorkloadStatus: params.DetailedStatus{
							Status: "active",
							Life:   life.Dying,
							Since:  since,
						},
					},
				},
			},
		},
	}
}

// statusWithSuspendedModel returns a status whose Model.ModelStatus is
// "suspended" with the given Since timestamp.
func statusWithSuspendedModel(modelName string, since *time.Time) *params.FullStatus {
	return &params.FullStatus{
		Model: params.ModelStatusInfo{
			Name: modelName,
			ModelStatus: params.DetailedStatus{
				Status: "suspended",
				Info:   "cloud credential invalid",
				Since:  since,
			},
		},
		Applications: map[string]params.ApplicationStatus{},
	}
}
