// Copyright 2026 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package ssh

// K8sLoginUser requests a Kubernetes login shell (SSHClient v7 onwards).
// It selects a shell, not an identity; the outer SSH hop authenticates and
// authorises the Juju user.
const K8sLoginUser = "juju-login"
