package catalog

import (
	"context"
	"log"
	"net/http"

	"github.com/mytheresa/go-hiring-challenge/app/api"
	"github.com/mytheresa/go-hiring-challenge/models"
)

// ProductsRepository is the slice of the products storage the catalog needs.
// It is declared on the consumer side so the handler can be exercised with a
// fake, without reaching for a database.
type ProductsRepository interface {
	List(ctx context.Context) ([]models.Product, error)
}

type Response struct {
	Products []Product `json:"products"`
}

type Product struct {
	Code  string  `json:"code"`
	Price float64 `json:"price"`
}

type CatalogHandler struct {
	repo ProductsRepository
}

func NewCatalogHandler(r ProductsRepository) *CatalogHandler {
	return &CatalogHandler{
		repo: r,
	}
}

func (h *CatalogHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	products, err := h.repo.List(r.Context())
	if err != nil {
		// The underlying error can name tables and columns, so it is logged
		// rather than handed to the client.
		log.Printf("catalog: listing products failed: %s", err)
		api.ErrorResponse(w, http.StatusInternalServerError, "could not retrieve the catalog")
		return
	}

	api.OKResponse(w, Response{Products: newProducts(products)})
}

// newProducts maps stored products onto their API representation.
func newProducts(products []models.Product) []Product {
	res := make([]Product, 0, len(products))
	for _, p := range products {
		res = append(res, Product{
			Code:  p.Code,
			Price: p.Price.InexactFloat64(),
		})
	}

	return res
}
