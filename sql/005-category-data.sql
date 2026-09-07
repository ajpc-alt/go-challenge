-- Insert the 3 categories
INSERT INTO categories (code, name) VALUES
('CLOTHING', 'Clothing'),
('SHOES', 'Shoes'),
('ACCESSORIES', 'Accessories');

-- Link each product to its category using the product code

UPDATE products SET category_id = (SELECT id FROM categories WHERE code = 'CLOTHING')
WHERE code IN ('PROD001', 'PROD004', 'PROD007');

UPDATE products SET category_id = (SELECT id FROM categories WHERE code = 'SHOES')
WHERE code IN ('PROD002', 'PROD006');

UPDATE products SET category_id = (SELECT id FROM categories WHERE code = 'ACCESSORIES')
WHERE code IN ('PROD003', 'PROD005', 'PROD008');

-- Every product is linked now, so the column can be constrained
ALTER TABLE products ALTER COLUMN category_id SET NOT NULL;
