package data

import (
	_ "embed"
)

var (
	//go:embed minimal_policy.rego
	MinimalPolicy string
	//go:embed updated_policy.rego
	UpdatedPolicy string
	//go:embed require_occurrences.rego
	RequireOccurrences string
)
