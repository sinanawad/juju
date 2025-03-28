// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package remoterelations

import (
	"context"

	"github.com/juju/errors"
	"github.com/juju/names/v6"
	"gopkg.in/macaroon.v2"

	common "github.com/juju/juju/apiserver/common/crossmodel"
	"github.com/juju/juju/core/crossmodel"
	"github.com/juju/juju/rpc/params"
	"github.com/juju/juju/state"
)

// RemoteRelationsState provides the subset of global state required by the
// remote relations facade.
type RemoteRelationsState interface {
	common.Backend

	// WatchRemoteApplications returns a StringsWatcher that notifies of changes to
	// the lifecycles of the remote applications in the model.
	WatchRemoteApplications() state.StringsWatcher

	// WatchRemoteApplicationRelations returns a StringsWatcher that notifies of
	// changes to the life-cycles of relations involving the specified remote
	// application.
	WatchRemoteApplicationRelations(applicationName string) (state.StringsWatcher, error)

	// WatchRemoteRelations returns a StringsWatcher that notifies of changes to
	// the lifecycles of remote relations in the model.
	WatchRemoteRelations() state.StringsWatcher

	// RemoveRemoteEntity removes the specified entity from the remote entities collection.
	RemoveRemoteEntity(entity names.Tag) error

	// SaveMacaroon saves the given macaroon for the specified entity.
	SaveMacaroon(entity names.Tag, mac *macaroon.Macaroon) error
}

// ControllerConfigAPI provides the subset of common.ControllerConfigAPI
// required by the remote firewaller facade
type ControllerConfigAPI interface {
	// ControllerConfig returns the controller's configuration.
	ControllerConfig(context.Context) (params.ControllerConfigResult, error)

	// ControllerAPIInfoForModels returns the controller api connection details for the specified models.
	ControllerAPIInfoForModels(ctx context.Context, args params.Entities) (params.ControllerAPIInfoResults, error)
}

// TODO - CAAS(ericclaudejones): This should contain state alone, model will be
// removed once all relevant methods are moved from state to model.
type stateShim struct {
	common.Backend
	st *state.State
}

func (st stateShim) RemoveRemoteEntity(entity names.Tag) error {
	r := st.st.RemoteEntities()
	return r.RemoveRemoteEntity(entity)
}

func (st stateShim) GetToken(entity names.Tag) (string, error) {
	r := st.st.RemoteEntities()
	return r.GetToken(entity)
}

func (st stateShim) SaveMacaroon(entity names.Tag, mac *macaroon.Macaroon) error {
	r := st.st.RemoteEntities()
	return r.SaveMacaroon(entity, mac)
}

func (st stateShim) WatchRemoteApplications() state.StringsWatcher {
	return st.st.WatchRemoteApplications()
}

func (st stateShim) WatchRemoteRelations() state.StringsWatcher {
	return st.st.WatchRemoteRelations()
}

func (st stateShim) WatchRemoteApplicationRelations(applicationName string) (state.StringsWatcher, error) {
	a, err := st.st.RemoteApplication(applicationName)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return a.WatchRelations(), nil
}

func (st stateShim) ApplicationOfferForUUID(offerUUID string) (*crossmodel.ApplicationOffer, error) {
	offers := state.NewApplicationOffers(st.st)
	return offers.ApplicationOfferForUUID(offerUUID)
}
