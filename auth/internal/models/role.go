package models

import "slices"

const (
	ROLE_SUPERADMIN = 1 + iota
	ROLE_ADMIN
	ROLE_VIEWER
	ROLE_CLIENT
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
		ROLE_VIEWER,
		ROLE_CLIENT,
	}, code)
}

func ValidateInitiator(initiatorCode int, newCode int) bool {
	switch initiatorCode {
	case ROLE_SUPERADMIN:
		return ValidateRoleCode(newCode)
	case ROLE_ADMIN:
		return slices.Contains([]int{
			ROLE_VIEWER,
			ROLE_CLIENT,
		}, newCode)
	default:
		return false
	}
}
