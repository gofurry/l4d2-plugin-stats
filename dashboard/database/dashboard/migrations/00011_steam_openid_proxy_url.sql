-- +goose Up
ALTER TABLE site_settings
  ADD COLUMN steam_openid_proxy_url TEXT NOT NULL DEFAULT '';

UPDATE site_settings
SET steam_openid_proxy_url = 'http://127.0.0.1:' || CAST(steam_openid_proxy_port AS TEXT)
WHERE steam_openid_proxy_port BETWEEN 1 AND 65535;

-- +goose Down
ALTER TABLE site_settings DROP COLUMN steam_openid_proxy_url;
