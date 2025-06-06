// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package juju

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"slices"
	"strings"

	"github.com/juju/errors"
	"github.com/juju/names/v6"

	"github.com/juju/juju/api"
	"github.com/juju/juju/core/network"
	internallogger "github.com/juju/juju/internal/logger"
	"github.com/juju/juju/internal/proxy"
	"github.com/juju/juju/jujuclient"
)

var logger = internallogger.GetLogger("juju.juju")

// NewAPIConnectionParams contains the parameters for creating a new Juju API
// connection.
type NewAPIConnectionParams struct {
	// ControllerName is the name of the controller to connect to.
	ControllerName string

	// ControllerStore is the jujuclient.ControllerStore from which the controller's
	// details will be fetched, and updated on address changes.
	ControllerStore jujuclient.ControllerStore

	// OpenAPI is the function that will be used to open API connections.
	OpenAPI api.OpenFunc

	// DialOpts contains the options used to dial the API connection.
	DialOpts api.DialOpts

	// AccountDetails contains the account details to use for logging
	// in to the Juju API. If this is nil, then no login will take
	// place. If AccountDetails.Password and AccountDetails.Macaroon
	// are zero, the login will be as an external user.
	// Updates to this value will be saved to the client store after login.
	AccountDetails *jujuclient.AccountDetails

	// ModelUUID is an optional model UUID. If specified, the API connection
	// will be scoped to the model with that UUID; otherwise it will be
	// scoped to the controller.
	ModelUUID string

	// APIEndpoints, if set, override any other api endpoints.
	APIEndpoints []string
}

var errNoAddresses = errors.ConstError("no API addresses")

// IsNoAddressesError reports whether the error (from NewAPIConnection) is an
// error due to the controller having no API addresses yet (likely because a
// bootstrap is still in progress).
func IsNoAddressesError(err error) bool {
	return errors.Is(err, errNoAddresses)
}

// NewAPIConnection returns an api.Connection to the specified Juju controller,
// with specified account credentials, optionally scoped to the specified model
// name.
func NewAPIConnection(ctx context.Context, args NewAPIConnectionParams) (_ api.Connection, err error) {
	if args.OpenAPI == nil {
		args.OpenAPI = api.Open
	}
	apiInfo, controller, err := connectionInfo(args)
	if err != nil {
		if errors.Is(err, errors.NotValid) {
			err = errors.NewNotValid(nil, fmt.Sprintf("%v\n"+
				"A user name may contain any case alpha-numeric characters, '+', '.', and '-'; \n"+
				"'@' to specify an optional domain. The user name and domain must begin and end \n"+
				"with alpha-numeric characters. Examples of valid users include bob, Bob@local, bob@somewhere-else, 0-a-f@123", err))
			return nil, errors.Trace(err)
		}
		return nil, errors.Annotatef(err, "cannot work out how to connect")
	}
	if len(apiInfo.Addrs) == 0 {
		return nil, errNoAddresses
	}

	// Copy the cache so we'll know whether it's changed so that
	// we'll update the entry correctly.
	dnsCache := dnsCacheMap(controller.DNSCache).copy()
	args.DialOpts.DNSCache = dnsCache
	logger.Infof(context.TODO(), "connecting to API addresses: %v", apiInfo.Addrs)
	st, err := args.OpenAPI(ctx, apiInfo, args.DialOpts)
	if err != nil {
		var redirErr *api.RedirectError
		if !errors.As(err, &redirErr) || !redirErr.FollowRedirect {
			return nil, errors.Trace(err)
		}
		// We've been told to connect to a different API server,
		// so do so. Note that we don't copy the account details
		// because the account on the redirected server may well
		// be different - we'll use macaroon authentication
		// directly without sending account details.
		// Copy the API info because it's possible that the
		// apiConfigConnect is still using it concurrently.
		apiInfo = &api.Info{
			ModelTag:       apiInfo.ModelTag,
			Addrs:          usableHostPorts(redirErr.Servers).Strings(),
			CACert:         redirErr.CACert,
			ControllerUUID: apiInfo.ControllerUUID,
		}
		st, err = args.OpenAPI(ctx, apiInfo, args.DialOpts)
		if err != nil {
			return nil, errors.Annotatef(err, "cannot connect to redirected address")
		}
		// TODO(rog) update cached model addresses.
		// TODO(rog) should we do something with the logged-in username?
		return st, nil
	}
	defer func() {
		if err != nil {
			_ = st.Close()
		}
	}()

	// If the account details are set, ensure that the user we've logged in as
	// matches the user we expected to log in as.
	// This only applies if the user was explicitly set.
	// Logging in via an external auth provider is allowed with specifying a
	// user in the args - that comes from the provider.
	if args.AccountDetails != nil &&
		st.AuthTag() != nil &&
		args.AccountDetails.User != "" &&
		args.AccountDetails.User != st.AuthTag().Id() {
		return nil, errors.Unauthorizedf("attempted login as %q for user %q", st.AuthTag().Id(), args.AccountDetails.User)
	}

	// Update API addresses if they've changed. Error is non-fatal.
	// Note that in the redirection case, we won't update the addresses
	// of the controller we first connected to. This shouldn't be
	// a problem in practice because the intended scenario for
	// controllers that redirect involves them having well known
	// public addresses that won't change over time.
	hostPorts := st.APIHostPorts()
	agentVersion := ""
	if v, ok := st.ServerVersion(); ok {
		agentVersion = v.String()
	}

	params := UpdateControllerParams{
		AgentVersion:     agentVersion,
		CurrentHostPorts: hostPorts,
		DNSCache:         dnsCache,
		CurrentConnection: &currentConnection{
			Proxied:   st.IsProxied(),
			Address:   st.Addr(),
			IPAddress: st.IPAddr(),
		},
	}
	if host := st.PublicDNSName(); host != "" {
		params.PublicDNSName = &host
	}
	err = updateControllerDetailsFromLogin(args.ControllerStore, args.ControllerName, controller, params)
	if err != nil {
		logger.Errorf(context.TODO(), "cannot cache API addresses: %v", err)
	}

	return st, nil
}

