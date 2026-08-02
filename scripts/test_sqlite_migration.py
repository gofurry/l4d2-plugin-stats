from __future__ import annotations

import sqlite3
from pathlib import Path


PROJECT_ROOT = Path(__file__).resolve().parent.parent
MIGRATION = PROJECT_ROOT / "database" / "migrations" / "sqlite" / "0001_initial.sql"


def main() -> None:
    sql = MIGRATION.read_text(encoding="utf-8")
    statements = [
        statement.strip()
        for statement in sql.split("-- statement-breakpoint")
        if statement.strip()
    ]

    database = sqlite3.connect(":memory:")
    try:
        for statement in statements:
            database.execute(statement)

        database.execute(
            "INSERT INTO lps_schema_migrations "
            "(version, name, applied_at) VALUES (1, 'initial_schema', 1)"
        )

        # Every DDL statement must be safe after a partially applied migration.
        for statement in statements:
            database.execute(statement)

        database.execute(
            "INSERT INTO lps_servers "
            "(server_key, display_name, first_seen_at, last_seen_at) "
            "VALUES ('test-01', 'Test', 1, 1) "
            "ON CONFLICT(server_key) DO UPDATE SET "
            "display_name = excluded.display_name, last_seen_at = excluded.last_seen_at"
        )
        database.execute(
            "INSERT INTO lps_server_boots "
            "(boot_id, server_key, started_at, ended_at, last_heartbeat_at, status) "
            "VALUES ('test-01:1:a', 'test-01', 1, NULL, 1, 'active') "
            "ON CONFLICT(boot_id) DO UPDATE SET "
            "last_heartbeat_at = excluded.last_heartbeat_at, "
            "ended_at = NULL, status = 'active'"
        )
        database.execute(
            "UPDATE lps_server_boots SET status = 'abandoned', "
            "ended_at = last_heartbeat_at WHERE server_key = 'test-01' "
            "AND boot_id <> 'test-01:2:b' AND status = 'active'"
        )

        database.execute(
            "INSERT INTO lps_players "
            "(steam_id, last_name, first_seen_at, last_seen_at) "
            "VALUES ('76561198000000000', 'First Name', 10, 10) "
            "ON CONFLICT(steam_id) DO UPDATE SET "
            "last_name = CASE WHEN excluded.last_seen_at >= lps_players.last_seen_at "
            "THEN excluded.last_name ELSE lps_players.last_name END, "
            "last_seen_at = MAX(lps_players.last_seen_at, excluded.last_seen_at)"
        )
        database.execute(
            "INSERT INTO lps_players "
            "(steam_id, last_name, first_seen_at, last_seen_at) "
            "VALUES ('76561198000000000', 'Latest Name', 10, 20) "
            "ON CONFLICT(steam_id) DO UPDATE SET "
            "last_name = CASE WHEN excluded.last_seen_at >= lps_players.last_seen_at "
            "THEN excluded.last_name ELSE lps_players.last_name END, "
            "last_seen_at = MAX(lps_players.last_seen_at, excluded.last_seen_at)"
        )

        session_insert = (
            "INSERT INTO lps_sessions "
            "(session_id, boot_id, server_key, steam_id, player_name, ip_address, "
            "started_at, ended_at, last_saved_at, connected_seconds, "
            "active_play_seconds, status, disconnect_reason, revision) "
            "VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) "
            "ON CONFLICT(session_id) DO UPDATE SET "
            "ended_at = excluded.ended_at, last_saved_at = excluded.last_saved_at, "
            "connected_seconds = excluded.connected_seconds, "
            "active_play_seconds = excluded.active_play_seconds, "
            "status = excluded.status, disconnect_reason = excluded.disconnect_reason, "
            "revision = excluded.revision"
        )
        session_id = "test-01:1:a:session:1"
        database.execute(
            session_insert,
            (
                session_id,
                "test-01:1:a",
                "test-01",
                "76561198000000000",
                "First Name",
                "127.0.0.1",
                10,
                None,
                20,
                10,
                7,
                "active",
                "",
                1,
            ),
        )
        database.execute(
            session_insert,
            (
                session_id,
                "test-01:1:a",
                "test-01",
                "76561198000000000",
                "First Name",
                "127.0.0.1",
                10,
                30,
                30,
                20,
                12,
                "closed",
                "client_disconnect",
                2,
            ),
        )

        tables = {
            row[0]
            for row in database.execute(
                "SELECT name FROM sqlite_master "
                "WHERE type = 'table' AND name LIKE 'lps_%'"
            )
        }
        indexes = {
            row[0]
            for row in database.execute(
                "SELECT name FROM sqlite_master "
                "WHERE type = 'index' AND name LIKE 'lps_idx_%'"
            )
        }

        assert len(tables) == 11, tables
        assert len(indexes) == 6, indexes
        status = database.execute(
            "SELECT status FROM lps_server_boots WHERE boot_id = 'test-01:1:a'"
        ).fetchone()
        assert status == ("abandoned",), status
        player = database.execute(
            "SELECT last_name, first_seen_at, last_seen_at FROM lps_players "
            "WHERE steam_id = '76561198000000000'"
        ).fetchone()
        assert player == ("Latest Name", 10, 20), player
        session = database.execute(
            "SELECT ip_address, ended_at, connected_seconds, active_play_seconds, "
            "status, disconnect_reason, revision FROM lps_sessions WHERE session_id = ?",
            (session_id,),
        ).fetchone()
        assert session == (
            "127.0.0.1",
            30,
            20,
            12,
            "closed",
            "client_disconnect",
            2,
        ), session
    finally:
        database.close()

    print(
        f"SQLite integration passed: {len(statements)} statements, "
        f"{len(tables)} tables, {len(indexes)} indexes."
    )


if __name__ == "__main__":
    main()
