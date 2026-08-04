package store

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var pveCombatMetrics = []string{
	"common_kills", "special_kills", "tank_kills", "witch_kills",
	"damage_to_special", "damage_to_tank", "damage_to_witch", "damage_taken_infected",
	"friendly_fire_to_humans", "friendly_fire_to_bots", "friendly_fire_taken",
	"incapacitations", "deaths", "incap_revives", "ledge_rescues", "defib_revives", "rescues_received",
	"medkits_used_self", "medkits_used_on_others", "medkit_healing_self", "medkit_healing_others",
	"pills_used", "adrenaline_used", "temporary_health_received",
	"chapter_participations", "chapter_completions_alive", "chapter_completions_dead", "campaign_completions",
}

var pveDetailMetrics = []string{
	"smoker_kills", "boomer_kills", "hunter_kills", "spitter_kills", "jockey_kills", "charger_kills",
	"damage_to_smoker", "damage_to_boomer", "damage_to_hunter", "damage_to_spitter", "damage_to_jockey", "damage_to_charger",
	"smoker_controls_received", "hunter_controls_received", "jockey_controls_received", "charger_controls_received",
	"smoker_controlled_seconds", "hunter_controlled_seconds", "jockey_controlled_seconds", "charger_controlled_seconds",
	"smoker_saves", "hunter_saves", "jockey_saves", "charger_saves",
	"melee_tongue_self_cuts", "tank_rocks_destroyed", "witch_oneshots", "witch_solo_kills",
	"tank_encounters", "tank_kill_participations", "witch_encounters", "witch_kill_participations",
	"incendiary_packs_deployed", "explosive_packs_deployed", "objective_interactions", "ammo_pile_uses",
	"incapacitated_seconds", "ledge_hanging_seconds", "black_white_teammates_restored",
}

var equipmentMetrics = []string{
	"actions", "common_kills", "special_kills", "tank_kills", "witch_kills", "headshot_kills",
	"damage_to_special", "damage_to_tank", "damage_to_witch",
}

var versusSurvivorMetrics = []string{
	"common_kills", "human_special_kills", "bot_special_kills", "human_tank_kills", "bot_tank_kills",
	"damage_to_human_special", "damage_to_bot_special", "damage_to_human_tank", "damage_to_bot_tank",
	"damage_taken_infected", "friendly_fire_to_humans", "friendly_fire_to_bots", "friendly_fire_taken",
	"incapacitations", "deaths", "incap_revives", "ledge_rescues", "defib_revives", "rescues_received",
	"medkits_used_self", "medkits_used_on_others", "medkit_healing_self", "medkit_healing_others",
	"pills_used", "adrenaline_used", "temporary_health_received", "witch_kills", "damage_to_witch",
	"molotovs_thrown", "pipe_bombs_thrown", "vomit_jars_thrown", "incendiary_packs_deployed", "explosive_packs_deployed",
	"melee_tongue_self_cuts", "tank_rocks_destroyed", "witch_oneshots", "witch_solo_kills",
}

var versusSurvivorClassMetrics = []string{
	"human_controller_kills", "bot_controller_kills", "damage_to_human_controllers", "damage_to_bot_controllers",
}

var versusInfectedMetrics = []string{
	"spawn_count", "damage_to_human_survivors", "damage_to_bot_survivors", "human_survivor_incaps",
	"bot_survivor_incaps", "human_survivor_kills", "bot_survivor_kills",
}

var versusInfectedClassMetrics = []string{
	"spawn_count", "damage_to_human_survivors", "damage_to_bot_survivors", "human_survivor_incaps",
	"bot_survivor_incaps", "human_survivor_kills", "bot_survivor_kills", "human_survivor_controls",
	"bot_survivor_controls", "human_survivor_control_seconds", "bot_survivor_control_seconds",
	"human_survivor_ability_hits", "bot_survivor_ability_hits", "human_survivor_ability_damage", "bot_survivor_ability_damage",
}

