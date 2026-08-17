package store

import (
	"context"
	"database/sql"
	"fmt"
)

// ensureIngameVisualV2Schema completes pre-release schema 17 databases that
// were created before Visual v2 added portal background settings. It runs
// before normal migrations so migration 18 can safely read the legacy tables.
func ensureIngameVisualV2Schema(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	columns := []struct {
		table     string
		name      string
		statement string
	}{
		{table: "ingame_settings", name: "background_url", statement: `ALTER TABLE ingame_settings ADD COLUMN background_url TEXT NOT NULL DEFAULT ''`},
		{table: "ingame_server_settings", name: "background_mode", statement: `ALTER TABLE ingame_server_settings ADD COLUMN background_mode TEXT NOT NULL DEFAULT 'inherit' CHECK (background_mode IN ('inherit', 'override', 'hidden'))`},
		{table: "ingame_server_settings", name: "background_url", statement: `ALTER TABLE ingame_server_settings ADD COLUMN background_url TEXT NOT NULL DEFAULT ''`},
	}
	for _, column := range columns {
		tableExists, existsErr := sqliteTableExists(ctx, tx, column.table)
		if existsErr != nil {
			return existsErr
		}
		if !tableExists {
			continue
		}
		exists, existsErr := sqliteColumnExists(ctx, tx, column.table, column.name)
		if existsErr != nil {
			return existsErr
		}
		if exists {
			continue
		}
		if _, execErr := tx.ExecContext(ctx, column.statement); execErr != nil {
			return fmt.Errorf("add %s.%s: %w", column.table, column.name, execErr)
		}
	}
	return tx.Commit()
}

func sqliteTableExists(ctx context.Context, tx *sql.Tx, table string) (bool, error) {
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&count); err != nil {
		return false, fmt.Errorf("inspect %s table: %w", table, err)
	}
	return count > 0, nil
}

func sqliteColumnExists(ctx context.Context, tx *sql.Tx, table, column string) (bool, error) {
	rows, err := tx.QueryContext(ctx, `PRAGMA table_info(`+table+`)`)
	if err != nil {
		return false, fmt.Errorf("inspect %s columns: %w", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notNull, primaryKey int64
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return false, fmt.Errorf("scan %s columns: %w", table, err)
		}
		if name == column {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("read %s columns: %w", table, err)
	}
	return false, nil
}
