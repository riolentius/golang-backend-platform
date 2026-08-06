-- +goose Up

ALTER TABLE transaction_items
ADD COLUMN IF NOT EXISTS discount_amount numeric(18, 2) NOT NULL DEFAULT 0 CHECK (discount_amount >= 0);

-- +goose Down

ALTER TABLE transaction_items DROP COLUMN IF EXISTS discount_amount;