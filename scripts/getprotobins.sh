#!/usr/bin/env bash

set -e

DEST_DIR=${PWD}/builds/proto
BASE_DOWNLOAD_URL=https://github.com/protocolbuffers/protobuf/releases/download

PROTOC_VERSION=3.15.8
PROTOC_GEN_GO_VERSION=1.28
PROTOC_GEN_GO_GRPC_VERSION=1.2
PROTOC_GEN_GRPC_GATEWAY_VERSION=2.11.0
PROTOC_GEN_OPENAPI_V2_VERSION=2.11.0

function refresh_dir {
    local dirpath="${1}"

    rm -rf ${dirpath}
    mkdir -p ${dirpath}
}

function download_bins {
    local platform="${1}"
    local dest_bin_dir="${DEST_DIR}/${platform}"

    echo -e "\nDownload proto bins for ${platform}"
    mkdir -p ${dest_bin_dir}

    curl -LO ${BASE_DOWNLOAD_URL}/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-${platform}-x86_64.zip
    unzip protoc-${PROTOC_VERSION}-${platform}-x86_64.zip -d "${dest_bin_dir}"
    rm -f protoc-${PROTOC_VERSION}-${platform}-x86_64.zip

    export GOBIN=${dest_bin_dir}
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v${PROTOC_GEN_GO_VERSION}
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v${PROTOC_GEN_GO_GRPC_VERSION}
    go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v${PROTOC_GEN_GRPC_GATEWAY_VERSION}
    go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v${PROTOC_GEN_OPENAPI_V2_VERSION}

    echo ""
}

function main {
    refresh_dir "${DEST_DIR}"

    download_bins "linux"
    download_bins "osx"

    echo "
NOTE:
  Keep these binaries and standard proto files in repository:
    - Do not download each time during docker build;
    - Do not download if a developer doesn't have it already;
    - Be aware of remote resources inaccessibility."

    echo -e "\nDONE"
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main
fi