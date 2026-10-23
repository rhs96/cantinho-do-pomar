-- +goose Up
CREATE TYPE item_kind AS ENUM ('seed', 'produce');

CREATE TABLE inventory (
    player_id uuid      NOT NULL REFERENCES players (id) ON DELETE CASCADE,
    kind      item_kind NOT NULL,
    item_id   text      NOT NULL,
    quantity  integer   NOT NULL CHECK (quantity >= 0),

    PRIMARY KEY (player_id, kind, item_id)
);

-- +goose Down
DROP TABLE inventory;
DROP TYPE item_kind;
