// Copyright 2021 The Rode Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.12.2
// source: proto/v1alpha1/rode-policy.proto

package v1alpha1

import (
	proto "github.com/golang/protobuf/proto"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

type EvaluatePolicyRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Policy      string `protobuf:"bytes,1,opt,name=policy,proto3" json:"policy,omitempty"`
	ResourceURI string `protobuf:"bytes,2,opt,name=resourceURI,proto3" json:"resourceURI,omitempty"`
}

func (x *EvaluatePolicyRequest) Reset() {
	*x = EvaluatePolicyRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_v1alpha1_rode_policy_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EvaluatePolicyRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EvaluatePolicyRequest) ProtoMessage() {}

func (x *EvaluatePolicyRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_v1alpha1_rode_policy_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EvaluatePolicyRequest.ProtoReflect.Descriptor instead.
func (*EvaluatePolicyRequest) Descriptor() ([]byte, []int) {
	return file_proto_v1alpha1_rode_policy_proto_rawDescGZIP(), []int{0}
}

func (x *EvaluatePolicyRequest) GetPolicy() string {
	if x != nil {
		return x.Policy
	}
	return ""
}

func (x *EvaluatePolicyRequest) GetResourceURI() string {
	if x != nil {
		return x.ResourceURI
	}
	return ""
}

type EvaluatePolicyResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Pass        bool                    `protobuf:"varint,1,opt,name=pass,proto3" json:"pass,omitempty"`
	Changed     bool                    `protobuf:"varint,2,opt,name=changed,proto3" json:"changed,omitempty"`
	Result      []*EvaluatePolicyResult `protobuf:"bytes,3,rep,name=result,proto3" json:"result,omitempty"`
	Explanation []string                `protobuf:"bytes,4,rep,name=explanation,proto3" json:"explanation,omitempty"`
}

func (x *EvaluatePolicyResponse) Reset() {
	*x = EvaluatePolicyResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_v1alpha1_rode_policy_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EvaluatePolicyResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EvaluatePolicyResponse) ProtoMessage() {}

func (x *EvaluatePolicyResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_v1alpha1_rode_policy_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EvaluatePolicyResponse.ProtoReflect.Descriptor instead.
func (*EvaluatePolicyResponse) Descriptor() ([]byte, []int) {
	return file_proto_v1alpha1_rode_policy_proto_rawDescGZIP(), []int{1}
}

func (x *EvaluatePolicyResponse) GetPass() bool {
	if x != nil {
		return x.Pass
	}
	return false
}

func (x *EvaluatePolicyResponse) GetChanged() bool {
	if x != nil {
		return x.Changed
	}
	return false
}

func (x *EvaluatePolicyResponse) GetResult() []*EvaluatePolicyResult {
	if x != nil {
		return x.Result
	}
	return nil
}

func (x *EvaluatePolicyResponse) GetExplanation() []string {
	if x != nil {
		return x.Explanation
	}
	return nil
}

type EvaluatePolicyResult struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Pass       bool                       `protobuf:"varint,1,opt,name=pass,proto3" json:"pass,omitempty"`
	Created    *timestamp.Timestamp       `protobuf:"bytes,2,opt,name=created,proto3" json:"created,omitempty"`
	Violations []*EvaluatePolicyViolation `protobuf:"bytes,3,rep,name=violations,proto3" json:"violations,omitempty"`
}

func (x *EvaluatePolicyResult) Reset() {
	*x = EvaluatePolicyResult{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_v1alpha1_rode_policy_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EvaluatePolicyResult) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EvaluatePolicyResult) ProtoMessage() {}

func (x *EvaluatePolicyResult) ProtoReflect() protoreflect.Message {
	mi := &file_proto_v1alpha1_rode_policy_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EvaluatePolicyResult.ProtoReflect.Descriptor instead.
func (*EvaluatePolicyResult) Descriptor() ([]byte, []int) {
	return file_proto_v1alpha1_rode_policy_proto_rawDescGZIP(), []int{2}
}

func (x *EvaluatePolicyResult) GetPass() bool {
	if x != nil {
		return x.Pass
	}
	return false
}

func (x *EvaluatePolicyResult) GetCreated() *timestamp.Timestamp {
	if x != nil {
		return x.Created
	}
	return nil
}

func (x *EvaluatePolicyResult) GetViolations() []*EvaluatePolicyViolation {
	if x != nil {
		return x.Violations
	}
	return nil
}

type EvaluatePolicyViolation struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id          string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Name        string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Description string `protobuf:"bytes,3,opt,name=description,proto3" json:"description,omitempty"`
	Message     string `protobuf:"bytes,4,opt,name=message,proto3" json:"message,omitempty"`
	Link        string `protobuf:"bytes,5,opt,name=link,proto3" json:"link,omitempty"`
	Pass        bool   `protobuf:"varint,6,opt,name=pass,proto3" json:"pass,omitempty"`
}

