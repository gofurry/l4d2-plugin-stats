-- +goose Up
ALTER TABLE site_settings ADD COLUMN browser_title TEXT NOT NULL DEFAULT 'L4D2 Stats';
ALTER TABLE site_settings ADD COLUMN a2s_refresh_seconds INTEGER NOT NULL DEFAULT 30
  CHECK (a2s_refresh_seconds IN (5, 10, 15, 30, 45, 60));
ALTER TABLE site_settings ADD COLUMN a2s_jitter_seconds INTEGER NOT NULL DEFAULT 2
  CHECK (a2s_jitter_seconds IN (2, 5));
ALTER TABLE site_settings ADD COLUMN a2s_retry_count INTEGER NOT NULL DEFAULT 1
  CHECK (a2s_retry_count IN (1, 2, 3));

-- +goose Down
ALTER TABLE site_settings DROP COLUMN a2s_retry_count;
ALTER TABLE site_settings DROP COLUMN a2s_jitter_seconds;
ALTER TABLE site_settings DROP COLUMN a2s_refresh_seconds;
ALTER TABLE site_settings DROP COLUMN browser_title;
