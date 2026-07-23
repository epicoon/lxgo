package repos

import (
	"errors"
	"fmt"

	cvn "github.com/epicoon/lxgo/auth/internal/conventions"
	"github.com/epicoon/lxgo/auth/internal/models"
	"gorm.io/gorm"
)

var ErrAdminNotFound = errors.New("admin not found")

/** @interface cvn.IAdminsRepo */
type AdminsRepo struct {
	*AbstractRepo
}

/** @constructor */
func NewAdminsRepo(app cvn.IApp) *AdminsRepo {
	return &AdminsRepo{AbstractRepo: &AbstractRepo{app: app}}
}

func (repo *AdminsRepo) Create(userID uint, roleID uint) (*models.Admin, error) {
	admin := &models.Admin{
		UserID: userID,
		RoleID: roleID,
	}

	if err := repo.DB().Create(admin).Error; err != nil {
		return nil, fmt.Errorf("can not create admin for user_id=%d: %s", userID, err)
	}

	return admin, nil
}

func (repo *AdminsRepo) FindByUserID(userID uint) (*models.Admin, error) {
	var admin models.Admin

	result := repo.DB().Preload("Role").Where("user_id = ?", userID).First(&admin)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) || result.RowsAffected == 0 {
		return nil, ErrAdminNotFound
	}
	if result.Error != nil {
		return nil, fmt.Errorf("error occurred while finding admin for user_id=%d: %s", userID, result.Error)
	}

	return &admin, nil
}
