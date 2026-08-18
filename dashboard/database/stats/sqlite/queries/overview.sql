-- name: GetSchemaVersion :one
SELECT CAST(COALESCE(MAX(version), 0) AS INTEGER) AS version FROM lps_schema_migrations;

-- name: GetCoreOverview :one
SELECT
  (SELECT COUNT(*) FROM lps_players) AS total_players,
  (SELECT COUNT(DISTINCT steam_id)
     FROM lps_sessions
    WHERE lps_sessions.last_saved_at >= ?1
      AND lps_sessions.active_play_seconds > 0) AS active_players_7d,
  (SELECT CAST(COALESCE(SUM(active_play_seconds), 0) AS INTEGER) FROM lps_sessions) AS total_active_play_seconds,
  (SELECT COUNT(*) FROM lps_runs
    WHERE mode_family = 'pve' AND game_mode IN ('coop', 'realism') AND status = 'completed') AS completed_pve_runs,
  (SELECT COUNT(*) FROM lps_runs
    WHERE mode_family = 'versus' AND game_mode = 'versus' AND status = 'completed') AS completed_versus_runs;

-- name: GetPVEOverview :one
SELECT
  CAST(COALESCE(SUM(p.common_kills), 0) AS INTEGER) AS common_kills,
  CAST(COALESCE(SUM(p.special_kills), 0) AS INTEGER) AS special_kills,
  CAST(COALESCE(SUM(p.tank_kills), 0) AS INTEGER) AS tank_kills,
  CAST(COALESCE(SUM(p.witch_kills), 0) AS INTEGER) AS witch_kills,
  CAST(COALESCE(SUM(p.incap_revives + p.ledge_rescues + p.defib_revives), 0) AS INTEGER) AS rescues
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
      AND r.mode_family = 'versus' AND r.game_mode = 'versus') AS completed_matches,
  (SELECT COUNT(*)
     FROM lps_versus_round_results rr
     JOIN lps_rounds r ON r.round_id = rr.round_id
    WHERE rr.stats_version = 1 AND rr.result_status = 'completed'
      AND r.mode_family = 'versus') AS completed_halves,
  (SELECT CAST(COALESCE(SUM(v.human_special_kills + v.human_tank_kills), 0) AS INTEGER)
     FROM lps_versus_survivor_stats v
     JOIN lps_player_segments s ON s.segment_id = v.segment_id
     JOIN lps_runs r ON r.run_id = s.run_id
    WHERE v.stats_version = 1 AND s.side = 'survivor'
      AND r.mode_family = 'versus' AND r.game_mode = 'versus') AS human_controlled_infected_kills,
  (SELECT CAST(COALESCE(SUM(c.human_survivor_controls), 0) AS INTEGER)
     FROM lps_versus_infected_class_stats c
     JOIN lps_player_segments s ON s.segment_id = c.segment_id
     JOIN lps_runs r ON r.run_id = s.run_id
    WHERE c.stats_version = 1 AND s.side = 'infected'
      AND r.mode_family = 'versus' AND r.game_mode = 'versus') AS human_survivor_controls;

-- name: GetServerRecent24h :one
SELECT
  (SELECT CAST(COUNT(DISTINCT steam_id) AS INTEGER)
     FROM lps_sessions
    WHERE lps_sessions.server_key = sqlc.arg(filter_server_key)
      AND lps_sessions.last_saved_at >= sqlc.arg(cutoff_at)
      AND lps_sessions.active_play_seconds > 0) AS active_players,
  (SELECT CAST(COALESCE(SUM(p.common_kills), 0) AS INTEGER)
     FROM lps_pve_segment_stats p
     JOIN lps_player_segments s ON s.segment_id = p.segment_id
     JOIN lps_runs r ON r.run_id = s.run_id
    WHERE s.server_key = sqlc.arg(filter_server_key)
      AND p.last_saved_at >= sqlc.arg(cutoff_at)
      AND p.stats_version = 1
      AND r.mode_family = 'pve'
      AND r.game_mode IN ('coop', 'realism')) AS common_kills,
  (SELECT CAST(COALESCE(SUM(p.special_kills), 0) AS INTEGER)
     FROM lps_pve_segment_stats p
     JOIN lps_player_segments s ON s.segment_id = p.segment_id
     JOIN lps_runs r ON r.run_id = s.run_id
    WHERE s.server_key = sqlc.arg(filter_server_key)
      AND p.last_saved_at >= sqlc.arg(cutoff_at)
      AND p.stats_version = 1
      AND r.mode_family = 'pve'
      AND r.game_mode IN ('coop', 'realism')) AS special_kills,
  (SELECT CAST(COUNT(*) AS INTEGER)
     FROM lps_runs
    WHERE lps_runs.server_key = sqlc.arg(filter_server_key)
      AND lps_runs.mode_family = 'pve'
      AND lps_runs.game_mode IN ('coop', 'realism')
      AND lps_runs.status = 'completed'
      AND COALESCE(lps_runs.ended_at, lps_runs.last_saved_at) >= sqlc.arg(cutoff_at)) AS completed_runs;
