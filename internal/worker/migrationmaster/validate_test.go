// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package migrationmaster_test

import (
	"context"

	"github.com/juju/clock"
	"github.com/juju/errors"
	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/api"
	"github.com/juju/juju/core/logger"
	"github.com/juju/juju/internal/migration"
	coretesting "github.com/juju/juju/internal/testing"
	"github.com/juju/juju/internal/worker/fortress"
	"github.com/juju/juju/internal/worker/migrationmaster"
)

type ValidateSuite struct {
	testing.IsolationSuite
}

var _ = gc.Suite(&ValidateSuite{})

func (*ValidateSuite) TestValid(c *gc.C) {
	err := validConfig().Validate()
	c.Check(err, jc.ErrorIsNil)
}

func (*ValidateSuite) TestMissingModelUUID(c *gc.C) {
	config := validConfig()
	config.ModelUUID = ""
	checkNotValid(c, config, `model UUID "" not valid`)
}

func (*ValidateSuite) TestMissingGuard(c *gc.C) {
	config := validConfig()
	config.Guard = nil
	checkNotValid(c, config, "nil Guard not valid")
}

func (*ValidateSuite) TestMissingFacade(c *gc.C) {
	config := validConfig()
	config.Facade = nil
	checkNotValid(c, config, "nil Facade not valid")
}

func (*ValidateSuite) TestMissingAPIOpen(c *gc.C) {
	config := validConfig()
	config.APIOpen = nil
	checkNotValid(c, config, "nil APIOpen not valid")
}

func (*ValidateSuite) TestMissingUploadBinaries(c *gc.C) {
	config := validConfig()
	config.UploadBinaries = nil
	checkNotValid(c, config, "nil UploadBinaries not valid")
}

func (*ValidateSuite) TestMissingCharmService(c *gc.C) {
	config := validConfig()
	config.CharmService = nil
	checkNotValid(c, config, "nil CharmService not valid")
}

func (*ValidateSuite) TestMissingAgentBinaryStore(c *gc.C) {
	config := validConfig()
	config.AgentBinaryStore = nil
	checkNotValid(c, config, "nil AgentBinaryStore not valid")
}

func (*ValidateSuite) TestMissingClock(c *gc.C) {
	config := validConfig()
	config.Clock = nil
	checkNotValid(c, config, "nil Clock not valid")
}

func validConfig() migrationmaster.Config {
	return migrationmaster.Config{
		ModelUUID:        coretesting.ModelTag.Id(),
		Guard:            struct{ fortress.Guard }{},
		Facade:           struct{ migrationmaster.Facade }{},
		APIOpen:          func(context.Context, *api.Info, api.DialOpts) (api.Connection, error) { return nil, nil },
		UploadBinaries:   func(context.Context, migration.UploadBinariesConfig, logger.Logger) error { return nil },
		CharmService:     struct{ migrationmaster.CharmService }{},
		AgentBinaryStore: struct{ migration.AgentBinaryStore }{},
		Clock:            struct{ clock.Clock }{},
	}
}

func checkNotValid(c *gc.C, config migrationmaster.Config, expect string) {
	check := func(err error) {
		c.Check(err, gc.ErrorMatches, expect)
		c.Check(err, jc.ErrorIs, errors.NotValid)
	}

	err := config.Validate()
	check(err)

	worker, err := migrationmaster.New(config)
	c.Check(worker, gc.IsNil)
	check(err)
}
