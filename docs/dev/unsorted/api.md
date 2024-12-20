(api)=
# API Design

The overall aim is to make agents and clients connect through a network
API rather than directly to the underlying database as is the case
currently.

This will have a few advantages:

- Operations that involve multiple round trips to the database can be
made more efficient because the API server is likely to be closer to
the database server.

- We can decide on, and enforce, an appropriate authorization policy
for each operation.

- The API can be made easy to use from multiple languages and client
types.

There are two general kinds of operation on the API: simple
requests and watch requests. I'll deal with simple requests
first.

* Simple requests

A simple request takes some parameters, possibly makes some changes to
the state database, and returns some results or an error.

When the request returns no data, it would theoretically be possible to
have the API server operate on the request without returning a reply,
but then the client would not know when the request has completed or if
it completed successfully. Therefore, I think it's better if all requests
return a reply.

Here is the list of all the State requests that are currently used by the
juju agents:
	XXX link
We will need to implement at least these requests (possibly
slightly changed, but hopefully as little as possible, to ensure as
little churn as possible in the agent code when moving to using the API)

41 out of the 59 requests are operating directly on a single state
entity, expressed as the receiver object in the Go API. For this reason
I believe it's appropriate to phrase the API requests in this way -
as requests on particular entities in the state. This leads to certain
implementation advantages (see "Implementation" below) and means there's
a close correspondence between the API protocol and the API as implemented
in Go (and hopefully other languages too).

To make the protocol accessible, we define all messages to be in JSON
format and we use a secure websocket for transport.
For security, we currently rely on a server-side certificate
and passwords sent over the connection to identify the client,
but it should be straightforward to enable the server to
do client certificate checking if desired.

Here's a sample request to change the instance id
associated with a machine, and its reply.  (I'll show JSON in rjson form, to keep the
noise down, see http://godoc.org/launchpad.net/rjson).

Client->Server
	{
		RequestId: 1234
		Type: "Machine"
		Id: "99"
		Request: "SetInstanceId"
		Params: {
			InstanceId: "i-43e55e5"
		}
	}
Server->Client
	{
		RequestId: 1234
		Error: ""
		Result: {
		}
	}

We use the RequestId field to associate the request and its
reply. The API client must not re-use a request id until
it has received the request's reply (the easiest way
to do that is simply to increment the request id each time).
We allow multiple requests to be outstanding on a connection
at once, and their replies can be received in any order.

In the request, the Id field may be omitted to specify
an empty Id, and Params may be omitted
to specify no request parameters. Similarly, in the
response, the Error field may be omitted to
signify no error, and the Result field may be
omitted to signify no result. To save space below,
I've omitted fields accordingly.

The Type field identifies the type of entity to act on,
and the Id field its identifier. Currently I envisage
the following types of entities:

	Admin
		Admin (a singleton) is used by a client when identifying itself
		to the server. It is the only thing that can be accessed
		before the client has authenticated.

	Client
	ClientWatcher
		Client (a singleton) is the access point for all the Dashboard client
		and other user-facing methods. This is only
		usable by clients, not by agents.

	State
	Machine
	Unit
	Relation
	RelationUnit
	Service
	Pinger
	MachineWatcher
	UnitWatcher
	LifecycleWatcher
	ServiceUnitsWatcher
	RemoteRelationsWatcher
	RelationScopeWatcher
	UnitsWatcher
	ConfigWatcher
	NotifyWatcher
	MachineUnitsWatcher
		These correspond directly to types exported by the
		juju state package.  They are usable only by agents,
		not clients.

The Request field specifies the action to perform, and Params holds the
parameters to that request.

In the reply message, the RequestId field must match that of the
request. If the request failed, then the Error field holds the description
of the error (it is possible we might add a Code field later, to help
diagnosing specific kinds of error).

