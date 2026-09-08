package catalog

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/mytheresa/go-hiring-challenge/app/api"
	"github.com/mytheresa/go-hiring-challenge/models"
	"github.com/shopspring/decimal"
)

// Pagination defaults and bounds for the catalog listing.
const (
	defaultLimit = 10
	minLimit     = 1
	maxLimit     = 100
)

// ProductsRepository is the slice of the products storage the catalog needs.
// It is declared on the consumer side so the handler can be exercised with a
// fake, without reaching for a database.
type ProductsRepository interface {
	List(ctx context.Context, f models.ProductFilter) ([]models.Product, int64, error)
	FindByCode(ctx context.Context, code string) (*models.Product, error)
}

type Response struct {
	Products   []Product  `json:"products"`
	Pagination Pagination `json:"pagination"`
}

type Product struct {
	Code     string   `json:"code"`
	Price    float64  `json:"price"`
	Category Category `json:"category"`
}

// Category is the product's category as exposed by the API. The internal ID is
// deliberately left out: the code is the public identifier.
type Category struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// ProductDetails is a catalog product plus the variants only the detail
// endpoint exposes.
type ProductDetails struct {
	Product
	Variants []Variant `json:"variants"`
}

type Variant struct {
	Name  string  `json:"name"`
	SKU   string  `json:"sku"`
	Price float64 `json:"price"`
}

// Pagination tells the client where the page sits and how much there is to page
// through, so it can work out whether another request is worth making.
type Pagination struct {
	Offset int   `json:"offset"`
	Limit  int   `json:"limit"`
	Total  int64 `json:"total"`
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
	filter, err := parseFilter(r.URL.Query())
	if err != nil {
		// These messages are ours, not the driver's, so they are safe to return.
		api.ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	products, total, err := h.repo.List(r.Context(), filter)
	if err != nil {
		// The underlying error can name tables and columns, so it is logged
		// rather than handed to the client.
		log.Printf("catalog: listing products failed: %s", err)
		api.ErrorResponse(w, http.StatusInternalServerError, "could not retrieve the catalog")
		return
	}

	api.OKResponse(w, Response{
		Products: newProducts(products),
		Pagination: Pagination{
			Offset: filter.Offset,
			Limit:  filter.Limit,
			Total:  total,
		},
	})
}

func (h *CatalogHandler) HandleGetByCode(w http.ResponseWriter, r *http.Request) {
	product, err := h.repo.FindByCode(r.Context(), r.PathValue("code"))

	switch {
	case errors.Is(err, models.ErrNotFound):
		api.ErrorResponse(w, http.StatusNotFound, "product not found")
		return
	case err != nil:
		log.Printf("catalog: fetching product failed: %s", err)
		api.ErrorResponse(w, http.StatusInternalServerError, "could not retrieve the product")
		return
	}

	api.OKResponse(w, newProductDetails(*product))
}

// parseFilter reads the listing parameters off the query string. A missing
// parameter falls back to its default, while one that cannot be parsed is a
// client mistake and is reported as such rather than silently ignored.
func parseFilter(q url.Values) (models.ProductFilter, error) {
	f := models.ProductFilter{Limit: defaultLimit}

	if raw := q.Get("offset"); raw != "" {
		offset, err := strconv.Atoi(raw)
		if err != nil || offset < 0 {
			return models.ProductFilter{}, errors.New("offset must be a non-negative integer")
		}

		f.Offset = offset
	}

	if raw := q.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil {
			return models.ProductFilter{}, errors.New("limit must be an integer")
		}

		// Clamped rather than rejected: a limit out of range still describes a
		// page the client can be served.
		f.Limit = min(max(limit, minLimit), maxLimit)
	}

	f.Category = q.Get("category")

	if raw := q.Get("price_less_than"); raw != "" {
		price, err := decimal.NewFromString(raw)
		if err != nil {
			return models.ProductFilter{}, errors.New("price_less_than must be a number")
		}

		f.PriceLessThan = &price
	}

	return f, nil
}

// newProducts maps stored products onto their API representation.
func newProducts(products []models.Product) []Product {
	res := make([]Product, 0, len(products))
	for _, p := range products {
		res = append(res, newProduct(p))
	}

	return res
}

func newProduct(p models.Product) Product {
	return Product{
		Code:  p.Code,
		Price: p.Price.InexactFloat64(),
		Category: Category{
			Code: p.Category.Code,
			Name: p.Category.Name,
		},
	}
}

func newProductDetails(p models.Product) ProductDetails {
	return ProductDetails{
		Product:  newProduct(p),
		Variants: newVariants(p),
	}
}

// newVariants maps a product's variants, resolving the price of the ones that
// do not carry their own.
func newVariants(p models.Product) []Variant {
	res := make([]Variant, 0, len(p.Variants))
	for _, v := range p.Variants {
		// A variant without a price of its own inherits the product's. The
		// pointer is what keeps that case apart from a genuine 0.00.
		price := p.Price
		if v.Price != nil {
			price = *v.Price
		}

		res = append(res, Variant{
			Name:  v.Name,
			SKU:   v.SKU,
			Price: price.InexactFloat64(),
		})
	}

	return res
}