func (x *EvaluatePolicyViolation) Reset() {
	*x = EvaluatePolicyViolation{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_v1alpha1_rode_policy_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EvaluatePolicyViolation) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EvaluatePolicyViolation) ProtoMessage() {}

func (x *EvaluatePolicyViolation) ProtoReflect() protoreflect.Message {
	mi := &file_proto_v1alpha1_rode_policy_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EvaluatePolicyViolation.ProtoReflect.Descriptor instead.
func (*EvaluatePolicyViolation) Descriptor() ([]byte, []int) {
	return file_proto_v1alpha1_rode_policy_proto_rawDescGZIP(), []int{3}
}

func (x *EvaluatePolicyViolation) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *EvaluatePolicyViolation) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *EvaluatePolicyViolation) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

func (x *EvaluatePolicyViolation) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

func (x *EvaluatePolicyViolation) GetLink() string {
	if x != nil {
		return x.Link
	}
	return ""
}

func (x *EvaluatePolicyViolation) GetPass() bool {
	if x != nil {
		return x.Pass
	}
	return false
}

type GetPolicyRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (x *GetPolicyRequest) Reset() {
	*x = GetPolicyRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_v1alpha1_rode_policy_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetPolicyRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetPolicyRequest) ProtoMessage() {}

func (x *GetPolicyRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_v1alpha1_rode_policy_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetPolicyRequest.ProtoReflect.Descriptor instead.
func (*GetPolicyRequest) Descriptor() ([]byte, []int) {
	return file_proto_v1alpha1_rode_policy_proto_rawDescGZIP(), []int{4}
}

func (x *GetPolicyRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type PolicyEntity struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name        string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Description string `protobuf:"bytes,2,opt,name=description,proto3" json:"description,omitempty"`
	// The rego code for the policy represented as a string
	RegoContent string `protobuf:"bytes,3,opt,name=rego_content,json=regoContent,proto3" json:"rego_content,omitempty"`
	// The location of the policy stored in source control
	SourcePath string `protobuf:"bytes,4,opt,name=source_path,json=sourcePath,proto3" json:"source_path,omitempty"`
}

func (x *PolicyEntity) Reset() {
	*x = PolicyEntity{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_v1alpha1_rode_policy_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PolicyEntity) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PolicyEntity) ProtoMessage() {}

func (x *PolicyEntity) ProtoReflect() protoreflect.Message {
	mi := &file_proto_v1alpha1_rode_policy_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PolicyEntity.ProtoReflect.Descriptor instead.
func (*PolicyEntity) Descriptor() ([]byte, []int) {
	return file_proto_v1alpha1_rode_policy_proto_rawDescGZIP(), []int{5}
}

func (x *PolicyEntity) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *PolicyEntity) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

func (x *PolicyEntity) GetRegoContent() string {
	if x != nil {
		return x.RegoContent
	}
	return ""
}

func (x *PolicyEntity) GetSourcePath() string {
	if x != nil {
		return x.SourcePath
	}
	return ""
}

type Policy struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Unique autogenerate id
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	// The auto incremented version of the policy. This will auto increment on all updates
	Version int32         `protobuf:"varint,2,opt,name=version,proto3" json:"version,omitempty"`
	Policy  *PolicyEntity `protobuf:"bytes,3,opt,name=policy,proto3" json:"policy,omitempty"`
}

func (x *Policy) Reset() {
	*x = Policy{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_v1alpha1_rode_policy_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Policy) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Policy) ProtoMessage() {}

func (x *Policy) ProtoReflect() protoreflect.Message {
	mi := &file_proto_v1alpha1_rode_policy_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Policy.ProtoReflect.Descriptor instead.
func (*Policy) Descriptor() ([]byte, []int) {
	return file_proto_v1alpha1_rode_policy_proto_rawDescGZIP(), []int{6}
}

func (x *Policy) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Policy) GetVersion() int32 {
	if x != nil {
		return x.Version
	}
	return 0
}

func (x *Policy) GetPolicy() *PolicyEntity {
	if x != nil {
		return x.Policy
	}
	return nil
}

var File_proto_v1alpha1_rode_policy_proto protoreflect.FileDescriptor

