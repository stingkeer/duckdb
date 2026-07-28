package duckdb_test

import "time"

// Shared test models used across all test files.

type Product struct {
	ID    uint `gorm:"primaryKey"`
	Name  string
	Price float64
}

type User struct {
	ID     uint `gorm:"primaryKey"`
	Name   string
	Email  string `gorm:"unique"`
	Active bool
}

type Post struct {
	ID        uint `gorm:"primaryKey"`
	Title     string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Item struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"size:50"`
	Description string `gorm:"type:text"`
	Stock       int
}
