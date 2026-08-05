package store

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestRetentionDeletesMoreThanOneBatch(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(filepath.Join(t.TempDir(), "retention.db")))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE lps_sessions (session_id TEXT PRIMARY KEY, ended_at INTEGER)`); err != nil {
		t.Fatal(err)
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 1001; index++ {
		if _, err := tx.Exec(`INSERT INTO lps_sessions (session_id, ended_at) VALUES (?, 10)`, fmt.Sprintf("old-%04d", index)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := tx.Exec(`INSERT INTO lps_sessions (session_id, ended_at) VALUES ('new', 1000)`); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	stats := &statsStore{db: db, driver: "sqlite", timeout: time.Second}
	deleted, err := stats.deleteRetentionBatches(context.Background(), retentionDeleteTarget{
		table: "lps_sessions", columns: []string{"session_id"},
		selectSQL: `SELECT session_id FROM lps_sessions WHERE ended_at IS NOT NULL AND ended_at < %s ORDER BY session_id LIMIT 500`,
	}, 100)
	if err != nil {
		t.Fatal(err)
	}
	var remaining int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM lps_sessions`).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if deleted != 1001 || remaining != 1 {
		t.Fatalf("deleted=%d remaining=%d", deleted, remaining)
	}
}
