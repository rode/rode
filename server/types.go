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

package server

// These Elasticsearch types were originally defined in rode/grafeas-elasticsearch; however,
// referencing them directly had the adverse side effect of protobuf namespace conflicts with the
// vendored Grafeas protobufs in Rode. As a result they're copied here
// original types: https://github.com/rode/grafeas-elasticsearch/blob/624ccb5d038b55d90fb7c6b3b5378125d7ad0aa5/go/v1beta1/storage/types.go#L32

type esBulkQueryFragment struct {
	Create *esBulkQueryCreateFragment `json:"create"`
}

type esBulkQueryCreateFragment struct {
	Id string `json:"_id"`
}

type esBulkResponse struct {
	Items  []*esBulkResponseActionItem `json:"items"`
	Errors bool                        `json:"errors"`
}

type esBulkResponseActionItem struct {
	Create *esBulkResponseItem `json:"create,omitempty"`
}

type esBulkResponseItem struct {
	Id      string                   `json:"_id"`
	Result  string                   `json:"result"`
	Version int                      `json:"_version"`
	Status  int                      `json:"status"`
	Error   *esBulkResponseItemError `json:"error,omitempty"`
}

type esBulkResponseItemError struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}
