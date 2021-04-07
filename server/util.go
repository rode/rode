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

package server

import (
	"fmt"
	"regexp"
)

var (
	resourceUriPatterns = []*regexp.Regexp{
		// Docker images
		regexp.MustCompile("(?P<name>.+)(@sha256:)(?P<version>.+)"),
		// Git repositories
		regexp.MustCompile("^(git:/{2})(?P<name>.+)@(?P<version>.+)"),
		// Maven packages
		regexp.MustCompile("^(gav:/{2})(?P<name>.+):(?P<version>.+)"),
		// Files
		regexp.MustCompile("^(file:/{2}sha256:)(?P<version>.+):(?P<name>.+)"),
		// NPM packages
		regexp.MustCompile("^(npm:/{2})(?P<name>.+):(?P<version>.+)"),
		// NuGet packages
		regexp.MustCompile("^(nuget:/{2})(?P<name>.+):(?P<version>.+)"),
		// pip packages
		regexp.MustCompile("^(pip:/{2})(?P<name>.+):(?P<version>.+)"),
		// Debian packages
		regexp.MustCompile("^(deb:/{2}).*:(?P<name>.+):(?P<version>.+)"),
		// RPM packages
		regexp.MustCompile("^(rpm:/{2}).*:(?P<name>.+):(?P<version>.+)"),
	}
)

type resourceUriComponents struct {
	name    string
	version string
}

func parseResourceUri(uri string) (*resourceUriComponents, error) {
	var resourceRegex *regexp.Regexp

	for _, pattern := range resourceUriPatterns {
		if pattern.MatchString(uri) {
			resourceRegex = pattern
			break
		}
	}

	if resourceRegex == nil {
		return nil, fmt.Errorf("unable to determine resource type for uri: %s", uri)
	}

	matches := resourceRegex.FindStringSubmatch(uri)
	name := matches[resourceRegex.SubexpIndex("name")]
	version := matches[resourceRegex.SubexpIndex("version")]

	return &resourceUriComponents{name, version}, nil
}
