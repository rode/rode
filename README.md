# rode
> \rōd\ - a line (as of rope or chain) used to attach an anchor to a boat

**rode** provides the collection, attestation and enforcement of policies in your software supply chain.

## Why rode?
Enterprises require a secure and reliable software delivery lifecycle to meet the needs of audit and compliance. This has traditionaly been implemented by applying governance and additional process. **rode** aims to meet this need by enabling **Automated Governance**. Automated Governance allows us to move the existing change approval process left by automating stages in software delivery that may currently exist as manual activities. This is possible by building a codified system of trust and authority for the entire software lifecycle. **rode** facilitates the collection and organization of important software supply chain metadata and provides a method of Automated Governance via **Policy as Code**.

## rode Architecture
The overall architecture of **rode** is built around bringing together tools built with the needs of governance in mind. The system of **rode** consists of **Collectors**, the **rode** API, **Grafeas**, and **Open Policy Agent**. We have extended the Grafeas storage backend to use **Elasticsearch**. These tools work together to enable Automated Governance.

![Rode Architecture](docs/img/rode-ag-architecture.svg)
### [Collectors](./docs/collectors.md)
[Collectors](./docs/collectors.md) package the metadata in the form of an "occurrence". These occurrences represent verifiable, individual software delivery process events. Collectors provide an entrypoint to the **rode** system by helping standardize the way metadata is brought in. They will be "purpose built" to collect metadata from any of the tools you are using in your software delivery toolchain.

### Grafeas

[Grafeas](https://github.com/grafeas/grafeas) is an open source project that provides an API and storage layer for artifact metadata.

Information that is gathered by collectors is ultimately stored within Grafeas as [Occurrences](https://github.com/grafeas/grafeas/blob/master/docs/grafeas_concepts.md#occurrences).
Occurrences represent a particular piece of metadata about an artifact, and they can be used by Rode when evaluating policies
against artifacts. Information such as an artifact's vulnerabilities, how it was built, and the quality of the codebase
that produced the artifact can be fed to a policy in order to determine whether that artifact meets a certain set of
standards set by your organization.

The primary way that Grafeas adds value to the Rode project is through its models around how artifact metadata should be
tracked and stored. Over time, we plan to add new types of Occurrences to Grafeas to represent artifact metadata concepts
that we believe are important, but aren't currently represented in the existing models.

We currently use a [custom backend](https://github.com/rode/grafeas-elasticsearch) for Grafeas that's based on Elasticsearch.

### Open Policy Agent
#### Policy Evaluation
..

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
