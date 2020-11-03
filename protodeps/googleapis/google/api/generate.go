// This fetches protocol buffer definition dependencies from the main googleapis repository

//go:generate curl --silent -LO https://raw.githubusercontent.com/googleapis/googleapis/fb6fa4cfb16917da8dc5d23c2494d422dd3e9cd4/google/api/annotations.proto
//go:generate curl --silent -LO https://raw.githubusercontent.com/googleapis/googleapis/fb6fa4cfb16917da8dc5d23c2494d422dd3e9cd4/google/api/client.proto
//go:generate curl --silent -LO https://raw.githubusercontent.com/googleapis/googleapis/fb6fa4cfb16917da8dc5d23c2494d422dd3e9cd4/google/api/field_behavior.proto
//go:generate curl --silent -LO https://raw.githubusercontent.com/googleapis/googleapis/fb6fa4cfb16917da8dc5d23c2494d422dd3e9cd4/google/api/http.proto
//go:generate curl --silent -LO https://raw.githubusercontent.com/googleapis/googleapis/fb6fa4cfb16917da8dc5d23c2494d422dd3e9cd4/google/api/resource.proto

package api
