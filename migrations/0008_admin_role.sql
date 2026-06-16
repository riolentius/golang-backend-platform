-- +goose Up
ALTER TABLE admins
ADD COLUMN IF NOT EXISTS role text NOT NULL DEFAULT 'admin' CHECK (
    role IN ('admin', 'superadmin')
);

-- +goose Down
ALTER TABLE admins DROP COLUMN IF EXISTS role;