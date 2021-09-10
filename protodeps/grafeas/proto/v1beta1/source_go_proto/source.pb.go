// Copyright 2018 The Grafeas Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.15.7
// source: proto/v1beta1/source.proto

package source_go_proto

import (
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

// The type of an alias.
type AliasContext_Kind int32

const (
	// Unknown.
	AliasContext_KIND_UNSPECIFIED AliasContext_Kind = 0
	// Git tag.
	AliasContext_FIXED AliasContext_Kind = 1
	// Git branch.
	AliasContext_MOVABLE AliasContext_Kind = 2
	// Used to specify non-standard aliases. For example, if a Git repo has a
	// ref named "refs/foo/bar".
	AliasContext_OTHER AliasContext_Kind = 4
)

// Enum value maps for AliasContext_Kind.
var (
	AliasContext_Kind_name = map[int32]string{
		0: "KIND_UNSPECIFIED",
		1: "FIXED",
		2: "MOVABLE",
		4: "OTHER",
	}
	AliasContext_Kind_value = map[string]int32{
		"KIND_UNSPECIFIED": 0,
		"FIXED":            1,
		"MOVABLE":          2,
		"OTHER":            4,
	}
)

func (x AliasContext_Kind) Enum() *AliasContext_Kind {
	p := new(AliasContext_Kind)
	*p = x
	return p
}

func (x AliasContext_Kind) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (AliasContext_Kind) Descriptor() protoreflect.EnumDescriptor {
	return file_proto_v1beta1_source_proto_enumTypes[0].Descriptor()
}

func (AliasContext_Kind) Type() protoreflect.EnumType {
	return &file_proto_v1beta1_source_proto_enumTypes[0]
}

func (x AliasContext_Kind) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use AliasContext_Kind.Descriptor instead.
func (AliasContext_Kind) EnumDescriptor() ([]byte, []int) {
	return file_proto_v1beta1_source_proto_rawDescGZIP(), []int{1, 0}
}

// A SourceContext is a reference to a tree of files. A SourceContext together
// with a path point to a unique revision of a single file or directory.
type SourceContext struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// A SourceContext can refer any one of the following types of repositories.
	//
	// Types that are assignable to Context:
	//	*SourceContext_CloudRepo
	//	*SourceContext_Gerrit
	//	*SourceContext_Git
	Context isSourceContext_Context `protobuf_oneof:"context"`
	// Labels with user defined metadata.
	Labels map[string]string `protobuf:"bytes,4,rep,name=labels,proto3" json:"labels,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *SourceContext) Reset() {
	*x = SourceContext{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_v1beta1_source_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SourceContext) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SourceContext) ProtoMessage() {}

func (x *SourceContext) ProtoReflect() protoreflect.Message {
	mi := &file_proto_v1beta1_source_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SourceContext.ProtoReflect.Descriptor instead.
func (*SourceContext) Descriptor() ([]byte, []int) {
	return file_proto_v1beta1_source_proto_rawDescGZIP(), []int{0}
}

func (m *SourceContext) GetContext() isSourceContext_Context {
	if m != nil {
		return m.Context
	}
	return nil
}

func (x *SourceContext) GetCloudRepo() *CloudRepoSourceContext {
	if x, ok := x.GetContext().(*SourceContext_CloudRepo); ok {
		return x.CloudRepo
	}
	return nil
}

func (x *SourceContext) GetGerrit() *GerritSourceContext {
	if x, ok := x.GetContext().(*SourceContext_Gerrit); ok {
		return x.Gerrit
	}
	return nil
}

func (x *SourceContext) GetGit() *GitSourceContext {
	if x, ok := x.GetContext().(*SourceContext_Git); ok {
		return x.Git
	}
	return nil
}

func (x *SourceContext) GetLabels() map[string]string {
	if x != nil {
		return x.Labels
	}
	return nil
}

type isSourceContext_Context interface {
	isSourceContext_Context()
}

type SourceContext_CloudRepo struct {
	// A SourceContext referring to a revision in a Google Cloud Source Repo.
	CloudRepo *CloudRepoSourceContext `protobuf:"bytes,1,opt,name=cloud_repo,json=cloudRepo,proto3,oneof"`
}

type SourceContext_Gerrit struct {
	// A SourceContext referring to a Gerrit project.
	Gerrit *GerritSourceContext `protobuf:"bytes,2,opt,name=gerrit,proto3,oneof"`
}

type SourceContext_Git struct {
	// A SourceContext referring to any third party Git repo (e.g., GitHub).
	Git *GitSourceContext `protobuf:"bytes,3,opt,name=git,proto3,oneof"`
}

func (*SourceContext_CloudRepo) isSourceContext_Context() {}

func (*SourceContext_Gerrit) isSourceContext_Context() {}

func (*SourceContext_Git) isSourceContext_Context() {}

// An alias to a repo revision.
type AliasContext struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The alias kind.
	Kind AliasContext_Kind `protobuf:"varint,1,opt,name=kind,proto3,enum=grafeas.v1beta1.source.AliasContext_Kind" json:"kind,omitempty"`
	// The alias name.
	Name string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
}

func (x *AliasContext) Reset() {
	*x = AliasContext{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_v1beta1_source_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AliasContext) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AliasContext) ProtoMessage() {}

func (x *AliasContext) ProtoReflect() protoreflect.Message {
	mi := &file_proto_v1beta1_source_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AliasContext.ProtoReflect.Descriptor instead.
func (*AliasContext) Descriptor() ([]byte, []int) {
	return file_proto_v1beta1_source_proto_rawDescGZIP(), []int{1}
}

func (x *AliasContext) GetKind() AliasContext_Kind {
	if x != nil {
		return x.Kind
	}
	return AliasContext_KIND_UNSPECIFIED
}

func (x *AliasContext) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

// A CloudRepoSourceContext denotes a particular revision in a Google Cloud
// Source Repo.
type CloudRepoSourceContext struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The ID of the repo.
	RepoId *RepoId `protobuf:"bytes,1,opt,name=repo_id,json=repoId,proto3" json:"repo_id,omitempty"`
	// A revision in a Cloud Repo can be identified by either its revision ID or
	// its alias.
	//
	// Types that are assignable to Revision:
	//	*CloudRepoSourceContext_RevisionId
	//	*CloudRepoSourceContext_AliasContext
	Revision isCloudRepoSourceContext_Revision `protobuf_oneof:"revision"`
}

func (x *CloudRepoSourceContext) Reset() {
	*x = CloudRepoSourceContext{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_v1beta1_source_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CloudRepoSourceContext) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CloudRepoSourceContext) ProtoMessage() {}

func (x *CloudRepoSourceContext) ProtoReflect() protoreflect.Message {
	mi := &file_proto_v1beta1_source_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CloudRepoSourceContext.ProtoReflect.Descriptor instead.
func (*CloudRepoSourceContext) Descriptor() ([]byte, []int) {
	return file_proto_v1beta1_source_proto_rawDescGZIP(), []int{2}
}

func (x *CloudRepoSourceContext) GetRepoId() *RepoId {
	if x != nil {
		return x.RepoId
	}
	return nil
}

func (m *CloudRepoSourceContext) GetRevision() isCloudRepoSourceContext_Revision {
	if m != nil {
		return m.Revision
	}
	return nil
}

func (x *CloudRepoSourceContext) GetRevisionId() string {
	if x, ok := x.GetRevision().(*CloudRepoSourceContext_RevisionId); ok {
		return x.RevisionId
	}
	return ""
}

func (x *CloudRepoSourceContext) GetAliasContext() *AliasContext {
	if x, ok := x.GetRevision().(*CloudRepoSourceContext_AliasContext); ok {
		return x.AliasContext
	}
	return nil
}

type isCloudRepoSourceContext_Revision interface {
	isCloudRepoSourceContext_Revision()
}

type CloudRepoSourceContext_RevisionId struct {
	// A revision ID.
	RevisionId string `protobuf:"bytes,2,opt,name=revision_id,json=revisionId,proto3,oneof"`
}

type CloudRepoSourceContext_AliasContext struct {
	// An alias, which may be a branch or tag.
	AliasContext *AliasContext `protobuf:"bytes,3,opt,name=alias_context,json=aliasContext,proto3,oneof"`
}

func (*CloudRepoSourceContext_RevisionId) isCloudRepoSourceContext_Revision() {}

func (*CloudRepoSourceContext_AliasContext) isCloudRepoSourceContext_Revision() {}

// A SourceContext referring to a Gerrit project.
type GerritSourceContext struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The URI of a running Gerrit instance.
	HostUri string `protobuf:"bytes,1,opt,name=host_uri,json=hostUri,proto3" json:"host_uri,omitempty"`
	// The full project name within the host. Projects may be nested, so
	// "project/subproject" is a valid project name. The "repo name" is the
	// hostURI/project.
	GerritProject string `protobuf:"bytes,2,opt,name=gerrit_project,json=gerritProject,proto3" json:"gerrit_project,omitempty"`
	// A revision in a Gerrit project can be identified by either its revision ID
	// or its alias.
	//
	// Types that are assignable to Revision:
	//	*GerritSourceContext_RevisionId
	//	*GerritSourceContext_AliasContext
	Revision isGerritSourceContext_Revision `protobuf_oneof:"revision"`
}

func (x *GerritSourceContext) Reset() {
	*x = GerritSourceContext{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_v1beta1_source_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GerritSourceContext) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GerritSourceContext) ProtoMessage() {}

func (x *GerritSourceContext) ProtoReflect() protoreflect.Message {
	mi := &file_proto_v1beta1_source_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GerritSourceContext.ProtoReflect.Descriptor instead.
func (*GerritSourceContext) Descriptor() ([]byte, []int) {
	return file_proto_v1beta1_source_proto_rawDescGZIP(), []int{3}
}

func (x *GerritSourceContext) GetHostUri() string {
	if x != nil {
		return x.HostUri
	}
	return ""
}

func (x *GerritSourceContext) GetGerritProject() string {
	if x != nil {
		return x.GerritProject
	}
	return ""
}

func (m *GerritSourceContext) GetRevision() isGerritSourceContext_Revision {
	if m != nil {
		return m.Revision
	}
	return nil
}

func (x *GerritSourceContext) GetRevisionId() string {
	if x, ok := x.GetRevision().(*GerritSourceContext_RevisionId); ok {
		return x.RevisionId
	}
	return ""
}

func (x *GerritSourceContext) GetAliasContext() *AliasContext {
	if x, ok := x.GetRevision().(*GerritSourceContext_AliasContext); ok {
		return x.AliasContext
	}
	return nil
}

type isGerritSourceContext_Revision interface {
	isGerritSourceContext_Revision()
}

type GerritSourceContext_RevisionId struct {
	// A revision (commit) ID.
	RevisionId string `protobuf:"bytes,3,opt,name=revision_id,json=revisionId,proto3,oneof"`
}

type GerritSourceContext_AliasContext struct {
	// An alias, which may be a branch or tag.
	AliasContext *AliasContext `protobuf:"bytes,4,opt,name=alias_context,json=aliasContext,proto3,oneof"`
}

func (*GerritSourceContext_RevisionId) isGerritSourceContext_Revision() {}

func (*GerritSourceContext_AliasContext) isGerritSourceContext_Revision() {}

// A GitSourceContext denotes a particular revision in a third party Git
// repository (e.g., GitHub).
type GitSourceContext struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Git repository URL.
	Url string `protobuf:"bytes,1,opt,name=url,proto3" json:"url,omitempty"`
	// Git commit hash.
	RevisionId string `protobuf:"bytes,2,opt,name=revision_id,json=revisionId,proto3" json:"revision_id,omitempty"`
}

func (x *GitSourceContext) Reset() {
	*x = GitSourceContext{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_v1beta1_source_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GitSourceContext) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GitSourceContext) ProtoMessage() {}

func (x *GitSourceContext) ProtoReflect() protoreflect.Message {
	mi := &file_proto_v1beta1_source_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GitSourceContext.ProtoReflect.Descriptor instead.
func (*GitSourceContext) Descriptor() ([]byte, []int) {
	return file_proto_v1beta1_source_proto_rawDescGZIP(), []int{4}
}

func (x *GitSourceContext) GetUrl() string {
	if x != nil {
		return x.Url
	}
	return ""
}

func (x *GitSourceContext) GetRevisionId() string {
	if x != nil {
		return x.RevisionId
	}
	return ""
}

// A unique identifier for a Cloud Repo.
type RepoId struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// A cloud repo can be identified by either its project ID and repository name
	// combination, or its globally unique identifier.
	//
	// Types that are assignable to Id:
	//	*RepoId_ProjectRepoId
	//	*RepoId_Uid
	Id isRepoId_Id `protobuf_oneof:"id"`
}

func (x *RepoId) Reset() {
	*x = RepoId{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_v1beta1_source_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RepoId) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RepoId) ProtoMessage() {}

func (x *RepoId) ProtoReflect() protoreflect.Message {
	mi := &file_proto_v1beta1_source_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RepoId.ProtoReflect.Descriptor instead.
func (*RepoId) Descriptor() ([]byte, []int) {
	return file_proto_v1beta1_source_proto_rawDescGZIP(), []int{5}
}

func (m *RepoId) GetId() isRepoId_Id {
	if m != nil {
		return m.Id
	}
	return nil
}

func (x *RepoId) GetProjectRepoId() *ProjectRepoId {
	if x, ok := x.GetId().(*RepoId_ProjectRepoId); ok {
		return x.ProjectRepoId
	}
	return nil
}

func (x *RepoId) GetUid() string {
	if x, ok := x.GetId().(*RepoId_Uid); ok {
		return x.Uid
	}
	return ""
}

type isRepoId_Id interface {
	isRepoId_Id()
}

type RepoId_ProjectRepoId struct {
	// A combination of a project ID and a repo name.
	ProjectRepoId *ProjectRepoId `protobuf:"bytes,1,opt,name=project_repo_id,json=projectRepoId,proto3,oneof"`
}

type RepoId_Uid struct {
	// A server-assigned, globally unique identifier.
	Uid string `protobuf:"bytes,2,opt,name=uid,proto3,oneof"`
}

func (*RepoId_ProjectRepoId) isRepoId_Id() {}

func (*RepoId_Uid) isRepoId_Id() {}

// Selects a repo using a Google Cloud Platform project ID (e.g.,
// winged-cargo-31) and a repo name within that project.
type ProjectRepoId struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The ID of the project.
	ProjectId string `protobuf:"bytes,1,opt,name=project_id,json=projectId,proto3" json:"project_id,omitempty"`
	// The name of the repo. Leave empty for the default repo.
	RepoName string `protobuf:"bytes,2,opt,name=repo_name,json=repoName,proto3" json:"repo_name,omitempty"`
}

func (x *ProjectRepoId) Reset() {
	*x = ProjectRepoId{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_v1beta1_source_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProjectRepoId) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProjectRepoId) ProtoMessage() {}

func (x *ProjectRepoId) ProtoReflect() protoreflect.Message {
	mi := &file_proto_v1beta1_source_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProjectRepoId.ProtoReflect.Descriptor instead.
func (*ProjectRepoId) Descriptor() ([]byte, []int) {
	return file_proto_v1beta1_source_proto_rawDescGZIP(), []int{6}
}

func (x *ProjectRepoId) GetProjectId() string {
	if x != nil {
		return x.ProjectId
	}
	return ""
}

func (x *ProjectRepoId) GetRepoName() string {
	if x != nil {
		return x.RepoName
	}
	return ""
}

var File_proto_v1beta1_source_proto protoreflect.FileDescriptor

var file_proto_v1beta1_source_proto_rawDesc = []byte{
	0x0a, 0x1a, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2f,
	0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x16, 0x67, 0x72,
	0x61, 0x66, 0x65, 0x61, 0x73, 0x2e, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2e, 0x73, 0x6f,
	0x75, 0x72, 0x63, 0x65, 0x22, 0xf6, 0x02, 0x0a, 0x0d, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x43,
	0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74, 0x12, 0x4f, 0x0a, 0x0a, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x5f,
	0x72, 0x65, 0x70, 0x6f, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2e, 0x2e, 0x67, 0x72, 0x61,
	0x66, 0x65, 0x61, 0x73, 0x2e, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2e, 0x73, 0x6f, 0x75,
	0x72, 0x63, 0x65, 0x2e, 0x43, 0x6c, 0x6f, 0x75, 0x64, 0x52, 0x65, 0x70, 0x6f, 0x53, 0x6f, 0x75,
	0x72, 0x63, 0x65, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74, 0x48, 0x00, 0x52, 0x09, 0x63, 0x6c,
	0x6f, 0x75, 0x64, 0x52, 0x65, 0x70, 0x6f, 0x12, 0x45, 0x0a, 0x06, 0x67, 0x65, 0x72, 0x72, 0x69,
	0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2b, 0x2e, 0x67, 0x72, 0x61, 0x66, 0x65, 0x61,
	0x73, 0x2e, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2e, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65,
	0x2e, 0x47, 0x65, 0x72, 0x72, 0x69, 0x74, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x43, 0x6f, 0x6e,
	0x74, 0x65, 0x78, 0x74, 0x48, 0x00, 0x52, 0x06, 0x67, 0x65, 0x72, 0x72, 0x69, 0x74, 0x12, 0x3c,
	0x0a, 0x03, 0x67, 0x69, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x28, 0x2e, 0x67, 0x72,
	0x61, 0x66, 0x65, 0x61, 0x73, 0x2e, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2e, 0x73, 0x6f,
	0x75, 0x72, 0x63, 0x65, 0x2e, 0x47, 0x69, 0x74, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x43, 0x6f,
	0x6e, 0x74, 0x65, 0x78, 0x74, 0x48, 0x00, 0x52, 0x03, 0x67, 0x69, 0x74, 0x12, 0x49, 0x0a, 0x06,
	0x6c, 0x61, 0x62, 0x65, 0x6c, 0x73, 0x18, 0x04, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x31, 0x2e, 0x67,
	0x72, 0x61, 0x66, 0x65, 0x61, 0x73, 0x2e, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2e, 0x73,
	0x6f, 0x75, 0x72, 0x63, 0x65, 0x2e, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x43, 0x6f, 0x6e, 0x74,
	0x65, 0x78, 0x74, 0x2e, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52,
	0x06, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x73, 0x1a, 0x39, 0x0a, 0x0b, 0x4c, 0x61, 0x62, 0x65, 0x6c,
	0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02,
	0x38, 0x01, 0x42, 0x09, 0x0a, 0x07, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74, 0x22, 0xa2, 0x01,
	0x0a, 0x0c, 0x41, 0x6c, 0x69, 0x61, 0x73, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74, 0x12, 0x3d,
	0x0a, 0x04, 0x6b, 0x69, 0x6e, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x29, 0x2e, 0x67,
	0x72, 0x61, 0x66, 0x65, 0x61, 0x73, 0x2e, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2e, 0x73,
	0x6f, 0x75, 0x72, 0x63, 0x65, 0x2e, 0x41, 0x6c, 0x69, 0x61, 0x73, 0x43, 0x6f, 0x6e, 0x74, 0x65,
	0x78, 0x74, 0x2e, 0x4b, 0x69, 0x6e, 0x64, 0x52, 0x04, 0x6b, 0x69, 0x6e, 0x64, 0x12, 0x12, 0x0a,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d,
	0x65, 0x22, 0x3f, 0x0a, 0x04, 0x4b, 0x69, 0x6e, 0x64, 0x12, 0x14, 0x0a, 0x10, 0x4b, 0x49, 0x4e,
	0x44, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46, 0x49, 0x45, 0x44, 0x10, 0x00, 0x12,
	0x09, 0x0a, 0x05, 0x46, 0x49, 0x58, 0x45, 0x44, 0x10, 0x01, 0x12, 0x0b, 0x0a, 0x07, 0x4d, 0x4f,
	0x56, 0x41, 0x42, 0x4c, 0x45, 0x10, 0x02, 0x12, 0x09, 0x0a, 0x05, 0x4f, 0x54, 0x48, 0x45, 0x52,
	0x10, 0x04, 0x22, 0xcd, 0x01, 0x0a, 0x16, 0x43, 0x6c, 0x6f, 0x75, 0x64, 0x52, 0x65, 0x70, 0x6f,
	0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74, 0x12, 0x37, 0x0a,
	0x07, 0x72, 0x65, 0x70, 0x6f, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1e,
	0x2e, 0x67, 0x72, 0x61, 0x66, 0x65, 0x61, 0x73, 0x2e, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31,
	0x2e, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x2e, 0x52, 0x65, 0x70, 0x6f, 0x49, 0x64, 0x52, 0x06,
	0x72, 0x65, 0x70, 0x6f, 0x49, 0x64, 0x12, 0x21, 0x0a, 0x0b, 0x72, 0x65, 0x76, 0x69, 0x73, 0x69,
	0x6f, 0x6e, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x0a, 0x72,
	0x65, 0x76, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x12, 0x4b, 0x0a, 0x0d, 0x61, 0x6c, 0x69,
	0x61, 0x73, 0x5f, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x24, 0x2e, 0x67, 0x72, 0x61, 0x66, 0x65, 0x61, 0x73, 0x2e, 0x76, 0x31, 0x62, 0x65, 0x74,
	0x61, 0x31, 0x2e, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x2e, 0x41, 0x6c, 0x69, 0x61, 0x73, 0x43,
	0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74, 0x48, 0x00, 0x52, 0x0c, 0x61, 0x6c, 0x69, 0x61, 0x73, 0x43,
	0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74, 0x42, 0x0a, 0x0a, 0x08, 0x72, 0x65, 0x76, 0x69, 0x73, 0x69,
	0x6f, 0x6e, 0x22, 0xd3, 0x01, 0x0a, 0x13, 0x47, 0x65, 0x72, 0x72, 0x69, 0x74, 0x53, 0x6f, 0x75,
	0x72, 0x63, 0x65, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74, 0x12, 0x19, 0x0a, 0x08, 0x68, 0x6f,
	0x73, 0x74, 0x5f, 0x75, 0x72, 0x69, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x68, 0x6f,
	0x73, 0x74, 0x55, 0x72, 0x69, 0x12, 0x25, 0x0a, 0x0e, 0x67, 0x65, 0x72, 0x72, 0x69, 0x74, 0x5f,
	0x70, 0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x67,
	0x65, 0x72, 0x72, 0x69, 0x74, 0x50, 0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74, 0x12, 0x21, 0x0a, 0x0b,
	0x72, 0x65, 0x76, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x5f, 0x69, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x09, 0x48, 0x00, 0x52, 0x0a, 0x72, 0x65, 0x76, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x12,
	0x4b, 0x0a, 0x0d, 0x61, 0x6c, 0x69, 0x61, 0x73, 0x5f, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x24, 0x2e, 0x67, 0x72, 0x61, 0x66, 0x65, 0x61, 0x73,
	0x2e, 0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2e, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x2e,
	0x41, 0x6c, 0x69, 0x61, 0x73, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74, 0x48, 0x00, 0x52, 0x0c,
	0x61, 0x6c, 0x69, 0x61, 0x73, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74, 0x42, 0x0a, 0x0a, 0x08,
	0x72, 0x65, 0x76, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x22, 0x45, 0x0a, 0x10, 0x47, 0x69, 0x74, 0x53,
	0x6f, 0x75, 0x72, 0x63, 0x65, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74, 0x12, 0x10, 0x0a, 0x03,
	0x75, 0x72, 0x6c, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x75, 0x72, 0x6c, 0x12, 0x1f,
	0x0a, 0x0b, 0x72, 0x65, 0x76, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x0a, 0x72, 0x65, 0x76, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x22,
	0x73, 0x0a, 0x06, 0x52, 0x65, 0x70, 0x6f, 0x49, 0x64, 0x12, 0x4f, 0x0a, 0x0f, 0x70, 0x72, 0x6f,
	0x6a, 0x65, 0x63, 0x74, 0x5f, 0x72, 0x65, 0x70, 0x6f, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x25, 0x2e, 0x67, 0x72, 0x61, 0x66, 0x65, 0x61, 0x73, 0x2e, 0x76, 0x31, 0x62,
	0x65, 0x74, 0x61, 0x31, 0x2e, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x2e, 0x50, 0x72, 0x6f, 0x6a,
	0x65, 0x63, 0x74, 0x52, 0x65, 0x70, 0x6f, 0x49, 0x64, 0x48, 0x00, 0x52, 0x0d, 0x70, 0x72, 0x6f,
	0x6a, 0x65, 0x63, 0x74, 0x52, 0x65, 0x70, 0x6f, 0x49, 0x64, 0x12, 0x12, 0x0a, 0x03, 0x75, 0x69,
	0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x03, 0x75, 0x69, 0x64, 0x42, 0x04,
	0x0a, 0x02, 0x69, 0x64, 0x22, 0x4b, 0x0a, 0x0d, 0x50, 0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74, 0x52,
	0x65, 0x70, 0x6f, 0x49, 0x64, 0x12, 0x1d, 0x0a, 0x0a, 0x70, 0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74,
	0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x70, 0x72, 0x6f, 0x6a, 0x65,
	0x63, 0x74, 0x49, 0x64, 0x12, 0x1b, 0x0a, 0x09, 0x72, 0x65, 0x70, 0x6f, 0x5f, 0x6e, 0x61, 0x6d,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x72, 0x65, 0x70, 0x6f, 0x4e, 0x61, 0x6d,
	0x65, 0x42, 0x69, 0x0a, 0x19, 0x69, 0x6f, 0x2e, 0x67, 0x72, 0x61, 0x66, 0x65, 0x61, 0x73, 0x2e,
	0x76, 0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2e, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x50, 0x01,
	0x5a, 0x44, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x72, 0x6f, 0x64,
	0x65, 0x2f, 0x72, 0x6f, 0x64, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x64, 0x65, 0x70, 0x73,
	0x2f, 0x67, 0x72, 0x61, 0x66, 0x65, 0x61, 0x73, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x76,
	0x31, 0x62, 0x65, 0x74, 0x61, 0x31, 0x2f, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x5f, 0x67, 0x6f,
	0x5f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0xa2, 0x02, 0x03, 0x47, 0x52, 0x41, 0x62, 0x06, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_proto_v1beta1_source_proto_rawDescOnce sync.Once
	file_proto_v1beta1_source_proto_rawDescData = file_proto_v1beta1_source_proto_rawDesc
)

func file_proto_v1beta1_source_proto_rawDescGZIP() []byte {
	file_proto_v1beta1_source_proto_rawDescOnce.Do(func() {
		file_proto_v1beta1_source_proto_rawDescData = protoimpl.X.CompressGZIP(file_proto_v1beta1_source_proto_rawDescData)
	})
	return file_proto_v1beta1_source_proto_rawDescData
}

var file_proto_v1beta1_source_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_proto_v1beta1_source_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_proto_v1beta1_source_proto_goTypes = []interface{}{
	(AliasContext_Kind)(0),         // 0: grafeas.v1beta1.source.AliasContext.Kind
	(*SourceContext)(nil),          // 1: grafeas.v1beta1.source.SourceContext
	(*AliasContext)(nil),           // 2: grafeas.v1beta1.source.AliasContext
	(*CloudRepoSourceContext)(nil), // 3: grafeas.v1beta1.source.CloudRepoSourceContext
	(*GerritSourceContext)(nil),    // 4: grafeas.v1beta1.source.GerritSourceContext
	(*GitSourceContext)(nil),       // 5: grafeas.v1beta1.source.GitSourceContext
	(*RepoId)(nil),                 // 6: grafeas.v1beta1.source.RepoId
	(*ProjectRepoId)(nil),          // 7: grafeas.v1beta1.source.ProjectRepoId
	nil,                            // 8: grafeas.v1beta1.source.SourceContext.LabelsEntry
}
var file_proto_v1beta1_source_proto_depIdxs = []int32{
	3, // 0: grafeas.v1beta1.source.SourceContext.cloud_repo:type_name -> grafeas.v1beta1.source.CloudRepoSourceContext
	4, // 1: grafeas.v1beta1.source.SourceContext.gerrit:type_name -> grafeas.v1beta1.source.GerritSourceContext
	5, // 2: grafeas.v1beta1.source.SourceContext.git:type_name -> grafeas.v1beta1.source.GitSourceContext
	8, // 3: grafeas.v1beta1.source.SourceContext.labels:type_name -> grafeas.v1beta1.source.SourceContext.LabelsEntry
	0, // 4: grafeas.v1beta1.source.AliasContext.kind:type_name -> grafeas.v1beta1.source.AliasContext.Kind
	6, // 5: grafeas.v1beta1.source.CloudRepoSourceContext.repo_id:type_name -> grafeas.v1beta1.source.RepoId
	2, // 6: grafeas.v1beta1.source.CloudRepoSourceContext.alias_context:type_name -> grafeas.v1beta1.source.AliasContext
	2, // 7: grafeas.v1beta1.source.GerritSourceContext.alias_context:type_name -> grafeas.v1beta1.source.AliasContext
	7, // 8: grafeas.v1beta1.source.RepoId.project_repo_id:type_name -> grafeas.v1beta1.source.ProjectRepoId
	9, // [9:9] is the sub-list for method output_type
	9, // [9:9] is the sub-list for method input_type
	9, // [9:9] is the sub-list for extension type_name
	9, // [9:9] is the sub-list for extension extendee
	0, // [0:9] is the sub-list for field type_name
}

func init() { file_proto_v1beta1_source_proto_init() }
func file_proto_v1beta1_source_proto_init() {
	if File_proto_v1beta1_source_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_proto_v1beta1_source_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SourceContext); i {
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
		file_proto_v1beta1_source_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AliasContext); i {
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
		file_proto_v1beta1_source_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CloudRepoSourceContext); i {
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
		file_proto_v1beta1_source_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GerritSourceContext); i {
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
		file_proto_v1beta1_source_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GitSourceContext); i {
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
		file_proto_v1beta1_source_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RepoId); i {
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
		file_proto_v1beta1_source_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ProjectRepoId); i {
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
	file_proto_v1beta1_source_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*SourceContext_CloudRepo)(nil),
		(*SourceContext_Gerrit)(nil),
		(*SourceContext_Git)(nil),
	}
	file_proto_v1beta1_source_proto_msgTypes[2].OneofWrappers = []interface{}{
		(*CloudRepoSourceContext_RevisionId)(nil),
		(*CloudRepoSourceContext_AliasContext)(nil),
	}
	file_proto_v1beta1_source_proto_msgTypes[3].OneofWrappers = []interface{}{
		(*GerritSourceContext_RevisionId)(nil),
		(*GerritSourceContext_AliasContext)(nil),
	}
	file_proto_v1beta1_source_proto_msgTypes[5].OneofWrappers = []interface{}{
		(*RepoId_ProjectRepoId)(nil),
		(*RepoId_Uid)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_proto_v1beta1_source_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proto_v1beta1_source_proto_goTypes,
		DependencyIndexes: file_proto_v1beta1_source_proto_depIdxs,
		EnumInfos:         file_proto_v1beta1_source_proto_enumTypes,
		MessageInfos:      file_proto_v1beta1_source_proto_msgTypes,
	}.Build()
	File_proto_v1beta1_source_proto = out.File
	file_proto_v1beta1_source_proto_rawDesc = nil
	file_proto_v1beta1_source_proto_goTypes = nil
	file_proto_v1beta1_source_proto_depIdxs = nil
}
