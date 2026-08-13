CREATE TABLE IF NOT EXISTS lps_round_contexts (
  round_id VARCHAR(128) PRIMARY KEY,
  context_version INTEGER NOT NULL,
  captured_at BIGINT NOT NULL,
  last_saved_at BIGINT NOT NULL,
  collector_version VARCHAR(32) NOT NULL DEFAULT '',
  ruleset_name VARCHAR(64) NOT NULL DEFAULT '',
  difficulty VARCHAR(16) NOT NULL DEFAULT '',
  survivor_limit INTEGER NOT NULL DEFAULT -1,
  max_player_zombies INTEGER NOT NULL DEFAULT -1,
  common_limit INTEGER NOT NULL DEFAULT -1,
  tank_health INTEGER NOT NULL DEFAULT -1,
  witch_health INTEGER NOT NULL DEFAULT -1,
  change_mask BIGINT NOT NULL DEFAULT 0,
  incident_capture_enabled INTEGER NOT NULL DEFAULT 0,
  incident_capture_complete INTEGER NOT NULL DEFAULT 0,
  incident_expected_count BIGINT NOT NULL DEFAULT 0,
  incident_dropped_count BIGINT NOT NULL DEFAULT 0,
  revision BIGINT NOT NULL DEFAULT 0
);
-- statement-breakpoint
CREATE INDEX lps_idx_round_contexts_saved
ON lps_round_contexts (last_saved_at, round_id);
-- statement-breakpoint
CREATE TABLE IF NOT EXISTS lps_incidents (
  round_id VARCHAR(128) NOT NULL,
  incident_seq BIGINT NOT NULL,
  incident_version INTEGER NOT NULL,
  incident_type INTEGER NOT NULL,
  occurred_at BIGINT NOT NULL,
  round_offset_ms BIGINT NOT NULL,
  duration_ms BIGINT NOT NULL DEFAULT 0,
  actor_kind INTEGER NOT NULL DEFAULT 0,
  actor_steam_id VARCHAR(32) NOT NULL DEFAULT '',
  target_kind INTEGER NOT NULL DEFAULT 0,
  target_steam_id VARCHAR(32) NOT NULL DEFAULT '',
  helper_kind INTEGER NOT NULL DEFAULT 0,
  helper_steam_id VARCHAR(32) NOT NULL DEFAULT '',
  infected_class INTEGER NOT NULL DEFAULT 0,
  end_reason INTEGER NOT NULL DEFAULT 0,
  detail_flags BIGINT NOT NULL DEFAULT 0,
  related_incident_seq BIGINT NOT NULL DEFAULT 0,
  pos_x INTEGER NULL,
  pos_y INTEGER NULL,
  pos_z INTEGER NULL,
  end_pos_x INTEGER NULL,
  end_pos_y INTEGER NULL,
  end_pos_z INTEGER NULL,
  PRIMARY KEY (round_id, incident_seq)
);
-- statement-breakpoint
CREATE INDEX lps_idx_incidents_occurred
ON lps_incidents (occurred_at, round_id, incident_seq);
-- statement-breakpoint
CREATE INDEX lps_idx_incidents_actor_time
ON lps_incidents (actor_steam_id, occurred_at);
-- statement-breakpoint
CREATE INDEX lps_idx_incidents_target_time
ON lps_incidents (target_steam_id, occurred_at);
-- statement-breakpoint
CREATE INDEX lps_idx_rounds_server_started_map
ON lps_rounds (server_key, started_at, map_name);
