-- +goose Up
CREATE TABLE ingame_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
  title TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  banner_url TEXT NOT NULL DEFAULT '',
  website_url TEXT NOT NULL DEFAULT '',
  show_announcements INTEGER NOT NULL DEFAULT 1 CHECK (show_announcements IN (0, 1)),
  show_players INTEGER NOT NULL DEFAULT 1 CHECK (show_players IN (0, 1)),
  show_highlights INTEGER NOT NULL DEFAULT 1 CHECK (show_highlights IN (0, 1)),
  highlight_metric_1 TEXT NOT NULL DEFAULT 'active_play_seconds',
  highlight_metric_2 TEXT NOT NULL DEFAULT 'special_kills',
  highlight_metric_3 TEXT NOT NULL DEFAULT 'rescues',
  home_cache_seconds INTEGER NOT NULL DEFAULT 30
    CHECK (home_cache_seconds IN (10, 30, 60, 120)),
  player_cache_seconds INTEGER NOT NULL DEFAULT 60
    CHECK (player_cache_seconds IN (30, 60, 120, 300)),
  ranking_cache_seconds INTEGER NOT NULL DEFAULT 120
    CHECK (ranking_cache_seconds IN (60, 120, 300, 600)),
  content_cache_seconds INTEGER NOT NULL DEFAULT 300
    CHECK (content_cache_seconds IN (60, 300, 600, 1800)),
  updated_at INTEGER NOT NULL
);

INSERT INTO ingame_settings (id, updated_at) VALUES (1, unixepoch());

CREATE TABLE ingame_server_settings (
  server_id TEXT PRIMARY KEY REFERENCES game_servers(id) ON DELETE CASCADE,
  title_mode TEXT NOT NULL DEFAULT 'inherit'
    CHECK (title_mode IN ('inherit', 'override')),
  title TEXT NOT NULL DEFAULT '',
  description_mode TEXT NOT NULL DEFAULT 'inherit'
    CHECK (description_mode IN ('inherit', 'override', 'hidden')),
  description TEXT NOT NULL DEFAULT '',
  banner_mode TEXT NOT NULL DEFAULT 'inherit'
    CHECK (banner_mode IN ('inherit', 'override', 'hidden')),
  banner_url TEXT NOT NULL DEFAULT '',
  website_mode TEXT NOT NULL DEFAULT 'inherit'
    CHECK (website_mode IN ('inherit', 'override', 'hidden')),
  website_url TEXT NOT NULL DEFAULT '',
  highlight_mode TEXT NOT NULL DEFAULT 'inherit'
    CHECK (highlight_mode IN ('inherit', 'override')),
  highlight_metric_1 TEXT NOT NULL DEFAULT '',
  highlight_metric_2 TEXT NOT NULL DEFAULT '',
  highlight_metric_3 TEXT NOT NULL DEFAULT '',
  updated_at INTEGER NOT NULL
);

CREATE TABLE server_documents (
  server_id TEXT NOT NULL REFERENCES game_servers(id) ON DELETE CASCADE,
  key TEXT NOT NULL CHECK (key IN ('introduction', 'commands', 'resources')),
  mode TEXT NOT NULL DEFAULT 'inherit'
    CHECK (mode IN ('inherit', 'override', 'hidden')),
  content_markdown TEXT NOT NULL DEFAULT '',
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (server_id, key)
);

-- +goose Down
DROP TABLE IF EXISTS server_documents;
DROP TABLE IF EXISTS ingame_server_settings;
DROP TABLE IF EXISTS ingame_settings;
