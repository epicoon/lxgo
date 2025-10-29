package reconf

import (
	"fmt"
	"net"
	"strings"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/config"
)

func Run(app kernel.IApp, conn net.Conn, test bool) {
	path := app.ConfigPath()
	if path == "" {
		conn.Write([]byte("Application configuration file path is unknown\n"))
		return
	}

	path = app.Pathfinder().GetAbsPath(path)
	newConf, err := config.Load(path)
	if err != nil {
		conn.Write(fmt.Appendf(nil, "can not read configuration file '%s'. Cause: %v\n", path, err))
		return
	}

	origConf := app.Config()
	diff := compareConfigs(origConf, newConf)

	if test {
		lines := []string{}
		if len(diff.errs) > 0 {
			lines = append(lines, "Errors:")
			for _, d := range diff.errs {
				lines = append(lines, fmt.Sprintf("* Param '%s': expected type (%s), passed type (%s), invalid value - %v", d.Path, d.OldType, d.NewType, d.New))
			}
		}
		if len(diff.changed) > 0 {
			lines = append(lines, "To be changed:")
			for _, d := range diff.changed {
				lines = append(lines, fmt.Sprintf("~ Param '%s': %v → %v", d.Path, d.Old, d.New))
			}
		}
		if len(diff.removed) > 0 {
			lines = append(lines, "To be removed:")
			for _, d := range diff.removed {
				lines = append(lines, fmt.Sprintf("- Param '%s': old value - %v", d.Path, d.Old))
			}
		}
		if len(diff.added) > 0 {
			lines = append(lines, "To be added:")
			for _, d := range diff.added {
				lines = append(lines, fmt.Sprintf("+ Param '%s': new value - %v", d.Path, d.New))
			}
		}

		var msg string
		if len(lines) == 0 {
			msg = "Nothing to change"
		} else {
			msg = strings.Join(lines, "\n")
		}

		conn.Write([]byte(msg))
		return
	}

	if len(diff.errs) > 0 {
		lines := []string{"Can not apply new config. Errors:"}
		for _, d := range diff.errs {
			lines = append(lines, fmt.Sprintf("* Param '%s': expected type (%s), passed type (%s), invalid value - %v", d.Path, d.OldType, d.NewType, d.New))
		}
		conn.Write([]byte(strings.Join(lines, "\n")))
		return
	}

	app.SetConfig(newConf)
	app.Events().Trigger(kernel.EVENT_CONFIG_REFRESHED)
	conn.Write([]byte("Done\n"))
}
