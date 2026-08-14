package store

import (
	"context"
	"fmt"
)

const doctorSampleLimit = 5

func (s *statsStore) DeepDataQuality(ctx context.Context, staleBootBefore int64) (StatsDataQuality, error) {
	result := StatsDataQuality{}
	var err error
	if result.SourceWatermark, err = s.sourceWatermark(ctx); err != nil {
		return result, fmt.Errorf("read source watermark: %w", err)
	}
	checks := []struct {
		name   string
		query  string
		target *DataQualityFinding
	}{
		{"stale active boots", fmt.Sprintf(`SELECT 'boot' AS source_name, boot_id AS internal_id FROM lps_server_boots WHERE status='active' AND last_heartbeat_at < %d`, staleBootBefore), &result.StaleActiveBoots},
		{"unknown stats versions", unknownStatsVersionQuery(), &result.UnknownStatsVersion},
		{"lifecycle links", lifecycleLinkQuery(), &result.LifecycleLinks},
		{"mode and side", modeSideQuery(), &result.ModeSideMismatch},
		{"PvE totals", pveTotalQuery(), &result.PVETotalMismatch},
		{"round context contract", contextContractQuery(), &result.ContextContract},
		{"incident contract", incidentContractQuery(), &result.IncidentContract},
		{"incident completeness", incidentCompletenessQuery(), &result.IncidentCompleteness},
		{"relationship contract", relationshipContractQuery(), &result.RelationshipContract},
		{"PvE assist contract", pveAssistContractQuery(), &result.PVEAssistContract},
		{"Versus assist contract", versusAssistContractQuery(), &result.VersusAssistContract},
	}
	for _, check := range checks {
		*check.target, err = s.dataQualityFinding(ctx, check.query)
		if err != nil {
			return result, fmt.Errorf("check %s: %w", check.name, err)
		}
	}
	return result, nil
}

func contextContractQuery() string {
	return `SELECT 'context' AS source_name, round_id AS internal_id FROM lps_round_contexts
WHERE context_version<>1 OR incident_expected_count<0 OR incident_dropped_count<0
OR incident_dropped_count>incident_expected_count
OR (incident_capture_complete=1 AND (incident_capture_enabled<>1 OR incident_dropped_count<>0))`
}

func incidentContractQuery() string {
	return `SELECT 'incident' AS source_name, i.round_id AS internal_id FROM lps_incidents i
WHERE i.incident_version<>1 OR i.incident_type NOT BETWEEN 1 AND 14
OR i.actor_kind NOT BETWEEN 0 AND 9 OR i.target_kind NOT BETWEEN 0 AND 9 OR i.helper_kind NOT BETWEEN 0 AND 9
OR i.round_offset_ms<0 OR i.duration_ms<0 OR i.end_reason NOT BETWEEN 0 AND 6
OR (i.incident_type=11 AND (i.detail_flags<0 OR i.detail_flags>1)) OR (i.incident_type<>11 AND i.detail_flags<>0)
OR (i.related_incident_seq<>0 AND NOT EXISTS (
  SELECT 1 FROM lps_incidents spawn WHERE spawn.round_id=i.round_id AND spawn.incident_seq=i.related_incident_seq
  AND ((i.incident_type=9 AND spawn.incident_type=8) OR (i.incident_type IN (11,12) AND spawn.incident_type=10))
))`
}

func relationshipContractQuery() string {
	return `SELECT 'relationship' AS source_name, rel.round_id AS internal_id
FROM lps_player_round_relationship_stats rel
LEFT JOIN lps_rounds r ON r.round_id=rel.round_id
LEFT JOIN lps_players actor ON actor.steam_id=rel.actor_steam_id
LEFT JOIN lps_players target ON target.steam_id=rel.target_steam_id
WHERE rel.relationship_version<>1 OR rel.actor_steam_id=rel.target_steam_id
OR r.round_id IS NULL OR actor.steam_id IS NULL OR target.steam_id IS NULL
OR rel.incap_revives<0 OR rel.ledge_rescues<0 OR rel.defib_revives<0
OR rel.smoker_rescues<0 OR rel.hunter_rescues<0 OR rel.jockey_rescues<0 OR rel.charger_rescues<0
OR rel.control_rescue_duration_ms<0 OR rel.medkits_used<0 OR rel.medkit_healing<0
OR rel.black_white_restores<0 OR rel.friendly_fire_damage<0
OR rel.incap_revives+rel.ledge_rescues+rel.defib_revives+rel.smoker_rescues+rel.hunter_rescues+rel.jockey_rescues+rel.charger_rescues+rel.control_rescue_duration_ms+rel.medkits_used+rel.medkit_healing+rel.black_white_restores+rel.friendly_fire_damage<=0
OR NOT EXISTS(SELECT 1 FROM lps_player_segments s WHERE s.round_id=rel.round_id AND s.steam_id=rel.actor_steam_id AND s.side='survivor')
OR NOT EXISTS(SELECT 1 FROM lps_player_segments s WHERE s.round_id=rel.round_id AND s.steam_id=rel.target_steam_id AND s.side='survivor')`
}

