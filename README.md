# rode

## Installation

### Helm

Add Helm repositories
```sh
helm repo add rode https://rode.github.io/charts
helm repo add elastic https://helm.elastic.co
helm repo update 
```

Install Rode
```sh
helm install rode rode/rode --set grafeas-elasticsearch.grafeas.elasticsearch.username=grafeas --set grafeas-elasticsearch.grafeas.elasticsearch.password=BAD_PASSWORD
```

See [Rode Helm chart](https://github.com/rode/charts/tree/main/charts/rode) for more details.

## Documentation

* [Development](docs/development.md)
* [API Reference](docs/api.md)
