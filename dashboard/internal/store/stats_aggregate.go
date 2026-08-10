package store

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
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

var runResultMetrics = []string{"run_count", "completed_runs"}
var versusResultMetrics = []string{"completed_results", "score_available"}

// AggregateRows creates a complete, rebuildable daily snapshot. The source
// database remains read-only; the caller persists the snapshot in Dashboard DB.
func (s *statsStore) AggregateRows(ctx context.Context) ([]AggregateRow, error) {
	queryCtx, cancel := context.WithTimeout(ctx, maxDuration(s.timeout, 2*time.Minute))
	defer cancel()
	return s.aggregateRowsRange(queryCtx, 0, 0)
}

func (s *statsStore) aggregateRowsRange(ctx context.Context, start, end int64) ([]AggregateRow, error) {
	rows := make([]AggregateRow, 0, 1024)
	steps := []func(context.Context) ([]AggregateRow, error){
		func(ctx context.Context) ([]AggregateRow, error) { return s.aggregateActivity(ctx, start, end) },
		func(ctx context.Context) ([]AggregateRow, error) { return s.aggregateModeActivity(ctx, start, end) },
		func(ctx context.Context) ([]AggregateRow, error) { return s.aggregateRuns(ctx, start, end) },
		func(ctx context.Context) ([]AggregateRow, error) { return s.aggregateVersusResults(ctx, start, end) },
		func(ctx context.Context) ([]AggregateRow, error) {
			return s.aggregateSegmentTable(ctx, "pve_combat", "lps_pve_segment_stats", "p", pveCombatMetrics, "", "s.side='survivor' AND p.stats_version=1 AND r.mode_family='pve' AND r.game_mode IN ('coop','realism')", start, end)
		},
		func(ctx context.Context) ([]AggregateRow, error) {
			return s.aggregateSegmentTable(ctx, "pve_detail", "lps_pve_segment_stats", "p", pveDetailMetrics, "", "s.side='survivor' AND p.stats_version=1 AND r.mode_family='pve' AND r.game_mode IN ('coop','realism')", start, end)
		},
		func(ctx context.Context) ([]AggregateRow, error) {
			return s.aggregateSegmentTable(ctx, "pve_equipment", "lps_pve_segment_equipment_stats", "p", equipmentMetrics, "p.equipment_id", "s.side='survivor' AND p.stats_version=1 AND r.mode_family='pve' AND r.game_mode IN ('coop','realism')", start, end)
		},
		func(ctx context.Context) ([]AggregateRow, error) {
			return s.aggregateSegmentTable(ctx, "versus_survivor", "lps_versus_survivor_stats", "p", versusSurvivorMetrics, "", "s.side='survivor' AND p.stats_version=1 AND r.mode_family='versus' AND r.game_mode='versus'", start, end)
		},
		func(ctx context.Context) ([]AggregateRow, error) {
			return s.aggregateSegmentTable(ctx, "versus_survivor_class", "lps_versus_survivor_infected_class_stats", "p", versusSurvivorClassMetrics, "p.infected_class", "s.side='survivor' AND p.stats_version=1 AND r.mode_family='versus' AND r.game_mode='versus'", start, end)
		},
		func(ctx context.Context) ([]AggregateRow, error) {
			return s.aggregateSegmentTable(ctx, "versus_infected", "lps_versus_infected_stats", "p", versusInfectedMetrics, "", "s.side='infected' AND p.stats_version=1 AND r.mode_family='versus' AND r.game_mode='versus'", start, end)
		},
		func(ctx context.Context) ([]AggregateRow, error) {
			return s.aggregateSegmentTable(ctx, "versus_infected_class", "lps_versus_infected_class_stats", "p", versusInfectedClassMetrics, "p.infected_class", "s.side='infected' AND p.stats_version=1 AND r.mode_family='versus' AND r.game_mode='versus'", start, end)
		},
	}
	for _, step := range steps {
		part, err := step(ctx)
		if err != nil {
			return nil, err
		}
		rows = append(rows, part...)
	}
	return rows, nil
}

