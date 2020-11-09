.PHONY: generate tools

GO111MODULE=on

tools:
	go generate ./tools

generate:
	docker build ./scripts/generate -t ghcr.io/liatrio/rode-api-generate:latest
	docker run -it --rm -v $$(pwd):/rode-api ghcr.io/liatrio/rode-api-generate:latest
