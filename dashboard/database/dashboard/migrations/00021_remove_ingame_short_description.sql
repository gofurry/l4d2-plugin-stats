-- +goose Up
ALTER TABLE ingame_server_settings DROP COLUMN short_description;

-- +goose Down
ALTER TABLE ingame_server_settings ADD COLUMN short_description TEXT NOT NULL DEFAULT ''
  CHECK (length(short_description) <= 80);
