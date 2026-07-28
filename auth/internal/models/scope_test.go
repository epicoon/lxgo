package models_test

import (
	"testing"

	"github.com/epicoon/lxgo/auth/internal/models"
)

func TestValidateScope(t *testing.T) {
	cases := map[string]bool{
		models.SCOPE_PROFILE:      true,
		models.SCOPE_PROFILE_DATA: true,
		"":                        false,
		"nonsense":                false,
	}
	for scope, want := range cases {
		if got := models.ValidateScope(scope); got != want {
			t.Errorf("ValidateScope(%q) = %v, want %v", scope, got, want)
		}
	}
}

func TestScopeIncludes(t *testing.T) {
	cases := []struct {
		granted, requested string
		want               bool
	}{
		{models.SCOPE_PROFILE, models.SCOPE_PROFILE, true},
		{models.SCOPE_PROFILE_DATA, models.SCOPE_PROFILE_DATA, true},
		{models.SCOPE_PROFILE_DATA, models.SCOPE_PROFILE, true},  // narrowing is fine
		{models.SCOPE_PROFILE, models.SCOPE_PROFILE_DATA, false}, // broadening is not
		{"nonsense", models.SCOPE_PROFILE, false},
		{models.SCOPE_PROFILE, "nonsense", false},
	}
	for _, c := range cases {
		if got := models.ScopeIncludes(c.granted, c.requested); got != c.want {
			t.Errorf("ScopeIncludes(%q, %q) = %v, want %v", c.granted, c.requested, got, c.want)
		}
	}
}
