// Copyright 2026 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package citizen_test

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"
	stdtesting "testing"
	"time"

	"github.com/juju/tc"

	"github.com/juju/juju/cmd/juju/citizen"
)

func TestFormatterSuite(t *stdtesting.T) {
	tc.Run(t, &formatterSuite{})
}

type formatterSuite struct{}

// sampleFindings mirrors the canonical example in
// specs/003-juju-citizen-cli/contracts/cli-contract.md.
func sampleFindings() []citizen.Finding {
	return []citizen.Finding{
		{
			CheckID:        "unit-blocked-stale",
			Severity:       citizen.SeverityCritical,
			Entity:         "postgresql/0",
			EntityKind:     citizen.EntityKindUnit,
			Owner:          citizen.OwnerMixed,
			Summary:        "Unit has been blocked for 9 days.",
			Recommendation: "Investigate blocking condition: charm hook message, peer state,\nor operator intervention required.",
			ProtocolRef:    "protocol://citizenship/4c#status/blocked-bounded",
		},
		{
			CheckID:        "charm-revision-aging",
			Severity:       citizen.SeverityWarning,
			Entity:         "nginx-ingress",
			EntityKind:     citizen.EntityKindApplication,
			Owner:          citizen.OwnerOperator,
			Summary:        "Application is behind its tracked channel.",
			Recommendation: "Run 'juju refresh nginx-ingress' to pick up newer revision.",
			ProtocolRef:    "protocol://citizenship/4c#revision/track-channel",
		},
		{
			CheckID:        "active-with-message",
			Severity:       citizen.SeverityInfo,
			Entity:         "postgresql/0",
			EntityKind:     citizen.EntityKindUnit,
			Owner:          citizen.OwnerCharmAuthor,
			Summary:        "Unit reports 'active' with a non-empty status message.",
			Recommendation: "Convention is that 'active' carries no message; the empty string\nis the visual signal of normal operation.",
			ProtocolRef:    "protocol://citizenship/4c#status/active-empty-msg",
		},
	}
}

func (s *formatterSuite) TestHybridEmpty(c *tc.C) {
	var buf bytes.Buffer
	err := citizen.FormatHybrid(&buf, []citizen.Finding{})
	c.Assert(err, tc.ErrorIsNil)
	c.Check(buf.String(), tc.Equals, citizen.NoFindingsLiteral+"\n")
}

func (s *formatterSuite) TestHybridGolden(c *tc.C) {
	// Disable color so the assertion pins the structural format only.
	// The snazzy variant (ANSI bold/dim + colored severity tag +
	// Unicode arrow) is exercised separately by the live demo.
	restore := citizen.SetTableFormatTestOverrides(time.Now, false, "")
	defer restore()

	var buf bytes.Buffer
	err := citizen.FormatHybrid(&buf, sampleFindings())
	c.Assert(err, tc.ErrorIsNil)
	expected := "" +
		"● CRITICAL postgresql/0 [unit-blocked-stale]\n" +
		"   -> Unit has been blocked for 9 days.\n" +
		"   -> Investigate blocking condition: charm hook message, peer state,\n" +
		"   -> or operator intervention required.\n" +
		"\n" +
		"▲ WARNING  nginx-ingress [charm-revision-aging]\n" +
		"   -> Application is behind its tracked channel.\n" +
		"   -> Run 'juju refresh nginx-ingress' to pick up newer revision.\n" +
		"\n" +
		"◆ INFO     postgresql/0 [active-with-message]\n" +
		"   -> Unit reports 'active' with a non-empty status message.\n" +
		"   -> Convention is that 'active' carries no message; the empty string\n" +
		"   -> is the visual signal of normal operation.\n"
	c.Check(buf.String(), tc.Equals, expected)
}

// TestFindingFieldSetExactlyEight enforces SC-003 at the contract
// boundary: a marshalled Finding has exactly the eight named fields.
// If a future change adds, removes, or renames a field, this test
// fails before merge.
func (s *formatterSuite) TestFindingFieldSetExactlyEight(c *tc.C) {
	f := citizen.Finding{
		CheckID:        "x",
		Severity:       citizen.SeverityInfo,
		Entity:         "e",
		EntityKind:     citizen.EntityKindUnit,
		Owner:          citizen.OwnerCharmAuthor,
		Summary:        "s",
		Recommendation: "r",
		ProtocolRef:    "p",
	}
	data, err := json.Marshal(f)
	c.Assert(err, tc.ErrorIsNil)
	var m map[string]any
	c.Assert(json.Unmarshal(data, &m), tc.ErrorIsNil)
	c.Assert(m, tc.HasLen, 8)
	got := make([]string, 0, len(m))
	for k := range m {
		got = append(got, k)
	}
	sort.Strings(got)
	want := []string{
		"check_id",
		"entity",
		"entity_kind",
		"owner",
		"protocol_ref",
		"recommendation",
		"severity",
		"summary",
	}
	c.Check(got, tc.DeepEquals, want)
}

