package repos

import (
	"errors"
	"fmt"

	cvn "github.com/epicoon/lxgo/auth/internal/conventions"
	"github.com/epicoon/lxgo/auth/internal/models"
	"gorm.io/gorm"
)

var ErrTokenNotFound = errors.New("token not found")
var ErrTokensNotFound = errors.New("tokens not found")
var ErrRefreshTokenExpired = errors.New("refresh token expired")

/** @interface cvn.ITokensRepo */
type TokensRepo struct {
	*AbstractRepo
}

/** @constructor */
func NewTokensRepo(app cvn.IApp) *TokensRepo {
	return &TokensRepo{AbstractRepo: &AbstractRepo{app: app}}
}

func (repo *TokensRepo) CreateAccessToken(client *models.Client, user *models.User) (*models.Token, error) {
	token := &models.Token{
		ClientID:  client.ID,
		UserID:    user.ID,
		IsRefresh: false,
	}
	token.Refresh(client)

	if err := repo.DB().Create(token).Error; err != nil {
		return nil, err
	}

	return token, nil
}

func (repo *TokensRepo) CreateRefreshToken(client *models.Client, user *models.User) (*models.Token, error) {
	token := &models.Token{
		ClientID:  client.ID,
		UserID:    user.ID,
		IsRefresh: true,
	}
	token.Refresh(client)

	if err := repo.DB().Create(token).Error; err != nil {
		return nil, err
	}

	return token, nil
}

func (repo *TokensRepo) FindAccessToken(client *models.Client, accessValue string) (*models.Token, error) {
	var accessToken models.Token

	err := repo.DB().
		Where("client_id = ? AND value = ? AND is_refresh = ?", client.ID, accessValue, false).
		First(&accessToken).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}

	return &accessToken, nil
}

func (repo *TokensRepo) FindTokens(client *models.Client, accessValue string) (accessToken, refreshToken *models.Token, err error) {
	result := repo.DB().
		Where("client_id = ? AND value = ? AND is_refresh = FALSE", client.ID, accessValue).
		First(&accessToken)
	if result.RowsAffected == 0 {
		err = ErrTokensNotFound
		return
	}
	if result.Error != nil {
		err = fmt.Errorf("can not find access token '%s' for client[%v]: %s", accessValue, client.ID, result.Error)
		return
	}

	result = repo.DB().
		Where("client_id = ? AND user_id = ? AND is_refresh = TRUE", client.ID, accessToken.UserID).
		First(&refreshToken)
	if result.Error != nil {
		err = fmt.Errorf("can not find refresh token for client[%v] with access token '%s': %s", client.ID, accessValue, result.Error)
		return
	}
	if result.RowsAffected == 0 {
		err = ErrTokensNotFound
		return
	}

	return
}

func (repo *TokensRepo) FindTokensByRefresh(client *models.Client, refreshValue string) (accessToken, refreshToken *models.Token, err error) {
	result := repo.DB().
		Where("client_id = ? AND value = ? AND is_refresh = TRUE", client.ID, refreshValue).
		Find(&refreshToken)
	if result.Error != nil {
		err = fmt.Errorf("can not find refresh token for client[%v] with value '%s': %s", client.ID, refreshValue, result.Error)
		return
	}
	if result.RowsAffected == 0 {
		err = ErrTokensNotFound
		return
	}
	if result.RowsAffected != 1 {
		err = fmt.Errorf("wrong number of refresh tokens found for client[%v] with value '%s': %v", client.ID, refreshValue, result.RowsAffected)
		return
	}
	if refreshToken.IsExpired() {
		err = ErrRefreshTokenExpired
		return
	}

	result = repo.DB().
		Where("client_id = ? AND is_refresh = FALSE", client.ID).
		Find(&accessToken)
	if result.Error != nil {
		err = fmt.Errorf("can not find access token for client[%v] with refresh-value '%s': %s", client.ID, refreshValue, result.Error)
		return
	}
	if result.RowsAffected == 0 {
		err = ErrTokensNotFound
		return
	}
	if result.RowsAffected != 1 {
		err = fmt.Errorf("wrong number of access tokens found for client[%v] with refresh-value '%s': %v", client.ID, refreshValue, result.RowsAffected)
		return
	}

	return
}

func (repo *TokensRepo) SaveTokens(accessToken, refreshToken *models.Token) error {
	db := repo.DB()

	result := db.Save(accessToken)
	if result.Error != nil {
		return fmt.Errorf("can not save access token: %s", result.Error)
	}

	result = db.Save(refreshToken)
	if result.Error != nil {
		return fmt.Errorf("can not save refresh token: %s", result.Error)
	}

	return nil
}

func (repo *TokensRepo) DropTokens(accessToken, refreshToken *models.Token) error {
	db := repo.DB()
	if err := db.Delete(accessToken).Error; err != nil {
		return err
	}
	if err := db.Delete(refreshToken).Error; err != nil {
		return err
	}
	return nil
}

func (repo *TokensRepo) DropTokensByUser(client *models.Client, user *models.User) error {
	db := repo.DB()
	result := db.Where("client_id = ? AND user_id = ?", client.ID, user.ID).Delete(&models.Token{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}
