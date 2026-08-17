package model

import (
	"testing"

	"github.com/epicoon/lxgo/cmd"
	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/apptest"
)

func TestNewCommand_PanicsWithoutApp(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected NewCommand to panic without CommandOptions.App")
		}
	}()
	NewCommand()
}

func TestCommand_Config(t *testing.T) {
	c := &Command{Command: cmd.NewCommand()}
	conf := c.Config()

	if _, ok := conf.Actions["db-status"]; !ok {
		t.Fatal(`expected a "db-status" action`)
	}
	migrateAction, ok := conf.Actions["db-migrate"]
	if !ok {
		t.Fatal(`expected a "db-migrate" action`)
	}
	if _, ok := conf.Actions["db-audit"]; !ok {
		t.Fatal(`expected a "db-audit" action`)
	}
	nameParam, ok := migrateAction.Params["name"]
	if !ok || !nameParam.Required || nameParam.Type != cmd.ParamTypeString {
		t.Fatalf(`"db-migrate"'s "name" param = %#v, want a required string`, nameParam)
	}
	applyParam, ok := migrateAction.Params["apply"]
	if !ok || applyParam.Type != cmd.ParamTypeBool {
		t.Fatalf(`"db-migrate"'s "apply" param = %#v, want a bool`, applyParam)
	}
	actualizeAction, ok := conf.Actions["actualize"]
	if !ok {
		t.Fatal(`expected an "actualize" action`)
	}
	if p, ok := actualizeAction.Params["name"]; !ok || p.Required || p.Type != cmd.ParamTypeString {
		t.Fatalf(`"actualize"'s "name" param = %#v, want an optional string`, p)
	}
	if p, ok := actualizeAction.Params["yes"]; !ok || p.Type != cmd.ParamTypeBool {
		t.Fatalf(`"actualize"'s "yes" param = %#v, want a bool`, p)
	}
}

func TestDbMigrate_MissingNamePrintsInsteadOfErroring(t *testing.T) {
	c := &Command{Command: cmd.NewCommand()}
	c.SetParams(map[string]any{})

	if err := dbMigrate(c); err != nil {
		t.Fatalf("dbMigrate: %v", err)
	}
}

func newTestAppNoModelManager(t *testing.T) kernel.IApp {
	t.Helper()
	app, err := apptest.New()
	if err != nil {
		t.Fatalf("apptest.New: %v", err)
	}
	return app
}

func TestDbStatus_NoModelManagerRegisteredPrintsInsteadOfErroring(t *testing.T) {
	app := newTestAppNoModelManager(t)
	c := &Command{Command: cmd.NewCommand(), app: app}

	if err := dbStatus(c); err != nil {
		t.Fatalf("dbStatus: %v", err)
	}
}

func TestDbMigrate_NoModelManagerRegisteredPrintsInsteadOfErroring(t *testing.T) {
	app := newTestAppNoModelManager(t)
	c := &Command{Command: cmd.NewCommand(), app: app}
	c.SetParams(map[string]any{"name": "test_migration"})

	if err := dbMigrate(c); err != nil {
		t.Fatalf("dbMigrate: %v", err)
	}
}

func TestDbAudit_NoModelManagerRegisteredPrintsInsteadOfErroring(t *testing.T) {
	app := newTestAppNoModelManager(t)
	c := &Command{Command: cmd.NewCommand(), app: app}

	if err := dbAudit(c); err != nil {
		t.Fatalf("dbAudit: %v", err)
	}
}

func TestActualize_NoModelManagerRegisteredPrintsInsteadOfErroring(t *testing.T) {
	app := newTestAppNoModelManager(t)
	c := &Command{Command: cmd.NewCommand(), app: app}

	if err := actualize(c); err != nil {
		t.Fatalf("actualize: %v", err)
	}
}
