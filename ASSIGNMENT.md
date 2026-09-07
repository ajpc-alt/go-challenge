# Instructions

Once we make sure we get all the requirements from [README.md](README.md), we will proceed to code implementation. The main objective is to keep code implementation as simple as possible, while ensuring we follow best practices and maintainability.

Writing Idiomatic Go code is crucial, so we will focus on clarity and simplicity.

## Tasks

### Catalog endpoint

1. Catalog endpoint depends on `models.ProductsRepository` to fetch products. Refactor this in a more idiomatic way.

2. Create a new model for Product Categories, and make sure products are linked to them based on the following:

   - Categories will have the following fields:
     - ID (internal use only)
     - Code (human-readable unique identifier)
     - Name (human-readable name)
   - There will be 3 categories: "Clothing", "Shoes", and "Accessories".
     - _PROD001, PROD004, PROD007_ will belong to "Clothing".
     - _PROD002, PROD006_ will belong to "Shoes".
     - _PROD003, PROD005, PROD008_ will belong to "Accessories".
   - Follow the pattern introduced for the migrations files, and the implementation of the gorm models.

3. Update the catalog handler and relevant repositories to include the product category in the response.

4. Update the catalog handler and relevant repositories to support offset pagination.

   - The endpoint should accept query parameters `offset` and `limit`.
   - If `offset` is not provided, default to 0.
   - If `limit` is not provided, default to 10. Maximum limit should be 100. Minimum limit should be 1.
   - The response should include the total number of products available.

5. Update the catalog handler to support filtering products by:

   - Category
   - Price Less Than

### Product details endpoint

1. Implement the product details endpoint at `/catalog/:code`.

- This endpoint should return the product details including its variants. Do note that variants without specific price should inherit the price from the product.
- The product details should include the product's category.
- Provide unit tests for this endpoint.

### Categories endpoint

1. Implement the categories endpoint at `/categories`.

- This endpoint should return a list of all categories.
- Provide unit tests for this endpoint.

2. Implement an endpoint to create new categories at `/categories`.

- This endpoint should accept a JSON body with the category details from the category model and create a new entry in the DB.
- Provide unit tests for this endpoint.

### Testing

1. Provide unit tests for `app/catalog/handler.go`. Make sure to cover the new features implemented in the catalog endpoint.

2. Implement the functions in `app/api/response.go` to satisfy the provided unit tests and refactor all handlers to use these functions where appropriate.
