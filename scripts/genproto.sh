#!/usr/bin/env bash

set -e

OUTPUT_DIR=./internal/adapters/primary/generated
DOCS_OUTPUT_DIR=./api/openapiv2
SRC_DIR=./internal/adapters/primary/proto
PROTOC_ROOT_DIR=${PWD}/builds/proto
PROTOC_LIBS=./third_party

function refresh_dir {
    local dirpath="${1}"

    rm -rf ${dirpath}
    mkdir -p ${dirpath}
}

function get_platform {
    if [[ "$OSTYPE" == "linux"* ]]; then
        echo "linux"
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        echo "osx"
    else
        echo ""
    fi
}

function gen_source {
    local platform="${1}"
    local src="${2}"

    local protoc_dir=${PROTOC_ROOT_DIR}/${platform}
    local protoc_bin=${protoc_dir}/bin/protoc
    local protoc_gen_validate=$(command -v protoc-gen-validate)

    if [[ -z "${protoc_gen_validate}" ]]; then
        echo "protoc-gen-validate not found in PATH" >&2
        exit 1
    fi

    "${protoc_bin}" \
        -I="${SRC_DIR}" "${src}"\
        -I="${PWD}/builds/proto/$(get_platform)/include" \
        -I="${PWD}/vendor/github.com/envoyproxy/protoc-gen-validate" \
        --proto_path="${PROTOC_LIBS}" \
        --plugin="protoc-gen-go=${protoc_dir}/protoc-gen-go" \
          --go_out=${OUTPUT_DIR} \
          --go_opt=paths=source_relative \
        --plugin="protoc-gen-go-grpc=${protoc_dir}/protoc-gen-go-grpc" \
          --go-grpc_out=${OUTPUT_DIR} \
          --go-grpc_opt=paths=source_relative\
        --plugin="protoc-gen-grpc-gateway=${protoc_dir}/protoc-gen-grpc-gateway" \
          --grpc-gateway_out=${OUTPUT_DIR} \
          --grpc-gateway_opt=paths=source_relative \
        --plugin="protoc-gen-validate=${protoc_gen_validate}" \
          --validate_out="lang=go,paths=source_relative:${OUTPUT_DIR}"
}

function main {
    refresh_dir "${OUTPUT_DIR}"
    refresh_dir "${DOCS_OUTPUT_DIR}"

    local platform=$(get_platform)
    if [[ -z "${platform}" ]]; then
        echo "Unknown platform" >&2
        exit 1
    fi

    local srcs=$(find ${SRC_DIR} -name '*.proto' -not -path '*/google/*')
    for src in ${srcs}; do
        echo "${src}"
        gen_source "${platform}" "${src}"
    done

    echo -e "\nDONE"
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main
fi
