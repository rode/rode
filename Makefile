.PHONY: generate

GO111MODULE=on

tools:
	go generate ./tools

generate:
	go generate ./protodeps/googleapis/google/api
	go generate ./protodeps/googleapis/google/rpc
	go generate ./protodeps/grafeas/proto/v1beta1
	protoc -I . -I ./protodeps/grafeas -I ./protodeps/googleapis --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative ./proto/v1alpha1/rode.proto
