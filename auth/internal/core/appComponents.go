package core

import (
	cvn "github.com/epicoon/lxgo/auth/internal/conventions"
	"github.com/epicoon/lxgo/session"
)

func setComponents(app cvn.IApp) error {
	if err := session.SetAppComponent(app, "Components.SessionStorage"); err != nil {
		return err
	}

	return nil
}
