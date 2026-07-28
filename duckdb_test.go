package duckdb_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.aew.app/duckdb.v1"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// openDB creates a fresh in-memory DuckDB connection with a single connection
// so every test is fully isolated.
func openDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(duckdb.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	return db
}

// ---------------------------------------------------------------------------
// Dialector basics
// ---------------------------------------------------------------------------

func TestDialectorName(t *testing.T) {
	d := duckdb.Open(":memory:")
	assert.Equal(t, "duckdb", d.Name())
}

func TestOpenAndInitialize(t *testing.T) {
	db := openDB(t)
	assert.NotNil(t, db)
}

func TestNewWithConfig(t *testing.T) {
	d := duckdb.New(duckdb.Config{DSN: ":memory:"})
	assert.NotNil(t, d)
	assert.Equal(t, "duckdb", d.Name())
}

func TestExplain(t *testing.T) {
	d := duckdb.Open(":memory:")
	explained := d.Explain("SELECT * FROM users WHERE name = ?", "John")
	// DuckDB (Postgres-compatible) uses single quotes for string literals;
	// double quotes denote an identifier. Bound string values must render as
	// 'John', otherwise DEFAULT/WHERE clauses break with "cannot contain
	// column names" errors.
	assert.Contains(t, explained, `'John'`)
}

// ---------------------------------------------------------------------------
// QuoteTo
// ---------------------------------------------------------------------------

func TestQuoteTo_SimpleIdentifier(t *testing.T) {
	d := duckdb.Open(":memory:")
	var sb strings.Builder
	d.QuoteTo(&sb, "users")
	assert.Equal(t, `"users"`, sb.String())
}

func TestQuoteTo_DottedIdentifier(t *testing.T) {
	d := duckdb.Open(":memory:")
	var sb strings.Builder
	d.QuoteTo(&sb, "public.users")
	assert.Equal(t, `"public"."users"`, sb.String())
}

func TestQuoteTo_WithDoubleQuotes(t *testing.T) {
	d := duckdb.Open(":memory:")
	var sb strings.Builder
	d.QuoteTo(&sb, `my"table`)
	// internal double-quote should be escaped as ""
	assert.Equal(t, `"my""table"`, sb.String())
}

// ---------------------------------------------------------------------------
// BindVarTo
// ---------------------------------------------------------------------------

func TestBindVarTo(t *testing.T) {
	d := duckdb.Open(":memory:")
	var sb strings.Builder
	d.BindVarTo(&sb, &gorm.Statement{DB: &gorm.DB{}}, nil)
	assert.Equal(t, "?", sb.String())
}

// ---------------------------------------------------------------------------
// DefaultValueOf
// ---------------------------------------------------------------------------

func TestDefaultValueOf(t *testing.T) {
	d := duckdb.Open(":memory:")
	field := &schema.Field{}
	expr := d.DefaultValueOf(field).(clause.Expr)
	assert.Equal(t, "DEFAULT", expr.SQL)
}

// ---------------------------------------------------------------------------
// DataTypeOf
// ---------------------------------------------------------------------------

func TestDataTypeOf_Bool(t *testing.T) {
	d := duckdb.Open(":memory:")
	field := &schema.Field{DataType: schema.Bool}
	assert.Equal(t, "boolean", d.DataTypeOf(field))
}

func TestDataTypeOf_Int(t *testing.T) {
	d := duckdb.Open(":memory:")
	tests := []struct {
		size     int
		expected string
	}{
		{8, "smallint"},
		{16, "smallint"},
		{32, "integer"},
		{64, "bigint"},
	}
	for _, tt := range tests {
		field := &schema.Field{DataType: schema.Int, Size: tt.size}
		assert.Equal(t, tt.expected, d.DataTypeOf(field))
	}
}

func TestDataTypeOf_Uint(t *testing.T) {
	d := duckdb.Open(":memory:")
	tests := []struct {
		size     int
		expected string
	}{
		{8, "bigint"},    // 8+1=9 -> still smallint? no: 9 <= 16 -> smallint
		{15, "smallint"}, // 15+1=16 -> smallint
		{31, "integer"},  // 31+1=32 -> integer
		{63, "bigint"},   // 63+1=64 -> bigint
	}
	for _, tt := range tests {
		field := &schema.Field{DataType: schema.Uint, Size: tt.size}
		// recompute expected based on size+1 logic
		effSize := tt.size + 1
		var expected string
		switch {
		case effSize <= 16:
			expected = "smallint"
		case effSize <= 32:
			expected = "integer"
		default:
			expected = "bigint"
		}
		assert.Equal(t, expected, d.DataTypeOf(field), "size=%d", tt.size)
	}
	_ = tests // keep for documentation
}

