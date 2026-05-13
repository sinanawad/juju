// Copyright 2026 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package citizen

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/juju/errors"
)

// noFindingsLiteral is the exact stdout content (without trailing
// newline) emitted in the hybrid format when no findings remain after
// filtering. See specs/003-juju-citizen-cli/contracts/cli-contract.md.
const noFindingsLiteral = "No citizenship findings."

// formatHybrid renders a list of findings in the byte-locked hybrid
// format defined in contracts/cli-contract.md. The value parameter
// must be []Finding; any other type is a programming error.
func formatHybrid(writer io.Writer, value any) error {
	findings, ok := value.([]Finding)
	if !ok {
		return errors.Errorf("expected []Finding, got %T", value)
	}
	if len(findings) == 0 {
		_, err := fmt.Fprintln(writer, noFindingsLiteral)
		return err
	}
	for i, f := range findings {
		if i > 0 {
			if _, err := fmt.Fprintln(writer); err != nil {
				return err
			}
		}
		if err := writeFinding(writer, f); err != nil {
			return err
		}
	}
	return nil
}

// writeFinding emits one finding in hybrid format.
func writeFinding(writer io.Writer, f Finding) error {
	// Severity padded to 8 chars right-aligned with spaces (CRITICAL
	// is already 8 chars; INFO is 4 -> pad to 8; WARNING is 7 -> pad
	// to 8).
	severityTag := fmt.Sprintf("%-8s", strings.ToUpper(string(f.Severity)))
	if _, err := fmt.Fprintf(writer, "%s %s [%s]\n", severityTag, f.Entity, f.CheckID); err != nil {
		return err
	}
	notes := noteLines(f)
	for _, n := range notes {
		if _, err := fmt.Fprintf(writer, "   -> %s\n", n); err != nil {
			return err
		}
	}
	return nil
}

// noteLines returns the 1-3 note lines for a finding's hybrid render:
// line 1 is always the summary; lines 2-3 are the first two non-empty
// lines of the recommendation.
func noteLines(f Finding) []string {
	out := []string{f.Summary}
	for _, line := range strings.Split(f.Recommendation, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
		if len(out) == 3 {
			break
		}
	}
	return out
}

// -- Table formatter (new default) ------------------------------------
//
// formatTable renders the byte-locked dashboard + table layout defined
// in docs/superpowers/specs/2026-05-13-citizen-table-format-design.md.

// Package-level overrides; tests flip these via export_test.go helpers.
// The defaults match production behaviour.
var (
	// nowFunc returns the time used for the "scanned:" line and for
	// age calculations. Tests pin it for deterministic golden output.
	nowFunc = time.Now

	// colorEnabled toggles ANSI escapes. When false, all colorize()
	// calls emit the underlying text unmodified.
	colorEnabled = true

	// modelNameForTest is the model name rendered in the dashboard
	// panel. The empty string is replaced with "<unknown>" at render
	// time. The CLI wires the real model name through this var.
	modelNameForTest = ""
)

// Dashboard panel geometry. The outer panel is 80 visible columns wide:
// one border char on each side, plus 78 inner columns.
const (
	panelWidth      = 80
	panelInnerWidth = panelWidth - 2 // = 78
	panelMargin     = 2              // leading/trailing spaces inside the borders
)

// ANSI escape sequences. Color is applied as a single unit around the
// "glyph space tag" block in the table and around the same block in
// the dashboard severity-counts line.
const (
	ansiReset        = "\x1b[0m"
	ansiRed          = "\x1b[31m"
	ansiYellow       = "\x1b[33m"
	ansiCyan         = "\x1b[36m"
	ansiBold         = "\x1b[1m"
	ansiBoldOff      = "\x1b[22m"
	ansiDim          = "\x1b[2m"
	ansiDimOff       = "\x1b[22m"
	ansiUnderline    = "\x1b[4m"
	ansiUnderlineOff = "\x1b[24m"
)

