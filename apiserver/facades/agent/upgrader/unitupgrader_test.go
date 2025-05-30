// Copyright 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package upgrader_test

import (
	"github.com/juju/testing"
	gc "gopkg.in/check.v1"
)

type unitUpgraderSuite struct {
	testing.IsolationSuite
}

var _ = gc.Suite(&unitUpgraderSuite{})

func (s *unitUpgraderSuite) TestStub(c *gc.C) {
	c.Skip(`This suite is missing tests for the following scenarios:
 - TestWatchAPIVersionNothing: Not an error to watch nothing
 - TestWatchAPIVersion: watch the api version
 - TestUpgraderAPIRefusesNonUnitAgent: UpgraderAPI refuses non-unit agent
 - TestWatchAPIVersionRefusesWrongAgent: WatchAPIVersion refuses wrong agent
 - TestToolsNothing: Not an error to watch nothing
 - TestToolsRefusesWrongAgent: Tools refuses wrong agent
 - TestToolsForAgent: Tools for agent
 - TestSetToolsNothing: Not an error to watch nothing	
 - TestSetToolsRefusesWrongAgent: SetTools refuses wrong agent
 - TestSetTools: SetTools
 - TestDesiredVersionNothing: Not an error to watch nothing
 - TestDesiredVersionRefusesWrongAgent: DesiredVersion refuses wrong agent
 - TestDesiredVersionNoticesMixedAgents: DesiredVersion notices mixed agents
 - TestDesiredVersionForAgent: DesiredVersion for agent
	`)
}
