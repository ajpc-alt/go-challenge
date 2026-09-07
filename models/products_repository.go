package models

import (
	"context"

	"gorm.io/gorm"
)

type ProductsRepository struct {
	db *gorm.DB
}

func NewProductsRepository(db *gorm.DB) *ProductsRepository {
	return &ProductsRepository{
		db: db,
	}
}

// List returns every product in the catalog. Variants are left out: the catalog
// listing does not expose them, and preloading them costs one extra query and a
// row per variant.
func (r *ProductsRepository) List(ctx context.Context) ([]Product, error) {
	var products []Product
	if err := r.db.WithContext(ctx).Find(&products).Error; err != nil {
		return nil, err
	}

	return products, nil
}
