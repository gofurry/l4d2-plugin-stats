-- +goose Up
CREATE TABLE player_profile_visibility (
  steam_id TEXT PRIMARY KEY,
  visible_sections_json TEXT NOT NULL,
  updated_at INTEGER NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS player_profile_visibility;