func (s *statsStore) AggregateChanges(ctx context.Context, after int64) (AggregateChangeSet, error) {
	queryCtx, cancel := context.WithTimeout(ctx, maxDuration(s.timeout, 2*time.Minute))
	defer cancel()
	watermark, err := s.sourceWatermark(queryCtx)
	if err != nil {
		return AggregateChangeSet{}, err
	}
	threshold := after
	if threshold > 0 {
		threshold--
	}
	days, sourceRows, err := s.changedDays(queryCtx, threshold)
	if err != nil {
		return AggregateChangeSet{}, err
	}
	if after == 0 {
		rows, err := s.aggregateRowsRange(queryCtx, 0, 0)
		return AggregateChangeSet{Rows: rows, Days: days, SourceWatermark: watermark, SourceRows: sourceRows, Full: true}, err
	}
	rows := make([]AggregateRow, 0, len(days)*64)
	for _, day := range days {
		part, err := s.aggregateRowsRange(queryCtx, day*86400, (day+1)*86400)
		if err != nil {
			return AggregateChangeSet{}, err
		}
		rows = append(rows, part...)
	}
	return AggregateChangeSet{Rows: rows, Days: days, SourceWatermark: watermark, SourceRows: sourceRows}, nil
}

func (s *statsStore) aggregateModeActivity(ctx context.Context, start, end int64) ([]AggregateRow, error) {
	day := s.dayExpression("s.started_at")
	rangeSQL := timeRangeSQL("s.started_at", start, end)
	query := fmt.Sprintf(`SELECT %s AS day, s.server_key, s.steam_id, r.game_mode, s.side,
COUNT(*) AS chapter_count, COALESCE(SUM(s.active_play_seconds),0) AS active_play_seconds
FROM lps_player_segments s JOIN lps_runs r ON r.run_id=s.run_id
WHERE ((r.mode_family='pve' AND r.game_mode IN ('coop','realism') AND s.side='survivor')
   OR (r.mode_family='versus' AND r.game_mode='versus' AND s.side IN ('survivor','infected')))%s
GROUP BY %s, s.server_key, s.steam_id, r.game_mode, s.side`, day, rangeSQL, day)
	result, err := s.queryAggregate(ctx, query, "mode_activity", "mode", "dimension", []string{"chapter_count", "active_play_seconds"})
	if err != nil {
		return nil, fmt.Errorf("aggregate mode activity: %w", err)
	}
	return result, nil
}

func (s *statsStore) aggregateActivity(ctx context.Context, start, end int64) ([]AggregateRow, error) {
	day := s.dayExpression("started_at")
	rangeSQL := timeRangeSQL("started_at", start, end)
	query := fmt.Sprintf(`SELECT %s AS day, server_key, steam_id,
COUNT(*) AS session_count, COALESCE(SUM(connected_seconds),0) AS connected_seconds,
COALESCE(SUM(active_play_seconds),0) AS active_play_seconds
FROM lps_sessions WHERE 1=1%s GROUP BY %s, server_key, steam_id`, day, rangeSQL, day)
	result, err := s.queryAggregate(ctx, query, "activity", "", "", []string{"session_count", "connected_seconds", "active_play_seconds"})
	if err != nil {
		return nil, fmt.Errorf("aggregate activity: %w", err)
	}
	return result, nil
}

