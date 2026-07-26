package query

import "gorm.io/gorm"

// IBaseRepo is the common CRUD contract every BaseRepo[T] satisfies -
// see NewBaseRepo.
type IBaseRepo[T any] interface {
	// DB returns the connection to run queries on - the active transaction
	// if IRepoTx.SetTx was called, the plain connection otherwise.
	DB() *gorm.DB

	// QueryBuilder starts a query beyond what the other methods here cover
	// (arbitrary condition trees, joins by relation, ordering/grouping/...).
	QueryBuilder() IQueryBuilder[T]

	// Count returns the total number of rows for T (no filtering - use
	// QueryBuilder().Count() to filter).
	Count() (uint64, error)
	// Create inserts entity.
	Create(entity *T) error
	// CreateFromMap builds a T from m (see cast.MapToStruct) and inserts it.
	CreateFromMap(m map[string]any) (*T, error)

	// ExistsByID reports whether a row with this ID exists.
	ExistsByID(ID uint64) (bool, error)
	// ReadByID fetches a single row by ID.
	ReadByID(ID uint64) (*T, error)
	// ReadByIDs fetches every row whose ID is in IDs (nil, nil if IDs is empty).
	ReadByIDs(IDs []uint64) ([]*T, error)
	// ReadBy fetches every row where field equals value.
	ReadBy(field string, value any) ([]*T, error)
	// ReadWhere fetches every row matching all of conditions (field: value,
	// ANDed together).
	ReadWhere(conditions map[string]any) ([]*T, error)
	// ReadAll fetches every row.
	ReadAll() ([]*T, error)

	// Update saves every field of entity - entity must already have its ID set.
	Update(entity *T) error
	// UpdateByID updates the row with this ID from entity's fields.
	UpdateByID(ID uint64, entity *T) error
	// UpdateFromMap updates the row with this ID from m (only the given keys).
	UpdateFromMap(ID uint64, m map[string]any) error

	// DeleteByID soft-deletes the row with this ID (sets DeletedAt).
	DeleteByID(ID uint64) error
	// ForceDeleteByID permanently deletes the row with this ID, bypassing
	// soft-delete.
	ForceDeleteByID(ID uint64) error
}

// IRepoTx lets a repo run its queries inside a shared transaction instead of
// the plain DB connection - see the "Transaction example" in the README.
type IRepoTx interface {
	// SetTx makes every subsequent query run against tx instead of the plain
	// connection.
	SetTx(tx *gorm.DB)
	// Tx returns the transaction set via SetTx, or nil if none was set.
	Tx() *gorm.DB
	// SyncTx adopts r's transaction (equivalent to SetTx(r.Tx())) - useful to
	// keep several repos on the same transaction.
	SyncTx(r IRepoTx)
	// DB returns the connection to run queries on - the active transaction
	// if SetTx was called, the plain connection otherwise.
	DB() *gorm.DB
}

// IQueryBuilder builds a query beyond IBaseRepo's fixed methods - arbitrary
// condition trees (see Node/And/Or/Eq/...), joins by relation name, JSONB
// fields, ordering, grouping, and pagination. Obtained via
// IBaseRepo.QueryBuilder(), not constructed directly.
type IQueryBuilder[T any] interface {
	// DB returns the underlying *gorm.DB query built so far (aliased to T's
	// table name).
	DB() *gorm.DB
	// Count returns the number of rows matching the query built so far.
	Count() (uint64, error)
	// All runs the query built so far and returns the matching rows.
	All() ([]*T, error)
	// With preloads relation (GORM's Preload) - an optional scope narrows or
	// modifies the preloaded relation itself.
	With(relation string, scope ...func(*gorm.DB) *gorm.DB) IQueryBuilder[T]
	// Where sets (replacing any previous one) the WHERE condition tree.
	Where(n Node) IQueryBuilder[T]
	// AndWhere ANDs n onto whatever Where already set (or sets it, if none yet).
	AndWhere(n Node) IQueryBuilder[T]
	// Or ORs n onto whatever Where already set (or sets it, if none yet).
	Or(n Node) IQueryBuilder[T]
	// Distinct adds SQL DISTINCT.
	Distinct() IQueryBuilder[T]
	// GroupBy adds a GROUP BY clause on fields.
	GroupBy(fields ...string) IQueryBuilder[T]
	// Having sets the HAVING condition tree (evaluated after GroupBy).
	Having(n Node) IQueryBuilder[T]
	// OrderBy adds a sort on field, descending if desc is true.
	OrderBy(field string, desc bool) IQueryBuilder[T]
	// OrderAsc is a shorthand for OrderBy(field, false).
	OrderAsc(field string) IQueryBuilder[T]
	// OrderDesc is a shorthand for OrderBy(field, true).
	OrderDesc(field string) IQueryBuilder[T]
	// PerPage sets the page size (SQL LIMIT) - call before Page.
	PerPage(n int) IQueryBuilder[T]
	// Page sets the 1-based page number (SQL OFFSET, computed from the size
	// set via PerPage) - has no effect if PerPage wasn't called first, or if
	// p <= 1.
	Page(p int) IQueryBuilder[T]
}
