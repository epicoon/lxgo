package model

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/epicoon/lxgo/cmd"
)

// modelsTargetByDir returns every Config.Targets entry keyed by its own
// resolved Schemas directory - the same resolution
// (App().Pathfinder().GetAbsPath) LoadModelSchemas itself applies before
// setting each loaded ModelSchema's own SourceDir, so a schema s's own
// Target is modelsTargetByDir()[s.SourceDir]. Two different schemas can
// share the same Name (see LoadModelSchemas's own doc), so SourceDir is
// the only key that resolves a schema's own Target unambiguously.
func (m *ModelManager) modelsTargetByDir() map[string]Target {
	byDir := make(map[string]Target, len(m.Config().Targets))
	for _, target := range m.Config().Targets {
		byDir[m.App().Pathfinder().GetAbsPath(target.Schemas)] = target
	}
	return byDir
}

// modelSchemaFilePath returns the yaml file s was itself loaded from -
// SourceDir plus s.Name (see loadModelSchemaFiles: a schema's Name is its
// file's own basename with ".yaml" trimmed off, so this reconstructs the
// exact original path).
func modelSchemaFilePath(s *ModelSchema) string {
	return filepath.Join(s.SourceDir, s.Name+".yaml")
}

// crossPackageRelations reports every RelationTypeOneToOne/
// RelationTypeManyToOne/RelationTypeManyToMany relation on s whose
// RelatedModel is known to be generated into a different directory than
// modelsDir (s's own) - BuildModelCode always references a related type
// as a bare identifier, assuming the same Go package (see its own doc),
// so such a relation would silently produce code that fails to compile
// with an "undefined: X" error. A RelatedModel not present in
// pkgDirByModel at all (its own Target has no Models configured, or it
// isn't one of the loaded schemas) isn't flagged - it may be a type the
// caller intends to provide by hand under the same name, not necessarily
// a mistake.
func crossPackageRelations(s *ModelSchema, modelsDir string, pkgDirByModel map[string]string) []string {
	var bad []string
	for _, r := range s.Relations {
		switch r.Type {
		case RelationTypeOneToOne, RelationTypeManyToOne, RelationTypeManyToMany:
		default:
			continue
		}
		if relatedDir, ok := pkgDirByModel[r.RelatedModel]; ok && relatedDir != modelsDir {
			bad = append(bad, fmt.Sprintf("%s (-> %s, generated into %s)", r.Name, r.RelatedModel, relatedDir))
		}
	}
	return bad
}

// goPackageNameFromDir derives a Go package name from modelsDir's own
// last path segment, the convention BuildModelCode's callers use - and
// reports whether that segment is actually a valid Go identifier. A
// directory whose basename isn't one (e.g. it's purely numeric, or
// contains a hyphen) would otherwise only surface as a confusing gofmt
// parse error out of BuildModelCode itself, with no indication that the
// real cause is the configured Models path's own last segment.
func goPackageNameFromDir(modelsDir string) (string, bool) {
	name := filepath.Base(modelsDir)
	if name == "" {
		return name, false
	}
	for i, r := range name {
		switch {
		case unicode.IsLetter(r) || r == '_':
		case unicode.IsDigit(r) && i > 0:
		default:
			return name, false
		}
	}
	return name, true
}

// codegenPlan reports which models still need a (re)generated Go model
// file and which still need a scaffolded repository file - the same
// mtime-based staleness check codegenStatus prints per model, factored
// out so model:actualize can show the same information in its own
// summary before asking to apply.
func (m *ModelManager) codegenPlan(schemas []*ModelSchema) (staleModels, missingRepos []string) {
	targets := m.modelsTargetByDir()
	for _, s := range schemas {
		target := targets[s.SourceDir]
		if target.Models == "" {
			continue
		}

		modelsDir := m.App().Pathfinder().GetAbsPath(target.Models)
		outPath := filepath.Join(modelsDir, ModelCodeFileName(s.Name))
		outInfo, err := os.Stat(outPath)
		if os.IsNotExist(err) {
			staleModels = append(staleModels, s.Name)
		} else if err == nil {
			if schemaInfo, err := os.Stat(modelSchemaFilePath(s)); err == nil && schemaInfo.ModTime().After(outInfo.ModTime()) {
				staleModels = append(staleModels, s.Name)
			}
		}

		if target.Repos == "" {
			continue
		}
		repoPath := filepath.Join(m.App().Pathfinder().GetAbsPath(target.Repos), RepoCodeFileName(s.Name))
		if _, err := os.Stat(repoPath); os.IsNotExist(err) {
			missingRepos = append(missingRepos, s.Name)
		}
	}
	return staleModels, missingRepos
}