func pveAssistContractQuery() string {
	return `SELECT 'pve_assist' AS source_name, segment_id AS internal_id FROM lps_pve_segment_stats
WHERE stats_version=1 AND (
  tank_kill_participations<tank_kills OR witch_kill_participations<witch_kills
  OR (CASE WHEN special_assists IS NULL THEN 1 ELSE 0 END+CASE WHEN smoker_assists IS NULL THEN 1 ELSE 0 END+CASE WHEN boomer_assists IS NULL THEN 1 ELSE 0 END+CASE WHEN hunter_assists IS NULL THEN 1 ELSE 0 END+CASE WHEN spitter_assists IS NULL THEN 1 ELSE 0 END+CASE WHEN jockey_assists IS NULL THEN 1 ELSE 0 END+CASE WHEN charger_assists IS NULL THEN 1 ELSE 0 END) NOT IN (0,7)
  OR (special_assists IS NOT NULL AND (special_assists<0 OR smoker_assists<0 OR boomer_assists<0 OR hunter_assists<0 OR spitter_assists<0 OR jockey_assists<0 OR charger_assists<0
    OR special_assists<>smoker_assists+boomer_assists+hunter_assists+spitter_assists+jockey_assists+charger_assists))
)`
}

func versusAssistContractQuery() string {
	return `SELECT 'versus_assist' AS source_name, s.segment_id AS internal_id
FROM lps_versus_survivor_stats s
LEFT JOIN (
  SELECT segment_id,
  COALESCE(SUM(CASE WHEN infected_class BETWEEN 1 AND 6 THEN human_controller_assists ELSE 0 END),0) human_special,
  COALESCE(SUM(CASE WHEN infected_class BETWEEN 1 AND 6 THEN bot_controller_assists ELSE 0 END),0) bot_special,
  COALESCE(SUM(CASE WHEN infected_class=8 THEN human_controller_assists ELSE 0 END),0) human_tank,
  COALESCE(SUM(CASE WHEN infected_class=8 THEN bot_controller_assists ELSE 0 END),0) bot_tank
  FROM lps_versus_survivor_infected_class_stats GROUP BY segment_id
) c ON c.segment_id=s.segment_id
WHERE s.stats_version=1 AND (
  (CASE WHEN s.human_special_assists IS NULL THEN 1 ELSE 0 END+CASE WHEN s.bot_special_assists IS NULL THEN 1 ELSE 0 END+CASE WHEN s.human_tank_assists IS NULL THEN 1 ELSE 0 END+CASE WHEN s.bot_tank_assists IS NULL THEN 1 ELSE 0 END+CASE WHEN s.witch_encounters IS NULL THEN 1 ELSE 0 END+CASE WHEN s.witch_kill_participations IS NULL THEN 1 ELSE 0 END+CASE WHEN s.black_white_teammates_restored IS NULL THEN 1 ELSE 0 END) NOT IN (0,7)
  OR (s.human_special_assists IS NOT NULL AND (s.human_special_assists<0 OR s.bot_special_assists<0 OR s.human_tank_assists<0 OR s.bot_tank_assists<0 OR s.witch_encounters<0 OR s.witch_kill_participations<0 OR s.black_white_teammates_restored<0
    OR s.witch_kill_participations<s.witch_kills
    OR s.human_special_assists<>COALESCE(c.human_special,0) OR s.bot_special_assists<>COALESCE(c.bot_special,0)
    OR s.human_tank_assists<>COALESCE(c.human_tank,0) OR s.bot_tank_assists<>COALESCE(c.bot_tank,0)))
)
UNION ALL SELECT 'versus_class_assist', segment_id FROM lps_versus_survivor_infected_class_stats
WHERE (CASE WHEN human_controller_assists IS NULL THEN 1 ELSE 0 END+CASE WHEN bot_controller_assists IS NULL THEN 1 ELSE 0 END) NOT IN (0,2)
OR (human_controller_assists IS NOT NULL AND (human_controller_assists<0 OR bot_controller_assists<0))`
}

