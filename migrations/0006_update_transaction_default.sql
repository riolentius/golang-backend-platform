-- +goose Up
ALTER TABLE transactions ALTER COLUMN status SET DEFAULT 'draft';

-- +goose Down
ALTER TABLE transactions ALTER COLUMN status SET DEFAULT 'pending';