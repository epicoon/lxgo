package query

import (
	"github.com/epicoon/lxgo/kernel/conv"
	"gorm.io/gorm"
)

/** @interface IBaseRepo */
/** @interface IRepoTx */

// BaseRepo is the default IBaseRepo/IRepoTx implementation - embed it (or
// use it directly) for a generic CRUD repo over T, backed by GORM. See
// NewBaseRepo.
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

// NewBaseRepo creates a BaseRepo for model T. allowed is currently unused by
// BaseRepo itself - it's kept for repos embedding BaseRepo that want to
// restrict which fields a caller may filter/update by.
func NewBaseRepo[T any](db *gorm.DB, allowed []string) *BaseRepo[T] {
	m := make(map[string]bool)
	for _, f := range allowed {
		m[f] = true
	}
	return &BaseRepo[T]{db: db, allowedFields: m}
}

// SetTx makes every subsequent query run against tx instead of the plain
// connection.
func (repo *BaseRepo[T]) SetTx(tx *gorm.DB) {
	repo.tx = tx
}

// Tx returns the transaction set via SetTx, or nil if none was set.
func (repo *BaseRepo[T]) Tx() *gorm.DB {
	return repo.tx
}

// SyncTx adopts r's transaction (equivalent to SetTx(r.Tx())).
func (repo *BaseRepo[T]) SyncTx(r IRepoTx) {
	repo.SetTx(r.Tx())
}

// DB returns the connection to run queries on - the active transaction if
// SetTx was called, the plain connection otherwise.
func (repo *BaseRepo[T]) DB() *gorm.DB {
	if repo.tx == nil {
		return repo.db
	}
	return repo.tx
}

// QueryBuilder starts a query beyond what BaseRepo's other methods cover.
func (r *BaseRepo[T]) QueryBuilder() IQueryBuilder[T] {
	return NewQueryBuilder(r)
}

// Count returns the total number of rows for T (no filtering).
func (r *BaseRepo[T]) Count() (uint64, error) {
	var cnt int64
	err := r.DB().Model(new(T)).Count(&cnt).Error
	return uint64(cnt), err
}

// Create inserts entity.
func (r *BaseRepo[T]) Create(entity *T) error {
	return r.DB().Create(entity).Error
}

// CreateFromMap builds a T from m and inserts it.
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

// ExistsByID reports whether a row with this ID exists.
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

// ReadByID fetches a single row by ID.
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

// ReadByIDs fetches every row whose ID is in IDs (nil, nil if IDs is empty).
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

// ReadBy fetches every row where field equals value.
//
// Example:
//
//	repo.ReadBy("region", "EU")
//	repo.ReadBy("status", 1)
func (r *BaseRepo[T]) ReadBy(field string, value any) ([]*T, error) {
	var items []*T

	err := r.DB().
		Model(new(T)).
		Where(field+" = ?", value).
		Find(&items).
		Error

	return items, err
}

// ReadWhere fetches every row matching all of conditions (field: value,
// ANDed together).
//
// Example:
//
//	repo.ReadWhere(map[string]any{
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

// ReadAll fetches every row.
func (r *BaseRepo[T]) ReadAll() ([]*T, error) {
	var items []*T
	err := r.DB().Find(&items).Error
	return items, err
}

// Update saves every field of entity - entity must already have its ID set.
func (r *BaseRepo[T]) Update(entity *T) error {
	return r.DB().
		Model(entity).
		Updates(entity).
		Error
}

// UpdateByID updates the row with this ID from entity's fields.
func (r *BaseRepo[T]) UpdateByID(ID uint64, entity *T) error {
	return r.DB().
		Model(new(T)).
		Where("id = ?", ID).
		Updates(entity).
		Error
}

// UpdateFromMap updates the row with this ID from m (only the given keys).
func (repo *BaseRepo[T]) UpdateFromMap(ID uint64, m map[string]any) error {
	result := repo.DB().Model(new(T)).
		Where("id = ?", ID).
		Updates(m)
	return result.Error
}

// DeleteByID soft-deletes the row with this ID (sets DeletedAt).
func (r *BaseRepo[T]) DeleteByID(ID uint64) error {
	return r.DB().Delete(new(T), ID).Error
}

// ForceDeleteByID permanently deletes the row with this ID, bypassing
// soft-delete.
func (r *BaseRepo[T]) ForceDeleteByID(ID uint64) error {
	return r.DB().Unscoped().Delete(new(T), ID).Error
}
