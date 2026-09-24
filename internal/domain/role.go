package domain

import "slices"

// Role codes. The roles table holds the same codes as FK targets;
// a new role needs a code constant here plus a seed row.
type Role string

const (
	RoleClient Role = "client"
	RoleMaster Role = "master"
	RoleAdmin  Role = "admin"
)

var roles = []Role{RoleClient, RoleMaster, RoleAdmin}

func ParseRole(s string) (Role, error) {
	return parseEnum("role", s, roles)
}

func (r Role) Valid() bool { return slices.Contains(roles, r) }
