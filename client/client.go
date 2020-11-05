package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/grafeas/grafeas/proto/v1beta1/common_go_proto"
	"github.com/grafeas/grafeas/proto/v1beta1/grafeas_go_proto"
	"github.com/grafeas/grafeas/proto/v1beta1/package_go_proto"
	"github.com/grafeas/grafeas/proto/v1beta1/vulnerability_go_proto"
	pb "github.com/liatrio/rode-api/proto/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	address = "localhost:50051"
)

func main() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewRodeClient(conn)

	// Contact the server and print out its response.
	// _ := defaultName
	// if len(os.Args) > 1 {
	// 	name = os.Args[1]
	// }
	occurrence := &grafeas_go_proto.Occurrence{
		Name: "abc",
		Resource: &grafeas_go_proto.Resource{
			Name: "testResource",
			Uri:  "test",
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	response, err := c.BatchCreateOccurrences(ctx, &grafeas_go_proto.BatchCreateOccurrencesRequest{
		Occurrences: []*grafeas_go_proto.Occurrence{occurrence},
		Parent:      "projects/test123",
	})
	if err != nil {
		log.Fatalf("could not create occurrence: %v", err)
	}
	fmt.Printf("%#v\n", response)
}
