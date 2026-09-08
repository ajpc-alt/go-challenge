package categories

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/mytheresa/go-hiring-challenge/app/api"
	"github.com/mytheresa/go-hiring-challenge/models"
)

// CategoriesRepository is the slice of the categories storage this handler
// needs. As in the catalog, it is declared on the consumer side so the handler
// can be exercised with a fake, without reaching for a database.
type CategoriesRepository interface {
	List(ctx context.Context) ([]models.Category, error)
	Create(ctx context.Context, c *models.Category) error
}

type Response struct {
	Categories []Category `json:"categories"`
}

// Category is both the API representation of a category and the shape accepted
// when creating one. The internal ID is left out: the code is the public
// identifier, and letting a client pick an ID is not the point of the endpoint.
type Category struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type CategoriesHandler struct {
	repo CategoriesRepository
}

func NewCategoriesHandler(r CategoriesRepository) *CategoriesHandler {
	return &CategoriesHandler{
		repo: r,
	}
}

func (h *CategoriesHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	categories, err := h.repo.List(r.Context())
	if err != nil {
		// The underlying error can name tables and columns, so it is logged
		// rather than handed to the client.
		log.Printf("categories: listing categories failed: %s", err)
		api.ErrorResponse(w, http.StatusInternalServerError, "could not retrieve the categories")
		return
	}

	api.OKResponse(w, Response{Categories: newCategories(categories)})
}

func (h *CategoriesHandler) HandlePost(w http.ResponseWriter, r *http.Request) {
	var body Category

	dec := json.NewDecoder(r.Body)
	// Unknown fields are rejected rather than dropped: a client sending "nmae"
	// should hear about it, not get back a category without a name.
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		api.ErrorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	category := models.Category{
		Code: strings.TrimSpace(body.Code),
		Name: strings.TrimSpace(body.Name),
	}

	// Both columns are NOT NULL, but empty strings would satisfy that and leave
	// a category nothing can refer to, so they are turned away here.
	if category.Code == "" || category.Name == "" {
		api.ErrorResponse(w, http.StatusBadRequest, "code and name are required")
		return
	}

	err := h.repo.Create(r.Context(), &category)

	switch {
	case errors.Is(err, models.ErrDuplicate):
		// 409 rather than 400: the body is well formed, it just conflicts with
		// a category that already exists.
		api.ErrorResponse(w, http.StatusConflict, "a category with this code already exists")
		return
	case err != nil:
		log.Printf("categories: creating category failed: %s", err)
		api.ErrorResponse(w, http.StatusInternalServerError, "could not create the category")
		return
	}

	api.CreatedResponse(w, newCategory(category))
}

// newCategories maps stored categories onto their API representation.
func newCategories(categories []models.Category) []Category {
	res := make([]Category, 0, len(categories))
	for _, c := range categories {
		res = append(res, newCategory(c))
	}

	return res
}

func newCategory(c models.Category) Category {
	return Category{
		Code: c.Code,
		Name: c.Name,
	}
}
