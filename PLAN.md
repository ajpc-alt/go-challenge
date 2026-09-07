# Implementation Plan — Go Hiring Challenge

Working document for the execution of [ASSIGNMENT.md](ASSIGNMENT.md).
It captures the starting state, the design decisions taken, and the phase-by-phase breakdown.

---

## 1. Starting state

Analysis of the code as it stands before starting. Every item here drives a decision further down.

| Piece | Current state | Implication |
| --- | --- | --- |
| Router | `net/http` stdlib with Go 1.22+ patterns (`GET /catalog`) | The assignment writes `/catalog/:code`, but this router's syntax is `GET /catalog/{code}` + `r.PathValue("code")` |
| `app/api/response.go` | Empty functions, tests already written | The tests define the contract: `200` + `Content-Type: application/json` for `OKResponse`, and `{"error":"..."}` for `ErrorResponse` |
| `app/catalog/handler.go` | Depends on the concrete struct `*models.ProductsRepository`, serialises JSON by hand, no `context` | Exactly what task *Catalog #1* asks to refactor; it also makes the handler untestable without a database |
| `models.Variant.Price` | `decimal.Decimal`, **not a pointer**, over a `NULL`able column | `NULL` and `0.00` are indistinguishable after the scan. This blocks the price inheritance rule |
| `models.ProductsRepository` | Single `GetAllProducts()` method, always `Preload("Variants")`, no `ctx`, no filters | Needs to grow filtering, pagination and lookup by code |
| CI (`.github/workflows/go.yml`) | Runs `go build ./...` and `go test -race ./...` **without starting Postgres** | **Hard constraint**: the whole test suite must be unit-level and run without a database |
| Migrations | `sql/000-truncate.sql` drops everything; `001`/`002` schema; `003` data. Executed in alphabetical order | New migrations are appended with the next numbers, leaving the existing ones untouched |
| Git | Branch `main`, a single commit, no remote configured | A branch, a remote and a PR are still needed |

---

## 2. Design decisions

Decisions taken by default. The ones marked with :grey_question: are those where another reading is equally defensible.

### 2.1 The consumer declares the interface

`app/catalog` defines the interface it needs; `models` keeps exporting concrete structs.

```go
// app/catalog/handler.go
type ProductsRepository interface {
    List(ctx context.Context, f models.ProductFilter) ([]models.Product, int64, error)
    FindByCode(ctx context.Context, code string) (*models.Product, error)
}
```

This is the idiomatic approach in Go (*accept interfaces, return structs*): the interface belongs to whoever consumes it, stays minimal, and does not force `models` to know about its clients. It is also what allows a fake to be injected in tests, satisfying the no-database CI constraint.

`models.ProductFilter` lives in `models`, not in `catalog`: it is a data-access concern, and placing it in `catalog` would create an import cycle (`models` → `catalog` → `models`).

### 2.2 `context.Context` on every repository method

Methods take a `ctx` and propagate it with `db.WithContext(ctx)`, starting from `r.Context()` in the handler. A cancelled HTTP request then aborts the Postgres query as well.

### 2.3 `Variant.Price` becomes `*decimal.Decimal`

A **required** change, not a cosmetic one. The rule *"variants without specific price should inherit the price from the product"* cannot be implemented with a non-pointer value: GORM scans `NULL` into a zero `decimal.Decimal`, which is indistinguishable from a genuine price of `0.00`. With a pointer, `nil` unambiguously means "no price of its own".

### 2.4 Append-only migrations

`sql/004-categories.sql` and `sql/005-category-data.sql` are added without modifying the existing files, following the repository's pattern. The alternative — editing `001` and `003`, since `make seed` rebuilds the database from scratch — would be more compact but breaks the convention the assignment explicitly asks to follow.

### 2.5 The category filter uses `code` :grey_question:

The assignment defines `Code` as a *"human-readable unique identifier"* and `ID` as *"internal use only"*, so the public API filters by code: `?category=SHOES`. The `ID` is never exposed in a response.

### 2.6 The listing stops preloading variants

`GET /catalog` does not include variants in its response, so `Preload("Variants")` is dropped from that query and kept only in the detail endpoint. With the current dataset that is 24 extra rows per page; with a real catalogue, hundreds per product.

### 2.7 Validation of pagination parameters :grey_question:

- Absent → default value (`offset=0`, `limit=10`).
- Not parseable as an integer, or a negative `offset` → `400 Bad Request` (a client error; swallowing it hides bugs).
- `limit` out of range → clamped to `[1, 100]` rather than rejected.

### 2.8 Prices are serialised as `float64` :grey_question:

`InexactFloat64()` is kept, as the current handler already does, for consistency with the existing contract. **Deliberate debt**: monetary amounts should be serialised as strings to avoid binary rounding. Documented in the PR instead of being changed unilaterally.