// Severity glyphs are stable Unicode codepoints.
const (
	glyphCritical = "●" // ● (U+25CF) BLACK CIRCLE
	glyphWarning  = "▲" // ▲ (U+25B2) BLACK UP-POINTING TRIANGLE
	glyphInfo     = "◆" // ◆ (U+25C6) BLACK DIAMOND
	glyphCheck    = "✓" // ✓ (U+2713) CHECK MARK
	glyphEllipsis = "…" // … (U+2026) HORIZONTAL ELLIPSIS
	glyphDash     = "—" // — (U+2014) EM DASH
	glyphBullet   = "•" // • (U+2022) BULLET
)

// Box-drawing characters.
const (
	boxTopLeft     = "┌" // ┌
	boxTopRight    = "┐" // ┐
	boxBottomLeft  = "└" // └
	boxBottomRight = "┘" // ┘
	boxVertical    = "│" // │
	boxHorizontal  = "─" // ─
)

// Column widths for the table. These are the visible character widths;
// ANSI escape bytes are not counted toward the width.
const (
	colSevWidth    = 8
	colEntityWidth = 24
	colOwnerWidth  = 6 // wide enough for "AUTHOR" (the longest short owner)
	colCheckWidth  = 22
	colAgeWidth    = 7
	colSeparator   = "  " // two spaces between columns
	rowLeftMargin  = "  " // two-space left margin on every row
)

// formatTable renders findings as a dashboard panel optionally followed
// by a six-column table. The value parameter must be []Finding; any
// other type is a programming error. On an empty slice the formatter
// emits only the dashboard's empty-state panel.
func formatTable(writer io.Writer, value any) error {
	findings, ok := value.([]Finding)
	if !ok {
		return errors.Errorf("expected []Finding, got %T", value)
	}
	now := nowFunc()
	if len(findings) == 0 {
		return writeDashboardEmpty(writer, now)
	}
	if err := writeDashboard(writer, findings, now); err != nil {
		return err
	}
	// Blank line between panel and table.
	if _, err := fmt.Fprintln(writer); err != nil {
		return err
	}
	return writeTable(writer, findings, now)
}

// writeDashboard emits the four-line dashboard panel for a non-empty
// finding list.
func writeDashboard(writer io.Writer, findings []Finding, now time.Time) error {
	model := modelNameForTest
	if model == "" {
		model = "<unknown>"
	}
	scanned := now.Format("15:04:05")

	if err := writeTopBorder(writer); err != nil {
		return err
	}
	if err := writePanelLineModel(writer, model, scanned); err != nil {
		return err
	}
	if err := writePanelLineSeverity(writer, findings); err != nil {
		return err
	}
	if err := writePanelLineOwners(writer, findings); err != nil {
		return err
	}
	return writeBottomBorder(writer)
}

// writeDashboardEmpty emits the three-line dashboard panel used when
// no findings remain. The third content line is omitted; only the
// model/scanned line and the "all units are good citizens" line are
// shown.
func writeDashboardEmpty(writer io.Writer, now time.Time) error {
	model := modelNameForTest
	if model == "" {
		model = "<unknown>"
	}
	scanned := now.Format("15:04:05")

	if err := writeTopBorder(writer); err != nil {
		return err
	}
	if err := writePanelLineModel(writer, model, scanned); err != nil {
		return err
	}
	// Empty-state second line replaces the severity counts; the
	// owners line is omitted entirely.
	body := fmt.Sprintf("findings: 0   %s all units are good citizens", glyphCheck)
	if err := writePanelLine(writer, body, 0); err != nil {
		return err
	}
	return writeBottomBorder(writer)
}

