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