### 2.9 `409 Conflict` when creating a category with a duplicate code :grey_question:

`code` has a unique index. `gorm.Config{TranslateError: true}` is enabled in `app/database/pg.go` so the error can be identified with `errors.Is(err, gorm.ErrDuplicatedKey)` and answered with a `409` instead of a generic `500`. The alternative would be `400`; `409` better describes a conflict with the state of an existing resource.

---

## 3. API contract

### `GET /catalog`

Query params: `offset` (default `0`), `limit` (default `10`, range `1..100`), `category` (code), `price_less_than`.

```json
{
  "products": [
    {
      "code": "PROD001",
      "price": 10.99,
      "category": { "code": "CLOTHING", "name": "Clothing" }
    }
  ],
  "pagination": { "offset": 0, "limit": 10, "total": 8 }
}
```

`products` is always an array, never `null`, even with no results.

### `GET /catalog/{code}`

```json
{
  "code": "PROD001",
  "price": 10.99,
  "category": { "code": "CLOTHING", "name": "Clothing" },
  "variants": [
    { "name": "Variant A", "sku": "SKU001A", "price": 11.99 },
    { "name": "Variant B", "sku": "SKU001B", "price": 10.99 }
  ]
}
```

`Variant B` has no price of its own in the database and inherits the product's. `404` if the code does not exist.

### `GET /categories`

```json
{ "categories": [ { "code": "CLOTHING", "name": "Clothing" } ] }
```

### `POST /categories`

Body `{"code": "BAGS", "name": "Bags"}` → `201 Created` with the created category.

Errors: `400` (invalid JSON or empty fields), `409` (duplicate code), `500` (internal error).

---

## 4. Phases

Each phase is an atomic commit that compiles and leaves the suite green. See §5 on why they are not separate PRs.

### Phase 0 — Setup

- [ ] `git checkout -b feature/catalog-categories-pagination`
- [ ] `make tidy && make docker-up && make seed && make test`
- [ ] Confirm the baseline: the two `app/api` tests fail because the functions are empty. That is expected.

### Phase 1 — `app/api/response.go`

Goes first because every other handler consumes it.

- [ ] Unexported `writeJSON(w, status, payload)` helper, in the right order: `Header().Set(...)` **before** `WriteHeader(...)`, or the `Content-Type` is dropped.
- [ ] `OKResponse(w, data)` → `200` + `application/json`.
- [ ] `ErrorResponse(w, status, message)` → `status` + `{"error": message}`.
- [ ] `CreatedResponse(w, data)` → `201`, on the same helper, for `POST /categories`. The provided tests keep passing unchanged.
- [ ] `go test ./app/api/...` green.

*Covers: Testing #2 (first half).*

### Phase 2 — Idiomatic catalog refactor

- [ ] Declare the `ProductsRepository` interface in `app/catalog`; `NewCatalogHandler` takes it instead of the concrete type.
- [ ] Define `models.ProductFilter` (offset, limit, optional category, optional max price).
- [ ] Add `ctx` to the repository methods.
- [ ] Replace the hand-rolled `json.NewEncoder` and the `http.Error` calls with `api.OKResponse` / `api.ErrorResponse`.
- [ ] Add `models.ErrNotFound` and translate `gorm.ErrRecordNotFound` in the repository, so the handler can return `404` without knowing about GORM.
- [ ] `gorm.Config{TranslateError: true}` in `app/database/pg.go`.

*Covers: Catalog #1, Testing #2 (second half).*

### Phase 3 — Category model and migrations

- [ ] `sql/004-categories.sql`: `CREATE TABLE categories` (id, unique code, name, timestamps), `ALTER TABLE products ADD COLUMN category_id INTEGER REFERENCES categories(id)`, index on `category_id`.
- [ ] `sql/005-category-data.sql`: insert `CLOTHING`, `SHOES`, `ACCESSORIES`; `UPDATE products SET category_id = ...` with the assignment's mapping (PROD001/004/007 → Clothing, PROD002/006 → Shoes, PROD003/005/008 → Accessories); and finally `ALTER COLUMN category_id SET NOT NULL`.
  > The backfill comes before the constraint: `products` already holds the rows inserted by `003`, so the column must start out nullable.
- [ ] `models/categories.go`: `Category{ID, Code, Name}` struct + `TableName()` with a pointer receiver, matching `Product` and `Variant`.
- [ ] `models/products.go`: `CategoryID uint` + `Category Category` with `foreignKey:CategoryID`.
- [ ] `models/categories_repository.go`: `List(ctx)`, `Create(ctx, *Category)`.
- [ ] Verify with `make seed` and a manual query.

*Covers: Catalog #2.*

### Phase 4 — Category, pagination and filters in `/catalog`

