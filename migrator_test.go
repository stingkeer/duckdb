package duckdb_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Migration: AutoMigrate
// ---------------------------------------------------------------------------

func TestAutoMigrate_CreateTable(t *testing.T) {
	db := openDB(t)
	err := db.AutoMigrate(&Product{})
	require.NoError(t, err)

	assert.True(t, db.Migrator().HasTable(&Product{}))
	assert.True(t, db.Migrator().HasColumn(&Product{}, "Name"))
	assert.True(t, db.Migrator().HasColumn(&Product{}, "Price"))
}

func TestAutoMigrate_MultipleTables(t *testing.T) {
	db := openDB(t)
	err := db.AutoMigrate(&Product{}, &User{}, &Post{})
	require.NoError(t, err)

	assert.True(t, db.Migrator().HasTable(&Product{}))
	assert.True(t, db.Migrator().HasTable(&User{}))
	assert.True(t, db.Migrator().HasTable(&Post{}))
}

func TestAutoMigrate_Idempotent(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))
	// Running again should not error
	err := db.AutoMigrate(&Product{})
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Migration: HasTable / GetTables
// ---------------------------------------------------------------------------

func TestHasTable_ByModel(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	assert.True(t, db.Migrator().HasTable(&Product{}))
	assert.False(t, db.Migrator().HasTable("nonexistent_table"))
}

func TestHasTable_ByName(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	assert.True(t, db.Migrator().HasTable("products"))
	assert.False(t, db.Migrator().HasTable("nope"))
}

func TestGetTables(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}, &User{}))

	tables, err := db.Migrator().GetTables()
	require.NoError(t, err)

	assert.Contains(t, tables, "products")
	assert.Contains(t, tables, "users")
}

// ---------------------------------------------------------------------------
// Migration: DropTable
// ---------------------------------------------------------------------------

func TestDropTable(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))
	assert.True(t, db.Migrator().HasTable(&Product{}))

	err := db.Migrator().DropTable(&Product{})
	require.NoError(t, err)
	assert.False(t, db.Migrator().HasTable(&Product{}))
}

func TestDropTable_ByName(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	err := db.Migrator().DropTable("products")
	require.NoError(t, err)
	assert.False(t, db.Migrator().HasTable("products"))
}

func TestDropTable_Idempotent(t *testing.T) {
	db := openDB(t)
	// Dropping a non-existent table should not error
	err := db.Migrator().DropTable("ghost_table")
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Migration: RenameTable
// ---------------------------------------------------------------------------

func TestRenameTable(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))
	require.True(t, db.Migrator().HasTable("products"))

	err := db.Migrator().RenameTable("products", "products_renamed")
	require.NoError(t, err)

	assert.False(t, db.Migrator().HasTable("products"))
	assert.True(t, db.Migrator().HasTable("products_renamed"))
}

// ---------------------------------------------------------------------------
// Migration: Columns
// ---------------------------------------------------------------------------

func TestHasColumn(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	assert.True(t, db.Migrator().HasColumn(&Product{}, "Name"))
	assert.True(t, db.Migrator().HasColumn(&Product{}, "Price"))
	assert.False(t, db.Migrator().HasColumn(&Product{}, "NonExistent"))
}

func TestAddColumn(t *testing.T) {
	// ProductWithNotes maps to the same "products" table so AddColumn
	// targets the correct table.
	type ProductWithNotes struct {
		ID    uint `gorm:"primaryKey"`
		Name  string
		Price float64
		Notes string `gorm:"column:notes"`
	}
	db := openDB(t)
	// Migrate with Product (no Notes column)
	require.NoError(t, db.AutoMigrate(&Product{}))

	// Point ProductWithNotes at the products table
	db2 := db.Table("products")
	err := db2.Migrator().AddColumn(&ProductWithNotes{}, "Notes")
	require.NoError(t, err)
	assert.True(t, db.Migrator().HasColumn(&Product{}, "notes"))
}

func TestDropColumn(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&User{}))
	require.True(t, db.Migrator().HasColumn(&User{}, "Active"))

	err := db.Migrator().DropColumn(&User{}, "Active")
	require.NoError(t, err)
	assert.False(t, db.Migrator().HasColumn(&User{}, "Active"))
}

func TestRenameColumn(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))
	require.True(t, db.Migrator().HasColumn(&Product{}, "Name"))

	err := db.Migrator().RenameColumn(&Product{}, "Name", "ProductName")
	require.NoError(t, err)
	assert.True(t, db.Migrator().HasColumn(&Product{}, "ProductName"))
	assert.False(t, db.Migrator().HasColumn(&Product{}, "Name"))
}

// ---------------------------------------------------------------------------
// Migration: Indexes
// ---------------------------------------------------------------------------

type IndexedModel struct {
	ID    uint   `gorm:"primaryKey"`
	Name  string `gorm:"index"`
	Email string `gorm:"uniqueIndex"`
}

func TestCreateIndex_HasIndex(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&IndexedModel{}))

	assert.True(t, db.Migrator().HasIndex(&IndexedModel{}, "idx_indexed_models_name"))
}

func TestDropIndex(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&IndexedModel{}))
	require.True(t, db.Migrator().HasIndex(&IndexedModel{}, "idx_indexed_models_name"))

	err := db.Migrator().DropIndex(&IndexedModel{}, "idx_indexed_models_name")
	require.NoError(t, err)
	assert.False(t, db.Migrator().HasIndex(&IndexedModel{}, "idx_indexed_models_name"))
}

func TestCompositeIndex(t *testing.T) {
	type CompIdx struct {
		ID     uint   `gorm:"primaryKey"`
		Field1 string `gorm:"index:comp_idx"`
		Field2 string `gorm:"index:comp_idx"`
	}
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&CompIdx{}))

	assert.True(t, db.Migrator().HasIndex(&CompIdx{}, "comp_idx"))
}

// ---------------------------------------------------------------------------
// Migration: Constraints
// ---------------------------------------------------------------------------

func TestHasConstraint_Unique(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&User{}))

	assert.True(t, db.Migrator().HasConstraint(&User{}, "uni_users_email"))
}

func TestDropConstraint_NotSupported(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&User{}))

	// DuckDB does not support ALTER TABLE DROP CONSTRAINT
	err := db.Migrator().DropConstraint(&User{}, "uni_users_email")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Migration: CurrentDatabase
// ---------------------------------------------------------------------------

func TestCurrentDatabase(t *testing.T) {
	db := openDB(t)
	name := db.Migrator().CurrentDatabase()
	assert.NotEmpty(t, name)
}

// ---------------------------------------------------------------------------
// Migration: Unsupported operations
// ---------------------------------------------------------------------------

func TestRenameIndex_NotSupported(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&IndexedModel{}))

	// DuckDB does not support ALTER INDEX RENAME TO
	err := db.Migrator().RenameIndex(&IndexedModel{}, "idx_indexed_models_name", "new_name")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Migration: AutoMigrate with data
// ---------------------------------------------------------------------------

func TestAutoMigrate_PreservesData(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	db.Create(&Product{Name: "Keep", Price: 1})

	// Re-run AutoMigrate; data should be preserved
	require.NoError(t, db.AutoMigrate(&Product{}))

	var p Product
	err := db.First(&p).Error
	require.NoError(t, err)
	assert.Equal(t, "Keep", p.Name)
}
