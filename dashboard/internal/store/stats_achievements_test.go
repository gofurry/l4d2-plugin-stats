package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func TestAchievementCompanionTieBreak(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "tie-break.sq3"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE lps_player_segments (
segment_id TEXT PRIMARY KEY, round_id TEXT NOT NULL, steam_id TEXT NOT NULL, side TEXT NOT NULL,
started_at INTEGER NOT NULL, ended_at INTEGER, last_saved_at INTEGER NOT NULL
);
INSERT INTO lps_player_segments VALUES
('s1','r1','1','survivor',0,100,100),
('s2','r2','1','survivor',0,50,50),
('p2','r1','2','survivor',0,100,100),
('p3a','r1','3','survivor',0,50,50),
('p3b','r2','3','survivor',0,50,50),
('p4a','r1','4','survivor',0,50,50),
('p4b','r2','4','survivor',0,50,50);`); err != nil {
		t.Fatal(err)
	}
	stats := &statsStore{db: db, driver: "sqlite", timeout: time.Second}
	values := make(map[string]AchievementMetricValue)
	if err := stats.loadAchievementCompanion(context.Background(), "1", values); err != nil {
		t.Fatal(err)
	}
	metric := values["relationship.max_peer_shared_seconds"]
	if !metric.Available || metric.Value != 100 || metric.EvidenceRounds != 2 || metric.EvidenceSteamID != "3" {
		t.Fatalf("tie-break metric=%#v", metric)
	}
}
