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

package mocks

//go:generate counterfeiter -o grafeas_client.go github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto.GrafeasV1Beta1Client
//go:generate counterfeiter -o grafeas_projects_client.go github.com/rode/rode/protodeps/grafeas/proto/v1beta1/project_go_proto.ProjectsClient
