#!/bin/bash

set -euo pipefail
set -x

readonly ROOT="$(git rev-parse --show-toplevel)"
readonly TEST_PLUGIN="${ROOT}/protoc-gen-rbi-out"

echo "if you need the latest version of elm-protobuf, run:"
echo "go get -u github.com/thematthopkins/elm-protobuf@forward"

cd "${ROOT}"
GO111MODULE=on go build -o "${TEST_PLUGIN}" ./

protoc \
    --rbi_out=./spec/proto \
    --rbi_opt="grpc=false" \
    --plugin=protoc-gen-rbi="${TEST_PLUGIN}" \
    --proto_path=spec/proto/ ./spec/proto/simple.proto \
    --experimental_allow_proto3_optional

protoc \
    --rbi_out=./testdata \
    --rbi_opt="grpc=false" \
    --plugin=protoc-gen-rbi="${TEST_PLUGIN}" \
    --proto_path=testdata ./testdata/validate.proto \
    --experimental_allow_proto3_optional


if [[ ! -f "../wallet/lib/frontend/frontend.proto" ]]; then
    echo skipping generating frontend.proto, because wallet directory not found
else
    protoc \
        --rbi_out=../wallet/ \
        --rbi_opt="grpc=false" \
        --plugin=protoc-gen-rbi="${TEST_PLUGIN}" \
        --proto_path=../wallet/lib/frontend ../wallet/lib/frontend/frontend.proto \
        --experimental_allow_proto3_optional
fi

bundle exec rspec
