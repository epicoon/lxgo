package model

import (
	"database/sql"
	"fmt"
	"sort"

	"github.com/epicoon/lxgo/kernel"
	lxApp "github.com/epicoon/lxgo/kernel/app"
	"github.com/epicoon/lxgo/migrator"
)

// APP_COMPONENT_KEY is the key ModelManager registers itself under - see
// SetAppComponent/AppComponent.
const APP_COMPONENT_KEY = "lxgo_model_manager"

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * Config
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface kernel.IAppComponentConfig */

// Target is one entry of Config.Targets - a directory of model schema
// files.
type Target struct {
	// Schemas is the directory model schema *.yaml files live in - see
	// LoadModelSchemas.
	Schemas string
	// Namespace is the Postgres schema this target's models live in by
	// default - overrides Config.Namespace for this directory. Empty
	// means "use Config.Namespace" (itself empty means no schema override
	// at all for this directory). A model's own Namespace (its schema
	// file's own `Namespace:` key) overrides this in turn -
	// see ModelSchema.EffectiveNamespace.
	Namespace string
	// BaseModel is the Go type this target's models embed as their
	// generated struct's base by default - overrides Config.BaseModel for
	// this directory. Empty means "use Config.BaseModel" (itself empty
	// means no base type override at all for this directory). A model's
	// own BaseModel (its schema file's own `BaseModel:` key) overrides
	// this in turn - see ModelSchema.EffectiveBaseModel. Not interpreted
	// by this package itself - a code generator's own concern.
	BaseModel string
	// Timestamps overrides Config.Timestamps for this directory's models -
	// whether execCreateTable adds created_at/updated_at/deleted_at columns
	// (see ModelSchema.EffectiveTimestamps). nil means "use
	// Config.Timestamps". A model's own Timestamps (its schema file's own
	// `Timestamps:` key) overrides this in turn.
	Timestamps *bool
	// Models is the directory a code generator writes this target's
	// generated Go model files into (one <model>_gen.go per schema in
	// Schemas, see BuildModelCode) - a relative path, meant to be resolved
	// through ModelManager.App().Pathfinder() the same way Schemas is, by
	// whatever generates the files (not this package itself - see
	// BuildModelCode's own doc). Unlike Namespace/BaseModel/Timestamps,
	// this has no cascade - it's meaningless at the component or model
	// level (every model in a Target shares the
	// same output directory/Go package by construction). Empty means this
	// target's models aren't code-generated at all.
	Models string
	// BaseRepo is the generic repository type this target's models embed
	// by default when scaffolding a repository (one <model>Repo per
	// schema in Schemas, see BuildRepoCode) - overrides Config.BaseRepo
	// for this directory. Empty means "use Config.BaseRepo". A model's
	// own BaseRepo (its schema file's own `BaseRepo:` key) overrides this
	// in turn - see ModelSchema.EffectiveBaseRepo. Not interpreted by
	// this package itself - a code generator's own concern, same as
	// BaseModel.
	BaseRepo string
	// Repos is the directory a code generator writes this target's
	// scaffolded repository files into (one <model>_repo.go per schema in
	// Schemas that doesn't already have one, see BuildRepoCode) - a
	// relative path, resolved the same way Models is. Unlike Models, this
	// governs a file that's written once and never overwritten again (see
	// BuildRepoCode's own doc), and has no cascade for the same reason
	// Models doesn't. Empty means this target's models get no repository
	// scaffold at all. Equal to Models (after resolving both) means the
	// scaffolded repository lives in the same Go package as the model it
	// wraps - a bare identifier reference, no import needed; different
	// from Models means a separate package, importing the model's own
	// package by its Go import path (resolved from the nearest go.mod
	// above Models, see goModuleImportPath).
	Repos string
}