The Result field holds the results of the request (in this case there
are none, so it's empty).

That completes the overview of simple requests,
so on to watching.

* Watching

To watch something in the state, we invoke a Watch request, which
returns a handle to a watcher object, that can then be used to find
out when changes happen by calling its Next method. To stop a watcher,
we call Stop on it.

For example, if an agent wishes to watch machine 99, the conversation
with the API server looks something like this:

Client->Server
	{
		RequestId: 1000
		Type: "Machine"
		Id: "99"
		Request: "Watch"
	}
Server->Client
	{
		RequestId: 1000
		Response: {
			NotifyWatcherId: "1"
		}
	}

At this point, the watcher is registered.  Subsequent Next calls will
only return when the entity has changed.

Client->Server
	{
		RequestId: 1001
		Type: "NotifyWatcher"
		Id: "1"
		Request: "Next"
	}

This reply will only sent when something has changed.  Note that for this
particular watcher, no data is sent with the Next response. This can vary
according to the particular kind of watcher - some watchers may return
deltas, for example, or the latest value of the thing being watched.

Server->Client
	{
		RequestId: 1001
	}

The client can carry on sending Next requests for
as long as it chooses, each one returning only
when the machine has changed since the previous
Next request.

Client->Server
	{
		RequestId: 1002
		Type: "NotifyWatcher"
		Id: "1"
		Request: "Next"
	}

Finally, the client decides to stop the watcher. This
causes any outstanding Next request to return too -
in no particular order with respect to the Stop reply.

Client->Server
	{
		RequestId: 1003
		Type: "NotifyWatcher"
		Id: "1"
		Request: "Stop"
	}
Server->Client
	{
		RequestId: 1002
	}
Server->Client
	{
		RequestId: 1003
	}

As you can see, we use exactly the same RPC mechanism for watching as
for simple requests. An alternative would have been to push watch change
notifications to clients without waiting for an explicit request.

Both schemes have advantages and disadvantages.  I've gone with the
above scheme mainly because it makes the protocol more obviously correct
in the face of clients that are not reading data fast enough - in the
face of a client with a slow network connection, we will not continue
saturating its link with changes that cannot be passed through the
pipe fast enough. Because Juju is state-based rather than event-based,
the number of possible changes is bounded by the size of the system,
so even if a client is very slow at reading the number of changes pushed
down to it will not grow without bound.

Allocating a watcher per client also implies that the server must
keep some per-client state, but preliminary measurements indicate
that the cost of that is unlikely to be prohibitive.

Using exactly the same mechanism for all interactions with the API has
advantages in simplicity too.

* Authentication and authorization

The API server is authenticated by TLS handshake before the
websocket connection is initiated; the client should check that
the server's certificate is signed by a trusted CA (in particular
the CA that's created as a part of the bootstrap process).

One wrinkle here is that before bootstrapping, we don't
know the DNS name of the API server (in general, from
a high-availability standpoint, we want to be able to serve the API from
any number of servers), so we cannot put it into
the certificate that we generate for the API server.
This doesn't sit well with the way that www authentication
usually works - hopefully there's a way around it in node.

The client authenticates to the server currently by providing
a user name and password in a Login request:

Client->Server
	{
		RequestId: 1
		Type: "Admin"
		Request: "Login"
		Params: {
			"Tag": "machine-1",
			Password: "a2eaa54323ae",
		}
	}
Server->Client
	{
		RequestId: 1
	}

Until the user has successfully logged in, the Login
request is the only one that the server will respond
to - all other requests yield a "permission denied"
error.

The exact form of the Login request is subject to change,
depending on what kind user authentication we might
end up with - it may even end up as two or more requests,
going through different stages of some authentication
process.

When logged in, requests are authorized both at the
type level (to filter out obviously inappropriate requests,
such as a client trying to access the agent API) and
at the request level (allowing a more fine-grained
approach).

* Versioning

I'm not currently sure of the best approach to versioning.
One possibility is to have a Version request that
allows the client to specify a desired version number;
the server could then reply with a lower (or
equal) version. The server would then serve the
version of the protocol that it replied with.

Unfortunately, this adds an extra round trip to the
session setup. This could be mitigated by sending
both the Version and the Login requests at the same
time.

* Implementation

The Go stack consists of the following levels (high to low):

	client interface ("github.com/juju/juju/state/api".State)
	rpc package ("github.com/juju/juju/rpc".Client)
	----- (json transport over secure websockets, implemented by 3rd party code)
	rpc package ("github.com/juju/juju/rpc".Server)
	server implementation ("github.com/juju/juju/state/api".Server)
	server backend ("github.com/juju/juju/state")
	mongo data store


