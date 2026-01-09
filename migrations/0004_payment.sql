-- +goose Up


CREATE TABLE IF NOT EXISTS payments (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  transaction_id uuid NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,

  method text NOT NULL CHECK (method IN ('cash', 'transfer')),
  amount numeric(18,2) NOT NULL CHECK (amount > 0),
  currency text NOT NULL DEFAULT 'IDR',

  paid_at timestamptz NOT NULL DEFAULT now(),

-- transfer metadata (optional)
sender_name text,
  reference text,

  note text,

  status text NOT NULL DEFAULT 'posted' CHECK (status IN ('posted', 'voided')),

  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_payments_transaction_id ON payments (transaction_id);

CREATE INDEX IF NOT EXISTS idx_payments_paid_at ON payments (paid_at);

ALTER TABLE transactions
ADD COLUMN IF NOT EXISTS paid_amount numeric(18, 2) NOT NULL DEFAULT 0 CHECK (paid_amount >= 0),
ADD COLUMN IF NOT EXISTS payment_status text NOT NULL DEFAULT 'unpaid' CHECK (
    payment_status IN (
        'unpaid',
        'partial',
        'paid',
        'overpaid'
    )
);

-- +goose Down

ALTER TABLE transactions
DROP COLUMN IF EXISTS payment_status,
DROP COLUMN IF EXISTS paid_amount;

DROP TABLE IF EXISTS payments;