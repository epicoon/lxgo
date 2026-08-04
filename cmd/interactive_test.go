package cmd

import (
	"bufio"
	"strings"
	"testing"
)

// withStdinInput points stdinReader at input for the duration of the test,
// restoring the original reader afterward.
func withStdinInput(t *testing.T, input string) {
	t.Helper()
	old := stdinReader
	stdinReader = bufio.NewReader(strings.NewReader(input))
	t.Cleanup(func() { stdinReader = old })
}

func TestPromptForParam_String(t *testing.T) {
	withStdinInput(t, "hello\n")
	val, err := promptForParam(nil, "name", ParamConfig{Type: ParamTypeString})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "hello" {
		t.Fatalf("got %v, want hello", val)
	}
}

func TestPromptForParam_Int(t *testing.T) {
	withStdinInput(t, "42\n")
	val, err := promptForParam(nil, "count", ParamConfig{Type: ParamTypeInt})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 42 {
		t.Fatalf("got %v, want 42", val)
	}
}

func TestPromptForParam_Bool(t *testing.T) {
	withStdinInput(t, "true\n")
	val, err := promptForParam(nil, "flag", ParamConfig{Type: ParamTypeBool})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != true {
		t.Fatalf("got %v, want true", val)
	}
}

// TestPromptForParam_RetriesOnInvalidInput is a regression test for a bug
// found while writing it: PromptString used to wrap os.Stdin in a brand new
// bufio.Reader on every call - since bufio reads ahead into its own
// internal buffer, the first call (asking for an int) could silently
// consume the SECOND line too, and throw it away when that reader instance
// was discarded, leaving the retry with nothing to read. Both prompts now
// share one persistent reader (stdinReader), so a rejected answer doesn't
// eat the next one.
func TestPromptForParam_RetriesOnInvalidInput(t *testing.T) {
	withStdinInput(t, "not-a-number\n7\n")
	val, err := promptForParam(nil, "count", ParamConfig{Type: ParamTypeInt})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 7 {
		t.Fatalf("got %v, want 7 (the second line, after the invalid first)", val)
	}
}

func TestPromptForParam_EmptyInputRetries(t *testing.T) {
	withStdinInput(t, "\nvalue\n")
	val, err := promptForParam(nil, "name", ParamConfig{Type: ParamTypeString})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "value" {
		t.Fatalf("got %v, want value", val)
	}
}

func TestPromptForParam_UsesDescriptionAsQuestion(t *testing.T) {
	// Just exercises the fallback-question branch without a real terminal
	// to print to - promptText's own behavior is covered indirectly via
	// the prompts above; this only checks promptForParam doesn't panic or
	// misbehave when Description is empty.
	withStdinInput(t, "x\n")
	if _, err := promptForParam(nil, "name", ParamConfig{Type: ParamTypeString, Description: ""}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestPromptForParam_Enum_StaticTypeDetails checks the plain TypeDetails
// path (no FTypeDetails): the chosen option's own value (not its index) is
// returned.
func TestPromptForParam_Enum_StaticTypeDetails(t *testing.T) {
	withStdinInput(t, "2\n")
	val, err := promptForParam(nil, "path", ParamConfig{
		Type:        ParamTypeEnum,
		TypeDetails: []string{"first", "second", "third"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "second" {
		t.Fatalf("got %v, want second", val)
	}
}

// TestPromptForParam_Enum_FTypeDetailsResolvesOptions checks the lazy path:
// FTypeDetails is called (with the command passed to promptForParam) to
// compute the options, instead of a static TypeDetails.
func TestPromptForParam_Enum_FTypeDetailsResolvesOptions(t *testing.T) {
	called := 0
	c := NewCommand()
	c.SetParam("dir", "configured-dir")

	withStdinInput(t, "1\n")
	val, err := promptForParam(c, "path", ParamConfig{
		Type: ParamTypeEnum,
		FTypeDetails: func(cmd ICommand) (any, error) {
			called++
			dir, _ := cmd.Param("dir").(string)
			return []string{dir}, nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called != 1 {
		t.Fatalf("FTypeDetails called %d times, want 1", called)
	}
	if val != "configured-dir" {
		t.Fatalf("got %v, want configured-dir", val)
	}
}

// TestPromptForParam_Enum_ElemTypeInt checks that an ElemType: ParamTypeInt
// enum resolves to an actual int, not its string rendering.
func TestPromptForParam_Enum_ElemTypeInt(t *testing.T) {
	withStdinInput(t, "2\n")
	val, err := promptForParam(nil, "count", ParamConfig{
		Type:        ParamTypeEnum,
		ElemType:    ParamTypeInt,
		TypeDetails: []int{10, 20, 30},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 20 {
		t.Fatalf("got %v (%T), want int 20", val, val)
	}
}

func TestPromptSelectFallback_ValidChoice(t *testing.T) {
	withStdinInput(t, "2\n")
	idx, err := promptSelectFallback("pick one", []string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != 1 {
		t.Fatalf("got index %d, want 1", idx)
	}
}

// TestPromptSelectFallback_RetriesOnOutOfRange checks that an out-of-range
// or non-numeric answer doesn't get accepted, and the next valid line is
// used instead.
func TestPromptSelectFallback_RetriesOnOutOfRange(t *testing.T) {
	withStdinInput(t, "0\nnope\n5\n3\n")
	idx, err := promptSelectFallback("pick one", []string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != 2 {
		t.Fatalf("got index %d, want 2 (the third option, from the last valid line '3')", idx)
	}
}
