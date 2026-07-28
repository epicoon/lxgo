package handlers

import "testing"

func TestValidateLogin(t *testing.T) {
	cases := map[string]bool{
		"ab":                    false, // too short
		"abc":                   true,
		"a23456789012345678901": false, // too long (21 chars)
		"valid_login.1":         true,
		"1invalid":              false, // must start with a letter
		"has..double":           false,
		"has__double":           false,
		"has.single_ok":         true,
	}
	for login, want := range cases {
		if got := validateLogin(login); got != want {
			t.Errorf("validateLogin(%q) = %v, want %v", login, got, want)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	cases := map[string]bool{
		"short1!":         false, // too short
		"nouppercase1!":   false,
		"NOLOWERCASE1!":   false,
		"NoDigitsHere!":   false,
		"NoSpecialChar1":  false,
		"Valid1Password!": true,
	}
	for password, want := range cases {
		if got := validatePassword(password); got != want {
			t.Errorf("validatePassword(%q) = %v, want %v", password, got, want)
		}
	}
}
