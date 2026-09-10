package models

import (
	"context"
	"errors"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ProductFilter narrows and pages a catalog listing. The zero value lists the
// whole catalog from the beginning; Limit is expected to be set by the caller.
type ProductFilter struct {
	Offset int
	Limit  int
	// Category is a category code. Empty means no category filter.
	Category string
	// PriceLessThan is an exclusive upper bound. Nil means no price filter.
	PriceLessThan *decimal.Decimal
}

type ProductsRepository struct {
	db *gorm.DB
}

func NewProductsRepository(db *gorm.DB) *ProductsRepository {
	return &ProductsRepository{
		db: db,
	}
}

// List returns a page of the catalog along with the total number of products
// matching the filter, which is what pagination has to be built on: the page
// itself only says how many rows fit in it.
//
// Variants are left out: the catalog listing does not expose them, and
// preloading them costs one extra query and a row per variant.
func (r *ProductsRepository) List(ctx context.Context, f ProductFilter) ([]Product, int64, error) {
	// Joins loads the category in the same query, unlike Preload, which would
	// issue a second one. Session makes the built query reusable, so the count
	// and the page can share it without carrying state over.
	q := r.db.WithContext(ctx).
		Model(&Product{}).
		Joins("Category").
		Session(&gorm.Session{})

	if f.Category != "" {
		// The alias GORM gives the joined table is quoted, so it has to be
		// quoted here too: unquoted, Postgres would fold it to lowercase.
		q = q.Where(`"Category".code = ?`, f.Category)
	}

	if f.PriceLessThan != nil {
		q = q.Where("products.price < ?", *f.PriceLessThan)
	}

	// Counted before the page is cut out, so the total covers the filter and
	// not just the rows that survived LIMIT/OFFSET.
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Without an explicit ORDER BY, Postgres promises no particular order, and
	// paging over it could repeat or skip rows between pages.
	var products []Product
	if err := q.Order("products.id").Offset(f.Offset).Limit(f.Limit).Find(&products).Error; err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

// FindByCode returns a single product with its category and its variants. It
// reports ErrNotFound when the code matches nothing, so the caller can tell that
// apart from a genuine failure without knowing about GORM.
func (r *ProductsRepository) FindByCode(ctx context.Context, code string) (*Product, error) {
	var product Product

	// Variants are preloaded here, unlike in List: the detail response does
	// expose them. They are ordered by id so the response is stable.
	err := r.db.WithContext(ctx).
		Joins("Category").
		Preload("Variants", func(db *gorm.DB) *gorm.DB {
			return db.Order("product_variants.id")
		}).
		Where("products.code = ?", code).
		First(&product).Error

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return nil, ErrNotFound
	case err != nil:
		return nil, err
	}

	return &product, nil
}
