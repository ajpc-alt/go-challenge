package catalog

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mytheresa/go-hiring-challenge/models"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// getByCode runs a GET /catalog/{code} against a handler backed by repo. The
// path value is set by hand because the test does not go through the router.
func getByCode(repo *fakeProductsRepository, code string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/catalog/"+code, nil)
	request.SetPathValue("code", code)

	recorder := httptest.NewRecorder()
	NewCatalogHandler(repo).HandleGetByCode(recorder, request)

	return recorder
}

func price(value string) *decimal.Decimal {
	d := decimal.RequireFromString(value)

	return &d
}

func TestHandleGetByCode(t *testing.T) {
	t.Run("returns the product with its category and variants", func(t *testing.T) {
		p := product("PROD001", "10.99", "CLOTHING", "Clothing")
		p.Variants = []models.Variant{
			{Name: "Variant A", SKU: "SKU001A", Price: price("11.99")},
			{Name: "Variant B", SKU: "SKU001B"},
		}
		repo := &fakeProductsRepository{product: &p}

		recorder := getByCode(repo, "PROD001")

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
		assert.Equal(t, "PROD001", repo.gotCode)
		assert.JSONEq(t, `{
			"code": "PROD001",
			"price": 10.99,
			"category": {"code": "CLOTHING", "name": "Clothing"},
			"variants": [
				{"name": "Variant A", "sku": "SKU001A", "price": 11.99},
				{"name": "Variant B", "sku": "SKU001B", "price": 10.99}
			]
		}`, recorder.Body.String())
	})

	t.Run("a variant without a price of its own inherits the product's", func(t *testing.T) {
		p := product("PROD007", "18.20", "CLOTHING", "Clothing")
		p.Variants = []models.Variant{
			{Name: "Variant A", SKU: "SKU007A"},
			{Name: "Variant B", SKU: "SKU007B"},
		}

		recorder := getByCode(&fakeProductsRepository{product: &p}, "PROD007")

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.JSONEq(t, `{
			"code": "PROD007",
			"price": 18.2,
			"category": {"code": "CLOTHING", "name": "Clothing"},
			"variants": [
				{"name": "Variant A", "sku": "SKU007A", "price": 18.2},
				{"name": "Variant B", "sku": "SKU007B", "price": 18.2}
			]
		}`, recorder.Body.String())
	})

	t.Run("a variant priced at zero keeps its own price", func(t *testing.T) {
		// The reason Variant.Price is a pointer: a free variant must not be
		// mistaken for one that has no price and end up charging the
		// product's.
		p := product("PROD006", "5.50", "SHOES", "Shoes")
		p.Variants = []models.Variant{{Name: "Freebie", SKU: "SKU006Z", Price: price("0.00")}}

		recorder := getByCode(&fakeProductsRepository{product: &p}, "PROD006")

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.JSONEq(t, `{
			"code": "PROD006",
			"price": 5.5,
			"category": {"code": "SHOES", "name": "Shoes"},
			"variants": [{"name": "Freebie", "sku": "SKU006Z", "price": 0}]
		}`, recorder.Body.String())
	})

	t.Run("serialises a product without variants as an array, never as null", func(t *testing.T) {
		p := product("PROD006", "5.50", "SHOES", "Shoes")

		recorder := getByCode(&fakeProductsRepository{product: &p}, "PROD006")

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.JSONEq(t, `{
			"code": "PROD006",
			"price": 5.5,
			"category": {"code": "SHOES", "name": "Shoes"},
			"variants": []
		}`, recorder.Body.String())
	})

	t.Run("answers 404 for a code that does not exist", func(t *testing.T) {
		recorder := getByCode(&fakeProductsRepository{err: models.ErrNotFound}, "NOPE")

		assert.Equal(t, http.StatusNotFound, recorder.Code)
		assert.JSONEq(t, `{"error": "product not found"}`, recorder.Body.String())
	})

	t.Run("answers 500 without leaking the repository error", func(t *testing.T) {
		recorder := getByCode(&fakeProductsRepository{err: errors.New(`pq: relation "products" does not exist`)}, "PROD001")

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.JSONEq(t, `{"error": "could not retrieve the product"}`, recorder.Body.String())
		assert.NotContains(t, recorder.Body.String(), "relation")
	})
}
