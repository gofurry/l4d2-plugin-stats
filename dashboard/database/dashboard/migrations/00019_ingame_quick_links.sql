-- +goose Up
CREATE TABLE ingame_quick_links (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  server_key TEXT NOT NULL
    CHECK (length(server_key) BETWEEN 1 AND 64)
    CHECK (server_key NOT GLOB '*[^A-Za-z0-9._-]*')
    CHECK (lower(server_key) <> 'change-me'),
  label TEXT NOT NULL CHECK (length(trim(label)) BETWEEN 1 AND 32),
  url TEXT NOT NULL
    CHECK (length(url) BETWEEN 1 AND 2048)
    CHECK (lower(url) GLOB 'http://*' OR lower(url) GLOB 'https://*')
    CHECK (instr(url, '@') = 0),
  sort_order INTEGER NOT NULL CHECK (sort_order BETWEEN 0 AND 7),
  enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
  UNIQUE (server_key, sort_order)
);

CREATE INDEX idx_ingame_quick_links_server_order
  ON ingame_quick_links(server_key, sort_order, id);

-- +goose Down
DROP TABLE IF EXISTS ingame_quick_links;
