-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Customer categories define pricing tiers/discount logic (you can expand later)
CREATE TABLE IF NOT EXISTS customer_categories (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid (),
    code text NOT NULL UNIQUE, -- e.g. REGULAR, VIP, WHOLESALE
    name text NOT NULL,
    description text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS customers (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid (),
    first_name text NOT NULL,
    last_name text,
    email text NOT NULL UNIQUE,
    phone text,
    identification_number text, -- optional: KTP/passport/etc
    category_id uuid REFERENCES customer_categories (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Addresses as a separate table (customers can have multiple addresses)
CREATE TABLE IF NOT EXISTS customer_addresses (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid (),
    customer_id uuid NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    label text, -- e.g. home, office
    address_line1 text NOT NULL,
    address_line2 text,
    city text,
    province text,
    postal_code text,
    country text NOT NULL DEFAULT 'ID',
    is_default boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_customer_addresses_customer_id ON customer_addresses (customer_id);

-- Product master
CREATE TABLE IF NOT EXISTS products (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid (),
    sku text UNIQUE,
    name text NOT NULL,
    description text,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Product prices separated (supports price history + category pricing)
CREATE TABLE IF NOT EXISTS product_prices (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid (),
    product_id uuid NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    category_id uuid REFERENCES customer_categories (id), -- NULL = default price for all categories
    currency text NOT NULL DEFAULT 'IDR',
    amount numeric(18, 2) NOT NULL CHECK (amount >= 0),
    valid_from timestamptz NOT NULL DEFAULT now(),
    valid_to timestamptz, -- NULL = current
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_product_prices_product_id ON product_prices (product_id);

CREATE INDEX IF NOT EXISTS idx_product_prices_category_id ON product_prices (category_id);

CREATE INDEX IF NOT EXISTS idx_product_prices_valid_range ON product_prices (valid_from, valid_to);

-- Transactions (simple version)
CREATE TABLE IF NOT EXISTS transactions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid (),
    customer_id uuid NOT NULL REFERENCES customers (id),
    status text NOT NULL DEFAULT 'pending', -- pending, paid, cancelled, refunded
    currency text NOT NULL DEFAULT 'IDR',
    total_amount numeric(18, 2) NOT NULL DEFAULT 0 CHECK (total_amount >= 0),
    notes text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_transactions_customer_id ON transactions (customer_id);

-- Transaction items (recommended, otherwise transaction can't store products)
CREATE TABLE IF NOT EXISTS transaction_items (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid (),
    transaction_id uuid NOT NULL REFERENCES transactions (id) ON DELETE CASCADE,
    product_id uuid NOT NULL REFERENCES products (id),
    qty integer NOT NULL CHECK (qty > 0),
    unit_amount numeric(18, 2) NOT NULL CHECK (unit_amount >= 0),
    line_total numeric(18, 2) NOT NULL CHECK (line_total >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_transaction_items_transaction_id ON transaction_items (transaction_id);

-- +goose Down
DROP TABLE IF EXISTS transaction_items;

DROP TABLE IF EXISTS transactions;

DROP TABLE IF EXISTS product_prices;

DROP TABLE IF EXISTS products;

DROP TABLE IF EXISTS customer_addresses;

DROP TABLE IF EXISTS customers;

DROP TABLE IF EXISTS customer_categories;

SQL