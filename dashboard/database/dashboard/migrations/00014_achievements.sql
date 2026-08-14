-- +goose Up
CREATE TABLE achievement_unlocks (
  steam_id TEXT NOT NULL,
  achievement_key TEXT NOT NULL,
  achievement_contract_version INTEGER NOT NULL DEFAULT 1 CHECK (achievement_contract_version = 1),
  unlocked_at INTEGER NOT NULL,
  grant_kind TEXT NOT NULL CHECK (grant_kind IN ('live', 'backfill')),
  value_at_unlock INTEGER NOT NULL CHECK (value_at_unlock >= 0),
  evidence_steam_id TEXT NOT NULL DEFAULT '',
  seen_at INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (steam_id, achievement_key)
);

CREATE INDEX achievement_unlocks_key
ON achievement_unlocks (achievement_key, steam_id);

CREATE INDEX achievement_unlocks_player_time
ON achievement_unlocks (steam_id, unlocked_at DESC, achievement_key);

CREATE TABLE achievement_evaluation_state (
  steam_id TEXT PRIMARY KEY,
  achievement_contract_version INTEGER NOT NULL DEFAULT 1 CHECK (achievement_contract_version = 1),
  source_watermark INTEGER NOT NULL DEFAULT 0,
  evaluated_at INTEGER NOT NULL
);

CREATE TABLE player_badge_showcase (
  steam_id TEXT NOT NULL,
  slot INTEGER NOT NULL CHECK (slot BETWEEN 1 AND 3),
  achievement_key TEXT NOT NULL,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (steam_id, slot),
  UNIQUE (steam_id, achievement_key),
  FOREIGN KEY (steam_id, achievement_key)
    REFERENCES achievement_unlocks (steam_id, achievement_key) ON DELETE CASCADE
);

CREATE TABLE achievement_engine_state (
  singleton_id INTEGER PRIMARY KEY CHECK (singleton_id = 1),
  achievement_contract_version INTEGER NOT NULL DEFAULT 1 CHECK (achievement_contract_version = 1),
  global_source_watermark INTEGER NOT NULL DEFAULT 0,
  dirty_cursor_watermark INTEGER NOT NULL DEFAULT 0,
  dirty_cursor_steam_id TEXT NOT NULL DEFAULT '',
  backfill_cursor TEXT NOT NULL DEFAULT '',
  backfill_complete INTEGER NOT NULL DEFAULT 0 CHECK (backfill_complete IN (0, 1)),
  last_run_at INTEGER NOT NULL DEFAULT 0,
  last_success_at INTEGER NOT NULL DEFAULT 0,
  last_error TEXT NOT NULL DEFAULT '',
  updated_at INTEGER NOT NULL
);

INSERT INTO achievement_engine_state (
  singleton_id, achievement_contract_version, updated_at
) VALUES (1, 1, CAST(strftime('%s', 'now') AS INTEGER));

-- +goose Down
DROP TABLE IF EXISTS player_badge_showcase;
DROP TABLE IF EXISTS achievement_evaluation_state;
DROP INDEX IF EXISTS achievement_unlocks_player_time;
DROP INDEX IF EXISTS achievement_unlocks_key;
DROP TABLE IF EXISTS achievement_unlocks;
DROP TABLE IF EXISTS achievement_engine_state;
