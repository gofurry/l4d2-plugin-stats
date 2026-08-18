-- +goose Up
ALTER TABLE ingame_server_settings ADD COLUMN short_description TEXT NOT NULL DEFAULT ''
  CHECK (length(short_description) <= 80);

CREATE TABLE ingame_map_names (
  map_name TEXT PRIMARY KEY
    CHECK (length(trim(map_name)) BETWEEN 1 AND 128),
  display_name TEXT NOT NULL
    CHECK (length(trim(display_name)) BETWEEN 1 AND 80),
  updated_at INTEGER NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS ingame_map_names;
ALTER TABLE ingame_server_settings DROP COLUMN short_description;
