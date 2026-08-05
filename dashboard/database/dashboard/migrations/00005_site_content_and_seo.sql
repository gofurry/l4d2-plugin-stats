-- +goose Up
CREATE TABLE site_seo_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  enabled INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0, 1)),
  description TEXT NOT NULL DEFAULT '',
  image_url TEXT NOT NULL DEFAULT '',
  updated_at INTEGER NOT NULL
);

INSERT INTO site_seo_settings (id, enabled, description, image_url, updated_at)
VALUES (1, 0, '', '', 0);

CREATE TABLE site_documents (
  key TEXT PRIMARY KEY CHECK (key IN ('introduction', 'commands', 'resources')),
  enabled INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0, 1)),
  content_markdown TEXT NOT NULL DEFAULT '',
  updated_at INTEGER NOT NULL
);

INSERT INTO site_documents (key, enabled, content_markdown, updated_at) VALUES
  ('introduction', 0, '', 0),
  ('commands', 0, '', 0),
  ('resources', 0, '', 0);

-- +goose Down
DROP TABLE IF EXISTS site_documents;
DROP TABLE IF EXISTS site_seo_settings;
