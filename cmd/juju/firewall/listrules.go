// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package firewall

import (
	"context"
	"fmt"

	"github.com/juju/errors"
	"github.com/juju/gnuflag"

	"github.com/juju/juju/api/client/modelconfig"
	jujucmd "github.com/juju/juju/cmd"
	"github.com/juju/juju/cmd/modelcmd"
	"github.com/juju/juju/core/network/firewall"
	"github.com/juju/juju/environs/config"
	"github.com/juju/juju/internal/cmd"
)

var listRulesHelpSummary = `
Prints the firewall rules.`[1:]

var listRulesHelpDetails = `
Lists the firewall rules which control ingress to well known services
within a Juju model.

DEPRECATION WARNING: %v

`

const listRulesHelpExamples = `
    juju firewall-rules

`

// NewListFirewallRulesCommand returns a command to list firewall rules.
func NewListFirewallRulesCommand() cmd.Command {
	cmd := &listFirewallRulesCommand{}
	cmd.newAPIFunc = func(ctx context.Context) (ListFirewallRulesAPI, error) {
		root, err := cmd.NewAPIRoot(ctx)
		if err != nil {
			return nil, errors.Trace(err)
		}
		return modelconfig.NewClient(root), nil

	}
	return modelcmd.Wrap(cmd)
}

type listFirewallRulesCommand struct {
	modelcmd.ModelCommandBase
	modelcmd.IAASOnlyCommand
	out cmd.Output

	newAPIFunc func(ctx context.Context) (ListFirewallRulesAPI, error)
}

// Info implements cmd.Command.
func (c *listFirewallRulesCommand) Info() *cmd.Info {
	return jujucmd.Info(&cmd.Info{
		Name:     "firewall-rules",
		Purpose:  listRulesHelpSummary,
		Doc:      fmt.Sprintf(listRulesHelpDetails, deprecationWarning),
		Aliases:  []string{"list-firewall-rules"},
		Examples: listRulesHelpExamples,
		SeeAlso: []string{
			"set-firewall-rule",
		},
	})
}

// SetFlags implements cmd.Command.
func (c *listFirewallRulesCommand) SetFlags(f *gnuflag.FlagSet) {
	c.out.AddFlags(f, "tabular", map[string]cmd.Formatter{
		"yaml":    cmd.FormatYaml,
		"json":    cmd.FormatJson,
		"tabular": formatListTabular,
	})
}

// Init implements cmd.Command.
func (c *listFirewallRulesCommand) Init(args []string) (err error) {
	return cmd.CheckEmpty(args)
}

// ListFirewallRulesAPI defines the API methods that the list firewall rules command uses.
type ListFirewallRulesAPI interface {
	Close() error
	ModelGet(ctx context.Context) (map[string]interface{}, error)
}

// Run implements cmd.Command.
func (c *listFirewallRulesCommand) Run(ctx *cmd.Context) error {
	ctx.Warningf(deprecationWarning + "\n")

	client, err := c.newAPIFunc(ctx)
	if err != nil {
		return err
	}
	defer client.Close()
	attrs, err := client.ModelGet(ctx)
	if err != nil {
		return err
	}
	cfg, err := config.New(config.NoDefaults, attrs)
	if err != nil {
		return err
	}

	rules := []firewallRule{{
		KnownService:   firewall.SSHRule,
		WhitelistCIDRS: cfg.SSHAllow(),
	}, {
		KnownService:   firewall.JujuApplicationOfferRule,
		WhitelistCIDRS: cfg.SAASIngressAllow(),
	}}
	return c.out.Write(ctx, rules)
}
