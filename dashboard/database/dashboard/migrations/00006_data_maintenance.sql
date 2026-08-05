-- +goose Up
CREATE TABLE data_maintenance_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  aggregate_interval_minutes INTEGER NOT NULL DEFAULT 30
    CHECK (aggregate_interval_minutes IN (15, 30, 60, 180, 300, 720, 1440)),
  detail_retention_days INTEGER NOT NULL DEFAULT 180 CHECK (detail_retention_days BETWEEN 30 AND 3650),
  session_retention_days INTEGER NOT NULL DEFAULT 365 CHECK (session_retention_days BETWEEN 30 AND 3650),
  result_retention_days INTEGER NOT NULL DEFAULT 365 CHECK (result_retention_days BETWEEN 30 AND 3650),
  updated_at INTEGER NOT NULL
);

INSERT INTO data_maintenance_settings (
  id, aggregate_interval_minutes, detail_retention_days,
  session_retention_days, result_retention_days, updated_at
) VALUES (1, 30, 180, 365, 365, 0);

ALTER TABLE aggregate_state ADD COLUMN source_watermark INTEGER NOT NULL DEFAULT 0;
ALTER TABLE aggregate_state ADD COLUMN last_duration_ms INTEGER NOT NULL DEFAULT 0;
ALTER TABLE aggregate_state ADD COLUMN last_changed_days INTEGER NOT NULL DEFAULT 0;
ALTER TABLE aggregate_state ADD COLUMN last_build_mode TEXT NOT NULL DEFAULT '';

CREATE TABLE retention_runs (
  id TEXT PRIMARY KEY,
  executed_at INTEGER NOT NULL,
  source_watermark INTEGER NOT NULL,
  detail_cutoff INTEGER NOT NULL,
  session_cutoff INTEGER NOT NULL,
  result_cutoff INTEGER NOT NULL,
  equipment_rows INTEGER NOT NULL,
  versus_class_rows INTEGER NOT NULL,
  session_rows INTEGER NOT NULL,
  versus_round_result_rows INTEGER NOT NULL,
  versus_run_result_rows INTEGER NOT NULL
);

CREATE INDEX retention_runs_executed_at ON retention_runs (executed_at DESC, id DESC);

-- +goose Down
DROP INDEX IF EXISTS retention_runs_executed_at;
DROP TABLE IF EXISTS retention_runs;
ALTER TABLE aggregate_state DROP COLUMN last_build_mode;
ALTER TABLE aggregate_state DROP COLUMN last_changed_days;
ALTER TABLE aggregate_state DROP COLUMN last_duration_ms;
ALTER TABLE aggregate_state DROP COLUMN source_watermark;
DROP TABLE IF EXISTS data_maintenance_settings;
