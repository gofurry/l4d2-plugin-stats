-- +goose Up
CREATE TABLE aggregate_monthly_rows (
  kind TEXT NOT NULL,
  month INTEGER NOT NULL,
  server_key TEXT NOT NULL DEFAULT '',
  steam_id TEXT NOT NULL DEFAULT '',
  mode TEXT NOT NULL DEFAULT '',
  dimension TEXT NOT NULL DEFAULT '',
  metrics_json TEXT NOT NULL,
  PRIMARY KEY (kind, month, server_key, steam_id, mode, dimension)
);

CREATE INDEX aggregate_monthly_rows_player
  ON aggregate_monthly_rows (steam_id, month, kind);

CREATE INDEX aggregate_monthly_rows_query
  ON aggregate_monthly_rows (kind, server_key, month, mode, steam_id);

CREATE TABLE aggregate_lifetime_rows (
  kind TEXT NOT NULL,
  server_key TEXT NOT NULL DEFAULT '',
  steam_id TEXT NOT NULL DEFAULT '',
  mode TEXT NOT NULL DEFAULT '',
  dimension TEXT NOT NULL DEFAULT '',
  metrics_json TEXT NOT NULL,
  PRIMARY KEY (kind, server_key, steam_id, mode, dimension)
);

CREATE INDEX aggregate_lifetime_rows_player
  ON aggregate_lifetime_rows (steam_id, kind);

CREATE INDEX aggregate_lifetime_rows_query
  ON aggregate_lifetime_rows (kind, server_key, mode, steam_id);

-- +goose Down
DROP INDEX IF EXISTS aggregate_lifetime_rows_query;
DROP INDEX IF EXISTS aggregate_lifetime_rows_player;
DROP TABLE IF EXISTS aggregate_lifetime_rows;
DROP INDEX IF EXISTS aggregate_monthly_rows_query;
DROP INDEX IF EXISTS aggregate_monthly_rows_player;
DROP TABLE IF EXISTS aggregate_monthly_rows;
