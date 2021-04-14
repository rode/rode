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

package util

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/rode/rode/proto/v1alpha1"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/common_go_proto"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/package_go_proto"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/vulnerability_go_proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	Address = "localhost:50051"
)

func CreateOccurrences(c pb.RodeClient) {
	occurrence := &grafeas_go_proto.Occurrence{
		Name: "abc",
		Resource: &grafeas_go_proto.Resource{
			Uri: "harbor.localhost/library/nginx@sha256:abc123",
		},
		NoteName:    "projects/abc/notes/123",
		Kind:        common_go_proto.NoteKind_VULNERABILITY,
		Remediation: "test",
		CreateTime:  timestamppb.Now(),
		Details: &grafeas_go_proto.Occurrence_Vulnerability{
			Vulnerability: &vulnerability_go_proto.Details{
				Type:             "test",
				Severity:         vulnerability_go_proto.Severity_CRITICAL,
				ShortDescription: "abc",
				LongDescription:  "abc123",
				RelatedUrls: []*common_go_proto.RelatedUrl{
					{
						Url:   "test",
						Label: "test",
					},
					{
						Url:   "test",
						Label: "test",
					},
				},
				EffectiveSeverity: vulnerability_go_proto.Severity_CRITICAL,
				PackageIssue: []*vulnerability_go_proto.PackageIssue{
					{
						SeverityName: "test",
						AffectedLocation: &vulnerability_go_proto.VulnerabilityLocation{
							CpeUri:  "test",
							Package: "test",
							Version: &package_go_proto.Version{
								Name:     "test",
								Revision: "test",
								Epoch:    35,
								Kind:     package_go_proto.Version_MINIMUM,
							},
						},
					},
				},
			},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
	defer cancel()
	response, err := c.BatchCreateOccurrences(ctx, &pb.BatchCreateOccurrencesRequest{
		Occurrences: []*grafeas_go_proto.Occurrence{occurrence},
	})
	if err != nil {
		log.Fatalf("could not create occurrence: %v", err)
	}
	fmt.Printf("%#v\n", response)
}
