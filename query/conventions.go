package query

import "gorm.io/gorm"

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * MODELS
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */
type IBaseRepo[T any] interface {
	DB() *gorm.DB

	QueryBuilder() IQueryBuilder[T]

	Count() (uint64, error)
	Create(entity *T) error
	CreateFromMap(m map[string]any) (*T, error)

	ExistsByID(ID uint64) (bool, error)
	ReadByID(ID uint64) (*T, error)
	ReadByIDs(IDs []uint64) ([]*T, error)
	ReadBy(field string, value any) ([]*T, error)
	ReadWhere(conditions map[string]any) ([]*T, error)
	ReadAll() ([]*T, error)

	Update(entity *T) error
	UpdateByID(ID uint64, entity *T) error
	UpdateFromMap(ID uint64, m map[string]any) error

	DeleteByID(ID uint64) error
	ForceDeleteByID(ID uint64) error
}

type IRepoTx interface {
	SetTx(tx *gorm.DB)
	Tx() *gorm.DB
	SyncTx(r IRepoTx)
	DB() *gorm.DB
}

type IQueryBuilder[T any] interface {
	DB() *gorm.DB
	Count() (uint64, error)
	All() ([]*T, error)
	With(relation string, scope ...func(*gorm.DB) *gorm.DB) IQueryBuilder[T]
	Where(n Node) IQueryBuilder[T]
	AndWhere(n Node) IQueryBuilder[T]
	Or(n Node) IQueryBuilder[T]
	Distinct() IQueryBuilder[T]
	GroupBy(fields ...string) IQueryBuilder[T]
	Having(n Node) IQueryBuilder[T]
	OrderBy(field string, desc bool) IQueryBuilder[T]
	OrderAsc(field string) IQueryBuilder[T]
	OrderDesc(field string) IQueryBuilder[T]
	PerPage(n int) IQueryBuilder[T]
	Page(p int) IQueryBuilder[T]
}
