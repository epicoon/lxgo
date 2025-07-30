package repos

import (
	cvn "github.com/epicoon/lxgo/auth/internal/conventions"
	"gorm.io/gorm"
)

type AbstractRepo struct {
	app cvn.IApp
	tx  *gorm.DB
}

func (repo *AbstractRepo) App() cvn.IApp {
	return repo.app
}

func (repo *AbstractRepo) SetTx(tx *gorm.DB) {
	repo.tx = tx
}

func (repo *AbstractRepo) DB() *gorm.DB {
	if repo.tx == nil {
		return repo.app.Gorm()
	}
	return repo.tx
}
