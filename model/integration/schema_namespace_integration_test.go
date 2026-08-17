//go:build integration

package model_test

import (
	"testing"

	"github.com/epicoon/lxgo/model"
)

// TestIntrospectModelSchema_NonPublicSchema checks the basic case: a table
// living in a Postgres schema other than "public" is read correctly when
// that schema is passed explicitly.
func TestIntrospectModelSchema_NonPublicSchema(t *testing.T) {
	db := setupDB(t)
	const ns = "model_test_ns_intro"
	if _, err := db.Exec("DROP SCHEMA IF EXISTS " + ns + " CASCADE"); err != nil {
		t.Fatalf("drop schema %s: %v", ns, err)
	}
	t.Cleanup(func() { db.Exec("DROP SCHEMA IF EXISTS " + ns + " CASCADE") })
	if _, err := db.Exec("CREATE SCHEMA " + ns); err != nil {
		t.Fatalf("create schema %s: %v", ns, err)
	}

	if _, err := db.Exec(`CREATE TABLE ` + ns + `.ns_intro_widgets (id serial PRIMARY KEY, name character varying(50) NOT NULL)`); err != nil {
		t.Fatalf("create %s.ns_intro_widgets: %v", ns, err)
	}

	schema, err := model.IntrospectModelSchema(db, "ns_intro_widgets", ns, false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema: %v", err)
	}
	name, ok := schema.FieldByName("name")
	if !ok || name.Type != model.FieldTypeString || !name.Required || name.Size != 50 {
		t.Fatalf("name = %#v, %v", name, ok)
	}
}

// TestIntrospectModelSchema_SchemaIsolation checks that the schema
// parameter actually scopes the lookup, not just accepted and ignored - two
// tables sharing the same bare name in different schemas, deliberately
// shaped differently, must each be read from their own schema only.
func TestIntrospectModelSchema_SchemaIsolation(t *testing.T) {
	db := setupDB(t)
	const ns = "model_test_ns_isolation"
	physTable := pgTableName("IntroIsolationWidgets")
	if _, err := db.Exec("DROP TABLE IF EXISTS " + physTable); err != nil {
		t.Fatalf("drop public.%s: %v", physTable, err)
	}
	if _, err := db.Exec("DROP SCHEMA IF EXISTS " + ns + " CASCADE"); err != nil {
		t.Fatalf("drop schema %s: %v", ns, err)
	}
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS " + physTable)
		db.Exec("DROP SCHEMA IF EXISTS " + ns + " CASCADE")
	})
	if _, err := db.Exec("CREATE SCHEMA " + ns); err != nil {
		t.Fatalf("create schema %s: %v", ns, err)
	}

	if _, err := db.Exec(`CREATE TABLE ` + physTable + ` (id serial PRIMARY KEY, public_only integer)`); err != nil {
		t.Fatalf("create public.%s: %v", physTable, err)
	}
	if _, err := db.Exec(`CREATE TABLE ` + ns + `.` + physTable + ` (id serial PRIMARY KEY, ns_only text)`); err != nil {
		t.Fatalf("create %s.%s: %v", ns, physTable, err)
	}

	publicSchema, err := model.IntrospectModelSchema(db, physTable, "public", false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(public): %v", err)
	}
	if _, ok := publicSchema.FieldByName("public_only"); !ok {
		t.Fatal("expected public_only field from the public-schema table")
	}
	if _, ok := publicSchema.FieldByName("ns_only"); ok {
		t.Fatal("public-schema introspection leaked a field from the other schema's table")
	}

	nsSchema, err := model.IntrospectModelSchema(db, physTable, ns, false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(%s): %v", ns, err)
	}
	if _, ok := nsSchema.FieldByName("ns_only"); !ok {
		t.Fatal("expected ns_only field from the custom-schema table")
	}
	if _, ok := nsSchema.FieldByName("public_only"); ok {
		t.Fatal("custom-schema introspection leaked a field from the public table")
	}
}

