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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/rode/rode/proto/v1alpha1"
	grafeas_proto "github.com/rode/rode/protodeps/grafeas/proto/v1beta1/grafeas_go_proto"
	. "github.com/rode/rode/test/util"
	"google.golang.org/grpc/codes"
)

var _ = Describe("Collectors", func() {
	var (
		ctx = context.Background()
	)

	Describe("When registering a collector", func() {
		When("the request is valid", func() {
			It("should register successfully", func() {
				response, err := rode.RegisterCollector(ctx, &v1alpha1.RegisterCollectorRequest{
					Id: fake.LetterN(15),
					Notes: []*grafeas_proto.Note{
						randomDiscoveryNote(),
					},
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(response.Notes).To(HaveLen(1))
			})
		})

		When("the collector id is missing from the request", func() {
			It("should return an error", func() {
				_, err := rode.RegisterCollector(ctx, &v1alpha1.RegisterCollectorRequest{
					Notes: []*grafeas_proto.Note{
						randomDiscoveryNote(),
					},
				})

				Expect(err).To(HaveGrpcStatus(codes.InvalidArgument))
			})
		})

		When("a collector tries to register multiple notes of the same type", func() {
			It("should return an error", func() {
				_, err := rode.RegisterCollector(ctx, &v1alpha1.RegisterCollectorRequest{
					Id: fake.LetterN(15),
					Notes: []*grafeas_proto.Note{
						randomDiscoveryNote(),
						randomDiscoveryNote(),
					},
				})

				Expect(err).To(HaveGrpcStatus(codes.InvalidArgument))
			})
		})

		When("no notes are included", func() {
			It("should return without an error", func() {
				_, err := rode.RegisterCollector(ctx, &v1alpha1.RegisterCollectorRequest{
					Id: fake.LetterN(15),
				})

				Expect(err).NotTo(HaveOccurred())
			})
		})

		DescribeTable("authorization", func(entry *AuthzTestEntry) {
			_, err := rode.WithRole(entry.Role).RegisterCollector(ctx, &v1alpha1.RegisterCollectorRequest{
				Id: fake.LetterN(15),
				Notes: []*grafeas_proto.Note{
					randomDiscoveryNote(),
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
})
