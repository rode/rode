# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [proto/v1alpha1/rode-policy.proto](#proto/v1alpha1/rode-policy.proto)
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
  
- [proto/v1alpha1/rode.proto](#proto/v1alpha1/rode.proto)
    - [BatchCreateOccurrencesRequest](#rode.v1alpha1.BatchCreateOccurrencesRequest)
    - [BatchCreateOccurrencesResponse](#rode.v1alpha1.BatchCreateOccurrencesResponse)
    - [GenericResource](#rode.v1alpha1.GenericResource)
    - [ListGenericResourcesRequest](#rode.v1alpha1.ListGenericResourcesRequest)
    - [ListGenericResourcesResponse](#rode.v1alpha1.ListGenericResourcesResponse)
    - [ListOccurrencesRequest](#rode.v1alpha1.ListOccurrencesRequest)
    - [ListOccurrencesResponse](#rode.v1alpha1.ListOccurrencesResponse)
    - [ListResourcesRequest](#rode.v1alpha1.ListResourcesRequest)
    - [ListResourcesResponse](#rode.v1alpha1.ListResourcesResponse)
    - [ListVersionedResourceOccurrencesRequest](#rode.v1alpha1.ListVersionedResourceOccurrencesRequest)
    - [ListVersionedResourceOccurrencesResponse](#rode.v1alpha1.ListVersionedResourceOccurrencesResponse)
    - [RegisterCollectorRequest](#rode.v1alpha1.RegisterCollectorRequest)
    - [RegisterCollectorResponse](#rode.v1alpha1.RegisterCollectorResponse)
    - [RegisterCollectorResponse.NotesEntry](#rode.v1alpha1.RegisterCollectorResponse.NotesEntry)
    - [UpdateOccurrenceRequest](#rode.v1alpha1.UpdateOccurrenceRequest)
  
    - [Rode](#rode.v1alpha1.Rode)
  
- [Scalar Value Types](#scalar-value-types)



<a name="proto/v1alpha1/rode-policy.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## proto/v1alpha1/rode-policy.proto



<a name="rode.v1alpha1.DeletePolicyRequest"></a>

### DeletePolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |






<a name="rode.v1alpha1.EvaluatePolicyRequest"></a>

### EvaluatePolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy | [string](#string) |  |  |
| resource_uri | [string](#string) |  |  |






<a name="rode.v1alpha1.EvaluatePolicyResponse"></a>

### EvaluatePolicyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pass | [bool](#bool) |  |  |
| changed | [bool](#bool) |  |  |
| result | [EvaluatePolicyResult](#rode.v1alpha1.EvaluatePolicyResult) | repeated |  |
| explanation | [string](#string) | repeated |  |






<a name="rode.v1alpha1.EvaluatePolicyResult"></a>

### EvaluatePolicyResult



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pass | [bool](#bool) |  |  |
| created | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  |  |
| violations | [EvaluatePolicyViolation](#rode.v1alpha1.EvaluatePolicyViolation) | repeated |  |






<a name="rode.v1alpha1.EvaluatePolicyViolation"></a>

### EvaluatePolicyViolation



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| name | [string](#string) |  |  |
| description | [string](#string) |  |  |
| message | [string](#string) |  |  |
| link | [string](#string) |  |  |
| pass | [bool](#bool) |  |  |






<a name="rode.v1alpha1.GetPolicyRequest"></a>

### GetPolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |






<a name="rode.v1alpha1.ListPoliciesRequest"></a>

### ListPoliciesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| filter | [string](#string) |  |  |
| page_size | [int32](#int32) |  |  |
| page_token | [string](#string) |  |  |






<a name="rode.v1alpha1.ListPoliciesResponse"></a>

### ListPoliciesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policies | [Policy](#rode.v1alpha1.Policy) | repeated |  |
| next_page_token | [string](#string) |  |  |






<a name="rode.v1alpha1.Policy"></a>

### Policy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Unique autogenerate id |
| policy | [PolicyEntity](#rode.v1alpha1.PolicyEntity) |  |  |
| created | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  |  |
| updated | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  |  |






<a name="rode.v1alpha1.PolicyEntity"></a>

### PolicyEntity



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| description | [string](#string) |  |  |
| rego_content | [string](#string) |  | The rego code for the policy represented as a string |
| source_path | [string](#string) |  | The location of the policy stored in source control |






<a name="rode.v1alpha1.UpdatePolicyRequest"></a>

### UpdatePolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | the auto-generate id of the occurrence |
| policy | [PolicyEntity](#rode.v1alpha1.PolicyEntity) |  |  |
| update_mask | [google.protobuf.FieldMask](#google.protobuf.FieldMask) |  | The fields to update. |






<a name="rode.v1alpha1.ValidatePolicyRequest"></a>

### ValidatePolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy | [string](#string) |  |  |






<a name="rode.v1alpha1.ValidatePolicyResponse"></a>

### ValidatePolicyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy | [string](#string) |  |  |
| compile | [bool](#bool) |  |  |
| errors | [string](#string) | repeated |  |





 

 

 

 



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






<a name="rode.v1alpha1.GenericResource"></a>

### GenericResource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |






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






<a name="rode.v1alpha1.ListVersionedResourceOccurrencesResponse"></a>

### ListVersionedResourceOccurrencesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| occurrences | [grafeas.v1beta1.Occurrence](#grafeas.v1beta1.Occurrence) | repeated |  |
| next_page_token | [string](#string) |  |  |






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





 

 

 


<a name="rode.v1alpha1.Rode"></a>

### Rode


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| BatchCreateOccurrences | [BatchCreateOccurrencesRequest](#rode.v1alpha1.BatchCreateOccurrencesRequest) | [BatchCreateOccurrencesResponse](#rode.v1alpha1.BatchCreateOccurrencesResponse) | Create occurrences |
| EvaluatePolicy | [EvaluatePolicyRequest](#rode.v1alpha1.EvaluatePolicyRequest) | [EvaluatePolicyResponse](#rode.v1alpha1.EvaluatePolicyResponse) | Verify that an artifact satisfies a policy |
| ListResources | [ListResourcesRequest](#rode.v1alpha1.ListResourcesRequest) | [ListResourcesResponse](#rode.v1alpha1.ListResourcesResponse) | List resource URI |
| ListGenericResources | [ListGenericResourcesRequest](#rode.v1alpha1.ListGenericResourcesRequest) | [ListGenericResourcesResponse](#rode.v1alpha1.ListGenericResourcesResponse) |  |
| ListVersionedResourceOccurrences | [ListVersionedResourceOccurrencesRequest](#rode.v1alpha1.ListVersionedResourceOccurrencesRequest) | [ListVersionedResourceOccurrencesResponse](#rode.v1alpha1.ListVersionedResourceOccurrencesResponse) |  |
| ListOccurrences | [ListOccurrencesRequest](#rode.v1alpha1.ListOccurrencesRequest) | [ListOccurrencesResponse](#rode.v1alpha1.ListOccurrencesResponse) |  |
| UpdateOccurrence | [UpdateOccurrenceRequest](#rode.v1alpha1.UpdateOccurrenceRequest) | [.grafeas.v1beta1.Occurrence](#grafeas.v1beta1.Occurrence) |  |
| CreatePolicy | [PolicyEntity](#rode.v1alpha1.PolicyEntity) | [Policy](#rode.v1alpha1.Policy) |  |
| GetPolicy | [GetPolicyRequest](#rode.v1alpha1.GetPolicyRequest) | [Policy](#rode.v1alpha1.Policy) |  |
| DeletePolicy | [DeletePolicyRequest](#rode.v1alpha1.DeletePolicyRequest) | [.google.protobuf.Empty](#google.protobuf.Empty) |  |
| ListPolicies | [ListPoliciesRequest](#rode.v1alpha1.ListPoliciesRequest) | [ListPoliciesResponse](#rode.v1alpha1.ListPoliciesResponse) |  |
| ValidatePolicy | [ValidatePolicyRequest](#rode.v1alpha1.ValidatePolicyRequest) | [ValidatePolicyResponse](#rode.v1alpha1.ValidatePolicyResponse) |  |
| UpdatePolicy | [UpdatePolicyRequest](#rode.v1alpha1.UpdatePolicyRequest) | [Policy](#rode.v1alpha1.Policy) |  |
| RegisterCollector | [RegisterCollectorRequest](#rode.v1alpha1.RegisterCollectorRequest) | [RegisterCollectorResponse](#rode.v1alpha1.RegisterCollectorResponse) | RegisterCollector accepts a collector ID and a list of notes that this collector will reference when creating occurrences. The response will contain the notes with the fully qualified note name. This operation is idempotent, so any notes that already exist will not be re-created. Collectors are expected to invoke this RPC each time they start. |

 



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

