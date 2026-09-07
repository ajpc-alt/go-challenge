package models

import (
	"context"

	"gorm.io/gorm"
)

type CategoriesRepository struct {
	db *gorm.DB
}

func NewCategoriesRepository(db *gorm.DB) *CategoriesRepository {
	return &CategoriesRepository{
		db: db,
	}
}

// List returns every category, ordered by code so the response is stable.
func (r *CategoriesRepository) List(ctx context.Context) ([]Category, error) {
	var categories []Category
	if err := r.db.WithContext(ctx).Order("code").Find(&categories).Error; err != nil {
		return nil, err
	}

	return categories, nil
}

// Create stores a new category and fills in its generated ID.
func (r *CategoriesRepository) Create(ctx context.Context, c *Category) error {
	return r.db.WithContext(ctx).Create(c).Error
}
