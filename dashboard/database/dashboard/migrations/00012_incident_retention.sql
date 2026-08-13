-- +goose Up
ALTER TABLE data_maintenance_settings ADD COLUMN incident_retention_days INTEGER NOT NULL DEFAULT 180
  CHECK (incident_retention_days BETWEEN 30 AND 3650);

CREATE TABLE incident_retention_runs (
  id TEXT PRIMARY KEY,
  executed_at INTEGER NOT NULL,
  incident_version INTEGER NOT NULL,
  cutoff INTEGER NOT NULL,
  incident_rows INTEGER NOT NULL
);

CREATE INDEX incident_retention_runs_executed_at
ON incident_retention_runs (executed_at DESC, id DESC);

-- +goose Down
DROP INDEX IF EXISTS incident_retention_runs_executed_at;
DROP TABLE IF EXISTS incident_retention_runs;
ALTER TABLE data_maintenance_settings DROP COLUMN incident_retention_days;
