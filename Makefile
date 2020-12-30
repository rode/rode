.PHONY: generate tools test fmtcheck vet fmt mocks
GOFMT_FILES?=$$(find . -name '*.go' | grep -v proto)

GO111MODULE=on

tools:
	go generate ./tools

generate:
	docker build ./scripts/generate -t ghcr.io/rode/rode-generate:latest
	docker run -it --rm -v $$(pwd):/rode ghcr.io/rode/rode-generate:latest

fmtcheck:
	lineCount=$(shell gofmt -l -s $(GOFMT_FILES) | wc -l | tr -d ' ') && exit $$lineCount

fmt:
	gofmt -w -s $(GOFMT_FILES)

vet:
	go vet ./...

test: fmtcheck vet
	go test ./... -coverprofile=coverage.txt -covermode atomic

mocks:
	mockgen -package mocks github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto GrafeasV1Beta1Client > mocks/grafeasV1Beta1Client.go
	mockgen -package mocks github.com/rode/rode/protodeps/grafeas/proto/v1beta1/project_go_proto ProjectsClient > mocks/grafeasProjectsClient.go
