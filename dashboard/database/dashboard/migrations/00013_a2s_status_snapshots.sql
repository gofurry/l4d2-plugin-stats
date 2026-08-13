-- +goose Up
CREATE TABLE a2s_status_snapshots (
  server_id TEXT PRIMARY KEY REFERENCES game_servers(id) ON DELETE CASCADE,
  status_json TEXT NOT NULL,
  checked_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS a2s_status_snapshots;
