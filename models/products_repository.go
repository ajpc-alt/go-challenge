package models

import (
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

func (r *ProductsRepository) GetAllProducts() ([]Product, error) {
	var products []Product
	if err := r.db.Preload("Variants").Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}
