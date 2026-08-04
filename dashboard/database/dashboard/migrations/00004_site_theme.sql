-- +goose Up
ALTER TABLE site_settings ADD COLUMN theme TEXT NOT NULL DEFAULT 'light'
  CHECK (theme IN ('light', 'dark'));

-- +goose Down
ALTER TABLE site_settings DROP COLUMN theme;
