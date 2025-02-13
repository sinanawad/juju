(charm-channel)=
# Charm channel

A **charm channel** is a way to use a charm in a particular stage of development.

```{note}

The notion of 'channel' in charms is entirely parallel to the notion of 'channel' in snaps / the *craft world more generally.
> See more: [Snapcraft | Channels](https://snapcraft.io/docs/channels)

```


## Components

A charm channel consists of three pieces, in this order: `<track>/<risk>/<branch>`.

(charm-channel-track)=
### Track

The `<track>` is a way to collect multiple supported releases of your charm under the same name.

When deploying a charm, specifying a track is optional; if you don't specify any, the default option is the `latest`.

To ensure consistency between tracks of the same charm, tracks must comply with a guardrail.

<!--
 - [Track guardrail](#heading--track-guardrail)
-->

(charm-channel-track-guardrail)=
#### Track guardrail

A **track guardrail** is a regex generated by a Charmhub admin at the request of a charm author whose purpose is to ensure that any new track of the charm complies with the specific pattern selected by the charm author for the charm, usually in conformity with the pattern established by the upstream workload (e.g., no numbers, cf, e.g., [OpenStack](https://docs.openstack.org/charm-guide/latest/project/charm-delivery.html); numbers in the major.minor format; just integers; etc.)

<!--
Their format is usually modeled on the upstream workload. For example, some don't use numbers (e.g., [tracks for the Charmed OpenStack project](https://docs.openstack.org/charm-guide/latest/project/charm-delivery.html)); others use numbers in the major.minor format; others use just integers; etc. To ensure consistency between tracks of the same charm, tracks must comply with a guardrail -- a regex that is generated by a Charmhub admin and which will enforce the track pattern you have chosen.
-->


### Risk


The `<risk>` refers to one of the following risk levels:

-   **stable**: (default) This is the latest, tested, working stable version of the charm.
-   **candidate**: A release candidate. There is high confidence this will work fine, but there may be minor bugs.
-   **beta**: A beta testing milestone release.
-   **edge**: The very latest version - expect bugs!

### Branch

Finally, the `<branch>`  is an optional finer subdivision of a channel for a published charm that allows for the creation of short-lived sequences of charms (guaranteed for only 30 days without modification) that can be pushed on demand by charm developers to help with fixes or temporary experimentation. Note that, if you use `--channel` to specify a branch, you must specify a track and a risk level as well.


<!--
In addition to offering the latest stable version of each operator, Charmhub also allows users to download or deploy operators in different stages of development.  Some users may be interested in the bleeding edge (in development) version of an operator while others may be part of a beta test group tasked with evaluating the next release candidate for a particular operator.

Juju refers to these stages using the term *channel*. Borrowing [the definition from Snapcraft](https://snapcraft.io/docs/channels), a channel consists of three pieces, in this order: `<track>/<risk>/<branch>`
-->
