-- +goose Up
CREATE TABLE player_badge_showcase_state (
  steam_id TEXT PRIMARY KEY,
  configured_at INTEGER NOT NULL
);

INSERT INTO player_badge_showcase_state (steam_id, configured_at)
SELECT steam_id, MAX(updated_at)
FROM player_badge_showcase
GROUP BY steam_id;

-- +goose Down
DROP TABLE IF EXISTS player_badge_showcase_state;
