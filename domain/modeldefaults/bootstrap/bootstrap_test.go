// Copyright 2023 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package bootstrap

import (
	"context"
	"database/sql"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/cloud"
	corecloud "github.com/juju/juju/core/cloud"
	coreuser "github.com/juju/juju/core/user"
	cloudbootstrap "github.com/juju/juju/domain/cloud/bootstrap"
	clouderrors "github.com/juju/juju/domain/cloud/errors"
	"github.com/juju/juju/domain/modeldefaults/state"
	schematesting "github.com/juju/juju/domain/schema/testing"
	_ "github.com/juju/juju/internal/provider/dummy"
)

type bootstrapSuite struct {
	schematesting.ControllerSuite
}

var _ = gc.Suite(&bootstrapSuite{})

func (*bootstrapSuite) TestBootstrapModelDefaults(c *gc.C) {
	provider := ModelDefaultsProvider(
		map[string]any{
			"foo":        "controller",
			"controller": "some value",
		},
		map[string]any{
			"foo":    "region",
			"region": "some value",
		},
		"dummy",
	)

	defaults, err := provider.ModelDefaults(context.Background())
	c.Check(err, jc.ErrorIsNil)
	c.Check(defaults["foo"].Region, gc.Equals, "region")
	c.Check(defaults["controller"].Controller, gc.Equals, "some value")
	c.Check(defaults["region"].Region, gc.Equals, "some value")

	configDefaults := state.ConfigDefaults(context.Background())
	for k, v := range configDefaults {
		c.Check(defaults[k].Default, gc.Equals, v)
	}
}

// TestSetCloudDefaultsNoExist asserts that if we try and set cloud defaults
// for a cloud that doesn't exist we get a [clouderrors.NotFound] error back.
func (s *bootstrapSuite) TestSetCloudDefaultsNoExist(c *gc.C) {
	set := SetCloudDefaults("noexist", map[string]any{
		"HTTP_PROXY": "[2001:0DB8::1]:80",
	})

	err := set(context.Background(), s.TxnRunner(), s.NoopTxnRunner())
	c.Check(err, jc.ErrorIs, clouderrors.NotFound)

	var count int
	row := s.DB().QueryRow("SELECT count(*) FROM cloud_defaults")
	err = row.Scan(&count)
	c.Check(err, jc.ErrorIsNil)
	c.Check(count, gc.Equals, 0)
}

// TestSetCloudDefaults is testing the happy path for setting cloud defaults. We
// expect no errors to be returned in this test and at the end of setting the
// clouds defaults for the same values to be reported back.
func (s *bootstrapSuite) TestSetCloudDefaults(c *gc.C) {
	cld := cloud.Cloud{
		Name:      "cirrus",
		Type:      "ec2",
		AuthTypes: cloud.AuthTypes{cloud.UserPassAuthType},
	}

	err := cloudbootstrap.InsertCloud(
		coreuser.AdminUserName, cld)(context.Background(), s.TxnRunner(), s.NoopTxnRunner())
	c.Check(err, jc.ErrorIsNil)

	set := SetCloudDefaults("cirrus", map[string]any{
		"HTTP_PROXY": "[2001:0DB8::1]:80",
	})

	err = set(context.Background(), s.TxnRunner(), s.NoopTxnRunner())
	c.Check(err, jc.ErrorIsNil)

	var cloudUUID string
	err = s.TxnRunner().StdTxn(context.Background(), func(ctx context.Context, tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, "SELECT uuid FROM cloud WHERE name = ?", "cirrus").Scan(&cloudUUID)
	})
	c.Check(err, jc.ErrorIsNil)

	st := state.NewState(s.TxnRunnerFactory())
	defaults, err := st.CloudDefaults(context.Background(), corecloud.UUID(cloudUUID))
	c.Check(err, jc.ErrorIsNil)
	c.Check(defaults, jc.DeepEquals, map[string]string{
		"HTTP_PROXY": "[2001:0DB8::1]:80",
	})
}

// TestSetCloudDefaultsOverrides is testing that repeated calls to
// [SetCloudDefaults] overrides existing cloud defaults that have been set.
func (s *bootstrapSuite) TestSetCloudDefaultsOverides(c *gc.C) {
	cld := cloud.Cloud{
		Name:      "cirrus",
		Type:      "ec2",
		AuthTypes: cloud.AuthTypes{cloud.UserPassAuthType},
	}
	err := cloudbootstrap.InsertCloud(
		coreuser.AdminUserName,
		cld,
	)(context.Background(), s.TxnRunner(), s.NoopTxnRunner())
	c.Check(err, jc.ErrorIsNil)

	set := SetCloudDefaults("cirrus", map[string]any{
		"HTTP_PROXY": "[2001:0DB8::1]:80",
	})

	err = set(context.Background(), s.TxnRunner(), s.NoopTxnRunner())
	c.Check(err, jc.ErrorIsNil)

	var cloudUUID string
	err = s.TxnRunner().StdTxn(context.Background(), func(ctx context.Context, tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, "SELECT uuid FROM cloud WHERE name = ?", "cirrus").Scan(&cloudUUID)
	})
	c.Check(err, jc.ErrorIsNil)

	st := state.NewState(s.TxnRunnerFactory())
	defaults, err := st.CloudDefaults(context.Background(), corecloud.UUID(cloudUUID))
	c.Check(err, jc.ErrorIsNil)
	c.Check(defaults, jc.DeepEquals, map[string]string{
		"HTTP_PROXY": "[2001:0DB8::1]:80",
	})

	// Second time around

	set = SetCloudDefaults("cirrus", map[string]any{
		"foo": "bar",
	})

	err = set(context.Background(), s.TxnRunner(), s.NoopTxnRunner())
	c.Check(err, jc.ErrorIsNil)

	defaults, err = st.CloudDefaults(context.Background(), corecloud.UUID(cloudUUID))
	c.Check(err, jc.ErrorIsNil)
	c.Check(defaults, jc.DeepEquals, map[string]string{
		"foo": "bar",
	})
}
