// Copyright 2025 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package storage

import (
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	schematesting "github.com/juju/juju/domain/schema/testing"
)

type provisioningStatusSuite struct {
	schematesting.ModelSuite
}

var _ = gc.Suite(&provisioningStatusSuite{})

// TestProvisioningStatusDBValues ensures there's no skew between what's in the
// database table for provisioning_status and the typed consts used in the state packages.
func (s *provisioningStatusSuite) TestProvisioningStatusDBValues(c *gc.C) {
	db := s.DB()
	rows, err := db.Query("SELECT id, name FROM storage_provisioning_status")
	c.Assert(err, jc.ErrorIsNil)
	defer func() { _ = rows.Close() }()

	dbValues := make(map[ProvisioningStatus]string)
	for rows.Next() {
		var (
			id    int
			value string
		)
		c.Assert(rows.Scan(&id, &value), jc.ErrorIsNil)
		dbValues[ProvisioningStatus(id)] = value
	}
	c.Assert(dbValues, jc.DeepEquals, map[ProvisioningStatus]string{
		ProvisioningStatusPending:     "pending",
		ProvisioningStatusProvisioned: "provisioned",
		ProvisioningStatusError:       "error",
	})
}
