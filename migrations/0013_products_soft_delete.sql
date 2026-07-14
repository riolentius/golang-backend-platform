-- +goose Up
ALTER TABLE products
ADD COLUMN IF NOT EXISTS deleted_at timestamptz NULL;

ALTER TABLE customers
ADD COLUMN IF NOT EXISTS deleted_at timestamptz NULL;

ALTER TABLE customer_addresses
ADD COLUMN IF NOT EXISTS deleted_at timestamptz NULL;

ALTER TABLE customer_categories
ADD COLUMN IF NOT EXISTS deleted_at timestamptz NULL;

ALTER TABLE product_categories
ADD COLUMN IF NOT EXISTS deleted_at timestamptz NULL;

ALTER TABLE admins
ADD COLUMN IF NOT EXISTS deleted_at timestamptz NULL;

CREATE INDEX IF NOT EXISTS idx_products_not_deleted ON products (id)
WHERE
    deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_customers_not_deleted ON customers (id)
WHERE
    deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_customer_addresses_not_deleted ON customer_addresses (id)
WHERE
    deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_customer_categories_not_deleted ON customer_categories (id)
WHERE
    deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_product_categories_not_deleted ON product_categories (id)
WHERE
    deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_admins_not_deleted ON admins (id)
WHERE
    deleted_at IS NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_products_not_deleted;

DROP INDEX IF EXISTS idx_customers_not_deleted;

DROP INDEX IF EXISTS idx_customer_addresses_not_deleted;

DROP INDEX IF EXISTS idx_customer_categories_not_deleted;

DROP INDEX IF EXISTS idx_product_categories_not_deleted;

DROP INDEX IF EXISTS idx_admins_not_deleted;

ALTER TABLE products DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE customers DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE customer_addresses DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE customer_categories DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE product_categories DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE admins DROP COLUMN IF EXISTS deleted_at;