// connectionInfo returns connection information suitable for
// connecting to the controller and model specified in the given
// parameters. If there are no addresses known for the controller,
// it may return a *api.Info with no APIEndpoints, but all other
// information will be populated.
func connectionInfo(args NewAPIConnectionParams) (*api.Info, *jujuclient.ControllerDetails, error) {
	controller, err := args.ControllerStore.ControllerByName(args.ControllerName)
	if err != nil {
		return nil, nil, errors.Annotate(err, "cannot get controller details")
	}
	if args.AccountDetails == nil {
		return nil, nil, errors.New("empty account details")
	}
	apiInfo := &api.Info{
		Addrs:          controller.APIEndpoints,
		CACert:         controller.CACert,
		ControllerUUID: controller.ControllerUUID,
	}
	if len(args.APIEndpoints) > 0 {
		apiInfo.Addrs = args.APIEndpoints
	}
	if controller.Proxy != nil {
		apiInfo.Proxier = controller.Proxy.Proxier
	}
	if args.ModelUUID != "" {
		apiInfo.ModelTag = names.NewModelTag(args.ModelUUID)
	}
	if controller.PublicDNSName != "" {
		apiInfo.SNIHostName = controller.PublicDNSName
	}
	account := args.AccountDetails
	if account.User != "" {
		if !names.IsValidUser(account.User) {
			return nil, nil, errors.NotValidf("user name %q", account.User)
		}
		userTag := names.NewUserTag(account.User)
		if userTag.IsLocal() {
			apiInfo.Tag = userTag
		}
	}
	if args.AccountDetails.Password != "" {
		// If a password is available, we always use that.
		// If no password is recorded, we'll attempt to
		// authenticate using macaroons.
		apiInfo.Password = account.Password
	} else {
		// Optionally the account may have macaroons to use.
		apiInfo.Macaroons = account.Macaroons
	}
	return apiInfo, controller, nil
}

// usableHostPorts returns the input MachineHostPort slice as DialAddresses
// with unusable and non-unique values filtered out.
func usableHostPorts(hps []network.MachineHostPorts) network.HostPorts {
	collapsed := network.CollapseToHostPorts(hps)
	return collapsed.FilterUnusable().Unique()
}

// addrsChanged reports whether the two slices are different.
// The first return tells if they are different in any way (including reordering),
// the second indicates if the set of addresses are different.
func addrsChanged(a, b []string) (bool, bool) {
	if len(a) != len(b) {
		return true, true
	}
	aKeys := make(map[string]struct{}, len(a))
	for _, k := range a {
		aKeys[k] = struct{}{}
	}
	outOfOrder := false
	for i := range a {
		bKey := b[i]
		if _, ok := aKeys[bKey]; ok {
			delete(aKeys, bKey)
		} else {
			// b has a key that is not in a, therefore we must not match at all
			return true, true
		}
		if a[i] != b[i] {
			outOfOrder = true
		}
	}
	return outOfOrder, false
}

// currentConnection represents information
// about a recently established connection.
type currentConnection struct {
	// Proxied indicates if the connection was proxied.
	Proxied bool

	// Address is an API address that has been recently
	// connected to.
	Address *url.URL

	// IPAddress is the IP address of Address
	// that has been recently connected to.
	IPAddress string
}

