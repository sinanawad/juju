// Copyright 2022 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package migrationtarget

import (
	"context"
	"reflect"

	"github.com/juju/juju/apiserver/facade"
	"github.com/juju/juju/core/facades"
	"github.com/juju/juju/core/model"
	"github.com/juju/juju/internal/errors"
)

// Register is called to expose a package of facades onto a given registry.
func Register(requiredMigrationFacadeVersions facades.FacadeVersions) func(registry facade.FacadeRegistry) {
	return func(registry facade.FacadeRegistry) {
		registry.MustRegisterForMultiModel("MigrationTarget", 4, func(stdCtx context.Context, ctx facade.MultiModelContext) (facade.Facade, error) {
			api, err := makeFacade(stdCtx, ctx, requiredMigrationFacadeVersions)
			if err != nil {
				return nil, errors.Errorf("making migration target version 4: %w", err)
			}
			return api, nil
		}, reflect.TypeOf((*API)(nil)))
	}
}

// makeFacade is responsible for constructing a new migration target facade and
// its dependencies.
func makeFacade(
	stdCtx context.Context,
	ctx facade.MultiModelContext,
	facadeVersions facades.FacadeVersions,
) (*API, error) {
	auth := ctx.Auth()
	st := ctx.State()
	if err := checkAuth(stdCtx, auth, st); err != nil {
		return nil, err
	}

	domainServices := ctx.DomainServices()

	modelMigrationServiceGetter := func(modelId model.UUID) ModelMigrationService {
		return ctx.DomainServicesForModel(modelId).ModelMigration()
	}
	modelAgentServiceGetter := func(modelId model.UUID) ModelAgentService {
		return ctx.DomainServicesForModel(modelId).Agent()
	}

	return NewAPI(
		ctx,
		auth,
		domainServices.ControllerConfig(),
		domainServices.ExternalController(),
		domainServices.Application(),
		domainServices.Upgrade(),
		modelAgentServiceGetter,
		modelMigrationServiceGetter,
		facadeVersions,
		ctx.LogDir(),
	)
}
