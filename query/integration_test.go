//go:build integration

package query_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/epicoon/lxgo/query"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Category and iUser are a minimal, deliberately irregular-plural/
// foreign-key-bearing pair of related models - iUser.CategoryID/Category
// exercise column/table naming (ID, a "XxxID" foreign key, a JOIN by
// relation whose table name doesn't just take a trailing "s") against a
// real, gorm-migrated Postgres schema, not just the package's own naming
// logic in isolation (see the unit tests for that).
type Category struct {
	query.BaseModel
	Name string
}

type iUser struct {
	query.BaseModel
	Name       string
	CategoryID uint64
	Category   Category
}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("LXGO_QUERY_TEST_DSN")
	if dsn == "" {
		dsn = "host=localhost user=lx password=123456 dbname=lxgoquerytest port=55432 sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}

	if err := db.AutoMigrate(&Category{}, &iUser{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	t.Cleanup(func() {
		db.Exec("DELETE FROM i_users")
		db.Exec("DELETE FROM categories")
	})

	return db
}

func newUserRepo(db *gorm.DB) query.IBaseRepo[iUser] {
	return query.NewBaseRepo[iUser](db, nil)
}

func newCategoryRepo(db *gorm.DB) query.IBaseRepo[Category] {
	return query.NewBaseRepo[Category](db, nil)
}

func TestBaseRepo_CreateAndReadByID(t *testing.T) {
	db := setupTestDB(t)
	repo := newUserRepo(db)

	if err := repo.Create(&iUser{Name: "Alice"}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	all, err := repo.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("ReadAll = %d rows, want 1", len(all))
	}

	got, err := repo.ReadByID(all[0].ID)
	if err != nil {
		t.Fatalf("ReadByID: %v", err)
	}
	if got.Name != "Alice" {
		t.Fatalf("Name = %q, want 'Alice'", got.Name)
	}
}

func TestBaseRepo_CreateFromMap(t *testing.T) {
	db := setupTestDB(t)
	repo := newUserRepo(db)

	got, err := repo.CreateFromMap(map[string]any{"Name": "Bob"})
	if err != nil {
		t.Fatalf("CreateFromMap: %v", err)
	}
	if got.ID == 0 || got.Name != "Bob" {
		t.Fatalf("got = %#v", got)
	}
}

func TestBaseRepo_ReadBy_And_ReadWhere(t *testing.T) {
	db := setupTestDB(t)
	repo := newUserRepo(db)

	if err := repo.Create(&iUser{Name: "Alice", CategoryID: 1}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.Create(&iUser{Name: "Bob", CategoryID: 2}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	byName, err := repo.ReadBy("name", "Alice")
	if err != nil {
		t.Fatalf("ReadBy: %v", err)
	}
	if len(byName) != 1 || byName[0].Name != "Alice" {
		t.Fatalf("ReadBy(name, Alice) = %#v", byName)
	}

	byWhere, err := repo.ReadWhere(map[string]any{"name": "Bob", "category_id": uint64(2)})
	if err != nil {
		t.Fatalf("ReadWhere: %v", err)
	}
	if len(byWhere) != 1 || byWhere[0].Name != "Bob" {
		t.Fatalf("ReadWhere = %#v", byWhere)
	}
}

func TestBaseRepo_Update_UpdateByID_UpdateFromMap(t *testing.T) {
	db := setupTestDB(t)
	repo := newUserRepo(db)

	u := &iUser{Name: "Alice"}
	if err := repo.Create(u); err != nil {
		t.Fatalf("Create: %v", err)
	}

	u.Name = "Alice2"
	if err := repo.Update(u); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, _ := repo.ReadByID(u.ID)
	if got.Name != "Alice2" {
		t.Fatalf("after Update, Name = %q", got.Name)
	}

	if err := repo.UpdateByID(u.ID, &iUser{Name: "Alice3"}); err != nil {
		t.Fatalf("UpdateByID: %v", err)
	}
	got, _ = repo.ReadByID(u.ID)
	if got.Name != "Alice3" {
		t.Fatalf("after UpdateByID, Name = %q", got.Name)
	}

	if err := repo.UpdateFromMap(u.ID, map[string]any{"name": "Alice4"}); err != nil {
		t.Fatalf("UpdateFromMap: %v", err)
	}
	got, _ = repo.ReadByID(u.ID)
	if got.Name != "Alice4" {
		t.Fatalf("after UpdateFromMap, Name = %q", got.Name)
	}
}

// TestBaseRepo_DeleteByID_IsSoftDelete confirms DeleteByID sets DeletedAt
// (row invisible to normal queries) rather than removing the row, and that
// ForceDeleteByID (Unscoped) actually removes it.
func TestBaseRepo_DeleteByID_IsSoftDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := newUserRepo(db)

	u := &iUser{Name: "Alice"}
	if err := repo.Create(u); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.DeleteByID(u.ID); err != nil {
		t.Fatalf("DeleteByID: %v", err)
	}

	exists, err := repo.ExistsByID(u.ID)
	if err != nil {
		t.Fatalf("ExistsByID: %v", err)
	}
	if exists {
		t.Fatal("expected ExistsByID to be false after a soft-delete")
	}

	var count int64
	db.Unscoped().Model(&iUser{}).Where("id = ?", u.ID).Count(&count)
	if count != 1 {
		t.Fatalf("expected the row to still exist (soft-deleted), Unscoped count = %d", count)
	}

	if err := repo.ForceDeleteByID(u.ID); err != nil {
		t.Fatalf("ForceDeleteByID: %v", err)
	}
	db.Unscoped().Model(&iUser{}).Where("id = ?", u.ID).Count(&count)
	if count != 0 {
		t.Fatalf("expected the row to be gone after ForceDeleteByID, Unscoped count = %d", count)
	}
}

// TestQueryBuilder_All_FilterByID is a real-DB regression test: filtering
// by "ID" through QueryBuilder used to reference a nonexistent "i_d"
// column.
func TestQueryBuilder_All_FilterByID(t *testing.T) {
	db := setupTestDB(t)
	repo := newUserRepo(db)

	if err := repo.Create(&iUser{Name: "Alice"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	all, _ := repo.ReadAll()
	target := all[0]

	got, err := repo.QueryBuilder().Where(query.Eq("ID", target.ID)).All()
	if err != nil {
		t.Fatalf("QueryBuilder Where ID: %v", err)
	}
	if len(got) != 1 || got[0].ID != target.ID {
		t.Fatalf("got = %#v, want just %#v", got, target)
	}
}

// TestQueryBuilder_With_JoinsAndFiltersByRelationField is an integration
// test of the JOIN path (With + a "Relation.Field" condition), and a
// real-DB regression for the JOIN table-naming fix - "Category" pluralizes
// irregularly ("categories", not "categorys"), so this also catches naive
// pluralization, unlike a relation named e.g. "Role" would (which happens
// to pluralize the same way both ways).
func TestQueryBuilder_With_JoinsAndFiltersByRelationField(t *testing.T) {
	db := setupTestDB(t)
	userRepo := newUserRepo(db)
	categoryRepo := newCategoryRepo(db)

	if err := categoryRepo.Create(&Category{Name: "admin"}); err != nil {
		t.Fatalf("Create category: %v", err)
	}
	if err := categoryRepo.Create(&Category{Name: "guest"}); err != nil {
		t.Fatalf("Create category: %v", err)
	}
	categories, _ := categoryRepo.ReadAll()
	var adminID, guestID uint64
	for _, c := range categories {
		if c.Name == "admin" {
			adminID = c.ID
		} else {
			guestID = c.ID
		}
	}

	if err := userRepo.Create(&iUser{Name: "Alice", CategoryID: adminID}); err != nil {
		t.Fatalf("Create user: %v", err)
	}
	if err := userRepo.Create(&iUser{Name: "Bob", CategoryID: guestID}); err != nil {
		t.Fatalf("Create user: %v", err)
	}

	got, err := userRepo.QueryBuilder().
		With("Category").
		Where(query.Eq("Category.Name", "admin")).
		All()
	if err != nil {
		t.Fatalf("QueryBuilder With/Where(Category.Name): %v", err)
	}
	if len(got) != 1 || got[0].Name != "Alice" {
		t.Fatalf("got = %#v, want just Alice", got)
	}
	if got[0].Category.Name != "admin" {
		t.Fatalf("expected With(\"Category\") to preload the relation, got Category = %#v", got[0].Category)
	}
}

// TestQueryBuilder_GroupBy_ByForeignKey is a real-DB regression test:
// GroupBy("CategoryID") used to emit a literal, wrongly-cased "CategoryID"
// instead of "category_id" - Postgres would reject that (no such column),
// where gorm's own Count() special-cases an active GROUP BY clause to
// return the number of distinct groups instead of erroring.
func TestQueryBuilder_GroupBy_ByForeignKey(t *testing.T) {
	db := setupTestDB(t)
	repo := newUserRepo(db)

	if err := repo.Create(&iUser{Name: "Alice", CategoryID: 1}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.Create(&iUser{Name: "Bob", CategoryID: 1}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.Create(&iUser{Name: "Carol", CategoryID: 2}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	cnt, err := repo.QueryBuilder().GroupBy("CategoryID").Count()
	if err != nil {
		t.Fatalf("GroupBy+Count: %v (a wrongly-cased GROUP BY column would fail here)", err)
	}
	if cnt != 2 {
		t.Fatalf("Count = %d, want 2 distinct CategoryID groups", cnt)
	}
}

func TestQueryBuilder_Count(t *testing.T) {
	db := setupTestDB(t)
	repo := newUserRepo(db)

	for i := 0; i < 3; i++ {
		if err := repo.Create(&iUser{Name: fmt.Sprintf("u%d", i)}); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	cnt, err := repo.QueryBuilder().Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if cnt != 3 {
		t.Fatalf("Count = %d, want 3", cnt)
	}
}
