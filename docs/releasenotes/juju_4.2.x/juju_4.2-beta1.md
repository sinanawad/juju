(juju42beta1)=
# Juju 4.2-beta1

## Kubernetes SSH shells

`juju ssh` into a Kubernetes container now starts `bash` when the image has
`bash` but not `sh`. An image with neither shell reports that Juju could not
find a shell, instead of a container-runtime "executable file not found"
error. Login sessions retain Bash login profiles and can fall back to `sh`.

Commandless login-shell fallback through the controller requires an updated
client and controller. Explicit commands and raw SSH shell requests need only
an updated controller for fallback. `--direct` also supports older
controllers, including Juju 3.6. See
{ref}`juju-ssh-kubernetes-vs-machines` for shell selection and compatibility.
