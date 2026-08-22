-- +goose Up
CREATE TABLE geoip_settings_v23 (
  singleton_id INTEGER PRIMARY KEY CHECK (singleton_id = 1),
  provider TEXT NOT NULL DEFAULT 'baidu' CHECK (provider = 'baidu'),
  api_key TEXT NOT NULL DEFAULT '',
  qps_limit INTEGER NOT NULL DEFAULT 2 CHECK (qps_limit BETWEEN 1 AND 3),
  cache_secret TEXT NOT NULL DEFAULT '',
  last_success_at INTEGER NOT NULL DEFAULT 0,
  last_error_at INTEGER NOT NULL DEFAULT 0,
  last_error_code TEXT NOT NULL DEFAULT '',
  ipv4_status TEXT NOT NULL DEFAULT 'unknown',
  ipv6_status TEXT NOT NULL DEFAULT 'unknown',
  updated_at INTEGER NOT NULL DEFAULT 0
);

INSERT INTO geoip_settings_v23 (
  singleton_id, provider, api_key, qps_limit, cache_secret,
  last_success_at, last_error_at, last_error_code,
  ipv4_status, ipv6_status, updated_at
)
SELECT singleton_id, provider, api_key, 2, cache_secret,
       last_success_at, last_error_at, last_error_code,
       ipv4_status, ipv6_status, updated_at
FROM geoip_settings;

DROP TABLE geoip_settings;
ALTER TABLE geoip_settings_v23 RENAME TO geoip_settings;

-- +goose Down
CREATE TABLE geoip_settings_v22 (
  singleton_id INTEGER PRIMARY KEY CHECK (singleton_id = 1),
  enabled INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0, 1)),
  provider TEXT NOT NULL DEFAULT 'baidu' CHECK (provider = 'baidu'),
  api_key TEXT NOT NULL DEFAULT '',
  cache_secret TEXT NOT NULL DEFAULT '',
  last_success_at INTEGER NOT NULL DEFAULT 0,
  last_error_at INTEGER NOT NULL DEFAULT 0,
  last_error_code TEXT NOT NULL DEFAULT '',
  ipv4_status TEXT NOT NULL DEFAULT 'unknown',
  ipv6_status TEXT NOT NULL DEFAULT 'unknown',
  updated_at INTEGER NOT NULL DEFAULT 0
);

INSERT INTO geoip_settings_v22 (
  singleton_id, enabled, provider, api_key, cache_secret,
  last_success_at, last_error_at, last_error_code,
  ipv4_status, ipv6_status, updated_at
)
SELECT singleton_id, CASE WHEN api_key <> '' THEN 1 ELSE 0 END,
       provider, api_key, cache_secret, last_success_at, last_error_at,
       last_error_code, ipv4_status, ipv6_status, updated_at
FROM geoip_settings;

DROP TABLE geoip_settings;
ALTER TABLE geoip_settings_v22 RENAME TO geoip_settings;
