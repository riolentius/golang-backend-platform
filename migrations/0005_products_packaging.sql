-- +goose Up

ALTER TABLE products
ADD COLUMN IF NOT EXISTS base_product_id uuid NULL,
ADD COLUMN IF NOT EXISTS pack_size numeric(18, 6) NOT NULL DEFAULT 1;

ALTER TABLE products
ADD CONSTRAINT fk_products_base_product FOREIGN KEY (base_product_id) REFERENCES products (id) ON DELETE SET NULL;

ALTER TABLE products
ADD CONSTRAINT chk_products_pack_size_positive CHECK (pack_size > 0);

CREATE INDEX IF NOT EXISTS idx_products_base_product_id ON products (base_product_id);

-- +goose Down

DROP INDEX IF EXISTS idx_products_base_product_id;

ALTER TABLE products
DROP CONSTRAINT IF EXISTS chk_products_pack_size_positive;

ALTER TABLE products
DROP CONSTRAINT IF EXISTS fk_products_base_product;

ALTER TABLE products
DROP COLUMN IF EXISTS pack_size,
DROP COLUMN IF EXISTS base_product_id;