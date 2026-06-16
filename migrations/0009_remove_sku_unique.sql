-- +goose Up
ALTER TABLE products DROP CONSTRAINT IF EXISTS products_sku_key;
 
-- +goose Down
ALTER TABLE products ADD CONSTRAINT products_sku_key UNIQUE (sku);
