package compiler

import "testing"

// TestAddModuleI18nPrefix_PreservesQuotes is a regression test: the prefixing
// regex used to match the opening quote (['"]?) but not include it back in
// the replacement, dropping it - turning lx.i18n('monday') into the
// syntactically broken lx.i18n(module-Calendar-monday'). FindMatchingBrace
// would then treat that orphaned closing quote as if it opened a new string,
// swallowing everything up to the next lx.i18n(...) call's own quote -
// merging two calls (and losing the second key entirely) instead of
// prefixing each one independently.
func TestAddModuleI18nPrefix_PreservesQuotes(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{
			"single_quoted",
			`lx.i18n('monday')`,
			`lx.i18n('module-Calendar-monday')`,
		},
		{
			"double_quoted",
			`lx.i18n("monday")`,
			`lx.i18n("module-Calendar-monday")`,
		},
		{
			"multiple_calls_on_one_line",
			`[lx.i18n('monday'), lx.i18n('tuesday'), lx.i18n('wednesday')]`,
			`[lx.i18n('module-Calendar-monday'), lx.i18n('module-Calendar-tuesday'), lx.i18n('module-Calendar-wednesday')]`,
		},
		{
			"no_quote",
			`lx.i18n(key)`,
			`lx.i18n(module-Calendar-key)`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := addModuleI18nPrefix(tc.in, "Calendar")
			if got != tc.want {
				t.Fatalf("addModuleI18nPrefix(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
