# Design Notes — Go Hiring Challenge

Why the code on this branch looks the way it does: the state it started from, the
decisions taken along the way, and the failure modes they guard against.

The endpoints themselves are documented in [README.md](README.md); the tasks they
answer to, in [ASSIGNMENT.md](ASSIGNMENT.md).

---

## 1. Starting state

Analysis of the code as it stood before starting. Every item here drives a
decision further down.

| Piece                           | State                                                                                                   | Implication                                                                                                                         |
|---------------------------------|---------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------|
| Router                          | `net/http` stdlib with Go 1.22+ patterns (`GET /catalog`)                                               | The assignment writes `/catalog/:code`, but this router's syntax is `GET /catalog/{code}` + `r.PathValue("code")`                   |
| `app/api/response.go`           | Empty functions, tests already written                                                                  | The tests define the contract: `200` + `Content-Type: application/json` for `OKResponse`, and `{"error":"..."}` for `ErrorResponse` |
| `app/catalog/handler.go`        | Depends on the concrete struct `*models.ProductsRepository`, serialises JSON by hand, no `context`      | Exactly what task *Catalog #1* asks to refactor; it also makes the handler untestable without a database                            |
| `models.Variant.Price`          | `decimal.Decimal`, **not a pointer**, over a `NULL`able column                                          | `NULL` and `0.00` are indistinguishable after the scan. This blocks the price inheritance rule                                      |
| `models.ProductsRepository`     | Single `GetAllProducts()` method, always `Preload("Variants")`, no `ctx`, no filters                    | Needs to grow filtering, pagination and lookup by code                                                                              |
| `products.code`                 | Declared `uniqueIndex;not null` on the model, but nullable and unconstrained in `sql/001`               | The tag is decorative: the schema comes from the SQL files, not from `AutoMigrate`                                                  |
| CI (`.github/workflows/go.yml`) | Runs `go build ./...` and `go test -race ./...` **without starting Postgres**                           | **Hard constraint**: the whole test suite must be unit-level and run without a database                                             |
| Migrations                      | `sql/000-truncate.sql` drops everything; `001`/`002` schema; `003` data. Executed in alphabetical order | New migrations are appended with the next numbers, leaving the existing ones untouched                                              |

---

## 2. Design decisions

Decisions taken by default. The ones marked with :grey_question: are those where
another reading is equally defensible.

### 2.1 The consumer declares the interface

`app/catalog` and `app/categories` each define the interface they need; `models`
keeps exporting concrete structs.

```go
// app/catalog/handler.go
type ProductsRepository interface {
    List(ctx context.Context, f models.ProductFilter) ([]models.Product, int64, error)
    FindByCode(ctx context.Context, code string) (*models.Product, error)
}
```

This is the idiomatic approach in Go (*accept interfaces, return structs*): the
interface belongs to whoever consumes it, stays minimal, and does not force
`models` to know about its clients. It is also what allows a fake to be injected
in tests, satisfying the no-database CI constraint.

`models.ProductFilter` lives in `models`, not in `catalog`: it is a data-access
concern, and placing it in `catalog` would create an import cycle
(`models` → `catalog` → `models`).

### 2.2 `context.Context` on every repository method

Methods take a `ctx` and propagate it with `db.WithContext(ctx)`, starting from
`r.Context()` in the handler. A cancelled HTTP request then aborts the Postgres
query as well.

### 2.3 The ORM stops at the repository

`gorm.ErrRecordNotFound` and `gorm.ErrDuplicatedKey` are translated into
`models.ErrNotFound` and `models.ErrDuplicate`, so handlers map errors onto status
codes without importing GORM. `gorm.Config{TranslateError: true}` is enabled in
`app/database/pg.go`, which is what makes the second one identifiable at all:
without it the driver's unique-violation arrives as an opaque error.

The alternative — reading GORM's sentinels straight from the handler — would tie
every handler to the ORM and make the `409` path untestable without a database.

### 2.4 `Variant.Price` becomes `*decimal.Decimal`

