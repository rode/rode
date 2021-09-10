FROM golang:1.17.0-alpine3.14

ENV PROTOC_VERSION="3.15.7-r1"
ENV PROTOC_GEN_GO_VERSION="v1.27.1"
ENV PROTOC_GEN_GO_GRPC_VERSION="v1.1.0"
ENV PROTOC_GEN_DOC_VERSION="v1.5.0"
ENV PROTOC_GEN_GRPC_GATEWAY_VERSION="v2.6.0"
ENV GRAFEAS_VERSION="v0.1.6"
ENV GOOGLE_APIS_VERSION="fb6fa4cfb16917da8dc5d23c2494d422dd3e9cd4"
ENV COUNTERFEITER_VERSION="v6@v6.4.1"

RUN apk update && apk add \
    protoc=${PROTOC_VERSION} \
    protobuf-dev=${PROTOC_VERSION} \
    curl \
    bash \
    build-base \
    git
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_VERSION}
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@${PROTOC_GEN_GO_GRPC_VERSION}
RUN go install github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc@${PROTOC_GEN_DOC_VERSION}
RUN go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@${PROTOC_GEN_GRPC_GATEWAY_VERSION}
RUN go install github.com/maxbrunsfeld/counterfeiter/${COUNTERFEITER_VERSION}

WORKDIR /rode

ENTRYPOINT ["/rode/scripts/generate/entrypoint.sh"]
