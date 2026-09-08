package categories

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mytheresa/go-hiring-challenge/models"
	"github.com/stretchr/testify/assert"
)

// fakeCategoriesRepository stands in for the storage, recording what it was
// asked to store so the request handling can be checked.
type fakeCategoriesRepository struct {
	categories []models.Category
	err        error

	createCalled bool
	created      models.Category
}

func (f *fakeCategoriesRepository) List(_ context.Context) ([]models.Category, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.categories, nil
}

func (f *fakeCategoriesRepository) Create(_ context.Context, c *models.Category) error {
	f.createCalled = true
	f.created = *c

	if f.err != nil {
		return f.err
	}

	// The real repository fills in the generated ID, so the fake does too.
	c.ID = 42

	return nil
}

func list(repo *fakeCategoriesRepository) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	NewCategoriesHandler(repo).HandleGet(recorder, httptest.NewRequest(http.MethodGet, "/categories", nil))

	return recorder
}

func create(repo *fakeCategoriesRepository, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	NewCategoriesHandler(repo).HandlePost(recorder, httptest.NewRequest(http.MethodPost, "/categories", strings.NewReader(body)))

	return recorder
}

func TestHandleGet(t *testing.T) {
	t.Run("returns the categories", func(t *testing.T) {
		repo := &fakeCategoriesRepository{
			categories: []models.Category{
				{ID: 1, Code: "CLOTHING", Name: "Clothing"},
				{ID: 2, Code: "SHOES", Name: "Shoes"},
			},
		}

		recorder := list(repo)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
		// The internal ID is not part of the response.
		assert.JSONEq(t, `{"categories": [
			{"code": "CLOTHING", "name": "Clothing"},
			{"code": "SHOES", "name": "Shoes"}
		]}`, recorder.Body.String())
	})

	t.Run("serialises an empty list as an array, never as null", func(t *testing.T) {
		recorder := list(&fakeCategoriesRepository{})

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.JSONEq(t, `{"categories": []}`, recorder.Body.String())
	})

	t.Run("answers 500 without leaking the repository error", func(t *testing.T) {
		recorder := list(&fakeCategoriesRepository{err: errors.New(`pq: relation "categories" does not exist`)})

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.JSONEq(t, `{"error": "could not retrieve the categories"}`, recorder.Body.String())
		assert.NotContains(t, recorder.Body.String(), "relation")
	})
}

func TestHandlePost(t *testing.T) {
	t.Run("creates the category and answers 201", func(t *testing.T) {
		repo := &fakeCategoriesRepository{}

		recorder := create(repo, `{"code": "BAGS", "name": "Bags"}`)

		assert.Equal(t, http.StatusCreated, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"code": "BAGS", "name": "Bags"}`, recorder.Body.String())
		assert.Equal(t, models.Category{Code: "BAGS", Name: "Bags"}, repo.created)
	})

	t.Run("trims the surrounding whitespace before storing", func(t *testing.T) {
		repo := &fakeCategoriesRepository{}

		recorder := create(repo, `{"code": "  BAGS  ", "name": "  Bags  "}`)

		assert.Equal(t, http.StatusCreated, recorder.Code)
		assert.Equal(t, models.Category{Code: "BAGS", Name: "Bags"}, repo.created)
	})

	t.Run("answers 409 when the code is already taken", func(t *testing.T) {
		recorder := create(&fakeCategoriesRepository{err: models.ErrDuplicate}, `{"code": "SHOES", "name": "Shoes"}`)

		assert.Equal(t, http.StatusConflict, recorder.Code)
		assert.JSONEq(t, `{"error": "a category with this code already exists"}`, recorder.Body.String())
	})

	t.Run("answers 500 without leaking the repository error", func(t *testing.T) {
		recorder := create(&fakeCategoriesRepository{err: errors.New(`pq: relation "categories" does not exist`)}, `{"code": "BAGS", "name": "Bags"}`)

		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.JSONEq(t, `{"error": "could not create the category"}`, recorder.Body.String())
		assert.NotContains(t, recorder.Body.String(), "relation")
	})
}

func TestHandlePostInvalidBody(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "malformed JSON",
			body: `{"code": "BAGS",`,
			want: "invalid request body",
		},
		{
			name: "an empty body",
			body: ``,
			want: "invalid request body",
		},
		{
			name: "a field of the wrong type",
			body: `{"code": 123, "name": "Bags"}`,
			want: "invalid request body",
		},
		{
			name: "an unknown field, which is usually a typo",
			body: `{"code": "BAGS", "nmae": "Bags"}`,
			want: "invalid request body",
		},
		{
			name: "the internal ID, which is not the client's to pick",
			body: `{"id": 99, "code": "BAGS", "name": "Bags"}`,
			want: "invalid request body",
		},
		{
			name: "a missing name",
			body: `{"code": "BAGS"}`,
			want: "code and name are required",
		},
		{
			name: "an empty code",
			body: `{"code": "", "name": "Bags"}`,
			want: "code and name are required",
		},
		{
			name: "a name that is only whitespace",
			body: `{"code": "BAGS", "name": "   "}`,
			want: "code and name are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeCategoriesRepository{}

			recorder := create(repo, tt.body)

			assert.Equal(t, http.StatusBadRequest, recorder.Code)
			assert.JSONEq(t, `{"error": "`+tt.want+`"}`, recorder.Body.String())
			// The request never reaches the storage: it was turned away first.
			assert.False(t, repo.createCalled)
		})
	}
}
