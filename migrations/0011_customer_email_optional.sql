-- +goose Up

ALTER TABLE customers ALTER COLUMN email DROP NOT NULL;

-- +goose Down
. ALTER TABLE customers ALTER COLUMN email SET NOT NULL;