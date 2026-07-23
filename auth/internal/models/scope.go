package models

const (
	SCOPE_PROFILE      = "profile"
	SCOPE_PROFILE_DATA = "profile:data"
)

const DefaultScope = SCOPE_PROFILE

// scopeLevel orders scopes from least to most access, so a "narrower or
// equal" check (see ScopeIncludes) can be done with a plain comparison
// instead of a full set-inclusion algorithm.
var scopeLevel = map[string]int{
	SCOPE_PROFILE:      1,
	SCOPE_PROFILE_DATA: 2,
}

func ValidateScope(scope string) bool {
	_, ok := scopeLevel[scope]
	return ok
}

// ScopeIncludes reports whether the granted scope covers the requested one,
// i.e. requested is granted itself or something narrower - used to reject
// scope broadening on /refresh (RFC 6749 §6).
func ScopeIncludes(granted, requested string) bool {
	g, gok := scopeLevel[granted]
	r, rok := scopeLevel[requested]
	return gok && rok && r <= g
}
