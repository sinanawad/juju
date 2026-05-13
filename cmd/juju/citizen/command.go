// Copyright 2026 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package citizen

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/juju/clock"
	"github.com/juju/errors"
	"github.com/juju/gnuflag"
	"github.com/juju/names/v6"

	"github.com/juju/juju/api/client/client"
	jujucmd "github.com/juju/juju/cmd"
	"github.com/juju/juju/cmd/cmd"
	"github.com/juju/juju/cmd/modelcmd"
	corestatus "github.com/juju/juju/core/status"
	"github.com/juju/juju/rpc/params"
)

// statusAPI is the slice of the model API the citizen command uses.
// Defining it locally lets tests substitute a fake without touching
// the real client. Includes the StatusHistory call used by stateful
// detectors (e.g., status-churn).
type statusAPI interface {
	Status(ctx context.Context, args *client.StatusArgs) (*params.FullStatus, error)
	StatusHistory(
		ctx context.Context,
		kind corestatus.HistoryKind,
		tag names.Tag,
		filter corestatus.StatusHistoryFilter,
	) (corestatus.History, error)
	Close() error
}

// citizenCommand surfaces deployment-level citizenship findings. The
// shape mirrors cmd/juju/block/list.go per constitution Principle VIII.
type citizenCommand struct {
	modelcmd.ModelCommandBase
	out cmd.Output

	// clock is injected for the time-based detector (Signal 3).
	// Defaults to clock.WallClock; tests substitute via export_test.go.
	clock clock.Clock

	// apiFunc opens the status API. Tests override via export_test.go.
	apiFunc func(ctx context.Context) (statusAPI, error)

	// noAI disables fixture-based recommendation enrichment.
	noAI bool

	// noColor disables ANSI escapes in the table formatter.
	noColor bool

	// severityFilter is empty (== all severities pass) by default.
	severityFilter severitySet
}

// NewCitizenCommand returns a wrapped instance of the citizen command.
func NewCitizenCommand() cmd.Command {
	c := &citizenCommand{
		clock: clock.WallClock,
	}
	c.apiFunc = c.defaultAPIFunc
	return modelcmd.Wrap(c)
}

const citizenCommandDoc = `
Surface deployment-level citizenship findings for the current model.

Findings are degradations caused by external factors (charms,
infrastructure) that are invisible to 'juju status'. Each finding
carries a severity (info/warning/critical), an owner (charm-author,
operator, mixed, or platform), the affected entity, a one-line
summary, a recommended action, and a citation to the citizenship
contract clause that was violated.

Examples:

    juju citizen
    juju citizen -o json
    juju citizen --severity=warning,critical
    juju citizen --no-ai
    juju citizen -m other-model
`

// Info implements cmd.Command.Info.
func (c *citizenCommand) Info() *cmd.Info {
	return jujucmd.Info(&cmd.Info{
		Name:    "citizen",
		Purpose: "Surface deployment-level citizenship findings.",
		Doc:     citizenCommandDoc,
		SeeAlso: []string{"status"},
	})
}

// Init implements cmd.Command.Init. No positional arguments.
func (c *citizenCommand) Init(args []string) error {
	return cmd.CheckEmpty(args)
}

// SetFlags implements cmd.Command.SetFlags.
func (c *citizenCommand) SetFlags(f *gnuflag.FlagSet) {
	c.ModelCommandBase.SetFlags(f)
	c.out.AddFlags(f, "table", map[string]cmd.Formatter{
		"table":   formatTable,
		"verbose": formatHybrid,
		"yaml":    cmd.FormatYaml,
		"json":    cmd.FormatJson,
	})
	f.BoolVar(&c.noAI, "no-ai", false, "Disable AI-enriched recommendations")
	f.BoolVar(&c.noColor, "no-color", false, "Disable ANSI color in the default table format")
	f.Var(&c.severityFilter, "severity",
		"Filter findings by severity (comma-separated: info,warning,critical)")
}

// Run implements cmd.Command.Run.
func (c *citizenCommand) Run(ctx *cmd.Context) error {
	api, err := c.apiFunc(ctx)
	if err != nil {
		return errors.Trace(err)
	}
	defer api.Close()

	status, err := api.Status(ctx, nil)
	if err != nil {
		return errors.Annotate(err, "status fetch failed")
	}

	now := c.clock.Now()
	findings := runDetectors(status, now)

	// Stateful detectors look at per-unit status history. Errors here
	// are advisory -- log to stderr and proceed with the pure findings
	// (constitution Principle VI: graceful degradation).
	stateful, sErr := runStatefulDetectors(ctx, api, status, now)
	if sErr != nil && ctx.Stderr != nil {
		fmt.Fprintf(ctx.Stderr,
			"WARNING citizenship: stateful detectors degraded: %s\n", sErr)
	}
	findings = append(findings, stateful...)

	if !c.noAI {
		findings = enrich(ctx, findings)
	}

	if !c.severityFilter.empty() {
		findings = c.severityFilter.filter(findings)
	}

	sort.SliceStable(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.Severity.rank() != b.Severity.rank() {
			return a.Severity.rank() < b.Severity.rank()
		}
		// Within same severity, oldest violation first (Since-desc as age).
		// Findings without Since (pure detectors) sort after those with one.
		if a.Since != nil && b.Since != nil && !a.Since.Equal(*b.Since) {
			return a.Since.Before(*b.Since)
		}
		if a.Since != nil && b.Since == nil {
			return true
		}
		if a.Since == nil && b.Since != nil {
			return false
		}
		if a.Entity != b.Entity {
			return a.Entity < b.Entity
		}
		return a.CheckID < b.CheckID
	})

	// Plumb the model name + color choice into the formatter package
	// before delegating to c.out.Write. Both are package-level vars in
	// formatter.go consulted at render time.
	modelName, _ := c.ModelIdentifier()
	modelNameForTest = modelName
	colorEnabled = !c.noColor

	// Ensure non-nil slice so YAML/JSON emit "[]" rather than "null"
	// when there are zero findings.
	if findings == nil {
		findings = []Finding{}
	}
	return c.out.Write(ctx, findings)
}

// defaultAPIFunc opens the real Juju status API.
func (c *citizenCommand) defaultAPIFunc(ctx context.Context) (statusAPI, error) {
	api, err := c.NewAPIClient(ctx)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return api, nil
}

// ----------------------------------------------------------------------
// severitySet -- the --severity flag value
// ----------------------------------------------------------------------

// severitySet implements gnuflag.Value as a comma-separated set of
// Severity values. Empty (flag absent) means "all severities pass".
type severitySet map[Severity]bool

// String renders the set for gnuflag's help output.
func (s severitySet) String() string {
	if len(s) == 0 {
		return ""
	}
	parts := make([]string, 0, len(s))
	for k := range s {
		parts = append(parts, string(k))
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

// Set parses the flag value. Whitespace around CSV entries is
// trimmed; unknown values produce an error naming the bad input.
func (s *severitySet) Set(value string) error {
	if *s == nil {
		*s = severitySet{}
	}
	for _, raw := range strings.Split(value, ",") {
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		switch Severity(v) {
		case SeverityInfo, SeverityWarning, SeverityCritical:
			(*s)[Severity(v)] = true
		default:
			return fmt.Errorf(
				"invalid --severity value %q: must be one of info, warning, critical",
				v,
			)
		}
	}
	return nil
}

func (s severitySet) empty() bool {
	return len(s) == 0
}

func (s severitySet) filter(in []Finding) []Finding {
	out := make([]Finding, 0, len(in))
	for _, f := range in {
		if s[f.Severity] {
			out = append(out, f)
		}
	}
	return out
}
