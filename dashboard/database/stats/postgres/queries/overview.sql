-- name: GetSchemaVersion :one
SELECT COALESCE(MAX(version), 0)::bigint AS version FROM lps_schema_migrations;

-- name: GetCoreOverview :one
SELECT
  (SELECT COUNT(*) FROM lps_players)::bigint AS total_players,
  (SELECT COUNT(DISTINCT steam_id)
     FROM lps_sessions
    WHERE lps_sessions.last_saved_at >= $1
      AND lps_sessions.active_play_seconds > 0)::bigint AS active_players_7d,
  (SELECT COALESCE(SUM(active_play_seconds), 0) FROM lps_sessions)::bigint AS total_active_play_seconds,
  (SELECT COUNT(*) FROM lps_runs
    WHERE mode_family = 'pve' AND game_mode IN ('coop', 'realism') AND status = 'completed')::bigint AS completed_pve_runs,
  (SELECT COUNT(*) FROM lps_runs
    WHERE mode_family = 'versus' AND game_mode = 'versus' AND status = 'completed')::bigint AS completed_versus_runs;

-- name: GetPVEOverview :one
SELECT
  COALESCE(SUM(p.common_kills), 0)::bigint AS common_kills,
  COALESCE(SUM(p.special_kills), 0)::bigint AS special_kills,
  COALESCE(SUM(p.tank_kills), 0)::bigint AS tank_kills,
  COALESCE(SUM(p.witch_kills), 0)::bigint AS witch_kills,
  COALESCE(SUM(p.incap_revives + p.ledge_rescues + p.defib_revives), 0)::bigint AS rescues
FROM lps_pve_segment_stats p
JOIN lps_player_segments s ON s.segment_id = p.segment_id
JOIN lps_runs r ON r.run_id = s.run_id
WHERE p.stats_version = 1
  AND r.mode_family = 'pve'
  AND r.game_mode IN ('coop', 'realism');

-- name: GetVersusOverview :one
SELECT
  (SELECT COUNT(*)
     FROM lps_versus_run_results vr
     JOIN lps_runs r ON r.run_id = vr.run_id
    WHERE vr.stats_version = 1 AND vr.result_status = 'completed'
      AND r.mode_family = 'versus' AND r.game_mode = 'versus')::bigint AS completed_matches,
  (SELECT COUNT(*)
     FROM lps_versus_round_results rr
     JOIN lps_rounds r ON r.round_id = rr.round_id
    WHERE rr.stats_version = 1 AND rr.result_status = 'completed'
      AND r.mode_family = 'versus')::bigint AS completed_halves,
  (SELECT COALESCE(SUM(v.human_special_kills + v.human_tank_kills), 0)
     FROM lps_versus_survivor_stats v
     JOIN lps_player_segments s ON s.segment_id = v.segment_id
     JOIN lps_runs r ON r.run_id = s.run_id
    WHERE v.stats_version = 1 AND s.side = 'survivor'
      AND r.mode_family = 'versus' AND r.game_mode = 'versus')::bigint AS human_controlled_infected_kills,
  (SELECT COALESCE(SUM(c.human_survivor_controls), 0)
     FROM lps_versus_infected_class_stats c
     JOIN lps_player_segments s ON s.segment_id = c.segment_id
     JOIN lps_runs r ON r.run_id = s.run_id
    WHERE c.stats_version = 1 AND s.side = 'infected'
      AND r.mode_family = 'versus' AND r.game_mode = 'versus')::bigint AS human_survivor_controls;
