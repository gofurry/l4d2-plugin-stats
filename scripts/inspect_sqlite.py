from __future__ import annotations

import argparse
import sqlite3
from pathlib import Path


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Inspect L4D2 Player Stats SQLite state without printing IP addresses."
    )
    parser.add_argument("database", type=Path, help="Path to l4d2_player_stats.sq3")
    args = parser.parse_args()

    if not args.database.is_file():
        raise SystemExit(f"Database file not found: {args.database}")

    database = sqlite3.connect(f"file:{args.database}?mode=ro", uri=True)
    try:
        schema = database.execute(
            "SELECT COALESCE(MAX(version), 0) FROM lps_schema_migrations"
        ).fetchone()[0]
        players = database.execute("SELECT COUNT(*) FROM lps_players").fetchone()[0]
        sessions = database.execute("SELECT COUNT(*) FROM lps_sessions").fetchone()[0]

        print(f"schema_version={schema} players={players} sessions={sessions}")
        print("recent_sessions (IP intentionally omitted):")
        rows = database.execute(
            "SELECT player_name, status, connected_seconds, active_play_seconds, "
            "disconnect_reason, revision FROM lps_sessions "
            "ORDER BY started_at DESC LIMIT 10"
        ).fetchall()
        for row in rows:
            print(
                "  "
                f"name={row[0]!r} status={row[1]} connected={row[2]}s "
                f"active={row[3]}s reason={row[4]!r} revision={row[5]}"
            )
    finally:
        database.close()


if __name__ == "__main__":
    main()
