package conventions

import (
	"github.com/epicoon/lxgo/auth/internal/models"
	"gorm.io/gorm"
)

type IRepo interface {
	App() IApp
	SetTx(tx *gorm.DB)
	DB() *gorm.DB
}

type IUsersRepo interface {
	IRepo
	CheckExists(login string) bool
	Create(login string, password string) (*models.User, error)
	FindByID(id uint) (*models.User, error)
	FindByLP(login string, password string) (*models.User, error)
	FindByToken(accessToken *models.Token) (*models.User, error)
	FindData(user *models.User, client *models.Client) (*models.UserData, error)
	SetData(user *models.User, client *models.Client, data models.JSONB) (*models.UserData, error)
}

type IClientsRepo interface {
	IRepo
	CheckIDExists(id uint) bool
	CheckExists(id uint, secret string) bool
	Create(client *models.Client) (string, error)
	Delete(id uint) error
	FindByID(id uint) (*models.Client, error)
	FindOne(id uint, secret string) (*models.Client, error)
}

type IAdminsRepo interface {
	IRepo
	Create(userID uint, roleID uint) (*models.Admin, error)
	FindByUserID(userID uint) (*models.Admin, error)
}

type ICodesRepo interface {
	IRepo
	Create(clientID uint, userID uint, scope string) (*models.Code, error)
	Delete(code *models.Code) error
	FindByValue(value string) (*models.Code, error)
}

type ITokensRepo interface {
	IRepo
	CreateAccessToken(client *models.Client, user *models.User, scope string) (*models.Token, error)
	CreateRefreshToken(client *models.Client, user *models.User, scope string) (*models.Token, error)
	FindAccessToken(client *models.Client, accessValue string) (accessToken *models.Token, err error)
	FindTokens(client *models.Client, accessValue string) (accessToken, refreshToken *models.Token, err error)
	FindTokensByRefresh(client *models.Client, refreshValue string) (accessToken, refreshToken *models.Token, err error)
	SaveTokens(accessToken, refreshToken *models.Token) error
	DropTokens(accessToken, refreshToken *models.Token) error
	DropTokensByUser(client *models.Client, user *models.User) error
}
