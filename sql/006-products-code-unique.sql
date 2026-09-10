-- products.code is the catalog's public identifier and the lookup key for
-- GET /catalog/{code}, but 001 created the column nullable and unconstrained,
-- so nothing stopped two products from sharing a code and First() from
-- returning either one.
--
-- The sibling table already constrains its own identifier (product_variants.sku
-- is UNIQUE) and the gorm model has always declared `uniqueIndex;not null`, so
-- this brings the schema in line with what both already assume.
ALTER TABLE products ALTER COLUMN code SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_products_code ON products(code);
