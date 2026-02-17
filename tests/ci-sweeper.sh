#!/usr/bin/env bash
# ci-sweeper.sh — Automated cleanup of stale CI resources.
#
# Sweeps Juju controllers, LXD containers, and K8s namespaces that were
# created by CI runs (matching the ci-* naming convention) and have exceeded
# the configured time-to-live (TTL).
#
# Designed to be idempotent — safe to run when no stale resources exist.
#
# Usage:
#   ./ci-sweeper.sh [--ttl-hours N] [--dry-run]
#
# Source: specs/002-ci-test-suite (Phase 2 — Resource Sweeper, FR-033-035)

set -euo pipefail

# --- Configuration ---

TTL_HOURS="${TTL_HOURS:-4}"
DRY_RUN="false"
SWEEP_JUJU="true"
SWEEP_LXD="true"
SWEEP_K8S="true"

# --- Argument parsing ---

while [[ $# -gt 0 ]]; do
	case "${1}" in
	--ttl-hours)
		TTL_HOURS="${2}"
		shift 2
		;;
	--dry-run)
		DRY_RUN="true"
		shift
		;;
	--no-juju)
		SWEEP_JUJU="false"
		shift
		;;
	--no-lxd)
		SWEEP_LXD="false"
		shift
		;;
	--no-k8s)
		SWEEP_K8S="false"
		shift
		;;
	-h | --help)
		echo "Usage: ci-sweeper.sh [--ttl-hours N] [--dry-run] [--no-juju] [--no-lxd] [--no-k8s]"
		echo ""
		echo "Sweep stale CI resources matching the ci-* naming convention."
		echo ""
		echo "Options:"
		echo "  --ttl-hours N   Hours before a resource is considered stale (default: 4)"
		echo "  --dry-run       Show what would be cleaned up without doing it"
		echo "  --no-juju       Skip Juju controller sweep"
		echo "  --no-lxd        Skip LXD container sweep"
		echo "  --no-k8s        Skip K8s namespace sweep"
		exit 0
		;;
	*)
		echo "Unknown option: ${1}" >&2
		exit 1
		;;
	esac
done

TTL_SECONDS=$((TTL_HOURS * 3600))
NOW=$(date +%s)
SWEPT=0
ERRORS=0

echo "==> CI Resource Sweeper"
echo "    TTL: ${TTL_HOURS}h (${TTL_SECONDS}s)"
echo "    Dry run: ${DRY_RUN}"
echo "    Timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo ""

# --- Helper functions ---

# controller_age_seconds returns the age of a Juju controller in seconds.
# Parses the controller's timestamp from `juju show-controller`.
controller_age_seconds() {
	local controller="${1}"
	local timestamp
	timestamp=$(juju show-controller "${controller}" --format=json 2>/dev/null \
		| jq -r ".[\"${controller}\"].details.\"api-endpoints\"[0]" 2>/dev/null || echo "")

	# If we can't get the timestamp from api-endpoints, fall back to checking
	# the controller's model creation time.
	timestamp=$(juju show-controller "${controller}" --format=json 2>/dev/null \
		| jq -r ".[\"${controller}\"].details.\"agent-version\" // empty" 2>/dev/null || echo "")

	# If we can't reliably determine age from the controller metadata, use
	# the models list and check the controller model's status timestamp.
	local ctrl_timestamp
	ctrl_timestamp=$(juju models -c "${controller}" --format=json 2>/dev/null \
		| jq -r '.models[] | select(.["is-controller"]) | .status["current-since"] // empty' 2>/dev/null || echo "")

	if [[ -z ${ctrl_timestamp} ]]; then
		# Cannot determine age — return max age so it gets swept.
		echo "${TTL_SECONDS}"
		return
	fi

	# Parse the timestamp. Juju timestamps are in RFC3339 or similar format.
	local created_epoch
	created_epoch=$(date -d "${ctrl_timestamp}" +%s 2>/dev/null || echo "0")
	if [[ ${created_epoch} -eq 0 ]]; then
		echo "${TTL_SECONDS}"
		return
	fi

	echo $(( NOW - created_epoch ))
}

# --- Juju controller sweep ---

sweep_juju_controllers() {
	echo "==> Sweeping Juju controllers matching ci-*"

	local controllers
	controllers=$(juju controllers --format=json 2>/dev/null \
		| jq -r '.controllers | keys[] | select(startswith("ci-"))' 2>/dev/null || echo "")

	if [[ -z ${controllers} ]]; then
		echo "    No ci-* controllers found."
		return
	fi

	local controller age age_hours
	while IFS= read -r controller; do
		[[ -z ${controller} ]] && continue

		age=$(controller_age_seconds "${controller}")
		age_hours=$(( age / 3600 ))

		if (( age >= TTL_SECONDS )); then
			echo "    STALE: ${controller} (age: ${age_hours}h, TTL: ${TTL_HOURS}h)"
			if [[ ${DRY_RUN} == "true" ]]; then
				echo "    [dry-run] Would destroy: ${controller}"
			else
				echo "    Destroying: ${controller}"
				if juju destroy-controller "${controller}" --destroy-all-models --destroy-storage --no-prompt -t 5m 2>&1; then
					echo "    Destroyed: ${controller}"
					SWEPT=$((SWEPT + 1))
				else
					echo "    ERROR: Failed to destroy ${controller}, attempting kill-controller" >&2
					juju kill-controller "${controller}" --no-prompt -t 2m 2>&1 || true
					ERRORS=$((ERRORS + 1))
				fi
			fi
		else
			echo "    OK: ${controller} (age: ${age_hours}h, TTL: ${TTL_HOURS}h)"
		fi
	done <<< "${controllers}"
}

