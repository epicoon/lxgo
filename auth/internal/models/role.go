package models

import "slices"

// Roles apply to Admin, not to Client - a Client (OAuth relying party) has
// no role of its own, see .claude/tasks/0062.md.
const (
	ROLE_SUPERADMIN = 1 + iota
	ROLE_ADMIN
)

type Role struct {
	ID     uint
	Name   string
	Rights []Right
}

func ValidateRoleCode(code int) bool {
	return slices.Contains([]int{
		ROLE_SUPERADMIN,
		ROLE_ADMIN,
	}, code)
}

// ValidateInitiator reports whether an actor with initiatorCode may create/
// promote an admin with newCode. Only superadmin can do so for now - there's
// no "admin creates another admin" API yet, so admin -> nothing is the
// conservative default until that need is concrete.
func ValidateInitiator(initiatorCode int, newCode int) bool {
	switch initiatorCode {
	case ROLE_SUPERADMIN:
		return ValidateRoleCode(newCode)
	default:
		return false
	}
}