func incidentCompletenessQuery() string {
	return `SELECT 'context_count' AS source_name, c.round_id AS internal_id
FROM lps_round_contexts c LEFT JOIN lps_incidents i ON i.round_id=c.round_id AND i.incident_version=1
WHERE c.context_version=1 AND c.incident_capture_complete=1
GROUP BY c.round_id,c.incident_expected_count HAVING COUNT(i.incident_seq)<>c.incident_expected_count`
}

func (s *statsStore) dataQualityFinding(ctx context.Context, query string) (DataQualityFinding, error) {
	var result DataQualityFinding
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM (`+query+`) quality_rows`).Scan(&result.Count); err != nil {
		return result, err
	}
	if result.Count == 0 {
		result.IDs = []string{}
		return result, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT source_name, internal_id FROM (`+query+`) quality_rows ORDER BY source_name, internal_id LIMIT 5`)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	result.IDs = make([]string, 0, doctorSampleLimit)
	for rows.Next() {
		var source, id any
		if err := rows.Scan(&source, &id); err != nil {
			return result, err
		}
		result.IDs = append(result.IDs, stringValue(source)+":"+stringValue(id))
	}
	return result, rows.Err()
}

func unknownStatsVersionQuery() string {
	return `SELECT 'pve' AS source_name, segment_id AS internal_id FROM lps_pve_segment_stats WHERE stats_version <> 1
UNION ALL SELECT 'pve_equipment', segment_id FROM lps_pve_segment_equipment_stats WHERE stats_version <> 1
UNION ALL SELECT 'versus_survivor', segment_id FROM lps_versus_survivor_stats WHERE stats_version <> 1
UNION ALL SELECT 'versus_survivor_class', segment_id FROM lps_versus_survivor_infected_class_stats WHERE stats_version <> 1
UNION ALL SELECT 'versus_infected', segment_id FROM lps_versus_infected_stats WHERE stats_version <> 1
UNION ALL SELECT 'versus_infected_class', segment_id FROM lps_versus_infected_class_stats WHERE stats_version <> 1
UNION ALL SELECT 'versus_round_result', round_id FROM lps_versus_round_results WHERE stats_version <> 1
UNION ALL SELECT 'versus_run_result', run_id FROM lps_versus_run_results WHERE stats_version <> 1`
}

func lifecycleLinkQuery() string {
	return `SELECT 'session' AS source_name, s.session_id AS internal_id
FROM lps_sessions s
LEFT JOIN lps_server_boots b ON b.boot_id=s.boot_id
LEFT JOIN lps_servers v ON v.server_key=s.server_key
LEFT JOIN lps_players p ON p.steam_id=s.steam_id
WHERE b.boot_id IS NULL OR v.server_key IS NULL OR p.steam_id IS NULL OR b.server_key <> s.server_key
UNION ALL SELECT 'run', r.run_id
FROM lps_runs r
LEFT JOIN lps_server_boots b ON b.boot_id=r.boot_id
LEFT JOIN lps_servers v ON v.server_key=r.server_key
WHERE b.boot_id IS NULL OR v.server_key IS NULL OR b.server_key <> r.server_key
UNION ALL SELECT 'round', rd.round_id
FROM lps_rounds rd
LEFT JOIN lps_runs r ON r.run_id=rd.run_id
LEFT JOIN lps_servers v ON v.server_key=rd.server_key
WHERE r.run_id IS NULL OR v.server_key IS NULL OR r.server_key <> rd.server_key OR r.mode_family <> rd.mode_family
UNION ALL SELECT 'segment', s.segment_id
FROM lps_player_segments s
LEFT JOIN lps_sessions se ON se.session_id=s.session_id
LEFT JOIN lps_runs r ON r.run_id=s.run_id
LEFT JOIN lps_rounds rd ON rd.round_id=s.round_id
LEFT JOIN lps_players p ON p.steam_id=s.steam_id
WHERE se.session_id IS NULL OR r.run_id IS NULL OR rd.round_id IS NULL OR p.steam_id IS NULL
   OR se.steam_id <> s.steam_id OR se.server_key <> s.server_key OR r.server_key <> s.server_key
   OR rd.server_key <> s.server_key OR rd.run_id <> s.run_id
UNION ALL SELECT 'pve', p.segment_id FROM lps_pve_segment_stats p LEFT JOIN lps_player_segments s ON s.segment_id=p.segment_id WHERE s.segment_id IS NULL
UNION ALL SELECT 'pve_equipment', p.segment_id FROM lps_pve_segment_equipment_stats p LEFT JOIN lps_player_segments s ON s.segment_id=p.segment_id WHERE s.segment_id IS NULL
UNION ALL SELECT 'versus_survivor', p.segment_id FROM lps_versus_survivor_stats p LEFT JOIN lps_player_segments s ON s.segment_id=p.segment_id WHERE s.segment_id IS NULL
UNION ALL SELECT 'versus_survivor_class', p.segment_id FROM lps_versus_survivor_infected_class_stats p LEFT JOIN lps_player_segments s ON s.segment_id=p.segment_id WHERE s.segment_id IS NULL
UNION ALL SELECT 'versus_infected', p.segment_id FROM lps_versus_infected_stats p LEFT JOIN lps_player_segments s ON s.segment_id=p.segment_id WHERE s.segment_id IS NULL
UNION ALL SELECT 'versus_infected_class', p.segment_id FROM lps_versus_infected_class_stats p LEFT JOIN lps_player_segments s ON s.segment_id=p.segment_id WHERE s.segment_id IS NULL
UNION ALL SELECT 'versus_round_result', p.round_id FROM lps_versus_round_results p LEFT JOIN lps_rounds rd ON rd.round_id=p.round_id WHERE rd.round_id IS NULL
UNION ALL SELECT 'versus_run_result', p.run_id FROM lps_versus_run_results p LEFT JOIN lps_runs r ON r.run_id=p.run_id WHERE r.run_id IS NULL`
}

