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

// updating go package for each protobuf definition to point to rode-api repository instead of grafeas
// this is gross, sorry. also, this probably only works with OSX's version of sed
//go:generate sh -c "grep -rl 'github.com/grafeas/grafeas' ./*.proto | xargs sed -i '' 's+github.com/grafeas/grafeas+github.com/liatrio/rode-api/protodeps/grafeas+g'"

// compile everything
//go:generate -command protoc protoc -I . -I ../.. -I ../../../googleapis --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative

//go:generate protoc attestation.proto
//go:generate rm -rf attestation_go_proto
//go:generate mkdir attestation_go_proto
//go:generate mv attestation.pb.go attestation_go_proto

//go:generate protoc build.proto
//go:generate rm -rf build_go_proto
//go:generate mkdir build_go_proto
//go:generate mv build.pb.go build_go_proto

//go:generate protoc common.proto
//go:generate rm -rf common_go_proto
//go:generate mkdir common_go_proto
//go:generate mv common.pb.go common_go_proto

//go:generate protoc cvss.proto
//go:generate rm -rf cvss_go_proto
//go:generate mkdir cvss_go_proto
//go:generate mv cvss.pb.go cvss_go_proto

//go:generate protoc deployment.proto
//go:generate rm -rf deployment_go_proto
//go:generate mkdir deployment_go_proto
//go:generate mv deployment.pb.go deployment_go_proto

//go:generate protoc discovery.proto
//go:generate rm -rf discovery_go_proto
//go:generate mkdir discovery_go_proto
//go:generate mv discovery.pb.go discovery_go_proto

//go:generate protoc grafeas.proto
//go:generate rm -rf grafeas_go_proto
//go:generate mkdir grafeas_go_proto
//go:generate mv grafeas.pb.go grafeas_go_proto
//go:generate mv grafeas_grpc.pb.go grafeas_go_proto

//go:generate protoc image.proto
//go:generate rm -rf image_go_proto
//go:generate mkdir image_go_proto
//go:generate mv image.pb.go image_go_proto

//go:generate protoc intoto.proto
//go:generate rm -rf intoto_go_proto
//go:generate mkdir intoto_go_proto
//go:generate mv intoto.pb.go intoto_go_proto

//go:generate protoc package.proto
//go:generate rm -rf package_go_proto
//go:generate mkdir package_go_proto
//go:generate mv package.pb.go package_go_proto

//go:generate protoc provenance.proto
//go:generate rm -rf provenance_go_proto
//go:generate mkdir provenance_go_proto
//go:generate mv provenance.pb.go provenance_go_proto

//go:generate protoc source.proto
//go:generate rm -rf source_go_proto
//go:generate mkdir source_go_proto
//go:generate mv source.pb.go source_go_proto

//go:generate protoc vulnerability.proto
//go:generate rm -rf vulnerability_go_proto
//go:generate mkdir vulnerability_go_proto
//go:generate mv vulnerability.pb.go vulnerability_go_proto

package v1beta1
