package core

import (
	"time"

	cvn "github.com/epicoon/lxgo/auth/internal/conventions"
	"github.com/epicoon/lxgo/auth/internal/repos"
	"github.com/epicoon/lxgo/kernel"
	lxApp "github.com/epicoon/lxgo/kernel/app"
	"github.com/epicoon/lxgo/kernel/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

/** @interface kernel.IApp */
type App struct {
	*lxApp.App
	gormConn *gorm.DB
}

/** @constructor */
func NewApp() cvn.IApp {
	return &App{App: lxApp.NewApp()}
}

func (app *App) Settings() *kernel.Config {
	s, err := config.GetParam[kernel.Config](app.Config(), "Settings")
	if err != nil {
		return &kernel.Config{}
	}
	return &s
}

func (app *App) Gorm() *gorm.DB {
	if app.gormConn == nil {
		gormDB, err := gorm.Open(postgres.New(postgres.Config{
			Conn: app.Connection().DB(),
		}), &gorm.Config{
			NowFunc: func() time.Time {
				return time.Now().UTC()
			},
		})
		if err != nil {
			panic("can not connect to gorm")
		}
		app.gormConn = gormDB
	}

	return app.gormConn
}

func (app *App) ClientsRepo() cvn.IClientsRepo {
	return repos.NewClientsRepo(app)
}

func (app *App) UsersRepo() cvn.IUsersRepo {
	return repos.NewUsersRepo(app)
}

func (app *App) CodesRepo() cvn.ICodesRepo {
	return repos.NewCodesRepo(app)
}

func (app *App) TokensRepo() cvn.ITokensRepo {
	return repos.NewTokensRepo(app)
}
