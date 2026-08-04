------------------------------------------------------------------------------------------------------------------------
Date: 2026.08.05
Version: v0.1.0-alpha.9
Changes:
- add: interactive mode - a missing required parameter can be read from stdin instead of failing validation right
  away, either automatically via `ParamConfig.Interactive` (command author marks a parameter as inherently
  human-filled) or manually via `--interactive` on the call
- add: `ParamTypeEnum` - a parameter restricted to a fixed set of options (`TypeDetails`, or lazily via
  `FTypeDetails`; `ElemType` for a non-string element type), prompted with a `PromptSelect` picklist; `FTypeDetails`
  (e.g. a filesystem scan) only ever runs once a missing, required, interactive enum parameter is actually being
  prompted for - never for an already-supplied value
- add: `PromptString`/`PromptSelect` - new interactive primitives usable directly from an action's own code, not
  just through the automatic missing-parameter prompt above; `PromptSelect` is an arrow-key/Enter menu, falling back
  to a plain numbered prompt when stdin isn't a real terminal
- new dependency: `golang.org/x/term` (pinned to v0.34.0, matching `x/sys` v0.35.0, to keep `go 1.23.2` across every
  exported package) - needed for `PromptSelect`'s raw-mode terminal input

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.28
Version: v0.1.0-alpha.8
Changes:
- fix: typo in error messages - "undefind (default) command" -> "undefined (default) command"
- refactor: argument parsing split out into a standalone `parseArgs` so it's testable without touching `os.Args`
- test: added unit tests for `parseArgs`, `defineConstructor`, `GetOptions[T]`, `Prepare`, `validate`

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.25
Version: v0.1.0-alpha.7
Changes:
- rename: `FConstructor` → `CCommand` - matches the package's own `C`-prefix constructor-type convention
  (`CommandsList` is now `map[string]CCommand`)
- docs: Go-doc comments for every exported declaration in the package (`ICommand`/`Command`, `Config` and friends,
  `ICommandOptions`, `GetOptions`, `Prepare`, `Init`/`Run`) - previously undocumented

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.24
Version: v0.1.0-alpha.6
Changes:
- docs: README's example command typos fixed (`anonimus`→`anonymous`, `by`/`By`→`bye`/`Bye`)
- docs: new "cmd.ICommandOptions" section explaining the existing options-passing mechanism, with a real lxgo-jspp example

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.12
Version: v0.1.0-alpha.5
Changes:
- fix typos

------------------------------------------------------------------------------------------------------------------------
Date: 2025.10.24
Version: v0.1.0-alpha.4
Changes:
- support of one-letter flags

------------------------------------------------------------------------------------------------------------------------
Date: 2025.08.14
Version: v0.1.0-alpha.3
Changes:
- fix print info without config
- fix validation without config

------------------------------------------------------------------------------------------------------------------------
Date: 2025.08.13
Version: v0.1.0-alpha.2
Changes:
- add configuration for command: command description; params description, required, type, default value
- add auto validation for params

------------------------------------------------------------------------------------------------------------------------
Date: 2025.07.24
Version: v0.1.0-alpha.1
