-- +goose Up

ALTER TABLE products
ADD COLUMN IF NOT EXISTS stock_on_hand integer NOT NULL DEFAULT 0 CHECK (stock_on_hand >= 0),
ADD COLUMN IF NOT EXISTS stock_reserved integer NOT NULL DEFAULT 0 CHECK (stock_reserved >= 0);

ALTER TABLE products
DROP CONSTRAINT IF EXISTS chk_products_stock_reserved_le_on_hand;

ALTER TABLE products
ADD CONSTRAINT chk_products_stock_reserved_le_on_hand CHECK (
    stock_reserved <= stock_on_hand
);

-- +goose Down

ALTER TABLE products
DROP CONSTRAINT IF EXISTS chk_products_stock_reserved_le_on_hand;

ALTER TABLE products DROP COLUMN IF EXISTS stock_reserved;

ALTER TABLE products DROP COLUMN IF EXISTS stock_on_hand;