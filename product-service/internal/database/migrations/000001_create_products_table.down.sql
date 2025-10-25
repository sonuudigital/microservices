DROP INDEX IF EXISTS idx_products_category_id;
DROP INDEX IF EXISTS idx_products_code;
DROP INDEX IF EXISTS idx_products_created_at;
DROP TABLE IF EXISTS product_categories;
DROP TABLE IF EXISTS products;
DROP EXTENSION IF EXISTS "uuid-ossp";
