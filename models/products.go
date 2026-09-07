package models

import (
	"github.com/shopspring/decimal"
)

// Product represents a product in the catalog.
// It includes a unique code, a price and the category it belongs to.
type Product struct {
	ID         uint            `gorm:"primaryKey"`
	Code       string          `gorm:"uniqueIndex;not null"`
	Price      decimal.Decimal `gorm:"type:decimal(10,2);not null"`
	CategoryID uint            `gorm:"not null"`
	Category   Category        `gorm:"foreignKey:CategoryID"`
	Variants   []Variant       `gorm:"foreignKey:ProductID"`
}

func (p *Product) TableName() string {
	return "products"
}
