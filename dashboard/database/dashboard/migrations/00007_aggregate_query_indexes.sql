-- +goose Up
CREATE INDEX aggregate_rows_kind_server_day
  ON aggregate_rows (kind, server_key, day, mode, steam_id);

CREATE INDEX aggregate_rows_day
  ON aggregate_rows (day);

-- +goose Down
DROP INDEX IF EXISTS aggregate_rows_day;
DROP INDEX IF EXISTS aggregate_rows_kind_server_day;
