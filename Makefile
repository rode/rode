.PHONY: generate tools test fmtcheck vet fmt mocks coverage
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

coverage: test
	go tool cover -html=coverage.txt

mocks:
	go install github.com/maxbrunsfeld/counterfeiter/v6@v6.4.1
	COUNTERFEITER_NO_GENERATE_WARNING="true" go generate ./...