func (s *statsStore) aggregateSegmentTable(ctx context.Context, kind, table, alias string, metrics []string, dimension, where string, start, end int64) ([]AggregateRow, error) {
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
WHERE %s%s
GROUP BY %s, s.server_key, s.steam_id, r.game_mode%s`, day, dimensionSelect, strings.Join(selects, ", "), table, alias, alias, where, timeRangeSQL("s.started_at", start, end), day, groupDimension)
	result, err := s.queryAggregate(ctx, query, kind, "mode", "dimension", metrics)
	if err != nil {
		return nil, fmt.Errorf("aggregate %s: %w", kind, err)
	}
	return result, nil
}

func (s *statsStore) aggregateRuns(ctx context.Context, start, end int64) ([]AggregateRow, error) {
	day := s.dayExpression("started_at")
	query := fmt.Sprintf(`SELECT %s AS day, server_key, '' AS steam_id, game_mode, mode_family,
COUNT(*) AS run_count, COALESCE(SUM(CASE WHEN status='completed' THEN 1 ELSE 0 END),0) AS completed_runs
FROM lps_runs WHERE ((mode_family='pve' AND game_mode IN ('coop','realism')) OR (mode_family='versus' AND game_mode='versus'))%s
GROUP BY %s, server_key, game_mode, mode_family`, day, timeRangeSQL("started_at", start, end), day)
	rows, err := s.queryAggregate(ctx, query, "run_result", "mode", "dimension", runResultMetrics)
	if err != nil {
		return nil, fmt.Errorf("aggregate runs: %w", err)
	}
	return rows, nil
}

func (s *statsStore) aggregateVersusResults(ctx context.Context, start, end int64) ([]AggregateRow, error) {
	day := s.dayExpression("r.started_at")
	roundQuery := fmt.Sprintf(`SELECT %s AS day, r.server_key, '' AS steam_id, 'versus' AS mode, 'round' AS dimension,
COUNT(*) AS completed_results, COALESCE(SUM(v.score_available),0) AS score_available
FROM lps_versus_round_results v JOIN lps_rounds r ON r.round_id=v.round_id
WHERE v.stats_version=1%s GROUP BY %s, r.server_key`, day, timeRangeSQL("r.started_at", start, end), day)
	roundRows, err := s.queryAggregate(ctx, roundQuery, "versus_result", "mode", "dimension", versusResultMetrics)
	if err != nil {
		return nil, fmt.Errorf("aggregate versus round results: %w", err)
	}
	runDay := s.dayExpression("r.started_at")
	runQuery := fmt.Sprintf(`SELECT %s AS day, r.server_key, '' AS steam_id, 'versus' AS mode, 'run' AS dimension,
COUNT(*) AS completed_results, COALESCE(SUM(v.score_available),0) AS score_available
FROM lps_versus_run_results v JOIN lps_runs r ON r.run_id=v.run_id
WHERE v.stats_version=1 AND v.finalized_at IS NOT NULL%s GROUP BY %s, r.server_key`, runDay, timeRangeSQL("r.started_at", start, end), runDay)
	runRows, err := s.queryAggregate(ctx, runQuery, "versus_result", "mode", "dimension", versusResultMetrics)
	if err != nil {
		return nil, fmt.Errorf("aggregate versus run results: %w", err)
	}
	return append(roundRows, runRows...), nil
}

func timeRangeSQL(column string, start, end int64) string {
	if start <= 0 || end <= start {
		return ""
	}
	return fmt.Sprintf(" AND %s >= %d AND %s < %d", column, start, column, end)
}

func (s *statsStore) sourceWatermark(ctx context.Context) (int64, error) {
	query := `SELECT COALESCE(MAX(saved_at),0) FROM (
SELECT last_saved_at AS saved_at FROM lps_sessions UNION ALL SELECT last_saved_at FROM lps_runs
UNION ALL SELECT last_saved_at FROM lps_rounds UNION ALL SELECT last_saved_at FROM lps_versus_round_results
UNION ALL SELECT last_saved_at FROM lps_versus_run_results UNION ALL SELECT last_saved_at FROM lps_player_segments
UNION ALL SELECT last_saved_at FROM lps_pve_segment_stats UNION ALL SELECT last_saved_at FROM lps_pve_segment_equipment_stats
UNION ALL SELECT last_saved_at FROM lps_versus_survivor_stats UNION ALL SELECT last_saved_at FROM lps_versus_survivor_infected_class_stats
UNION ALL SELECT last_saved_at FROM lps_versus_infected_stats UNION ALL SELECT last_saved_at FROM lps_versus_infected_class_stats
) source_rows`
	var value int64
	if err := s.db.QueryRowContext(ctx, query).Scan(&value); err != nil {
		return 0, fmt.Errorf("read stats source watermark: %w", err)
	}
	return value, nil
}

func (s *statsStore) changedDays(ctx context.Context, after int64) ([]int64, int64, error) {
	query := `SELECT day, COUNT(*) FROM (
SELECT {{plain}} AS day FROM lps_sessions WHERE last_saved_at >= {{after}}
UNION ALL SELECT {{plain}} FROM lps_runs WHERE last_saved_at >= {{after}}
UNION ALL SELECT {{plain}} FROM lps_player_segments WHERE last_saved_at >= {{after}}
UNION ALL SELECT {{joined}} FROM lps_pve_segment_stats p JOIN lps_player_segments s ON s.segment_id=p.segment_id WHERE p.last_saved_at >= {{after}}
UNION ALL SELECT {{joined}} FROM lps_pve_segment_equipment_stats p JOIN lps_player_segments s ON s.segment_id=p.segment_id WHERE p.last_saved_at >= {{after}}
UNION ALL SELECT {{joined}} FROM lps_versus_survivor_stats p JOIN lps_player_segments s ON s.segment_id=p.segment_id WHERE p.last_saved_at >= {{after}}
UNION ALL SELECT {{joined}} FROM lps_versus_survivor_infected_class_stats p JOIN lps_player_segments s ON s.segment_id=p.segment_id WHERE p.last_saved_at >= {{after}}
UNION ALL SELECT {{joined}} FROM lps_versus_infected_stats p JOIN lps_player_segments s ON s.segment_id=p.segment_id WHERE p.last_saved_at >= {{after}}
UNION ALL SELECT {{joined}} FROM lps_versus_infected_class_stats p JOIN lps_player_segments s ON s.segment_id=p.segment_id WHERE p.last_saved_at >= {{after}}
UNION ALL SELECT {{round}} FROM lps_versus_round_results v JOIN lps_rounds r ON r.round_id=v.round_id WHERE v.last_saved_at >= {{after}}
UNION ALL SELECT {{round}} FROM lps_versus_run_results v JOIN lps_runs r ON r.run_id=v.run_id WHERE v.last_saved_at >= {{after}}
) changed GROUP BY day ORDER BY day`
	query = strings.NewReplacer(
		"{{plain}}", s.dayExpression("started_at"),
		"{{joined}}", s.dayExpression("s.started_at"),
		"{{round}}", s.dayExpression("r.started_at"),
		"{{after}}", strconv.FormatInt(after, 10),
	).Replace(query)
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, 0, fmt.Errorf("find changed aggregate days: %w", err)
	}
	defer rows.Close()
	var days []int64
	var total int64
	for rows.Next() {
		var day, count int64
		if err := rows.Scan(&day, &count); err != nil {
			return nil, 0, err
		}
		days = append(days, day)
		total += count
	}
	return days, total, rows.Err()
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
		row := AggregateRow{Version: AggregateContractVersion, Kind: kind, Day: integerValue(values[0]), ServerKey: stringValue(values[1]), SteamID: stringValue(values[2]), Metrics: make(map[string]int64, len(metrics))}
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
		return "CAST(FLOOR(" + column + " / 86400) AS SIGNED)"
	default:
		return "CAST(FLOOR(" + column + " / 86400) AS BIGINT)"
	}
}

func (s *statsStore) RetentionPlan(ctx context.Context, detailCutoff, sessionCutoff, resultCutoff int64) (RetentionPlan, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	placeholder1, placeholder2, placeholder3 := "?", "?", "?"
	if s.driver == "postgres" {
		placeholder1, placeholder2, placeholder3 = "$1", "$2", "$3"
	}
	query := fmt.Sprintf(`SELECT
  (SELECT COUNT(*) FROM lps_pve_segment_equipment_stats e JOIN lps_player_segments s ON s.segment_id=e.segment_id WHERE s.ended_at IS NOT NULL AND s.ended_at < %s),
  ((SELECT COUNT(*) FROM lps_versus_survivor_infected_class_stats c JOIN lps_player_segments s ON s.segment_id=c.segment_id WHERE s.ended_at IS NOT NULL AND s.ended_at < %s) +
   (SELECT COUNT(*) FROM lps_versus_infected_class_stats c JOIN lps_player_segments s ON s.segment_id=c.segment_id WHERE s.ended_at IS NOT NULL AND s.ended_at < %s)),
	(SELECT COUNT(*) FROM lps_sessions WHERE ended_at IS NOT NULL AND ended_at < %s),
	(SELECT COUNT(*) FROM lps_versus_round_results WHERE finalized_at < %s),
	(SELECT COUNT(*) FROM lps_versus_run_results WHERE finalized_at IS NOT NULL AND finalized_at < %s)`, placeholder1, placeholder1, placeholder1, placeholder2, placeholder3, placeholder3)
	var equipment, classes, sessions, roundResults, runResults int64
	args := []any{detailCutoff, sessionCutoff, resultCutoff}
	if s.driver != "postgres" {
		args = []any{detailCutoff, detailCutoff, detailCutoff, sessionCutoff, resultCutoff, resultCutoff}
	}
	if err := s.db.QueryRowContext(queryCtx, query, args...).Scan(&equipment, &classes, &sessions, &roundResults, &runResults); err != nil {
		return RetentionPlan{}, err
	}
	watermark, err := s.sourceWatermark(queryCtx)
	if err != nil {
		return RetentionPlan{}, err
	}
	return RetentionPlan{AggregateVersion: AggregateContractVersion, GeneratedAt: time.Now().Unix(), DetailCutoff: detailCutoff, SessionCutoff: sessionCutoff, ResultCutoff: resultCutoff, EquipmentRowsEligible: equipment, VersusClassRowsEligible: classes, SessionRowsEligible: sessions, VersusRoundResultsEligible: roundResults, VersusRunResultsEligible: runResults, SourceWatermark: watermark}, nil
}

func (s *statsStore) ApplyRetention(ctx context.Context, plan RetentionPlan) (RetentionResult, error) {
	if err := validateAggregateVersion(plan.AggregateVersion); err != nil {
		return RetentionResult{}, fmt.Errorf("apply retention: %w", err)
	}
	queryCtx, cancel := context.WithTimeout(ctx, maxDuration(s.timeout, 10*time.Minute))
	defer cancel()
	equipment, err := s.deleteRetentionBatches(queryCtx, retentionDeleteTarget{
		table: "lps_pve_segment_equipment_stats", columns: []string{"segment_id", "equipment_id"},
		selectSQL: `SELECT e.segment_id, e.equipment_id FROM lps_pve_segment_equipment_stats e JOIN lps_player_segments s ON s.segment_id=e.segment_id WHERE s.ended_at IS NOT NULL AND s.ended_at < %s ORDER BY e.segment_id, e.equipment_id LIMIT 500`,
	}, plan.DetailCutoff)
	if err != nil {
		return RetentionResult{}, fmt.Errorf("delete equipment detail: %w", err)
	}
	survivorClasses, err := s.deleteRetentionBatches(queryCtx, retentionDeleteTarget{
		table: "lps_versus_survivor_infected_class_stats", columns: []string{"segment_id", "infected_class"},
		selectSQL: `SELECT c.segment_id, c.infected_class FROM lps_versus_survivor_infected_class_stats c JOIN lps_player_segments s ON s.segment_id=c.segment_id WHERE s.ended_at IS NOT NULL AND s.ended_at < %s ORDER BY c.segment_id, c.infected_class LIMIT 500`,
	}, plan.DetailCutoff)
	if err != nil {
		return RetentionResult{}, fmt.Errorf("delete versus survivor class detail: %w", err)
	}
	infectedClasses, err := s.deleteRetentionBatches(queryCtx, retentionDeleteTarget{
		table: "lps_versus_infected_class_stats", columns: []string{"segment_id", "infected_class"},
		selectSQL: `SELECT c.segment_id, c.infected_class FROM lps_versus_infected_class_stats c JOIN lps_player_segments s ON s.segment_id=c.segment_id WHERE s.ended_at IS NOT NULL AND s.ended_at < %s ORDER BY c.segment_id, c.infected_class LIMIT 500`,
	}, plan.DetailCutoff)
	if err != nil {
		return RetentionResult{}, fmt.Errorf("delete versus infected class detail: %w", err)
	}
	sessions, err := s.deleteRetentionBatches(queryCtx, retentionDeleteTarget{
		table: "lps_sessions", columns: []string{"session_id"},
		selectSQL: `SELECT session_id FROM lps_sessions WHERE ended_at IS NOT NULL AND ended_at < %s ORDER BY session_id LIMIT 500`,
	}, plan.SessionCutoff)
	if err != nil {
		return RetentionResult{}, fmt.Errorf("delete closed sessions: %w", err)
	}
	roundResults, err := s.deleteRetentionBatches(queryCtx, retentionDeleteTarget{
		table: "lps_versus_round_results", columns: []string{"round_id"},
		selectSQL: `SELECT round_id FROM lps_versus_round_results WHERE finalized_at < %s ORDER BY round_id LIMIT 500`,
	}, plan.ResultCutoff)
	if err != nil {
		return RetentionResult{}, fmt.Errorf("delete versus round results: %w", err)
	}
	runResults, err := s.deleteRetentionBatches(queryCtx, retentionDeleteTarget{
		table: "lps_versus_run_results", columns: []string{"run_id"},
		selectSQL: `SELECT run_id FROM lps_versus_run_results WHERE finalized_at IS NOT NULL AND finalized_at < %s ORDER BY run_id LIMIT 500`,
	}, plan.ResultCutoff)
	if err != nil {
		return RetentionResult{}, fmt.Errorf("delete versus run results: %w", err)
	}
	return RetentionResult{
		RunID: uuid.NewString(), ExecutedAt: time.Now().Unix(), EquipmentRows: equipment,
		VersusClassRows: survivorClasses + infectedClasses, SessionRows: sessions,
		VersusRoundResultRows: roundResults, VersusRunResultRows: runResults,
	}, nil
}

type retentionDeleteTarget struct {
	table     string
	columns   []string
	selectSQL string
}

func (s *statsStore) deleteRetentionBatches(ctx context.Context, target retentionDeleteTarget, cutoff int64) (int64, error) {
	var total int64
	for {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return total, err
		}
		placeholder := "?"
		if s.driver == "postgres" {
			placeholder = "$1"
		}
		rows, err := tx.QueryContext(ctx, fmt.Sprintf(target.selectSQL, placeholder), cutoff)
		if err != nil {
			tx.Rollback()
			return total, err
		}
		keys := make([][]any, 0, 500)
		for rows.Next() {
			values := make([]any, len(target.columns))
			destinations := make([]any, len(values))
			for index := range values {
				destinations[index] = &values[index]
			}
			if err := rows.Scan(destinations...); err != nil {
				rows.Close()
				tx.Rollback()
				return total, err
			}
			keys = append(keys, values)
		}
		if err := rows.Close(); err != nil {
			tx.Rollback()
			return total, err
		}
		if len(keys) == 0 {
			if err := tx.Commit(); err != nil {
				return total, err
			}
			return total, nil
		}
		conditions := make([]string, 0, len(keys))
		args := make([]any, 0, len(keys)*len(target.columns))
		position := 1
		for _, key := range keys {
			parts := make([]string, len(target.columns))
			for index, column := range target.columns {
				marker := "?"
				if s.driver == "postgres" {
					marker = "$" + strconv.Itoa(position)
				}
				parts[index] = column + " = " + marker
				args = append(args, key[index])
				position++
			}
			conditions = append(conditions, "("+strings.Join(parts, " AND ")+")")
		}
		result, err := tx.ExecContext(ctx, `DELETE FROM `+target.table+` WHERE `+strings.Join(conditions, " OR "), args...)
		if err != nil {
			tx.Rollback()
			return total, err
		}
		deleted, err := result.RowsAffected()
		if err != nil {
			tx.Rollback()
			return total, err
		}
		if err := tx.Commit(); err != nil {
			return total, err
		}
		total += deleted
	}
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
