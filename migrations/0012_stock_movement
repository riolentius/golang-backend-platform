-- +goose Up
CREATE TABLE IF NOT EXISTS stock_movements (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid (),
    product_id uuid NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    direction text NOT NULL CHECK (direction IN ('in', 'out')),
    quantity integer NOT NULL CHECK (quantity > 0),
    source text NOT NULL CHECK (
        source IN (
            'manual',
            'transaction',
            'initial',
            'adjustment'
        )
    ),
    reference_id uuid NULL,
    note text NULL,
    created_by_email text NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_stock_movements_product_id ON stock_movements (product_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS stock_movements;