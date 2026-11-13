// Copyright 2019 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package exec

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/juju/errors"
	"k8s.io/client-go/util/exec"
)

// ExitError exposes what we need from k8s exec.ExitError
type ExitError interface {
	error
	String() string
	ExitStatus() int
}

var _ ExitError = exec.CodeExitError{}

// ContainerNotRunningError error is returned when the container is valid
// but not currently running, so the operation is retryable.
type ContainerNotRunningError struct {
	err string
}

var _ error = &ContainerNotRunningError{}

func (e ContainerNotRunningError) Error() string {
	return e.err
}

func containerNotRunningError(container string) error {
	return &ContainerNotRunningError{
		err: fmt.Sprintf("container %q not running", container),
	}
}

// IsContainerNotRunningError returns true when the supplied error is
// caused by a ContainerNotRunningError.
func IsContainerNotRunningError(err error) bool {
	_, ok := errors.Cause(err).(*ContainerNotRunningError)
	return ok
}

var kubeletContainerNotFoundRegexp = regexp.MustCompile(`^.*container not found \("([a-zA-Z0-9\-]+)"\)$`)

// ErrNoShell is returned when a container has neither sh nor bash.
const ErrNoShell errors.ConstError = "Juju couldn't find any shell in the container " +
	"(/bin/bash or /bin/sh). The command can't be executed"

// missingShellRegexp matches exec/stat failures naming a shell, excluding
// missing paths such as the working directory.
var missingShellRegexp = regexp.MustCompile(`(?:exec: "?|stat |exec )(?:[^ :"]*/)?(sh|bash)"?: (?:executable file not found(?: in \$PATH)?|no such file or directory)`)

// isMissingExecutable reports whether the runtime could not find shell.
// Command exit errors and failures to find other resources do not match.
func isMissingExecutable(err error, shell string) bool {
	if err == nil {
		return false
	}
	cause := errors.Cause(err)
	if _, ok := cause.(ExitError); ok {
		return false
	}
	msg := cause.Error()
	if !strings.Contains(msg, "OCI runtime exec failed") {
		return false
	}
	match := missingShellRegexp.FindStringSubmatch(msg)
	return len(match) > 1 && match[1] == shell
}

func handleContainerNotFoundError(err error) error {
	match := kubeletContainerNotFoundRegexp.FindStringSubmatch(err.Error())
	if match == nil {
		return err
	}
	return containerNotRunningError(match[1])
}