// Config is ModelManager's app-component configuration.
type Config struct {
	*lxApp.ComponentConfig
	// Targets are the model schema directories.
	Targets []Target
	// Namespace is the Postgres schema every model lives in by default,
	// unless a Target or the model's own schema file overrides it - see
	// Target.Namespace/ModelSchema.Namespace. Empty means no schema
	// override anywhere.
	Namespace string
	// BaseModel is the Go type every model embeds as its generated
	// struct's base by default, unless a Target or the model's own schema
	// file overrides it - see Target.BaseModel/ModelSchema.BaseModel.
	// Empty means no base type override anywhere - a code generator's own
	// default applies. Not interpreted by this package itself.
	BaseModel string
	// Timestamps is whether execCreateTable adds created_at/updated_at/
	// deleted_at columns for every model by default, unless a Target or the
	// model's own schema file overrides it - see Target.Timestamps/
	// ModelSchema.Timestamps. False (the zero value) means no model gets
	// them unless something overrides it.
	Timestamps bool
	// BaseRepo is the generic repository type every model embeds by
	// default when scaffolding a repository, unless a Target or the
	// model's own schema file overrides it - see Target.BaseRepo/
	// ModelSchema.BaseRepo. Empty means no override anywhere - a code
	// generator's own default applies. Not interpreted by this package
	// itself.
	BaseRepo string
}

/** @constructor kernel.CAppComponentConfig */

