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

package data

import (
	_ "embed"
)

var (
	//go:embed minimal_policy.rego
	MinimalPolicy string
	//go:embed updated_policy.rego
	UpdatedPolicy string
	//go:embed require_occurrences_policy.rego
	RequireOccurrencesPolicy string
	//go:embed no_vulnerabilities_policy.rego
	NoVulnerabilitiesPolicy string
)