// tableFixedNow is the pinned reference timestamp used by the table
// golden tests. Two findings carry Since offsets relative to this
// instant; the dashboard's "scanned:" line renders at this moment.
var tableFixedNow = time.Date(2026, 5, 13, 13, 54, 21, 0, time.UTC)

// sampleFindingsWithSince returns the four-row fixture used by
// TestFormatTableGolden. Ordering matters: tests pin row order to
// produce ages 5m, 18m, —, — and exercise both severity colors and
// the nil-Since em-dash path.
func sampleFindingsWithSince() []citizen.Finding {
	since5m := tableFixedNow.Add(-5 * time.Minute)
	since18m := tableFixedNow.Add(-18 * time.Minute)
	return []citizen.Finding{
		{
			CheckID:        "hook-error",
			Severity:       citizen.SeverityCritical,
			Entity:         "db/0",
			EntityKind:     citizen.EntityKindUnit,
			Owner:          citizen.OwnerCharmAuthor,
			Summary:        "Recent uncaught hook exception",
			Recommendation: "Investigate hook traceback.",
			ProtocolRef:    "protocol://citizenship/4c#agent/error",
			Since:          &since5m,
		},
		{
			CheckID:        "status-churn",
			Severity:       citizen.SeverityWarning,
			Entity:         "bad-churn/0",
			EntityKind:     citizen.EntityKindUnit,
			Owner:          citizen.OwnerCharmAuthor,
			Summary:        "Workload status churning",
			Recommendation: "Stabilize status reporting.",
			ProtocolRef:    "protocol://citizenship/4c#status/churn",
			Since:          &since18m,
		},
		{
			CheckID:        "blocked-no-message",
			Severity:       citizen.SeverityWarning,
			Entity:         "bad-blocked/0",
			EntityKind:     citizen.EntityKindUnit,
			Owner:          citizen.OwnerCharmAuthor,
			Summary:        "Blocked w/o actionable msg",
			Recommendation: "Set a message describing the block.",
			ProtocolRef:    "protocol://citizenship/4c#status/blocked-msg",
		},
		{
			CheckID:        "active-with-message",
			Severity:       citizen.SeverityInfo,
			Entity:         "bad-active/0",
			EntityKind:     citizen.EntityKindUnit,
			Owner:          citizen.OwnerCharmAuthor,
			Summary:        "Active unit carries non-empty msg",
			Recommendation: "Clear the active-status message.",
			ProtocolRef:    "protocol://citizenship/4c#status/active-empty-msg",
		},
	}
}

// TestFormatTableGolden pins the byte-for-byte output of the new
// default table format against a fixed four-finding fixture with
// color disabled. Any change to dashboard geometry, column widths,
// or row layout is required to update this expected string.
//
// This test is a top-level function (not a suite method) so that
// `go test -run 'FormatTable'` picks it up without a slash prefix.
func TestFormatTableGolden(t *stdtesting.T) {
	tc.Run(t, &tableFormatGoldenSuite{})
}

type tableFormatGoldenSuite struct{}

func (s *tableFormatGoldenSuite) TestGolden(c *tc.C) {
	restore := citizen.SetTableFormatTestOverrides(
		func() time.Time { return tableFixedNow },
		false, // color off
		"",    // empty model -> "<unknown>"
	)
	defer restore()

	var buf bytes.Buffer
	err := citizen.FormatTable(&buf, sampleFindingsWithSince())
	c.Assert(err, tc.ErrorIsNil)

	expected := tableGoldenExpected()
	if buf.String() != expected {
		c.Logf("actual output (quoted):\n%q", buf.String())
		c.Logf("expected output (quoted):\n%q", expected)
	}
	c.Check(buf.String(), tc.Equals, expected)
}

// TestFormatTableEmpty pins the three-line empty-state panel: no
// table area follows. The second content line ends in the checkmark
// "all units are good citizens" affirmation.
func TestFormatTableEmpty(t *stdtesting.T) {
	tc.Run(t, &tableFormatEmptySuite{})
}

type tableFormatEmptySuite struct{}