// NewConfig constructs a Config.
func NewConfig() kernel.IAppComponentConfig {
	return &Config{ComponentConfig: lxApp.NewComponentConfigStruct(), Targets: []Target{}}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * ModelManager
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

/** @interface kernel.IAppComponent */

// ModelManager is the app component wiring model schemas/migrations into
// an application - see SetAppComponent to register one.
type ModelManager struct {
	*lxApp.AppComponent
}

var _ kernel.IAppComponent = (*ModelManager)(nil)

// SetAppComponent registers a new ModelManager on app under
// APP_COMPONENT_KEY, configured from the config section named by configKey.
func SetAppComponent(app kernel.IApp, configKey string) error {
	if app.HasComponent(APP_COMPONENT_KEY) {
		return fmt.Errorf("the application already has component: %s", APP_COMPONENT_KEY)
	}

	mm := NewModelManager()
	if err := lxApp.InitComponent(mm, app, configKey); err != nil {
		return fmt.Errorf("can not init model manager component: %s", err)
	}

	app.SetComponent(APP_COMPONENT_KEY, mm)
	return nil
}

// AppComponent returns the ModelManager registered on app under
// APP_COMPONENT_KEY.
func AppComponent(app kernel.IApp) (*ModelManager, error) {
	c := app.Component(APP_COMPONENT_KEY)
	if c == nil {
		return nil, fmt.Errorf("application component '%s' not found", APP_COMPONENT_KEY)
	}

	mm, ok := c.(*ModelManager)
	if !ok {
		return nil, fmt.Errorf("application component '%s' is not '*ModelManager'", APP_COMPONENT_KEY)
	}

	return mm, nil
}

/** @constructor */

// NewModelManager constructs a bare ModelManager - normally reached
// through SetAppComponent instead of calling this directly.
func NewModelManager() *ModelManager {
	return &ModelManager{AppComponent: lxApp.NewAppComponent()}
}

// Name returns the component's name - see kernel.IAppComponent.
func (m *ModelManager) Name() string {
	return "ModelManager"
}

// LogCategory returns the category the component's log methods write under.
func (m *ModelManager) LogCategory() string {
	return "ModelManager"
}

// CConfig returns Config's constructor - see kernel.IAppComponent.
func (m *ModelManager) CConfig() kernel.CAppComponentConfig {
	return NewConfig
}

// Config returns the component's Config.
func (m *ModelManager) Config() *Config {
	return (m.GetConfig()).(*Config)
}

// AfterInit registers Apply/Invert with migrator's migration type registry
// under MigrationType - see kernel.IAppComponent.
func (m *ModelManager) AfterInit() {
	if err := migrator.RegisterMigrationType(MigrationType, Apply, Invert); err != nil {
		m.LogError("can not register migration type %q: %s", MigrationType, err)
	}
}

// DB returns the application's database connection.
func (m *ModelManager) DB() *sql.DB {
	return m.App().Connection().DB()
}

// LoadModelSchemas loads every model schema across all configured
// Targets, resolving each one's ResolvedNamespace/ResolvedBaseModel/
// ResolvedBaseRepo/ResolvedTimestamps along the way - a model's own
// Namespace/BaseModel/BaseRepo/Timestamps if it declares one, else its
// own Target's, else the component-wide Config.Namespace/Config.BaseModel/
// Config.BaseRepo/Config.Timestamps (see
// ModelSchema.EffectiveNamespace/EffectiveBaseModel/EffectiveBaseRepo/
// EffectiveTimestamps for the values to actually read afterward). Two models sharing a Name is only an error if
// they also resolve to the same schema - different schemas may each
// contain a model of the same name. A model whose resolved Timestamps is
// on but that also declares a Field colliding with created_at/updated_at/
// deleted_at is also an error - those three columns are implicit under
// Timestamps (see IntrospectModelSchema), so an explicit Field under one
// of their names could never actually be represented; with Timestamps
// off there's no such restriction, a model is free to declare any of the
// three as an ordinary field. Also validates relations across the
// whole combined set (not directory by directory - a relation to a model
// declared in a different directory is otherwise wrongly rejected) and
// sorts the result by name.
func (m *ModelManager) LoadModelSchemas() ([]*ModelSchema, error) {
	targs := m.Config().Targets

	var schemas []*ModelSchema
	// Keyed by (Name, ResolvedNamespace) rather than Name alone - DDL/
	// introspection schema-qualify every physical object now (see
	// IntrospectModelSchema/Apply/Invert), so two models sharing a Name
	// only actually collide if they also resolve to the same schema.
	type seenKey struct{ name, namespace string }
	seenIn := make(map[seenKey]string, len(targs))
	for _, target := range targs {
		path := m.App().Pathfinder().GetAbsPath(target.Schemas)
		dirSchemas, err := loadModelSchemaFiles(path)
		if err != nil {
			return nil, err
		}
		for _, s := range dirSchemas {
			if s.Namespace != "" {
				s.ResolvedNamespace = s.Namespace
			} else if target.Namespace != "" {
				s.ResolvedNamespace = target.Namespace
			} else {
				s.ResolvedNamespace = m.Config().Namespace
			}
			if s.BaseModel != "" {
				s.ResolvedBaseModel = s.BaseModel
			} else if target.BaseModel != "" {
				s.ResolvedBaseModel = target.BaseModel
			} else {
				s.ResolvedBaseModel = m.Config().BaseModel
			}
			if s.BaseRepo != "" {
				s.ResolvedBaseRepo = s.BaseRepo
			} else if target.BaseRepo != "" {
				s.ResolvedBaseRepo = target.BaseRepo
			} else {
				s.ResolvedBaseRepo = m.Config().BaseRepo
			}
			var resolvedTimestamps bool
			switch {
			case s.Timestamps != nil:
				resolvedTimestamps = *s.Timestamps
			case target.Timestamps != nil:
				resolvedTimestamps = *target.Timestamps
			default:
				resolvedTimestamps = m.Config().Timestamps
			}
			s.ResolvedTimestamps = &resolvedTimestamps
			if resolvedTimestamps {
				for _, f := range s.Fields {
					if isTimestampColumn(pgColumnName(f.Name)) {
						return nil, fmt.Errorf("model %q: Timestamps is enabled, field %q collides with the implicit created_at/updated_at/deleted_at columns it adds", s.Name, f.Name)
					}
				}
			}

			key := seenKey{s.Name, s.ResolvedNamespace}
			if prevDir, ok := seenIn[key]; ok {
				return nil, fmt.Errorf("model %q (schema %q) is declared in both %q and %q",
					s.Name, pgResolveSchema(s.ResolvedNamespace), prevDir, target.Schemas)
			}
			seenIn[key] = target.Schemas
			schemas = append(schemas, s)
		}
	}

	sort.Slice(schemas, func(i, j int) bool { return schemas[i].Name < schemas[j].Name })
	if err := validateRelations(schemas); err != nil {
		return nil, err
	}
	return schemas, nil
}
