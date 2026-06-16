-- +goose Up

CREATE TABLE IF NOT EXISTS product_categories (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid (),
    name text NOT NULL,
    description text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE products
ADD COLUMN IF NOT EXISTS category_id uuid REFERENCES product_categories (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_products_category_id ON products (category_id);

-- +goose Down

DROP INDEX IF EXISTS idx_products_category_id;

ALTER TABLE products DROP COLUMN IF EXISTS category_id;

DROP TABLE IF EXISTS product_categories;