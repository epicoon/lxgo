package query

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

/** @interface IQueryBuilder */
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
func NewQueryBuilder[T any](repo IBaseRepo[T]) IQueryBuilder[T] {
	table := tableName[T]()
	return &QueryBuilder[T]{
		repo:  repo,
		alias: table,
	}
}

func (qb *QueryBuilder[T]) DB() *gorm.DB {
	table := tableName[T]()
	return qb.repo.DB().Model(new(T)).Table(table + " as " + qb.alias)
}

func (qb *QueryBuilder[T]) Count() (uint64, error) {
	var cnt int64
	err := qb.build().Count(&cnt).Error
	return uint64(cnt), err
}

func (qb *QueryBuilder[T]) All() ([]*T, error) {
	var items []*T
	err := qb.build().Find(&items).Error
	return items, err
}

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

func (qb *QueryBuilder[T]) Where(n Node) IQueryBuilder[T] {
	qb.root = n
	return qb
}

func (qb *QueryBuilder[T]) AndWhere(n Node) IQueryBuilder[T] {
	if qb.root == nil {
		qb.root = n
		return qb
	}

	qb.root = And(qb.root, n)
	return qb
}

func (qb *QueryBuilder[T]) Or(n Node) IQueryBuilder[T] {
	if qb.root == nil {
		qb.root = n
		return qb
	}
	qb.root = Or(qb.root, n)
	return qb
}

func (qb *QueryBuilder[T]) Distinct() IQueryBuilder[T] {
	qb.distinct = true
	return qb
}

func (qb *QueryBuilder[T]) GroupBy(fields ...string) IQueryBuilder[T] {
	qb.groupBy = append(qb.groupBy, fields...)
	return qb
}

func (qb *QueryBuilder[T]) Having(n Node) IQueryBuilder[T] {
	qb.having = n
	return qb
}

func (qb *QueryBuilder[T]) OrderBy(field string, desc bool) IQueryBuilder[T] {
	qb.orders = append(qb.orders, orderClause{field, desc})
	return qb
}

func (qb *QueryBuilder[T]) OrderAsc(field string) IQueryBuilder[T] {
	return qb.OrderBy(field, false)
}

func (qb *QueryBuilder[T]) OrderDesc(field string) IQueryBuilder[T] {
	return qb.OrderBy(field, true)
}

func (qb *QueryBuilder[T]) PerPage(n int) IQueryBuilder[T] {
	qb.limit = n
	return qb
}

func (qb *QueryBuilder[T]) Page(p int) IQueryBuilder[T] {
	if p > 1 {
		qb.offset = (p - 1) * qb.limit
	}
	return qb
}

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
