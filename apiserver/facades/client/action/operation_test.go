// Copyright 2020 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package action_test

import (
	gc "gopkg.in/check.v1"
)

type operationSuite struct {
	baseSuite
}

var _ = gc.Suite(&operationSuite{})

func (s *operationSuite) TestStub(c *gc.C) {
	c.Skip(`This suite is missing tests for the following scenarios:
- ListOperations querying by status.
- ListOperations querying by action names.
- ListOperations querying by application names.
- ListOperations querying by unit names.
- ListOperations querying by machines.
- ListOperations querying with multiple filters - result is union.
- Operations based on input entity tags.
- EnqueueOperation with some units
- EnqueueOperation but AddAction fails
- EnqueueOperation with a unit specified with a leader receiver
`)
}
