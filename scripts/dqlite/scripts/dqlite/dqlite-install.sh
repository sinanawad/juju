#!/bin/bash

set -e

source "$(dirname $0)/../env.sh"

check_dependencies sha256sum

sha() {
	case ${BUILD_ARCH} in
		amd64) echo "28d35a2c524f96df872ef2bb70e537170fac43985e6d260439c8cbcc80aa728b" ;;
		arm64) echo "d68a758d0d4c6f514d35ea6190f14f3e41b6f67564e1f579ccb0a63f13e1a220" ;;
		s390x) echo "8561238d7cdc2036fee321b7f8f1b563500325b4b1ed172002a56aca79ddb936" ;;
		ppc64le) echo "950f55a4aa10a7209ede86cf4c023ec1dc79a31317f9e6d5378d7897beb26b35" ;;
		*) { echo "Unsupported arch ${BUILD_ARCH}."; exit 1; } ;;
	esac
}

FILE="$(mktemp -d)/latest-dqlite-deps-${BUILD_ARCH}.tar.bz2"

retrieve() {
	local filenames sha

	sha=${1}

	filenames=( "${sha}.tar.bz2" )
	for name in "${filenames[@]}"; do
		echo "Retrieving ${name}"
		curl --fail -o ${FILE} -s https://dqlite-static-libs.s3.amazonaws.com/${name} && return || {
			echo " + Failed to retrieve ${name}";
			rm -f ${FILE} || true;
			true;
		}
	done
}

install() {
	mkdir -p ${EXTRACTED_DEPS_PATH}
	SHA=$(sha)
	retrieve ${SHA}
	if [ ! -f ${FILE} ]; then
		echo "Failed to retrieve dqlite static libs"
		exit 1
	fi

    SUM=$(sha256sum ${FILE} | awk '{print $1}')
    if [ "${SUM}" != ${SHA} ]; then
        echo "sha256sum mismatch (${SUM}, expected $(sha))"
        exit 1
    fi

    echo "${EXTRACTED_DEPS_PATH} ${FILE}"

    tar xjf ${FILE} -C ${EXTRACTED_DEPS_PATH} || { echo "Failed to extract ${FILE}"; exit 1; }
    mv ${EXTRACTED_DEPS_PATH}/juju-dqlite-static-lib-deps ${EXTRACTED_DEPS_ARCH_PATH} || { echo "Failed to move ${EXTRACTED_DEPS_PATH}/juju-dqlite-static-lib-deps to ${EXTRACTED_DEPS_ARCH_PATH}"; exit 1; }
}
