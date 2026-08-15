package store

import (
	"context"
	"database/sql"
	"fmt"
)

const achievementMetricSource = `
SELECT s.steam_id,s.last_saved_at watermark FROM lps_player_segments s
UNION ALL SELECT s.steam_id,p.last_saved_at FROM lps_pve_segment_stats p JOIN lps_player_segments s ON s.segment_id=p.segment_id
UNION ALL SELECT s.steam_id,v.last_saved_at FROM lps_versus_survivor_stats v JOIN lps_player_segments s ON s.segment_id=v.segment_id
UNION ALL SELECT s.steam_id,v.last_saved_at FROM lps_versus_infected_stats v JOIN lps_player_segments s ON s.segment_id=v.segment_id
UNION ALL SELECT s.steam_id,e.last_saved_at FROM lps_pve_segment_equipment_stats e JOIN lps_player_segments s ON s.segment_id=e.segment_id
UNION ALL SELECT rel.actor_steam_id,rel.last_saved_at FROM lps_player_round_relationship_stats rel
UNION ALL SELECT rel.target_steam_id,rel.last_saved_at FROM lps_player_round_relationship_stats rel`

func (s *statsStore) PlayerAchievementWatermark(ctx context.Context, steamID string) (int64, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	statement := `SELECT COALESCE(MAX(watermark),0) FROM (` + achievementMetricSource + `) source WHERE steam_id=` + s.bind(1)
	var watermark int64
	if err := s.db.QueryRowContext(queryCtx, statement, steamID).Scan(&watermark); err != nil {
		return 0, fmt.Errorf("read player achievement watermark: %w", err)
	}
	return watermark, nil
}

