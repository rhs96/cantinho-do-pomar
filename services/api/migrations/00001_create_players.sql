-- +goose Up
CREATE TABLE players (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    display_name text        NOT NULL CHECK (char_length(display_name) BETWEEN 1 AND 32),
    coins        bigint      NOT NULL DEFAULT 0 CHECK (coins >= 0),
    xp           bigint      NOT NULL DEFAULT 0 CHECK (xp >= 0),
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE players;
