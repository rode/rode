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

package v1alpha1_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/rode/rode/proto/v1alpha1"
	build_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/build_go_proto"
	common_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/common_go_proto"
	discovery_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/discovery_go_proto"
	grafeas_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	provenance_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/provenance_go_proto"
	source_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/source_go_proto"
	. "github.com/rode/rode/test/util"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
)

var _ = Describe("Grafeas", func() {
	var ctx = context.Background()

	Describe("Creating occurrences", func() {
		When("the occurrences are valid", func() {
			It("should create the occurrences in Grafeas", func() {
				expectedNumberOfOccurrences := fake.Number(2, 5)
				var occurrences []*grafeas_proto.Occurrence
				for i := 0; i < expectedNumberOfOccurrences; i++ {
					occurrences = append(occurrences, randomBuildOccurrence())
				}

				actualOccurrences, err := rode.BatchCreateOccurrences(ctx, &v1alpha1.BatchCreateOccurrencesRequest{
					Occurrences: occurrences,
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(actualOccurrences.Occurrences).To(HaveLen(expectedNumberOfOccurrences))

				filteredOccurrences, err := rode.ListOccurrences(ctx, &v1alpha1.ListOccurrencesRequest{
					Filter: selectOccurrencesFilter(actualOccurrences.Occurrences...),
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(filteredOccurrences.Occurrences).To(HaveLen(expectedNumberOfOccurrences))
			})
		})

		DescribeTable("authorization", func(entry *AuthzTestEntry) {
			_, err := rode.WithRole(entry.Role).BatchCreateOccurrences(ctx, &v1alpha1.BatchCreateOccurrencesRequest{
				Occurrences: []*grafeas_proto.Occurrence{
					randomBuildOccurrence(),
				},
			})

			if entry.Permitted {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(HaveGrpcStatus(codes.PermissionDenied))
			}
		},
			NewAuthzTableTest("Administrator", "Collector")...,
		)
	})

	Describe("Updating an occurrence", func() {
		var occurrence *grafeas_proto.Occurrence

		BeforeEach(func() {
			occurrences, err := rode.BatchCreateOccurrences(ctx, &v1alpha1.BatchCreateOccurrencesRequest{
				Occurrences: []*grafeas_proto.Occurrence{randomBuildOccurrence()},
			})
			Expect(err).NotTo(HaveOccurred())

			occurrence = occurrences.Occurrences[0]
		})

		When("the update is valid", func() {
			It("should update the occurrence in Grafeas", func() {
				newArtifact := &provenance_proto.Artifact{
					Id: randomContainerImageUri(),
					Names: []string{
						fmt.Sprintf("%s:%s", fake.Word(), fake.Word()),
					},
				}
				occurrence.GetBuild().Provenance.BuiltArtifacts = append(occurrence.GetBuild().Provenance.BuiltArtifacts, newArtifact)

				_, err := rode.UpdateOccurrence(ctx, &v1alpha1.UpdateOccurrenceRequest{
					Occurrence: occurrence,
					Id:         extractOccurrenceIdFromName(occurrence.Name),
					UpdateMask: &field_mask.FieldMask{
						Paths: []string{"details.build.provenance.built_artifacts"},
					},
				})
				Expect(err).NotTo(HaveOccurred())

				filteredOccurrences, err := rode.ListOccurrences(ctx, &v1alpha1.ListOccurrencesRequest{
					Filter: selectOccurrencesFilter(occurrence),
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(filteredOccurrences.Occurrences).To(HaveLen(1))
				Expect(filteredOccurrences.Occurrences[0].GetBuild().Provenance.BuiltArtifacts).To(HaveLen(2))
			})
		})

		DescribeTable("authorization", func(entry *AuthzTestEntry) {
			occurrence.Remediation = fake.Word()
			_, err := rode.WithRole(entry.Role).UpdateOccurrence(ctx, &v1alpha1.UpdateOccurrenceRequest{
				Occurrence: occurrence,
				Id:         extractOccurrenceIdFromName(occurrence.Name),
				UpdateMask: &field_mask.FieldMask{
					Paths: []string{"remediation"},
				},
			})

			if entry.Permitted {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(HaveGrpcStatus(codes.PermissionDenied))
			}
		},
			NewAuthzTableTest("Administrator", "Collector")...,
		)
	})

	Describe("Creating notes", func() {
		When("the note is valid", func() {
			It("should create the note in Grafeas", func() {
				_, err := rode.CreateNote(ctx, &v1alpha1.CreateNoteRequest{
					NoteId: fake.LetterN(10),
					Note:   randomDiscoveryNote(),
				})

				Expect(err).NotTo(HaveOccurred())
			})
		})

		DescribeTable("authorization", func(entry *AuthzTestEntry) {
			_, err := rode.WithRole(entry.Role).CreateNote(ctx, &v1alpha1.CreateNoteRequest{
				NoteId: fake.LetterN(10),
				Note:   randomDiscoveryNote(),
			})

			if entry.Permitted {
				Expect(err).NotTo(HaveOccurred())
			} else {
				Expect(err).To(HaveGrpcStatus(codes.PermissionDenied))
			}
		},
			NewAuthzTableTest("Administrator", "Collector")...,
		)
	})
})

func randomBuildOccurrence() *grafeas_proto.Occurrence {
	return &grafeas_proto.Occurrence{
		Name: fake.LetterN(10),
		Resource: &grafeas_proto.Resource{
			Uri: randomGitUri(),
		},
		NoteName: fmt.Sprintf("projects/rode/notes/%s", fake.LetterN(15)),
		Kind:     common_proto.NoteKind_BUILD,
		Details: &grafeas_proto.Occurrence_Build{
			Build: &build_proto.Details{
				Provenance: &provenance_proto.BuildProvenance{
					Id:        fake.UUID(),
					ProjectId: fake.LetterN(10),
					BuiltArtifacts: []*provenance_proto.Artifact{
						{
							Id: randomContainerImageUri(),
							Names: []string{
								fmt.Sprintf("%s:%s", fake.Word(), fake.Word()),
							},
						},
					},
					SourceProvenance: &provenance_proto.Source{
						Context: &source_proto.SourceContext{
							Context: &source_proto.SourceContext_Git{
								Git: &source_proto.GitSourceContext{},
							},
						},
					},
				},
			},
		},
	}
}

func randomDiscoveryNote() *grafeas_proto.Note {
	return &grafeas_proto.Note{
		Name:             fake.LetterN(10),
		ShortDescription: fake.Word(),
		LongDescription:  fake.Word(),
		Kind:             common_proto.NoteKind_DISCOVERY,
		Type: &grafeas_proto.Note_Discovery{
			Discovery: &discovery_proto.Discovery{
				AnalysisKind: common_proto.NoteKind_VULNERABILITY,
			},
		},
	}
}

func selectOccurrencesFilter(occurrences ...*grafeas_proto.Occurrence) string {
	var occurrenceNamesFilter []string
	for i := 0; i < len(occurrences); i++ {
		nameFilter := fmt.Sprintf("name == '%s'", occurrences[i].Name)
		occurrenceNamesFilter = append(occurrenceNamesFilter, nameFilter)
	}

	return strings.Join(occurrenceNamesFilter, " || ")
}

func randomContainerImageUri() string {
	return fmt.Sprintf("%s/%s@sha256:%s", fake.Word(), fake.Word(), randomHex(64))
}

func randomGitUri() string {
	gitHost := fake.DomainName()
	owner := fake.Word()
	repo := fake.Word()
	commit := randomHex(40)

	return fmt.Sprintf("git://%s/%s/%s@%s", gitHost, owner, repo, commit)
}

func randomHex(length int) string {
	hexBytes := make([]byte, length/2)
	fake.Rand.Read(hexBytes)

	return hex.EncodeToString(hexBytes)
}

func extractOccurrenceIdFromName(occurrenceName string) string {
	pieces := strings.Split(occurrenceName, "/")

	return pieces[len(pieces)-1]
}
