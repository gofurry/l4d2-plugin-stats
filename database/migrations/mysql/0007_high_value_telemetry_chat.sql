ALTER TABLE lps_pve_segment_stats ADD COLUMN teammate_protections BIGINT NULL;
-- statement-breakpoint
ALTER TABLE lps_pve_segment_stats ADD COLUMN ledge_grabs BIGINT NULL;
-- statement-breakpoint
ALTER TABLE lps_pve_segment_stats ADD COLUMN tank_rock_hits_received BIGINT NULL;
-- statement-breakpoint
ALTER TABLE lps_pve_segment_stats ADD COLUMN hunter_skeets BIGINT NULL;
-- statement-breakpoint
ALTER TABLE lps_pve_segment_stats ADD COLUMN charger_levels BIGINT NULL;
-- statement-breakpoint
ALTER TABLE lps_versus_survivor_stats ADD COLUMN teammate_protections BIGINT NULL;
-- statement-breakpoint
ALTER TABLE lps_versus_survivor_stats ADD COLUMN ledge_grabs BIGINT NULL;
-- statement-breakpoint
ALTER TABLE lps_versus_survivor_stats ADD COLUMN tank_rock_hits_received BIGINT NULL;
-- statement-breakpoint
ALTER TABLE lps_versus_survivor_stats ADD COLUMN hunter_skeets BIGINT NULL;
-- statement-breakpoint
ALTER TABLE lps_versus_survivor_stats ADD COLUMN charger_levels BIGINT NULL;
-- statement-breakpoint
CREATE TABLE lps_chat_outbox (
  message_id VARCHAR(192) NOT NULL PRIMARY KEY,
  boot_id VARCHAR(128) NOT NULL,
  server_key VARCHAR(65) NOT NULL,
  chat_seq BIGINT NOT NULL,
  session_id VARCHAR(128) NULL,
  steam_id VARCHAR(32) NULL,
  source_user_id INTEGER NOT NULL,
  player_name VARCHAR(128) NOT NULL,
  occurred_at BIGINT NOT NULL,
  map_name VARCHAR(128) NOT NULL,
  game_mode VARCHAR(32) NOT NULL,
  team VARCHAR(16) NOT NULL,
  channel VARCHAR(16) NOT NULL,
  alive INTEGER NOT NULL,
  command_like INTEGER NOT NULL,
  content VARCHAR(512) NOT NULL,
  CONSTRAINT lps_chat_outbox_sequence_positive CHECK (chat_seq > 0),
  CONSTRAINT lps_chat_outbox_alive_bool CHECK (alive IN (0, 1)),
  CONSTRAINT lps_chat_outbox_command_bool CHECK (command_like IN (0, 1)),
  CONSTRAINT lps_chat_outbox_channel CHECK (channel IN ('global', 'team')),
  UNIQUE KEY lps_uq_chat_outbox_boot_seq (boot_id, chat_seq),
  KEY lps_idx_chat_outbox_occurred (occurred_at, message_id),
  KEY lps_idx_chat_outbox_server (server_key, occurred_at),
  KEY lps_idx_chat_outbox_steam (steam_id, occurred_at)
);
-- statement-breakpoint
CREATE TABLE lps_chat_capture_state (
  boot_id VARCHAR(128) NOT NULL PRIMARY KEY,
  server_key VARCHAR(65) NOT NULL,
  capture_version INTEGER NOT NULL,
  capture_enabled INTEGER NOT NULL,
  started_at BIGINT NOT NULL,
  ended_at BIGINT NULL,
  last_saved_at BIGINT NOT NULL,
  observed_count BIGINT NOT NULL DEFAULT 0,
  persisted_count BIGINT NOT NULL DEFAULT 0,
  dropped_count BIGINT NOT NULL DEFAULT 0,
  last_chat_seq BIGINT NOT NULL DEFAULT 0,
  oldest_retained_seq BIGINT NOT NULL DEFAULT 0,
  revision BIGINT NOT NULL DEFAULT 0,
  CONSTRAINT lps_chat_capture_version CHECK (capture_version = 1),
  CONSTRAINT lps_chat_capture_enabled_bool CHECK (capture_enabled IN (0, 1)),
  CONSTRAINT lps_chat_capture_counts CHECK (observed_count >= 0 AND persisted_count >= 0 AND dropped_count >= 0),
  CONSTRAINT lps_chat_capture_sequences CHECK (last_chat_seq >= 0 AND oldest_retained_seq >= 0 AND revision >= 0)
);
