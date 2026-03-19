package query

import (
	"github.com/epicoon/lxgo/kernel/conv"
	"gorm.io/gorm"
)

/** @interface IBaseRepo */
/** @interface IRepoTx */
type BaseRepo[T any] struct {
	db *gorm.DB
	tx *gorm.DB

	allowedFields map[string]bool
}

var (
	_ IBaseRepo[any] = (*BaseRepo[any])(nil)
	_ IRepoTx        = (*BaseRepo[any])(nil)
)

/** @constructor */
func NewBaseRepo[T any](db *gorm.DB, allowed []string) *BaseRepo[T] {
	m := make(map[string]bool)
	for _, f := range allowed {
		m[f] = true
	}
	return &BaseRepo[T]{db: db, allowedFields: m}
}

func (repo *BaseRepo[T]) SetTx(tx *gorm.DB) {
	repo.tx = tx
}

func (repo *BaseRepo[T]) Tx() *gorm.DB {
	return repo.tx
}

func (repo *BaseRepo[T]) SyncTx(r IRepoTx) {
	repo.SetTx(r.Tx())
}

func (repo *BaseRepo[T]) DB() *gorm.DB {
	if repo.tx == nil {
		return repo.db
	}
	return repo.tx
}

func (r *BaseRepo[T]) QueryBuilder() IQueryBuilder[T] {
	return NewQueryBuilder(r)
}

func (r *BaseRepo[T]) Count() (uint64, error) {
	var cnt int64
	err := r.DB().Model(new(T)).Count(&cnt).Error
	return uint64(cnt), err
}

func (r *BaseRepo[T]) Create(entity *T) error {
	return r.DB().Create(entity).Error
}

func (r *BaseRepo[T]) CreateFromMap(m map[string]any) (*T, error) {
	var entity T
	if err := conv.MapToStruct(m, &entity); err != nil {
		return nil, err
	}

	if err := r.DB().Create(&entity).Error; err != nil {
		return nil, err
	}

	return &entity, nil
}

func (r *BaseRepo[T]) ExistsByID(ID uint64) (bool, error) {
	var cnt int64

	err := r.DB().
		Model(new(T)).
		Where("id = ?", ID).
		Count(&cnt).
		Error

	if err != nil {
		return false, err
	}

	return cnt > 0, nil
}

func (r *BaseRepo[T]) ReadByID(ID uint64) (*T, error) {
	var entity T

	err := r.DB().
		Model(new(T)).
		Where("id = ?", ID).
		First(&entity).
		Error

	if err != nil {
		return nil, err
	}

	return &entity, nil
}

func (repo *BaseRepo[T]) ReadByIDs(IDs []uint64) ([]*T, error) {
	if len(IDs) == 0 {
		return nil, nil
	}

	db := repo.DB()

	var stats []*T
	if err := db.
		Where("id IN ?", IDs).
		Find(&stats).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

// Examples:
//
//	repo.FindBy("region", "EU")
//	repo.FindBy("status", 1)
func (r *BaseRepo[T]) ReadBy(field string, value any) ([]*T, error) {
	var items []*T

	err := r.DB().
		Model(new(T)).
		Where(field+" = ?", value).
		Find(&items).
		Error

	return items, err
}

// Example:
//
//	repo.FindWhere(map[string]any{
//	    "region": "EU",
//	    "status": 1,
//	})
func (r *BaseRepo[T]) ReadWhere(conditions map[string]any) ([]*T, error) {
	var items []*T

	err := r.DB().
		Model(new(T)).
		Where(conditions).
		Find(&items).
		Error

	return items, err
}

func (r *BaseRepo[T]) ReadAll() ([]*T, error) {
	var items []*T
	err := r.DB().Find(&items).Error
	return items, err
}

// Only for entities with defined ID!!!
func (r *BaseRepo[T]) Update(entity *T) error {
	return r.DB().
		Model(entity).
		Updates(entity).
		Error
}

func (r *BaseRepo[T]) UpdateByID(ID uint64, entity *T) error {
	return r.DB().
		Model(new(T)).
		Where("id = ?", ID).
		Updates(entity).
		Error
}

func (repo *BaseRepo[T]) UpdateFromMap(ID uint64, m map[string]any) error {
	result := repo.DB().Model(new(T)).
		Where("id = ?", ID).
		Updates(m)
	return result.Error
}

func (r *BaseRepo[T]) DeleteByID(ID uint64) error {
	return r.DB().Delete(new(T), ID).Error
}

func (r *BaseRepo[T]) ForceDeleteByID(ID uint64) error {
	return r.DB().Unscoped().Delete(new(T), ID).Error
}
