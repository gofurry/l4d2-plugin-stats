-- +goose Up
CREATE TABLE dashboard_metadata (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE site_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  title TEXT NOT NULL,
  footer_text TEXT NOT NULL DEFAULT '',
  updated_at INTEGER NOT NULL
);

CREATE TABLE footer_links (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  label TEXT NOT NULL,
  url TEXT NOT NULL,
  sort_order INTEGER NOT NULL DEFAULT 0,
  open_new_tab INTEGER NOT NULL DEFAULT 0 CHECK (open_new_tab IN (0, 1)),
  enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE game_servers (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  server_key TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  connect_address TEXT NOT NULL,
  query_address TEXT NOT NULL,
  is_primary INTEGER NOT NULL DEFAULT 0 CHECK (is_primary IN (0, 1)),
  enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE UNIQUE INDEX game_servers_single_primary
  ON game_servers (is_primary)
  WHERE is_primary = 1;

CREATE INDEX game_servers_display_order
  ON game_servers (enabled, sort_order, id);

-- +goose Down
DROP INDEX IF EXISTS game_servers_display_order;
DROP INDEX IF EXISTS game_servers_single_primary;
DROP TABLE IF EXISTS game_servers;
DROP TABLE IF EXISTS footer_links;
DROP TABLE IF EXISTS site_settings;
DROP TABLE IF EXISTS dashboard_metadata;
