package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/config"
)

const (
	contractBaseTime  int64 = 1700000000
	contractWatermark int64 = contractBaseTime + 500
)

func openDatabaseContractFixture(t *testing.T) StatsDatabase {
	t.Helper()
	driver := strings.ToLower(strings.TrimSpace(os.Getenv("LPS_CONTRACT_DRIVER")))
	if driver == "" {
		driver = "sqlite"
	}
	dsn := strings.TrimSpace(os.Getenv("LPS_CONTRACT_DSN"))
	sqlDriver := driver
	switch driver {
	case "sqlite":
		if dsn == "" {
			dsn = filepath.Join(t.TempDir(), "database-contract.sq3")
		}
	case "mysql":
		if dsn == "" {
			t.Fatal("LPS_CONTRACT_DSN is required for the mysql contract leg")
		}
	case "postgres", "pgsql", "postgresql":
		driver = "postgres"
		sqlDriver = "pgx"
		if dsn == "" {
			t.Fatal("LPS_CONTRACT_DSN is required for the postgres contract leg")
		}
	default:
		t.Fatalf("unsupported LPS_CONTRACT_DRIVER %q", driver)
	}

	db, err := sql.Open(sqlDriver, dsn)
	if err != nil {
		t.Fatalf("open %s fixture database: %v", driver, err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping %s fixture database: %v", driver, err)
	}
	applyContractMigration(t, db, driver)
	insertContractFixture(t, db)
	if err := db.Close(); err != nil {
		t.Fatalf("close fixture writer: %v", err)
	}

	stats, err := OpenStats(context.Background(), config.StatsDatabaseConfig{
		Driver:          driver,
		DSN:             dsn,
		QueryTimeout:    config.Duration(10 * time.Second),
		MaxOpenConns:    4,
		MaxIdleConns:    2,
		ConnMaxLifetime: config.Duration(time.Minute),
	})
	if err != nil {
		t.Fatalf("open %s stats store: %v", driver, err)
	}
	t.Cleanup(func() { _ = stats.Close() })
	return stats
}

func applyContractMigration(t *testing.T, db *sql.DB, driver string) {
	t.Helper()
	migrationDriver := driver
	if driver == "postgres" {
		migrationDriver = "pgsql"
	}
	_, current, _, _ := runtime.Caller(0)
	directory := filepath.Clean(filepath.Join(filepath.Dir(current), "..", "..", "..", "database", "migrations", migrationDriver))
	paths, err := filepath.Glob(filepath.Join(directory, "*.sql"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("list %s migrations: %v", driver, err)
	}
	for _, path := range paths {
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s migration: %v", driver, err)
		}
		for _, statement := range strings.Split(string(contents), "-- statement-breakpoint") {
			statement = strings.TrimSpace(statement)
			if statement == "" {
				continue
			}
			if _, err := db.Exec(statement); err != nil {
				t.Fatalf("apply %s migration %s: %v\n%s", driver, filepath.Base(path), err, statement)
			}
		}
	}
}

func insertContractFixture(t *testing.T, db *sql.DB) {
	t.Helper()
	b := contractBaseTime
	w := contractWatermark
	statements := []string{
		fmt.Sprintf(`INSERT INTO lps_schema_migrations (version,name,applied_at) VALUES (1,'initial',%d),(2,'car_alarms_triggered',%d),(3,'versus_objective_interactions',%d),(4,'analysis_foundation',%d)`, b, b+1, b+2, b+3),
		fmt.Sprintf(`INSERT INTO lps_servers (server_key,display_name,first_seen_at,last_seen_at) VALUES ('one','Server One',%d,%d),('two','Server Two',%d,%d)`, b-100, b+500, b-100, b+500),
		fmt.Sprintf(`INSERT INTO lps_server_boots (boot_id,server_key,started_at,ended_at,last_heartbeat_at,status) VALUES ('boot-one','one',%d,%d,%d,'closed'),('boot-two','two',%d,%d,%d,'closed')`, b-10, b+400, b+400, b-10, b+400, b+400),
		fmt.Sprintf(`INSERT INTO lps_players (steam_id,last_name,first_seen_at,last_seen_at) VALUES ('1','Alice',%d,%d),('2','Bob',%d,%d)`, b-100, b+400, b-50, b+200),
		fmt.Sprintf(`INSERT INTO lps_sessions (session_id,boot_id,server_key,steam_id,player_name,ip_address,started_at,ended_at,last_saved_at,connected_seconds,active_play_seconds,status,disconnect_reason) VALUES ('session-alice','boot-one','one','1','Alice','127.0.0.1',%d,%d,%d,300,240,'closed','quit'),('session-bob','boot-two','two','2','Bob','127.0.0.2',%d,%d,%d,100,60,'closed','timeout')`, b, b+300, w, b+5, b+105, b+300),
		fmt.Sprintf(`INSERT INTO lps_runs (run_id,boot_id,server_key,mode_family,game_mode,campaign_key,started_at,ended_at,last_saved_at,status,round_count,completed_round_count) VALUES ('run-pve','boot-one','one','pve','coop','c1',%d,%d,%d,'completed',1,1),('run-versus','boot-one','one','versus','versus','c5',%d,%d,%d,'completed',1,1)`, b+10, b+200, b+410, b+20, b+300, b+420),
		fmt.Sprintf(`INSERT INTO lps_rounds (round_id,run_id,server_key,mode_family,map_name,round_seq,map_seq,attempt_no,half_no,started_at,ended_at,last_saved_at,status) VALUES ('round-pve','run-pve','one','pve','c1m1_hotel',1,1,1,0,%d,%d,%d,'completed'),('round-versus','run-versus','one','versus','c5m1_waterfront',1,1,1,1,%d,%d,%d,'completed')`, b+20, b+190, b+411, b+30, b+290, b+421),
		fmt.Sprintf(`INSERT INTO lps_player_segments (segment_id,session_id,run_id,round_id,server_key,steam_id,side,started_at,ended_at,last_saved_at,active_play_seconds,status) VALUES ('segment-pve','session-alice','run-pve','round-pve','one','1','survivor',%d,%d,%d,90,'closed'),('segment-vs','session-alice','run-versus','round-versus','one','1','survivor',%d,%d,%d,70,'closed'),('segment-vi','session-alice','run-versus','round-versus','one','1','infected',%d,%d,%d,60,'closed')`, b+30, b+180, b+430, b+40, b+200, b+440, b+50, b+210, b+450),
		fmt.Sprintf(`INSERT INTO lps_pve_segment_stats (segment_id,stats_version,last_saved_at,common_kills,special_kills,tank_kills,witch_kills,damage_to_special,damage_to_tank,damage_to_witch,damage_taken_infected,friendly_fire_to_humans,friendly_fire_to_bots,friendly_fire_taken,incapacitations,deaths,incap_revives,ledge_rescues,defib_revives,rescues_received,medkits_used_self,medkits_used_on_others,medkit_healing_self,medkit_healing_others,pills_used,adrenaline_used,temporary_health_received,chapter_participations,chapter_completions_alive,campaign_completions,smoker_kills,damage_to_smoker,smoker_controls_received,smoker_controlled_seconds,smoker_saves,melee_tongue_self_cuts,tank_rocks_destroyed,witch_oneshots,witch_solo_kills,tank_encounters,tank_kill_participations,witch_encounters,witch_kill_participations,incendiary_packs_deployed,explosive_packs_deployed,objective_interactions,ammo_pile_uses,incapacitated_seconds,ledge_hanging_seconds,black_white_teammates_restored,car_alarms_triggered) VALUES ('segment-pve',1,%d,100,12,2,1,1200,400,200,300,10,5,7,3,1,2,3,4,1,2,1,80,40,3,2,50,1,1,1,4,500,2,30,3,1,2,1,1,2,1,1,1,1,1,2,4,20,10,1,3)`, b+460),
		fmt.Sprintf(`INSERT INTO lps_pve_segment_equipment_stats (segment_id,equipment_id,stats_version,last_saved_at,actions,common_kills,special_kills,tank_kills,witch_kills,headshot_kills,damage_to_special,damage_to_tank,damage_to_witch) VALUES ('segment-pve',7,1,%d,15,25,4,1,1,8,300,100,50)`, b+461),
		fmt.Sprintf(`INSERT INTO lps_versus_survivor_stats (segment_id,stats_version,last_saved_at,common_kills,human_special_kills,bot_special_kills,human_tank_kills,bot_tank_kills,damage_to_human_special,damage_to_bot_special,damage_to_human_tank,damage_to_bot_tank,damage_taken_infected,friendly_fire_to_humans,friendly_fire_to_bots,friendly_fire_taken,incapacitations,deaths,incap_revives,ledge_rescues,defib_revives,rescues_received,medkits_used_self,medkits_used_on_others,medkit_healing_self,medkit_healing_others,pills_used,adrenaline_used,temporary_health_received,witch_kills,damage_to_witch,molotovs_thrown,pipe_bombs_thrown,vomit_jars_thrown,incendiary_packs_deployed,explosive_packs_deployed,melee_tongue_self_cuts,tank_rocks_destroyed,witch_oneshots,witch_solo_kills,objective_interactions,car_alarms_triggered) VALUES ('segment-vs',1,%d,40,7,3,2,1,100,50,200,75,80,4,2,3,2,1,1,2,1,2,1,2,30,50,2,1,20,1,90,1,2,3,1,2,1,1,1,1,5,2)`, b+470),
		fmt.Sprintf(`INSERT INTO lps_versus_survivor_infected_class_stats (segment_id,infected_class,stats_version,last_saved_at,human_controller_kills,bot_controller_kills,damage_to_human_controllers,damage_to_bot_controllers) VALUES ('segment-vs',3,1,%d,5,2,120,60)`, b+471),
		fmt.Sprintf(`INSERT INTO lps_versus_infected_stats (segment_id,stats_version,last_saved_at,spawn_count,damage_to_human_survivors,damage_to_bot_survivors,human_survivor_incaps,bot_survivor_incaps,human_survivor_kills,bot_survivor_kills) VALUES ('segment-vi',1,%d,6,450,120,3,2,1,1)`, b+480),
		fmt.Sprintf(`INSERT INTO lps_versus_infected_class_stats (segment_id,infected_class,stats_version,last_saved_at,spawn_count,damage_to_human_survivors,damage_to_bot_survivors,human_survivor_incaps,bot_survivor_incaps,human_survivor_kills,bot_survivor_kills,human_survivor_controls,bot_survivor_controls,human_survivor_control_seconds,bot_survivor_control_seconds,human_survivor_ability_hits,bot_survivor_ability_hits,human_survivor_ability_damage,bot_survivor_ability_damage) VALUES ('segment-vi',3,1,%d,6,450,120,3,2,1,1,11,4,75,20,9,3,180,40)`, b+481),
		fmt.Sprintf(`INSERT INTO lps_versus_round_results (round_id,stats_version,last_saved_at,scoring_team_slot,team_0_map_score,team_1_map_score,score_available,result_status,finalized_at) VALUES ('round-versus',1,%d,0,500,400,1,'completed',%d)`, b+490, b+290),
		fmt.Sprintf(`INSERT INTO lps_versus_run_results (run_id,stats_version,last_saved_at,team_0_campaign_score,team_1_campaign_score,winner_team_slot,score_available,result_status,finalized_at) VALUES ('run-versus',1,%d,500,400,0,1,'completed',%d)`, w, b+300),
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("insert contract fixture: %v\n%s", err, statement)
		}
	}
}