A **required** change, not a cosmetic one. The rule *"variants without specific
price should inherit the price from the product"* cannot be implemented with a
non-pointer value: GORM scans `NULL` into a zero `decimal.Decimal`, which is
indistinguishable from a genuine price of `0.00`. With a pointer, `nil`
unambiguously means "no price of its own".

### 2.5 Append-only migrations

`sql/004` through `sql/006` are added without modifying the existing files,
following the repository's pattern. The alternative — editing `001` and `003`,
since `make seed` rebuilds the database from scratch — would be more compact but
breaks the convention the assignment explicitly asks to follow.

`category_id` is added nullable because `003` has already inserted the products;
`005` backfills it and only then applies `SET NOT NULL`. The foreign key is
`ON DELETE RESTRICT`, unlike the `CASCADE` on variants: deleting a category must
not delete its products.

### 2.6 `products.code` gets the constraint its model already claims :grey_question:

`sql/006` adds the `UNIQUE` index and `NOT NULL` that `sql/001` left out. Strictly
speaking this is outside the assignment, but `FindByCode` — added by this
branch — depends on the code identifying one product: `First()` over a duplicate
returns an arbitrary row, silently. The sibling table already constrains its own
identifier (`product_variants.sku`), so this is an omission rather than a design
stance.

### 2.7 The category filter uses `code` :grey_question:

The assignment defines `Code` as a *"human-readable unique identifier"* and `ID`
as *"internal use only"*, so the public API filters by code: `?category=SHOES`.
The `ID` is never exposed in a response, nor accepted when creating a category.

### 2.8 The listing stops preloading variants

`GET /catalog` does not include variants in its response, so `Preload("Variants")`
is dropped from that query and kept only in the detail endpoint. With the current
dataset that is 22 extra rows per page; with a real catalogue, hundreds per
product.

The listing loads the category with `Joins` rather than `Preload`, which resolves
it in the same query instead of a second one. The count runs over the filtered
query **before** `LIMIT`/`OFFSET`, so the total describes the catalogue and not
the page.

### 2.9 Validation of pagination parameters :grey_question:

- Absent → default value (`offset=0`, `limit=10`).
- Not parseable as an integer, or a negative `offset` → `400 Bad Request` (a
  client error; swallowing it hides bugs).
- `limit` out of range → clamped to `[1, 100]` rather than rejected, since it
  still describes a page that can be served.

### 2.10 Prices are serialised as `float64` :grey_question:

`InexactFloat64()` is kept, as the original handler already did, for consistency
with the existing contract. **Deliberate debt**: monetary amounts should be
serialised as strings to avoid binary rounding, and the effect is visible in the
responses — `15.00` comes back as `15`. Documented rather than changed
unilaterally, because it would break the contract the endpoint already had.

### 2.11 `409 Conflict` when creating a category with a duplicate code :grey_question:

The body is well formed; it just conflicts with a category that already exists.
The alternative would be `400`; `409` better describes a conflict with the state
of an existing resource.

---

## 3. Failure modes and how they are handled

| Failure mode                                                               | How it is prevented                                                                         |
|----------------------------------------------------------------------------|---------------------------------------------------------------------------------------------|
| A misplaced `Count` returns the page total rather than the catalogue total | Count over the filtered query before applying `LIMIT`/`OFFSET`; tested with `total > limit` |
| Non-deterministic pagination without `ORDER BY`                            | Explicit `Order("products.id")`                                                             |
| A variant's `NULL` confused with `0.00`                                    | `*decimal.Decimal` (§2.4), with tests for both inheritance and a genuinely free variant     |
| Tests that need a database break CI                                        | The whole suite runs against fakes of the interfaces                                        |
| `Content-Type` ignored in `response.go`                                    | Headers set before `WriteHeader`; covered by the provided tests                             |
| `SET NOT NULL` fails if a product is left without a category               | The backfill `UPDATE` runs first and covers all eight products                              |
| Serialising `null` instead of `[]`                                         | `make([]T, 0, n)` in every list mapping, with a test per endpoint                           |
| A query failure leaking table and column names to the client               | Repository errors are logged; the response carries a generic message                        |
