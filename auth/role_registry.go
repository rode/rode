package auth

type Permission string
type Role string

var (
	RoleAnonymous            Role = "Anonymous"
	RoleEnforcer             Role = "Enforcer"
	RoleCollector            Role = "Collector"
	RoleApplicationDeveloper Role = "Application Developer"
	RolePolicyDeveloper      Role = "Policy Developer"
	RolePolicyAdministrator  Role = "Policy Administrator"
	RoleAdministrator        Role = "Administrator"

	PermissionCollectorRegister      Permission = "rode.collector.register"
	PermissionEvaluationResultRead   Permission = "rode.evaluationResult.read"
	PermissionOccurrenceRead         Permission = "rode.occurrence.read"
	PermissionOccurrenceWrite        Permission = "rode.occurrence.write"
	PermissionPolicyDelete           Permission = "rode.policy.delete"
	PermissionPolicyEvaluate         Permission = "rode.policy.evaluate"
	PermissionPolicyRead             Permission = "rode.policy.read"
	PermissionPolicyValidate         Permission = "rode.policy.validate"
	PermissionPolicyWrite            Permission = "rode.policy.write"
	PermissionPolicyAssignmentDelete Permission = "rode.policyAssignment.delete"
	PermissionPolicyAssignmentRead   Permission = "rode.policyAssignment.read"
	PermissionPolicyAssignmentWrite  Permission = "rode.policyAssignment.write"
	PermissionPolicyGroupDelete      Permission = "rode.policyGroup.delete"
	PermissionPolicyGroupRead        Permission = "rode.policyGroup.read"
	PermissionPolicyGroupWrite       Permission = "rode.policyGroup.write"
	PermissionResourceEvaluate       Permission = "rode.resource.evaluate"
	PermissionResourceRead           Permission = "rode.resource.read"
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
				PermissionEvaluationResultRead,
				PermissionOccurrenceRead,
				PermissionPolicyAssignmentRead,
				PermissionPolicyGroupRead,
				PermissionPolicyRead,
				PermissionResourceRead,
			},
			RoleEnforcer: {
				PermissionEvaluationResultRead,
				PermissionPolicyGroupRead,
				PermissionResourceEvaluate,
				PermissionResourceRead,
			},
			RoleCollector: {
				PermissionCollectorRegister,
				PermissionOccurrenceRead,
				PermissionOccurrenceWrite,
			},
			RoleApplicationDeveloper: {
				PermissionOccurrenceRead,
				PermissionPolicyAssignmentRead,
				PermissionPolicyEvaluate,
				PermissionPolicyGroupRead,
				PermissionPolicyRead,
				PermissionPolicyValidate,
				PermissionResourceRead,
			},
			RolePolicyDeveloper: {
				PermissionEvaluationResultRead,
				PermissionOccurrenceRead,
				PermissionPolicyAssignmentRead,
				PermissionPolicyEvaluate,
				PermissionPolicyGroupRead,
				PermissionPolicyRead,
				PermissionPolicyValidate,
				PermissionPolicyWrite,
				PermissionResourceEvaluate,
				PermissionResourceRead,
			},
			RolePolicyAdministrator: {
				PermissionEvaluationResultRead,
				PermissionOccurrenceRead,
				PermissionPolicyAssignmentDelete,
				PermissionPolicyAssignmentRead,
				PermissionPolicyAssignmentWrite,
				PermissionPolicyDelete,
				PermissionPolicyEvaluate,
				PermissionPolicyGroupDelete,
				PermissionPolicyGroupRead,
				PermissionPolicyGroupWrite,
				PermissionPolicyRead,
				PermissionPolicyValidate,
				PermissionPolicyWrite,
				PermissionResourceEvaluate,
				PermissionResourceRead,
			},
			RoleAdministrator: {
				PermissionEvaluationResultRead,
				PermissionOccurrenceRead,
				PermissionOccurrenceWrite,
				PermissionPolicyAssignmentDelete,
				PermissionPolicyAssignmentRead,
				PermissionPolicyAssignmentWrite,
				PermissionPolicyDelete,
				PermissionPolicyEvaluate,
				PermissionPolicyGroupDelete,
				PermissionPolicyGroupRead,
				PermissionPolicyGroupWrite,
				PermissionPolicyRead,
				PermissionPolicyValidate,
				PermissionPolicyWrite,
				PermissionResourceEvaluate,
				PermissionResourceRead,
				PermissionCollectorRegister,
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
