-- Insert 8 products
INSERT INTO products (code, price) VALUES
('PROD001', 10.99),
('PROD002', 12.49),
('PROD003', 8.75),
('PROD004', 15.00),
('PROD005', 22.99),
('PROD006', 5.50),
('PROD007', 18.20),
('PROD008', 9.99);

-- Insert variants for each product using product code to look up product_id

-- Product 1: 3 variants
INSERT INTO product_variants (product_id, name, sku, price) VALUES
((SELECT id FROM products WHERE code = 'PROD001'), 'Variant A', 'SKU001A', 11.99),
((SELECT id FROM products WHERE code = 'PROD001'), 'Variant B', 'SKU001B', NULL),
((SELECT id FROM products WHERE code = 'PROD001'), 'Variant C', 'SKU001C', NULL);

-- Product 2: 2 variants
INSERT INTO product_variants (product_id, name, sku, price) VALUES
((SELECT id FROM products WHERE code = 'PROD002'), 'Variant A', 'SKU002A', NULL),
((SELECT id FROM products WHERE code = 'PROD002'), 'Variant B', 'SKU002B', NULL);

-- Product 3: 1 variant
INSERT INTO product_variants (product_id, name, sku, price) VALUES
((SELECT id FROM products WHERE code = 'PROD003'), 'Variant A', 'SKU003A', 8.99);

-- Product 4: 4 variants
INSERT INTO product_variants (product_id, name, sku, price) VALUES
((SELECT id FROM products WHERE code = 'PROD004'), 'Variant A', 'SKU004A', 15.50),
((SELECT id FROM products WHERE code = 'PROD004'), 'Variant B', 'SKU004B', 16.00),
((SELECT id FROM products WHERE code = 'PROD004'), 'Variant C', 'SKU004C', NULL),
((SELECT id FROM products WHERE code = 'PROD004'), 'Variant D', 'SKU004D', 16.99);

-- Product 5: 6 variants
INSERT INTO product_variants (product_id, name, sku, price) VALUES
((SELECT id FROM products WHERE code = 'PROD005'), 'Variant A', 'SKU005A', 23.99),
((SELECT id FROM products WHERE code = 'PROD005'), 'Variant B', 'SKU005B', NULL),
((SELECT id FROM products WHERE code = 'PROD005'), 'Variant C', 'SKU005C', NULL),
((SELECT id FROM products WHERE code = 'PROD005'), 'Variant D', 'SKU005D', 22.99),
((SELECT id FROM products WHERE code = 'PROD005'), 'Variant E', 'SKU005E', 23.49),
((SELECT id FROM products WHERE code = 'PROD005'), 'Variant F', 'SKU005F', NULL);

-- Product 6: 2 variants
-- No variants for this product

-- Product 7: 5 variants
INSERT INTO product_variants (product_id, name, sku, price) VALUES
((SELECT id FROM products WHERE code = 'PROD007'), 'Variant A', 'SKU007A', NULL),
((SELECT id FROM products WHERE code = 'PROD007'), 'Variant B', 'SKU007B', NULL),
((SELECT id FROM products WHERE code = 'PROD007'), 'Variant C', 'SKU007C', NULL),
((SELECT id FROM products WHERE code = 'PROD007'), 'Variant D', 'SKU007D', NULL),
((SELECT id FROM products WHERE code = 'PROD007'), 'Variant E', 'SKU007E', 18.75);

-- Product 8: 1 variant
INSERT INTO product_variants (product_id, name, sku, price) VALUES
((SELECT id FROM products WHERE code = 'PROD008'), 'Variant A', 'SKU008A', 10.49);
