test_ssh_machine() {
	if [ "$(skip 'test_ssh_machine')" ]; then
		echo "==> TEST SKIPPED: machine SSH and SCP tests"
		return
	fi

	local model_name local_file remote_file copied_file output key_file
	local machine_hostname
	model_name="test-ssh-machine"
	add_model "${model_name}"
	juju deploy juju-qa-test ssh-test --base ubuntu@22.04
	wait_for "ssh-test" "$(idle_condition "ssh-test")"

	# Verify command execution on the machine hosting the unit. The
	# hostname must match the one Juju recorded for the machine at agent
	# start, which proves the command ran on the machine hosting the unit
	# regardless of the hostname conventions of the cloud.
	# shellcheck disable=SC2016
	output=$(juju ssh ssh-test/0 -- printf 'ssh-machine:%s $(hostname)')
	machine_hostname=$(juju show-machine 0 --format json |
		yq -r '.machines["0"]["hostname"]')
	if [[ -z ${machine_hostname} ]]; then
		echo "ERROR: machine 0 has no hostname"
		return 1
	fi
	check_contains "${output}" "ssh-machine:${machine_hostname}"

	# Verify command execution using the machine target syntax.
	output=$(juju ssh 0 -- printf machine-target)
	check_contains "${output}" "machine-target"

	local_file="${TEST_DIR}/machine-local.txt"
	remote_file="/tmp/juju-ssh-suite.txt"
	copied_file="${TEST_DIR}/machine-copied.txt"
	echo "machine-scp-test" >"${local_file}"

	# Copy to the machine and read the result remotely.
	juju scp "${local_file}" "0:${remote_file}"
	output=$(juju ssh 0 -- cat "${remote_file}")
	check_contains "${output}" "machine-scp-test"

	# Copy back from the machine and verify the local file contents.
	juju scp "0:${remote_file}" "${copied_file}"
	check_contains "$(cat "${copied_file}")" "machine-scp-test"

	# A key registered on a different model must be refused here, with the
	# key comment included in the error to help identify the offending key.
	key_file="${TEST_DIR}/other-model-key"
	ssh-keygen -q -t ed25519 -N "" -C "other-model-key" -f "${key_file}"
	add_model "${model_name}-other"
	juju add-ssh-key -m "${model_name}-other" "$(cat "${key_file}.pub")"

	juju switch "${model_name}"
	output=$(juju ssh --ssh-key "${key_file}" ssh-test/0 -- printf should-not-run 2>&1 || true)
	check_contains "${output}" 'public key "other-model-key" used to authenticate is not associated with model'
	destroy_model "${model_name}-other"

	destroy_model "${model_name}"
}

test_ssh_k8s() {
	if [ "$(skip 'test_ssh_k8s')" ]; then
		echo "==> TEST SKIPPED: Kubernetes SSH and SCP tests"
		return
	fi

	local model_name local_file remote_file copied_file output key_file
	model_name="test-ssh-k8s"
	add_model "${model_name}"
	juju deploy snappass-test ssh-test
	wait_for "ssh-test" "$(idle_condition "ssh-test")"

	# The default target is the charm container. The explicit container target
	# exercises the same route used by sidecar workloads.
	output=$(juju ssh ssh-test/0 -- printf operator-container)
	check_contains "${output}" "operator-container"

	output=$(juju ssh --container redis ssh-test/0 -- printf workload-container)
	check_contains "${output}" "workload-container"

	local_file="${TEST_DIR}/k8s-local.txt"
	remote_file="/tmp/juju-ssh-suite.txt"
	copied_file="${TEST_DIR}/k8s-copied.txt"
	echo "k8s-scp-test" >"${local_file}"

	# Copy to and from the sidecar container.
	juju scp --container redis "${local_file}" "ssh-test/0:${remote_file}"
	output=$(juju ssh --container redis ssh-test/0 -- cat "${remote_file}")
	check_contains "${output}" "k8s-scp-test"

	juju scp --container redis "ssh-test/0:${remote_file}" "${copied_file}"
	check_contains "$(cat "${copied_file}")" "k8s-scp-test"

	# A key registered on a different model must be refused here, with the
	# key comment included in the error to help identify the offending key.
	key_file="${TEST_DIR}/other-model-key"
	ssh-keygen -q -t ed25519 -N "" -C "other-model-key" -f "${key_file}"
	add_model "${model_name}-other"
	juju add-ssh-key -m "${model_name}-other" "$(cat "${key_file}.pub")"

	juju switch "${model_name}"
	output=$(juju ssh --ssh-key "${key_file}" ssh-test/0 -- printf should-not-run 2>&1 || true)
	check_contains "${output}" 'public key "other-model-key" used to authenticate is not associated with model'
	destroy_model "${model_name}-other"

	destroy_model "${model_name}"
}

