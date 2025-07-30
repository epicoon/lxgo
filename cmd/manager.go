package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type manager struct {
	list     map[string]FConstructor
	cmdRoute string
	cmdName  string
	subName  string
	params   map[string]any
}

var m = new(manager)

func (m *manager) prepare() {
	args := os.Args[1:]

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
		}
	}

	m.params = params
}

func (m *manager) defineConstructor() (FConstructor, error) {
	_, exists := m.list[m.cmdName]
	if !exists {
		if m.cmdName == "" {
			return nil, errors.New("undefind default command")
		} else {
			return nil, fmt.Errorf("undefind command '%s'", m.cmdName)
		}
	}

	return m.list[m.cmdName], nil
}
