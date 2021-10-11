# Development

## Tools

- [Go 1.17](https://golang.org/dl/)
- Local Kubernetes cluster, ex: Docker for Desktop with Kubernetes enabled  

## Technology

Rode is built using the [Go](https://golang.org/) programming language, with an API that is exposed via [gRPC](https://grpc.io/).

Messages in gRPC typically use [Protocol Buffers](https://developers.google.com/protocol-buffers/docs/proto3), with Rode using
the `proto3` syntax. For clients that cannot use gRPC directly, Rode exposes a JSON HTTP API using 
[grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway). For defining and enforcing policy, Rode uses [Open Policy Agent](https://www.openpolicyagent.org/docs/latest/).

## Project Layout

- `auth`: authentication & authorization code.
- `common`: a Rode client wrapper that's used by the collectors and Terraform provider. 
- `config`: defines flags for configuring Rode.
- `hack`: small test programs for working with Rode.
- `mappings`: Elasticsearch index [mappings](https://www.elastic.co/guide/en/elasticsearch/reference/current/explicit-mapping.html).
- `mocks`: third party library mocks (mocks for Rode interfaces are nested under the package containing the interfaces)
- `opa`: an Open Policy Agent client
- `pkg`: contains structs that implement portions of the API and are embedded in the gRPC server.
- `proto`: Rode protocol buffer definitions and generated code.
- `protodeps`: Third party protocol buffer definitions that Rode consumes. 
- `server`: The Rode gRPC server implementation from the RPCs defined in `proto`.  
- `test`: Integration tests.

## Local Environment

There are several options for running Rode and its dependencies locally, and which one to use depends on the use case. The demo
is a more complete environment, but can be resource-intensive; while `rode-dev-env` may be easier to start with, it does 
not contain all of the services that may be needed to implement a feature. 

### Demo

The Rode [demo](https://github.com/rode/demo) contains Terraform for installing Rode, its dependencies, and several collectors. 
It's a more real-world example of using Rode and is useful when working on collectors or trying to integrate Rode into
a CI pipeline. See the [demo README](https://github.com/rode/demo#usage) for more information. 

### Rode Dev Env

Another option for local development is [`rode-dev-env`](https://github.com/rode/rode-dev-env), which contains a 
docker-compose file that can stand up Rode or just the services that Rode depends on. 

To start everything, use `docker compose up rode`. For working on Rode itself, use `docker compose up grafeas opa` to run
just those services. Then you can Rode locally with the following:

```shell
go run main.go \
  --debug \
  --grafeas-host=localhost:8080 \
  --elasticsearch-host=http://localhost:9200 \
  --opa-host=http://localhost:8181
```

Alternatively, you can set environment variables and then invoke `go run`:

```shell
GRAFEAS_HOST=localhost:8080
ELASTICSEARCH_HOST=http://localhost:9200
OPA_HOST=http://localhost:8181
```

Additionally, any of the [config](../config/config.go) flags can be set as environment variables.

To run Rode with authentication, set the `--oidc-issuer` to a OpenID Connect provider. Here's an example with Keycloak: 

```shell
go run main.go --debug \
  --grafeas-host=localhost:8080 \
  --elasticsearch-host=http://localhost:9200 \
  --opa-host=http://localhost:8181 \
  --oidc-issuer=https://keycloak.localhost/auth/realms/rode-demo \
  --oidc-required-audience=rode \
  --oidc-tls-insecure-skip-verify=true \
  --oidc-role-claim-path=resource_access.rode.roles
```

And an example using `ISSUER_URL=http://localhost:3000 docker compose up oidc-provider grafeas opa` in `rode-dev-env`:

```shell
go run main.go --debug \
  --grafeas-host=localhost:8080 \
  --elasticsearch-host=http://localhost:9200 \
  --opa-host=http://localhost:8181 \
  --oidc-issuer=http://localhost:3000 \
  --oidc-required-audience=rode \
  --oidc-tls-insecure-skip-verify=true \
  --oidc-role-claim-path=roles
```

## Testing

Rode has unit and integration test suites; both use the [`ginkgo`](https://github.com/onsi/ginkgo) testing framework 
with [`gomega`](https://github.com/onsi/gomega) for assertions.

### Unit Tests

Run the unit tests with `make test`, which will also check formatting and run `go vet`. To visualize code coverage,
use `make coverage`, which will open the coverage report in a browser. 

For creating mocks, Rode uses [`counterfeiter`](https://github.com/maxbrunsfeld/counterfeiter). 
Add the [directives to generate mocks](https://github.com/maxbrunsfeld/counterfeiter#step-2a---add-gogenerate-directives)
and then run `make mocks` to have counterfeiter generate fakes. 

### Integration Tests

Integration tests live in the `test/` folder at the project root. They can be run with `make integration`. 

The integration tests use Rode with authentication enabled, and they also check the level of access that different roles
have. The tests assume that an OpenID provider and certain clients exist, but this can be overwritten with environment 
variables:

| Environment Variable                  |
|---------------------------------------|
| `COLLECTOR_CLIENT_ID`                 |
| `COLLECTOR_CLIENT_SECRET`             |
| `ENFORCER_CLIENT_ID`                  |
| `ENFORCER_CLIENT_SECRET`              |
| `APPLICATION_DEVELOPER_CLIENT_ID`     |
| `APPLICATION_DEVELOPER_CLIENT_SECRET` |
| `POLICY_DEVELOPER_CLIENT_ID`          |
| `POLICY_DEVELOPER_CLIENT_SECRET`      |
| `POLICY_ADMINISTRATOR_CLIENT_ID`      |
| `POLICY_ADMINISTRATOR_CLIENT_SECRET`  |
| `ADMINISTRATOR_CLIENT_ID`             |
| `ADMINISTRATOR_CLIENT_SECRET`         |
| `TOKEN_URL`                           |
| `RODE_URL`                            |

The default values should work with `rode-dev-env`. 