// AggregateRows creates a complete, rebuildable daily snapshot. The source
// database remains read-only; the caller persists the snapshot in Dashboard DB.
func (s *statsStore) AggregateRows(ctx context.Context) ([]AggregateRow, error) {
	queryCtx, cancel := context.WithTimeout(ctx, maxDuration(s.timeout, 2*time.Minute))
	defer cancel()
	rows := make([]AggregateRow, 0, 1024)
	steps := []func(context.Context) ([]AggregateRow, error){
		s.aggregateActivity,
		s.aggregateModeActivity,
		func(ctx context.Context) ([]AggregateRow, error) {
			return s.aggregateSegmentTable(ctx, "pve_combat", "lps_pve_segment_stats", "p", pveCombatMetrics, "", "s.side='survivor' AND p.stats_version=1 AND r.mode_family='pve' AND r.game_mode IN ('coop','realism')")
		},
		func(ctx context.Context) ([]AggregateRow, error) {
			return s.aggregateSegmentTable(ctx, "pve_detail", "lps_pve_segment_stats", "p", pveDetailMetrics, "", "s.side='survivor' AND p.stats_version=1 AND r.mode_family='pve' AND r.game_mode IN ('coop','realism')")
		},
		func(ctx context.Context) ([]AggregateRow, error) {
			return s.aggregateSegmentTable(ctx, "pve_equipment", "lps_pve_segment_equipment_stats", "p", equipmentMetrics, "p.equipment_id", "s.side='survivor' AND p.stats_version=1 AND r.mode_family='pve' AND r.game_mode IN ('coop','realism')")
		},
		func(ctx context.Context) ([]AggregateRow, error) {
			return s.aggregateSegmentTable(ctx, "versus_survivor", "lps_versus_survivor_stats", "p", versusSurvivorMetrics, "", "s.side='survivor' AND p.stats_version=1 AND r.mode_family='versus' AND r.game_mode='versus'")
		},
		func(ctx context.Context) ([]AggregateRow, error) {
			return s.aggregateSegmentTable(ctx, "versus_survivor_class", "lps_versus_survivor_infected_class_stats", "p", versusSurvivorClassMetrics, "p.infected_class", "s.side='survivor' AND p.stats_version=1 AND r.mode_family='versus' AND r.game_mode='versus'")
		},
		func(ctx context.Context) ([]AggregateRow, error) {
			return s.aggregateSegmentTable(ctx, "versus_infected", "lps_versus_infected_stats", "p", versusInfectedMetrics, "", "s.side='infected' AND p.stats_version=1 AND r.mode_family='versus' AND r.game_mode='versus'")
		},
		func(ctx context.Context) ([]AggregateRow, error) {
			return s.aggregateSegmentTable(ctx, "versus_infected_class", "lps_versus_infected_class_stats", "p", versusInfectedClassMetrics, "p.infected_class", "s.side='infected' AND p.stats_version=1 AND r.mode_family='versus' AND r.game_mode='versus'")
		},
	}
	for _, step := range steps {
		part, err := step(queryCtx)
		if err != nil {
			return nil, err
		}
		rows = append(rows, part...)
	}
	return rows, nil
}

func (s *statsStore) aggregateModeActivity(ctx context.Context) ([]AggregateRow, error) {
	day := s.dayExpression("s.started_at")
	query := fmt.Sprintf(`SELECT %s AS day, s.server_key, s.steam_id, r.game_mode, s.side,
COUNT(*) AS chapter_count, COALESCE(SUM(s.active_play_seconds),0) AS active_play_seconds
FROM lps_player_segments s JOIN lps_runs r ON r.run_id=s.run_id
WHERE (r.mode_family='pve' AND r.game_mode IN ('coop','realism') AND s.side='survivor')
   OR (r.mode_family='versus' AND r.game_mode='versus' AND s.side IN ('survivor','infected'))
GROUP BY %s, s.server_key, s.steam_id, r.game_mode, s.side`, day, day)
	result, err := s.queryAggregate(ctx, query, "mode_activity", "mode", "dimension", []string{"chapter_count", "active_play_seconds"})
	if err != nil {
		return nil, fmt.Errorf("aggregate mode activity: %w", err)
	}
	return result, nil
}

func (s *statsStore) aggregateActivity(ctx context.Context) ([]AggregateRow, error) {
	day := s.dayExpression("started_at")
	query := fmt.Sprintf(`SELECT %s AS day, server_key, steam_id,
COUNT(*) AS session_count, COALESCE(SUM(connected_seconds),0) AS connected_seconds,
COALESCE(SUM(active_play_seconds),0) AS active_play_seconds
FROM lps_sessions GROUP BY %s, server_key, steam_id`, day, day)
	result, err := s.queryAggregate(ctx, query, "activity", "", "", []string{"session_count", "connected_seconds", "active_play_seconds"})
	if err != nil {
		return nil, fmt.Errorf("aggregate activity: %w", err)
	}
	return result, nil
}

