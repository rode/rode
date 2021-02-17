FROM golang:1.15.4-alpine3.12

ENV GO111MODULE="on"
ENV PROTOC_VERSION="3.12.2-r0"
ENV PROTOC_GEN_GO_VERSION="v1.25.0"
ENV PROTOC_GEN_GO_GRPC_VERSION="v1.0.1"
ENV PROTOC_GEN_DOC_VERSION="v1.3.2"
ENV PROTOC_GEN_GRPC_GATEWAY_VERSION="v2.2.0"
ENV GRAFEAS_VERSION="v0.1.6"
ENV GOOGLE_APIS_VERSION="fb6fa4cfb16917da8dc5d23c2494d422dd3e9cd4"

RUN apk update && apk add \
    protoc=${PROTOC_VERSION} \
    protobuf-dev=${PROTOC_VERSION} \
    curl \
    bash
RUN go get google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_VERSION}
RUN go get google.golang.org/grpc/cmd/protoc-gen-go-grpc@${PROTOC_GEN_GO_GRPC_VERSION}
RUN go get github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc@${PROTOC_GEN_DOC_VERSION}
RUN go get github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@${PROTOC_GEN_GRPC_GATEWAY_VERSION}

WORKDIR /rode

ENTRYPOINT ["/rode/scripts/generate/entrypoint.sh"]
