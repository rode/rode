#!/usr/bin/env bash

# fetch grafeas protodeps

GOOGLE_API_PROTOS_DIR="/rode/protodeps/googleapis/google/api"
GOOGLE_API_PROTOS=("annotations" "client" "field_behavior" "http" "resource")
mkdir -p ${GOOGLE_API_PROTOS_DIR}
cd ${GOOGLE_API_PROTOS_DIR}
for api in ${GOOGLE_API_PROTOS[@]} ; do
    curl --silent -LO https://raw.githubusercontent.com/googleapis/googleapis/${GOOGLE_APIS_VERSION}/google/api/${api}.proto
done

GOOGLE_API_RPC_PROTOS_DIR="/rode/protodeps/googleapis/google/rpc"
GOOGLE_API_RPC_PROTOS=("status")
mkdir -p ${GOOGLE_API_RPC_PROTOS_DIR}
cd ${GOOGLE_API_RPC_PROTOS_DIR}
for api in ${GOOGLE_API_RPC_PROTOS[@]} ; do
    curl --silent -LO https://raw.githubusercontent.com/googleapis/googleapis/${GOOGLE_APIS_VERSION}/google/rpc/${api}.proto
done

GRAFEAS_PROTOS_DIR="/rode/protodeps/grafeas/proto/v1beta1"
GRAFEAS_PROTOS=("grafeas" "vulnerability" "build" "image" "package" "deployment" "attestation" "intoto" "common" "provenance" "source" "discovery" "cvss" "project")
mkdir -p ${GRAFEAS_PROTOS_DIR}
cd ${GRAFEAS_PROTOS_DIR}
for api in ${GRAFEAS_PROTOS[@]} ; do
    curl --silent -LO https://raw.githubusercontent.com/grafeas/grafeas/${GRAFEAS_VERSION}/proto/v1beta1/${api}.proto
done

# we want the go code generated from the grafeas protobufs to reference the `rode` package when importing
# so we need to rewrite the "go_package" option after fetching these protobuf definitions to point to the `rode` package

grep -rl 'github.com/grafeas/grafeas' ./*.proto | xargs sed -i 's+github.com/grafeas/grafeas+github.com/rode/rode/protodeps/grafeas+g'

# next, generate go code from the grafeas protobuf definitions

cd /rode/protodeps/grafeas

function generate {
    protoc -I . -I ../googleapis --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/v1beta1/$1.proto
    rm -rf "proto/v1beta1/${1}_go_proto"
    mkdir "proto/v1beta1/${1}_go_proto"
    mv "proto/v1beta1/${1}.pb.go" "proto/v1beta1/${1}_go_proto"
    if test -f "proto/v1beta1/${1}_grpc.pb.go"; then
        mv "proto/v1beta1/${1}_grpc.pb.go" "proto/v1beta1/${1}_go_proto"
    fi
}

for api in ${GRAFEAS_PROTOS[@]} ; do
    generate ${api}
done

# finally, compile rode protobufs

cd /rode
protoc -I . \
  -I ./protodeps/grafeas \
  -I ./protodeps/googleapis \
  --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  --doc_out=docs --doc_opt=markdown,grpc.md \
  --grpc-gateway_out=. --grpc-gateway_opt paths=source_relative \
  ./proto/v1alpha1/*.proto
