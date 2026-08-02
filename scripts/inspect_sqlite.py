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

        integrity = database.execute("PRAGMA integrity_check").fetchone()[0]
        orphan_rounds = database.execute(
            "SELECT COUNT(*) FROM lps_rounds r LEFT JOIN lps_runs u "
            "ON u.run_id = r.run_id WHERE u.run_id IS NULL"
        ).fetchone()[0]
        orphan_segments = database.execute(
            "SELECT COUNT(*) FROM lps_player_segments g "
            "LEFT JOIN lps_sessions s ON s.session_id = g.session_id "
            "LEFT JOIN lps_runs u ON u.run_id = g.run_id "
            "LEFT JOIN lps_rounds r ON r.round_id = g.round_id "
            "WHERE s.session_id IS NULL OR u.run_id IS NULL OR r.round_id IS NULL"
        ).fetchone()[0]
        segment_run_mismatches = database.execute(
            "SELECT COUNT(*) FROM lps_player_segments g "
            "JOIN lps_rounds r ON r.round_id = g.round_id "
            "WHERE r.run_id <> g.run_id"
        ).fetchone()[0]
        invalid_times = sum(
            database.execute(query).fetchone()[0]
            for query in (
                "SELECT COUNT(*) FROM lps_sessions WHERE connected_seconds < 0 "
                "OR active_play_seconds < 0 OR active_play_seconds > connected_seconds "
                "OR (ended_at IS NOT NULL AND ended_at < started_at)",
                "SELECT COUNT(*) FROM lps_runs WHERE ended_at IS NOT NULL "
                "AND ended_at < started_at",
                "SELECT COUNT(*) FROM lps_rounds WHERE ended_at IS NOT NULL "
                "AND ended_at < started_at",
                "SELECT COUNT(*) FROM lps_player_segments WHERE active_play_seconds < 0 "
                "OR (ended_at IS NOT NULL AND ended_at < started_at)",
            )
        )
        active_records = sum(
            database.execute(
                f"SELECT COUNT(*) FROM {table} WHERE status = 'active'"
            ).fetchone()[0]
            for table in (
                "lps_sessions",
                "lps_runs",
                "lps_rounds",
                "lps_player_segments",
            )
        )

        print(
            f"schema_version={schema} players={players} sessions={sessions} "
            f"runs={runs} rounds={rounds} segments={segments}"
        )
        print(
            f"health integrity={integrity} orphan_rounds={orphan_rounds} "
            f"orphan_segments={orphan_segments} "
            f"segment_run_mismatches={segment_run_mismatches} "
            f"invalid_times={invalid_times} active_records={active_records}"
        )
        print("recent_sessions (IP intentionally omitted):")
        rows = database.execute(
            "SELECT session_id, player_name, status, connected_seconds, "
            "active_play_seconds, disconnect_reason, revision FROM lps_sessions "
            "ORDER BY started_at DESC LIMIT 10"
        ).fetchall()
        for row in rows:
            print(
                "  "
                f"session={row[0]!r} name={row[1]!r} status={row[2]} "
                f"connected={row[3]}s active={row[4]}s "
                f"reason={row[5]!r} revision={row[6]}"
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
