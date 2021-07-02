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
package auth

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/scylladb/go-set/strset"
)

var _ = Describe("RoleRegistry", func() {

	var registry = NewRoleRegistry()

	Context("GetRoleByName", func() {
		DescribeTable("Role mappings", func(roleName string, expectedRole Role) {
			actualRole := registry.GetRoleByName(roleName)

			Expect(actualRole).To(Equal(expectedRole))
		},
			Entry("Anonymous", "Anonymous", RoleAnonymous),
			Entry("Enforcer", "Enforcer", RoleEnforcer),
			Entry("Collector", "Collector", RoleCollector),
			Entry("Application Developer", "Application Developer", RoleApplicationDeveloper),
			Entry("Policy Developer", "Policy Developer", RolePolicyDeveloper),
			Entry("Policy Administrator", "Policy Administrator", RolePolicyAdministrator),
			Entry("Administrator", "Administrator", RoleAdministrator),
		)

		When("the role does not exist", func() {
			var actualRole Role
			BeforeEach(func() {
				actualRole = registry.GetRoleByName(fake.Word())
			})

			It("should return an empty string", func() {
				Expect(actualRole).To(BeEmpty())
			})
		})
	})

	Context("GetRolePermissions", func() {
		DescribeTable("role to permission mapping", func(role Role) {
			permissions := registry.GetRolePermissions(role)
			permissionSet := createPermissionSet(permissions)

			// each role should have at least one permission
			Expect(permissions).NotTo(BeEmpty())
			// no role should contain the same permission more than once
			Expect(permissionSet.Size()).To(Equal(len(permissions)))
		},
			Entry("Anonymous", RoleAnonymous),
			Entry("Enforcer", RoleEnforcer),
			Entry("Collector", RoleCollector),
			Entry("Application Developer", RoleApplicationDeveloper),
			Entry("Policy Developer", RolePolicyDeveloper),
			Entry("Policy Administrator", RolePolicyAdministrator),
			Entry("Administrator", RoleAdministrator),
		)

		When("the Administrator role is requested", func() {
			It("should return all roles", func() {
				Expect(registry.GetRolePermissions(RoleAdministrator)).To(HaveLen(18))
			})
		})

		When("the Anonymous role is requested", func() {
			It("should return read-only permissions", func() {
				permissions := registry.GetRolePermissions(RoleAnonymous)

				for _, permission := range permissions {
					Expect(string(permission)).To(HaveSuffix(".read"))
				}
			})
		})
	})
})

func createPermissionSet(rolePermissions []Permission) *strset.Set {
	var permissions []string
	for _, p := range rolePermissions {
		permissions = append(permissions, string(p))
	}
	return strset.New(permissions...)
}
