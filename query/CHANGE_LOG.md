------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.28
Version: v0.1.0-alpha.5
Changes:
- fix: go.mod was missing its `require` block entirely (only the `module`/`go` directives) - the module could not
  be resolved/built standalone outside this monorepo's `go.work`

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.27
Version: v0.1.0-alpha.4
Changes:
- internal: `BaseRepo.CreateFromMap` now builds the entity via `lxgo-kernel`'s new `cast.MapToStruct` instead of the
  now-removed `conv.MapToStruct` - same behavior, no change to `CreateFromMap`'s signature or semantics

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.25
Version: v0.1.0-alpha.3
Changes:
- docs: Go-doc comments for every exported declaration in the package (`BaseRepo`/`IBaseRepo`, `QueryBuilder`/
  `IQueryBuilder`, `Node`/`Condition`/`Group`, `BaseModel`, `SubQuery`) - previously undocumented
- docs: README examples fixed - the `BaseRepo` examples referenced a non-existent `lxModels` import alias (real
  import is `github.com/epicoon/lxgo/query`), and the `QueryBuilder` example shadowed the `query` package name with a
  local variable; also expanded the operators table with per-operator gotchas and documented the previously-missing
  half of `QueryBuilder`'s API (`AndWhere`/`Or`, `Distinct`/`GroupBy`/`Having`, `OrderBy`/`OrderAsc`/`OrderDesc`,
  `With`'s optional scope), and gave the "Remain" section's `BaseModel.ID uint64` note actual context

------------------------------------------------------------------------------------------------------------------------
Date: 2026.07.24
Version: v0.1.0-alpha.2
Changes:
- docs: add Apache 2.0 LICENSE

------------------------------------------------------------------------------------------------------------------------
Date: 2026.03.19
Version: v0.1.0-alpha.1