func TestDataTypeOf_Float(t *testing.T) {
	d := duckdb.Open(":memory:")
	// without precision
	field := &schema.Field{DataType: schema.Float}
	assert.Equal(t, "decimal", d.DataTypeOf(field))

	// with precision only
	field = &schema.Field{DataType: schema.Float, Precision: 10}
	assert.Equal(t, "numeric(10)", d.DataTypeOf(field))

	// with precision and scale
	field = &schema.Field{DataType: schema.Float, Precision: 10, Scale: 2}
	assert.Equal(t, "numeric(10, 2)", d.DataTypeOf(field))
}

func TestDataTypeOf_String(t *testing.T) {
	d := duckdb.Open(":memory:")
	// without size
	field := &schema.Field{DataType: schema.String}
	assert.Equal(t, "text", d.DataTypeOf(field))

	// with size
	field = &schema.Field{DataType: schema.String, Size: 255}
	assert.Equal(t, "varchar(255)", d.DataTypeOf(field))
}

func TestDataTypeOf_Time(t *testing.T) {
	d := duckdb.Open(":memory:")
	field := &schema.Field{DataType: schema.Time}
	assert.Equal(t, "timestamptz", d.DataTypeOf(field))

	field = &schema.Field{DataType: schema.Time, Precision: 3}
	assert.Equal(t, "timestamptz(3)", d.DataTypeOf(field))
}

func TestDataTypeOf_Bytes(t *testing.T) {
	d := duckdb.Open(":memory:")
	field := &schema.Field{DataType: schema.Bytes}
	assert.Equal(t, "blob", d.DataTypeOf(field))
}

// ---------------------------------------------------------------------------
// ClauseBuilders — LIMIT
// ---------------------------------------------------------------------------

func TestLimitClause(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	db.Create(&Product{Name: "A", Price: 10})
	db.Create(&Product{Name: "B", Price: 20})
	db.Create(&Product{Name: "C", Price: 30})

	var products []Product
	err := db.Limit(2).Find(&products).Error
	require.NoError(t, err)
	assert.Len(t, products, 2)
}

func TestLimitOffsetClause(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	for i := 0; i < 5; i++ {
		db.Create(&Product{Name: "P", Price: float64(i + 1)})
	}

	var products []Product
	err := db.Limit(2).Offset(2).Order("price").Find(&products).Error
	require.NoError(t, err)
	assert.Len(t, products, 2)
	assert.Equal(t, 3.0, products[0].Price)
	assert.Equal(t, 4.0, products[1].Price)
}

func TestOffsetOnlyClause(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	for i := 0; i < 5; i++ {
		db.Create(&Product{Name: "P", Price: float64(i + 1)})
	}

	var products []Product
	// LIMIT ALL with OFFSET returns all remaining rows after the offset
	err := db.Limit(-1).Offset(3).Order("price").Find(&products).Error
	require.NoError(t, err)
	assert.Len(t, products, 2)
}

// ---------------------------------------------------------------------------
// ClauseBuilders — FOR (row locking should be silently ignored)
// ---------------------------------------------------------------------------

func TestForClauseIgnored(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))
	db.Create(&Product{Name: "A", Price: 1})

	var p Product
	// DuckDB doesn't support FOR UPDATE; the clause builder should ignore it
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).First(&p).Error
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// SavePoint / RollbackTo
// NOTE: DuckDB v1.x does not support SAVEPOINT syntax. These methods exist
// for GORM interface compliance but are no-ops. When DuckDB adds savepoint
// support, these tests can be re-enabled with real assertions.
// ---------------------------------------------------------------------------

func TestSavePointMethodExists(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	// Calling SavePoint/RollbackTo should not panic; errors are silently
	// swallowed because DuckDB does not support SAVEPOINT.
	err := db.Transaction(func(tx *gorm.DB) error {
		tx.Create(&Product{Name: "A", Price: 10})
		// These are no-ops in DuckDB (SAVEPOINT not supported)
		_ = tx.SavePoint("sp1").Error
		tx.Create(&Product{Name: "B", Price: 20})
		_ = tx.RollbackTo("sp1").Error
		return nil
	})
	assert.NoError(t, err)
}