# --- LXD container sweep ---

sweep_lxd_containers() {
	echo "==> Sweeping LXD containers matching ci-*"

	if ! command -v lxc &>/dev/null; then
		echo "    lxc not found — skipping LXD sweep."
		return
	fi

	local containers
	containers=$(lxc list --format=json 2>/dev/null \
		| jq -r '.[] | select(.name | startswith("ci-")) | .name' 2>/dev/null || echo "")

	if [[ -z ${containers} ]]; then
		echo "    No ci-* LXD containers found."
		return
	fi

	local container created_at created_epoch age age_hours
	while IFS= read -r container; do
		[[ -z ${container} ]] && continue

		created_at=$(lxc info "${container}" --format=json 2>/dev/null \
			| jq -r '.created_at // empty' 2>/dev/null || echo "")

		if [[ -z ${created_at} ]]; then
			age=${TTL_SECONDS}
		else
			created_epoch=$(date -d "${created_at}" +%s 2>/dev/null || echo "0")
			if [[ ${created_epoch} -eq 0 ]]; then
				age=${TTL_SECONDS}
			else
				age=$(( NOW - created_epoch ))
			fi
		fi

		age_hours=$(( age / 3600 ))

		if (( age >= TTL_SECONDS )); then
			echo "    STALE: ${container} (age: ${age_hours}h)"
			if [[ ${DRY_RUN} == "true" ]]; then
				echo "    [dry-run] Would delete: ${container}"
			else
				echo "    Deleting: ${container}"
				lxc delete "${container}" --force 2>&1 || {
					echo "    ERROR: Failed to delete LXD container ${container}" >&2
					ERRORS=$((ERRORS + 1))
				}
				SWEPT=$((SWEPT + 1))
			fi
		else
			echo "    OK: ${container} (age: ${age_hours}h)"
		fi
	done <<< "${containers}"
}

# --- K8s namespace sweep ---

sweep_k8s_namespaces() {
	echo "==> Sweeping K8s namespaces matching ci-*"

	if ! command -v microk8s &>/dev/null; then
		echo "    microk8s not found — skipping K8s sweep."
		return
	fi

	local namespaces
	namespaces=$(microk8s kubectl get namespaces -o json 2>/dev/null \
		| jq -r '.items[] | select(.metadata.name | startswith("ci-")) | .metadata.name' 2>/dev/null || echo "")

	if [[ -z ${namespaces} ]]; then
		echo "    No ci-* K8s namespaces found."
		return
	fi

	local ns created_at created_epoch age age_hours
	while IFS= read -r ns; do
		[[ -z ${ns} ]] && continue

		created_at=$(microk8s kubectl get namespace "${ns}" -o json 2>/dev/null \
			| jq -r '.metadata.creationTimestamp // empty' 2>/dev/null || echo "")

		if [[ -z ${created_at} ]]; then
			age=${TTL_SECONDS}
		else
			created_epoch=$(date -d "${created_at}" +%s 2>/dev/null || echo "0")
			if [[ ${created_epoch} -eq 0 ]]; then
				age=${TTL_SECONDS}
			else
				age=$(( NOW - created_epoch ))
			fi
		fi

		age_hours=$(( age / 3600 ))

		if (( age >= TTL_SECONDS )); then
			echo "    STALE: ${ns} (age: ${age_hours}h)"
			if [[ ${DRY_RUN} == "true" ]]; then
				echo "    [dry-run] Would delete namespace: ${ns}"
			else
				echo "    Deleting namespace: ${ns}"
				microk8s kubectl delete namespace "${ns}" --timeout=5m 2>&1 || {
					echo "    ERROR: Failed to delete K8s namespace ${ns}" >&2
					ERRORS=$((ERRORS + 1))
				}
				SWEPT=$((SWEPT + 1))
			fi
		else
			echo "    OK: ${ns} (age: ${age_hours}h)"
		fi
	done <<< "${namespaces}"
}

# --- Main ---

if [[ ${SWEEP_JUJU} == "true" ]]; then
	sweep_juju_controllers
	echo ""
fi

if [[ ${SWEEP_LXD} == "true" ]]; then
	sweep_lxd_containers
	echo ""
fi

if [[ ${SWEEP_K8S} == "true" ]]; then
	sweep_k8s_namespaces
	echo ""
fi

echo "==> Sweep complete: ${SWEPT} resources cleaned, ${ERRORS} errors"
if [[ ${ERRORS} -gt 0 ]]; then
	echo "    WARNING: ${ERRORS} cleanup errors occurred — manual intervention may be needed."
	exit 1
fi
exit 0
