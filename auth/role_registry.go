package auth

type Permission string
type Role string

var (
	RoleAnonymous           Role = "Anonymous"
	RolePolicyAdministrator Role = "Policy Administrator"
)

type RoleRegistry interface {
	GetRolePermissions(Role) []Permission
	GetRoleByName(string) Role
}

type roleRegistry struct {
	registry map[Role][]Permission
}

func NewRoleRegistry() RoleRegistry {
	return &roleRegistry{
		registry: map[Role][]Permission{
			RoleAnonymous: {
				"rode.occurrence.read",
				"rode.resource.read",
				"rode.policy.read",
				"rode.policyGroup.read",
				"rode.policyAssignment.read",
				"rode.evaluationResult.read",
			},
			RolePolicyAdministrator: {
				"rode.occurrence.read",

				"rode.resource.read",
				"rode.resource.evaluate",

				"rode.policy.read",
				"rode.policy.write",
				"rode.policy.delete",
				"rode.policy.evaluate",
				"rode.policy.validate",

				"rode.policyGroup.read",
				"rode.policyGroup.write",
				"rode.policyGroup.delete",

				"rode.policyAssignment.read",
				"rode.policyAssignment.write",
				"rode.policyAssignment.delete",

				"rode.evaluationResult.read",
			},
		},
	}
}

func (r *roleRegistry) GetRoleByName(roleName string) Role {
	role := Role(roleName)

	_, ok := r.registry[role]

	if !ok {
		return ""
	}

	return role
}

func (r *roleRegistry) GetRolePermissions(role Role) []Permission {
	return r.registry[role]
}
