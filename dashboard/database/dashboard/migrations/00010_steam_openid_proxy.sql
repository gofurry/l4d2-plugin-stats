-- +goose Up
ALTER TABLE site_settings
  ADD COLUMN steam_openid_proxy_port INTEGER NOT NULL DEFAULT 0
  CHECK (steam_openid_proxy_port BETWEEN 0 AND 65535);

-- +goose Down
ALTER TABLE site_settings DROP COLUMN steam_openid_proxy_port;
