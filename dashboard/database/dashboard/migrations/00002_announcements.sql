-- +goose Up
CREATE TABLE announcements (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  content_markdown TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX announcements_display_order
  ON announcements (created_at DESC, id DESC);

-- +goose Down
DROP INDEX IF EXISTS announcements_display_order;
DROP TABLE IF EXISTS announcements;