- [ ] Repository `List(ctx, filter)`: base query with `Joins("Category")`, plus `WHERE categories.code = ?` and `products.price < ?` depending on the filters.
- [ ] `Count` over the query **already filtered but without `LIMIT`/`OFFSET`**, then apply `Offset`/`Limit`.
- [ ] Explicit `Order("products.id")`: without an `ORDER BY`, Postgres guarantees no stable ordering and pagination can repeat or skip rows across pages.
- [ ] Handler: extract `parseFilter(r) (models.ProductFilter, error)` as a separately testable function.
- [ ] Map the response with the `pagination` block, and `make([]Product, 0, len(res))` so an empty result never serialises as `null`.

*Covers: Catalog #3, #4, #5.*

### Phase 5 — `GET /catalog/{code}`

- [ ] Repository `FindByCode`: `Preload("Variants").Joins("Category").First(...)`, translating to `models.ErrNotFound`.
- [ ] `HandleGetByCode` handler: `404` when missing, `500` on a repository error.
- [ ] Price inheritance in the mapping: `if v.Price == nil { price = p.Price }`. Pure logic, no dependencies, trivial to test.
- [ ] Register `GET /catalog/{code}` in `cmd/server/main.go`.

*Covers: Product details #1 (implementation).*

### Phase 6 — `/categories`

- [ ] `app/categories` package with its own consumer-side interface (`List`, `Create`).
- [ ] `GET /categories` → `api.OKResponse`.
- [ ] `POST /categories`: decode with `DisallowUnknownFields`, validate that `code` and `name` are non-empty after `TrimSpace`, answer `201` with the created category.
- [ ] Error mapping: `400` for invalid JSON or empty fields, `409` for a duplicate, `500` for the rest.
- [ ] Register both routes in `cmd/server/main.go`.

*Covers: Categories #1, #2 (implementation).*

### Phase 7 — Tests

All of them against fakes implementing the handlers' interfaces. **No dependency on Postgres**, compatible with the current CI.

- [ ] `app/catalog/handler_test.go`, table-driven:
  - pagination defaults (`offset=0`, `limit=10`)
  - `limit` clamped to `1` and to `100`
  - non-numeric parameters and negative `offset` → `400`
  - filtering by category, by price, and both combined
  - `total` correct and independent of the page
  - empty result → `"products": []`
  - repository error → `500`
- [ ] `app/catalog/handler_details_test.go`: product with category and variants, **price inheritance for a variant without its own price**, `404`, `500`.
- [ ] `app/categories/handler_test.go`: listing, `201` creation, `400` (malformed JSON and empty fields), `409` duplicate, `500`.
- [ ] `make test` green under `-race`; review `coverage.out`.

*Covers: Testing #1, plus the tests for Product details #1 and Categories #1/#2.*

### Phase 8 — Wrap-up

- [ ] Update `README.md` with the endpoint table, parameters and `curl` examples.
- [ ] Review the full diff before publishing.
- [ ] Push the branch and open the PR against `main`, documenting the decisions and trade-offs (§2).

---

## 5. Delivery strategy: a single PR

**Phases are commits, not PRs.** `README.md` asks for one branch and one pull request against `main`. Three reasons back this up:

1. **The phases are not independent deliverables.** Phase 2 leaves the code compiling but adds no new functionality. Phase 3 introduces `category_id NOT NULL` and only makes sense together with Phase 4, which consumes it. A PR should be mergeable and coherent on its own; these are not.
2. **The review is of the whole.** Splitting into eight PRs over a repository with no prior history forces the reviewer to reconstruct the work by jumping between diffs.
3. **There is no CI with a database and no incremental deployment environment**, which is the usual reason to split a delivery.

Eight atomic commits, each compiling with a green suite, make the PR reviewable commit by commit without sacrificing the coherence of the whole.

The only defensible alternative split would be a preliminary PR with `Phase 1 + Phase 2` (a pure refactor, with no observable behaviour change) followed by the rest. For the scope of this challenge it adds nothing.

---

## 6. Risks and things to watch

| Risk | Mitigation |
| --- | --- |
| A misplaced `Count` returns the page total rather than the catalogue total | Count over the filtered query before applying `LIMIT`/`OFFSET`; test with `total > limit` |
| Non-deterministic pagination without `ORDER BY` | Explicit `Order("products.id")` |
| A variant's `NULL` confused with `0.00` | `*decimal.Decimal` (§2.3), with a dedicated inheritance test |
| Tests that need a database break CI | The whole suite runs against fakes of the interfaces |
| `Content-Type` ignored in `response.go` | Headers before `WriteHeader`; already covered by the provided tests |
| `SET NOT NULL` fails if a product is left without a category | The backfill `UPDATE` runs first and covers all eight products |
| Serialising `null` instead of `[]` | `make([]T, 0, n)` in every list mapping |
