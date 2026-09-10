# Go Hiring Challenge

This repository contains a Go application for managing products and their prices, including functionalities for CRUD operations and seeding the database with initial data.

## Project Structure

1. **cmd/**: Contains the main application and seed command entry points.

   - `server/main.go`: The main application entry point, serves the REST API.
   - `seed/main.go`: Command to seed the database with initial product data.

2. **app/**: Contains the application logic.
3. **sql/**: Contains a very simple database migration scripts setup.
4. **models/**: Contains the data models and repositories used in the application.
5. `.env`: Environment variables file for configuration.

## Setup Code Repository

1. Create a github/bitbucket/gitlab repository and push all this code as-is.
2. Create a new branch, and provide a pull-request against the main branch with your changes. Instructions to follow.

## Application Setup

- Ensure you have Go installed on your machine.
- Ensure you have Docker installed on your machine.
- Important makefile targets:
  - `make tidy`: will install all dependencies.
  - `make docker-up`: will start the required infrastructure services via docker containers.
  - `make seed`: ⚠️ Will destroy and re-create the database tables.
  - `make test`: Will run the tests.
  - `make run`: Will start the application.
  - `make docker-down`: Will stop the docker containers.

## API

The server listens on `HTTP_PORT` (`8484` by default). Every response is JSON, and
failures carry a single `error` field.

| Method | Path              | Description                             |
|--------|-------------------|-----------------------------------------|
| `GET`  | `/catalog`        | Paginated product listing, with filters |
| `GET`  | `/catalog/{code}` | A single product with its variants      |
| `GET`  | `/categories`     | Every category                          |
| `POST` | `/categories`     | Create a category                       |

### `GET /catalog`

| Parameter         | Default | Notes                            |
|-------------------|---------|----------------------------------|
| `offset`          | `0`     | Must not be negative             |
| `limit`           | `10`    | Clamped to the `1..100` range    |
| `category`        | —       | A category code, such as `SHOES` |
| `price_less_than` | —       | Exclusive upper bound            |

A parameter that cannot be parsed is answered with `400`, while a `limit` outside
its range is clamped rather than rejected.

```sh
curl 'http://localhost:8484/catalog?category=SHOES&price_less_than=30&limit=2'
```

```json
{
  "products": [
    { "code": "PROD002", "price": 12.49, "category": { "code": "SHOES", "name": "Shoes" } },
    { "code": "PROD006", "price": 5.5, "category": { "code": "SHOES", "name": "Shoes" } }
  ],
  "pagination": { "offset": 0, "limit": 2, "total": 2 }
}
```

`total` counts every product matching the filter, not the ones on the page, so a
client can tell whether another request is worth making.

### `GET /catalog/{code}`

Returns `404` when the code matches nothing. A variant without a price of its own
inherits the product's — `Variant B` and `Variant C` below.

```sh
curl http://localhost:8484/catalog/PROD001
```

```json
{
  "code": "PROD001",
  "price": 10.99,
  "category": { "code": "CLOTHING", "name": "Clothing" },
  "variants": [
    { "name": "Variant A", "sku": "SKU001A", "price": 11.99 },
    { "name": "Variant B", "sku": "SKU001B", "price": 10.99 },
    { "name": "Variant C", "sku": "SKU001C", "price": 10.99 }
  ]
}
```

### `GET /categories`

```sh
curl http://localhost:8484/categories
```

```json
{
  "categories": [
    { "code": "ACCESSORIES", "name": "Accessories" },
    { "code": "CLOTHING", "name": "Clothing" },
    { "code": "SHOES", "name": "Shoes" }
  ]
}
```

### `POST /categories`

```sh
curl -X POST http://localhost:8484/categories -d '{"code": "BAGS", "name": "Bags"}'
```

```json
{ "code": "BAGS", "name": "Bags" }
```

| Status | When                                                                              |
|--------|-----------------------------------------------------------------------------------|
| `201`  | Created                                                                           |
| `400`  | The body is malformed, carries an unknown field, or leaves `code` or `name` blank |
| `409`  | A category with that code already exists                                          |

Follow up for the assignment here: [ASSIGNMENT.md](ASSIGNMENT.md)
