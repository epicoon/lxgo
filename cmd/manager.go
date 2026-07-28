package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type manager struct {
	list     map[string]CCommand
	cmdRoute string
	cmdName  string
	subName  string
	params   map[string]any
}

var m = new(manager)

func (m *manager) prepare() {
	m.parseArgs(os.Args[1:])
}

// parseArgs fills in cmdRoute/cmdName/subName/params from args.
func (m *manager) parseArgs(args []string) {
	m.cmdRoute = ""
	m.cmdName = ""
	m.subName = ""

	if len(args) == 0 {
		return
	}

	m.cmdRoute = args[0]
	parts := strings.SplitN(m.cmdRoute, ":", 2)
	m.cmdName = parts[0]
	if len(parts) > 1 {
		m.subName = parts[1]
	}

	params := make(map[string]any)
	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "--") {
			parts := strings.SplitN(arg[2:], "=", 2)
			key := parts[0]
			if len(parts) > 1 {
				params[key] = parts[1]
			} else {
				params[key] = true
			}
		} else if strings.HasPrefix(arg, "-") {
			flags, _ := strings.CutPrefix(arg, "-")
			for _, l := range flags {
				params[string(l)] = true
			}
		}
	}

	m.params = params
}

func (m *manager) defineConstructor() (CCommand, error) {
	_, exists := m.list[m.cmdName]
	if !exists {
		if m.cmdName == "" {
			return nil, errors.New("undefined default command")
		} else {
			return nil, fmt.Errorf("undefined command '%s'", m.cmdName)
		}
	}

	return m.list[m.cmdName], nil
}
