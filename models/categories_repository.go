package models

import (
	"context"
	"errors"

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

// Create stores a new category and fills in its generated ID. A collision with
// the unique code is reported as ErrDuplicate: it is the caller's business to
// decide what that means, and it should not have to read GORM's sentinels to
// find out. The translation relies on TranslateError being set on the
// connection, in app/database.
func (r *CategoriesRepository) Create(ctx context.Context, c *Category) error {
	err := r.db.WithContext(ctx).Create(c).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrDuplicate
	}

	return err
}
