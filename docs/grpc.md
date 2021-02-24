# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [proto/v1alpha1/rode-attest.proto](#proto/v1alpha1/rode-attest.proto)
    - [AttestPolicyAttestation](#rode.v1alpha1.AttestPolicyAttestation)
    - [AttestPolicyRequest](#rode.v1alpha1.AttestPolicyRequest)
    - [AttestPolicyResponse](#rode.v1alpha1.AttestPolicyResponse)
    - [AttestPolicyViolation](#rode.v1alpha1.AttestPolicyViolation)
  
- [proto/v1alpha1/rode.proto](#proto/v1alpha1/rode.proto)
    - [BatchCreateOccurrencesRequest](#rode.v1alpha1.BatchCreateOccurrencesRequest)
    - [BatchCreateOccurrencesResponse](#rode.v1alpha1.BatchCreateOccurrencesResponse)
    - [ListOccurrencesRequest](#rode.v1alpha1.ListOccurrencesRequest)
    - [ListOccurrencesResponse](#rode.v1alpha1.ListOccurrencesResponse)
    - [ListResourcesRequest](#rode.v1alpha1.ListResourcesRequest)
    - [ListResourcesResponse](#rode.v1alpha1.ListResourcesResponse)
  
    - [Rode](#rode.v1alpha1.Rode)
  
- [Scalar Value Types](#scalar-value-types)



<a name="proto/v1alpha1/rode-attest.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## proto/v1alpha1/rode-attest.proto



<a name="rode.v1alpha1.AttestPolicyAttestation"></a>

### AttestPolicyAttestation



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| allow | [bool](#bool) |  |  |
| created | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  |  |
| violations | [AttestPolicyViolation](#rode.v1alpha1.AttestPolicyViolation) | repeated |  |






<a name="rode.v1alpha1.AttestPolicyRequest"></a>

### AttestPolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy | [string](#string) |  |  |
| resourceURI | [string](#string) |  |  |






<a name="rode.v1alpha1.AttestPolicyResponse"></a>

### AttestPolicyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| allow | [bool](#bool) |  |  |
| changed | [bool](#bool) |  |  |
| attestations | [AttestPolicyAttestation](#rode.v1alpha1.AttestPolicyAttestation) | repeated |  |






<a name="rode.v1alpha1.AttestPolicyViolation"></a>

### AttestPolicyViolation



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| name | [string](#string) |  |  |
| description | [string](#string) |  |  |
| link | [string](#string) |  |  |





 

 

 

 



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






<a name="rode.v1alpha1.ListOccurrencesRequest"></a>

### ListOccurrencesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| filter | [string](#string) |  |  |
| page_size | [int32](#int32) |  |  |
| page_token | [string](#string) |  |  |






<a name="rode.v1alpha1.ListOccurrencesResponse"></a>

### ListOccurrencesResponse
Response for listing occurrences.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| occurrences | [grafeas.v1beta1.Occurrence](#grafeas.v1beta1.Occurrence) | repeated | The occurrences requested. |
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





 

 

 


<a name="rode.v1alpha1.Rode"></a>

### Rode


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| BatchCreateOccurrences | [BatchCreateOccurrencesRequest](#rode.v1alpha1.BatchCreateOccurrencesRequest) | [BatchCreateOccurrencesResponse](#rode.v1alpha1.BatchCreateOccurrencesResponse) | Create occurrences |
| AttestPolicy | [AttestPolicyRequest](#rode.v1alpha1.AttestPolicyRequest) | [AttestPolicyResponse](#rode.v1alpha1.AttestPolicyResponse) | Verify that an artifact satisfies a policy |
| ListResources | [ListResourcesRequest](#rode.v1alpha1.ListResourcesRequest) | [ListResourcesResponse](#rode.v1alpha1.ListResourcesResponse) |  |
| ListOccurrences | [ListOccurrencesRequest](#rode.v1alpha1.ListOccurrencesRequest) | [ListOccurrencesResponse](#rode.v1alpha1.ListOccurrencesResponse) |  |

 



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

