// Copyright 2026 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package citizen_test

import (
	"context"
	"errors"
	stdtesting "testing"
	"time"

	"github.com/juju/clock/testclock"
	"github.com/juju/names/v6"
	"github.com/juju/tc"

	"github.com/juju/juju/api/client/client"
	"github.com/juju/juju/api/jujuclient/jujuclienttesting"
	"github.com/juju/juju/cmd/cmd"
	"github.com/juju/juju/cmd/cmd/cmdtesting"
	"github.com/juju/juju/cmd/juju/citizen"
	corestatus "github.com/juju/juju/core/status"
	"github.com/juju/juju/internal/testing"
	"github.com/juju/juju/rpc/params"
)

func TestCommandSuite(t *stdtesting.T) {
	tc.Run(t, &commandSuite{})
}

type commandSuite struct {
	testing.FakeJujuXDGDataHomeSuite
}

// fakeStatusAPI is the test-only statusAPI implementation. It returns
// the canned status and error; Close is a no-op.
type fakeStatusAPI struct {
	status *params.FullStatus
	err    error
	closed bool
}

func (f *fakeStatusAPI) Status(ctx context.Context, args *client.StatusArgs) (*params.FullStatus, error) {
	return f.status, f.err
}

// StatusHistory is a no-op stub for command-suite tests that don't
// exercise the stateful detector path. Returns empty history so the
// status-churn detector emits nothing.
func (f *fakeStatusAPI) StatusHistory(
	ctx context.Context,
	kind corestatus.HistoryKind,
	tag names.Tag,
	filter corestatus.StatusHistoryFilter,
) (corestatus.History, error) {
	return corestatus.History{}, nil
}

func (f *fakeStatusAPI) Close() error {
	f.closed = true
	return nil
}

// commandRefTime pins the time injected into the command's clock.
var commandRefTime = time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC)

func (s *commandSuite) newCommand(api *fakeStatusAPI) cmd.Command {
	store := jujuclienttesting.MinimalStore()
	ck := testclock.NewClock(commandRefTime)
	return citizen.NewCitizenCommandForTest(store, api, nil, ck)
}

func (s *commandSuite) newCommandWithErr(apiErr error) cmd.Command {
	store := jujuclienttesting.MinimalStore()
	ck := testclock.NewClock(commandRefTime)
	return citizen.NewCitizenCommandForTest(store, &fakeStatusAPI{err: apiErr}, nil, ck)
}

// ------------------------------------------------------------------
// Info / Init
// ------------------------------------------------------------------

func (s *commandSuite) TestInfo(c *tc.C) {
	cmd := s.newCommand(&fakeStatusAPI{status: &params.FullStatus{}})
	info := cmd.Info()
	c.Check(info.Name, tc.Equals, "citizen")
	c.Check(info.Purpose, tc.Not(tc.Equals), "")
}

func (s *commandSuite) TestInitRejectsPositionalArgs(c *tc.C) {
	cmd := s.newCommand(&fakeStatusAPI{status: &params.FullStatus{}})
	err := cmdtesting.InitCommand(cmd, []string{"extra-arg"})
	c.Check(err, tc.ErrorMatches, `unrecognized args: \["extra-arg"\]`)
}

// ------------------------------------------------------------------
// Empty model
// ------------------------------------------------------------------

func (s *commandSuite) TestEmptyModelVerbosePrintsLiteralAndEmptyStderr(c *tc.C) {
	// FR-011's "No citizenship findings." literal is preserved in the
	// verbose (formerly hybrid) format. The new default table format
	// emits a dashboard panel instead for empty state.
	cmd := s.newCommand(&fakeStatusAPI{status: &params.FullStatus{}})
	ctx, err := cmdtesting.RunCommand(c, cmd, "--format", "verbose")
	c.Assert(err, tc.ErrorIsNil)
	c.Check(cmdtesting.Stdout(ctx), tc.Equals, "No citizenship findings.\n")
	c.Check(cmdtesting.Stderr(ctx), tc.Equals, "")
}

// ------------------------------------------------------------------
// Three signals, hybrid
// ------------------------------------------------------------------

func threeSignalStatus() *params.FullStatus {
	// Signal 1: active-with-message on grafana/0
	// Signal 2: CanUpgradeTo non-empty on nginx
	// Signal 3: blocked for 9 days on postgresql/0 (critical)
	blockedSince := commandRefTime.Add(-9 * 24 * time.Hour)
	return &params.FullStatus{
		Applications: map[string]params.ApplicationStatus{
			"grafana": {
				Units: map[string]params.UnitStatus{
					"grafana/0": {
						WorkloadStatus: params.DetailedStatus{
							Status: "active",
							Info:   "ready",
						},
					},
				},
			},
			"nginx": {
				CanUpgradeTo: "ch:nginx-99",
			},
			"postgresql": {
				Units: map[string]params.UnitStatus{
					"postgresql/0": {
						WorkloadStatus: params.DetailedStatus{
							Status: "blocked",
							Info:   "missing required config",
							Since:  &blockedSince,
						},
					},
				},
			},
		},
	}
}