func (s *statsStore) PlayerAchievementMetrics(ctx context.Context, steamID string) (PlayerAchievementMetrics, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	result := PlayerAchievementMetrics{SteamID: steamID, Values: make(map[string]AchievementMetricValue)}
	var err error
	result.Watermark, err = s.PlayerAchievementWatermark(queryCtx, steamID)
	if err != nil {
		return result, err
	}

	var active sql.NullInt64
	if err := s.db.QueryRowContext(queryCtx, `SELECT SUM(active_play_seconds) FROM lps_player_segments WHERE steam_id=`+s.bind(1), steamID).Scan(&active); err != nil {
		return result, fmt.Errorf("read achievement activity metric: %w", err)
	}
	putAchievementMetric(result.Values, "career.active_play_seconds", active)

	pveNames := []string{
		"pve.campaign_completions", "pve.common_kills", "pve.special_kills", "pve.special_assists",
		"pve.incap_revives", "pve.smoker_saves", "pve.hunter_saves", "pve.jockey_saves", "pve.charger_saves",
		"pve.medkit_healing_others", "pve.tank_kills", "pve.witch_kills", "pve.tank_kill_participations", "pve.witch_kill_participations",
		"tank_rocks_destroyed", "witch_oneshots", "witch_solo_kills", "melee_tongue_self_cuts",
		"pve.defib_revives", "pve.black_white_restores", "pve.survivor_fall_deaths", "pve.survivor_car_alarms",
		"pve.survivor_friendly_fire_to_humans", "pve.survivor_incapacitations",
		"pve.objective_interactions", "pve.pills_used", "pve.adrenaline_used",
		"pve.incendiary_packs_deployed", "pve.explosive_packs_deployed",
	}
	pveValues := make([]sql.NullInt64, len(pveNames))
	pveDest := make([]any, len(pveValues))
	for i := range pveValues {
		pveDest[i] = &pveValues[i]
	}
	pveSQL := `SELECT SUM(p.campaign_completions),SUM(p.common_kills),SUM(p.special_kills),SUM(p.special_assists),
SUM(p.incap_revives),SUM(p.smoker_saves),SUM(p.hunter_saves),SUM(p.jockey_saves),SUM(p.charger_saves),
SUM(p.medkit_healing_others),SUM(p.tank_kills),SUM(p.witch_kills),SUM(p.tank_kill_participations),SUM(p.witch_kill_participations),
SUM(p.tank_rocks_destroyed),SUM(p.witch_oneshots),SUM(p.witch_solo_kills),SUM(p.melee_tongue_self_cuts),
SUM(p.defib_revives),SUM(p.black_white_teammates_restored),SUM(p.fall_deaths),SUM(p.car_alarms_triggered),
SUM(p.friendly_fire_to_humans),SUM(p.incapacitations),
SUM(p.objective_interactions),SUM(p.pills_used),SUM(p.adrenaline_used),
SUM(p.incendiary_packs_deployed),SUM(p.explosive_packs_deployed)
FROM lps_pve_segment_stats p JOIN lps_player_segments s ON s.segment_id=p.segment_id
WHERE p.stats_version=1 AND s.steam_id=` + s.bind(1)
	if err := s.db.QueryRowContext(queryCtx, pveSQL, steamID).Scan(pveDest...); err != nil {
		return result, fmt.Errorf("read PvE achievement metrics: %w", err)
	}
	for i, name := range pveNames {
		putAchievementMetric(result.Values, name, pveValues[i])
	}

	versusNames := []string{
		"versus.human_special_kills", "versus.defib_revives", "versus.black_white_restores",
		"versus.survivor_fall_deaths", "versus.survivor_car_alarms",
		"versus.survivor_friendly_fire_to_humans", "versus.survivor_incapacitations",
		"versus.objective_interactions", "versus.pills_used", "versus.adrenaline_used",
		"versus.incendiary_packs_deployed", "versus.explosive_packs_deployed",
		"versus.molotovs_thrown", "versus.pipe_bombs_thrown", "versus.vomit_jars_thrown",
	}
	versusValues := make([]sql.NullInt64, len(versusNames))
	versusDest := make([]any, len(versusValues))
	for i := range versusValues {
		versusDest[i] = &versusValues[i]
	}
	versusSQL := `SELECT SUM(v.human_special_kills),SUM(v.defib_revives),SUM(v.black_white_teammates_restored),
SUM(v.fall_deaths),SUM(v.car_alarms_triggered),SUM(v.friendly_fire_to_humans),SUM(v.incapacitations),
SUM(v.objective_interactions),SUM(v.pills_used),SUM(v.adrenaline_used),
SUM(v.incendiary_packs_deployed),SUM(v.explosive_packs_deployed),
SUM(v.molotovs_thrown),SUM(v.pipe_bombs_thrown),SUM(v.vomit_jars_thrown)
FROM lps_versus_survivor_stats v JOIN lps_player_segments s ON s.segment_id=v.segment_id
WHERE v.stats_version=1 AND s.steam_id=` + s.bind(1)
	if err := s.db.QueryRowContext(queryCtx, versusSQL, steamID).Scan(versusDest...); err != nil {
		return result, fmt.Errorf("read Versus survivor achievement metrics: %w", err)
	}
	for i, name := range versusNames {
		putAchievementMetric(result.Values, name, versusValues[i])
	}

	var infectedDamage sql.NullInt64
	infectedSQL := `SELECT SUM(v.damage_to_human_survivors) FROM lps_versus_infected_stats v
JOIN lps_player_segments s ON s.segment_id=v.segment_id WHERE v.stats_version=1 AND s.steam_id=` + s.bind(1)
	if err := s.db.QueryRowContext(queryCtx, infectedSQL, steamID).Scan(&infectedDamage); err != nil {
		return result, fmt.Errorf("read Versus infected achievement metrics: %w", err)
	}
	putAchievementMetric(result.Values, "versus.infected_damage_to_human_survivors", infectedDamage)

	if err := s.loadAchievementCompanion(queryCtx, steamID, result.Values); err != nil {
		return result, err
	}
	combineAchievementMetrics(result.Values, "pve.special_rescues", "pve.smoker_saves", "pve.hunter_saves", "pve.jockey_saves", "pve.charger_saves")
	combineAchievementMetrics(result.Values, "defib_revives", "pve.defib_revives", "versus.defib_revives")
	combineAchievementMetrics(result.Values, "black_white_restores", "pve.black_white_restores", "versus.black_white_restores")
	combineAchievementMetrics(result.Values, "survivor_fall_deaths", "pve.survivor_fall_deaths", "versus.survivor_fall_deaths")
	combineAchievementMetrics(result.Values, "survivor_car_alarms", "pve.survivor_car_alarms", "versus.survivor_car_alarms")
	combineAchievementMetrics(result.Values, "survivor_friendly_fire_to_humans", "pve.survivor_friendly_fire_to_humans", "versus.survivor_friendly_fire_to_humans")
	combineAchievementMetrics(result.Values, "survivor_incapacitations", "pve.survivor_incapacitations", "versus.survivor_incapacitations")
	combineAchievementMetrics(result.Values, "survivor.objective_interactions", "pve.objective_interactions", "versus.objective_interactions")
	combineAchievementMetrics(result.Values, "survivor.temp_health_items_used", "pve.pills_used", "pve.adrenaline_used", "versus.pills_used", "versus.adrenaline_used")
	combineAchievementMetrics(result.Values, "survivor.upgrade_packs_deployed", "pve.incendiary_packs_deployed", "pve.explosive_packs_deployed", "versus.incendiary_packs_deployed", "versus.explosive_packs_deployed")
	combineAchievementMetrics(result.Values, "versus.throwables_used", "versus.molotovs_thrown", "versus.pipe_bombs_thrown", "versus.vomit_jars_thrown")

	tankParts, tankKills := result.Values["pve.tank_kill_participations"], result.Values["pve.tank_kills"]
	witchParts, witchKills := result.Values["pve.witch_kill_participations"], result.Values["pve.witch_kills"]
	if tankParts.Available && tankKills.Available && witchParts.Available && witchKills.Available {
		if tankParts.Value < tankKills.Value || witchParts.Value < witchKills.Value {
			return result, fmt.Errorf("Boss participation invariant failed for player %s", steamID)
		}
		result.Values["pve.boss_assists"] = AchievementMetricValue{Available: true, Value: tankParts.Value - tankKills.Value + witchParts.Value - witchKills.Value}
	}
	return result, nil
}

