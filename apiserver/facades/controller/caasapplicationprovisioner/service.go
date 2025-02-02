// Copyright 2024 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package caasapplicationprovisioner

import (
	"context"

	"github.com/juju/juju/controller"
	"github.com/juju/juju/core/charm"
	"github.com/juju/juju/core/leadership"
	"github.com/juju/juju/core/life"
	"github.com/juju/juju/core/model"
	"github.com/juju/juju/core/unit"
	"github.com/juju/juju/core/watcher"
	"github.com/juju/juju/domain/application/service"
	"github.com/juju/juju/environs/config"
	internalcharm "github.com/juju/juju/internal/charm"
	"github.com/juju/juju/internal/charm/resource"
)

// ControllerConfigService provides the controller configuration.
type ControllerConfigService interface {
	// ControllerConfig returns the config values for the controller.
	ControllerConfig(ctx context.Context) (controller.Config, error)
	// WatchControllerConfig returns a watcher that returns keys for any
	// changes to controller config.
	WatchControllerConfig() (watcher.StringsWatcher, error)
}

// ModelConfigService provides access to the model configuration.
type ModelConfigService interface {
	// ModelConfig returns the current config for the model.
	ModelConfig(context.Context) (*config.Config, error)
	// Watch returns a watcher that returns keys for any changes to model
	// config.
	Watch() (watcher.StringsWatcher, error)
}

// ModelInfoService describe the service for interacting and reading the underlying
// model information.
type ModelInfoService interface {
	// GetModelInfo returns the readonly model information for the model in
	// question.
	GetModelInfo(context.Context) (model.ModelInfo, error)
}

// ApplicationService describes the service for accessing application scaling info.
type ApplicationService interface {
	SetApplicationScalingState(ctx context.Context, name string, scaleTarget int, scaling bool) error
	GetApplicationScalingState(ctx context.Context, name string) (service.ScalingState, error)
	GetApplicationScale(ctx context.Context, name string) (int, error)
	GetApplicationLife(ctx context.Context, name string) (life.Value, error)
	GetUnitLife(context.Context, unit.Name) (life.Value, error)
	GetCharmIDByApplicationName(ctx context.Context, name string) (charm.ID, error)
	GetCharmMetadataStorage(ctx context.Context, id charm.ID) (map[string]internalcharm.Storage, error)
	GetCharmMetadataResources(ctx context.Context, id charm.ID) (map[string]resource.Meta, error)
	IsCharmAvailable(ctx context.Context, id charm.ID) (bool, error)
	DestroyUnit(context.Context, unit.Name) error
	RemoveUnit(context.Context, unit.Name, leadership.Revoker) error
	UpdateCAASUnit(context.Context, unit.Name, service.UpdateCAASUnitParams) error
}
