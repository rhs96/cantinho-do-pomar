-- +goose Up
-- updated_at is really the instant growth was last accumulated, and it has no
-- meaning for plots without a crop.
ALTER TABLE plots RENAME COLUMN updated_at TO growth_at;
ALTER TABLE plots ALTER COLUMN growth_at DROP NOT NULL;
ALTER TABLE plots ALTER COLUMN growth_at DROP DEFAULT;
UPDATE plots SET growth_at = NULL WHERE state IN ('grass', 'plowed');

-- +goose Down
UPDATE plots SET growth_at = now() WHERE growth_at IS NULL;
ALTER TABLE plots ALTER COLUMN growth_at SET DEFAULT now();
ALTER TABLE plots ALTER COLUMN growth_at SET NOT NULL;
ALTER TABLE plots RENAME COLUMN growth_at TO updated_at;
