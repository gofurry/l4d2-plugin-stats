package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type aggregateSummaryKey struct {
	Period    int64
	Kind      string
	ServerKey string
	SteamID   string
	Mode      string
	Dimension string
}

func aggregateMonth(day int64) int64 {
	value := time.Unix(day*86400, 0).UTC()
	return time.Date(value.Year(), value.Month(), 1, 0, 0, 0, 0, time.UTC).Unix() / 86400
}

func addAggregateMetrics(target map[string]int64, values map[string]int64, sign int64) {
	for name, value := range values {
		target[name] += sign * value
		if target[name] == 0 {
			delete(target, name)
		}
	}
}

func accumulateAggregateRows(target map[aggregateSummaryKey]map[string]int64, rows []AggregateRow, monthly bool, sign int64) {
	for _, row := range rows {
		period := int64(0)
		if monthly {
			period = aggregateMonth(row.Day)
		}
		key := aggregateSummaryKey{Period: period, Kind: row.Kind, ServerKey: row.ServerKey, SteamID: row.SteamID, Mode: row.Mode, Dimension: row.Dimension}
		metrics := target[key]
		if metrics == nil {
			metrics = make(map[string]int64)
			target[key] = metrics
		}
		addAggregateMetrics(metrics, row.Metrics, sign)
	}
}

func readAggregateRowsForDays(ctx context.Context, tx *sql.Tx, days []int64) ([]AggregateRow, error) {
	if len(days) == 0 {
		return nil, nil
	}
	query := `SELECT aggregate_version, kind, day, server_key, steam_id, mode, dimension, metrics_json FROM aggregate_rows WHERE day IN (` + strings.TrimSuffix(strings.Repeat("?,", len(days)), ",") + `)`
	args := make([]any, len(days))
	for i, day := range days {
		args[i] = day
	}
	result, err := scanAggregateRows(ctx, tx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("read replaced aggregate days: %w", err)
	}
	return result, nil
}

func scanAggregateRows(ctx context.Context, queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, query string, args ...any) ([]AggregateRow, error) {
	rows, err := queryer.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]AggregateRow, 0)
	for rows.Next() {
		var row AggregateRow
		var metricsJSON string
		if err := rows.Scan(&row.Version, &row.Kind, &row.Day, &row.ServerKey, &row.SteamID, &row.Mode, &row.Dimension, &metricsJSON); err != nil {
			return nil, err
		}
		if err := validateAggregateVersion(row.Version); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(metricsJSON), &row.Metrics); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func replaceAggregateRollups(ctx context.Context, tx *sql.Tx, rows []AggregateRow) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM aggregate_monthly_rows`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM aggregate_lifetime_rows`); err != nil {
		return err
	}
	monthly := make(map[aggregateSummaryKey]map[string]int64)
	lifetime := make(map[aggregateSummaryKey]map[string]int64)
	accumulateAggregateRows(monthly, rows, true, 1)
	accumulateAggregateRows(lifetime, rows, false, 1)
	if err := insertAggregateSummaries(ctx, tx, "aggregate_monthly_rows", "month", monthly); err != nil {
		return err
	}
	return insertAggregateSummaries(ctx, tx, "aggregate_lifetime_rows", "", lifetime)
}

func insertAggregateSummaries(ctx context.Context, tx *sql.Tx, table, periodColumn string, summaries map[aggregateSummaryKey]map[string]int64) error {
	for key, metrics := range summaries {
		encoded, err := json.Marshal(metrics)
		if err != nil {
			return err
		}
		var query string
		var args []any
		if periodColumn != "" {
			query = `INSERT INTO ` + table + ` (kind, ` + periodColumn + `, server_key, steam_id, mode, dimension, metrics_json, aggregate_version) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
			args = []any{key.Kind, key.Period, key.ServerKey, key.SteamID, key.Mode, key.Dimension, string(encoded), AggregateContractVersion}
		} else {
			query = `INSERT INTO ` + table + ` (kind, server_key, steam_id, mode, dimension, metrics_json, aggregate_version) VALUES (?, ?, ?, ?, ?, ?, ?)`
			args = []any{key.Kind, key.ServerKey, key.SteamID, key.Mode, key.Dimension, string(encoded), AggregateContractVersion}
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}
	return nil
}

func applyAggregateRollupDeltas(ctx context.Context, tx *sql.Tx, table, periodColumn string, deltas map[aggregateSummaryKey]map[string]int64) error {
	for key, delta := range deltas {
		where := `kind = ? AND server_key = ? AND steam_id = ? AND mode = ? AND dimension = ?`
		args := []any{key.Kind, key.ServerKey, key.SteamID, key.Mode, key.Dimension}
		if periodColumn != "" {
			where = `kind = ? AND ` + periodColumn + ` = ? AND server_key = ? AND steam_id = ? AND mode = ? AND dimension = ?`
			args = []any{key.Kind, key.Period, key.ServerKey, key.SteamID, key.Mode, key.Dimension}
		}
		var encoded string
		var version int64
		exists := true
		current := make(map[string]int64)
		err := tx.QueryRowContext(ctx, `SELECT aggregate_version, metrics_json FROM `+table+` WHERE `+where, args...).Scan(&version, &encoded)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == nil {
			if err := validateAggregateVersion(version); err != nil {
				return err
			}
			if err := json.Unmarshal([]byte(encoded), &current); err != nil {
				return err
			}
		} else {
			exists = false
		}
		addAggregateMetrics(current, delta, 1)
		for name, value := range current {
			if value < 0 {
				return fmt.Errorf("negative aggregate metric %s for %s", name, key.Kind)
			}
		}
		if len(current) == 0 {
			if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE `+where, args...); err != nil {
				return err
			}
			continue
		}
		value, err := json.Marshal(current)
		if err != nil {
			return err
		}
		if !exists {
			if err := insertAggregateSummaries(ctx, tx, table, periodColumn, map[aggregateSummaryKey]map[string]int64{key: current}); err != nil {
				return err
			}
		} else if _, err := tx.ExecContext(ctx, `UPDATE `+table+` SET metrics_json = ? WHERE `+where, append([]any{string(value)}, args...)...); err != nil {
			return err
		}
	}
	return nil
}

func (s *dashboardStore) ensureAggregateRollups(ctx context.Context) error {
	var daily, lifetime int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM aggregate_rows`).Scan(&daily); err != nil {
		return err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM aggregate_lifetime_rows`).Scan(&lifetime); err != nil {
		return err
	}
	if daily == 0 || lifetime > 0 {
		return nil
	}
	rows, err := scanAggregateRows(ctx, s.db, `SELECT aggregate_version, kind, day, server_key, steam_id, mode, dimension, metrics_json FROM aggregate_rows`)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := replaceAggregateRollups(ctx, tx, rows); err != nil {
		return err
	}
	return tx.Commit()
}
