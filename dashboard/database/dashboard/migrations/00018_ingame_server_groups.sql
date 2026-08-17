-- +goose Up
ALTER TABLE ingame_settings ADD COLUMN show_server_intro INTEGER NOT NULL DEFAULT 1
  CHECK (show_server_intro IN (0, 1));
ALTER TABLE ingame_settings ADD COLUMN show_server_status INTEGER NOT NULL DEFAULT 1
  CHECK (show_server_status IN (0, 1));

CREATE TABLE ingame_server_settings_v18 (
  server_key TEXT PRIMARY KEY
    CHECK (length(server_key) BETWEEN 1 AND 64)
    CHECK (server_key NOT GLOB '*[^A-Za-z0-9._-]*')
    CHECK (lower(server_key) <> 'change-me'),
  title_mode TEXT NOT NULL DEFAULT 'inherit'
    CHECK (title_mode IN ('inherit', 'override')),
  title TEXT NOT NULL DEFAULT '',
  description_mode TEXT NOT NULL DEFAULT 'inherit'
    CHECK (description_mode IN ('inherit', 'override', 'hidden')),
  description TEXT NOT NULL DEFAULT '',
  banner_mode TEXT NOT NULL DEFAULT 'inherit'
    CHECK (banner_mode IN ('inherit', 'override', 'hidden')),
  banner_url TEXT NOT NULL DEFAULT '',
  background_mode TEXT NOT NULL DEFAULT 'inherit'
    CHECK (background_mode IN ('inherit', 'override', 'hidden')),
  background_url TEXT NOT NULL DEFAULT '',
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

WITH mapped AS (
  SELECT legacy.*,
         trim(CASE WHEN json_valid(snapshot.status_json) THEN json_extract(snapshot.status_json, '$.server_key') ELSE '' END) AS mapped_server_key,
         row_number() OVER (
           PARTITION BY trim(CASE WHEN json_valid(snapshot.status_json) THEN json_extract(snapshot.status_json, '$.server_key') ELSE '' END)
           ORDER BY legacy.updated_at DESC, legacy.server_id DESC
         ) AS winner
  FROM ingame_server_settings AS legacy
  JOIN a2s_status_snapshots AS snapshot ON snapshot.server_id = legacy.server_id
)
INSERT INTO ingame_server_settings_v18 (
  server_key, title_mode, title, description_mode, description,
  banner_mode, banner_url, background_mode, background_url,
  website_mode, website_url, highlight_mode,
  highlight_metric_1, highlight_metric_2, highlight_metric_3, updated_at
)
SELECT mapped_server_key, title_mode, title, description_mode, description,
       banner_mode, banner_url, background_mode, background_url,
       website_mode, website_url, highlight_mode,
       highlight_metric_1, highlight_metric_2, highlight_metric_3, updated_at
FROM mapped
WHERE winner = 1
  AND length(mapped_server_key) BETWEEN 1 AND 64
  AND mapped_server_key NOT GLOB '*[^A-Za-z0-9._-]*'
  AND lower(mapped_server_key) <> 'change-me';

CREATE TABLE server_documents_v18 (
  server_key TEXT NOT NULL
    CHECK (length(server_key) BETWEEN 1 AND 64)
    CHECK (server_key NOT GLOB '*[^A-Za-z0-9._-]*')
    CHECK (lower(server_key) <> 'change-me'),
  key TEXT NOT NULL CHECK (key IN ('introduction', 'commands', 'resources')),
  mode TEXT NOT NULL DEFAULT 'inherit'
    CHECK (mode IN ('inherit', 'override', 'hidden')),
  content_markdown TEXT NOT NULL DEFAULT '',
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (server_key, key)
);

WITH mapped AS (
  SELECT legacy.*,
         trim(CASE WHEN json_valid(snapshot.status_json) THEN json_extract(snapshot.status_json, '$.server_key') ELSE '' END) AS mapped_server_key,
         row_number() OVER (
           PARTITION BY trim(CASE WHEN json_valid(snapshot.status_json) THEN json_extract(snapshot.status_json, '$.server_key') ELSE '' END), legacy.key
           ORDER BY legacy.updated_at DESC, legacy.server_id DESC
         ) AS winner
  FROM server_documents AS legacy
  JOIN a2s_status_snapshots AS snapshot ON snapshot.server_id = legacy.server_id
)
INSERT INTO server_documents_v18 (server_key, key, mode, content_markdown, updated_at)
SELECT mapped_server_key, key, mode, content_markdown, updated_at
FROM mapped
WHERE winner = 1
  AND length(mapped_server_key) BETWEEN 1 AND 64
  AND mapped_server_key NOT GLOB '*[^A-Za-z0-9._-]*'
  AND lower(mapped_server_key) <> 'change-me';

DROP TABLE server_documents;
DROP TABLE ingame_server_settings;
ALTER TABLE ingame_server_settings_v18 RENAME TO ingame_server_settings;
ALTER TABLE server_documents_v18 RENAME TO server_documents;

-- +goose Down
CREATE TABLE ingame_server_settings_v17 (
  server_id TEXT PRIMARY KEY REFERENCES game_servers(id) ON DELETE CASCADE,
  title_mode TEXT NOT NULL DEFAULT 'inherit' CHECK (title_mode IN ('inherit', 'override')),
  title TEXT NOT NULL DEFAULT '',
  description_mode TEXT NOT NULL DEFAULT 'inherit' CHECK (description_mode IN ('inherit', 'override', 'hidden')),
  description TEXT NOT NULL DEFAULT '',
  banner_mode TEXT NOT NULL DEFAULT 'inherit' CHECK (banner_mode IN ('inherit', 'override', 'hidden')),
  banner_url TEXT NOT NULL DEFAULT '',
  background_mode TEXT NOT NULL DEFAULT 'inherit' CHECK (background_mode IN ('inherit', 'override', 'hidden')),
  background_url TEXT NOT NULL DEFAULT '',
  website_mode TEXT NOT NULL DEFAULT 'inherit' CHECK (website_mode IN ('inherit', 'override', 'hidden')),
  website_url TEXT NOT NULL DEFAULT '',
  highlight_mode TEXT NOT NULL DEFAULT 'inherit' CHECK (highlight_mode IN ('inherit', 'override')),
  highlight_metric_1 TEXT NOT NULL DEFAULT '',
  highlight_metric_2 TEXT NOT NULL DEFAULT '',
  highlight_metric_3 TEXT NOT NULL DEFAULT '',
  updated_at INTEGER NOT NULL
);

INSERT INTO ingame_server_settings_v17 (
  server_id, title_mode, title, description_mode, description,
  banner_mode, banner_url, background_mode, background_url,
  website_mode, website_url, highlight_mode,
  highlight_metric_1, highlight_metric_2, highlight_metric_3, updated_at
)
SELECT snapshot.server_id, current.title_mode, current.title,
       current.description_mode, current.description,
       current.banner_mode, current.banner_url,
       current.background_mode, current.background_url,
       current.website_mode, current.website_url,
       current.highlight_mode, current.highlight_metric_1,
       current.highlight_metric_2, current.highlight_metric_3, current.updated_at
FROM a2s_status_snapshots AS snapshot
JOIN ingame_server_settings AS current
  ON trim(CASE WHEN json_valid(snapshot.status_json) THEN json_extract(snapshot.status_json, '$.server_key') ELSE '' END) = current.server_key;

CREATE TABLE server_documents_v17 (
  server_id TEXT NOT NULL REFERENCES game_servers(id) ON DELETE CASCADE,
  key TEXT NOT NULL CHECK (key IN ('introduction', 'commands', 'resources')),
  mode TEXT NOT NULL DEFAULT 'inherit' CHECK (mode IN ('inherit', 'override', 'hidden')),
  content_markdown TEXT NOT NULL DEFAULT '',
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (server_id, key)
);

INSERT INTO server_documents_v17 (server_id, key, mode, content_markdown, updated_at)
SELECT snapshot.server_id, current.key, current.mode, current.content_markdown, current.updated_at
FROM a2s_status_snapshots AS snapshot
JOIN server_documents AS current
  ON trim(CASE WHEN json_valid(snapshot.status_json) THEN json_extract(snapshot.status_json, '$.server_key') ELSE '' END) = current.server_key;

DROP TABLE server_documents;
DROP TABLE ingame_server_settings;
ALTER TABLE ingame_server_settings_v17 RENAME TO ingame_server_settings;
ALTER TABLE server_documents_v17 RENAME TO server_documents;

ALTER TABLE ingame_settings RENAME TO ingame_settings_v18;
CREATE TABLE ingame_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
  title TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  banner_url TEXT NOT NULL DEFAULT '',
  background_url TEXT NOT NULL DEFAULT '',
  website_url TEXT NOT NULL DEFAULT '',
  show_announcements INTEGER NOT NULL DEFAULT 1 CHECK (show_announcements IN (0, 1)),
  show_players INTEGER NOT NULL DEFAULT 1 CHECK (show_players IN (0, 1)),
  show_highlights INTEGER NOT NULL DEFAULT 1 CHECK (show_highlights IN (0, 1)),
  highlight_metric_1 TEXT NOT NULL DEFAULT 'active_play_seconds',
  highlight_metric_2 TEXT NOT NULL DEFAULT 'special_kills',
  highlight_metric_3 TEXT NOT NULL DEFAULT 'rescues',
  home_cache_seconds INTEGER NOT NULL DEFAULT 30 CHECK (home_cache_seconds IN (10, 30, 60, 120)),
  player_cache_seconds INTEGER NOT NULL DEFAULT 60 CHECK (player_cache_seconds IN (30, 60, 120, 300)),
  ranking_cache_seconds INTEGER NOT NULL DEFAULT 120 CHECK (ranking_cache_seconds IN (60, 120, 300, 600)),
  content_cache_seconds INTEGER NOT NULL DEFAULT 300 CHECK (content_cache_seconds IN (60, 300, 600, 1800)),
  updated_at INTEGER NOT NULL
);
INSERT INTO ingame_settings (
  id, enabled, title, description, banner_url, background_url, website_url,
  show_announcements, show_players, show_highlights,
  highlight_metric_1, highlight_metric_2, highlight_metric_3,
  home_cache_seconds, player_cache_seconds, ranking_cache_seconds,
  content_cache_seconds, updated_at
)
SELECT id, enabled, title, description, banner_url, background_url, website_url,
       show_announcements, show_players, show_highlights,
       highlight_metric_1, highlight_metric_2, highlight_metric_3,
       home_cache_seconds, player_cache_seconds, ranking_cache_seconds,
       content_cache_seconds, updated_at
FROM ingame_settings_v18;
DROP TABLE ingame_settings_v18;
