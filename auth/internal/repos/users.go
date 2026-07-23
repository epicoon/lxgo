package repos

import (
	"errors"
	"fmt"

	cvn "github.com/epicoon/lxgo/auth/internal/conventions"
	"github.com/epicoon/lxgo/auth/internal/models"
	"github.com/epicoon/lxgo/kernel/utils"
	"gorm.io/gorm"
)

var ErrUserNotFound = errors.New("user not found")
var ErrUserAlreadyExists = errors.New("user with this login already exists")

/** @interface cvn.ITokensRepo */
type UsersRepo struct {
	*AbstractRepo
}

/** @constructor */
func NewUsersRepo(app cvn.IApp) *UsersRepo {
	return &UsersRepo{AbstractRepo: &AbstractRepo{app: app}}
}

func (repo *UsersRepo) CheckExists(login string) bool {

	//TODO do we need this method?

	return false
}

func (repo *UsersRepo) Create(login string, password string) (*models.User, error) {
	// Hashing Password
	pwdHash, err := utils.GenHash(password)
	if err != nil {
		return nil, fmt.Errorf("can not generate hash for password='%s'", password)
	}

	db := repo.DB()

	var existingUser models.User
	err = db.Where("login = ?", login).First(&existingUser).Error
	if err == nil {
		return nil, ErrUserAlreadyExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	newUser := models.User{
		Login:    login,
		Password: pwdHash,
	}

	if err := db.Create(&newUser).Error; err != nil {
		return nil, err
	}
	return &newUser, nil
}

func (repo *UsersRepo) FindByID(id uint) (*models.User, error) {
	db := repo.DB()
	user := &models.User{}
	result := db.First(user, id)
	if result.RowsAffected == 0 {
		return nil, ErrUserNotFound
	}
	if result.Error != nil {
		return nil, fmt.Errorf("error occurred while finding user with id=%d: %s", id, result.Error)
	}

	return user, nil
}

func (repo *UsersRepo) FindByLP(login string, password string) (*models.User, error) {
	db := repo.DB()
	var users []models.User

	result := db.Where("login = ?", login).Find(&users)
	if result.RowsAffected > 1 {
		errStr := fmt.Sprintf("more than one user found for login '%s'", login)
		repo.app.LogError(errStr, "App")
		return nil, errors.New(errStr)
	}
	if result.Error != nil {
		errStr := fmt.Sprintf("error occurred while finding user '%s': %s", login, result.Error)
		repo.app.LogError(errStr, "App")
		return nil, errors.New(errStr)
	}
	//TODO use ErrUserNotFound
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("user '%s' not found", login)
	}

	user := users[0]
	if !utils.CheckHash(password, user.Password) {
		return nil, fmt.Errorf("wrong password for user '%s'", login)
	}

	return &user, nil
}

func (repo *UsersRepo) FindByToken(accessToken *models.Token) (*models.User, error) {
	user := models.User{Model: gorm.Model{ID: accessToken.UserID}}

	result := repo.DB().First(&user)
	if result.Error != nil {
		return nil, fmt.Errorf("can not find user id=%d: %s", accessToken.UserID, result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, ErrUserNotFound
	}

	return &user, nil
}

func (repo *UsersRepo) FindData(user *models.User, client *models.Client) (*models.UserData, error) {
	db := repo.DB()
	var data models.UserData

	result := db.Where("user_id = ? AND client_id = ?", user.ID, client.ID).First(&data)
	if result.RowsAffected == 0 {
		return nil, nil
	}
	if result.Error != nil {
		return nil, fmt.Errorf("error occurred while finding user data for user_id=%d, client_id=%d: %s", user.ID, client.ID, result.Error)
	}

	return &data, nil
}

func (repo *UsersRepo) SetData(user *models.User, client *models.Client, data models.JSONB) (*models.UserData, error) {
	db := repo.DB()
	var existing models.UserData

	result := db.Where("user_id = ? AND client_id = ?", user.ID, client.ID).First(&existing)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("error occurred while finding user data for user_id=%d, client_id=%d: %s", user.ID, client.ID, result.Error)
	}

	if result.RowsAffected == 0 {
		existing = models.UserData{
			UserID:   user.ID,
			ClientID: client.ID,
			Data:     data,
		}
		if err := db.Create(&existing).Error; err != nil {
			return nil, fmt.Errorf("can not create user data for user_id=%d, client_id=%d: %s", user.ID, client.ID, err)
		}
		return &existing, nil
	}

	existing.Data = data
	if err := db.Save(&existing).Error; err != nil {
		return nil, fmt.Errorf("can not update user data for user_id=%d, client_id=%d: %s", user.ID, client.ID, err)
	}

	return &existing, nil
}