func (s *commandSuite) TestDispatchesAllThreeDetectors(c *tc.C) {
	cmd := s.newCommand(&fakeStatusAPI{status: threeSignalStatus()})
	ctx, err := cmdtesting.RunCommand(c, cmd, "--format", "json")
	c.Assert(err, tc.ErrorIsNil)
	out := cmdtesting.Stdout(ctx)
	// Quick sanity: critical comes before warning before info in the
	// sorted output.
	criticalIdx := indexOf(out, `"severity":"critical"`)
	warningIdx := indexOf(out, `"severity":"warning"`)
	infoIdx := indexOf(out, `"severity":"info"`)
	c.Check(criticalIdx, tc.Not(tc.Equals), -1)
	c.Check(warningIdx, tc.Not(tc.Equals), -1)
	c.Check(infoIdx, tc.Not(tc.Equals), -1)
	c.Check(criticalIdx < warningIdx, tc.IsTrue,
		tc.Commentf("expected critical before warning in output:\n%s", out))
	c.Check(warningIdx < infoIdx, tc.IsTrue,
		tc.Commentf("expected warning before info in output:\n%s", out))
}

// ------------------------------------------------------------------
// Status() failure -> hard error (FR-018)
// ------------------------------------------------------------------

func (s *commandSuite) TestStatusFailureHardError(c *tc.C) {
	cmd := s.newCommandWithErr(errors.New("controller unreachable"))
	ctx, err := cmdtesting.RunCommand(c, cmd)
	c.Assert(err, tc.NotNil)
	c.Check(err.Error(), tc.Matches, ".*status fetch failed.*controller unreachable.*")
	c.Check(cmdtesting.Stdout(ctx), tc.Equals, "")
}

// ------------------------------------------------------------------
// --severity flag
// ------------------------------------------------------------------

func (s *commandSuite) TestSeverityFilterCritical(c *tc.C) {
	cmd := s.newCommand(&fakeStatusAPI{status: threeSignalStatus()})
	ctx, err := cmdtesting.RunCommand(c, cmd, "--severity", "critical", "--format", "json")
	c.Assert(err, tc.ErrorIsNil)
	out := cmdtesting.Stdout(ctx)
	c.Check(indexOf(out, `"severity":"critical"`), tc.Not(tc.Equals), -1)
	c.Check(indexOf(out, `"severity":"warning"`), tc.Equals, -1)
	c.Check(indexOf(out, `"severity":"info"`), tc.Equals, -1)
}

func (s *commandSuite) TestSeverityFilterMulti(c *tc.C) {
	cmd := s.newCommand(&fakeStatusAPI{status: threeSignalStatus()})
	ctx, err := cmdtesting.RunCommand(c, cmd,
		"--severity", "warning,critical", "--format", "json")
	c.Assert(err, tc.ErrorIsNil)
	out := cmdtesting.Stdout(ctx)
	c.Check(indexOf(out, `"severity":"critical"`), tc.Not(tc.Equals), -1)
	c.Check(indexOf(out, `"severity":"warning"`), tc.Not(tc.Equals), -1)
	c.Check(indexOf(out, `"severity":"info"`), tc.Equals, -1)
}

func (s *commandSuite) TestSeverityFilterWhitespace(c *tc.C) {
	cmd := s.newCommand(&fakeStatusAPI{status: threeSignalStatus()})
	ctx, err := cmdtesting.RunCommand(c, cmd,
		"--severity", " critical , warning ", "--format", "json")
	c.Assert(err, tc.ErrorIsNil)
	out := cmdtesting.Stdout(ctx)
	c.Check(indexOf(out, `"severity":"critical"`), tc.Not(tc.Equals), -1)
	c.Check(indexOf(out, `"severity":"warning"`), tc.Not(tc.Equals), -1)
}

func (s *commandSuite) TestSeverityFilterRejectsInvalid(c *tc.C) {
	cmd := s.newCommand(&fakeStatusAPI{status: &params.FullStatus{}})
	_, err := cmdtesting.RunCommand(c, cmd, "--severity", "bogus")
	c.Assert(err, tc.NotNil)
	c.Check(err.Error(), tc.Matches, `.*invalid --severity value "bogus".*`)
}

// ------------------------------------------------------------------
// --no-ai flag
// ------------------------------------------------------------------

func (s *commandSuite) TestNoAIPreservesHandwrittenRecommendation(c *tc.C) {
	// Run twice -- once with AI on (default), once with --no-ai.
	// Only the recommendation field should differ; every other field
	// must be byte-identical (SC-005).
	st := threeSignalStatus()

	noAICmd := s.newCommand(&fakeStatusAPI{status: st})
	ctx1, err := cmdtesting.RunCommand(c, noAICmd, "--no-ai", "--format", "yaml")
	c.Assert(err, tc.ErrorIsNil)
	noAIOut := cmdtesting.Stdout(ctx1)

	aiCmd := s.newCommand(&fakeStatusAPI{status: st})
	ctx2, err := cmdtesting.RunCommand(c, aiCmd, "--format", "yaml")
	c.Assert(err, tc.ErrorIsNil)
	aiOut := cmdtesting.Stdout(ctx2)

	// They differ overall:
	c.Check(aiOut, tc.Not(tc.Equals), noAIOut)

	// But the check_id values are present in both:
	for _, id := range []string{
		"active-with-message", "charm-revision-aging", "unit-blocked-stale",
	} {
		c.Check(indexOf(noAIOut, id), tc.Not(tc.Equals), -1,
			tc.Commentf("--no-ai output missing %q", id))
		c.Check(indexOf(aiOut, id), tc.Not(tc.Equals), -1,
			tc.Commentf("AI output missing %q", id))
	}
}

// ------------------------------------------------------------------
// helpers
// ------------------------------------------------------------------

// indexOf is a tiny strings.Index wrapper for inline assertions.
func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