// writeTopBorder emits the panel's top border with the inset title.
func writeTopBorder(writer io.Writer) error {
	// Format: ┌─ juju citizen <fill ─> ┐
	// Inner width is 78. The prefix "─ juju citizen " occupies 15
	// inner columns (1 dash + 1 space + 12 letters + 1 space); the
	// remainder is dashes.
	prefix := boxHorizontal + " juju citizen "
	prefixWidth := visibleWidth(prefix)
	fill := strings.Repeat(boxHorizontal, panelInnerWidth-prefixWidth)
	_, err := fmt.Fprintf(writer, "%s%s%s%s\n", boxTopLeft, prefix, fill, boxTopRight)
	return err
}

// writeBottomBorder emits the panel's bottom border (└ + 78 ─ + ┘).
func writeBottomBorder(writer io.Writer) error {
	fill := strings.Repeat(boxHorizontal, panelInnerWidth)
	_, err := fmt.Fprintf(writer, "%s%s%s\n", boxBottomLeft, fill, boxBottomRight)
	return err
}

// writePanelLineModel emits the model+scanned content line. The model
// label is left-aligned at the 2-char inner margin; scanned is right-
// aligned with a matching 2-char trailing margin.
func writePanelLineModel(writer io.Writer, model, scanned string) error {
	// Polish: bold the labels; cyan the scanned time value.
	modelLabel := "model:"
	scannedLabel := "scanned:"
	scannedValue := scanned
	if colorEnabled {
		modelLabel = ansiBold + modelLabel + ansiBoldOff
		scannedLabel = ansiBold + scannedLabel + ansiBoldOff
		scannedValue = ansiCyan + scanned + ansiReset
	}
	left := fmt.Sprintf("%s %s", modelLabel, model)
	right := fmt.Sprintf("%s %s", scannedLabel, scannedValue)
	// Inner usable width after the 2-char left + 2-char right margin.
	usable := panelInnerWidth - 2*panelMargin
	leftW := visibleWidth(left)
	rightW := visibleWidth(right)
	gap := usable - leftW - rightW
	if gap < 1 {
		gap = 1
	}
	body := left + strings.Repeat(" ", gap) + right
	return writePanelLine(writer, body, 0)
}

// writePanelLineSeverity emits the severity-counts content line.
// Zero counts are intentionally shown so the line geometry is stable
// across runs.
func writePanelLineSeverity(writer io.Writer, findings []Finding) error {
	var crit, warn, info int
	for _, f := range findings {
		switch f.Severity {
		case SeverityCritical:
			crit++
		case SeverityWarning:
			warn++
		case SeverityInfo:
			info++
		}
	}
	total := crit + warn + info
	critTag := colorize(fmt.Sprintf("%s %d critical", glyphCritical, crit), ansiRed)
	warnTag := colorize(fmt.Sprintf("%s %d warning", glyphWarning, warn), ansiYellow)
	infoTag := colorize(fmt.Sprintf("%s %d info", glyphInfo, info), ansiCyan)
	findingsLabel := "findings:"
	if colorEnabled {
		findingsLabel = ansiBold + findingsLabel + ansiBoldOff
	}
	body := fmt.Sprintf("%s %d   %s   %s   %s", findingsLabel, total, critTag, warnTag, infoTag)
	return writePanelLine(writer, body, 0)
}

// writePanelLineOwners emits the owners content line. All four owner
// kinds are shown, even when their counts are zero, so operators see
// the full distribution at a glance.
func writePanelLineOwners(writer io.Writer, findings []Finding) error {
	counts := map[Owner]int{
		OwnerCharmAuthor: 0,
		OwnerOperator:    0,
		OwnerMixed:       0,
		OwnerPlatform:    0,
	}
	for _, f := range findings {
		counts[f.Owner]++
	}
	ownersLabel := "owners:"
	if colorEnabled {
		ownersLabel = ansiBold + ownersLabel + ansiBoldOff
	}
	body := fmt.Sprintf(
		"%s       AUTHOR %d  %s  OP %d  %s  MIX %d  %s  PLAT %d",
		ownersLabel,
		counts[OwnerCharmAuthor], glyphBullet,
		counts[OwnerOperator], glyphBullet,
		counts[OwnerMixed], glyphBullet,
		counts[OwnerPlatform],
	)
	return writePanelLine(writer, body, 0)
}

