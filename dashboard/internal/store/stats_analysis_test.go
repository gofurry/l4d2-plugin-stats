package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/config"
	_ "modernc.org/sqlite"
)

func TestSynchronizedControlParticipationUsesHalfOpenIntervals(t *testing.T) {
	intervals := []ControlInterval{
		{ID: "a", Actor: "infected-1", Target: "survivor-1", StartMS: 0, EndMS: 100},
		{ID: "b", Actor: "infected-2", Target: "survivor-2", StartMS: 20, EndMS: 80},
		{ID: "c", Actor: "infected-3", Target: "survivor-3", StartMS: 40, EndMS: 60},
		{ID: "d", Actor: "infected-4", Target: "survivor-4", StartMS: 50, EndMS: 70},
		// Starts exactly when c ends, so it must not make c a 5-cap episode.
		{ID: "e", Actor: "infected-5", Target: "survivor-5", StartMS: 60, EndMS: 90},
	}
	result := SynchronizedControlParticipation(intervals)
	if result["infected-1"] != [3]int64{1, 1, 1} || result["infected-3"] != [3]int64{1, 1, 1} {
		t.Fatalf("unexpected synchronized participation: %#v", result)
	}
}

func TestSQLiteCompanionsRequireSameSidePositiveOverlap(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "analysis.sq3")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	applyCollectorSchema(t, db)
	statements := []string{
		`INSERT INTO lps_players VALUES ('1','Alice',1,1),('2','Bob',1,2),('3','Carol',1,3),('4','Dave',1,4)`,
		`INSERT INTO lps_player_segments (segment_id,session_id,run_id,round_id,server_key,steam_id,side,started_at,ended_at,last_saved_at,status) VALUES
		('a1','s','r','round-1','one','1','survivor',10,100,100,'closed'),
		('b1','s','r','round-1','one','2','survivor',20,80,80,'closed'),
		('c1','s','r','round-1','one','3','infected',20,80,80,'closed'),
		('d1','s','r','round-1','one','4','survivor',100,120,120,'closed'),
		('a2','s','r','round-2','one','1','survivor',200,NULL,260,'active'),
		('b2','s','r','round-2','one','2','survivor',220,NULL,250,'active')`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	stats, err := OpenStats(ctx, config.StatsDatabaseConfig{Driver: "sqlite", DSN: path, QueryTimeout: config.Duration(5 * time.Second), MaxOpenConns: 2, MaxIdleConns: 1, ConnMaxLifetime: config.Duration(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	defer stats.Close()
	items, err := stats.(StatsAnalysisStore).PlayerCompanions(ctx, "1")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].PlayerName != "Bob" || items[0].SharedSeconds != 90 || items[0].SharedRounds != 2 {
		t.Fatalf("companions=%#v", items)
	}
}
