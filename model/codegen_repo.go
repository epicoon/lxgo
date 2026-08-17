package model

import (
	"fmt"
	"go/format"
	"strings"
)

// baseRepoDefault is the generic repository type BuildRepoCode embeds when
// a model's own EffectiveBaseRepo is empty anywhere in the cascade.
const baseRepoDefault = "github.com/epicoon/lxgo/query.BaseRepo"

// RepoCodeFileName returns the file name BuildRepoCode's own output for
// modelName is conventionally written under - <snake_case(modelName)>_repo.go,
// the same snake_case conversion ModelCodeFileName uses, without a "_gen"
// suffix - unlike a model file, a repository file is written once and
// never overwritten again (see BuildRepoCode's own doc), so it isn't
// "generated" in the same ongoing sense.
func RepoCodeFileName(modelName string) string {
	return pgColumnName(modelName) + "_repo.go"
}

// BuildRepoCode generates the Go source of a named repository type
// wrapping schema's own resolved BaseRepo (EffectiveBaseRepo, or
// baseRepoDefault if that's empty) - "<Model>Repo", embedding
// "<BaseRepoType>[<Model>]", plus a "New<Model>Repo(db *gorm.DB)
// *<Model>Repo" constructor calling the base type's own
// "New<BaseRepoTypeName>[<Model>](db, nil)" constructor. This mirrors
// query.BaseRepo/query.NewBaseRepo's own naming exactly - a custom
// BaseRepo is expected to follow the same convention, since its
// unexported fields (db/tx/...) leave no way to construct one from
// outside its own package other than through a same-named constructor
// function.
//
// modelPkg is the model's own Go import path if it's generated into a
// different package than the repository (the model is then referenced as
// "<alias>.<Model>", with an explicit import under that alias - see
// packageAlias), or empty if they're the same package (a bare "<Model>"
// reference, no import needed). Resolving which of the two applies (and
// modelPkg's own value, when needed) is a caller's concern - see
// goModuleImportPath.
//
// Unlike BuildModelCode's own output, the result carries no
// "// Code generated ...; DO NOT EDIT." banner - a repository file is
// meant to be hand-edited immediately after it's scaffolded (custom
// finder methods, etc.), so marking it generated/off-limits would be
// actively wrong. For the same reason, a caller must never call this for
// a file that already exists - see codegenRepos in cmd_codegen.go.
func BuildRepoCode(pkgName string, schema *ModelSchema, modelPkg string) ([]byte, error) {
	baseRepo := schema.EffectiveBaseRepo()
	if baseRepo == "" {
		baseRepo = baseRepoDefault
	}
	ref, err := parseGoTypeRef("BaseRepo", baseRepo)
	if err != nil {
		return nil, fmt.Errorf("model %q: %w", schema.Name, err)
	}

	imps := goImports{}
	imps.add(ref.importPath)
	imps.add("gorm.io/gorm")

	modelType := schema.Name
	if modelPkg != "" {
		imps.add(modelPkg)
		modelType = packageAlias(modelPkg) + "." + schema.Name
	}

	repoAlias := imps[ref.importPath]
	repoType := repoAlias + "." + ref.typeName
	newRepoFunc := repoAlias + ".New" + ref.typeName

	var b strings.Builder
	fmt.Fprintf(&b, "package %s\n\n", pkgName)
	b.WriteString(imps.render())
	b.WriteString("\n")
	fmt.Fprintf(&b, "type %sRepo struct {\n\t*%s[%s]\n}\n\n", schema.Name, repoType, modelType)
	fmt.Fprintf(&b, "func New%sRepo(db *gorm.DB) *%sRepo {\n\treturn &%sRepo{%s: %s[%s](db, nil)}\n}\n",
		schema.Name, schema.Name, schema.Name, ref.typeName, newRepoFunc, modelType)

	formatted, err := format.Source([]byte(b.String()))
	if err != nil {
		return nil, fmt.Errorf("model %q: generated invalid Go repo source: %w", schema.Name, err)
	}
	return formatted, nil
}
