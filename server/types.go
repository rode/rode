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

import (
	"encoding/json"
	"github.com/rode/grafeas-elasticsearch/go/v1beta1/storage/filtering"
)

// These Elasticsearch types were originally defined in rode/grafeas-elasticsearch; however,
// referencing them directly had the adverse side effect of protobuf namespace conflicts with the
// vendored Grafeas protobufs in Rode. As a result they're copied here
// original types: https://github.com/rode/grafeas-elasticsearch/blob/624ccb5d038b55d90fb7c6b3b5378125d7ad0aa5/go/v1beta1/storage/types.go#L32
type esCollapse struct {
	Field string `json:"field,omitempty"`
}

type esSearch struct {
	Query    *filtering.Query `json:"query,omitempty"`
	Collapse *esCollapse      `json:"collapse,omitempty"`
}

type esSearchResponse struct {
	Took int                   `json:"took"`
	Hits *esSearchResponseHits `json:"hits"`
}

type esSearchResponseHits struct {
	Total *esSearchResponseTotal `json:"total"`
	Hits  []*esSearchResponseHit `json:"hits"`
}

type esSearchResponseTotal struct {
	Value int `json:"value"`
}

type esSearchResponseHit struct {
	ID         string          `json:"_id"`
	Source     json.RawMessage `json:"_source"`
	Highlights json.RawMessage `json:"highlight"`
	Sort       []interface{}   `json:"sort"`
}

type esMGetResponse struct {
	ID         string          `json:"_id"`
	Found      bool            `json:"found"`
}
