package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/config"
	_ "modernc.org/sqlite"
)

func TestSQLiteOverviewUsesFrozenModeAndVersionBoundaries(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "stats.sq3")
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		t.Fatal(err)
	}
	applyCollectorSchema(t, db)
	now := time.Now().Unix()
	statements := []string{
		`INSERT INTO lps_schema_migrations VALUES (1, 'initial', 1)`,
		`INSERT INTO lps_players VALUES ('1', 'Alice', 1, 1), ('2', 'Bob', 1, 1)`,
		`INSERT INTO lps_sessions (session_id, boot_id, server_key, steam_id, player_name, ip_address, started_at, last_saved_at, connected_seconds, active_play_seconds, status) VALUES
		 ('s1','b','one','1','Alice','127.0.0.1',1,` + intString(now) + `,500,300,'active'),
		 ('s2','b','two','2','Bob','127.0.0.2',1,1,100,0,'closed')`,
		`INSERT INTO lps_runs (run_id, boot_id, server_key, mode_family, game_mode, started_at, last_saved_at, status) VALUES
		 ('pve','b','one','pve','coop',1,1,'completed'),
		 ('versus','b','two','versus','versus',1,1,'completed'),
		 ('mutation','b','two','pve','mutation12',1,1,'completed')`,
		`INSERT INTO lps_rounds (round_id, run_id, server_key, mode_family, map_name, round_seq, map_seq, attempt_no, started_at, last_saved_at, status) VALUES
		 ('rp','pve','one','pve','c1m1_hotel',1,1,1,1,1,'completed'),
		 ('rv','versus','two','versus','c5m1_waterfront',1,1,1,1,1,'completed')`,
		`INSERT INTO lps_player_segments (segment_id, session_id, run_id, round_id, server_key, steam_id, side, started_at, last_saved_at, status) VALUES
		 ('sp','s1','pve','rp','one','1','survivor',1,1,'closed'),
		 ('sv','s1','versus','rv','two','1','survivor',1,1,'closed'),
		 ('si','s1','versus','rv','two','1','infected',1,1,'closed')`,
		`INSERT INTO lps_pve_segment_stats (segment_id, stats_version, last_saved_at, common_kills, special_kills, tank_kills, witch_kills, incap_revives, ledge_rescues, defib_revives)
		 VALUES ('sp',1,1,100,12,2,1,2,3,4)`,
		`INSERT INTO lps_versus_run_results (run_id, stats_version, last_saved_at, result_status, finalized_at) VALUES ('versus',1,1,'completed',1)`,
		`INSERT INTO lps_versus_round_results (round_id, stats_version, last_saved_at, result_status, finalized_at) VALUES ('rv',1,1,'completed',1)`,
		`INSERT INTO lps_versus_survivor_stats (segment_id, stats_version, last_saved_at, common_kills, human_special_kills, bot_special_kills, human_tank_kills, bot_tank_kills, damage_to_human_special, damage_to_bot_special, damage_to_human_tank, damage_to_bot_tank, deaths, incap_revives, ledge_rescues, defib_revives) VALUES ('sv',1,1,40,7,3,2,1,100,50,200,75,4,2,1,1)`,
		`INSERT INTO lps_versus_infected_stats (segment_id, stats_version, last_saved_at, spawn_count, damage_to_human_survivors, human_survivor_incaps, human_survivor_kills) VALUES ('si',1,1,6,450,3,1)`,
		`INSERT INTO lps_versus_infected_class_stats (segment_id, infected_class, stats_version, last_saved_at, human_survivor_controls) VALUES ('si',3,1,1,11)`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("fixture statement failed: %v\n%s", err, statement)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	stats, err := OpenStats(ctx, config.StatsDatabaseConfig{
		Driver: "sqlite", DSN: databasePath, QueryTimeout: config.Duration(5 * time.Second),
		MaxOpenConns: 2, MaxIdleConns: 1, ConnMaxLifetime: config.Duration(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stats.Close()
	version, err := stats.SchemaVersion(ctx)
	if err != nil || version != 1 {
		t.Fatalf("SchemaVersion() = %d, %v", version, err)
	}
	overview, err := stats.Overview(ctx, time.Now().Add(-7*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if overview.Core.TotalPlayers != 2 || overview.Core.ActivePlayers7Days != 1 || overview.Core.TotalActivePlaySeconds != 300 {
		t.Fatalf("core = %#v", overview.Core)
	}
	if overview.Core.CompletedPVERuns != 1 || overview.Core.CompletedVersusRuns != 1 {
		t.Fatalf("run counts = %#v", overview.Core)
	}
	if overview.PVE != (PVEOverview{CommonKills: 100, SpecialKills: 12, TankKills: 2, WitchKills: 1, Rescues: 9}) {
		t.Fatalf("pve = %#v", overview.PVE)
	}
	if overview.Versus != (VersusOverview{CompletedMatches: 1, CompletedHalves: 1, HumanControlledKills: 9, HumanSurvivorControls: 11}) {
		t.Fatalf("versus = %#v", overview.Versus)
	}
	summary, err := stats.PlayerSummary(ctx, "1")
	if err != nil || summary == nil || summary.LastName != "Alice" || summary.SessionCount != 1 || summary.ActiveSeconds != 300 {
		t.Fatalf("player summary = %#v, %v", summary, err)
	}
	missing, err := stats.PlayerSummary(ctx, "404")
	if err != nil || missing != nil {
		t.Fatalf("missing player = %#v, %v", missing, err)
	}
	active, err := stats.ActivePlayers(ctx, "one", now-600)
	if err != nil || len(active) != 1 || active[0].SteamID != "1" || active[0].Name != "Alice" {
		t.Fatalf("active players = %#v, %v", active, err)
	}
	stale, err := stats.ActivePlayers(ctx, "one", now+1)
	if err != nil || len(stale) != 0 {
		t.Fatalf("stale active players = %#v, %v", stale, err)
	}
	pve, err := stats.PlayerPVE(ctx, "1", 0)
	if err != nil || pve.CommonKills != 100 || pve.SpecialKills != 12 || pve.Revives != 9 {
		t.Fatalf("player pve = %#v, %v", pve, err)
	}
	versus, err := stats.PlayerVersus(ctx, "1", 0)
	if err != nil || versus.SurvivorCommonKills != 40 || versus.HumanSpecialKills != 7 || versus.SurvivorDamage != 425 || versus.InfectedSpawns != 6 || versus.HumanSurvivorControls != 11 {
		t.Fatalf("player versus = %#v, %v", versus, err)
	}
	sessions, err := stats.PlayerSessions(ctx, "1", 0, "", 20)
	if err != nil || len(sessions) != 1 || sessions[0].ServerKey != "one" {
		t.Fatalf("player sessions = %#v, %v", sessions, err)
	}
	chapters, err := stats.PlayerChapters(ctx, "1", 0, "", 20)
	if err != nil || len(chapters) != 3 {
		t.Fatalf("player chapters = %#v, %v", chapters, err)
	}
}

func TestSQLiteIncrementalAggregationAndRetention(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "retention.sq3")
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		t.Fatal(err)
	}
	applyCollectorSchema(t, db)
	old := time.Now().AddDate(-2, 0, 0).Unix()
	saved := time.Now().Add(-time.Minute).Unix()
	statements := []string{
		`INSERT INTO lps_schema_migrations VALUES (1,'initial',1)`,
		`INSERT INTO lps_players VALUES ('1','Alice',1,1)`,
		fmt.Sprintf(`INSERT INTO lps_sessions (session_id,boot_id,server_key,steam_id,player_name,ip_address,started_at,ended_at,last_saved_at,connected_seconds,active_play_seconds,status) VALUES ('s','b','one','1','Alice','127.0.0.1',%d,%d,%d,100,80,'closed')`, old, old+100, saved),
		fmt.Sprintf(`INSERT INTO lps_runs (run_id,boot_id,server_key,mode_family,game_mode,started_at,ended_at,last_saved_at,status) VALUES ('p','b','one','pve','coop',%d,%d,%d,'completed'),('v','b','one','versus','versus',%d,%d,%d,'completed')`, old, old+100, saved, old, old+100, saved),
		fmt.Sprintf(`INSERT INTO lps_rounds (round_id,run_id,server_key,mode_family,map_name,round_seq,map_seq,attempt_no,started_at,ended_at,last_saved_at,status) VALUES ('rp','p','one','pve','map',1,1,1,%d,%d,%d,'completed'),('rv','v','one','versus','map',1,1,1,%d,%d,%d,'completed')`, old, old+100, saved, old, old+100, saved),
		fmt.Sprintf(`INSERT INTO lps_player_segments (segment_id,session_id,run_id,round_id,server_key,steam_id,side,started_at,ended_at,last_saved_at,status) VALUES ('sp','s','p','rp','one','1','survivor',%d,%d,%d,'closed'),('sv','s','v','rv','one','1','survivor',%d,%d,%d,'closed'),('si','s','v','rv','one','1','infected',%d,%d,%d,'closed')`, old, old+100, saved, old, old+100, saved, old, old+100, saved),
		fmt.Sprintf(`INSERT INTO lps_pve_segment_stats (segment_id,stats_version,last_saved_at,common_kills) VALUES ('sp',1,%d,10)`, saved),
		fmt.Sprintf(`INSERT INTO lps_pve_segment_equipment_stats (segment_id,equipment_id,stats_version,last_saved_at,actions) VALUES ('sp',1,1,%d,3)`, saved),
		fmt.Sprintf(`INSERT INTO lps_versus_survivor_infected_class_stats (segment_id,infected_class,stats_version,last_saved_at,human_controller_kills) VALUES ('sv',1,1,%d,2)`, saved),
		fmt.Sprintf(`INSERT INTO lps_versus_infected_class_stats (segment_id,infected_class,stats_version,last_saved_at,spawn_count) VALUES ('si',1,1,%d,4)`, saved),
		fmt.Sprintf(`INSERT INTO lps_versus_round_results (round_id,stats_version,last_saved_at,result_status,finalized_at) VALUES ('rv',1,%d,'completed',%d)`, saved, old+100),
		fmt.Sprintf(`INSERT INTO lps_versus_run_results (run_id,stats_version,last_saved_at,result_status,finalized_at) VALUES ('v',1,%d,'completed',%d)`, saved, old+100),
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("fixture: %v\n%s", err, statement)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	cfg := config.StatsDatabaseConfig{Driver: "sqlite", DSN: databasePath, QueryTimeout: config.Duration(5 * time.Second), MaxOpenConns: 2, MaxIdleConns: 1, ConnMaxLifetime: config.Duration(time.Minute)}
	stats, err := OpenStats(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	change, err := stats.AggregateChanges(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]bool{}
	for _, row := range change.Rows {
		kinds[row.Kind] = true
	}
	for _, kind := range []string{"activity", "pve_equipment", "versus_survivor_class", "versus_infected_class", "run_result", "versus_result"} {
		if !kinds[kind] {
			t.Fatalf("missing aggregate kind %s", kind)
		}
	}
	plan, err := stats.RetentionPlan(ctx, time.Now().AddDate(0, 0, -180).Unix(), time.Now().AddDate(-1, 0, 0).Unix(), time.Now().AddDate(-1, 0, 0).Unix())
	if err != nil {
		t.Fatal(err)
	}
	if plan.EquipmentRowsEligible != 1 || plan.VersusClassRowsEligible != 2 || plan.SessionRowsEligible != 1 || plan.VersusRoundResultsEligible != 1 || plan.VersusRunResultsEligible != 1 {
		t.Fatalf("plan=%+v", plan)
	}
	stats.Close()
	maintenance, err := OpenStatsMaintenance(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	result, err := maintenance.ApplyRetention(ctx, plan)
	if err != nil {
		t.Fatal(err)
	}
	maintenance.Close()
	if result.EquipmentRows != 1 || result.VersusClassRows != 2 || result.SessionRows != 1 || result.VersusRoundResultRows != 1 || result.VersusRunResultRows != 1 {
		t.Fatalf("result=%+v", result)
	}
	verify, err := sql.Open("sqlite", databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer verify.Close()
	for _, table := range []string{"lps_pve_segment_equipment_stats", "lps_versus_survivor_infected_class_stats", "lps_versus_infected_class_stats", "lps_sessions", "lps_versus_round_results", "lps_versus_run_results"} {
		var count int
		if err := verify.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s count=%d err=%v", table, count, err)
		}
	}
	var coreCount int
	if err := verify.QueryRow("SELECT COUNT(*) FROM lps_pve_segment_stats").Scan(&coreCount); err != nil || coreCount != 1 {
		t.Fatalf("core stats count=%d err=%v", coreCount, err)
	}
}

func applyCollectorSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	_, current, _, _ := runtime.Caller(0)
	path := filepath.Clean(filepath.Join(filepath.Dir(current), "..", "..", "..", "database", "migrations", "sqlite", "0001_initial.sql"))
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range strings.Split(string(contents), "-- statement-breakpoint") {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("apply collector schema: %v\n%s", err, statement)
		}
	}
}

func intString(value int64) string {
	return strconv.FormatInt(value, 10)
}