// writePanelLine emits one content line of the panel: a left border,
// 2-char left margin, body, right-pad to the inner width, 2-char right
// margin, right border. hiddenBytes is the number of bytes in `body`
// that are NOT visible (ANSI escapes), used to compute right-pad.
func writePanelLine(writer io.Writer, body string, hiddenBytes int) error {
	// Use visibleWidth so ANSI escapes in `body` are accounted for
	// automatically. The hiddenBytes parameter is retained for caller
	// signature compatibility but no longer needed.
	_ = hiddenBytes
	bodyVisible := visibleWidth(body)
	leftPad := strings.Repeat(" ", panelMargin)
	usable := panelInnerWidth - 2*panelMargin
	rightPad := usable - bodyVisible
	if rightPad < 0 {
		rightPad = 0
	}
	rightPadStr := strings.Repeat(" ", rightPad)
	tailPad := strings.Repeat(" ", panelMargin)
	_, err := fmt.Fprintf(writer, "%s%s%s%s%s%s\n",
		boxVertical, leftPad, body, rightPadStr, tailPad, boxVertical)
	return err
}

// writeTable emits the header row followed by one row per finding.
// Findings are rendered in the order given; sorting is the caller's
// responsibility.
func writeTable(writer io.Writer, findings []Finding, now time.Time) error {
	headerBody := padOrTruncate("SEV", colSevWidth) + colSeparator +
		padOrTruncate("ENTITY", colEntityWidth) + colSeparator +
		padOrTruncate("OWNER", colOwnerWidth) + colSeparator +
		padOrTruncate("CHECK", colCheckWidth) + colSeparator +
		padAgeRight("AGE", colAgeWidth) + colSeparator +
		"SUMMARY"
	// Polish: underline the header line when color is enabled.
	if colorEnabled {
		headerBody = ansiUnderline + headerBody + ansiUnderlineOff
	}
	if _, err := fmt.Fprintf(writer, "%s%s\n", rowLeftMargin, headerBody); err != nil {
		return err
	}
	for _, f := range findings {
		if err := writeTableRow(writer, f, now); err != nil {
			return err
		}
	}
	return nil
}

// writeTableRow emits a single data row. SUMMARY is unbounded and
// has no trailing column separator. Polish (color-enabled only):
// ENTITY is bold, OWNER is dim, AGE is yellow > 30m / red > 2h.
func writeTableRow(writer io.Writer, f Finding, now time.Time) error {
	sevCell := formatSeverityCell(f.Severity)
	entityCell := padOrTruncate(f.Entity, colEntityWidth)
	ownerCell := padOrTruncate(shortOwner(f.Owner), colOwnerWidth)
	checkCell := padOrTruncate(f.CheckID, colCheckWidth)
	ageCell := padAgeRight(formatAge(f.Since, now), colAgeWidth)

	if colorEnabled {
		entityCell = ansiBold + entityCell + ansiBoldOff
		ownerCell = ansiDim + ownerCell + ansiDimOff
		if ansi := ageColor(f.Since, now); ansi != "" {
			ageCell = ansi + ageCell + ansiReset
		}
	}
	_, err := fmt.Fprintf(writer, "%s%s%s%s%s%s%s%s%s%s%s%s\n",
		rowLeftMargin,
		sevCell, colSeparator,
		entityCell, colSeparator,
		ownerCell, colSeparator,
		checkCell, colSeparator,
		ageCell, colSeparator,
		f.Summary,
	)
	return err
}