/** @handler cmd.FAction */
func codegenStatus(com cmd.ICommand) error {
	c := com.(*Command)
	mm, err := AppComponent(c.app)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil
	}

	schemas, err := mm.LoadModelSchemas()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil
	}
	targets := mm.modelsTargetByDir()

	configured := false
	for _, s := range schemas {
		target := targets[s.SourceDir]
		if target.Models == "" {
			continue
		}
		configured = true

		modelsDir := mm.App().Pathfinder().GetAbsPath(target.Models)
		outPath := filepath.Join(modelsDir, ModelCodeFileName(s.Name))

		outInfo, err := os.Stat(outPath)
		if os.IsNotExist(err) {
			fmt.Printf("- %s: not generated (%s)\n", s.Name, outPath)
			continue
		}
		if err != nil {
			fmt.Printf("- %s: error checking %s: %s\n", s.Name, outPath, err)
			continue
		}

		schemaInfo, err := os.Stat(modelSchemaFilePath(s))
		if err != nil {
			fmt.Printf("- %s: error checking schema file: %s\n", s.Name, err)
			continue
		}

		if schemaInfo.ModTime().After(outInfo.ModTime()) {
			fmt.Printf("- %s: stale (schema changed since last generation)\n", s.Name)
		} else {
			fmt.Printf("- %s: up to date\n", s.Name)
		}

		if target.Repos != "" {
			repoPath := filepath.Join(mm.App().Pathfinder().GetAbsPath(target.Repos), RepoCodeFileName(s.Name))
			if _, err := os.Stat(repoPath); os.IsNotExist(err) {
				fmt.Printf("    repo: not scaffolded (%s)\n", repoPath)
			} else if err != nil {
				fmt.Printf("    repo: error checking %s: %s\n", repoPath, err)
			} else {
				fmt.Printf("    repo: scaffolded (%s)\n", repoPath)
			}
		}
	}

	if !configured {
		fmt.Println("No Target has Models configured - nothing to check")
	}
	return nil
}

/** @handler cmd.FAction */
func codegenGenerate(com cmd.ICommand) error {
	c := com.(*Command)
	mm, err := AppComponent(c.app)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil
	}

	schemas, err := mm.LoadModelSchemas()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil
	}
	targets := mm.modelsTargetByDir()

	// pkgDirByModel resolves a RelatedModel name to its own generated
	// package directory, for crossPackageRelations below - keyed by name
	// since Relation.RelatedModel carries nothing else, so it can't tell
	// apart two schemas that share a Name the way a SourceDir-keyed
	// lookup could (see LoadModelSchemas's own doc).
	pkgDirByModel := make(map[string]string, len(schemas))
	for _, s := range schemas {
		if target := targets[s.SourceDir]; target.Models != "" {
			pkgDirByModel[s.Name] = mm.App().Pathfinder().GetAbsPath(target.Models)
		}
	}

	generated := 0
	for _, s := range schemas {
		target := targets[s.SourceDir]
		if target.Models == "" {
			continue
		}
		modelsDir := mm.App().Pathfinder().GetAbsPath(target.Models)

		if bad := crossPackageRelations(s, modelsDir, pkgDirByModel); len(bad) > 0 {
			fmt.Printf("Error generating %s: relations reference a different generated package:\n", s.Name)
			for _, b := range bad {
				fmt.Printf("    %s\n", b)
			}
			continue
		}

		pkgName, ok := goPackageNameFromDir(modelsDir)
		if !ok {
			fmt.Printf("Error generating %s: %q is not a valid Go package name - rename %s's last path segment (see Target.Models)\n", s.Name, pkgName, modelsDir)
			continue
		}

		code, err := BuildModelCode(pkgName, s)
		if err != nil {
			fmt.Printf("Error generating %s: %s\n", s.Name, err)
			continue
		}

		if err := os.MkdirAll(modelsDir, 0755); err != nil {
			fmt.Printf("Error generating %s: %s\n", s.Name, err)
			continue
		}
		outPath := filepath.Join(modelsDir, ModelCodeFileName(s.Name))
		if err := os.WriteFile(outPath, code, 0644); err != nil {
			fmt.Printf("Error generating %s: %s\n", s.Name, err)
			continue
		}

		fmt.Printf("Generated %s\n", outPath)
		generated++
	}

	if generated == 0 {
		fmt.Println("Nothing to generate - no Target has Models configured")
	}
	return nil
}

