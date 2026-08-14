ALTER TABLE lps_pve_segment_stats ADD COLUMN special_assists BIGINT NULL;
-- statement-breakpoint
ALTER TABLE lps_pve_segment_stats ADD COLUMN smoker_assists BIGINT NULL;
-- statement-breakpoint
ALTER TABLE lps_pve_segment_stats ADD COLUMN boomer_assists BIGINT NULL;
-- statement-breakpoint
ALTER TABLE lps_pve_segment_stats ADD COLUMN hunter_assists BIGINT NULL;
-- statement-breakpoint
ALTER TABLE lps_pve_segment_stats ADD COLUMN spitter_assists BIGINT NULL;
-- statement-breakpoint
ALTER TABLE lps_pve_segment_stats ADD COLUMN jockey_assists BIGINT NULL;
-- statement-breakpoint
ALTER TABLE lps_pve_segment_stats ADD COLUMN charger_assists BIGINT NULL;
-- statement-breakpoint
ALTER TABLE lps_versus_survivor_stats ADD COLUMN human_special_assists BIGINT NULL;
-- statement-breakpoint
ALTER TABLE lps_versus_survivor_stats ADD COLUMN bot_special_assists BIGINT NULL;
-- statement-breakpoint
ALTER TABLE lps_versus_survivor_stats ADD COLUMN human_tank_assists BIGINT NULL;
-- statement-breakpoint
ALTER TABLE lps_versus_survivor_stats ADD COLUMN bot_tank_assists BIGINT NULL;
-- statement-breakpoint
ALTER TABLE lps_versus_survivor_stats ADD COLUMN witch_encounters BIGINT NULL;
-- statement-breakpoint
ALTER TABLE lps_versus_survivor_stats ADD COLUMN witch_kill_participations BIGINT NULL;
-- statement-breakpoint
ALTER TABLE lps_versus_survivor_stats ADD COLUMN black_white_teammates_restored BIGINT NULL;
-- statement-breakpoint
ALTER TABLE lps_versus_survivor_infected_class_stats ADD COLUMN human_controller_assists BIGINT NULL;
-- statement-breakpoint
ALTER TABLE lps_versus_survivor_infected_class_stats ADD COLUMN bot_controller_assists BIGINT NULL;
-- statement-breakpoint
CREATE TABLE lps_player_round_relationship_stats (
  round_id VARCHAR(128) NOT NULL,
  actor_steam_id VARCHAR(32) NOT NULL,
  target_steam_id VARCHAR(32) NOT NULL,
  relationship_version INTEGER NOT NULL,
  incap_revives BIGINT NOT NULL DEFAULT 0,
  ledge_rescues BIGINT NOT NULL DEFAULT 0,
  defib_revives BIGINT NOT NULL DEFAULT 0,
  smoker_rescues BIGINT NOT NULL DEFAULT 0,
  hunter_rescues BIGINT NOT NULL DEFAULT 0,
  jockey_rescues BIGINT NOT NULL DEFAULT 0,
  charger_rescues BIGINT NOT NULL DEFAULT 0,
  control_rescue_duration_ms BIGINT NOT NULL DEFAULT 0,
  medkits_used BIGINT NOT NULL DEFAULT 0,
  medkit_healing BIGINT NOT NULL DEFAULT 0,
  black_white_restores BIGINT NOT NULL DEFAULT 0,
  friendly_fire_damage BIGINT NOT NULL DEFAULT 0,
  last_saved_at BIGINT NOT NULL,
  revision BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (round_id, actor_steam_id, target_steam_id),
  CHECK (relationship_version = 1),
  CHECK (actor_steam_id <> target_steam_id),
  CHECK (
    incap_revives >= 0 AND ledge_rescues >= 0 AND defib_revives >= 0 AND
    smoker_rescues >= 0 AND hunter_rescues >= 0 AND jockey_rescues >= 0 AND charger_rescues >= 0 AND
    control_rescue_duration_ms >= 0 AND medkits_used >= 0 AND medkit_healing >= 0 AND
    black_white_restores >= 0 AND friendly_fire_damage >= 0
  ),
  CHECK (
    incap_revives + ledge_rescues + defib_revives + smoker_rescues + hunter_rescues +
    jockey_rescues + charger_rescues + control_rescue_duration_ms + medkits_used +
    medkit_healing + black_white_restores + friendly_fire_damage > 0
  )
);
-- statement-breakpoint
CREATE INDEX lps_idx_relationship_actor_round
ON lps_player_round_relationship_stats (actor_steam_id, round_id);
-- statement-breakpoint
CREATE INDEX lps_idx_relationship_target_round
ON lps_player_round_relationship_stats (target_steam_id, round_id);
