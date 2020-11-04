// This file downloads dependencies that are required for working with protobufs

//go:generate env GO111MODULE=on go get google.golang.org/protobuf/cmd/protoc-gen-go@v1.25.0
//go:generate env GO111MODULE=on go get google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.0.1

package tools