// TestIntrospectModelSchema_NonPublicSchema_ManyToOne checks that an
// outgoing relation is restored correctly for a table living outside
// "public" - loadOutgoingForeignKeys/columnHasIndex must scope their own
// catalog queries to the given schema, not "public".
func TestIntrospectModelSchema_NonPublicSchema_ManyToOne(t *testing.T) {
	db := setupDB(t)
	const ns = "model_test_ns_rel"
	if _, err := db.Exec("DROP SCHEMA IF EXISTS " + ns + " CASCADE"); err != nil {
		t.Fatalf("drop schema %s: %v", ns, err)
	}
	t.Cleanup(func() {
		db.Exec("DROP SCHEMA IF EXISTS " + ns + " CASCADE")
		db.Exec(`DELETE FROM lx_sys.model_relations WHERE home_table = $1`, ns+".rel_ns_many")
	})
	if _, err := db.Exec("CREATE SCHEMA " + ns); err != nil {
		t.Fatalf("create schema %s: %v", ns, err)
	}

	if _, err := db.Exec(`CREATE TABLE ` + ns + `.rel_ns_one (id serial PRIMARY KEY)`); err != nil {
		t.Fatalf("create %s.rel_ns_one: %v", ns, err)
	}
	fkName := "fk_rel_ns_many_one"
	if _, err := db.Exec(`
		CREATE TABLE ` + ns + `.rel_ns_many (
			id serial PRIMARY KEY,
			one_id integer NOT NULL CONSTRAINT ` + fkName + ` REFERENCES ` + ns + `.rel_ns_one(id)
		)
	`); err != nil {
		t.Fatalf("create %s.rel_ns_many: %v", ns, err)
	}
	fk := model.RelationFk{
		Type: model.RelationTypeManyToOne, HomeTable: ns + ".rel_ns_many", HomeModel: "RelNsMany", HomeAttribute: "one",
		RelatedTable: ns + ".rel_ns_one", RelatedModel: "RelNsOne", RelatedAttribute: "manys",
	}
	if err := model.SetRelationFk(db, fkName, fk); err != nil {
		t.Fatalf("SetRelationFk: %v", err)
	}

	schema, err := model.IntrospectModelSchema(db, "rel_ns_many", ns, false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema: %v", err)
	}
	rel, ok := schema.RelationByName("one")
	if !ok || rel.Type != model.RelationTypeManyToOne || rel.RelatedModel != "RelNsOne" {
		t.Fatalf("one = %#v, %v", rel, ok)
	}
}

