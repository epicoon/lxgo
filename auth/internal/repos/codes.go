package repos

import (
	"errors"
	"fmt"

	cvn "github.com/epicoon/lxgo/auth/internal/conventions"
	"github.com/epicoon/lxgo/auth/internal/models"
	"github.com/epicoon/lxgo/kernel/utils"
)

var ErrorCodeNotFound = errors.New("code not found")

/** @interface cvn.ITokensRepo */
type CodesRepo struct {
	*AbstractRepo
}

/** @constructor */
func NewCodesRepo(app cvn.IApp) *CodesRepo {
	return &CodesRepo{AbstractRepo: &AbstractRepo{app: app}}
}

func (repo *CodesRepo) Create(clientID uint, userID uint) (*models.Code, error) {
	code := &models.Code{
		ClientID: clientID,
		UserID:   userID,
		Value:    utils.GenRandomHash(16),
	}

	db := repo.DB()
	if err := db.Create(code).Error; err != nil {
		return nil, err
	}

	return code, nil
}

func (repo *CodesRepo) Delete(code *models.Code) error {
	db := repo.DB()
	if err := db.Delete(code).Error; err != nil {
		return err
	}

	return nil
}

func (repo *CodesRepo) FindByValue(value string) (*models.Code, error) {
	db := repo.DB()
	var codes []models.Code

	result := db.Where("value = ?", value).Find(&codes)
	if result.RowsAffected > 1 {
		errStr := fmt.Sprintf("more then one code '%s'", value)
		repo.app.LogError(errStr, "App")
		return nil, errors.New(errStr)
	}
	if result.RowsAffected == 0 {
		return nil, ErrorCodeNotFound
	}
	if result.Error != nil {
		errStr := fmt.Sprintf("error occured while finding code: %s", result.Error)
		repo.app.LogError(errStr, "App")
		return nil, errors.New(errStr)
	}

	return &codes[0], nil
}