// UpdateControllerParams holds values used to update a controller details
// after bootstrap or a login operation.
type UpdateControllerParams struct {
	// AgentVersion is the version of the controller agent.
	AgentVersion string

	// CurrentHostPorts are the available api addresses.
	CurrentHostPorts []network.MachineHostPorts

	// CurrentConnection provides information on the address
	// we are connected to.
	CurrentConnection *currentConnection

	// Proxier
	Proxier proxy.Proxier

	// DNSCache holds entries in the DNS cache.
	DNSCache map[string][]string

	// PublicDNSName (when set) holds the public host name of the controller.
	PublicDNSName *string

	// ControllerMachineCount (when set) is the total number of controller machines in the environment.
	ControllerMachineCount *int

	// MachineCount (when set) is the total number of machines in the models.
	MachineCount *int
}

// UpdateControllerDetailsFromLogin writes any new api addresses and other relevant details
// to the client controller file.
// Controller may be specified by a UUID or name, and must already exist.
func UpdateControllerDetailsFromLogin(
	store jujuclient.ControllerStore, controllerName string,
	params UpdateControllerParams,
) error {
	controllerDetails, err := store.ControllerByName(controllerName)
	if err != nil {
		return errors.Trace(err)
	}
	return updateControllerDetailsFromLogin(store, controllerName, controllerDetails, params)
}

func updateControllerDetailsFromLogin(
	store jujuclient.ControllerStore,
	controllerName string, details *jujuclient.ControllerDetails,
	params UpdateControllerParams,
) error {
	addresses := makeUsableAddresses(&params)

	newDetails := new(jujuclient.ControllerDetails)
	*newDetails = *details

	if params.Proxier != nil {
		newDetails.Proxy = &jujuclient.ProxyConfWrapper{
			Proxier: params.Proxier,
		}
	}

	newDetails.AgentVersion = params.AgentVersion
	newDetails.APIEndpoints = addresses
	newDetails.DNSCache = params.DNSCache
	if params.MachineCount != nil {
		newDetails.MachineCount = params.MachineCount
	}
	if params.ControllerMachineCount != nil {
		newDetails.ControllerMachineCount = *params.ControllerMachineCount
	}
	if params.PublicDNSName != nil {
		newDetails.PublicDNSName = *params.PublicDNSName
	}
	if reflect.DeepEqual(newDetails, details) {
		// Nothing has changed - no need to update the controller details.
		return nil
	}
	reordered, diffContents := addrsChanged(newDetails.APIEndpoints, details.APIEndpoints)
	if diffContents {
		logger.Infof(context.TODO(), "API endpoints changed from %v to %v", details.APIEndpoints, newDetails.APIEndpoints)
	} else if reordered {
		logger.Tracef(context.TODO(), "API endpoints reordered from %v to %v", details.APIEndpoints, newDetails.APIEndpoints)
	}
	err := store.UpdateController(controllerName, *newDetails)
	return errors.Trace(err)
}

// makeUsableAddresses returns a list of controller addresses
// in a format appropriate for persisting in the Juju client store.
// The addresses will be a URL but omit any scheme i.e. <domain>:<port>/<path>
// The addresses are filtered to only those unique and usable.
// Finally, the params.DNSCache will be updated.
func makeUsableAddresses(params *UpdateControllerParams) []string {
	addresses := usableHostPorts(params.CurrentHostPorts).Strings()

	// Ignore the currently connected address if there is no current
	// connection (during bootstrap) or if the connection is proxied.
	if params.CurrentConnection == nil ||
		params.CurrentConnection.Address == nil ||
		params.CurrentConnection.Proxied {
		return addresses
	}

	connectedUrl := params.CurrentConnection.Address
	urlWithoutScheme := connectedUrl.Host + connectedUrl.RequestURI()
	urlWithoutScheme, _ = strings.CutSuffix(urlWithoutScheme, "/")

	// Move the connected-to host to the front of the address list.
	if !slices.Contains(addresses, urlWithoutScheme) {
		addresses = slices.Insert(addresses, 0, urlWithoutScheme)
	}

	// Move the IP address used to the front of the DNS cache entry
	// so that it will be the first address dialed.
	ipHost, _, err := net.SplitHostPort(params.CurrentConnection.IPAddress)
	if err == nil {
		host := connectedUrl.Hostname()
		moveToFront(ipHost, params.DNSCache[host])
	}
	return addresses
}

// dnsCacheMap implements api.DNSCache by
// caching entries in a map.
type dnsCacheMap map[string][]string

func (m dnsCacheMap) Lookup(host string) []string {
	return m[host]
}

func (m dnsCacheMap) copy() dnsCacheMap {
	m1 := make(dnsCacheMap)
	for host, ips := range m {
		m1[host] = append([]string{}, ips...)
	}
	return m1
}

func (m dnsCacheMap) Add(host string, ips []string) {
	m[host] = append([]string{}, ips...)
}

// moveToFront moves the given item (if present)
// to the front of the given slice.
func moveToFront(item string, xs []string) {
	for i, x := range xs {
		if x != item {
			continue
		}
		if i == 0 {
			return
		}
		copy(xs[1:], xs[0:i])
		xs[0] = item
		return
	}
}