// TestIntrospectModelSchema_NonPublicSchema_ManyToMany exercises
// loadManyToManyRelationFks/manyToManyJoinColumn's schema-qualified lookup
// (RelationFk.HomeTable/JoinTable, see pgQualifiedTable) against a real
// non-"public" schema - the code path most exposed to a schema/bare-name
// mismatch, since it matches a stored "schema.table" value against a
// physical pg_catalog join keyed by schema and bare table name separately.
func TestIntrospectModelSchema_NonPublicSchema_ManyToMany(t *testing.T) {
	db := setupDB(t)
	const ns = "model_test_ns_m2m"
	if _, err := db.Exec("DROP SCHEMA IF EXISTS " + ns + " CASCADE"); err != nil {
		t.Fatalf("drop schema %s: %v", ns, err)
	}
	t.Cleanup(func() {
		db.Exec("DROP SCHEMA IF EXISTS " + ns + " CASCADE")
		db.Exec(`DELETE FROM lx_sys.model_relations WHERE home_table IN ($1, $2)`, ns+".rel_ns_m2m_a", ns+".rel_ns_m2m_b")
	})
	if _, err := db.Exec("CREATE SCHEMA " + ns); err != nil {
		t.Fatalf("create schema %s: %v", ns, err)
	}

	if _, err := db.Exec(`CREATE TABLE ` + ns + `.rel_ns_m2m_a (id serial PRIMARY KEY)`); err != nil {
		t.Fatalf("create %s.rel_ns_m2m_a: %v", ns, err)
	}
	if _, err := db.Exec(`CREATE TABLE ` + ns + `.rel_ns_m2m_b (id serial PRIMARY KEY)`); err != nil {
		t.Fatalf("create %s.rel_ns_m2m_b: %v", ns, err)
	}
	if _, err := db.Exec(`
		CREATE TABLE ` + ns + `.rel_ns_m2m_join (
			a_id integer NOT NULL CONSTRAINT fk_rel_ns_m2m_a_side REFERENCES ` + ns + `.rel_ns_m2m_a(id),
			b_id integer NOT NULL CONSTRAINT fk_rel_ns_m2m_b_side REFERENCES ` + ns + `.rel_ns_m2m_b(id)
		)
	`); err != nil {
		t.Fatalf("create %s.rel_ns_m2m_join: %v", ns, err)
	}
	// Only a_id gets an index - proves NoIndex is restored per side too,
	// via manyToManyJoinColumn/columnHasIndex under the custom schema.
	if _, err := db.Exec(`CREATE INDEX ON ` + ns + `.rel_ns_m2m_join (a_id)`); err != nil {
		t.Fatalf("create index: %v", err)
	}

	fkA := model.RelationFk{
		Type: model.RelationTypeManyToMany, HomeTable: ns + ".rel_ns_m2m_a", HomeModel: "RelNsM2mA", HomeAttribute: "bs",
		RelatedTable: ns + ".rel_ns_m2m_b", RelatedModel: "RelNsM2mB", RelatedAttribute: "as",
	}
	fkB := model.RelationFk{
		Type: model.RelationTypeManyToMany, HomeTable: ns + ".rel_ns_m2m_b", HomeModel: "RelNsM2mB", HomeAttribute: "as",
		RelatedTable: ns + ".rel_ns_m2m_a", RelatedModel: "RelNsM2mA", RelatedAttribute: "bs",
	}
	if err := model.SetRelationFk(db, "fk_rel_ns_m2m_a_side", fkA); err != nil {
		t.Fatalf("SetRelationFk A: %v", err)
	}
	if err := model.SetRelationFk(db, "fk_rel_ns_m2m_b_side", fkB); err != nil {
		t.Fatalf("SetRelationFk B: %v", err)
	}

	schemaA, err := model.IntrospectModelSchema(db, "rel_ns_m2m_a", ns, false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(A): %v", err)
	}
	relA, ok := schemaA.RelationByName("bs")
	if !ok || relA.Type != model.RelationTypeManyToMany || relA.RelatedModel != "RelNsM2mB" || relA.NoIndex {
		t.Fatalf("A.bs = %#v, %v, want NoIndex false (a_id is indexed)", relA, ok)
	}

	schemaB, err := model.IntrospectModelSchema(db, "rel_ns_m2m_b", ns, false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema(B): %v", err)
	}
	relB, ok := schemaB.RelationByName("as")
	if !ok || !relB.NoIndex {
		t.Fatalf("B.as = %#v, %v, want NoIndex true (b_id has no index)", relB, ok)
	}
}

// TestSetColumnType_NonPublicSchema checks that a recorded column-type
// override round-trips correctly for a table outside "public" -
// loadColumnOverrides must match systemTypesTable's schema-qualified
// table_name (see pgQualifiedTable), not a bare name.
func TestSetColumnType_NonPublicSchema(t *testing.T) {
	db := setupDB(t)
	const ns = "model_test_ns_types"
	if _, err := db.Exec("DROP SCHEMA IF EXISTS " + ns + " CASCADE"); err != nil {
		t.Fatalf("drop schema %s: %v", ns, err)
	}
	t.Cleanup(func() {
		db.Exec("DROP SCHEMA IF EXISTS " + ns + " CASCADE")
		db.Exec(`DELETE FROM lx_sys.model_types WHERE table_name = $1`, ns+".ns_types_gizmo")
	})
	if _, err := db.Exec("CREATE SCHEMA " + ns); err != nil {
		t.Fatalf("create schema %s: %v", ns, err)
	}
	if _, err := db.Exec(`CREATE TABLE ` + ns + `.ns_types_gizmo (id serial PRIMARY KEY, settings text)`); err != nil {
		t.Fatalf("create %s.ns_types_gizmo: %v", ns, err)
	}

	if err := model.SetColumnType(db, ns+".ns_types_gizmo", "settings", model.Field{Type: model.FieldTypeDict}); err != nil {
		t.Fatalf("SetColumnType: %v", err)
	}

	schema, err := model.IntrospectModelSchema(db, "ns_types_gizmo", ns, false)
	if err != nil {
		t.Fatalf("IntrospectModelSchema: %v", err)
	}
	settings, ok := schema.FieldByName("settings")
	if !ok || settings.Type != model.FieldTypeDict {
		t.Fatalf("settings = %#v, %v, want FieldTypeDict", settings, ok)
	}
}
