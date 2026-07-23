-- +goose Up

CREATE TABLE IF NOT EXISTS returns (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid (),
    transaction_id uuid NOT NULL REFERENCES transactions (id) ON DELETE CASCADE,
    total_amount numeric(18, 2) NOT NULL DEFAULT 0 CHECK (total_amount >= 0),
    currency text NOT NULL DEFAULT 'IDR',
    note text,
    created_by_email text,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_returns_transaction_id ON returns (
    transaction_id,
    created_at DESC
);

CREATE TABLE IF NOT EXISTS return_items (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid (),
    return_id uuid NOT NULL REFERENCES returns (id) ON DELETE CASCADE,
    transaction_item_id uuid NOT NULL REFERENCES transaction_items (id) ON DELETE CASCADE,
    product_id uuid NOT NULL REFERENCES products (id),
    qty integer NOT NULL CHECK (qty > 0),
    unit_amount numeric(18, 2) NOT NULL CHECK (unit_amount >= 0),
    line_total numeric(18, 2) NOT NULL CHECK (line_total >= 0),
    restock boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_return_items_return_id ON return_items (return_id);

CREATE INDEX IF NOT EXISTS idx_return_items_transaction_item_id ON return_items (transaction_item_id);

ALTER TABLE stock_movements
DROP CONSTRAINT IF EXISTS stock_movements_source_check;

ALTER TABLE stock_movements
ADD CONSTRAINT stock_movements_source_check CHECK (
    source IN (
        'manual',
        'transaction',
        'initial',
        'adjustment',
        'return'
    )
);

-- +goose Down

ALTER TABLE stock_movements
DROP CONSTRAINT IF EXISTS stock_movements_source_check;

ALTER TABLE stock_movements
ADD CONSTRAINT stock_movements_source_check CHECK (
    source IN (
        'manual',
        'transaction',
        'initial',
        'adjustment'
    )
);

DROP TABLE IF EXISTS return_items;

DROP TABLE IF EXISTS returns;