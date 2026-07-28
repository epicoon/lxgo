package cmd

import (
	"reflect"
	"testing"
)

func TestParseArgs_NoArgs(t *testing.T) {
	m := &manager{}
	m.parseArgs(nil)

	if m.cmdRoute != "" || m.cmdName != "" || m.subName != "" {
		t.Fatalf("expected everything empty, got route=%q name=%q sub=%q", m.cmdRoute, m.cmdName, m.subName)
	}
}

func TestParseArgs_CommandOnly(t *testing.T) {
	m := &manager{}
	m.parseArgs([]string{"migrator"})

	if m.cmdName != "migrator" || m.subName != "" {
		t.Fatalf("name=%q sub=%q, want name='migrator' sub=''", m.cmdName, m.subName)
	}
}

func TestParseArgs_CommandAndAction(t *testing.T) {
	m := &manager{}
	m.parseArgs([]string{"migrator:up"})

	if m.cmdName != "migrator" || m.subName != "up" {
		t.Fatalf("name=%q sub=%q, want name='migrator' sub='up'", m.cmdName, m.subName)
	}
}

func TestParseArgs_LongFlagWithValue(t *testing.T) {
	m := &manager{}
	m.parseArgs([]string{"migrator:create", "--name=add_users"})

	if got := m.params["name"]; got != "add_users" {
		t.Fatalf("params[name] = %#v, want 'add_users'", got)
	}
}

func TestParseArgs_LongFlagWithoutValue_IsBoolTrue(t *testing.T) {
	m := &manager{}
	m.parseArgs([]string{"migrator", "--help"})

	if got := m.params["help"]; got != true {
		t.Fatalf("params[help] = %#v, want true", got)
	}
}

func TestParseArgs_ShortFlagsAreCombinedBooleans(t *testing.T) {
	m := &manager{}
	m.parseArgs([]string{"migrator", "-abc"})

	want := map[string]any{"a": true, "b": true, "c": true}
	if !reflect.DeepEqual(m.params, want) {
		t.Fatalf("params = %#v, want %#v", m.params, want)
	}
}

func TestParseArgs_MixedLongAndShortFlags(t *testing.T) {
	m := &manager{}
	m.parseArgs([]string{"migrator:show", "--count=5", "-v"})

	if m.params["count"] != "5" || m.params["v"] != true {
		t.Fatalf("params = %#v", m.params)
	}
}

func TestDefineConstructor_Found(t *testing.T) {
	sentinel := func(opt ...ICommandOptions) ICommand { return nil }
	m := &manager{list: map[string]CCommand{"migrator": sentinel}, cmdName: "migrator"}

	got, err := m.defineConstructor()
	if err != nil {
		t.Fatalf("defineConstructor: %v", err)
	}
	if reflect.ValueOf(got).Pointer() != reflect.ValueOf(sentinel).Pointer() {
		t.Fatal("expected the registered constructor to be returned")
	}
}

func TestDefineConstructor_UnknownCommand(t *testing.T) {
	m := &manager{list: map[string]CCommand{}, cmdName: "nope"}
	if _, err := m.defineConstructor(); err == nil {
		t.Fatal("expected an error for an unregistered command name")
	}
}

func TestDefineConstructor_NoDefaultCommand(t *testing.T) {
	m := &manager{list: map[string]CCommand{}, cmdName: ""}
	if _, err := m.defineConstructor(); err == nil {
		t.Fatal("expected an error when no default command (\"\") is registered")
	}
}
