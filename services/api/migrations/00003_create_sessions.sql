-- +goose Up
CREATE TABLE sessions (
    token_hash bytea       PRIMARY KEY,
    player_id  uuid        NOT NULL REFERENCES players (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

CREATE INDEX sessions_player_id_idx ON sessions (player_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

-- +goose Down
DROP TABLE sessions;
