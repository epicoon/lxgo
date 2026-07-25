package query

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

/** @interface IQueryBuilder */

// QueryBuilder is the default IQueryBuilder implementation - obtained via
// IBaseRepo.QueryBuilder(), not constructed directly.
type QueryBuilder[T any] struct {
	repo  IBaseRepo[T]
	root  Node
	alias string

	preloads []preloadItem

	distinct bool
	groupBy  []string
	orders   []orderClause
	having   Node

	limit  int
	offset int
}

var _ IQueryBuilder[any] = (*QueryBuilder[any])(nil)

/** @constructor */

// NewQueryBuilder creates a QueryBuilder for repo's model T - normally
// reached via IBaseRepo.QueryBuilder() instead of calling this directly.
func NewQueryBuilder[T any](repo IBaseRepo[T]) IQueryBuilder[T] {
	table := tableName[T]()
	return &QueryBuilder[T]{
		repo:  repo,
		alias: table,
	}
}

// DB returns the underlying *gorm.DB query built so far (aliased to T's
// table name).
func (qb *QueryBuilder[T]) DB() *gorm.DB {
	table := tableName[T]()
	return qb.repo.DB().Model(new(T)).Table(table + " as " + qb.alias)
}

// Count returns the number of rows matching the query built so far.
func (qb *QueryBuilder[T]) Count() (uint64, error) {
	var cnt int64
	err := qb.build().Count(&cnt).Error
	return uint64(cnt), err
}

// All runs the query built so far and returns the matching rows.
func (qb *QueryBuilder[T]) All() ([]*T, error) {
	var items []*T
	err := qb.build().Find(&items).Error
	return items, err
}

// With preloads relation (GORM's Preload) - an optional scope narrows or
// modifies the preloaded relation itself.
func (qb *QueryBuilder[T]) With(relation string, scope ...func(*gorm.DB) *gorm.DB) IQueryBuilder[T] {
	item := preloadItem{
		relation: relation,
	}

	if len(scope) > 0 {
		item.scope = scope[0]
	}

	qb.preloads = append(qb.preloads, item)
	return qb
}

// Where sets (replacing any previous one) the WHERE condition tree.
func (qb *QueryBuilder[T]) Where(n Node) IQueryBuilder[T] {
	qb.root = n
	return qb
}

// AndWhere ANDs n onto whatever Where already set (or sets it, if none yet).
func (qb *QueryBuilder[T]) AndWhere(n Node) IQueryBuilder[T] {
	if qb.root == nil {
		qb.root = n
		return qb
	}

	qb.root = And(qb.root, n)
	return qb
}

// Or ORs n onto whatever Where already set (or sets it, if none yet).
func (qb *QueryBuilder[T]) Or(n Node) IQueryBuilder[T] {
	if qb.root == nil {
		qb.root = n
		return qb
	}
	qb.root = Or(qb.root, n)
	return qb
}

// Distinct adds SQL DISTINCT.
func (qb *QueryBuilder[T]) Distinct() IQueryBuilder[T] {
	qb.distinct = true
	return qb
}

// GroupBy adds a GROUP BY clause on fields.
func (qb *QueryBuilder[T]) GroupBy(fields ...string) IQueryBuilder[T] {
	qb.groupBy = append(qb.groupBy, fields...)
	return qb
}

// Having sets the HAVING condition tree (evaluated after GroupBy).
func (qb *QueryBuilder[T]) Having(n Node) IQueryBuilder[T] {
	qb.having = n
	return qb
}

// OrderBy adds a sort on field, descending if desc is true.
func (qb *QueryBuilder[T]) OrderBy(field string, desc bool) IQueryBuilder[T] {
	qb.orders = append(qb.orders, orderClause{field, desc})
	return qb
}

// OrderAsc is a shorthand for OrderBy(field, false).
func (qb *QueryBuilder[T]) OrderAsc(field string) IQueryBuilder[T] {
	return qb.OrderBy(field, false)
}

// OrderDesc is a shorthand for OrderBy(field, true).
func (qb *QueryBuilder[T]) OrderDesc(field string) IQueryBuilder[T] {
	return qb.OrderBy(field, true)
}

// PerPage sets the page size (SQL LIMIT) - call before Page.
func (qb *QueryBuilder[T]) PerPage(n int) IQueryBuilder[T] {
	qb.limit = n
	return qb
}

// Page sets the 1-based page number (SQL OFFSET, computed from the size set
// via PerPage) - has no effect if PerPage wasn't called first, or if p <= 1.
func (qb *QueryBuilder[T]) Page(p int) IQueryBuilder[T] {
	if p > 1 {
		qb.offset = (p - 1) * qb.limit
	}
	return qb
}

// SubQuery wraps a *gorm.DB to use as a nested query - pass to In/Exists.
type SubQuery struct {
	Query *gorm.DB
}

func (s *SubQuery) compile(*compiler) (string, []any) {
	return "(?)", []any{s.Query}
}

/* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
 * PRIVATE
 * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * */

type preloadItem struct {
	relation string
	scope    func(*gorm.DB) *gorm.DB
}

type orderClause struct {
	field string
	desc  bool
}

func (qb *QueryBuilder[T]) build() *gorm.DB {
	db := qb.DB()
	comp := newCompiler(qb.alias)

	// WHERE
	if qb.root != nil {
		sql, args := comp.compileNode(qb.root)
		db = db.Where(sql, args...)
	}

	// JOINS
	for rel, alias := range comp.joins {
		db = db.Joins(
			fmt.Sprintf(
				"LEFT JOIN %ss %s ON %s.%s_id = %s.id",
				toSnake(rel),
				alias,
				qb.alias,
				toSnake(rel),
				alias,
			),
		)
	}

	// PRELOADS (nested supported automatically by GORM)
	for _, p := range qb.preloads {
		if p.scope != nil {
			db = db.Preload(p.relation, p.scope)
		} else {
			db = db.Preload(p.relation)
		}
	}

	// DISTINCT
	if qb.distinct {
		db = db.Distinct()
	}

	// GROUP BY
	if len(qb.groupBy) > 0 {
		db = db.Group(strings.Join(qb.groupBy, ","))
	}

	// HAVING
	if qb.having != nil {
		sql, args := qb.having.compile(comp)
		db = db.Having(sql, args...)
	}

	// ORDER
	for _, o := range qb.orders {
		col := comp.column(o.field)
		if o.desc {
			db = db.Order(col + " DESC")
		} else {
			db = db.Order(col + " ASC")
		}
	}

	// LIMIT
	if qb.limit > 0 {
		db = db.Limit(qb.limit).Offset(qb.offset)
	}

	return db
}
