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
	}
	for _, check := range checks {
		*check.target, err = s.dataQualityFinding(ctx, check.query)
		if err != nil {
			return result, fmt.Errorf("check %s: %w", check.name, err)
		}
	}
	return result, nil
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