func (s *tableFormatEmptySuite) TestEmpty(c *tc.C) {
	restore := citizen.SetTableFormatTestOverrides(
		func() time.Time { return tableFixedNow },
		false,
		"",
	)
	defer restore()

	var buf bytes.Buffer
	err := citizen.FormatTable(&buf, []citizen.Finding{})
	c.Assert(err, tc.ErrorIsNil)

	expected := tableEmptyExpected()
	if buf.String() != expected {
		c.Logf("actual output (quoted):\n%q", buf.String())
		c.Logf("expected output (quoted):\n%q", expected)
	}
	c.Check(buf.String(), tc.Equals, expected)
	// Sanity: empty state contains the checkmark and no table header.
	c.Check(strings.Contains(buf.String(), "✓ all units are good citizens"), tc.IsTrue)
	c.Check(strings.Contains(buf.String(), "SEV"), tc.IsFalse)
}

// tableGoldenExpected returns the pinned 4-finding golden output.
// Geometry is derived: the panel is 80 visible columns wide, with a
// 78-column inner area and 2-column margins on each side.
func tableGoldenExpected() string {
	// Panel borders are exactly 78 inner chars between the corners.
	topBorder := "┌─ juju citizenship report " + strings.Repeat("─", 52) + "┐\n"
	bottomBorder := "└" + strings.Repeat("─", 78) + "┘\n"

	// Inner usable width = 78 - 2*2 (margins) = 74.
	innerPad := func(body string) string {
		visible := len([]rune(body))
		pad := 74 - visible
		if pad < 0 {
			pad = 0
		}
		return "│  " + body + strings.Repeat(" ", pad) + "  │\n"
	}

	// Line 1: model="<unknown>" (16) + gap + scanned (17) = 74 inner.
	line1 := innerPad("model: <unknown>" + strings.Repeat(" ", 74-16-17) + "scanned: 13:54:21")
	// Line 2: severity counts (51 visible chars) + 23 trailing spaces.
	line2 := innerPad("findings: 4   ● 1 critical   ▲ 2 warning   ◆ 1 info")
	// Line 3: owners (70 visible chars) + 4 trailing spaces.
	line3 := innerPad("owners:       AUTHOR 4  •  OP 0  •  MIX 0  •  PLAT 0")

	// Header and rows: 2-space left margin, column widths
	// 8/24/13/22/7 with 2-space separator, then SUMMARY.
	pad := func(s string, w int) string {
		r := []rune(s)
		if len(r) > w {
			return string(r[:w-1]) + "…"
		}
		return s + strings.Repeat(" ", w-len(r))
	}
	rpad := func(s string, w int) string {
		r := []rune(s)
		if len(r) >= w {
			return s
		}
		return strings.Repeat(" ", w-len(r)) + s
	}
	header := "  " +
		pad("SEV", 8) + "  " +
		pad("ENTITY", 24) + "  " +
		pad("OWNER", 6) + "  " +
		pad("CHECK", 22) + "  " +
		rpad("AGE", 7) + "  " +
		"SUMMARY\n"

	row := func(sev, entity, owner, check, age, summary string) string {
		return "  " +
			pad(sev, 8) + "  " +
			pad(entity, 24) + "  " +
			pad(owner, 6) + "  " +
			pad(check, 22) + "  " +
			rpad(age, 7) + "  " +
			summary + "\n"
	}

	rows := "" +
		row("● crit", "db/0", "AUTHOR", "hook-error", "5m",
			"Recent uncaught hook exception") +
		row("▲ warn", "bad-churn/0", "AUTHOR", "status-churn", "18m",
			"Workload status churning") +
		row("▲ warn", "bad-blocked/0", "AUTHOR", "blocked-no-message", "—",
			"Blocked w/o actionable msg") +
		row("◆ info", "bad-active/0", "AUTHOR", "active-with-message", "—",
			"Active unit carries non-empty msg")

	return topBorder + line1 + line2 + line3 + bottomBorder + "\n" + header + rows
}

// tableEmptyExpected returns the pinned empty-state golden output:
// three content lines, no table area, no trailing newline beyond the
// bottom border.
func tableEmptyExpected() string {
	topBorder := "┌─ juju citizenship report " + strings.Repeat("─", 52) + "┐\n"
	bottomBorder := "└" + strings.Repeat("─", 78) + "┘\n"
	innerPad := func(body string) string {
		visible := len([]rune(body))
		pad := 74 - visible
		if pad < 0 {
			pad = 0
		}
		return "│  " + body + strings.Repeat(" ", pad) + "  │\n"
	}
	line1 := innerPad("model: <unknown>" + strings.Repeat(" ", 74-16-17) + "scanned: 13:54:21")
	line2 := innerPad("findings: 0   ✓ all units are good citizens")
	return topBorder + line1 + line2 + bottomBorder
}

// TestHybridSeverityPadding deleted: the byte-position alignment
// check no longer holds now that the verbose format leads with a
// multi-byte Unicode glyph. The byte-locked TestHybridGolden above
// supersedes it.
