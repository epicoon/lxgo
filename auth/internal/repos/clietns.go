package repos

import (
	"errors"
	"fmt"

	cvn "github.com/epicoon/lxgo/auth/internal/conventions"
	"github.com/epicoon/lxgo/auth/internal/models"
	"github.com/epicoon/lxgo/kernel/utils"
	"gorm.io/gorm"
)

var ErrClientNotFound = errors.New("client not found")

/** @interface cvn.ITokensRepo */
type ClietnsRepo struct {
	*AbstractRepo
}

/** @constructor */
func NewClientsRepo(app cvn.IApp) *ClietnsRepo {
	return &ClietnsRepo{AbstractRepo: &AbstractRepo{app: app}}
}

func (repo *ClietnsRepo) CheckIDExists(id uint) bool {
	var count int64
	repo.DB().Model(&models.Client{}).Where("id = ?", id).Count(&count)
	return count > 0
}

func (repo *ClietnsRepo) CheckExists(id uint, secret string) bool {
	var client models.Client

	result := repo.DB().Where("id = ?", id).First(&client)
	if result.Error != nil || result.RowsAffected == 0 {
		return false
	}

	return utils.CheckHash(secret, client.Secret)
}

func (repo *ClietnsRepo) Create(client *models.Client) (string, error) {
	secret := client.Secret
	if secret == "" {
		secret = client.GenSecret(16)
	}

	secretHash, err := utils.GenHash(secret)
	if err != nil {
		return "", fmt.Errorf("can not create client: %s", err)
	}
	client.Secret = secretHash

	db := repo.DB()
	db.Save(client)

	return secret, nil
}

func (repo *ClietnsRepo) FindByID(id uint) (*models.Client, error) {
	client := models.Client{Model: gorm.Model{ID: id}}

	result := repo.DB().First(&client)
	if result.RowsAffected == 0 {
		return nil, ErrClientNotFound
	}
	if result.Error != nil {
		return nil, fmt.Errorf("can not find client id=%d: %s", id, result.Error)
	}

	return &client, nil
}

func (repo *ClietnsRepo) FindOne(id uint, secret string) (*models.Client, error) {
	var clients []models.Client

	result := repo.DB().Preload("Role").Where("id = ?", id).Find(&clients)
	if result.Error != nil {
		return nil, fmt.Errorf("can not find client id=%d: %s", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, ErrClientNotFound
	}

	for _, client := range clients {
		if utils.CheckHash(secret, client.Secret) {
			return &client, nil
		}
	}

	return nil, fmt.Errorf("client with id=%d and secret=%s not found", id, secret)
}
