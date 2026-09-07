CREATE TABLE IF NOT EXISTS categories (
    id SERIAL PRIMARY KEY,
    code VARCHAR(32) UNIQUE NOT NULL,
    name VARCHAR(256) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Added as nullable: products already hold rows at this point. The backfill and
-- the NOT NULL constraint come in the next migration.
-- ON DELETE RESTRICT: removing a category must not take its products with it.
ALTER TABLE products
    ADD COLUMN IF NOT EXISTS category_id INTEGER REFERENCES categories(id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_products_category_id ON products(category_id);