# test_ssh_k8s_shell exercises both SSH routes with bash-only, sh-only,
# both-shell and shell-less workloads. Set SSH_TEST_BASH_IMAGE to opt in with
# a cluster-reachable image containing bash but no sh, such as bash_bins.
test_ssh_k8s_shell() {
	if [ "$(skip 'test_ssh_k8s_shell')" ]; then
		echo "==> TEST SKIPPED: Kubernetes SSH shell probes"
		return
	fi
	if [[ -z ${SSH_TEST_BASH_IMAGE:-} ]]; then
		echo "==> TEST SKIPPED: Kubernetes SSH shell probes; set SSH_TEST_BASH_IMAGE to enable"
		return
	fi

	local model_name charm_file output repo_root bash_image route app status expected_status
	local key_file raw_command pty
	local -a ssh_options
	model_name="test-ssh-k8s-shell"
	repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)

	check_dependencies charmcraft python3
	bash_image=${SSH_TEST_BASH_IMAGE}

	charm_file=$(pack_resource_charm "${repo_root}/testcharms/charms/ssh-shell" "${TEST_DIR}/ssh-shell")

	add_model "${model_name}"

	juju deploy "${charm_file}" ssh-bash \
		--resource "workload=${bash_image}"
	wait_for "ssh-bash" "$(idle_condition "ssh-bash")"
	juju deploy "${charm_file}" ssh-sh \
		--resource "workload=${SSH_TEST_SH_IMAGE:-busybox:1.36.1}"
	wait_for "ssh-sh" "$(idle_condition "ssh-sh")"
	juju deploy "${charm_file}" ssh-both \
		--resource "workload=${SSH_TEST_BOTH_IMAGE:-ubuntu:24.04}"
	wait_for "ssh-both" "$(idle_condition "ssh-both")"
	juju deploy "${charm_file}" ssh-noshell \
		--resource "workload=${SSH_TEST_NOSHELL_IMAGE:-registry.k8s.io/pause:3.10}"
	wait_for "ssh-noshell" "$(idle_condition "ssh-noshell")"
	# A login shell must read this profile, including in ordinary workloads.
	juju ssh --container workload --pty=false ssh-both/0 \
		"printf 'export JUJU_SSH_PROFILE=loaded\\n' > /etc/profile.d/juju-ssh-test.sh"

	for route in controller direct; do
		ssh_options=(--container workload)
		expected_status=7
		if [[ ${route} == direct ]]; then
			ssh_options+=(--direct)
			expected_status=1
		fi

		# Check the fixtures too: a replacement image containing both shells
		# would otherwise make a fallback test pass without taking the retry.
		# shellcheck disable=SC2016
		output=$(juju ssh "${ssh_options[@]}" --pty=false ssh-bash/0 \
			'test -n "$BASH_VERSION" && ! command -v sh && printf BASH_ONLY')
		check_contains "${output}" BASH_ONLY
		# shellcheck disable=SC2016
		output=$(juju ssh "${ssh_options[@]}" --pty=false ssh-sh/0 \
			'test -z "$BASH_VERSION" && ! command -v bash && printf SH_ONLY')
		check_contains "${output}" SH_ONLY

		for app in ssh-bash ssh-sh ssh-both; do
			output=$(juju ssh "${ssh_options[@]}" --pty=false "${app}/0" echo SSH_OK)
			check_contains "${output}" SSH_OK

			# Include two lines to catch both initial and later input loss.
			output=$(printf 'echo STDIN_FIRST\necho STDIN_LAST\n' |
				juju ssh "${ssh_options[@]}" --pty=false "${app}/0" 2>&1)
			check_contains "${output}" STDIN_FIRST
			check_contains "${output}" STDIN_LAST

			status=0
			output=$(juju ssh "${ssh_options[@]}" --pty=false "${app}/0" \
				'printf "REMOTE_ERROR\n" >&2; exit 7' 2>&1) || status=$?
			if [[ ${status} -ne ${expected_status} ]]; then
				echo "${route}: exit 7 in ${app} returned ${status}, expected ${expected_status}" >&2
				return 1
			fi
			check_contains "${output}" REMOTE_ERROR
			# Controller SSH passes through the remote status silently. Direct
			# exec reports it in a diagnostic and returns the CLI error status.
			if [[ ${route} == direct ]]; then
				check_contains "${output}" 'exit.*7'
			fi
			check_not_contains "${output}" "couldn't find any shell"
		done

		# Explicit commands retain sh preference on an image with both shells.
		# shellcheck disable=SC2016
		output=$(juju ssh "${ssh_options[@]}" --pty=false ssh-both/0 \
			'test -z "$BASH_VERSION" && command -v bash && printf SH_COMMAND_OK')
		check_contains "${output}" SH_COMMAND_OK
		if [[ ${route} == controller ]]; then
			# Controller-mediated CLI sessions have always requested Bash login,
			# even without a PTY. Verify shell choice and actual profile loading.
			# shellcheck disable=SC2016
			output=$(printf '%s\n' 'test -n "$BASH_VERSION" && shopt -q login_shell && test "$JUJU_SSH_PROFILE" = loaded && printf LOGIN_PROFILE_OK' |
				juju ssh "${ssh_options[@]}" --pty=false ssh-both/0)
			check_contains "${output}" LOGIN_PROFILE_OK
		else
			# Direct sessions without a terminal retain their historical sh.
			# shellcheck disable=SC2016
			output=$(printf '%s\n' 'test -z "$BASH_VERSION" && test -z "$JUJU_SSH_PROFILE" && printf SH_DEFAULT_OK' |
				juju ssh "${ssh_options[@]}" --pty=false ssh-both/0)
			check_contains "${output}" SH_DEFAULT_OK
		fi
		# An explicit request for Bash remains an ordinary user command.
		# shellcheck disable=SC2016
		output=$(printf '%s\n' 'test "$JUJU_SSH_PROFILE" = loaded && printf EXPLICIT_LOGIN_OK' |
			juju ssh "${ssh_options[@]}" --pty=false ssh-both/0 'exec bash --login')
		check_contains "${output}" EXPLICIT_LOGIN_OK

		# The helper supplies a real local terminal and checks remote stdin
		# is a terminal too. CLI login sessions fall back to sh on ssh-sh.
		python3 "${repo_root}/tests/suites/ssh/ssh_pty.py" --resize -- \
			juju ssh "${ssh_options[@]}" --pty=true ssh-bash/0
		python3 "${repo_root}/tests/suites/ssh/ssh_pty.py" -- \
			juju ssh "${ssh_options[@]}" --pty=true ssh-sh/0
		python3 "${repo_root}/tests/suites/ssh/ssh_pty.py" --shell bash-login -- \
			juju ssh "${ssh_options[@]}" --pty=true ssh-both/0

		check_ssh_no_shell juju ssh "${ssh_options[@]}" --pty=false ssh-noshell/0 echo SSH_OK
		check_ssh_no_shell juju ssh "${ssh_options[@]}" --pty=false ssh-noshell/0
		python3 "${repo_root}/tests/suites/ssh/ssh_pty.py" --no-shell -- \
			juju ssh "${ssh_options[@]}" --pty=true ssh-noshell/0
	done

	# Raw OpenSSH shell requests keep sh preference. Only the final-hop user
	# differs from the generated CLI command; authentication at the outer hop
	# uses a normal model key. Disable prompts for these disposable test pods.
	# add_model already registers this client key; keep it out of test artifacts.
	key_file="${JUJU_DATA:-${HOME}/.local/share/juju}/ssh/juju_id_ed25519"
	for pty in false true; do
		raw_command=$(juju ssh --show-command --ssh-key "${key_file}" \
			--container workload --pty="${pty}" ssh-both/0)
		check_contains "${raw_command}" 'juju-login@'
		raw_command=${raw_command/juju-login@/ubuntu@}
		raw_command=${raw_command//ssh /ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=\/dev\/null }
		if [[ ${pty} == true ]]; then
			python3 "${repo_root}/tests/suites/ssh/ssh_pty.py" --shell sh -- \
				bash -c "${raw_command}"
		else
			# shellcheck disable=SC2016
			output=$(printf '%s\n' 'test -z "$BASH_VERSION" && test -z "$JUJU_SSH_PROFILE" && printf SH_DEFAULT_OK' |
				bash -c "${raw_command}")
			check_contains "${output}" SH_DEFAULT_OK
		fi
	done

	destroy_model "${model_name}"
}

# check_ssh_no_shell requires a non-zero status as well as the diagnostic.
check_ssh_no_shell() {
	local output
	if output=$("$@" </dev/null 2>&1); then
		echo "SSH unexpectedly succeeded in a container without shells" >&2
		return 1
	fi
	check_contains "${output}" "couldn't find any shell"
	check_not_contains "${output}" "executable file not found"
	check_not_contains "${output}" "no such file or directory"
}
