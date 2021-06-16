# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [proto/v1alpha1/rode.proto](#proto/v1alpha1/rode.proto)
    - [BatchCreateOccurrencesRequest](#rode.v1alpha1.BatchCreateOccurrencesRequest)
    - [BatchCreateOccurrencesResponse](#rode.v1alpha1.BatchCreateOccurrencesResponse)
    - [CreateNoteRequest](#rode.v1alpha1.CreateNoteRequest)
    - [ListOccurrencesRequest](#rode.v1alpha1.ListOccurrencesRequest)
    - [ListOccurrencesResponse](#rode.v1alpha1.ListOccurrencesResponse)
    - [ListVersionedResourceOccurrencesRequest](#rode.v1alpha1.ListVersionedResourceOccurrencesRequest)
    - [ListVersionedResourceOccurrencesResponse](#rode.v1alpha1.ListVersionedResourceOccurrencesResponse)
    - [ListVersionedResourceOccurrencesResponse.RelatedNotesEntry](#rode.v1alpha1.ListVersionedResourceOccurrencesResponse.RelatedNotesEntry)
    - [RegisterCollectorRequest](#rode.v1alpha1.RegisterCollectorRequest)
    - [RegisterCollectorResponse](#rode.v1alpha1.RegisterCollectorResponse)
    - [RegisterCollectorResponse.NotesEntry](#rode.v1alpha1.RegisterCollectorResponse.NotesEntry)
    - [UpdateOccurrenceRequest](#rode.v1alpha1.UpdateOccurrenceRequest)
  
    - [Rode](#rode.v1alpha1.Rode)
  
- [proto/v1alpha1/rode_policy.proto](#proto/v1alpha1/rode_policy.proto)
    - [DeletePolicyRequest](#rode.v1alpha1.DeletePolicyRequest)
    - [EvaluatePolicyInput](#rode.v1alpha1.EvaluatePolicyInput)
    - [EvaluatePolicyRequest](#rode.v1alpha1.EvaluatePolicyRequest)
    - [EvaluatePolicyResponse](#rode.v1alpha1.EvaluatePolicyResponse)
    - [EvaluatePolicyResult](#rode.v1alpha1.EvaluatePolicyResult)
    - [EvaluatePolicyViolation](#rode.v1alpha1.EvaluatePolicyViolation)
    - [GetPolicyGroupRequest](#rode.v1alpha1.GetPolicyGroupRequest)
    - [GetPolicyRequest](#rode.v1alpha1.GetPolicyRequest)
    - [ListPoliciesRequest](#rode.v1alpha1.ListPoliciesRequest)
    - [ListPoliciesResponse](#rode.v1alpha1.ListPoliciesResponse)
    - [ListPolicyGroupsRequest](#rode.v1alpha1.ListPolicyGroupsRequest)
    - [ListPolicyGroupsResponse](#rode.v1alpha1.ListPolicyGroupsResponse)
    - [ListPolicyVersionsRequest](#rode.v1alpha1.ListPolicyVersionsRequest)
    - [ListPolicyVersionsResponse](#rode.v1alpha1.ListPolicyVersionsResponse)
    - [Policy](#rode.v1alpha1.Policy)
    - [PolicyEntity](#rode.v1alpha1.PolicyEntity)
    - [PolicyEvaluation](#rode.v1alpha1.PolicyEvaluation)
    - [PolicyGroup](#rode.v1alpha1.PolicyGroup)
    - [UpdatePolicyRequest](#rode.v1alpha1.UpdatePolicyRequest)
    - [ValidatePolicyRequest](#rode.v1alpha1.ValidatePolicyRequest)
    - [ValidatePolicyResponse](#rode.v1alpha1.ValidatePolicyResponse)
  
- [proto/v1alpha1/rode_resource.proto](#proto/v1alpha1/rode_resource.proto)
    - [ListResourceVersionsRequest](#rode.v1alpha1.ListResourceVersionsRequest)
    - [ListResourceVersionsResponse](#rode.v1alpha1.ListResourceVersionsResponse)
    - [ListResourcesRequest](#rode.v1alpha1.ListResourcesRequest)
    - [ListResourcesResponse](#rode.v1alpha1.ListResourcesResponse)
    - [Resource](#rode.v1alpha1.Resource)
    - [ResourceEvaluation](#rode.v1alpha1.ResourceEvaluation)
    - [ResourceEvaluationSource](#rode.v1alpha1.ResourceEvaluationSource)
    - [ResourceVersion](#rode.v1alpha1.ResourceVersion)
  
    - [ResourceType](#rode.v1alpha1.ResourceType)
  
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





 

 

 


<a name="rode.v1alpha1.Rode"></a>

### Rode


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| BatchCreateOccurrences | [BatchCreateOccurrencesRequest](#rode.v1alpha1.BatchCreateOccurrencesRequest) | [BatchCreateOccurrencesResponse](#rode.v1alpha1.BatchCreateOccurrencesResponse) | Create occurrences |
| EvaluatePolicy | [EvaluatePolicyRequest](#rode.v1alpha1.EvaluatePolicyRequest) | [EvaluatePolicyResponse](#rode.v1alpha1.EvaluatePolicyResponse) | Verify that an artifact satisfies a policy |
| ListResources | [ListResourcesRequest](#rode.v1alpha1.ListResourcesRequest) | [ListResourcesResponse](#rode.v1alpha1.ListResourcesResponse) |  |
| ListResourceVersions | [ListResourceVersionsRequest](#rode.v1alpha1.ListResourceVersionsRequest) | [ListResourceVersionsResponse](#rode.v1alpha1.ListResourceVersionsResponse) | ListResourceVersions can be used to list all known versions of a resource. Versions will always include the unique identifier (in the case of Docker images, the sha256) and will optionally include any related names (in the case of Docker images, any associated tags for the image). |
| ListVersionedResourceOccurrences | [ListVersionedResourceOccurrencesRequest](#rode.v1alpha1.ListVersionedResourceOccurrencesRequest) | [ListVersionedResourceOccurrencesResponse](#rode.v1alpha1.ListVersionedResourceOccurrencesResponse) |  |
| ListOccurrences | [ListOccurrencesRequest](#rode.v1alpha1.ListOccurrencesRequest) | [ListOccurrencesResponse](#rode.v1alpha1.ListOccurrencesResponse) |  |
| UpdateOccurrence | [UpdateOccurrenceRequest](#rode.v1alpha1.UpdateOccurrenceRequest) | [.grafeas.v1beta1.Occurrence](#grafeas.v1beta1.Occurrence) |  |
| CreatePolicy | [Policy](#rode.v1alpha1.Policy) | [Policy](#rode.v1alpha1.Policy) |  |
| GetPolicy | [GetPolicyRequest](#rode.v1alpha1.GetPolicyRequest) | [Policy](#rode.v1alpha1.Policy) |  |
| DeletePolicy | [DeletePolicyRequest](#rode.v1alpha1.DeletePolicyRequest) | [.google.protobuf.Empty](#google.protobuf.Empty) |  |
| ListPolicies | [ListPoliciesRequest](#rode.v1alpha1.ListPoliciesRequest) | [ListPoliciesResponse](#rode.v1alpha1.ListPoliciesResponse) |  |
| ListPolicyVersions | [ListPolicyVersionsRequest](#rode.v1alpha1.ListPolicyVersionsRequest) | [ListPolicyVersionsResponse](#rode.v1alpha1.ListPolicyVersionsResponse) |  |
| ValidatePolicy | [ValidatePolicyRequest](#rode.v1alpha1.ValidatePolicyRequest) | [ValidatePolicyResponse](#rode.v1alpha1.ValidatePolicyResponse) |  |
| UpdatePolicy | [UpdatePolicyRequest](#rode.v1alpha1.UpdatePolicyRequest) | [Policy](#rode.v1alpha1.Policy) |  |
| RegisterCollector | [RegisterCollectorRequest](#rode.v1alpha1.RegisterCollectorRequest) | [RegisterCollectorResponse](#rode.v1alpha1.RegisterCollectorResponse) | RegisterCollector accepts a collector ID and a list of notes that this collector will reference when creating occurrences. The response will contain the notes with the fully qualified note name. This operation is idempotent, so any notes that already exist will not be re-created. Collectors are expected to invoke this RPC each time they start. |
| CreateNote | [CreateNoteRequest](#rode.v1alpha1.CreateNoteRequest) | [.grafeas.v1beta1.Note](#grafeas.v1beta1.Note) | CreateNote acts as a simple proxy to the grafeas CreateNote rpc |
| CreatePolicyGroup | [PolicyGroup](#rode.v1alpha1.PolicyGroup) | [PolicyGroup](#rode.v1alpha1.PolicyGroup) |  |
| ListPolicyGroups | [ListPolicyGroupsRequest](#rode.v1alpha1.ListPolicyGroupsRequest) | [ListPolicyGroupsResponse](#rode.v1alpha1.ListPolicyGroupsResponse) |  |
| GetPolicyGroup | [GetPolicyGroupRequest](#rode.v1alpha1.GetPolicyGroupRequest) | [PolicyGroup](#rode.v1alpha1.PolicyGroup) |  |
| UpdatePolicyGroup | [PolicyGroup](#rode.v1alpha1.PolicyGroup) | [PolicyGroup](#rode.v1alpha1.PolicyGroup) |  |

 



<a name="proto/v1alpha1/rode_policy.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## proto/v1alpha1/rode_policy.proto



<a name="rode.v1alpha1.DeletePolicyRequest"></a>

### DeletePolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Id is the autogenerated id of the policy. |






<a name="rode.v1alpha1.EvaluatePolicyInput"></a>

### EvaluatePolicyInput
EvaluatePolicyInput is used as the input when evaluating a policy in OPA.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| occurrences | [grafeas.v1beta1.Occurrence](#grafeas.v1beta1.Occurrence) | repeated |  |






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






<a name="rode.v1alpha1.GetPolicyGroupRequest"></a>

### GetPolicyGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name is the unique identifier for the PolicyGroup. |






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






<a name="rode.v1alpha1.ListPolicyGroupsRequest"></a>

### ListPolicyGroupsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| filter | [string](#string) |  | Filter is a CEL (common expression language) filter that works off the fields in the PolicyGroup. |
| page_size | [int32](#int32) |  | PageSize is the maximum number of results. Use the ListPolicyGroupsResponse.NextPageToken to retrieve the next set. |
| page_token | [string](#string) |  | PageToken can be used to retrieve a specific page of results. |






<a name="rode.v1alpha1.ListPolicyGroupsResponse"></a>

### ListPolicyGroupsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy_groups | [PolicyGroup](#rode.v1alpha1.PolicyGroup) | repeated | PolicyGroups is the list of results from applying ListPolicyGroupsRequest.Filter, with a maximum number set by ListPolicyGroupsRequest.PageSize |
| next_page_token | [string](#string) |  | NextPageToken can be used to retrieve the subsequent page of results by setting ListPolicyGroupsRequest.NextPageToken |






<a name="rode.v1alpha1.ListPolicyVersionsRequest"></a>

### ListPolicyVersionsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Id is the unique policy identifier |
| filter | [string](#string) |  | Filter is a CEL expression that can be used to constrain which policy versions are returned. |
| page_size | [int32](#int32) |  | PageSize controls the number of results. |
| page_token | [string](#string) |  | PageToken can be used to retrieve a specific page of results. The response will include the next page token. |






<a name="rode.v1alpha1.ListPolicyVersionsResponse"></a>

### ListPolicyVersionsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| versions | [PolicyEntity](#rode.v1alpha1.PolicyEntity) | repeated | Versions is the list of policy versions matching the filter. |
| next_page_token | [string](#string) |  | NextPageToken can be used to retrieve the next set of results. |






<a name="rode.v1alpha1.Policy"></a>

### Policy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Id is the unique autogenerated identifier of a policy. |
| name | [string](#string) |  | Name of the policy |
| description | [string](#string) |  | Description should be a brief message about the intention of the policy. Updates to a policy can be described in the PolicyEntity.Message field. |
| current_version | [uint32](#uint32) |  | CurrentVersion is the default policy version that&#39;s used when a policy is retrieved or evaluated. It&#39;s not necessarily the latest, as it may be overwritten if an older policy version should be used instead. |
| policy | [PolicyEntity](#rode.v1alpha1.PolicyEntity) |  | Policy contains the Rego policy code or a source location. The PolicyEntity.Version matches CurrentVersion unless it was otherwise specified. |
| created | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  | Created is when the policy was first stored. |
| updated | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  | Updated indicates when either an edit occurred on the policy itself or a new policy version was created. |
| deleted | [bool](#bool) |  | Deleted is a flag controlling soft deletes. Deleted policies won&#39;t be returned by the ListPolicies RPC, but can still be retrieved and evaluated. |






<a name="rode.v1alpha1.PolicyEntity"></a>

### PolicyEntity



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| version | [uint32](#uint32) |  | Version is a number that represents revisions of a policy. Policy contents are immutable, so changes to the source are represented as new versions. |
| message | [string](#string) |  | Message should contain a brief summary of the changes to the policy code between the current version and the previous version. |
| rego_content | [string](#string) |  | RegoContent contains the Rego code for a given policy. Only one of RegoContent and SourcePath should be specified. |
| source_path | [string](#string) |  | SourcePath is the location of the policy stored in source control. |
| created | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  | Created represents when this policy version was stored. Policy contents are immutable, so there is no corresponding Updated field. |






<a name="rode.v1alpha1.PolicyEvaluation"></a>

### PolicyEvaluation
PolicyEvaluation describes the result of a request to evaluate a particular resource version against a specific policy.
This is a child of ResourceEvaluation.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Id represents the unique identifier (UUID) for this particular policy evaluation. |
| resource_evaluation_id | [string](#string) |  | ResourceEvaluationId represents the unique identifier (UUID) of the resource evaluation that triggered this policy evaluation. |
| pass | [bool](#bool) |  | Pass represents the overall status for this policy evaluation. |
| policy_version_id | [string](#string) |  | PolicyVersionId represents the ID of the policy version that was evaluated. |
| violations | [EvaluatePolicyViolation](#rode.v1alpha1.EvaluatePolicyViolation) | repeated | Violations is a list of rule results. Even if a rule passed, its output will be included in Violations. |






<a name="rode.v1alpha1.PolicyGroup"></a>

### PolicyGroup
PolicyGroup is used to apply multiple policies in a single resource evaluation. It&#39;s linked to a policy via a PolicyAssignment.
A PolicyGroup is meant to be open-ended -- it can represent an environment (e.g., dev) or
policies around a certain compliance framework (e.g., PCI).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name is the unique identifier for the PolicyGroup. It may only contain lowercase alphanumeric characters, dashes, and underscores. It cannot be changed after creation. |
| description | [string](#string) |  | Description is a brief summary of the intended use for the PolicyGroup. |
| created | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  |  |
| updated | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  |  |






<a name="rode.v1alpha1.UpdatePolicyRequest"></a>

### UpdatePolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy | [Policy](#rode.v1alpha1.Policy) |  | Policy is the Policy message. Only Policy.Name, Policy.Description, and Policy.CurrentVersion can be updated. Changes to Policy.Policy are represented as new versions of a policy. |






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





 

 

 

 



<a name="proto/v1alpha1/rode_resource.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## proto/v1alpha1/rode_resource.proto



<a name="rode.v1alpha1.ListResourceVersionsRequest"></a>

### ListResourceVersionsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| filter | [string](#string) |  |  |
| page_size | [int32](#int32) |  |  |
| page_token | [string](#string) |  |  |






<a name="rode.v1alpha1.ListResourceVersionsResponse"></a>

### ListResourceVersionsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| versions | [ResourceVersion](#rode.v1alpha1.ResourceVersion) | repeated |  |
| next_page_token | [string](#string) |  |  |






<a name="rode.v1alpha1.ListResourcesRequest"></a>

### ListResourcesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| filter | [string](#string) |  |  |
| page_size | [int32](#int32) |  |  |
| page_token | [string](#string) |  |  |






<a name="rode.v1alpha1.ListResourcesResponse"></a>

### ListResourcesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resources | [Resource](#rode.v1alpha1.Resource) | repeated |  |
| next_page_token | [string](#string) |  |  |






<a name="rode.v1alpha1.Resource"></a>

### Resource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Id represents the unique id of the resource. This is usually the resource prefix plus the name, except in the case of Docker images. The id is used as a parameter for the ListResourceVersions RPC. |
| name | [string](#string) |  | Name represents the name of this resource as seen on the UI. |
| type | [ResourceType](#rode.v1alpha1.ResourceType) |  | Type represents the resource type for this resource, such as &#34;DOCKER&#34; or &#34;GIT&#34; |
| created | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  |  |






<a name="rode.v1alpha1.ResourceEvaluation"></a>

### ResourceEvaluation
ResourceEvaluation describes the result of a request to evaluate a particular resource version against a group of policies.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | Id represents the unique identifier (UUID) for this particular resource evaluation. |
| pass | [bool](#bool) |  | Pass represents the overall status for this resource evaluation. This is determined by looking at each policy evaluation result and performing an AND on each one. If Pass is true, this means that the referenced resource version passed each policy within the policy group at the time that the evaluation was performed. |
| source | [ResourceEvaluationSource](#rode.v1alpha1.ResourceEvaluationSource) |  | Source represents the source of the resource evaluation request. This should be set by the enforcer or entity performing the request. |
| created | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  |  |
| resource_version | [ResourceVersion](#rode.v1alpha1.ResourceVersion) |  | ResourceVersion represents the specific resource version that was evaluated in this request. |
| policy_group | [string](#string) |  | PolicyGroup represents the name of the policy group that was evaluated in this request. |






<a name="rode.v1alpha1.ResourceEvaluationSource"></a>

### ResourceEvaluationSource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| url | [string](#string) |  |  |






<a name="rode.v1alpha1.ResourceVersion"></a>

### ResourceVersion



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| version | [string](#string) |  | Version represents the unique artifact version as a fully qualified URI. Example: a Docker image version might look like this: harbor.liatr.io/rode-demo/node-app@sha256:a235554754f9bf075ac1c1b70c224ef5997176b776f0c56e340aeb63f429ace8 |
| names | [string](#string) | repeated | Names represents related artifact names, if they exist. This information will be sourced from build occurrences. Example: a Docker image name might look like this: harbor.liatr.io/rode-demo/node-app:latest |
| created | [google.protobuf.Timestamp](#google.protobuf.Timestamp) |  |  |





 


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

