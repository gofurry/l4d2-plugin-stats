-- +goose Up
CREATE TABLE dashboard_metadata (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE site_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  language TEXT NOT NULL DEFAULT 'zh-CN'
    CHECK (language IN ('zh-CN', 'en')),
  footer_enabled INTEGER NOT NULL DEFAULT 0
    CHECK (footer_enabled IN (0, 1)),
  background_image_url TEXT NOT NULL DEFAULT '',
  public_origin TEXT NOT NULL DEFAULT '',
  steam_openid_enabled INTEGER NOT NULL DEFAULT 0
    CHECK (steam_openid_enabled IN (0, 1)),
  updated_at INTEGER NOT NULL
);

CREATE TABLE footer_links (
  id TEXT PRIMARY KEY,
  label TEXT NOT NULL,
  url TEXT NOT NULL,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE game_servers (
  id TEXT PRIMARY KEY,
  display_name TEXT NOT NULL,
  address TEXT NOT NULL UNIQUE,
  enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX game_servers_display_order
  ON game_servers (enabled, sort_order, id);

CREATE TABLE admin_account (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  username TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  jwt_secret TEXT NOT NULL,
  token_version INTEGER NOT NULL DEFAULT 1,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  password_changed_at INTEGER NOT NULL
);

CREATE TABLE aggregate_rows (
  kind TEXT NOT NULL,
  day INTEGER NOT NULL,
  server_key TEXT NOT NULL,
  steam_id TEXT NOT NULL,
  mode TEXT NOT NULL,
  dimension TEXT NOT NULL DEFAULT '',
  metrics_json TEXT NOT NULL,
  PRIMARY KEY (kind, day, server_key, steam_id, mode, dimension)
);

CREATE INDEX aggregate_rows_player_day
  ON aggregate_rows (steam_id, day, kind);

CREATE INDEX aggregate_rows_ranking
  ON aggregate_rows (kind, mode, day, server_key);

CREATE TABLE aggregate_state (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  state TEXT NOT NULL,
  last_started_at INTEGER NOT NULL DEFAULT 0,
  last_finished_at INTEGER NOT NULL DEFAULT 0,
  source_rows INTEGER NOT NULL DEFAULT 0,
  aggregate_rows INTEGER NOT NULL DEFAULT 0,
  last_error TEXT NOT NULL DEFAULT ''
);

INSERT INTO aggregate_state (id, state) VALUES (1, 'empty');

-- +goose Down
DROP TABLE IF EXISTS aggregate_state;
DROP INDEX IF EXISTS aggregate_rows_ranking;
DROP INDEX IF EXISTS aggregate_rows_player_day;
DROP TABLE IF EXISTS aggregate_rows;
DROP TABLE IF EXISTS admin_account;
DROP INDEX IF EXISTS game_servers_display_order;
DROP TABLE IF EXISTS game_servers;
DROP TABLE IF EXISTS footer_links;
DROP TABLE IF EXISTS site_settings;
DROP TABLE IF EXISTS dashboard_metadata;
