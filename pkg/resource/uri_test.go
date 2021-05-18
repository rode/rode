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

package resource

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	pb "github.com/rode/rode/proto/v1alpha1"

	. "github.com/onsi/gomega"
)

var _ = Describe("uri", func() {
	Describe("parseResourceUri", func() {
		DescribeTable("valid resource types", func(resourceUri string, expected *uriComponents) {
			actual, err := parseResourceUri(resourceUri)

			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(Equal(expected))
		},
			Entry("Docker image", "https://gcr.io/scanning-customer/dockerimage@sha256:244fd47e07d1004f0aed9c156aa09083c82bf8944eceb67c946ff7430510a77b", &uriComponents{
				name:         "https://gcr.io/scanning-customer/dockerimage",
				version:      "244fd47e07d1004f0aed9c156aa09083c82bf8944eceb67c946ff7430510a77b",
				resourceType: pb.ResourceType_DOCKER,
			}),
			Entry("Debian package", "deb://lucid:i386:acl:2.2.49-2", &uriComponents{
				name:         "acl",
				version:      "2.2.49-2",
				resourceType: pb.ResourceType_DEBIAN,
			}),
			Entry("Debian package without a specified distribution", "deb://arm64:build-essential:12.9", &uriComponents{
				name:         "build-essential",
				version:      "12.9",
				resourceType: pb.ResourceType_DEBIAN,
			}),
			Entry("Generic file", "file://sha256:244fd47e07d1004f0aed9c156aa09083c82bf8944eceb67c946ff7430510a77b:foo.jar", &uriComponents{
				name:         "foo.jar",
				version:      "244fd47e07d1004f0aed9c156aa09083c82bf8944eceb67c946ff7430510a77b",
				resourceType: pb.ResourceType_FILE,
			}),
			Entry("Maven package", "gav://ant:ant:1.6.5", &uriComponents{
				name:         "ant:ant",
				version:      "1.6.5",
				resourceType: pb.ResourceType_MAVEN,
			}),
			Entry("npm package", "npm://mocha:2.4.5", &uriComponents{
				name:         "mocha",
				version:      "2.4.5",
				resourceType: pb.ResourceType_NPM,
			}),
			Entry("scoped npm package", "npm://@babel/core:7.13.14", &uriComponents{
				name:         "@babel/core",
				version:      "7.13.14",
				resourceType: pb.ResourceType_NPM,
			}),
			Entry("NuGet", "nuget://log4net:9.0.1", &uriComponents{
				name:         "log4net",
				version:      "9.0.1",
				resourceType: pb.ResourceType_NUGET,
			}),
			Entry("pip package", "pip://raven:5.13.0", &uriComponents{
				name:         "raven",
				version:      "5.13.0",
				resourceType: pb.ResourceType_PIP,
			}),
			Entry("RPM package", "rpm://el6:i386:ImageMagick:6.7.2.7-4", &uriComponents{
				name:         "ImageMagick",
				version:      "6.7.2.7-4",
				resourceType: pb.ResourceType_RPM,
			}),
			Entry("RPM without a specified distribution", "rpm://i386:ImageMagick:6.7.2.7-4", &uriComponents{
				name:         "ImageMagick",
				version:      "6.7.2.7-4",
				resourceType: pb.ResourceType_RPM,
			}),
			Entry("Git repository (GitHub)", "git://github.com/rode/rode@bca0e1b89be42a61131b6de09fd2836e7b00c252", &uriComponents{
				name:         "github.com/rode/rode",
				version:      "bca0e1b89be42a61131b6de09fd2836e7b00c252",
				resourceType: pb.ResourceType_GIT,
			}),
			Entry("Git repository (Azure DevOps)", "git://dev.azure.com/rode/rode/_git/rode@bca0e1b89be42a61131b6de09fd2836e7b00c252", &uriComponents{
				name:         "dev.azure.com/rode/rode/_git/rode",
				version:      "bca0e1b89be42a61131b6de09fd2836e7b00c252",
				resourceType: pb.ResourceType_GIT,
			}),
		)

		Describe("invalid resource uris", func() {
			When("a resource uri contains an unexpected type", func() {
				It("should return an error", func() {
					invalidUri := "foo://bar"
					actual, err := parseResourceUri(invalidUri)

					Expect(actual).To(BeNil())
					Expect(err).To(MatchError("unable to determine resource type for uri: " + invalidUri))
				})
			})
		})
	})
})
