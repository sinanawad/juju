// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package agent

import (
	"context"
	"os"

	"github.com/juju/errors"
	"github.com/juju/utils/v4"
)

var ErrNoStateServingInfo = errors.New("StateServingInfo missing")

func WriteSystemIdentityFile(c Config) error {
	info, ok := c.StateServingInfo()
	if !ok {
		return errors.Trace(ErrNoStateServingInfo)
	}
	// Write non-empty contents to the file, otherwise delete it
	if info.SystemIdentity != "" {
		logger.Infof(context.TODO(), "writing system identity file")
		err := utils.AtomicWriteFile(c.SystemIdentityPath(), []byte(info.SystemIdentity), 0600)
		if err != nil {
			return errors.Annotate(err, "cannot write system identity")
		}
	} else {
		logger.Infof(context.TODO(), "removing system identity file")
		os.Remove(c.SystemIdentityPath())
	}
	return nil
}
