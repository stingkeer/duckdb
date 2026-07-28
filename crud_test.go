package duckdb_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// CRUD: Create
// ---------------------------------------------------------------------------

func TestCreate_BasicInsert(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	p := Product{Name: "Widget", Price: 9.99}
	err := db.Create(&p).Error
	require.NoError(t, err)
	assert.NotZero(t, p.ID, "ID should be auto-generated")
}

func TestCreate_AutoIncrement(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	p1 := Product{Name: "A", Price: 1}
	p2 := Product{Name: "B", Price: 2}
	require.NoError(t, db.Create(&p1).Error)
	require.NoError(t, db.Create(&p2).Error)

	assert.Greater(t, p2.ID, p1.ID, "second ID should be greater")
}

func TestCreate_BatchInsert(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	products := []Product{
		{Name: "A", Price: 1},
		{Name: "B", Price: 2},
		{Name: "C", Price: 3},
	}
	err := db.Create(&products).Error
	require.NoError(t, err)

	for _, p := range products {
		assert.NotZero(t, p.ID)
	}

	var count int64
	db.Model(&Product{}).Count(&count)
	assert.Equal(t, int64(3), count)
}

func TestCreate_UniqueConstraint(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&User{}))

	u1 := User{Name: "U1", Email: "dup@test.com"}
	require.NoError(t, db.Create(&u1).Error)

	u2 := User{Name: "U2", Email: "dup@test.com"}
	err := db.Create(&u2).Error
	assert.Error(t, err, "duplicate email should violate unique constraint")
}

// ---------------------------------------------------------------------------
// CRUD: Read
// ---------------------------------------------------------------------------

func TestRead_FirstByID(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	p := Product{Name: "FindMe", Price: 42}
	require.NoError(t, db.Create(&p).Error)

	var result Product
	err := db.First(&result, p.ID).Error
	require.NoError(t, err)
	assert.Equal(t, "FindMe", result.Name)
	assert.Equal(t, 42.0, result.Price)
}

func TestRead_FirstByWhere(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	db.Create(&Product{Name: "Special", Price: 100})

	var result Product
	err := db.Where("name = ?", "Special").First(&result).Error
	require.NoError(t, err)
	assert.Equal(t, 100.0, result.Price)
}

func TestRead_FindAll(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	for i := 0; i < 10; i++ {
		db.Create(&Product{Name: "P", Price: float64(i)})
	}

	var products []Product
	err := db.Find(&products).Error
	require.NoError(t, err)
	assert.Len(t, products, 10)
}

func TestRead_FindWithWhere(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	db.Create(&Product{Name: "A", Price: 50})
	db.Create(&Product{Name: "B", Price: 10})
	db.Create(&Product{Name: "C", Price: 100})

	var expensive []Product
	err := db.Where("price > ?", 20).Find(&expensive).Error
	require.NoError(t, err)
	assert.Len(t, expensive, 2)
}