var file_proto_v1alpha1_rode_policy_proto_rawDesc = []byte{
	0x0a, 0x20, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31,
	0x2f, 0x72, 0x6f, 0x64, 0x65, 0x2d, 0x70, 0x6f, 0x6c, 0x69, 0x63, 0x79, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x0d, 0x72, 0x6f, 0x64, 0x65, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x31, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x22, 0x51, 0x0a, 0x15, 0x45, 0x76, 0x61, 0x6c, 0x75, 0x61, 0x74, 0x65, 0x50, 0x6f,
	0x6c, 0x69, 0x63, 0x79, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x70,
	0x6f, 0x6c, 0x69, 0x63, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x70, 0x6f, 0x6c,
	0x69, 0x63, 0x79, 0x12, 0x20, 0x0a, 0x0b, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x55,
	0x52, 0x49, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x72, 0x65, 0x73, 0x6f, 0x75, 0x72,
	0x63, 0x65, 0x55, 0x52, 0x49, 0x22, 0xa5, 0x01, 0x0a, 0x16, 0x45, 0x76, 0x61, 0x6c, 0x75, 0x61,
	0x74, 0x65, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x79, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x12, 0x0a, 0x04, 0x70, 0x61, 0x73, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x04,
	0x70, 0x61, 0x73, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x68, 0x61, 0x6e, 0x67, 0x65, 0x64, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x63, 0x68, 0x61, 0x6e, 0x67, 0x65, 0x64, 0x12, 0x3b,
	0x0a, 0x06, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x23,
	0x2e, 0x72, 0x6f, 0x64, 0x65, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x45,
	0x76, 0x61, 0x6c, 0x75, 0x61, 0x74, 0x65, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x79, 0x52, 0x65, 0x73,
	0x75, 0x6c, 0x74, 0x52, 0x06, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x12, 0x20, 0x0a, 0x0b, 0x65,
	0x78, 0x70, 0x6c, 0x61, 0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x04, 0x20, 0x03, 0x28, 0x09,
	0x52, 0x0b, 0x65, 0x78, 0x70, 0x6c, 0x61, 0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0xa8, 0x01,
	0x0a, 0x14, 0x45, 0x76, 0x61, 0x6c, 0x75, 0x61, 0x74, 0x65, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x79,
	0x52, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x70, 0x61, 0x73, 0x73, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x08, 0x52, 0x04, 0x70, 0x61, 0x73, 0x73, 0x12, 0x34, 0x0a, 0x07, 0x63, 0x72,
	0x65, 0x61, 0x74, 0x65, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69,
	0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x07, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64,
	0x12, 0x46, 0x0a, 0x0a, 0x76, 0x69, 0x6f, 0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x03,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x26, 0x2e, 0x72, 0x6f, 0x64, 0x65, 0x2e, 0x76, 0x31, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x31, 0x2e, 0x45, 0x76, 0x61, 0x6c, 0x75, 0x61, 0x74, 0x65, 0x50, 0x6f, 0x6c,
	0x69, 0x63, 0x79, 0x56, 0x69, 0x6f, 0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0a, 0x76, 0x69,
	0x6f, 0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x22, 0xa1, 0x01, 0x0a, 0x17, 0x45, 0x76, 0x61,
	0x6c, 0x75, 0x61, 0x74, 0x65, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x79, 0x56, 0x69, 0x6f, 0x6c, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x02, 0x69, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x20, 0x0a, 0x0b, 0x64, 0x65, 0x73, 0x63,
	0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x64,
	0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x6c, 0x69, 0x6e, 0x6b, 0x18, 0x05, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x04, 0x6c, 0x69, 0x6e, 0x6b, 0x12, 0x12, 0x0a, 0x04, 0x70, 0x61, 0x73, 0x73,
	0x18, 0x06, 0x20, 0x01, 0x28, 0x08, 0x52, 0x04, 0x70, 0x61, 0x73, 0x73, 0x22, 0x22, 0x0a, 0x10,
	0x47, 0x65, 0x74, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x79, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64,
	0x22, 0x88, 0x01, 0x0a, 0x0c, 0x50, 0x6f, 0x6c, 0x69, 0x63, 0x79, 0x45, 0x6e, 0x74, 0x69, 0x74,
	0x79, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x20, 0x0a, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70,
	0x74, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x64, 0x65, 0x73, 0x63,
	0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x21, 0x0a, 0x0c, 0x72, 0x65, 0x67, 0x6f, 0x5f,
	0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x72,
	0x65, 0x67, 0x6f, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x12, 0x1f, 0x0a, 0x0b, 0x73, 0x6f,
	0x75, 0x72, 0x63, 0x65, 0x5f, 0x70, 0x61, 0x74, 0x68, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x0a, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x50, 0x61, 0x74, 0x68, 0x22, 0x67, 0x0a, 0x06, 0x50,
	0x6f, 0x6c, 0x69, 0x63, 0x79, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12,
	0x33, 0x0a, 0x06, 0x70, 0x6f, 0x6c, 0x69, 0x63, 0x79, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x1b, 0x2e, 0x72, 0x6f, 0x64, 0x65, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e,
	0x50, 0x6f, 0x6c, 0x69, 0x63, 0x79, 0x45, 0x6e, 0x74, 0x69, 0x74, 0x79, 0x52, 0x06, 0x70, 0x6f,
	0x6c, 0x69, 0x63, 0x79, 0x42, 0x25, 0x5a, 0x23, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63,
	0x6f, 0x6d, 0x2f, 0x72, 0x6f, 0x64, 0x65, 0x2f, 0x72, 0x6f, 0x64, 0x65, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_proto_v1alpha1_rode_policy_proto_rawDescOnce sync.Once
	file_proto_v1alpha1_rode_policy_proto_rawDescData = file_proto_v1alpha1_rode_policy_proto_rawDesc
)

