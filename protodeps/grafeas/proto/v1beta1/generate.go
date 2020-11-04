// This fetches protocol buffer definitions from the main Grafeas repository

//go:generate curl --silent -LO https://raw.githubusercontent.com/grafeas/grafeas/v0.1.6/proto/v1beta1/grafeas.proto
//go:generate curl --silent -LO https://raw.githubusercontent.com/grafeas/grafeas/v0.1.6/proto/v1beta1/vulnerability.proto
//go:generate curl --silent -LO https://raw.githubusercontent.com/grafeas/grafeas/v0.1.6/proto/v1beta1/build.proto
//go:generate curl --silent -LO https://raw.githubusercontent.com/grafeas/grafeas/v0.1.6/proto/v1beta1/image.proto
//go:generate curl --silent -LO https://raw.githubusercontent.com/grafeas/grafeas/v0.1.6/proto/v1beta1/package.proto
//go:generate curl --silent -LO https://raw.githubusercontent.com/grafeas/grafeas/v0.1.6/proto/v1beta1/deployment.proto
//go:generate curl --silent -LO https://raw.githubusercontent.com/grafeas/grafeas/v0.1.6/proto/v1beta1/attestation.proto
//go:generate curl --silent -LO https://raw.githubusercontent.com/grafeas/grafeas/v0.1.6/proto/v1beta1/intoto.proto
//go:generate curl --silent -LO https://raw.githubusercontent.com/grafeas/grafeas/v0.1.6/proto/v1beta1/common.proto
//go:generate curl --silent -LO https://raw.githubusercontent.com/grafeas/grafeas/v0.1.6/proto/v1beta1/provenance.proto
//go:generate curl --silent -LO https://raw.githubusercontent.com/grafeas/grafeas/v0.1.6/proto/v1beta1/source.proto
//go:generate curl --silent -LO https://raw.githubusercontent.com/grafeas/grafeas/v0.1.6/proto/v1beta1/discovery.proto
//go:generate curl --silent -LO https://raw.githubusercontent.com/grafeas/grafeas/v0.1.6/proto/v1beta1/cvss.proto

// run protoc here

package v1beta1
