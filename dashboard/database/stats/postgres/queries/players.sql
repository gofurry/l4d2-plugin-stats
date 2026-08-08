-- name: GetPlayerSummary :one
SELECT p.steam_id, p.last_name, p.first_seen_at, p.last_seen_at,
  (SELECT COUNT(*) FROM lps_sessions s WHERE s.steam_id=p.steam_id)::bigint AS session_count,
  (SELECT COALESCE(SUM(s.connected_seconds),0) FROM lps_sessions s WHERE s.steam_id=p.steam_id)::bigint AS connected_seconds,
  (SELECT COALESCE(SUM(s.active_play_seconds),0) FROM lps_sessions s WHERE s.steam_id=p.steam_id)::bigint AS active_play_seconds
FROM lps_players p WHERE p.steam_id=sqlc.arg(steam_id);

-- name: SearchPlayers :many
SELECT steam_id, last_name
FROM lps_players
WHERE sqlc.arg(search_query) = ''
   OR last_name ILIKE '%' || sqlc.arg(search_query) || '%'
   OR steam_id LIKE '%' || sqlc.arg(search_query) || '%'
ORDER BY last_seen_at DESC, steam_id ASC
LIMIT sqlc.arg(page_limit);

-- name: ListActivePlayersByServer :many
SELECT steam_id, player_name, started_at, last_saved_at, connected_seconds
FROM lps_sessions
WHERE server_key = sqlc.arg(server_key)
  AND status = 'active'
  AND ended_at IS NULL
  AND last_saved_at >= sqlc.arg(fresh_since)
ORDER BY started_at ASC, steam_id ASC;

-- name: GetPlayerPVE :one
SELECT
  COALESCE(SUM(p.common_kills),0)::bigint common_kills,
  COALESCE(SUM(p.special_kills),0)::bigint special_kills,
  COALESCE(SUM(p.tank_kills),0)::bigint tank_kills,
  COALESCE(SUM(p.witch_kills),0)::bigint witch_kills,
  COALESCE(SUM(p.damage_to_special),0)::bigint damage_to_special,
  COALESCE(SUM(p.damage_to_tank),0)::bigint damage_to_tank,
  COALESCE(SUM(p.damage_to_witch),0)::bigint damage_to_witch,
  COALESCE(SUM(p.damage_taken_infected),0)::bigint damage_taken,
  COALESCE(SUM(p.friendly_fire_to_humans+p.friendly_fire_to_bots),0)::bigint friendly_fire,
  COALESCE(SUM(p.incapacitations),0)::bigint incapacitations,
  COALESCE(SUM(p.deaths),0)::bigint deaths,
  COALESCE(SUM(p.incap_revives+p.ledge_rescues+p.defib_revives),0)::bigint revives,
  COALESCE(SUM(p.rescues_received),0)::bigint rescues_received,
  COALESCE(SUM(p.medkits_used_self+p.medkits_used_on_others),0)::bigint medkits_used,
  COALESCE(SUM(p.medkit_healing_self+p.medkit_healing_others),0)::bigint healing,
  COALESCE(SUM(p.chapter_participations),0)::bigint chapter_participations,
  COALESCE(SUM(p.chapter_completions_alive+p.chapter_completions_dead),0)::bigint chapter_completions,
  COALESCE(SUM(p.campaign_completions),0)::bigint campaign_completions
FROM lps_pve_segment_stats p
JOIN lps_player_segments s ON s.segment_id=p.segment_id JOIN lps_runs r ON r.run_id=s.run_id
WHERE s.steam_id=sqlc.arg(steam_id) AND s.side='survivor' AND p.stats_version=1
  AND r.mode_family='pve' AND r.game_mode IN ('coop','realism')
  AND (sqlc.arg(cutoff)::bigint=0 OR s.started_at>=sqlc.arg(cutoff)::bigint);

