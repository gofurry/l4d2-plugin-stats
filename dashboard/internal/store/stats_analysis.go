package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

const incidentContractVersion int64 = 1

func (s *statsStore) PlayerCompanions(ctx context.Context, steamID string) ([]PlayerCompanion, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	least, greatest := "MIN", "MAX"
	if s.driver != "sqlite" {
		least, greatest = "LEAST", "GREATEST"
	}
	statement := fmt.Sprintf(`WITH overlap_rows AS (
SELECT b.steam_id AS peer_steam_id, a.round_id,
CASE WHEN %[1]s(COALESCE(a.ended_at,a.last_saved_at),COALESCE(b.ended_at,b.last_saved_at)) > %[2]s(a.started_at,b.started_at)
THEN %[1]s(COALESCE(a.ended_at,a.last_saved_at),COALESCE(b.ended_at,b.last_saved_at)) - %[2]s(a.started_at,b.started_at) ELSE 0 END AS overlap_seconds
FROM lps_player_segments a JOIN lps_player_segments b
ON b.round_id=a.round_id AND b.side=a.side AND b.steam_id<>a.steam_id
WHERE a.steam_id=%[3]s)
SELECT o.peer_steam_id, MAX(p.last_name), SUM(o.overlap_seconds), COUNT(DISTINCT o.round_id)
FROM overlap_rows o JOIN lps_players p ON p.steam_id=o.peer_steam_id
WHERE o.overlap_seconds>0 GROUP BY o.peer_steam_id
ORDER BY SUM(o.overlap_seconds) DESC, COUNT(DISTINCT o.round_id) DESC, o.peer_steam_id ASC LIMIT 3`, least, greatest, s.bind(1))
	rows, err := s.db.QueryContext(queryCtx, statement, steamID)
	if err != nil {
		return nil, fmt.Errorf("query player companions: %w", err)
	}
	defer rows.Close()
	result := make([]PlayerCompanion, 0, 3)
	for rows.Next() {
		var peerSteamID string
		var item PlayerCompanion
		if err := rows.Scan(&peerSteamID, &item.PlayerName, &item.SharedSeconds, &item.SharedRounds); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *statsStore) analysisWhere(filter AnalysisFilter, alias string) (string, []any) {
	clauses := []string{"1=1"}
	args := make([]any, 0, 4)
	add := func(column string, value any) {
		args = append(args, value)
		clauses = append(clauses, column+"="+s.bind(len(args)))
	}
	if filter.Cutoff > 0 {
		args = append(args, filter.Cutoff)
		clauses = append(clauses, alias+".started_at>="+s.bind(len(args)))
	}
	if filter.ServerKey != "" {
		add(alias+".server_key", filter.ServerKey)
	}
	if filter.Mode == "pve" {
		clauses = append(clauses, alias+".mode_family='pve'")
	} else if filter.Mode == "versus" {
		clauses = append(clauses, alias+".mode_family='versus'")
	}
	if filter.CampaignKey != "" {
		add("ru.campaign_key", filter.CampaignKey)
	}
	return strings.Join(clauses, " AND "), args
}

func (s *statsStore) AnalysisOptions(ctx context.Context, filter AnalysisFilter) (AnalysisOptions, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	filter.ServerKey = ""
	filter.CampaignKey = ""
	where, args := s.analysisWhere(filter, "r")
	rows, err := s.db.QueryContext(queryCtx, `SELECT DISTINCT r.server_key,ru.campaign_key
FROM lps_rounds r JOIN lps_runs ru ON ru.run_id=r.run_id WHERE `+where+`
AND EXISTS(SELECT 1 FROM lps_player_segments ps WHERE ps.round_id=r.round_id)
ORDER BY r.server_key ASC,ru.campaign_key ASC`, args...)
	if err != nil {
		return AnalysisOptions{}, fmt.Errorf("query analysis options: %w", err)
	}
	defer rows.Close()
	serverSet, campaignSet := make(map[string]struct{}), make(map[string]struct{})
	for rows.Next() {
		var serverKey, campaignKey string
		if err := rows.Scan(&serverKey, &campaignKey); err != nil {
			return AnalysisOptions{}, err
		}
		if serverKey != "" {
			serverSet[serverKey] = struct{}{}
		}
		if campaignKey != "" {
			campaignSet[campaignKey] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return AnalysisOptions{}, err
	}
	result := AnalysisOptions{Servers: make([]string, 0, len(serverSet)), Campaigns: make([]string, 0, len(campaignSet))}
	for value := range serverSet {
		result.Servers = append(result.Servers, value)
	}
	for value := range campaignSet {
		result.Campaigns = append(result.Campaigns, value)
	}
	sort.Strings(result.Servers)
	sort.Strings(result.Campaigns)
	return result, nil
}

func (s *statsStore) AnalysisMaps(ctx context.Context, filter AnalysisFilter) (AnalysisMaps, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	where, args := s.analysisWhere(filter, "r")
	statement := `WITH eligible AS (
SELECT r.round_id,r.map_name,r.status,r.attempt_no,r.started_at,COALESCE(r.ended_at,r.last_saved_at)-r.started_at duration_seconds
FROM lps_rounds r JOIN lps_runs ru ON ru.run_id=r.run_id WHERE ` + where + `
AND EXISTS(SELECT 1 FROM lps_player_segments ps WHERE ps.round_id=r.round_id)),
complete AS (SELECT e.round_id FROM eligible e JOIN lps_round_contexts c ON c.round_id=e.round_id
WHERE c.context_version=1 AND c.incident_capture_enabled=1 AND c.incident_capture_complete=1 AND c.incident_dropped_count=0),
incident_counts AS (SELECT i.round_id,
SUM(CASE WHEN i.incident_type=1 THEN 1 ELSE 0 END) controls,
SUM(CASE WHEN i.incident_type=2 THEN 1 ELSE 0 END) incaps,
SUM(CASE WHEN i.incident_type=3 THEN 1 ELSE 0 END) deaths
FROM lps_incidents i JOIN complete c ON c.round_id=i.round_id WHERE i.incident_version=1 GROUP BY i.round_id)
SELECT e.map_name,COUNT(*),SUM(CASE WHEN e.status='completed' THEN 1 ELSE 0 END),SUM(CASE WHEN e.status='failed' THEN 1 ELSE 0 END),
AVG(CASE WHEN e.status='completed' THEN e.attempt_no END),AVG(CASE WHEN e.duration_seconds>=0 THEN e.duration_seconds END),
COUNT(c.round_id),COALESCE(SUM(ic.controls),0),COALESCE(SUM(ic.incaps),0),COALESCE(SUM(ic.deaths),0)
FROM eligible e LEFT JOIN complete c ON c.round_id=e.round_id LEFT JOIN incident_counts ic ON ic.round_id=e.round_id
GROUP BY e.map_name ORDER BY e.map_name ASC`
	rows, err := s.db.QueryContext(queryCtx, statement, args...)
	if err != nil {
		return AnalysisMaps{}, fmt.Errorf("query analysis maps: %w", err)
	}
	defer rows.Close()
	result := AnalysisMaps{IncidentVersion: incidentContractVersion, Maps: make([]AnalysisMapRow, 0)}
	var completedAttempts float64
	var completedForAttempts int64
	for rows.Next() {
		var item AnalysisMapRow
		var averageAttempt, averageDuration sql.NullFloat64
		if err := rows.Scan(&item.MapName, &item.EligibleRounds, &item.CompletedRounds, &item.FailedRounds, &averageAttempt, &averageDuration, &item.CompleteIncidentRounds, &item.Controls, &item.Incaps, &item.Deaths); err != nil {
			return AnalysisMaps{}, err
		}
		item.AverageCompletedAttempt = nullableFloat(averageAttempt)
		item.AverageDurationSeconds = nullableFloat(averageDuration)
		result.EligibleRounds += item.EligibleRounds
		result.CompleteIncidentCoverage += float64(item.CompleteIncidentRounds)
		if item.AverageCompletedAttempt != nil {
			completedAttempts += *item.AverageCompletedAttempt * float64(item.CompletedRounds)
			completedForAttempts += item.CompletedRounds
		}
		result.Maps = append(result.Maps, item)
	}
	if err := rows.Err(); err != nil {
		return AnalysisMaps{}, err
	}
	var completed, failed int64
	for _, item := range result.Maps {
		completed += item.CompletedRounds
		failed += item.FailedRounds
	}
	if completed+failed > 0 {
		value := float64(completed) / float64(completed+failed)
		result.CompletionRate = &value
	}
	if completedForAttempts > 0 {
		value := completedAttempts / float64(completedForAttempts)
		result.AverageCompletedAttempt = &value
	}
	if result.EligibleRounds > 0 {
		result.CompleteIncidentCoverage /= float64(result.EligibleRounds)
	}
	incidentWindow := `SELECT COALESCE(MIN(i.occurred_at),0),COALESCE(MAX(i.occurred_at),0)
FROM lps_incidents i JOIN lps_rounds r ON r.round_id=i.round_id JOIN lps_runs ru ON ru.run_id=r.run_id
WHERE i.incident_version=1 AND ` + where
	if err := s.db.QueryRowContext(queryCtx, incidentWindow, args...).Scan(&result.EarliestIncidentAt, &result.LatestIncidentAt); err != nil {
		return AnalysisMaps{}, err
	}
	return result, nil
}

func (s *statsStore) AnalysisMapDetail(ctx context.Context, filter AnalysisFilter, mapName string) (AnalysisMapDetail, error) {
	maps, err := s.AnalysisMaps(ctx, filter)
	if err != nil {
		return AnalysisMapDetail{}, err
	}
	result := AnalysisMapDetail{Timeline: make([]AnalysisTimelinePoint, 0)}
	found := false
	for _, item := range maps.Maps {
		if item.MapName == mapName {
			result.Summary, found = item, true
			break
		}
	}
	if !found {
		return result, nil
	}
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	where, args := s.analysisWhere(filter, "r")
	args = append(args, mapName)
	mapBind := s.bind(len(args))
	base := `WITH complete AS (SELECT r.round_id,COALESCE(r.ended_at,r.last_saved_at)-r.started_at duration_seconds
FROM lps_rounds r JOIN lps_runs ru ON ru.run_id=r.run_id JOIN lps_round_contexts c ON c.round_id=r.round_id
WHERE ` + where + ` AND r.map_name=` + mapBind + ` AND c.context_version=1 AND c.incident_capture_enabled=1
AND c.incident_capture_complete=1 AND c.incident_dropped_count=0
AND EXISTS(SELECT 1 FROM lps_player_segments ps WHERE ps.round_id=r.round_id)) `
	composition := base + `SELECT
COALESCE(SUM(CASE WHEN i.incident_type=1 THEN 1 ELSE 0 END),0),COALESCE(SUM(CASE WHEN i.incident_type=2 THEN 1 ELSE 0 END),0),
COALESCE(SUM(CASE WHEN i.incident_type=3 THEN 1 ELSE 0 END),0),COALESCE(SUM(CASE WHEN i.incident_type=4 THEN 1 ELSE 0 END),0),
COALESCE(SUM(CASE WHEN i.incident_type=5 THEN 1 ELSE 0 END),0),COALESCE(SUM(CASE WHEN i.incident_type=6 THEN 1 ELSE 0 END),0),
COALESCE(SUM(CASE WHEN i.incident_type=7 THEN 1 ELSE 0 END),0)
FROM complete c LEFT JOIN lps_incidents i ON i.round_id=c.round_id AND i.incident_version=1`
	if err := s.db.QueryRowContext(queryCtx, composition, args...).Scan(&result.Composition.Controls, &result.Composition.Incaps, &result.Composition.Deaths, &result.Composition.Revives, &result.Composition.LedgeRescues, &result.Composition.DefibRevives, &result.Composition.CarAlarms); err != nil {
		return AnalysisMapDetail{}, err
	}
	bucketExpression := "i.round_offset_ms/60000"
	if s.driver == "mysql" {
		bucketExpression = "i.round_offset_ms DIV 60000"
	}
	timeline := base + `, buckets AS (SELECT ` + bucketExpression + ` bucket,
SUM(CASE WHEN i.incident_type=1 THEN 1 ELSE 0 END) controls,SUM(CASE WHEN i.incident_type=2 THEN 1 ELSE 0 END) incaps,
SUM(CASE WHEN i.incident_type=3 THEN 1 ELSE 0 END) deaths FROM lps_incidents i JOIN complete c ON c.round_id=i.round_id
WHERE i.incident_version=1 AND i.incident_type IN (1,2,3) GROUP BY i.round_offset_ms/60000)
SELECT b.bucket,(SELECT COUNT(*) FROM complete c WHERE c.duration_seconds>=b.bucket*60),b.controls,b.incaps,b.deaths FROM buckets b ORDER BY b.bucket`
	rows, err := s.db.QueryContext(queryCtx, timeline, args...)
	if err != nil {
		return AnalysisMapDetail{}, err
	}
	for rows.Next() {
		var bucket, reached, controls, incaps, deaths int64
		if err := rows.Scan(&bucket, &reached, &controls, &incaps, &deaths); err != nil {
			rows.Close()
			return AnalysisMapDetail{}, err
		}
		point := AnalysisTimelinePoint{BucketSeconds: bucket * 60, RoundsReached: reached}
		if reached > 0 {
			point.Controls = float64(controls) * 100 / float64(reached)
			point.Incaps = float64(incaps) * 100 / float64(reached)
			point.Deaths = float64(deaths) * 100 / float64(reached)
		}
		result.Timeline = append(result.Timeline, point)
	}
	if err := rows.Close(); err != nil {
		return AnalysisMapDetail{}, err
	}
	if err := s.scanBossAnalysis(queryCtx, base, args, 8, 9, false, &result.Tank); err != nil {
		return AnalysisMapDetail{}, err
	}
	if err := s.scanBossAnalysis(queryCtx, base, args, 10, 11, true, &result.Witch); err != nil {
		return AnalysisMapDetail{}, err
	}
	return result, nil
}

func (s *statsStore) scanBossAnalysis(ctx context.Context, base string, args []any, spawnType, deathType int, witch bool, target *BossAnalysis) error {
	statement := base + `SELECT
COALESCE(SUM(CASE WHEN i.incident_type=` + fmt.Sprint(spawnType) + ` THEN 1 ELSE 0 END),0),
COALESCE(SUM(CASE WHEN i.incident_type=` + fmt.Sprint(deathType) + ` THEN 1 ELSE 0 END),0),
COALESCE(SUM(CASE WHEN i.incident_type=` + fmt.Sprint(deathType) + ` AND i.related_incident_seq>0 THEN 1 ELSE 0 END),0),
AVG(CASE WHEN i.incident_type=` + fmt.Sprint(deathType) + ` AND i.related_incident_seq>0 THEN i.duration_ms/1000.0 END),
MAX(CASE WHEN i.incident_type=` + fmt.Sprint(deathType) + ` AND i.related_incident_seq>0 THEN i.duration_ms/1000.0 END),
COALESCE(SUM(CASE WHEN i.incident_type=` + fmt.Sprint(deathType) + ` AND (i.detail_flags & 1)=1 THEN 1 ELSE 0 END),0)
FROM complete c LEFT JOIN lps_incidents i ON i.round_id=c.round_id AND i.incident_version=1`
	var average, maximum sql.NullFloat64
	if err := s.db.QueryRowContext(ctx, statement, args...).Scan(&target.SpawnCount, &target.DeathCount, &target.MatchedPairs, &average, &maximum, &target.OneShotDeaths); err != nil {
		return err
	}
	target.AverageLifetime = nullableFloat(average)
	target.MaximumLifetime = nullableFloat(maximum)
	if !witch {
		target.OneShotDeaths = 0
	}
	return nil
}

func (s *statsStore) AnalysisContexts(ctx context.Context, filter AnalysisFilter) (AnalysisContexts, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	where, args := s.analysisWhere(filter, "r")
	base := `WITH eligible AS (SELECT r.round_id,r.status,COALESCE(r.ended_at,r.last_saved_at)-r.started_at duration_seconds
FROM lps_rounds r JOIN lps_runs ru ON ru.run_id=r.run_id WHERE ` + where + `
AND EXISTS(SELECT 1 FROM lps_player_segments ps WHERE ps.round_id=r.round_id)) `
	result := AnalysisContexts{Contexts: make([]AnalysisContextRow, 0)}
	counts := base + `SELECT COUNT(*),COALESCE(SUM(CASE WHEN c.round_id IS NOT NULL AND c.context_version=1 AND c.change_mask=0 THEN 1 ELSE 0 END),0),
COALESCE(SUM(CASE WHEN c.round_id IS NOT NULL AND (c.context_version<>1 OR c.change_mask<>0) THEN 1 ELSE 0 END),0),
COALESCE(SUM(CASE WHEN c.round_id IS NULL THEN 1 ELSE 0 END),0) FROM eligible e LEFT JOIN lps_round_contexts c ON c.round_id=e.round_id`
	if err := s.db.QueryRowContext(queryCtx, counts, args...).Scan(&result.EligibleRounds, &result.StableContextRounds, &result.ChangedRuleRounds, &result.NoContextRounds); err != nil {
		return result, err
	}
	statement := base + `SELECT c.context_version,c.ruleset_name,c.difficulty,c.survivor_limit,c.max_player_zombies,c.common_limit,c.tank_health,c.witch_health,
COUNT(*),SUM(CASE WHEN e.status='completed' THEN 1 ELSE 0 END),SUM(CASE WHEN e.status='failed' THEN 1 ELSE 0 END),
AVG(CASE WHEN e.duration_seconds>=0 THEN e.duration_seconds END),SUM(CASE WHEN c.incident_capture_enabled=1 AND c.incident_capture_complete=1 AND c.incident_dropped_count=0 THEN 1 ELSE 0 END)
FROM eligible e JOIN lps_round_contexts c ON c.round_id=e.round_id WHERE c.context_version=1 AND c.change_mask=0
GROUP BY c.context_version,c.ruleset_name,c.difficulty,c.survivor_limit,c.max_player_zombies,c.common_limit,c.tank_health,c.witch_health
ORDER BY COUNT(*) DESC,c.ruleset_name ASC,c.difficulty ASC`
	rows, err := s.db.QueryContext(queryCtx, statement, args...)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var version int64
		var item AnalysisContextRow
		var duration sql.NullFloat64
		if err := rows.Scan(&version, &item.RulesetName, &item.Difficulty, &item.SurvivorLimit, &item.MaxPlayerZombies, &item.CommonLimit, &item.TankHealth, &item.WitchHealth, &item.RoundCount, &item.CompletedRounds, &item.FailedRounds, &duration, &item.CompleteIncidentRounds); err != nil {
			return result, err
		}
		item.AverageDurationSeconds = nullableFloat(duration)
		canonical := fmt.Sprintf("%d\n%s\n%s\n%d\n%d\n%d\n%d\n%d", version, item.RulesetName, item.Difficulty, item.SurvivorLimit, item.MaxPlayerZombies, item.CommonLimit, item.TankHealth, item.WitchHealth)
		hash := sha256.Sum256([]byte(canonical))
		item.Fingerprint = fmt.Sprintf("ctx-%x", hash[:6])
		result.Contexts = append(result.Contexts, item)
	}
	return result, rows.Err()
}

func (s *statsStore) PlayerAnalysisTotals(ctx context.Context, steamID string, filter PlayerFilter, view string) (PlayerAnalysisTotals, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	where := " WHERE ps.steam_id=" + s.bind(1)
	args := []any{steamID}
	if filter.Cutoff > 0 {
		args = append(args, filter.Cutoff)
		where += " AND ps.started_at>=" + s.bind(len(args))
	}
	if filter.ServerKey != "" {
		args = append(args, filter.ServerKey)
		where += " AND ps.server_key=" + s.bind(len(args))
	}
	var statement string
	switch view {
	case "pve":
		statement = `SELECT COALESCE(SUM(ps.active_play_seconds),0),COALESCE(SUM(st.special_kills),0),COALESCE(SUM(st.incap_revives+st.ledge_rescues+st.defib_revives),0),COALESCE(SUM(st.incapacitations),0),COALESCE(SUM(st.deaths),0),COALESCE(SUM(st.friendly_fire_to_humans+st.friendly_fire_to_bots),0),COALESCE(SUM(st.tank_encounters),0),COALESCE(SUM(st.tank_kill_participations),0),COALESCE(SUM(st.witch_encounters),0),COALESCE(SUM(st.witch_kill_participations),0) FROM lps_player_segments ps JOIN lps_runs r ON r.run_id=ps.run_id JOIN lps_pve_segment_stats st ON st.segment_id=ps.segment_id AND st.stats_version=1` + where + ` AND r.mode_family='pve' AND ps.side='survivor'`
	case "versus_survivor":
		statement = `SELECT COALESCE(SUM(ps.active_play_seconds),0),COALESCE(SUM(st.human_special_kills+st.human_tank_kills),0),COALESCE(SUM(st.incap_revives+st.ledge_rescues+st.defib_revives),0),COALESCE(SUM(st.incapacitations),0),COALESCE(SUM(st.damage_to_human_special+st.damage_to_bot_special+st.damage_to_human_tank+st.damage_to_bot_tank),0) FROM lps_player_segments ps JOIN lps_versus_survivor_stats st ON st.segment_id=ps.segment_id AND st.stats_version=1` + where + ` AND ps.side='survivor'`
	case "versus_infected":
		statement = `SELECT COALESCE(SUM(ps.active_play_seconds),0),COALESCE(SUM(st.damage_to_human_survivors),0),COALESCE(SUM(st.spawn_count),0),COALESCE(SUM(st.human_survivor_incaps),0),COALESCE(SUM(st.human_survivor_kills),0),COALESCE(SUM(cls.human_survivor_controls),0),COALESCE(SUM(cls.human_survivor_control_seconds),0) FROM lps_player_segments ps JOIN lps_versus_infected_stats st ON st.segment_id=ps.segment_id AND st.stats_version=1 LEFT JOIN (SELECT segment_id,SUM(human_survivor_controls) human_survivor_controls,SUM(human_survivor_control_seconds) human_survivor_control_seconds FROM lps_versus_infected_class_stats WHERE stats_version=1 GROUP BY segment_id) cls ON cls.segment_id=ps.segment_id` + where + ` AND ps.side='infected'`
	default:
		return PlayerAnalysisTotals{}, fmt.Errorf("unsupported player analysis view %q", view)
	}
	var result PlayerAnalysisTotals
	var err error
	switch view {
	case "pve":
		err = s.db.QueryRowContext(queryCtx, statement, args...).Scan(&result.ActiveSeconds, &result.SpecialKills, &result.Rescues, &result.Incaps, &result.Deaths, &result.FriendlyFire, &result.TankEncounters, &result.TankParticipations, &result.WitchEncounters, &result.WitchParticipations)
	case "versus_survivor":
		err = s.db.QueryRowContext(queryCtx, statement, args...).Scan(&result.ActiveSeconds, &result.SpecialKills, &result.Rescues, &result.Incaps, &result.Damage)
	case "versus_infected":
		err = s.db.QueryRowContext(queryCtx, statement, args...).Scan(&result.ActiveSeconds, &result.Damage, &result.Spawns, &result.Incaps, &result.Kills, &result.Controls, &result.ControlSeconds)
	}
	return result, err
}

func (s *statsStore) PlayerIncidentAnalysis(ctx context.Context, steamID string, filter PlayerFilter) (PlayerIncidentAnalysis, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	result := PlayerIncidentAnalysis{ControlClasses: make([]PlayerIncidentClass, 0), TopRescuers: make([]PlayerRescuer, 0)}
	args := []any{steamID}
	subject := "(SELECT steam_id FROM subject)"
	where := " WHERE i.incident_version=1"
	if filter.Cutoff > 0 {
		args = append(args, filter.Cutoff)
		where += " AND i.occurred_at>=" + s.bind(len(args))
	}
	if filter.ServerKey != "" {
		args = append(args, filter.ServerKey)
		where += " AND r.server_key=" + s.bind(len(args))
	}
	withSubject := `WITH subject AS (SELECT ` + s.bind(1) + ` AS steam_id) `
	statement := withSubject + `SELECT COALESCE(MIN(i.occurred_at),0),COALESCE(MAX(i.occurred_at),0),
COALESCE(SUM(CASE WHEN i.incident_type=1 AND i.target_steam_id=` + subject + ` THEN 1 ELSE 0 END),0),
AVG(CASE WHEN i.incident_type=1 AND i.target_steam_id=` + subject + ` THEN i.duration_ms/1000.0 END),
COALESCE(SUM(CASE WHEN i.incident_type=2 AND i.target_steam_id=` + subject + ` THEN 1 ELSE 0 END),0),
COALESCE(SUM(CASE WHEN i.incident_type=3 AND i.target_steam_id=` + subject + ` THEN 1 ELSE 0 END),0),
COALESCE(SUM(CASE WHEN i.incident_type IN (4,5,6) AND i.actor_steam_id=` + subject + ` THEN 1 ELSE 0 END),0),
COALESCE(SUM(CASE WHEN ((i.incident_type=1 AND i.target_steam_id=` + subject + ` AND i.helper_steam_id<>'') OR (i.incident_type IN (4,5,6) AND i.target_steam_id=` + subject + ` AND i.actor_steam_id<>'')) THEN 1 ELSE 0 END),0)
FROM lps_incidents i JOIN lps_rounds r ON r.round_id=i.round_id` + where
	var average sql.NullFloat64
	if err := s.db.QueryRowContext(queryCtx, statement, args...).Scan(&result.EarliestIncidentAt, &result.LatestIncidentAt, &result.ControlsReceived, &average, &result.Incaps, &result.Deaths, &result.TeammatesRescued, &result.RescuedByTeammates); err != nil {
		return result, err
	}
	result.AverageControlSeconds = nullableFloat(average)
	classRows, err := s.db.QueryContext(queryCtx, withSubject+`SELECT i.infected_class,COUNT(*),AVG(i.duration_ms/1000.0) FROM lps_incidents i JOIN lps_rounds r ON r.round_id=i.round_id`+where+` AND i.incident_type=1 AND i.target_steam_id=`+subject+` GROUP BY i.infected_class ORDER BY COUNT(*) DESC,i.infected_class ASC`, args...)
	if err != nil {
		return result, err
	}
	for classRows.Next() {
		var item PlayerIncidentClass
		if err := classRows.Scan(&item.InfectedClass, &item.Controls, &item.AverageDurationSeconds); err != nil {
			classRows.Close()
			return result, err
		}
		result.ControlClasses = append(result.ControlClasses, item)
	}
	classRows.Close()
	rescueStatement := withSubject + `SELECT MAX(p.last_name),COUNT(*) FROM lps_incidents i JOIN lps_rounds r ON r.round_id=i.round_id JOIN lps_players p ON p.steam_id=CASE WHEN i.incident_type=1 THEN i.helper_steam_id ELSE i.actor_steam_id END` + where + ` AND ((i.incident_type=1 AND i.target_steam_id=` + subject + ` AND i.helper_steam_id<>'') OR (i.incident_type IN (4,5,6) AND i.target_steam_id=` + subject + ` AND i.actor_steam_id<>'')) GROUP BY CASE WHEN i.incident_type=1 THEN i.helper_steam_id ELSE i.actor_steam_id END ORDER BY COUNT(*) DESC,CASE WHEN i.incident_type=1 THEN i.helper_steam_id ELSE i.actor_steam_id END ASC LIMIT 5`
	rescueRows, err := s.db.QueryContext(queryCtx, rescueStatement, args...)
	if err != nil {
		return result, err
	}
	for rescueRows.Next() {
		var item PlayerRescuer
		if err := rescueRows.Scan(&item.PlayerName, &item.Rescues); err != nil {
			rescueRows.Close()
			return result, err
		}
		result.TopRescuers = append(result.TopRescuers, item)
	}
	rescueRows.Close()
	if err := s.populateSynchronizedControls(queryCtx, steamID, filter, &result); err != nil {
		return result, err
	}
	return result, nil
}

func (s *statsStore) populateSynchronizedControls(ctx context.Context, steamID string, filter PlayerFilter, result *PlayerIncidentAnalysis) error {
	args := []any{steamID}
	subjectWhere := "i.actor_steam_id=" + s.bind(1)
	if filter.Cutoff > 0 {
		args = append(args, filter.Cutoff)
		subjectWhere += " AND i.occurred_at>=" + s.bind(len(args))
	}
	if filter.ServerKey != "" {
		args = append(args, filter.ServerKey)
		subjectWhere += " AND r.server_key=" + s.bind(len(args))
	}
	statement := `WITH subject_rounds AS (SELECT DISTINCT i.round_id FROM lps_incidents i JOIN lps_rounds r ON r.round_id=i.round_id
WHERE i.incident_version=1 AND i.incident_type=1 AND i.duration_ms>0 AND r.mode_family='versus' AND ` + subjectWhere + `)
SELECT i.round_id,i.incident_seq,i.actor_steam_id,i.target_steam_id,i.round_offset_ms,i.round_offset_ms+i.duration_ms
FROM lps_incidents i JOIN subject_rounds sr ON sr.round_id=i.round_id
WHERE i.incident_version=1 AND i.incident_type=1 AND i.duration_ms>0 AND i.actor_steam_id<>'' AND i.target_steam_id<>''
ORDER BY i.round_id,i.round_offset_ms,i.incident_seq`
	rows, err := s.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	byRound := make(map[string][]ControlInterval)
	for rows.Next() {
		var roundID string
		var sequence int64
		var interval ControlInterval
		if err := rows.Scan(&roundID, &sequence, &interval.Actor, &interval.Target, &interval.StartMS, &interval.EndMS); err != nil {
			return err
		}
		interval.ID = fmt.Sprintf("%s:%d", roundID, sequence)
		byRound[roundID] = append(byRound[roundID], interval)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, intervals := range byRound {
		participation := SynchronizedControlParticipation(intervals)[steamID]
		result.TwoCapEpisodes += participation[0]
		result.ThreeCapEpisodes += participation[1]
		result.FourCapEpisodes += participation[2]
	}
	return nil
}

func nullableFloat(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	result := value.Float64
	return &result
}

// SynchronizedControlParticipation implements the frozen half-open interval
// sweep. End boundaries sort before starts at the same timestamp.
func SynchronizedControlParticipation(intervals []ControlInterval) map[string][3]int64 {
	type boundary struct {
		at         int64
		start      bool
		target, id string
	}
	boundaries := make([]boundary, 0, len(intervals)*2)
	for _, interval := range intervals {
		if interval.EndMS <= interval.StartMS || interval.ID == "" || interval.Target == "" {
			continue
		}
		boundaries = append(boundaries, boundary{interval.StartMS, true, interval.Target, interval.ID}, boundary{interval.EndMS, false, interval.Target, interval.ID})
	}
	sort.Slice(boundaries, func(i, j int) bool {
		if boundaries[i].at != boundaries[j].at {
			return boundaries[i].at < boundaries[j].at
		}
		if boundaries[i].start != boundaries[j].start {
			return !boundaries[i].start
		}
		return boundaries[i].id < boundaries[j].id
	})
	active := map[string]string{}
	maximum := map[string]int{}
	for _, point := range boundaries {
		if point.start {
			active[point.id] = point.target
		} else {
			delete(active, point.id)
		}
		targets := map[string]struct{}{}
		for _, target := range active {
			targets[target] = struct{}{}
		}
		count := len(targets)
		for id := range active {
			if count > maximum[id] {
				maximum[id] = count
			}
		}
	}
	result := map[string][3]int64{}
	for _, interval := range intervals {
		value := result[interval.Actor]
		if maximum[interval.ID] >= 2 {
			value[0]++
		}
		if maximum[interval.ID] >= 3 {
			value[1]++
		}
		if maximum[interval.ID] >= 4 {
			value[2]++
		}
		result[interval.Actor] = value
	}
	return result
}

type ControlInterval struct {
	ID, Actor, Target string
	StartMS, EndMS    int64
}
