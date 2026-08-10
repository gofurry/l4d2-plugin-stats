-- +goose Up
ALTER TABLE aggregate_rows ADD COLUMN aggregate_version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE aggregate_monthly_rows ADD COLUMN aggregate_version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE aggregate_lifetime_rows ADD COLUMN aggregate_version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE aggregate_state ADD COLUMN aggregate_version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE retention_runs ADD COLUMN aggregate_version INTEGER NOT NULL DEFAULT 1;

-- +goose Down
ALTER TABLE retention_runs DROP COLUMN aggregate_version;
ALTER TABLE aggregate_state DROP COLUMN aggregate_version;
ALTER TABLE aggregate_lifetime_rows DROP COLUMN aggregate_version;
ALTER TABLE aggregate_monthly_rows DROP COLUMN aggregate_version;
ALTER TABLE aggregate_rows DROP COLUMN aggregate_version;
