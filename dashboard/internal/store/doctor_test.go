package store

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/config"
)

func TestDeepDataQualityDetectsContractAnomalies(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "doctor.sq3")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	applyCollectorSchema(t, db)
	now := time.Now().Unix()
	statements := []string{
		`INSERT INTO lps_schema_migrations VALUES (1,'initial',1)`,
		fmt.Sprintf(`INSERT INTO lps_servers VALUES ('one','One',1,%d)`, now),
		fmt.Sprintf(`INSERT INTO lps_server_boots (boot_id,server_key,started_at,last_heartbeat_at,status) VALUES ('boot','one',1,%d,'active')`, now),
		`INSERT INTO lps_players VALUES ('1','Alice',1,1)`,
		fmt.Sprintf(`INSERT INTO lps_sessions (session_id,boot_id,server_key,steam_id,player_name,ip_address,started_at,last_saved_at,status) VALUES ('session','boot','one','1','Alice','127.0.0.1',1,%d,'active')`, now),
		fmt.Sprintf(`INSERT INTO lps_runs (run_id,boot_id,server_key,mode_family,game_mode,started_at,last_saved_at,status) VALUES ('run','boot','one','pve','coop',1,%d,'active')`, now),
		fmt.Sprintf(`INSERT INTO lps_rounds (round_id,run_id,server_key,mode_family,map_name,round_seq,map_seq,attempt_no,started_at,last_saved_at,status) VALUES ('round','run','one','pve','map',1,1,1,1,%d,'active')`, now),
		fmt.Sprintf(`INSERT INTO lps_player_segments (segment_id,session_id,run_id,round_id,server_key,steam_id,side,started_at,last_saved_at,status) VALUES ('segment','session','run','round','one','1','survivor',1,%d,'active')`, now),
		fmt.Sprintf(`INSERT INTO lps_pve_segment_stats (segment_id,stats_version,last_saved_at,special_kills,damage_to_special,smoker_kills,boomer_kills,hunter_kills,spitter_kills,jockey_kills,charger_kills,damage_to_smoker,damage_to_boomer,damage_to_hunter,damage_to_spitter,damage_to_jockey,damage_to_charger) VALUES ('segment',1,%d,6,60,1,1,1,1,1,1,10,10,10,10,10,10)`, now),
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("healthy fixture: %v\n%s", err, statement)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	stats := openDoctorStats(t, ctx, path)
	healthy, err := stats.DeepDataQuality(ctx, now-15*60)
	if err != nil {
		t.Fatal(err)
	}
	if healthy.StaleActiveBoots.Count+healthy.UnknownStatsVersion.Count+healthy.LifecycleLinks.Count+healthy.ModeSideMismatch.Count+healthy.PVETotalMismatch.Count != 0 {
		t.Fatalf("healthy quality report=%+v", healthy)
	}
	stats.Close()

	db, err = sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	anomalies := []string{
		fmt.Sprintf(`UPDATE lps_server_boots SET last_heartbeat_at=%d WHERE boot_id='boot'`, now-16*60),
		`UPDATE lps_player_segments SET side='infected' WHERE segment_id='segment'`,
		`UPDATE lps_pve_segment_stats SET special_kills=7, damage_to_special=61 WHERE segment_id='segment'`,
		`UPDATE lps_pve_segment_stats SET special_assists=2,smoker_assists=1,boomer_assists=0,hunter_assists=0,spitter_assists=0,jockey_assists=0,charger_assists=0 WHERE segment_id='segment'`,
		`INSERT INTO lps_players VALUES ('2','Bob',1,1)`,
		fmt.Sprintf(`INSERT INTO lps_player_round_relationship_stats (round_id,actor_steam_id,target_steam_id,relationship_version,friendly_fire_damage,last_saved_at,revision) VALUES ('round','2','1',1,1,%d,1)`, now),
		fmt.Sprintf(`INSERT INTO lps_pve_segment_equipment_stats (segment_id,equipment_id,stats_version,last_saved_at) VALUES ('missing-segment',1,2,%d)`, now),
	}
	for _, statement := range anomalies {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	stats = openDoctorStats(t, ctx, path)
	defer stats.Close()
	report, err := stats.DeepDataQuality(ctx, now-15*60)
	if err != nil {
		t.Fatal(err)
	}
	for name, finding := range map[string]DataQualityFinding{
		"stale boot": report.StaleActiveBoots, "stats version": report.UnknownStatsVersion,
		"lifecycle": report.LifecycleLinks, "mode/side": report.ModeSideMismatch, "pve totals": report.PVETotalMismatch,
		"relationship": report.RelationshipContract, "pve assist": report.PVEAssistContract,
	} {
		if finding.Count != 1 || len(finding.IDs) != 1 {
			t.Errorf("%s finding=%+v", name, finding)
		}
	}
}

func openDoctorStats(t *testing.T, ctx context.Context, path string) StatsDatabase {
	t.Helper()
	stats, err := OpenStats(ctx, config.StatsDatabaseConfig{
		Driver: "sqlite", DSN: path, QueryTimeout: config.Duration(5 * time.Second),
		MaxOpenConns: 2, MaxIdleConns: 1, ConnMaxLifetime: config.Duration(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	return stats
}