// goModuleImportPath returns the Go import path dir itself has - its
// nearest ancestor go.mod's own module directive, plus dir's own path
// relative to that go.mod's directory (empty relative path - dir is the
// module root itself - returns the module path unchanged). The same
// resolution "go build"/"go list" apply themselves, needed here because
// this package otherwise only ever deals in filesystem paths
// (App().Pathfinder()), never Go import paths - codegenRepos needs one to
// reference a model generated into a different package than the
// repository wrapping it (see BuildRepoCode's own modelPkg parameter).
func goModuleImportPath(dir string) (string, error) {
	modDir := dir
	for {
		data, err := os.ReadFile(filepath.Join(modDir, "go.mod"))
		if err == nil {
			modulePath, err := parseGoModModule(data)
			if err != nil {
				return "", fmt.Errorf("%s: %w", filepath.Join(modDir, "go.mod"), err)
			}
			rel, err := filepath.Rel(modDir, dir)
			if err != nil {
				return "", err
			}
			if rel == "." {
				return modulePath, nil
			}
			return modulePath + "/" + filepath.ToSlash(rel), nil
		}
		parent := filepath.Dir(modDir)
		if parent == modDir {
			return "", fmt.Errorf("no go.mod found above %q", dir)
		}
		modDir = parent
	}
}

// parseGoModModule extracts go.mod's own "module ..." directive value
// from its raw content - the first line (after trimming whitespace)
// starting with "module ".
func parseGoModModule(data []byte) (string, error) {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", fmt.Errorf("no 'module' directive found")
}

/** @handler cmd.FAction */
func codegenRepos(com cmd.ICommand) error {
	c := com.(*Command)
	mm, err := AppComponent(c.app)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil
	}

	schemas, err := mm.LoadModelSchemas()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return nil
	}
	targets := mm.modelsTargetByDir()

	scaffolded := 0
	for _, s := range schemas {
		target := targets[s.SourceDir]
		if target.Repos == "" {
			continue
		}
		if target.Models == "" {
			fmt.Printf("Error scaffolding %s: Target.Repos is set but Target.Models isn't - a repository needs a generated model to wrap\n", s.Name)
			continue
		}

		reposDir := mm.App().Pathfinder().GetAbsPath(target.Repos)
		modelsDir := mm.App().Pathfinder().GetAbsPath(target.Models)
		outPath := filepath.Join(reposDir, RepoCodeFileName(s.Name))

		if _, err := os.Stat(outPath); err == nil {
			fmt.Printf("Skipping %s: %s already exists\n", s.Name, outPath)
			continue
		} else if !os.IsNotExist(err) {
			fmt.Printf("Error scaffolding %s: checking %s: %s\n", s.Name, outPath, err)
			continue
		}

		var modelPkg string
		if reposDir != modelsDir {
			modelPkg, err = goModuleImportPath(modelsDir)
			if err != nil {
				fmt.Printf("Error scaffolding %s: resolving the models package's own import path: %s\n", s.Name, err)
				continue
			}
		}

		pkgName, ok := goPackageNameFromDir(reposDir)
		if !ok {
			fmt.Printf("Error scaffolding %s: %q is not a valid Go package name - rename %s's last path segment (see Target.Repos)\n", s.Name, pkgName, reposDir)
			continue
		}

		code, err := BuildRepoCode(pkgName, s, modelPkg)
		if err != nil {
			fmt.Printf("Error scaffolding %s: %s\n", s.Name, err)
			continue
		}

		if err := os.MkdirAll(reposDir, 0755); err != nil {
			fmt.Printf("Error scaffolding %s: %s\n", s.Name, err)
			continue
		}
		if err := os.WriteFile(outPath, code, 0644); err != nil {
			fmt.Printf("Error scaffolding %s: %s\n", s.Name, err)
			continue
		}

		fmt.Printf("Scaffolded %s\n", outPath)
		scaffolded++
	}

	if scaffolded == 0 {
		fmt.Println("Nothing to scaffold - no Target has Repos configured, or every repository file already exists")
	}
	return nil
}
