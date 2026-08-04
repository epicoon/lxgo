package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// stdinReader is shared across every PromptString/promptSelectFallback call
// in the process, instead of each wrapping os.Stdin in a fresh bufio.Reader
// - bufio reads ahead into its own internal buffer, so a fresh reader per
// call would silently lose whatever the previous call's (now-discarded)
// buffer had already read past the line it returned. Tests reassign this
// directly to a reader over canned input.
var stdinReader = bufio.NewReader(os.Stdin)

// PromptString prints question as a prompt and reads one line from stdin,
// trimming the trailing newline - used to fill in a parameter interactively
// (see the "interactive" flag handling in validate).
func PromptString(question string) (string, error) {
	fmt.Print(promptText(question))

	line, err := stdinReader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// promptForParam interactively fills in a missing required parameter -
// used by validate when the parameter is going interactive (see
// ParamConfig.Interactive and the "--interactive" flag). Loops until it
// gets input matching pConf.Type, using pConf.Description as the prompt
// (falling back to pName/pConf.Type when there's no description). c is
// only used for ParamTypeEnum's FTypeDetails, if set.
func promptForParam(c ICommand, pName string, pConf ParamConfig) (any, error) {
	question := pConf.Description
	if question == "" {
		question = fmt.Sprintf("Enter value for '%s' (%s)", pName, pConf.Type)
	}

	if pConf.Type == ParamTypeEnum {
		return promptForEnumParam(c, question, pConf)
	}

	for {
		raw, err := PromptString(question)
		if err != nil {
			return nil, err
		}
		if raw == "" {
			fmt.Println("a value is required")
			continue
		}

		switch pConf.Type {
		case ParamTypeInt:
			i, err := strconv.Atoi(raw)
			if err != nil {
				fmt.Printf("expected an integer, got '%s'\n", raw)
				continue
			}
			return i, nil
		case ParamTypeBool:
			switch raw {
			case "true", "false":
				return raw == "true", nil
			default:
				fmt.Println("expected 'true' or 'false'")
				continue
			}
		default:
			return raw, nil
		}
	}
}

// promptForEnumParam resolves a ParamTypeEnum parameter's options - from
// pConf.TypeDetails, or by calling pConf.FTypeDetails(c) if set (this is
// the only place FTypeDetails is ever called, so a filesystem scan or
// similar behind it only runs once we're actually about to prompt) - and
// lets the user pick one with PromptSelect.
func promptForEnumParam(c ICommand, question string, pConf ParamConfig) (any, error) {
	details := pConf.TypeDetails
	if pConf.FTypeDetails != nil {
		var err error
		details, err = pConf.FTypeDetails(c)
		if err != nil {
			return nil, err
		}
	}

	display, values, err := enumDisplayOptions(pConf.ElemType, details)
	if err != nil {
		return nil, err
	}

	idx, err := PromptSelect(question, display)
	if err != nil {
		return nil, err
	}
	return values[idx], nil
}

// enumDisplayOptions turns a ParamTypeEnum parameter's resolved details
// into display strings for PromptSelect, alongside the actual typed value
// each one corresponds to - so e.g. an ElemType: ParamTypeInt parameter
// resolves to an int, not its string rendering.
func enumDisplayOptions(elemType ParamType, details any) (display []string, values []any, err error) {
	switch elemType {
	case ParamTypeInt:
		ints, ok := details.([]int)
		if !ok {
			return nil, nil, fmt.Errorf("enum parameter details must be []int for ElemType %s, got %T", elemType, details)
		}
		display = make([]string, len(ints))
		values = make([]any, len(ints))
		for i, v := range ints {
			display[i] = strconv.Itoa(v)
			values[i] = v
		}
	default:
		strs, ok := details.([]string)
		if !ok {
			return nil, nil, fmt.Errorf("enum parameter details must be []string, got %T", details)
		}
		display = make([]string, len(strs))
		values = make([]any, len(strs))
		for i, v := range strs {
			display[i] = v
			values[i] = v
		}
	}

	if len(display) == 0 {
		return nil, nil, errors.New("enum parameter has no options to choose from")
	}
	return display, values, nil
}

func promptText(question string) string {
	q := strings.TrimRight(question, " ")
	if q == "" {
		return "> "
	}
	switch q[len(q)-1] {
	case ':', '?':
		return q + " "
	default:
		return q + ": "
	}
}

// PromptSelect prints question followed by options as a menu and lets the
// user pick one with the up/down arrows and Enter (Ctrl+C aborts) - returns
// the chosen option's index. Falls back to a plain numbered prompt when
// stdin isn't a real terminal (e.g. piped input, a non-interactive CI run).
func PromptSelect(question string, options []string) (int, error) {
	if len(options) == 0 {
		return -1, errors.New("no options to choose from")
	}

	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return promptSelectFallback(question, options)
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return -1, fmt.Errorf("can not switch terminal to raw mode: %w", err)
	}
	defer term.Restore(fd, oldState)

	if question != "" {
		fmt.Print(question + "\r\n")
	}

	selected := 0
	renderSelectMenu(options, selected, false)

	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return -1, err
		}

		switch {
		case n >= 1 && (buf[0] == '\r' || buf[0] == '\n'):
			fmt.Print("\r\n")
			return selected, nil
		case n >= 1 && buf[0] == 3: // Ctrl+C
			fmt.Print("\r\n")
			return -1, errors.New("selection aborted")
		case n == 3 && buf[0] == 0x1b && buf[1] == '[' && buf[2] == 'A': // up
			selected = (selected - 1 + len(options)) % len(options)
			renderSelectMenu(options, selected, true)
		case n == 3 && buf[0] == 0x1b && buf[1] == '[' && buf[2] == 'B': // down
			selected = (selected + 1) % len(options)
			renderSelectMenu(options, selected, true)
		}
	}
}

// renderSelectMenu draws options, marking the selected one - redraw moves
// the cursor back up over the previous render first (raw mode disables the
// terminal's own line editing, so \r\n is used explicitly instead of \n).
func renderSelectMenu(options []string, selected int, redraw bool) {
	if redraw {
		fmt.Printf("\x1b[%dA", len(options))
	}
	for i, opt := range options {
		fmt.Print("\x1b[2K\r")
		if i == selected {
			fmt.Printf("> %s\r\n", opt)
		} else {
			fmt.Printf("  %s\r\n", opt)
		}
	}
}

// promptSelectFallback is PromptSelect's non-terminal path: options are
// printed as a numbered list, and the user types the number and Enter.
func promptSelectFallback(question string, options []string) (int, error) {
	if question != "" {
		fmt.Println(question)
	}
	for i, opt := range options {
		fmt.Printf("  %d) %s\n", i+1, opt)
	}

	for {
		fmt.Print("Enter a number: ")
		line, readErr := stdinReader.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return -1, readErr
		}
		line = strings.TrimSpace(line)

		if line == "" && readErr == io.EOF {
			return -1, errors.New("no input received")
		}

		n, err := strconv.Atoi(line)
		if err != nil || n < 1 || n > len(options) {
			fmt.Printf("enter a number between 1 and %d\n", len(options))
			continue
		}
		return n - 1, nil
	}
}
