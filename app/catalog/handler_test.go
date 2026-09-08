package catalog

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mytheresa/go-hiring-challenge/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// fakeProductsRepository stands in for the storage. Besides returning what the
// test sets up, it records what it was asked for, which is how the query string
// parsing gets checked.
type fakeProductsRepository struct {
	products []models.Product
	total    int64
	product  *models.Product
	err      error

	listCalled bool
	gotFilter  models.ProductFilter
	gotCode    string
}

func (f *fakeProductsRepository) List(_ context.Context, filter models.ProductFilter) ([]models.Product, int64, error) {
	f.listCalled = true
	f.gotFilter = filter

	if f.err != nil {
		return nil, 0, f.err
	}

	return f.products, f.total, nil
}

func (f *fakeProductsRepository) FindByCode(_ context.Context, code string) (*models.Product, error) {
	f.gotCode = code

	if f.err != nil {
		return nil, f.err
	}

	return f.product, nil
}

// get runs a GET /catalog with the given query string against a handler backed
// by repo.
func get(repo *fakeProductsRepository, query string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	NewCatalogHandler(repo).HandleGet(recorder, httptest.NewRequest(http.MethodGet, "/catalog?"+query, nil))

	return recorder
}

func product(code, price, categoryCode, categoryName string) models.Product {
	return models.Product{
		Code:  code,
		Price: decimal.RequireFromString(price),
		Category: models.Category{
			Code: categoryCode,
			Name: categoryName,
		},
	}
}

func TestHandleGet(t *testing.T) {
	t.Run("returns the products with their category and the pagination block", func(t *testing.T) {
		repo := &fakeProductsRepository{
			products: []models.Product{
				product("PROD001", "10.99", "CLOTHING", "Clothing"),
				product("PROD002", "12.49", "SHOES", "Shoes"),
			},
			total: 2,
		}

		recorder := get(repo, "")

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
		assert.JSONEq(t, `{
			"products": [
				{"code": "PROD001", "price": 10.99, "category": {"code": "CLOTHING", "name": "Clothing"}},
				{"code": "PROD002", "price": 12.49, "category": {"code": "SHOES", "name": "Shoes"}}
			],
			"pagination": {"offset": 0, "limit": 10, "total": 2}
		}`, recorder.Body.String())
	})

	t.Run("serialises an empty page as an array, never as null", func(t *testing.T) {
		recorder := get(&fakeProductsRepository{total: 0}, "offset=99")

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.JSONEq(t, `{"products": [], "pagination": {"offset": 99, "limit": 10, "total": 0}}`, recorder.Body.String())
	})

	t.Run("reports the total for the whole filter and not for the page", func(t *testing.T) {
		// One product on the page, eight in the catalog: the total has to
		// describe the catalog, or the client cannot tell there is more to ask
		// for.
		repo := &fakeProductsRepository{
			products: []models.Product{product("PROD001", "10.99", "CLOTHING", "Clothing")},
			total:    8,
		}

		recorder := get(repo, "limit=1")

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.JSONEq(t, `{
			"products": [{"code": "PROD001", "price": 10.99, "category": {"code": "CLOTHING", "name": "Clothing"}}],
			"pagination": {"offset": 0, "limit": 1, "total": 8}
		}`, recorder.Body.String())
	})

	t.Run("answers 500 without leaking the repository error", func(t *testing.T) {
		recorder := get(&fakeProductsRepository{err: errors.New(`pq: column "products.nope" does not exist`)}, "")

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.JSONEq(t, `{"error": "could not retrieve the catalog"}`, recorder.Body.String())
		assert.NotContains(t, recorder.Body.String(), "products.nope")
	})
}

// TestHandleGetFilter covers the query string, by checking what the handler ends
// up asking the repository for.
func TestHandleGetFilter(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		wantOffset   int
		wantLimit    int
		wantCategory string
		wantPrice    string // empty means no price filter
	}{
		{
			name:       "no parameters falls back to the defaults",
			query:      "",
			wantOffset: 0,
			wantLimit:  10,
		},
		{
			name:       "empty parameters fall back to the defaults too",
			query:      "offset=&limit=&category=&price_less_than=",
			wantOffset: 0,
			wantLimit:  10,
		},
		{
			name:       "offset and limit are taken as given",
			query:      "offset=5&limit=3",
			wantOffset: 5,
			wantLimit:  3,
		},
		{
			name:       "a zero limit is clamped to the minimum",
			query:      "limit=0",
			wantOffset: 0,
			wantLimit:  1,
		},
		{
			name:       "a negative limit is clamped to the minimum",
			query:      "limit=-5",
			wantOffset: 0,
			wantLimit:  1,
		},
		{
			name:       "a limit above the maximum is clamped to it",
			query:      "limit=500",
			wantOffset: 0,
			wantLimit:  100,
		},
		{
			name:         "category is passed through as a code",
			query:        "category=SHOES",
			wantOffset:   0,
			wantLimit:    10,
			wantCategory: "SHOES",
		},
		{
			name:       "price_less_than is parsed as a decimal",
			query:      "price_less_than=15.50",
			wantOffset: 0,
			wantLimit:  10,
			wantPrice:  "15.5",
		},
		{
			name:         "filters and pagination combine",
			query:        "offset=2&limit=4&category=SHOES&price_less_than=30",
			wantOffset:   2,
			wantLimit:    4,
			wantCategory: "SHOES",
			wantPrice:    "30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeProductsRepository{}

			recorder := get(repo, tt.query)

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.True(t, repo.listCalled)
			assert.Equal(t, tt.wantOffset, repo.gotFilter.Offset)
			assert.Equal(t, tt.wantLimit, repo.gotFilter.Limit)
			assert.Equal(t, tt.wantCategory, repo.gotFilter.Category)

			if tt.wantPrice == "" {
				assert.Nil(t, repo.gotFilter.PriceLessThan)
				return
			}

			if assert.NotNil(t, repo.gotFilter.PriceLessThan) {
				assert.Equal(t, tt.wantPrice, repo.gotFilter.PriceLessThan.String())
			}
		})
	}
}

func TestHandleGetInvalidFilter(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "a non-numeric limit",
			query: "limit=abc",
			want:  "limit must be an integer",
		},
		{
			name:  "a non-numeric offset",
			query: "offset=xyz",
			want:  "offset must be a non-negative integer",
		},
		{
			name:  "a negative offset",
			query: "offset=-1",
			want:  "offset must be a non-negative integer",
		},
		{
			name:  "a non-numeric price",
			query: "price_less_than=cheap",
			want:  "price_less_than must be a number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeProductsRepository{}

			recorder := get(repo, tt.query)

			assert.Equal(t, http.StatusBadRequest, recorder.Code)
			assert.JSONEq(t, `{"error": "`+tt.want+`"}`, recorder.Body.String())
			// The request never reaches the storage: it was turned away first.
			assert.False(t, repo.listCalled)
		})
	}
}
