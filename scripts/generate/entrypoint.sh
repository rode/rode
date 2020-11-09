#!/usr/bin/env bash

# fetch grafeas protodeps

GOOGLE_API_PROTOS_DIR="/rode-api/protodeps/googleapis/google/api"
GOOGLE_API_PROTOS=("annotations" "client" "field_behavior" "http" "resource")
mkdir -p ${GOOGLE_API_PROTOS_DIR}
cd ${GOOGLE_API_PROTOS_DIR}
for api in ${GOOGLE_API_PROTOS[@]} ; do
    curl --silent -LO https://raw.githubusercontent.com/googleapis/googleapis/${GOOGLE_APIS_VERSION}/google/api/${api}.proto
done

GOOGLE_API_RPC_PROTOS_DIR="/rode-api/protodeps/googleapis/google/rpc"
GOOGLE_API_RPC_PROTOS=("status")
mkdir -p ${GOOGLE_API_RPC_PROTOS_DIR}
cd ${GOOGLE_API_RPC_PROTOS_DIR}
for api in ${GOOGLE_API_RPC_PROTOS[@]} ; do
    curl --silent -LO https://raw.githubusercontent.com/googleapis/googleapis/${GOOGLE_APIS_VERSION}/google/rpc/${api}.proto
done

GRAFEAS_PROTOS_DIR="/rode-api/protodeps/grafeas/proto/v1beta1"
GRAFEAS_PROTOS=("grafeas" "vulnerability" "build" "image" "package" "deployment" "attestation" "intoto" "common" "provenance" "source" "discovery" "cvss")
mkdir -p ${GRAFEAS_PROTOS_DIR}
cd ${GRAFEAS_PROTOS_DIR}
for api in ${GRAFEAS_PROTOS[@]} ; do
    curl --silent -LO https://raw.githubusercontent.com/grafeas/grafeas/${GRAFEAS_VERSION}/proto/v1beta1/${api}.proto
done

# we want the go code generated from the grafeas protobufs to reference the `rode-api` package when importing
# so we need to rewrite the "go_package" option after fetching these protobuf definitions to point to the `rode-api` package

grep -rl 'github.com/grafeas/grafeas' ./*.proto | xargs sed -i 's+github.com/grafeas/grafeas+github.com/liatrio/rode-api/protodeps/grafeas+g'

# next, generate go code from the grafeas protobuf definitions

function generate {
  protoc -I . -I ../.. -I ../../../googleapis --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative $1.proto
  rm -rf "${1}_go_proto"
  mkdir "${1}_go_proto"
  mv ${1}.pb.go ${1}_go_proto
  if test -f "${1}_grpc.pb.go"; then
    mv ${1}_grpc.pb.go ${1}_go_proto
  fi
}

for api in ${GRAFEAS_PROTOS[@]} ; do
  generate ${api}
done

# finally, compile rode protobufs

cd /rode-api
protoc -I . -I ./protodeps/grafeas -I ./protodeps/googleapis --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative ./proto/v1alpha1/rode.proto
