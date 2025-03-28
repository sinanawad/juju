// Copyright 2013, 2015 Canonical Ltd.
// Copyright 2015 Cloudbase Solutions SRL
// Licensed under the AGPLv3, see LICENCE file for details.

package providerinit

import (
	"context"

	"github.com/juju/errors"

	"github.com/juju/juju/core/os/ostype"
	"github.com/juju/juju/internal/cloudconfig"
	"github.com/juju/juju/internal/cloudconfig/cloudinit"
	"github.com/juju/juju/internal/cloudconfig/instancecfg"
	"github.com/juju/juju/internal/cloudconfig/providerinit/renderers"
	internallogger "github.com/juju/juju/internal/logger"
)

var logger = internallogger.GetLogger("juju.cloudconfig.providerinit")

func configureCloudinit(icfg *instancecfg.InstanceConfig, cloudcfg cloudinit.CloudConfig) (cloudconfig.UserdataConfig, error) {
	// When bootstrapping, we only want to apt-get update/upgrade
	// and setup the SSH keys. The rest we leave to cloudinit/sshinit.
	udata, err := cloudconfig.NewUserdataConfig(icfg, cloudcfg)
	if err != nil {
		return nil, err
	}
	if icfg.Bootstrap != nil {
		err = udata.ConfigureBasic()
		if err != nil {
			return nil, err
		}
		err = udata.ConfigureCustomOverrides()
		if err != nil {
			return nil, err
		}
		return udata, nil
	}
	err = udata.Configure()
	if err != nil {
		return nil, err
	}
	return udata, nil
}

// ComposeUserData fills out the provided cloudinit configuration structure
// so it is suitable for initialising a machine with the given configuration,
// and then renders it and encodes it using the supplied renderer.
// When calling ComposeUserData a encoding implementation must be chosen from
// the providerinit/encoders package according to the need of the provider.
//
// If the provided cloudcfg is nil, a new one will be created internally.
func ComposeUserData(icfg *instancecfg.InstanceConfig, cloudcfg cloudinit.CloudConfig, renderer renderers.ProviderRenderer) ([]byte, error) {
	if cloudcfg == nil {
		var err error
		cloudcfg, err = cloudinit.New(icfg.Base.OS)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}
	_, err := configureCloudinit(icfg, cloudcfg)
	if err != nil {
		return nil, errors.Trace(err)
	}
	udata, err := renderer.Render(cloudcfg, ostype.OSTypeForName(icfg.Base.OS))
	if err != nil {
		return nil, errors.Trace(err)
	}
	logger.Tracef(context.TODO(), "Generated cloud init:\n%s", string(udata))
	return udata, err
}
