-- +goose Up
CREATE TABLE farms (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id  uuid        NOT NULL UNIQUE REFERENCES players (id) ON DELETE CASCADE,
    width      smallint    NOT NULL CHECK (width BETWEEN 1 AND 24),
    height     smallint    NOT NULL CHECK (height BETWEEN 1 AND 24),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TYPE plot_state AS ENUM ('grass', 'plowed', 'planted', 'withered');

CREATE TABLE plots (
    farm_id    uuid        NOT NULL REFERENCES farms (id) ON DELETE CASCADE,
    col        smallint    NOT NULL CHECK (col >= 0),
    "row"      smallint    NOT NULL CHECK ("row" >= 0),
    state      plot_state  NOT NULL DEFAULT 'grass',
    crop_id    text,
    growth_ms  bigint      NOT NULL DEFAULT 0 CHECK (growth_ms >= 0),
    updated_at timestamptz NOT NULL DEFAULT now(),
    wet_until  timestamptz,

    PRIMARY KEY (farm_id, col, "row"),
    CONSTRAINT plots_crop_matches_state
        CHECK ((state IN ('planted', 'withered')) = (crop_id IS NOT NULL))
);

-- +goose Down
DROP TABLE plots;
DROP TYPE plot_state;
DROP TABLE farms;
