package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
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

func TestSQLiteAnalysisOptionsFollowRangeAndMode(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "analysis-options.sq3")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	applyCollectorSchema(t, db)
	statements := []string{
		`INSERT INTO lps_server_boots (boot_id,server_key,started_at,last_heartbeat_at,status) VALUES ('b1','one',1,1,'closed'),('b2','two',1,1,'closed')`,
		`INSERT INTO lps_runs (run_id,boot_id,server_key,mode_family,game_mode,campaign_key,started_at,last_saved_at,status) VALUES ('r1','b1','one','pve','coop','c1',100,100,'completed'),('r2','b2','two','pve','realism','c2',200,200,'completed'),('r3','b1','one','versus','versus','c5',300,300,'completed')`,
		`INSERT INTO lps_rounds (round_id,run_id,server_key,mode_family,map_name,round_seq,map_seq,attempt_no,started_at,last_saved_at,status) VALUES ('round-1','r1','one','pve','m1',1,1,1,100,100,'completed'),('round-2','r2','two','pve','m2',1,1,1,200,200,'completed'),('round-3','r3','one','versus','m3',1,1,1,300,300,'completed')`,
		`INSERT INTO lps_player_segments (segment_id,session_id,run_id,round_id,server_key,steam_id,side,started_at,last_saved_at,status) VALUES ('s1','x','r1','round-1','one','1','survivor',100,100,'closed'),('s2','x','r2','round-2','two','1','survivor',200,200,'closed'),('s3','x','r3','round-3','one','1','infected',300,300,'closed')`,
		`INSERT INTO lps_round_contexts (round_id,context_version,captured_at,last_saved_at,collector_version,ruleset_name,difficulty,survivor_limit,max_player_zombies,common_limit,tank_health,witch_health,change_mask,incident_capture_enabled,incident_capture_complete,incident_expected_count,incident_dropped_count,revision) VALUES
		('round-1',1,100,100,'1.3.0','Alpha','Normal',4,4,30,4000,1000,0,1,1,0,0,1),
		('round-2',1,200,200,'1.3.0','Bravo','Hard',8,8,20,8000,2000,0,1,1,0,0,1)`,
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
	options, err := stats.(StatsAnalysisStore).AnalysisOptions(ctx, AnalysisFilter{Mode: "pve", Cutoff: 150, ServerKey: "ignored", CampaignKey: "ignored"})
	if err != nil {
		t.Fatal(err)
	}
	if len(options.Servers) != 1 || options.Servers[0] != "two" || len(options.Campaigns) != 1 || options.Campaigns[0] != "c2" {
		t.Fatalf("options=%#v", options)
	}
	maps, err := stats.(StatsAnalysisStore).AnalysisMaps(ctx, AnalysisFilter{Mode: "pve", Page: 1, PageSize: 1, Sort: "map_name", Order: "desc"})
	if err != nil {
		t.Fatal(err)
	}
	if maps.Page != 1 || maps.PageSize != 1 || maps.Total != 2 || maps.EligibleRounds != 2 || len(maps.Maps) != 1 || maps.Maps[0].MapName != "m2" {
		t.Fatalf("paginated maps=%#v", maps)
	}
	for _, sortName := range []string{"map_name", "eligible_rounds", "completion_rate", "average_completed_attempt", "average_duration_seconds", "incaps_per_complete_round", "deaths_per_complete_round", "controls_per_complete_round"} {
		if _, err := stats.(StatsAnalysisStore).AnalysisMaps(ctx, AnalysisFilter{Mode: "pve", Page: 1, PageSize: 1, Sort: sortName, Order: "desc"}); err != nil {
			t.Errorf("sort maps by %s: %v", sortName, err)
		}
	}
	detail, err := stats.(StatsAnalysisStore).AnalysisMapDetail(ctx, AnalysisFilter{Mode: "pve", Page: 2, PageSize: 1, Sort: "map_name", Order: "desc"}, "m1")
	if err != nil || detail.Summary.MapName != "m1" {
		t.Fatalf("map detail=%#v err=%v", detail, err)
	}
	if detail.Timeline == nil || detail.RecentIncidents == nil {
		t.Fatalf("legacy map detail must use empty collections instead of nil: %#v", detail)
	}
	encoded, err := json.Marshal(detail)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), `"timeline":null`) || strings.Contains(string(encoded), `"recent_incidents":null`) {
		t.Fatalf("legacy map detail contains null collections: %s", encoded)
	}
	contexts, err := stats.(StatsAnalysisStore).AnalysisContexts(ctx, AnalysisFilter{Mode: "pve", Page: 2, PageSize: 1, Sort: "ruleset_name", Order: "asc"})
	if err != nil {
		t.Fatal(err)
	}
	if contexts.Page != 2 || contexts.PageSize != 1 || contexts.Total != 2 || contexts.EligibleRounds != 2 || len(contexts.Contexts) != 1 || contexts.Contexts[0].RulesetName != "Bravo" {
		t.Fatalf("paginated contexts=%#v", contexts)
	}
	for _, sortName := range []string{"ruleset_name", "round_count", "completion_rate", "average_duration_seconds", "complete_incident_coverage"} {
		if _, err := stats.(StatsAnalysisStore).AnalysisContexts(ctx, AnalysisFilter{Mode: "pve", Page: 1, PageSize: 1, Sort: sortName, Order: "asc"}); err != nil {
			t.Errorf("sort contexts by %s: %v", sortName, err)
		}
	}
}
