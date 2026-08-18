-- +goose Up
CREATE TABLE geoip_settings (
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

INSERT INTO geoip_settings (singleton_id) VALUES (1);

CREATE TABLE geoip_cache (
  ip_hash TEXT NOT NULL,
  provider TEXT NOT NULL CHECK (provider = 'baidu'),
  country TEXT NOT NULL DEFAULT '',
  country_code TEXT NOT NULL DEFAULT '',
  province TEXT NOT NULL DEFAULT '',
  city TEXT NOT NULL DEFAULT '',
  district TEXT NOT NULL DEFAULT '',
  adcode TEXT NOT NULL DEFAULT '',
  longitude REAL NULL,
  latitude REAL NULL,
  coordinate_system TEXT NOT NULL DEFAULT 'bd09ll',
  precision TEXT NOT NULL DEFAULT 'city',
  status TEXT NOT NULL,
  error_code TEXT NOT NULL DEFAULT '',
  resolved_at INTEGER NOT NULL,
  expires_at INTEGER NOT NULL,
  PRIMARY KEY (ip_hash, provider)
);

CREATE INDEX geoip_cache_expires_idx ON geoip_cache (expires_at, ip_hash);

CREATE TABLE chat_audit_settings (
  singleton_id INTEGER PRIMARY KEY CHECK (singleton_id = 1),
  enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
  retention_days INTEGER NOT NULL DEFAULT 30 CHECK (retention_days IN (0, 7, 14, 30, 60, 90, 180, 365)),
  last_cleanup_at INTEGER NOT NULL DEFAULT 0,
  updated_at INTEGER NOT NULL DEFAULT 0
);

INSERT INTO chat_audit_settings (singleton_id) VALUES (1);

CREATE TABLE chat_export_audit (
  export_id TEXT PRIMARY KEY,
  exported_at INTEGER NOT NULL,
  admin_identity TEXT NOT NULL,
  output_format TEXT NOT NULL CHECK (output_format IN ('csv', 'jsonl')),
  filter_summary TEXT NOT NULL,
  row_count INTEGER NULL,
  completed INTEGER NOT NULL DEFAULT 0 CHECK (completed IN (0, 1))
);

CREATE INDEX chat_export_audit_time_idx ON chat_export_audit (exported_at DESC, export_id);

-- +goose Down
DROP TABLE chat_export_audit;
DROP TABLE chat_audit_settings;
DROP TABLE geoip_cache;
DROP TABLE geoip_settings;
