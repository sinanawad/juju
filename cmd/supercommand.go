// Copyright 2012 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package cmd

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"golang.org/x/crypto/ssh/terminal"

	jujuversion "github.com/juju/juju/core/version"
	"github.com/juju/juju/internal/cmd"
	internallogger "github.com/juju/juju/internal/logger"
	"github.com/juju/juju/juju/osenv"
)

func init() {
	// If the environment key is empty, ConfigureLoggers returns nil and does
	// nothing.
	err := internallogger.ConfigureLoggers(os.Getenv(osenv.JujuStartupLoggingConfigEnvKey))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR parsing %s: %s\n\n", osenv.JujuStartupLoggingConfigEnvKey, err)
	}
}

var logger = internallogger.GetLogger("juju.cmd")

// DefaultLog is the default command logging implementation.
var DefaultLog = &cmd.Log{
	DefaultConfig: os.Getenv(osenv.JujuLoggingConfigEnvKey),
}

// NewSuperCommand is like cmd.NewSuperCommand but
// it adds juju-specific functionality:
// - The default logging configuration is taken from the environment;
// - The version is configured to the current juju version;
// - The additional version information is sourced from juju/juju/version;
// - The command emits a log message when a command runs.
func NewSuperCommand(p cmd.SuperCommandParams) *cmd.SuperCommand {
	if p.NotifyRun != nil {
		messenger := p.NotifyRun
		p.NotifyRun = func(str string) {
			messenger(str)
			runNotifier(str)
		}
	} else {
		p.NotifyRun = runNotifier
	}
	p.FlagKnownAs = "option"
	return cmd.NewSuperCommand(p)
}

func runNotifier(name string) {
	logger.Infof(context.TODO(), "running %s [%s %s %s %s]", name, jujuversion.Current, jujuversion.GitCommit, runtime.Compiler, runtime.Version())
	logger.Debugf(context.TODO(), "  args: %#v", os.Args)
}

func Info(i *cmd.Info) *cmd.Info {
	info := *i
	info.FlagKnownAs = "option"
	info.ShowSuperFlags = []string{"show-log", "debug", "logging-config", "verbose", "quiet", "h", "help"}
	return &info
}

// IsPiped determines if the command was used in a pipe and,
// hence, it's stdin is not usable for user input.
func IsPiped(ctx *cmd.Context) bool {
	stdIn, ok := ctx.Stdin.(*os.File)
	return ok && !terminal.IsTerminal(int(stdIn.Fd()))
}
