# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [proto/v1alpha1/rode.proto](#proto/v1alpha1/rode.proto)
    - [BatchCreateOccurrencesRequest](#rode.v1alpha1.BatchCreateOccurrencesRequest)
    - [BatchCreateOccurrencesResponse](#rode.v1alpha1.BatchCreateOccurrencesResponse)
    - [CreateNoteRequest](#rode.v1alpha1.CreateNoteRequest)
    - [GenericResource](#rode.v1alpha1.GenericResource)
    - [GenericResourceVersion](#rode.v1alpha1.GenericResourceVersion)
    - [ListGenericResourceVersionsRequest](#rode.v1alpha1.ListGenericResourceVersionsRequest)
    - [ListGenericResourceVersionsResponse](#rode.v1alpha1.ListGenericResourceVersionsResponse)
    - [ListGenericResourcesRequest](#rode.v1alpha1.ListGenericResourcesRequest)
    - [ListGenericResourcesResponse](#rode.v1alpha1.ListGenericResourcesResponse)
    - [ListOccurrencesRequest](#rode.v1alpha1.ListOccurrencesRequest)
    - [ListOccurrencesResponse](#rode.v1alpha1.ListOccurrencesResponse)
    - [ListResourcesRequest](#rode.v1alpha1.ListResourcesRequest)
    - [ListResourcesResponse](#rode.v1alpha1.ListResourcesResponse)
    - [ListVersionedResourceOccurrencesRequest](#rode.v1alpha1.ListVersionedResourceOccurrencesRequest)
    - [ListVersionedResourceOccurrencesResponse](#rode.v1alpha1.ListVersionedResourceOccurrencesResponse)
    - [ListVersionedResourceOccurrencesResponse.RelatedNotesEntry](#rode.v1alpha1.ListVersionedResourceOccurrencesResponse.RelatedNotesEntry)
    - [RegisterCollectorRequest](#rode.v1alpha1.RegisterCollectorRequest)
    - [RegisterCollectorResponse](#rode.v1alpha1.RegisterCollectorResponse)
    - [RegisterCollectorResponse.NotesEntry](#rode.v1alpha1.RegisterCollectorResponse.NotesEntry)
    - [UpdateOccurrenceRequest](#rode.v1alpha1.UpdateOccurrenceRequest)
  
    - [ResourceType](#rode.v1alpha1.ResourceType)
  
    - [Rode](#rode.v1alpha1.Rode)
  
- [proto/v1alpha1/rode_policy.proto](#proto/v1alpha1/rode_policy.proto)
    - [DeletePolicyRequest](#rode.v1alpha1.DeletePolicyRequest)
    - [EvaluatePolicyRequest](#rode.v1alpha1.EvaluatePolicyRequest)
    - [EvaluatePolicyResponse](#rode.v1alpha1.EvaluatePolicyResponse)
    - [EvaluatePolicyResult](#rode.v1alpha1.EvaluatePolicyResult)
    - [EvaluatePolicyViolation](#rode.v1alpha1.EvaluatePolicyViolation)
    - [GetPolicyRequest](#rode.v1alpha1.GetPolicyRequest)
    - [ListPoliciesRequest](#rode.v1alpha1.ListPoliciesRequest)
    - [ListPoliciesResponse](#rode.v1alpha1.ListPoliciesResponse)
    - [Policy](#rode.v1alpha1.Policy)
    - [PolicyEntity](#rode.v1alpha1.PolicyEntity)
    - [UpdatePolicyRequest](#rode.v1alpha1.UpdatePolicyRequest)
    - [ValidatePolicyRequest](#rode.v1alpha1.ValidatePolicyRequest)
    - [ValidatePolicyResponse](#rode.v1alpha1.ValidatePolicyResponse)
  
- [Scalar Value Types](#scalar-value-types)



<a name="proto/v1alpha1/rode.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## proto/v1alpha1/rode.proto



<a name="rode.v1alpha1.BatchCreateOccurrencesRequest"></a>

### BatchCreateOccurrencesRequest
Request to create occurrences in batch.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| occurrences | [grafeas.v1beta1.Occurrence](#grafeas.v1beta1.Occurrence) | repeated | The occurrences to create. |






<a name="rode.v1alpha1.BatchCreateOccurrencesResponse"></a>

### BatchCreateOccurrencesResponse
Response for creating occurrences in batch.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| occurrences | [grafeas.v1beta1.Occurrence](#grafeas.v1beta1.Occurrence) | repeated | The occurrences that were created. |






<a name="rode.v1alpha1.CreateNoteRequest"></a>

### CreateNoteRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| note_id | [string](#string) |  |  |
| note | [grafeas.v1beta1.Note](#grafeas.v1beta1.Note) |  |  |






<a name="rode.v1alpha1.GenericResource"></a>

### GenericResource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Id represents the unique id of the generic resource. This is usually the resource prefix plus the name, except in the case of Docker images. The id is used as a parameter for the ListGenericResourceVersions RPC. |
| name | [string](#string) |  | Name represents the name of this generic resource as seen on the UI. |
| type | [ResourceType](#rode.v1alpha1.ResourceType) |  | Type represents the resource type for this generic resource, such as &#34;DOCKER&#34; or &#34;GIT&#34; |
| created | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  |  |






<a name="rode.v1alpha1.GenericResourceVersion"></a>

### GenericResourceVersion



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| version | [string](#string) |  | Version represents the unique artifact version as a fully qualified URI. Example: a Docker image version might look like this: harbor.liatr.io/rode-demo/node-app@sha256:a235554754f9bf075ac1c1b70c224ef5997176b776f0c56e340aeb63f429ace8 |
| names | [string](#string) | repeated | Names represents related artifact names, if they exist. This information will be sourced from build occurrences. Example: a Docker image name might look like this: harbor.liatr.io/rode-demo/node-app:latest |
| created | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  |  |






<a name="rode.v1alpha1.ListGenericResourceVersionsRequest"></a>

### ListGenericResourceVersionsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| filter | [string](#string) |  |  |
| page_size | [int32](#int32) |  |  |
| page_token | [string](#string) |  |  |






<a name="rode.v1alpha1.ListGenericResourceVersionsResponse"></a>

### ListGenericResourceVersionsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| versions | [GenericResourceVersion](#rode.v1alpha1.GenericResourceVersion) | repeated |  |
| next_page_token | [string](#string) |  |  |






<a name="rode.v1alpha1.ListGenericResourcesRequest"></a>

### ListGenericResourcesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| filter | [string](#string) |  |  |
| page_size | [int32](#int32) |  |  |
| page_token | [string](#string) |  |  |






<a name="rode.v1alpha1.ListGenericResourcesResponse"></a>

### ListGenericResourcesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| generic_resources | [GenericResource](#rode.v1alpha1.GenericResource) | repeated |  |
| next_page_token | [string](#string) |  |  |






<a name="rode.v1alpha1.ListOccurrencesRequest"></a>

### ListOccurrencesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| filter | [string](#string) |  |  |
| page_size | [int32](#int32) |  |  |
| page_token | [string](#string) |  |  |






<a name="rode.v1alpha1.ListOccurrencesResponse"></a>

### ListOccurrencesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| occurrences | [grafeas.v1beta1.Occurrence](#grafeas.v1beta1.Occurrence) | repeated |  |
| next_page_token | [string](#string) |  |  |






<a name="rode.v1alpha1.ListResourcesRequest"></a>

### ListResourcesRequest
modeled after Grafeas&#39; ListOccurrence request/response
https://github.com/grafeas/grafeas/blob/5b072a9930eace404066502b49a72e5b420d3576/proto/v1beta1/grafeas.proto#L345-L374


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| filter | [string](#string) |  |  |
| page_size | [int32](#int32) |  |  |
| page_token | [string](#string) |  |  |






<a name="rode.v1alpha1.ListResourcesResponse"></a>

### ListResourcesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resources | [grafeas.v1beta1.Resource](#grafeas.v1beta1.Resource) | repeated |  |
| next_page_token | [string](#string) |  |  |






<a name="rode.v1alpha1.ListVersionedResourceOccurrencesRequest"></a>

### ListVersionedResourceOccurrencesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource_uri | [string](#string) |  |  |
| page_size | [int32](#int32) |  |  |
| page_token | [string](#string) |  |  |
| fetch_related_notes | [bool](#bool) |  | FetchRelatedNotes represents whether or not the notes attached to each occurrence should also be returned in the response. |






<a name="rode.v1alpha1.ListVersionedResourceOccurrencesResponse"></a>

### ListVersionedResourceOccurrencesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| occurrences | [grafeas.v1beta1.Occurrence](#grafeas.v1beta1.Occurrence) | repeated |  |
| next_page_token | [string](#string) |  |  |
| related_notes | [ListVersionedResourceOccurrencesResponse.RelatedNotesEntry](#rode.v1alpha1.ListVersionedResourceOccurrencesResponse.RelatedNotesEntry) | repeated | RelatedNotes are returned when FetchRelatedNotes on the request is set to true. |






<a name="rode.v1alpha1.ListVersionedResourceOccurrencesResponse.RelatedNotesEntry"></a>

### ListVersionedResourceOccurrencesResponse.RelatedNotesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [grafeas.v1beta1.Note](#grafeas.v1beta1.Note) |  |  |






<a name="rode.v1alpha1.RegisterCollectorRequest"></a>

### RegisterCollectorRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| notes | [grafeas.v1beta1.Note](#grafeas.v1beta1.Note) | repeated |  |






<a name="rode.v1alpha1.RegisterCollectorResponse"></a>

### RegisterCollectorResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| notes | [RegisterCollectorResponse.NotesEntry](#rode.v1alpha1.RegisterCollectorResponse.NotesEntry) | repeated |  |






<a name="rode.v1alpha1.RegisterCollectorResponse.NotesEntry"></a>

### RegisterCollectorResponse.NotesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [grafeas.v1beta1.Note](#grafeas.v1beta1.Note) |  |  |






<a name="rode.v1alpha1.UpdateOccurrenceRequest"></a>

### UpdateOccurrenceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| occurrence | [grafeas.v1beta1.Occurrence](#grafeas.v1beta1.Occurrence) |  |  |
| update_mask | [google.protobuf.FieldMask](#google.protobuf.FieldMask) |  |  |





 


<a name="rode.v1alpha1.ResourceType"></a>

### ResourceType


| Name | Number | Description |
| ---- | ------ | ----------- |
| RESOURCE_TYPE_UNSPECIFIED | 0 |  |
| DOCKER | 1 |  |
| GIT | 2 |  |
| MAVEN | 3 |  |
| FILE | 4 |  |
| NPM | 5 |  |
| NUGET | 6 |  |
| PIP | 7 |  |
| DEBIAN | 8 |  |
| RPM | 9 |  |


 

 


<a name="rode.v1alpha1.Rode"></a>

### Rode


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| BatchCreateOccurrences | [BatchCreateOccurrencesRequest](#rode.v1alpha1.BatchCreateOccurrencesRequest) | [BatchCreateOccurrencesResponse](#rode.v1alpha1.BatchCreateOccurrencesResponse) | Create occurrences |
| EvaluatePolicy | [EvaluatePolicyRequest](#rode.v1alpha1.EvaluatePolicyRequest) | [EvaluatePolicyResponse](#rode.v1alpha1.EvaluatePolicyResponse) | Verify that an artifact satisfies a policy |
| ListResources | [ListResourcesRequest](#rode.v1alpha1.ListResourcesRequest) | [ListResourcesResponse](#rode.v1alpha1.ListResourcesResponse) | List resource URI |
| ListGenericResources | [ListGenericResourcesRequest](#rode.v1alpha1.ListGenericResourcesRequest) | [ListGenericResourcesResponse](#rode.v1alpha1.ListGenericResourcesResponse) |  |
| ListGenericResourceVersions | [ListGenericResourceVersionsRequest](#rode.v1alpha1.ListGenericResourceVersionsRequest) | [ListGenericResourceVersionsResponse](#rode.v1alpha1.ListGenericResourceVersionsResponse) | ListGenericResourceVersions can be used to list all known versions of a generic resource. Versions will always include the unique identifier (in the case of Docker images, the sha256) and will optionally include any related names (in the case of Docker images, any associated tags for the image). |
| ListVersionedResourceOccurrences | [ListVersionedResourceOccurrencesRequest](#rode.v1alpha1.ListVersionedResourceOccurrencesRequest) | [ListVersionedResourceOccurrencesResponse](#rode.v1alpha1.ListVersionedResourceOccurrencesResponse) |  |
| ListOccurrences | [ListOccurrencesRequest](#rode.v1alpha1.ListOccurrencesRequest) | [ListOccurrencesResponse](#rode.v1alpha1.ListOccurrencesResponse) |  |
| UpdateOccurrence | [UpdateOccurrenceRequest](#rode.v1alpha1.UpdateOccurrenceRequest) | [.grafeas.v1beta1.Occurrence](#grafeas.v1beta1.Occurrence) |  |
| CreatePolicy | [Policy](#rode.v1alpha1.Policy) | [Policy](#rode.v1alpha1.Policy) |  |
| GetPolicy | [GetPolicyRequest](#rode.v1alpha1.GetPolicyRequest) | [Policy](#rode.v1alpha1.Policy) |  |
| DeletePolicy | [DeletePolicyRequest](#rode.v1alpha1.DeletePolicyRequest) | [.google.protobuf.Empty](#google.protobuf.Empty) |  |
| ListPolicies | [ListPoliciesRequest](#rode.v1alpha1.ListPoliciesRequest) | [ListPoliciesResponse](#rode.v1alpha1.ListPoliciesResponse) |  |
| ValidatePolicy | [ValidatePolicyRequest](#rode.v1alpha1.ValidatePolicyRequest) | [ValidatePolicyResponse](#rode.v1alpha1.ValidatePolicyResponse) |  |
| UpdatePolicy | [UpdatePolicyRequest](#rode.v1alpha1.UpdatePolicyRequest) | [Policy](#rode.v1alpha1.Policy) |  |
| RegisterCollector | [RegisterCollectorRequest](#rode.v1alpha1.RegisterCollectorRequest) | [RegisterCollectorResponse](#rode.v1alpha1.RegisterCollectorResponse) | RegisterCollector accepts a collector ID and a list of notes that this collector will reference when creating occurrences. The response will contain the notes with the fully qualified note name. This operation is idempotent, so any notes that already exist will not be re-created. Collectors are expected to invoke this RPC each time they start. |
| CreateNote | [CreateNoteRequest](#rode.v1alpha1.CreateNoteRequest) | [.grafeas.v1beta1.Note](#grafeas.v1beta1.Note) | CreateNote acts as a simple proxy to the grafeas CreateNote rpc |

 



<a name="proto/v1alpha1/rode_policy.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## proto/v1alpha1/rode_policy.proto



<a name="rode.v1alpha1.DeletePolicyRequest"></a>

### DeletePolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Id is the autogenerated id of the policy. |






<a name="rode.v1alpha1.EvaluatePolicyRequest"></a>

### EvaluatePolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy | [string](#string) |  | Policy is the unique identifier of a policy. |
| resource_uri | [string](#string) |  | ResourceUri is used to identify occurrences that should be passed to the policy evaluation in Open Policy Agent. |






<a name="rode.v1alpha1.EvaluatePolicyResponse"></a>

### EvaluatePolicyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pass | [bool](#bool) |  | Pass indicates whether the entire evaluation succeeded or failed. |
| changed | [bool](#bool) |  | Changed designates if the evaluation result differs from the last evaluation. |
| result | [EvaluatePolicyResult](#rode.v1alpha1.EvaluatePolicyResult) | repeated | Result is a list of evaluation outputs built up by the policy. |
| explanation | [string](#string) | repeated | Explanation is the raw diagnostic output from Open Policy Agent when the explain parameter is included in the request. It&#39;s intended to be used when writing or debugging policy. |






<a name="rode.v1alpha1.EvaluatePolicyResult"></a>

### EvaluatePolicyResult



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pass | [bool](#bool) |  | Pass designates if this individual result succeeded or failed. |
| created | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  | Created is a timestamp set after the call to Open Policy Agent. |
| violations | [EvaluatePolicyViolation](#rode.v1alpha1.EvaluatePolicyViolation) | repeated | Violations is a set of rule results. Even if a rule succeeded, its output will be included in Violations. |






<a name="rode.v1alpha1.EvaluatePolicyViolation"></a>

### EvaluatePolicyViolation



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Id is a Rode-specific requirement that can be used to match up a rule output with the result. |
| name | [string](#string) |  | Name is a human-friendly description of the rule. |
| description | [string](#string) |  | Description is a longer message that explains the intention of the rule. |
| message | [string](#string) |  | Message is a computed result with more information about why the rule was violated (e.g., number of high severity vulnerabilities discovered). |
| link | [string](#string) |  |  |
| pass | [bool](#bool) |  | Pass indicates whether this rule succeeded or failed. |






<a name="rode.v1alpha1.GetPolicyRequest"></a>

### GetPolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Id is the autogenerated id of the policy. |






<a name="rode.v1alpha1.ListPoliciesRequest"></a>

### ListPoliciesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| filter | [string](#string) |  | Filter is a CEL (common expression language) filter that can be used to limit results. If a filter isn&#39;t specified, all policies are returned. |
| page_size | [int32](#int32) |  | PageSize controls the number of results. |
| page_token | [string](#string) |  | PageToken can be used to retrieve a specific page of results. The ListPoliciesResponse will include the next page token. |






<a name="rode.v1alpha1.ListPoliciesResponse"></a>

### ListPoliciesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policies | [Policy](#rode.v1alpha1.Policy) | repeated | Policies is the list of policies, with the number of results controlled by the ListPoliciesRequest.Filter and ListPoliciesRequest.PageSize. |
| next_page_token | [string](#string) |  | NextPageToken can be used to retrieve the next page of results. It will be empty if the caller has reached the end of the result set. |






<a name="rode.v1alpha1.Policy"></a>

### Policy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Id is the unique autogenerated identifier of a policy. |
| name | [string](#string) |  | Name of the policy |
| description | [string](#string) |  | Description should be a brief message about the intention of the policy. Updates to a policy can be described in the PolicyEntity.Message field. |
| current_version | [int32](#int32) |  | CurrentVersion is the default policy version that&#39;s used when a policy is retrieved or evaluated. It&#39;s not necessarily the latest, as it may be overwritten if an older policy version should be used instead. |
| policy | [PolicyEntity](#rode.v1alpha1.PolicyEntity) |  | Policy contains the Rego policy code or a source location. The PolicyEntity.Version matches CurrentVersion unless it was otherwise specified. |
| created | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  | Created is when the policy was first stored. |
| updated | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  | Updated indicates when either an edit occurred on the policy itself or a new policy version was created. |
| deleted | [bool](#bool) |  | Deleted is a flag controlling soft deletes. Deleted policies won&#39;t be returned by the ListPolicies RPC, but can still be retrieved and evaluated. |






<a name="rode.v1alpha1.PolicyEntity"></a>

### PolicyEntity



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| version | [int32](#int32) |  | Version is a number that represents revisions of a policy. Policy contents are immutable, so changes to the source are represented as new versions. |
| message | [string](#string) |  | Message should contain a brief summary of the changes to the policy code between the current version and the previous version. |
| rego_content | [string](#string) |  | RegoContent contains the Rego code for a given policy. Only one of RegoContent and SourcePath should be specified. |
| source_path | [string](#string) |  | SourcePath is the location of the policy stored in source control. |
| created | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  | Created represents when this policy version was stored. Policy contents are immutable, so there is no corresponding Updated field. |






<a name="rode.v1alpha1.UpdatePolicyRequest"></a>

### UpdatePolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Id is the autogenerated id of the policy. |
| policy | [Policy](#rode.v1alpha1.Policy) |  | Policy is the Policy message. Only Policy.Name, Policy.Description, and Policy.CurrentVersion can be updated. Changes to Policy.Policy are represented as new versions of a policy. |
| update_mask | [google.protobuf.FieldMask](#google.protobuf.FieldMask) |  | UpdateMask controls which fields should be updated with the values from Policy. |






<a name="rode.v1alpha1.ValidatePolicyRequest"></a>

### ValidatePolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy | [string](#string) |  | Policy is the raw Rego code to be validated. |






<a name="rode.v1alpha1.ValidatePolicyResponse"></a>

### ValidatePolicyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy | [string](#string) |  | Policy is the raw Rego code. |
| compile | [bool](#bool) |  | Compile is a flag that indicates whether compilation of the Rego code was successful. |
| errors | [string](#string) | repeated | Errors is a list of validation errors. |





 

 

 

 



## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |

