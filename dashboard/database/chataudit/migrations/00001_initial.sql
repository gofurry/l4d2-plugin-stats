-- +goose Up
CREATE TABLE chat_schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at INTEGER NOT NULL
);

INSERT INTO chat_schema_migrations (version, applied_at) VALUES (1, CAST(strftime('%s','now') AS INTEGER));

CREATE TABLE chat_messages (
  message_id TEXT PRIMARY KEY,
  server_key TEXT NOT NULL,
  boot_id TEXT NOT NULL,
  chat_seq INTEGER NOT NULL CHECK (chat_seq > 0),
  session_id TEXT NULL,
  steam_id TEXT NULL,
  source_user_id INTEGER NOT NULL,
  player_name TEXT NOT NULL,
  occurred_at INTEGER NOT NULL,
  map_name TEXT NOT NULL,
  game_mode TEXT NOT NULL,
  team TEXT NOT NULL,
  channel TEXT NOT NULL CHECK (channel IN ('global', 'team')),
  alive INTEGER NOT NULL CHECK (alive IN (0, 1)),
  command_like INTEGER NOT NULL CHECK (command_like IN (0, 1)),
  content TEXT NOT NULL,
  UNIQUE (boot_id, chat_seq)
);

CREATE INDEX chat_messages_time_idx ON chat_messages (occurred_at DESC, message_id DESC);
CREATE INDEX chat_messages_server_time_idx ON chat_messages (server_key, occurred_at DESC, message_id DESC);
CREATE INDEX chat_messages_steam_time_idx ON chat_messages (steam_id, occurred_at DESC, message_id DESC);

CREATE TABLE chat_ingest_cursors (
  boot_id TEXT PRIMARY KEY,
  server_key TEXT NOT NULL,
  last_chat_seq INTEGER NOT NULL DEFAULT 0 CHECK (last_chat_seq >= 0),
  gap_count INTEGER NOT NULL DEFAULT 0 CHECK (gap_count >= 0),
  last_gap_from INTEGER NOT NULL DEFAULT 0,
  last_gap_to INTEGER NOT NULL DEFAULT 0,
  updated_at INTEGER NOT NULL
);

-- +goose Down
DROP TABLE chat_ingest_cursors;
DROP TABLE chat_messages;
DROP TABLE chat_schema_migrations;
