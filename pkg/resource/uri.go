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
	"fmt"
	pb "github.com/rode/rode/proto/v1alpha1"
	"regexp"
)

var (
	uriPrefixes = map[pb.ResourceType]string{
		pb.ResourceType_DOCKER: "",
		pb.ResourceType_GIT:    "git://",
		pb.ResourceType_MAVEN:  "gav://",
		pb.ResourceType_FILE:   "file://",
		pb.ResourceType_NPM:    "npm://",
		pb.ResourceType_NUGET:  "nuget://",
		pb.ResourceType_PIP:    "pip://",
		pb.ResourceType_DEBIAN: "deb://",
		pb.ResourceType_RPM:    "rpm://",
	}
	uriPatterns = map[pb.ResourceType]*regexp.Regexp{
		pb.ResourceType_DOCKER: regexp.MustCompile("(?P<name>.+)(@sha256:)(?P<version>.+)"),
		pb.ResourceType_GIT:    regexp.MustCompile("^git:/{2}(?P<name>.+)@(?P<version>.+)"),
		pb.ResourceType_MAVEN:  regexp.MustCompile("^gav:/{2}(?P<name>.+):(?P<version>.+)"),
		pb.ResourceType_FILE:   regexp.MustCompile("^file:/{2}sha256:(?P<version>.+):(?P<name>.+)"),
		pb.ResourceType_NPM:    regexp.MustCompile("^npm:/{2}(?P<name>.+):(?P<version>.+)"),
		pb.ResourceType_NUGET:  regexp.MustCompile("^nuget:/{2}(?P<name>.+):(?P<version>.+)"),
		pb.ResourceType_PIP:    regexp.MustCompile("^pip:/{2}(?P<name>.+):(?P<version>.+)"),
		pb.ResourceType_DEBIAN: regexp.MustCompile("^deb:/{2}.*:(?P<name>.+):(?P<version>.+)"),
		pb.ResourceType_RPM:    regexp.MustCompile("^rpm:/{2}.*:(?P<name>.+):(?P<version>.+)"),
	}
)

type uriComponents struct {
	name         string
	version      string
	resourceType pb.ResourceType
	prefixedName string
}

func parseResourceUri(uri string) (*uriComponents, error) {
	var (
		resourceRegex *regexp.Regexp
		resourceType  pb.ResourceType
	)

	for t, pattern := range uriPatterns {
		if pattern.MatchString(uri) {
			resourceRegex = pattern
			resourceType = t
			break
		}
	}

	if resourceRegex == nil {
		return nil, fmt.Errorf("unable to determine resource type for uri: %s", uri)
	}

	matches := resourceRegex.FindStringSubmatch(uri)
	name := matches[resourceRegex.SubexpIndex("name")]
	version := matches[resourceRegex.SubexpIndex("version")]
	prefixedName := uriPrefixes[resourceType] + name

	return &uriComponents{name, version, resourceType, prefixedName}, nil
}