-- name: GetPlayerVersus :one
WITH survivor AS (
  SELECT COALESCE(SUM(v.common_kills),0)::bigint common_kills,
    COALESCE(SUM(v.human_special_kills),0)::bigint human_special_kills,
    COALESCE(SUM(v.bot_special_kills),0)::bigint bot_special_kills,
    COALESCE(SUM(v.human_tank_kills),0)::bigint human_tank_kills,
    COALESCE(SUM(v.bot_tank_kills),0)::bigint bot_tank_kills,
    COALESCE(SUM(v.damage_to_human_special+v.damage_to_bot_special+v.damage_to_human_tank+v.damage_to_bot_tank),0)::bigint survivor_damage,
    COALESCE(SUM(v.deaths),0)::bigint survivor_deaths,
    COALESCE(SUM(v.incap_revives+v.ledge_rescues+v.defib_revives),0)::bigint survivor_revives
  FROM lps_versus_survivor_stats v JOIN lps_player_segments s ON s.segment_id=v.segment_id JOIN lps_runs r ON r.run_id=s.run_id
  WHERE s.steam_id=sqlc.arg(steam_id) AND s.side='survivor' AND v.stats_version=1
    AND r.mode_family='versus' AND r.game_mode='versus' AND (sqlc.arg(cutoff)::bigint=0 OR s.started_at>=sqlc.arg(cutoff)::bigint)
), infected AS (
  SELECT COALESCE(SUM(v.spawn_count),0)::bigint infected_spawns,
    COALESCE(SUM(v.damage_to_human_survivors),0)::bigint damage_to_human_survivors,
    COALESCE(SUM(v.human_survivor_incaps),0)::bigint human_survivor_incaps,
    COALESCE(SUM(v.human_survivor_kills),0)::bigint human_survivor_kills
  FROM lps_versus_infected_stats v JOIN lps_player_segments s ON s.segment_id=v.segment_id JOIN lps_runs r ON r.run_id=s.run_id
  WHERE s.steam_id=sqlc.arg(steam_id) AND s.side='infected' AND v.stats_version=1
    AND r.mode_family='versus' AND r.game_mode='versus' AND (sqlc.arg(cutoff)::bigint=0 OR s.started_at>=sqlc.arg(cutoff)::bigint)
), controls AS (
  SELECT COALESCE(SUM(c.human_survivor_controls),0)::bigint human_survivor_controls,
    COALESCE(SUM(c.human_survivor_control_seconds),0)::bigint human_survivor_control_seconds
  FROM lps_versus_infected_class_stats c JOIN lps_player_segments s ON s.segment_id=c.segment_id JOIN lps_runs r ON r.run_id=s.run_id
  WHERE s.steam_id=sqlc.arg(steam_id) AND s.side='infected' AND c.stats_version=1
    AND r.mode_family='versus' AND r.game_mode='versus' AND (sqlc.arg(cutoff)::bigint=0 OR s.started_at>=sqlc.arg(cutoff)::bigint)
)
SELECT survivor.*, infected.*, controls.* FROM survivor CROSS JOIN infected CROSS JOIN controls;

-- name: ListPlayerSessions :many
SELECT session_id,server_key,player_name,started_at,ended_at,connected_seconds,active_play_seconds,status,disconnect_reason
FROM lps_sessions WHERE steam_id=sqlc.arg(steam_id)
  AND (sqlc.arg(cursor_started_at)::bigint=0 OR started_at<sqlc.arg(cursor_started_at)::bigint
    OR (started_at=sqlc.arg(cursor_started_at)::bigint AND session_id<sqlc.arg(cursor_id)))
ORDER BY started_at DESC,session_id DESC LIMIT sqlc.arg(page_limit);

-- name: ListPlayerChapters :many
SELECT s.segment_id,s.server_key,r.mode_family,r.game_mode,rd.map_name,s.side,s.started_at,s.ended_at,s.active_play_seconds,s.status
FROM lps_player_segments s JOIN lps_runs r ON r.run_id=s.run_id JOIN lps_rounds rd ON rd.round_id=s.round_id
WHERE s.steam_id=sqlc.arg(steam_id)
  AND (sqlc.arg(cursor_started_at)::bigint=0 OR s.started_at<sqlc.arg(cursor_started_at)::bigint
    OR (s.started_at=sqlc.arg(cursor_started_at)::bigint AND s.segment_id<sqlc.arg(cursor_id)))
ORDER BY s.started_at DESC,s.segment_id DESC LIMIT sqlc.arg(page_limit);