func (s *statsStore) aggregateSegmentTable(ctx context.Context, kind, table, alias string, metrics []string, dimension, where string) ([]AggregateRow, error) {
	day := s.dayExpression("s.started_at")
	selects := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		selects = append(selects, fmt.Sprintf("COALESCE(SUM(%s.%s),0) AS %s", alias, metric, metric))
	}
	dimensionSelect := "''"
	groupDimension := ""
	if dimension != "" {
		dimensionSelect = dimension
		groupDimension = ", " + dimension
	}
	query := fmt.Sprintf(`SELECT %s AS day, s.server_key, s.steam_id, r.game_mode, %s AS dimension, %s
FROM %s %s
JOIN lps_player_segments s ON s.segment_id=%s.segment_id
JOIN lps_runs r ON r.run_id=s.run_id
WHERE %s
GROUP BY %s, s.server_key, s.steam_id, r.game_mode%s`, day, dimensionSelect, strings.Join(selects, ", "), table, alias, alias, where, day, groupDimension)
	result, err := s.queryAggregate(ctx, query, kind, "mode", "dimension", metrics)
	if err != nil {
		return nil, fmt.Errorf("aggregate %s: %w", kind, err)
	}
	return result, nil
}

func (s *statsStore) queryAggregate(ctx context.Context, query, kind, modeColumn, dimensionColumn string, metrics []string) ([]AggregateRow, error) {
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]AggregateRow, 0)
	for rows.Next() {
		fixed := 3
		if modeColumn != "" {
			fixed += 2
		}
		values := make([]any, fixed+len(metrics))
		pointers := make([]any, len(values))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}
		row := AggregateRow{Kind: kind, Day: integerValue(values[0]), ServerKey: stringValue(values[1]), SteamID: stringValue(values[2]), Metrics: make(map[string]int64, len(metrics))}
		metricStart := 3
		if modeColumn != "" {
			row.Mode = stringValue(values[3])
			row.Dimension = stringValue(values[4])
			metricStart = 5
		}
		for i, metric := range metrics {
			row.Metrics[metric] = integerValue(values[metricStart+i])
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (s *statsStore) dayExpression(column string) string {
	switch s.driver {
	case "mysql":
		return "CAST(" + column + " / 86400 AS SIGNED)"
	default:
		return "CAST(" + column + " / 86400 AS BIGINT)"
	}
}

func (s *statsStore) RetentionPlan(ctx context.Context, detailCutoff, segmentCutoff int64) (RetentionPlan, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	placeholder1, placeholder2 := "?", "?"
	if s.driver == "postgres" {
		placeholder1, placeholder2 = "$1", "$2"
	}
	query := fmt.Sprintf(`SELECT
  (SELECT COUNT(*) FROM lps_pve_segment_equipment_stats e JOIN lps_player_segments s ON s.segment_id=e.segment_id WHERE s.ended_at IS NOT NULL AND s.ended_at < %s),
  ((SELECT COUNT(*) FROM lps_versus_survivor_infected_class_stats c JOIN lps_player_segments s ON s.segment_id=c.segment_id WHERE s.ended_at IS NOT NULL AND s.ended_at < %s) +
   (SELECT COUNT(*) FROM lps_versus_infected_class_stats c JOIN lps_player_segments s ON s.segment_id=c.segment_id WHERE s.ended_at IS NOT NULL AND s.ended_at < %s)),
  (SELECT COUNT(*) FROM lps_player_segments WHERE ended_at IS NOT NULL AND ended_at < %s)`, placeholder1, placeholder1, placeholder1, placeholder2)
	var equipment, classes, segments int64
	args := []any{detailCutoff, segmentCutoff}
	if s.driver != "postgres" {
		args = []any{detailCutoff, detailCutoff, detailCutoff, segmentCutoff}
	}
	if err := s.db.QueryRowContext(queryCtx, query, args...).Scan(&equipment, &classes, &segments); err != nil {
		return RetentionPlan{}, err
	}
	return RetentionPlan{GeneratedAt: time.Now().Unix(), DetailCutoff: detailCutoff, SegmentCutoff: segmentCutoff, EquipmentRowsEligible: equipment, VersusClassRowsEligible: classes, SegmentRowsEligible: segments, DeletionEnabled: false}, nil
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

func stringValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case nil:
		return ""
	default:
		return fmt.Sprint(v)
	}
}

func integerValue(value any) int64 {
	switch v := value.(type) {
	case int64:
		return v
	case int32:
		return int64(v)
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case []byte:
		n, _ := strconv.ParseInt(string(v), 10, 64)
		return n
	case string:
		n, _ := strconv.ParseInt(v, 10, 64)
		return n
	case nil:
		return 0
	default:
		n, _ := strconv.ParseInt(fmt.Sprint(v), 10, 64)
		return n
	}
}
