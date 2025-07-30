package conventions

import (
	"github.com/epicoon/lxgo/kernel"
	"gorm.io/gorm"
)

type IApp interface {
	kernel.IApp
	Settings() *kernel.Config
	Gorm() *gorm.DB
	ClientsRepo() IClientsRepo
	UsersRepo() IUsersRepo
	CodesRepo() ICodesRepo
	TokensRepo() ITokensRepo
}
