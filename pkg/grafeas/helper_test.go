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

package grafeas

import (
	"context"
	"errors"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rode/rode/mocks"
	"github.com/rode/rode/pkg/constants"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/build_go_proto"
	grafeas_common_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/common_go_proto"
	grafeas_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	"github.com/rode/rode/protodeps/grafeas/proto/v1beta1/provenance_go_proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"strings"
)

var _ = Describe("grafeas helper", func() {
	var (
		ctx           context.Context
		grafeasClient *mocks.FakeGrafeasV1Beta1Client

		helper Helper
	)

	BeforeEach(func() {
		ctx = context.Background()
		grafeasClient = &mocks.FakeGrafeasV1Beta1Client{}

		helper = NewHelper(logger, grafeasClient)
	})

	Context("ListVersionedResourceOccurrences", func() {
		var (
			listBuildOccurrencesResponse *grafeas_proto.ListOccurrencesResponse
			listBuildOccurrencesError    error

			listAllOccurrencesResponse *grafeas_proto.ListOccurrencesResponse
			listAllOccurrencesError    error

			resourceUri    string
			gitResourceUri string

			nextPageToken    string
			currentPageToken string
			pageSize         int32

			actualOccurrences   []*grafeas_proto.Occurrence
			actualNextPageToken string
			actualError         error
		)

		BeforeEach(func() {
			resourceUri = fake.URL()
			nextPageToken = fake.Word()
			currentPageToken = fake.Word()
			pageSize = fake.Int32()

			gitResourceUri = fmt.Sprintf("git://%s", fake.DomainName())

			listBuildOccurrencesResponse = &grafeas_proto.ListOccurrencesResponse{
				Occurrences: []*grafeas_proto.Occurrence{
					{
						Resource: &grafeas_proto.Resource{
							Uri: gitResourceUri,
						},
						Kind: grafeas_common_proto.NoteKind_BUILD,
						Details: &grafeas_proto.Occurrence_Build{
							Build: &build_go_proto.Details{
								Provenance: &provenance_go_proto.BuildProvenance{
									BuiltArtifacts: []*provenance_go_proto.Artifact{
										{
											Id: resourceUri,
										},
									},
								},
							},
						},
					},
				},
			}
			listBuildOccurrencesError = nil

			occurrences := []*grafeas_proto.Occurrence{
				createRandomOccurrence(grafeas_common_proto.NoteKind_VULNERABILITY),
				createRandomOccurrence(grafeas_common_proto.NoteKind_BUILD),
			}

			listAllOccurrencesResponse = &grafeas_proto.ListOccurrencesResponse{
				Occurrences:   occurrences,
				NextPageToken: nextPageToken,
			}
			listAllOccurrencesError = nil
		})

		JustBeforeEach(func() {
			grafeasClient.ListOccurrencesReturnsOnCall(0, listBuildOccurrencesResponse, listBuildOccurrencesError)
			grafeasClient.ListOccurrencesReturnsOnCall(1, listAllOccurrencesResponse, listAllOccurrencesError)

			actualOccurrences, actualNextPageToken, actualError = helper.ListVersionedResourceOccurrences(ctx, resourceUri, currentPageToken, pageSize)
		})

		It("should list build occurrences for the resource uri", func() {
			_, buildOccurrencesRequest, _ := grafeasClient.ListOccurrencesArgsForCall(0)

			Expect(buildOccurrencesRequest).NotTo(BeNil())
			Expect(buildOccurrencesRequest.Parent).To(Equal(constants.RodeProjectSlug))
			Expect(buildOccurrencesRequest.Filter).To(ContainSubstring(fmt.Sprintf(`build.provenance.builtArtifacts.nestedFilter(id == "%s")`, resourceUri)))
			Expect(buildOccurrencesRequest.Filter).To(ContainSubstring(fmt.Sprintf(`resource.uri == "%s"`, resourceUri)))
			Expect(buildOccurrencesRequest.PageSize).To(Equal(int32(1000)))
		})

		It("should use the build occurrence to find all occurrences", func() {
			expectedFilter := []string{
				fmt.Sprintf(`resource.uri == "%s"`, resourceUri),
				fmt.Sprintf(`resource.uri == "%s"`, gitResourceUri),
			}

			_, allOccurrencesRequest, _ := grafeasClient.ListOccurrencesArgsForCall(1)

			Expect(allOccurrencesRequest).NotTo(BeNil())
			Expect(allOccurrencesRequest.Parent).To(Equal("projects/rode"))
			Expect(allOccurrencesRequest.PageSize).To(Equal(pageSize))
			Expect(allOccurrencesRequest.PageToken).To(Equal(currentPageToken))

			filterParts := strings.Split(allOccurrencesRequest.Filter, " || ")
			Expect(filterParts).To(ConsistOf(expectedFilter))
		})

		It("should return the occurrences and page token from the call to list all occurrences", func() {
			Expect(actualOccurrences).To(BeEquivalentTo(listAllOccurrencesResponse.Occurrences))
			Expect(actualNextPageToken).To(BeEquivalentTo(listAllOccurrencesResponse.NextPageToken))
			Expect(actualError).ToNot(HaveOccurred())
		})

		When("there are no build occurrences", func() {
			BeforeEach(func() {
				listBuildOccurrencesResponse.Occurrences = []*grafeas_proto.Occurrence{}
			})

			It("should list occurrences for the resource uri", func() {
				_, allOccurrencesRequest, _ := grafeasClient.ListOccurrencesArgsForCall(1)

				Expect(allOccurrencesRequest.Filter).To(Equal(fmt.Sprintf(`resource.uri == "%s"`, resourceUri)))
			})
		})

		When("an error occurs listing build occurrences", func() {
			BeforeEach(func() {
				listBuildOccurrencesError = errors.New("error listing build occurrences")
			})

			It("should return an error", func() {
				Expect(actualOccurrences).To(BeNil())
				Expect(actualNextPageToken).To(BeEmpty())
				Expect(actualError).To(HaveOccurred())
			})

			It("should not attempt to list all occurrences", func() {
				Expect(grafeasClient.ListOccurrencesCallCount()).To(Equal(1))
			})
		})

		When("an error occurs listing all occurrences", func() {
			BeforeEach(func() {
				listAllOccurrencesError = errors.New("error listing all occurrences")
			})

			It("should return an error", func() {
				Expect(actualOccurrences).To(BeNil())
				Expect(actualNextPageToken).To(BeEmpty())
				Expect(actualError).To(HaveOccurred())
			})
		})
	})
})

func createRandomOccurrence(kind grafeas_common_proto.NoteKind) *grafeas_proto.Occurrence {
	return &grafeas_proto.Occurrence{
		Name: fake.LetterN(10),
		Resource: &grafeas_proto.Resource{
			Uri: fmt.Sprintf("%s@sha256:%s", fake.URL(), fake.LetterN(10)),
		},
		NoteName:    fake.LetterN(10),
		Kind:        kind,
		Remediation: fake.LetterN(10),
		CreateTime:  timestamppb.New(fake.Date()),
		UpdateTime:  timestamppb.New(fake.Date()),
		Details:     nil,
	}
}
