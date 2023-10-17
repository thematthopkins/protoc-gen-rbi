#!/bin/bash

set -euo pipefail
set -x

readonly ROOT="$(git rev-parse --show-toplevel)"
readonly TEST_PLUGIN="${ROOT}/protoc-gen-rbi-out"

cd "${ROOT}"
GO111MODULE=on go build -o "${TEST_PLUGIN}" ./

protoc \
    --rbi_out=../wallet/ --ruby_out=../wallet/lib \
    --rbi_opt="grpc=false" \
    --plugin=protoc-gen-rbi="${TEST_PLUGIN}" \
    --proto_path=../wallet/lib/frontend ../wallet/lib/frontend/frontend.proto \
    --experimental_allow_proto3_optional

