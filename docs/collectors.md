# Collectors

In order to use Rode to drive Automated Governance, we need a strategy for _collecting_
metadata from events in our toolchain. In Rode, **collectors** are responsible for
recieving events from exte rnal systems and converting them into
[occurrences](https://github.com/grafeas/grafeas/blob/master/docs/grafeas_concepts.md#occurrences).
These occurrences can be used to track the history of events for a particular
artifact. These collectors are to be built as individual services and can be
maintained independently of the central Rode API. This enables developers to build
collector services with a language of their choosing([_provided it supports gRPC_](https://grpc.io/docs/languages/))
and add new functionality at their own pace.

## Development

As mentioned previously, collectors are independent services and can be built with
any language or framework that has support for gRPC. Collectors typically have two
primary responsibilities:

- Listening for incoming event metadata associated with artifacts _for a single service_
- Publishing structured data to the Rode API in the form of _occurrences_

Keeping the responsibilites limited to these tasks make these fairly trivial to
maintain and reduces the complexity of interacting with many services in a single
code base. However since these collectors are developed independently you can extend
them as needed. For example if you need to retrieve more data from the service or
notify other endpoints feel free to integrate this logic into your implementation.

## Collecting Metadata

In order to store event data for an artifact we first need to collect information
about what action was performed as well as the result. There are several approaches
you could take dependening on the tool or service you are working with.

### Pull-based Collectors

![](img/pull-based-collector.svg)

The ideal approach is to rely on an external platform as the source of truth for
metadata collection. Using SonarQube as an example, we could utilize webhooks to
publish static analysis scan results to our collector. With some additional configuration
we could pass auth tokens in our webhook events to verify we only recieve events
from the offical SonarQube instance, avoiding the potential for developers to "fake"
their SonarQube results.

#### Considerations

Relying on external services requires additional work to configure the platform
you'd like to integrate with such as creating webhooks and alerts. We'd also need
to integrate an auth scheme into our collector to prevent developers from
standing up their own instance to "fake" events. Make sure to consider the
following when developing new collectors:

| Service Configuration           | Webhooks & alerts need to be enabled as well as network access so the service can reach our collector to publish new events.                                                                                                                                                                                                        |
| :------------------------------ | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Authenticity of the Service** | **There will need to be safeguards in place to prevent users from circumventing Rode policy. Many popular services support passing JWT or other forms of credentials that can be used to prevent unrestricted access. In the case these are not available, other restrictions such as Network Policies may need to be considered.** |

### Push-based Collectors

![](img/push-based-collector.svg)

There may some cases where you need to directly perform an action against an
artifact within the pipeline. A external service may not be required or communication
with it is not possible. It then becomes the pipeline stage's responsibility to:

1. Perform the action against the artifact(_ex: scanning, linting, testing_)
2. Aggregate the results
3. Publish the resulting metadata directly to the corresponding collector

How your collector receives metadata is dependent on the data and implementation
of the service. A common approach is to create a REST endpoint that accepts JSON
payloads containing pipeline metadata, allowing the collector to focus on transforming
this data into occurrences. Publishing results to Rode is the same regardless of
your collector's implementation; utilize your language's gRPC library to format
and send occurrences to the central API.

#### Considerations

Push-based collectors requires more thought to security and the ability to validate
metadata sent to the collector. If the execution of the pipeline stage is not
protected or we lack the ability to verify the source of the results, we lose
confidence in the ability to **attest** that the metadata for an artifact is
credible. Some important factors to consider:

| Origin of Metadata       | Prevent malicious developers from bypassing policy enforcement by publishing fradulent results.                                                                                                                                                                                                                           |
| :----------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Validity of Metadata** | **Developers may attempt to publish results claiming to be associated with an artifact when in fact they are results from another application**                                                                                                                                                                           |
| **Valid Execution**      | **Depending on the tool/service in question, parameters or configuration can drastically change the results of an action(_consider running static analysis with flags that omit entire directories of code_). It is important to be able to attest to not only the results but also the way the results were generated.** |

## Communicating with Rode

Since we will be publishing occurrences to Rode via gRPC, we will need to make
sure our language of choice has the necessary stub code to perform these calls.
In order to use gRPC you will need to generate client code using the proto
definitions provided by the Rode project (learn more about gRPC and generating
stub code [here](https://grpc.io/docs/)). Using the tools provided by the gRPC
project we can use our generated code to create occurrence objects and send them
to the Rode API. Although gRPC supports many languages, each will require it's
own stub code in order to make it's own calls.

# Existing Collectors

Here are several example of collectors that currently work with Rode.

| Collector           | Link                                        |
| ------------------- | ------------------------------------------- |
| collector-coverity  | https://github.com/rode/collector-coverity  |
| collector-ecr       | https://github.com/rode/collector-ecr       |
| collector-clair     | https://github.com/rode/collector-clair     |
| collector-harbor    | https://github.com/rode/collector-harbor    |
| collector-sonarqube | https://github.com/rode/collector-sonarqube |