func (s *statsStore) loadAchievementCompanion(ctx context.Context, steamID string, values map[string]AchievementMetricValue) error {
	args := []any{steamID}
	subjectEnd := "COALESCE(s.ended_at,s.last_saved_at)"
	peerEnd := "COALESCE(p.ended_at,p.last_saved_at)"
	latestStart := "CASE WHEN s.started_at>p.started_at THEN s.started_at ELSE p.started_at END"
	earliestEnd := "CASE WHEN " + subjectEnd + "<" + peerEnd + " THEN " + subjectEnd + " ELSE " + peerEnd + " END"
	overlap := "(" + earliestEnd + "-" + latestStart + ")"
	statement := `SELECT p.steam_id,COUNT(DISTINCT s.round_id),COALESCE(SUM(` + overlap + `),0)
FROM lps_player_segments s JOIN lps_player_segments p ON p.round_id=s.round_id AND p.side=s.side AND p.steam_id<>s.steam_id
WHERE s.steam_id=` + s.bind(1) + ` AND ` + earliestEnd + `>` + latestStart + `
GROUP BY p.steam_id ORDER BY 3 DESC,2 DESC,p.steam_id ASC LIMIT 1`
	var peer string
	var rounds, seconds int64
	err := s.db.QueryRowContext(ctx, statement, args...).Scan(&peer, &rounds, &seconds)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read achievement companion metric: %w", err)
	}
	values["relationship.max_peer_shared_seconds"] = AchievementMetricValue{Value: seconds, Available: true, EvidenceSteamID: peer, EvidenceRounds: rounds}
	return nil
}

func (s *statsStore) AchievementDirtyPlayers(ctx context.Context, afterWatermark int64, afterSteamID string, limit int32) ([]AchievementSourcePlayer, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	statement := `SELECT steam_id,MAX(watermark) player_watermark FROM (` + achievementMetricSource + `) source
GROUP BY steam_id HAVING MAX(watermark)>` + s.bind(1) + ` OR (MAX(watermark)=` + s.bind(2) + ` AND steam_id>` + s.bind(3) + `)
ORDER BY player_watermark,steam_id LIMIT ` + s.bind(4)
	rows, err := s.db.QueryContext(queryCtx, statement, afterWatermark, afterWatermark, afterSteamID, limit)
	if err != nil {
		return nil, fmt.Errorf("discover dirty achievement players: %w", err)
	}
	defer rows.Close()
	result := make([]AchievementSourcePlayer, 0, limit)
	for rows.Next() {
		var player AchievementSourcePlayer
		if err := rows.Scan(&player.SteamID, &player.Watermark); err != nil {
			return nil, err
		}
		result = append(result, player)
	}
	return result, rows.Err()
}

func (s *statsStore) AchievementBackfillPlayers(ctx context.Context, afterSteamID string, limit int32) ([]AchievementSourcePlayer, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	statement := `SELECT steam_id,MAX(watermark) FROM (` + achievementMetricSource + `) source
WHERE steam_id>` + s.bind(1) + ` GROUP BY steam_id ORDER BY steam_id LIMIT ` + s.bind(2)
	rows, err := s.db.QueryContext(queryCtx, statement, afterSteamID, limit)
	if err != nil {
		return nil, fmt.Errorf("list achievement backfill players: %w", err)
	}
	defer rows.Close()
	result := make([]AchievementSourcePlayer, 0, limit)
	for rows.Next() {
		var player AchievementSourcePlayer
		if err := rows.Scan(&player.SteamID, &player.Watermark); err != nil {
			return nil, err
		}
		result = append(result, player)
	}
	return result, rows.Err()
}

func (s *statsStore) AchievementEligiblePlayerCount(ctx context.Context) (int64, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	var count int64
	if err := s.db.QueryRowContext(queryCtx, `SELECT COUNT(DISTINCT steam_id) FROM lps_player_segments`).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func putAchievementMetric(values map[string]AchievementMetricValue, name string, value sql.NullInt64) {
	if value.Valid {
		values[name] = AchievementMetricValue{Value: value.Int64, Available: true}
	}
}

func combineAchievementMetrics(values map[string]AchievementMetricValue, target string, sources ...string) {
	var total int64
	available := false
	for _, source := range sources {
		if value, ok := values[source]; ok && value.Available {
			total += value.Value
			available = true
		}
	}
	if available {
		values[target] = AchievementMetricValue{Value: total, Available: true}
	}
}
