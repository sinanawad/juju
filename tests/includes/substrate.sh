#!/usr/bin/env bash
# substrate.sh — Provider-aware substrate verification helpers.
#
# These functions verify that Juju operations are reflected on the actual
# infrastructure (K8s, LXD) rather than only in Juju's model. Each function
# detects the current provider from BOOTSTRAP_PROVIDER and calls the
# appropriate substrate tool (microk8s kubectl, lxc).
#
# Source: specs/002-ci-test-suite (Phase 4 — Substrate Verification)
# Sourced automatically by tests/main.sh via import_subdir_files.

# Default timeout for substrate verification polling (seconds).
SUBSTRATE_TIMEOUT="${SUBSTRATE_TIMEOUT:-60}"

# Polling interval for substrate checks (seconds).
SUBSTRATE_POLL_INTERVAL="${SUBSTRATE_POLL_INTERVAL:-5}"

# --- Internal helpers ---

# _substrate_provider returns the normalised provider kind: "k8s", "lxd", or "other".
_substrate_provider() {
	case "${BOOTSTRAP_PROVIDER:-}" in
	"microk8s" | "k8s")
		echo "k8s"
		;;
	"lxd")
		echo "lxd"
		;;
	*)
		echo "other"
		;;
	esac
}

# _substrate_namespace resolves the K8s namespace for a Juju model.
# Usage: _substrate_namespace [model]
# If model is omitted, uses the current model name from JUJU_MODEL or juju switch.
_substrate_namespace() {
	local model="${1:-}"
	if [[ -z ${model} ]]; then
		model=$(juju switch 2>/dev/null | cut -d: -f2 | cut -d/ -f1)
	fi
	echo "${model}"
}

# _substrate_poll retries a command until it succeeds or the timeout expires.
# Usage: _substrate_poll <timeout> <description> <command...>
_substrate_poll() {
	local timeout="${1}" desc="${2}"
	shift 2

	local start elapsed
	start=$(date +%s)
	while true; do
		if "$@" 2>/dev/null; then
			return 0
		fi
		elapsed=$(( $(date +%s) - start ))
		if (( elapsed >= timeout )); then
			echo "substrate verification: ${desc} — timed out after ${timeout}s" >&2
			return 1
		fi
		sleep "${SUBSTRATE_POLL_INTERVAL}"
	done
}

# --- K8s (MicroK8s) verification functions ---

# substrate_check_pod_exists verifies at least one pod exists for an application.
# Usage: substrate_check_pod_exists <app> [namespace]
substrate_check_pod_exists() {
	: # stub — Phase 4 (T014)
}

# substrate_check_pod_count verifies the number of running pods for an application.
# Usage: substrate_check_pod_count <app> <expected> [namespace]
substrate_check_pod_count() {
	: # stub — Phase 4 (T014)
}

# substrate_check_namespace_exists verifies a K8s namespace exists.
# Usage: substrate_check_namespace_exists <namespace>
substrate_check_namespace_exists() {
	: # stub — Phase 4 (T014)
}

# substrate_check_namespace_gone verifies a K8s namespace no longer exists.
# Usage: substrate_check_namespace_gone <namespace>
substrate_check_namespace_gone() {
	: # stub — Phase 4 (T014)
}

# substrate_check_pvc_exists verifies a PVC exists.
# Usage: substrate_check_pvc_exists <name> [namespace]
substrate_check_pvc_exists() {
	: # stub — Phase 4 (T014)
}

# substrate_check_pvc_count verifies the number of PVCs in a namespace.
# Usage: substrate_check_pvc_count <expected> [namespace]
substrate_check_pvc_count() {
	: # stub — Phase 4 (T014)
}

# substrate_check_service_exists verifies a K8s Service exists.
# Usage: substrate_check_service_exists <name> [namespace]
substrate_check_service_exists() {
	: # stub — Phase 4 (T014)
}

# --- LXD verification functions ---

# substrate_check_container_exists verifies a LXD container exists.
# Usage: substrate_check_container_exists <name>
substrate_check_container_exists() {
	: # stub — Phase 4 (T015)
}

# substrate_check_container_gone verifies a LXD container no longer exists.
# Usage: substrate_check_container_gone <name>
substrate_check_container_gone() {
	: # stub — Phase 4 (T015)
}

# substrate_check_container_count verifies the number of LXD containers matching a pattern.
# Usage: substrate_check_container_count <expected>
substrate_check_container_count() {
	: # stub — Phase 4 (T015)
}

# --- Provider-aware dispatch functions ---

# substrate_verify_deploy verifies an application's deployment on the substrate.
# Usage: substrate_verify_deploy <app> [model]
substrate_verify_deploy() {
	: # stub — Phase 4 (T016)
}

# substrate_verify_destroy_model verifies a model's resources are gone from the substrate.
# Usage: substrate_verify_destroy_model <model>
substrate_verify_destroy_model() {
	: # stub — Phase 4 (T016)
}

# substrate_verify_scale verifies an application has the expected number of units on the substrate.
# Usage: substrate_verify_scale <app> <expected> [model]
substrate_verify_scale() {
	: # stub — Phase 4 (T016)
}