func file_proto_v1alpha1_rode_policy_proto_rawDescGZIP() []byte {
	file_proto_v1alpha1_rode_policy_proto_rawDescOnce.Do(func() {
		file_proto_v1alpha1_rode_policy_proto_rawDescData = protoimpl.X.CompressGZIP(file_proto_v1alpha1_rode_policy_proto_rawDescData)
	})
	return file_proto_v1alpha1_rode_policy_proto_rawDescData
}

var file_proto_v1alpha1_rode_policy_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_proto_v1alpha1_rode_policy_proto_goTypes = []interface{}{
	(*EvaluatePolicyRequest)(nil),   // 0: rode.v1alpha1.EvaluatePolicyRequest
	(*EvaluatePolicyResponse)(nil),  // 1: rode.v1alpha1.EvaluatePolicyResponse
	(*EvaluatePolicyResult)(nil),    // 2: rode.v1alpha1.EvaluatePolicyResult
	(*EvaluatePolicyViolation)(nil), // 3: rode.v1alpha1.EvaluatePolicyViolation
	(*GetPolicyRequest)(nil),        // 4: rode.v1alpha1.GetPolicyRequest
	(*PolicyEntity)(nil),            // 5: rode.v1alpha1.PolicyEntity
	(*Policy)(nil),                  // 6: rode.v1alpha1.Policy
	(*timestamp.Timestamp)(nil),     // 7: google.protobuf.Timestamp
}
var file_proto_v1alpha1_rode_policy_proto_depIdxs = []int32{
	2, // 0: rode.v1alpha1.EvaluatePolicyResponse.result:type_name -> rode.v1alpha1.EvaluatePolicyResult
	7, // 1: rode.v1alpha1.EvaluatePolicyResult.created:type_name -> google.protobuf.Timestamp
	3, // 2: rode.v1alpha1.EvaluatePolicyResult.violations:type_name -> rode.v1alpha1.EvaluatePolicyViolation
	5, // 3: rode.v1alpha1.Policy.policy:type_name -> rode.v1alpha1.PolicyEntity
	4, // [4:4] is the sub-list for method output_type
	4, // [4:4] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_proto_v1alpha1_rode_policy_proto_init() }
func file_proto_v1alpha1_rode_policy_proto_init() {
	if File_proto_v1alpha1_rode_policy_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_proto_v1alpha1_rode_policy_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EvaluatePolicyRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_v1alpha1_rode_policy_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EvaluatePolicyResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_v1alpha1_rode_policy_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EvaluatePolicyResult); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_v1alpha1_rode_policy_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EvaluatePolicyViolation); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_v1alpha1_rode_policy_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetPolicyRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_v1alpha1_rode_policy_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PolicyEntity); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_v1alpha1_rode_policy_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Policy); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_proto_v1alpha1_rode_policy_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proto_v1alpha1_rode_policy_proto_goTypes,
		DependencyIndexes: file_proto_v1alpha1_rode_policy_proto_depIdxs,
		MessageInfos:      file_proto_v1alpha1_rode_policy_proto_msgTypes,
	}.Build()
	File_proto_v1alpha1_rode_policy_proto = out.File
	file_proto_v1alpha1_rode_policy_proto_rawDesc = nil
	file_proto_v1alpha1_rode_policy_proto_goTypes = nil
	file_proto_v1alpha1_rode_policy_proto_depIdxs = nil
}
