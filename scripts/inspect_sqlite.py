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
        runs = database.execute("SELECT COUNT(*) FROM lps_runs").fetchone()[0]
        rounds = database.execute("SELECT COUNT(*) FROM lps_rounds").fetchone()[0]
        segments = database.execute(
            "SELECT COUNT(*) FROM lps_player_segments"
        ).fetchone()[0]

        print(
            f"schema_version={schema} players={players} sessions={sessions} "
            f"runs={runs} rounds={rounds} segments={segments}"
        )
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

        print("recent_runs:")
        rows = database.execute(
            "SELECT mode_family, game_mode, campaign_key, status, round_count, "
            "completed_round_count, failed_round_count, revision FROM lps_runs "
            "ORDER BY started_at DESC LIMIT 10"
        ).fetchall()
        for row in rows:
            print(
                "  "
                f"family={row[0]} mode={row[1]} campaign={row[2]!r} "
                f"status={row[3]} rounds={row[4]} completed={row[5]} "
                f"failed={row[6]} revision={row[7]}"
            )

        print("recent_rounds:")
        rows = database.execute(
            "SELECT map_name, mode_family, round_seq, map_seq, attempt_no, half_no, "
            "status, revision FROM lps_rounds ORDER BY started_at DESC LIMIT 10"
        ).fetchall()
        for row in rows:
            print(
                "  "
                f"map={row[0]!r} family={row[1]} round={row[2]} map_seq={row[3]} "
                f"attempt={row[4]} half={row[5]} status={row[6]} revision={row[7]}"
            )

        print("recent_segments:")
        rows = database.execute(
            "SELECT side, active_play_seconds, status, revision "
            "FROM lps_player_segments ORDER BY started_at DESC LIMIT 10"
        ).fetchall()
        for row in rows:
            print(
                "  "
                f"side={row[0]} active={row[1]}s status={row[2]} revision={row[3]}"
            )
    finally:
        database.close()


if __name__ == "__main__":
    main()