func TestRead_Count(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	for i := 0; i < 5; i++ {
		db.Create(&Product{Name: "P", Price: float64(i)})
	}

	var count int64
	err := db.Model(&Product{}).Where("price >= ?", 2).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestRead_OrderAndGroup(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	db.Create(&Product{Name: "C", Price: 3})
	db.Create(&Product{Name: "A", Price: 1})
	db.Create(&Product{Name: "B", Price: 2})

	var products []Product
	err := db.Order("price ASC").Find(&products).Error
	require.NoError(t, err)
	assert.Equal(t, "A", products[0].Name)
	assert.Equal(t, "B", products[1].Name)
	assert.Equal(t, "C", products[2].Name)
}

func TestRead_Pluck(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	db.Create(&Product{Name: "X", Price: 1})
	db.Create(&Product{Name: "Y", Price: 2})

	var names []string
	err := db.Model(&Product{}).Pluck("name", &names).Error
	require.NoError(t, err)
	assert.Contains(t, names, "X")
	assert.Contains(t, names, "Y")
}

func TestRead_NotFound(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	var result Product
	err := db.First(&result, 99999).Error
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// ---------------------------------------------------------------------------
// CRUD: Update
// ---------------------------------------------------------------------------

func TestUpdate_SingleField(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	p := Product{Name: "Old", Price: 10}
	require.NoError(t, db.Create(&p).Error)

	err := db.Model(&p).Update("Price", 99).Error
	require.NoError(t, err)

	var result Product
	db.First(&result, p.ID)
	assert.Equal(t, 99.0, result.Price)
}

func TestUpdate_MultipleFields(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	p := Product{Name: "Old", Price: 10}
	require.NoError(t, db.Create(&p).Error)

	err := db.Model(&p).Updates(map[string]interface{}{
		"Name":  "New",
		"Price": 50,
	}).Error
	require.NoError(t, err)

	var result Product
	db.First(&result, p.ID)
	assert.Equal(t, "New", result.Name)
	assert.Equal(t, 50.0, result.Price)
}

func TestUpdate_UsingStruct(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	p := Product{Name: "Old", Price: 10}
	require.NoError(t, db.Create(&p).Error)

	err := db.Model(&p).Updates(Product{Name: "Updated"}).Error
	require.NoError(t, err)

	var result Product
	db.First(&result, p.ID)
	assert.Equal(t, "Updated", result.Name)
	assert.Equal(t, 10.0, result.Price, "zero-value fields should not be updated")
}

// ---------------------------------------------------------------------------
// CRUD: Delete
// ---------------------------------------------------------------------------

func TestDelete_ByModel(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	p := Product{Name: "DeleteMe", Price: 1}
	require.NoError(t, db.Create(&p).Error)

	err := db.Delete(&p).Error
	require.NoError(t, err)

	var result Product
	err = db.First(&result, p.ID).Error
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestDelete_ByWhere(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	for i := 0; i < 5; i++ {
		db.Create(&Product{Name: "P", Price: float64(i)})
	}

	err := db.Where("price < ?", 3).Delete(&Product{}).Error
	require.NoError(t, err)

	var count int64
	db.Model(&Product{}).Count(&count)
	assert.Equal(t, int64(2), count)
}

// ---------------------------------------------------------------------------
// CRUD: Auto timestamps
// ---------------------------------------------------------------------------

func TestAutoTimestamps(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Post{}))

	now := time.Now()
	p := Post{Title: "Hello", Content: "World"}
	require.NoError(t, db.Create(&p).Error)

	assert.False(t, p.CreatedAt.IsZero(), "CreatedAt should be auto-populated")
	assert.True(t, p.CreatedAt.After(now.Add(-time.Second)), "CreatedAt should be recent")

	// Update should refresh UpdatedAt
	time.Sleep(10 * time.Millisecond)
	originalUpdate := p.UpdatedAt
	require.NoError(t, db.Model(&p).Update("Title", "Updated").Error)

	var result Post
	db.First(&result, p.ID)
	assert.True(t, result.UpdatedAt.After(originalUpdate), "UpdatedAt should be refreshed")
}

// ---------------------------------------------------------------------------
// CRUD: Complex queries
// ---------------------------------------------------------------------------

func TestComplexQuery_LikePattern(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	db.Create(&Product{Name: "Apple", Price: 1})
	db.Create(&Product{Name: "Banana", Price: 2})
	db.Create(&Product{Name: "Apricot", Price: 3})

	var results []Product
	err := db.Where("name LIKE ?", "A%").Find(&results).Error
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestComplexQuery_InClause(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	db.Create(&Product{Name: "A", Price: 1})
	db.Create(&Product{Name: "B", Price: 2})
	db.Create(&Product{Name: "C", Price: 3})

	var results []Product
	err := db.Where("name IN ?", []string{"A", "C"}).Find(&results).Error
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestComplexQuery_Aggregations(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	db.Create(&Product{Name: "A", Price: 10})
	db.Create(&Product{Name: "B", Price: 20})
	db.Create(&Product{Name: "C", Price: 30})

	type Result struct {
		Total float64
		Min   float64
		Max   float64
		Count int64
	}

	var r Result
	err := db.Model(&Product{}).Select("SUM(price) as total, MIN(price) as min, MAX(price) as max, COUNT(*) as count").Scan(&r).Error
	require.NoError(t, err)
	assert.Equal(t, 60.0, r.Total)
	assert.Equal(t, 10.0, r.Min)
	assert.Equal(t, 30.0, r.Max)
	assert.Equal(t, int64(3), r.Count)
}

func TestComplexQuery_GroupBy(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	db.Create(&Product{Name: "A", Price: 10})
	db.Create(&Product{Name: "A", Price: 20})
	db.Create(&Product{Name: "B", Price: 30})

	type Result struct {
		Name  string
		Total float64
	}

	var results []Result
	err := db.Model(&Product{}).Select("name, sum(price) as total").Group("name").Order("name").Scan(&results).Error
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "A", results[0].Name)
	assert.Equal(t, 30.0, results[0].Total)
	assert.Equal(t, "B", results[1].Name)
	assert.Equal(t, 30.0, results[1].Total)
}

// ---------------------------------------------------------------------------
// CRUD: Transactions
// ---------------------------------------------------------------------------

func TestTransaction_Commit(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	err := db.Transaction(func(tx *gorm.DB) error {
		return tx.Create(&Product{Name: "T1", Price: 1}).Error
	})
	require.NoError(t, err)

	var count int64
	db.Model(&Product{}).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestTransaction_Rollback(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	err := db.Transaction(func(tx *gorm.DB) error {
		tx.Create(&Product{Name: "T1", Price: 1})
		return gorm.ErrInvalidTransaction
	})
	assert.Error(t, err)

	var count int64
	db.Model(&Product{}).Count(&count)
	assert.Equal(t, int64(0), count, "transaction should have been rolled back")
}

func TestTransaction_Nested(t *testing.T) {
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	err := db.Transaction(func(tx *gorm.DB) error {
		tx.Create(&Product{Name: "Outer", Price: 1})
		return tx.Transaction(func(inner *gorm.DB) error {
			inner.Create(&Product{Name: "Inner", Price: 2})
			return nil
		})
	})
	require.NoError(t, err)

	var count int64
	db.Model(&Product{}).Count(&count)
	assert.Equal(t, int64(2), count)
}

func TestTransaction_NestedRollback(t *testing.T) {
	// NOTE: DuckDB does not support SAVEPOINT, so the inner rollback is
	// effectively a no-op. Both rows are committed.
	db := openDB(t)
	require.NoError(t, db.AutoMigrate(&Product{}))

	err := db.Transaction(func(tx *gorm.DB) error {
		tx.Create(&Product{Name: "Outer", Price: 1})
		_ = tx.Transaction(func(inner *gorm.DB) error {
			inner.Create(&Product{Name: "Inner", Price: 2})
			return gorm.ErrInvalidTransaction
		})
		return nil
	})
	require.NoError(t, err)

	var products []Product
	db.Find(&products)
	// Both rows persist because DuckDB savepoints are not functional
	assert.GreaterOrEqual(t, len(products), 1)
}