// formatSeverityCell returns the "glyph space tag" cell, colored if
// color is enabled. The visible width is exactly colSevWidth (8); the
// ANSI escape wraps only the visible glyph+tag block so trailing
// spaces remain uncolored.
func formatSeverityCell(s Severity) string {
	var glyph, tag, ansi string
	switch s {
	case SeverityCritical:
		glyph, tag, ansi = glyphCritical, "crit", ansiRed
	case SeverityWarning:
		glyph, tag, ansi = glyphWarning, "warn", ansiYellow
	case SeverityInfo:
		glyph, tag, ansi = glyphInfo, "info", ansiCyan
	default:
		// Unknown severity: render the raw value with no color so
		// the regression is visible.
		return padOrTruncate(string(s), colSevWidth)
	}
	core := fmt.Sprintf("%s %s", glyph, tag)
	visW := visibleWidth(core)
	return colorize(core, ansi) + strings.Repeat(" ", colSevWidth-visW)
}

// padOrTruncate pads s with trailing spaces to width visible chars,
// or truncates with an ellipsis (…) at visible position width-1.
// Width is measured in runes (visible chars), not bytes.
func padOrTruncate(s string, width int) string {
	runes := []rune(s)
	if len(runes) > width {
		// Reserve 1 rune for the ellipsis.
		return string(runes[:width-1]) + glyphEllipsis
	}
	if len(runes) == width {
		return s
	}
	return s + strings.Repeat(" ", width-len(runes))
}

// padAgeRight right-aligns s in a field of width chars.
func padAgeRight(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(runes)) + s
}

// formatAge renders the time elapsed since `since` relative to `now`
// using the bands defined in the design contract:
//   - nil  → "—" (em dash, U+2014)
//   - <60s → "<1m"
//   - 60s..59m → "5m", "18m"
//   - 60m..23h → "3h"
//   - >=24h → "2d" (integer days)
//
// shortOwner maps an Owner to its short table-column label.
// AUTHOR / OP / MIX / PLAT. Unknown values fall back to the raw
// enum string so the regression is visible in golden diffs.
func shortOwner(o Owner) string {
	switch o {
	case OwnerCharmAuthor:
		return "AUTHOR"
	case OwnerOperator:
		return "OP"
	case OwnerMixed:
		return "MIX"
	case OwnerPlatform:
		return "PLAT"
	}
	return string(o)
}

// ageColor returns an ANSI color code for an age value, or "" for
// uncolored. Bands: >30m yellow, >2h red. Nil since is uncolored.
func ageColor(since *time.Time, now time.Time) string {
	if since == nil {
		return ""
	}
	d := now.Sub(*since)
	if d > 2*time.Hour {
		return ansiRed
	}
	if d > 30*time.Minute {
		return ansiYellow
	}
	return ""
}

func formatAge(since *time.Time, now time.Time) string {
	if since == nil {
		return glyphDash
	}
	d := now.Sub(*since)
	if d < 0 {
		// Future timestamp (clock skew): treat as just-now.
		return "<1m"
	}
	if d < time.Minute {
		return "<1m"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d/time.Minute))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d/time.Hour))
	}
	return fmt.Sprintf("%dd", int(d/(24*time.Hour)))
}

// colorize wraps text in an ANSI escape if color is enabled; otherwise
// returns text unmodified.
func colorize(text, ansi string) string {
	if !colorEnabled {
		return text
	}
	return ansi + text + ansiReset
}

// visibleWidth returns the number of visible chars (runes) in s. ANSI
// escape sequences are stripped before counting. This is the column
// width metric used by all panel-width calculations.
func visibleWidth(s string) int {
	stripped := stripANSI(s)
	return len([]rune(stripped))
}

// stripANSI removes ANSI CSI sequences (the "\x1b[...m" form we use)
// from s. It is intentionally narrow: only CSI sequences ending in
// 'm' are recognised, which covers every escape we emit.
func stripANSI(s string) string {
	if !strings.Contains(s, "\x1b[") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if i+1 < len(s) && s[i] == '\x1b' && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && s[j] != 'm' {
				j++
			}
			// Skip past the closing 'm' (or to end of string).
			if j < len(s) {
				i = j + 1
			} else {
				i = j
			}
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