func modeSideQuery() string {
	return `SELECT 'pve' AS source_name, p.segment_id AS internal_id
FROM lps_pve_segment_stats p JOIN lps_player_segments s ON s.segment_id=p.segment_id JOIN lps_runs r ON r.run_id=s.run_id
WHERE s.side <> 'survivor' OR r.mode_family <> 'pve' OR r.game_mode NOT IN ('coop','realism')
UNION ALL SELECT 'pve_equipment', p.segment_id
FROM lps_pve_segment_equipment_stats p JOIN lps_player_segments s ON s.segment_id=p.segment_id JOIN lps_runs r ON r.run_id=s.run_id
WHERE s.side <> 'survivor' OR r.mode_family <> 'pve' OR r.game_mode NOT IN ('coop','realism')
UNION ALL SELECT 'versus_survivor', p.segment_id
FROM lps_versus_survivor_stats p JOIN lps_player_segments s ON s.segment_id=p.segment_id JOIN lps_runs r ON r.run_id=s.run_id
WHERE s.side <> 'survivor' OR r.mode_family <> 'versus' OR r.game_mode <> 'versus'
UNION ALL SELECT 'versus_survivor_class', p.segment_id
FROM lps_versus_survivor_infected_class_stats p JOIN lps_player_segments s ON s.segment_id=p.segment_id JOIN lps_runs r ON r.run_id=s.run_id
WHERE s.side <> 'survivor' OR r.mode_family <> 'versus' OR r.game_mode <> 'versus'
UNION ALL SELECT 'versus_infected', p.segment_id
FROM lps_versus_infected_stats p JOIN lps_player_segments s ON s.segment_id=p.segment_id JOIN lps_runs r ON r.run_id=s.run_id
WHERE s.side <> 'infected' OR r.mode_family <> 'versus' OR r.game_mode <> 'versus'
UNION ALL SELECT 'versus_infected_class', p.segment_id
FROM lps_versus_infected_class_stats p JOIN lps_player_segments s ON s.segment_id=p.segment_id JOIN lps_runs r ON r.run_id=s.run_id
WHERE s.side <> 'infected' OR r.mode_family <> 'versus' OR r.game_mode <> 'versus'
UNION ALL SELECT 'versus_round_result', p.round_id
FROM lps_versus_round_results p JOIN lps_rounds rd ON rd.round_id=p.round_id JOIN lps_runs r ON r.run_id=rd.run_id
WHERE rd.mode_family <> 'versus' OR r.mode_family <> 'versus' OR r.game_mode <> 'versus'
UNION ALL SELECT 'versus_run_result', p.run_id
FROM lps_versus_run_results p JOIN lps_runs r ON r.run_id=p.run_id
WHERE r.mode_family <> 'versus' OR r.game_mode <> 'versus'`
}

func pveTotalQuery() string {
	return `SELECT 'pve' AS source_name, segment_id AS internal_id
FROM lps_pve_segment_stats
WHERE stats_version=1 AND (
  special_kills <> smoker_kills + boomer_kills + hunter_kills + spitter_kills + jockey_kills + charger_kills
  OR damage_to_special <> damage_to_smoker + damage_to_boomer + damage_to_hunter + damage_to_spitter + damage_to_jockey + damage_to_charger
)`
}
