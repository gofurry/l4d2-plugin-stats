from __future__ import annotations

import sqlite3
from pathlib import Path


PROJECT_ROOT = Path(__file__).resolve().parent.parent
MIGRATION_ROOT = PROJECT_ROOT / "database" / "migrations" / "sqlite"
INITIAL_MIGRATION = MIGRATION_ROOT / "0001_initial.sql"
INCIDENT_MIGRATION = MIGRATION_ROOT / "0002_car_alarms_triggered.sql"
VERSUS_CONTRACT_CHECKS = (
    PROJECT_ROOT / "database" / "queries" / "versus_contract_checks.sql"
)

PVE_STAT_COLUMNS = [
    "stats_version", "last_saved_at", "common_kills", "special_kills",
    "tank_kills", "witch_kills", "damage_to_special", "damage_to_tank",
    "damage_to_witch", "damage_taken_infected", "friendly_fire_to_humans",
    "friendly_fire_to_bots", "friendly_fire_taken", "incapacitations",
    "deaths", "incap_revives", "ledge_rescues", "defib_revives",
    "rescues_received", "medkits_used_self", "medkits_used_on_others",
    "medkit_healing_self", "medkit_healing_others", "pills_used",
    "adrenaline_used", "temporary_health_received", "chapter_participations",
    "chapter_completions_alive", "chapter_completions_dead",
    "campaign_completions", "smoker_kills", "boomer_kills", "hunter_kills",
    "spitter_kills", "jockey_kills", "charger_kills", "damage_to_smoker",
    "damage_to_boomer", "damage_to_hunter", "damage_to_spitter",
    "damage_to_jockey", "damage_to_charger", "smoker_controls_received",
    "hunter_controls_received", "jockey_controls_received",
    "charger_controls_received", "smoker_controlled_seconds",
    "hunter_controlled_seconds", "jockey_controlled_seconds",
    "charger_controlled_seconds", "smoker_saves", "hunter_saves",
    "jockey_saves", "charger_saves", "melee_tongue_self_cuts",
    "tank_rocks_destroyed", "witch_oneshots", "witch_solo_kills",
    "tank_encounters", "tank_kill_participations", "witch_encounters",
    "witch_kill_participations", "incendiary_packs_deployed",
    "explosive_packs_deployed", "objective_interactions", "ammo_pile_uses",
    "incapacitated_seconds", "ledge_hanging_seconds",
    "black_white_teammates_restored", "revision", "car_alarms_triggered",
]

EQUIPMENT_STAT_COLUMNS = [
    "equipment_id", "stats_version", "last_saved_at", "actions",
    "common_kills", "special_kills", "tank_kills", "witch_kills",
    "headshot_kills", "damage_to_special", "damage_to_tank",
    "damage_to_witch", "revision",
]

VERSUS_SURVIVOR_STAT_COLUMNS = [
    "stats_version", "last_saved_at", "common_kills",
    "human_special_kills", "bot_special_kills", "human_tank_kills",
    "bot_tank_kills", "damage_to_human_special", "damage_to_bot_special",
    "damage_to_human_tank", "damage_to_bot_tank", "damage_taken_infected",
    "friendly_fire_to_humans", "friendly_fire_to_bots",
    "friendly_fire_taken", "incapacitations", "deaths", "incap_revives",
    "ledge_rescues", "defib_revives", "rescues_received",
    "medkits_used_self", "medkits_used_on_others", "medkit_healing_self",
    "medkit_healing_others", "pills_used", "adrenaline_used",
    "temporary_health_received", "witch_kills", "damage_to_witch",
    "molotovs_thrown", "pipe_bombs_thrown", "vomit_jars_thrown",
    "incendiary_packs_deployed", "explosive_packs_deployed",
    "melee_tongue_self_cuts", "tank_rocks_destroyed", "witch_oneshots",
    "witch_solo_kills", "revision", "car_alarms_triggered",
]

VERSUS_SURVIVOR_CLASS_STAT_COLUMNS = [
    "infected_class", "stats_version", "last_saved_at",
    "human_controller_kills", "bot_controller_kills",
    "damage_to_human_controllers", "damage_to_bot_controllers", "revision",
]

VERSUS_INFECTED_STAT_COLUMNS = [
    "stats_version", "last_saved_at", "spawn_count",
    "damage_to_human_survivors", "damage_to_bot_survivors",
    "human_survivor_incaps", "bot_survivor_incaps",
    "human_survivor_kills", "bot_survivor_kills", "revision",
]

VERSUS_INFECTED_CLASS_STAT_COLUMNS = [
    "infected_class", "stats_version", "last_saved_at", "spawn_count",
    "damage_to_human_survivors", "damage_to_bot_survivors",
    "human_survivor_incaps", "bot_survivor_incaps",
    "human_survivor_kills", "bot_survivor_kills",
    "human_survivor_controls", "bot_survivor_controls",
    "human_survivor_control_seconds", "bot_survivor_control_seconds",
    "human_survivor_ability_hits", "bot_survivor_ability_hits",
    "human_survivor_ability_damage", "bot_survivor_ability_damage",
    "revision",
]


def build_snapshot_upsert(table: str, key_columns: list[str], columns: list[str]) -> str:
    all_columns = ["segment_id", *columns]
    updates = ", ".join(
        f"{column} = excluded.{column}"
        for column in columns
        if column not in key_columns
    )
    conflict = ", ".join(["segment_id", *key_columns])
    return (
        f"INSERT INTO {table} ({', '.join(all_columns)}) "
        f"VALUES ({', '.join(['?'] * len(all_columns))}) "
        f"ON CONFLICT({conflict}) DO UPDATE SET {updates}"
    )


def run_versus_contract_checks(database: sqlite3.Connection) -> dict[str, int]:
    statements = [
        statement.strip()
        for statement in VERSUS_CONTRACT_CHECKS.read_text(encoding="utf-8").split(
            "-- statement-breakpoint"
        )
        if statement.strip()
    ]
    results: dict[str, int] = {}
    for statement in statements:
        row = database.execute(statement).fetchone()
        if row is None or len(row) != 2:
            raise AssertionError(f"invalid Versus contract check result: {row!r}")
        name, violations = row
        results[str(name)] = int(violations)
    return results


def main() -> None:
    sql = INITIAL_MIGRATION.read_text(encoding="utf-8")
    statements = [
        statement.strip()
        for statement in sql.split("-- statement-breakpoint")
        if statement.strip()
    ]

    database = sqlite3.connect(":memory:")
    try:
        for statement in statements:
            database.execute(statement)

        incident_statements = [
            statement.strip()
            for statement in INCIDENT_MIGRATION.read_text(encoding="utf-8").split(
                "-- statement-breakpoint"
            )
            if statement.strip()
        ]
        for statement in incident_statements:
            database.execute(statement)
        database.execute(
            "INSERT INTO lps_schema_migrations "
            "(version, name, applied_at) VALUES (2, 'car_alarms_triggered', 2)"
        )
        for table in ("lps_pve_segment_stats", "lps_versus_survivor_stats"):
            column = next(
                row for row in database.execute(f"PRAGMA table_info({table})")
                if row[1] == "car_alarms_triggered"
            )
            assert column[3] == 0, column

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

        run_insert = (
            "INSERT INTO lps_runs "
            "(run_id, boot_id, server_key, mode_family, game_mode, campaign_key, "
            "started_at, ended_at, last_saved_at, status, round_count, "
            "completed_round_count, failed_round_count, revision) "
            "VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) "
            "ON CONFLICT(run_id) DO UPDATE SET "
            "ended_at = excluded.ended_at, last_saved_at = excluded.last_saved_at, "
            "status = excluded.status, round_count = excluded.round_count, "
            "completed_round_count = excluded.completed_round_count, "
            "failed_round_count = excluded.failed_round_count, revision = excluded.revision"
        )
        run_id = "test-01:1:a:run:1"
        database.execute(
            run_insert,
            (
                run_id, "test-01:1:a", "test-01", "pve", "coop", "c1", 10,
                None, 20, "active", 1, 0, 0, 1,
            ),
        )
        database.execute(
            run_insert,
            (
                run_id, "test-01:1:a", "test-01", "pve", "coop", "c1", 10,
                90, 90, "completed", 2, 1, 1, 4,
            ),
        )

        round_insert = (
            "INSERT INTO lps_rounds "
            "(round_id, run_id, server_key, mode_family, map_name, round_seq, "
            "map_seq, attempt_no, half_no, started_at, ended_at, last_saved_at, "
            "status, revision) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) "
            "ON CONFLICT(round_id) DO UPDATE SET "
            "ended_at = excluded.ended_at, last_saved_at = excluded.last_saved_at, "
            "status = excluded.status, revision = excluded.revision"
        )
        round_id = "test-01:1:a:round:1"
        database.execute(
            round_insert,
            (
                round_id, run_id, "test-01", "pve", "c1m1_hotel", 1, 1, 1, 0,
                10, None, 20, "active", 1,
            ),
        )
        database.execute(
            round_insert,
            (
                round_id, run_id, "test-01", "pve", "c1m1_hotel", 1, 1, 1, 0,
                10, 45, 45, "failed", 2,
            ),
        )

        segment_insert = (
            "INSERT INTO lps_player_segments "
            "(segment_id, session_id, run_id, round_id, server_key, steam_id, side, "
            "started_at, ended_at, last_saved_at, active_play_seconds, status, revision) "
            "VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) "
            "ON CONFLICT(segment_id) DO UPDATE SET "
            "ended_at = excluded.ended_at, last_saved_at = excluded.last_saved_at, "
            "active_play_seconds = excluded.active_play_seconds, "
            "status = excluded.status, revision = excluded.revision"
        )
        segment_id = "test-01:1:a:segment:1"
        database.execute(
            segment_insert,
            (
                segment_id, session_id, run_id, round_id, "test-01",
                "76561198000000000", "survivor", 12, None, 20, 8, "active", 1,
            ),
        )
        database.execute(
            segment_insert,
            (
                segment_id, session_id, run_id, round_id, "test-01",
                "76561198000000000", "survivor", 12, 45, 45, 31, "closed", 2,
            ),
        )

        pve_stats_insert = build_snapshot_upsert(
            "lps_pve_segment_stats", [], PVE_STAT_COLUMNS
        )
        database.execute(
            pve_stats_insert,
            (
                segment_id, 1, 20, 5, 1, 0, 0, 80, 0, 0, 12, 0, 3, 0,
                0, 0, 0, 0, 0, 0, 1, 0, 42, 0, 1, 0, 50, 0, 0, 0, 0,
                *([0] * 39), 1, 0,
            ),
        )
        # Absolute snapshots replace the stored values; they must never be
        # interpreted as deltas and added again during a retry.
        database.execute(
            pve_stats_insert,
            (
                segment_id, 1, 45, 12, 3, 1, 1, 250, 400, 90, 35, 7, 8, 4,
                2, 1, 1, 1, 1, 3, 2, 1, 80, 55, 2, 1, 125, 1, 1, 0, 0,
                1, 0, 1, 0, 1, 0, 100, 20, 50, 10, 30, 40,
                1, 2, 3, 4, 5, 6, 7, 8, 2, 1, 0, 1,
                1, 2, 1, 1, 1, 1, 1, 1, 1, 2, 3, 2, 11, 4, 1, 2, 3,
            ),
        )

        equipment_stats_insert = build_snapshot_upsert(
            "lps_pve_segment_equipment_stats",
            ["equipment_id"],
            EQUIPMENT_STAT_COLUMNS,
        )
        # Equipment 12 is the official M16; equipment 1 is the single fixed
        # Other Firearm bucket used by every unknown/custom firearm.
        database.execute(
            equipment_stats_insert,
            (segment_id, 12, 1, 20, 0, 3, 1, 0, 0, 2, 80, 0, 0, 1),
        )
        database.execute(
            equipment_stats_insert,
            (segment_id, 12, 1, 45, 0, 7, 2, 1, 0, 4, 180, 400, 0, 2),
        )
        database.execute(
            equipment_stats_insert,
            (segment_id, 1, 1, 45, 0, 2, 1, 0, 0, 1, 70, 0, 0, 1),
        )

        # Simulate a process that disappeared while a Versus lifecycle was active.
        # Registration of the next boot must close every stale active layer.
        stale_session_id = "test-01:1:a:session:2"
        stale_run_id = "test-01:1:a:run:2"
        stale_round_id = "test-01:1:a:round:2"
        stale_segment_id = "test-01:1:a:segment:2"
        stale_infected_segment_id = "test-01:1:a:segment:3"
        database.execute(
            session_insert,
            (
                stale_session_id, "test-01:1:a", "test-01",
                "76561198000000000", "Latest Name", "127.0.0.1",
                100, None, 125, 25, 20, "active", "", 3,
            ),
        )
        database.execute(
            run_insert,
            (
                stale_run_id, "test-01:1:a", "test-01", "versus", "versus",
                "c5", 100, None, 125, "active", 1, 1, 0, 3,
            ),
        )
        database.execute(
            round_insert,
            (
                stale_round_id, stale_run_id, "test-01", "versus",
                "c5m1_waterfront", 1, 1, 1, 1, 100, None, 125, "active", 2,
            ),
        )
        versus_round_result_insert = (
            "INSERT INTO lps_versus_round_results "
            "(round_id, stats_version, last_saved_at, scoring_team_slot, "
            "teams_flipped, team_0_map_score, team_1_map_score, "
            "team_0_campaign_score, team_1_campaign_score, raw_winner_team, "
            "score_available, result_status, finalized_at, revision) "
            "VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) "
            "ON CONFLICT(round_id) DO UPDATE SET "
            "last_saved_at = excluded.last_saved_at, "
            "scoring_team_slot = excluded.scoring_team_slot, "
            "teams_flipped = excluded.teams_flipped, "
            "team_0_map_score = excluded.team_0_map_score, "
            "team_1_map_score = excluded.team_1_map_score, "
            "team_0_campaign_score = excluded.team_0_campaign_score, "
            "team_1_campaign_score = excluded.team_1_campaign_score, "
            "raw_winner_team = excluded.raw_winner_team, "
            "score_available = excluded.score_available, "
            "result_status = excluded.result_status, "
            "finalized_at = excluded.finalized_at, revision = excluded.revision"
        )
        database.execute(
            versus_round_result_insert,
            (
                stale_round_id, 1, 125, 0, 0, 420, -1, 0, 0, 2, 1,
                "completed", 125, 1,
            ),
        )
        # A restarted half reclassifies the same authoritative snapshot rather
        # than creating a second result row.
        database.execute(
            versus_round_result_insert,
            (
                stale_round_id, 1, 126, 0, 0, 420, -1, 0, 0, 2, 1,
                "abandoned", 125, 2,
            ),
        )

        versus_run_result_insert = (
            "INSERT INTO lps_versus_run_results "
            "(run_id, stats_version, last_saved_at, team_0_campaign_score, "
            "team_1_campaign_score, winner_team_slot, raw_winner_team, "
            "score_available, result_status, finalized_at, revision) "
            "VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) "
            "ON CONFLICT(run_id) DO UPDATE SET "
            "last_saved_at = excluded.last_saved_at, "
            "team_0_campaign_score = excluded.team_0_campaign_score, "
            "team_1_campaign_score = excluded.team_1_campaign_score, "
            "winner_team_slot = excluded.winner_team_slot, "
            "raw_winner_team = excluded.raw_winner_team, "
            "score_available = excluded.score_available, "
            "result_status = excluded.result_status, "
            "finalized_at = excluded.finalized_at, revision = excluded.revision"
        )
        database.execute(
            versus_run_result_insert,
            (stale_run_id, 1, 125, 0, 0, -1, -1, 1, "active", None, 1),
        )
        database.execute(
            segment_insert,
            (
                stale_segment_id, stale_session_id, stale_run_id, stale_round_id,
                "test-01", "76561198000000000", "survivor",
                100, None, 125, 20, "active", 2,
            ),
        )
        database.execute(
            segment_insert,
            (
                stale_infected_segment_id, stale_session_id, stale_run_id,
                stale_round_id, "test-01", "76561198000000000", "infected",
                105, None, 125, 15, "active", 2,
            ),
        )

        versus_survivor_insert = build_snapshot_upsert(
            "lps_versus_survivor_stats", [], VERSUS_SURVIVOR_STAT_COLUMNS
        )
        database.execute(
            versus_survivor_insert,
            (
                stale_segment_id, 1, 115, 4, 1, 2, 0, 1, 100, 80, 0, 200,
                25, 2, 3, 4, 1, 0, 1, 0, 0, 1, 1, 0, 30, 0, 1, 0, 50,
                0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0,
            ),
        )
        # A retry replaces the absolute snapshot rather than incrementing it.
        database.execute(
            versus_survivor_insert,
            (
                stale_segment_id, 1, 125, 12, 3, 4, 1, 2, 300, 200, 500,
                400, 70, 10, 11, 12, 2, 1, 3, 1, 1, 4, 1, 2, 80, 90, 2,
                1, 125, 2, 300, 2, 1, 1, 1, 1, 1, 2, 1, 1, 2, 4,
            ),
        )

        versus_survivor_class_insert = build_snapshot_upsert(
            "lps_versus_survivor_infected_class_stats",
            ["infected_class"],
            VERSUS_SURVIVOR_CLASS_STAT_COLUMNS,
        )
        database.execute(
            versus_survivor_class_insert,
            (stale_segment_id, 1, 1, 115, 0, 1, 50, 20, 1),
        )
        # Absolute retry replaces Smoker's earlier row.
        database.execute(
            versus_survivor_class_insert,
            (stale_segment_id, 1, 1, 125, 1, 1, 100, 50, 2),
        )
        database.execute(
            versus_survivor_class_insert,
            (stale_segment_id, 3, 1, 125, 1, 2, 100, 100, 2),
        )
        database.execute(
            versus_survivor_class_insert,
            (stale_segment_id, 6, 1, 125, 1, 1, 100, 50, 2),
        )
        database.execute(
            versus_survivor_class_insert,
            (stale_segment_id, 8, 1, 125, 1, 2, 500, 400, 2),
        )

        versus_infected_insert = build_snapshot_upsert(
            "lps_versus_infected_stats", [], VERSUS_INFECTED_STAT_COLUMNS
        )
        database.execute(
            versus_infected_insert,
            (
                stale_infected_segment_id, 1, 115, 2, 150, 60, 1, 0, 0, 0, 1,
            ),
        )

        versus_infected_class_insert = build_snapshot_upsert(
            "lps_versus_infected_class_stats",
            ["infected_class"],
            VERSUS_INFECTED_CLASS_STAT_COLUMNS,
        )
        # Class IDs use L4D2's stable zombie-class values: 1-6 and Tank 8.
        database.execute(
            versus_infected_class_insert,
            (
                stale_infected_segment_id, 1, 1, 115, 1, 50, 10, 1, 0, 0, 0,
                1, 0, 4, 0, 0, 0, 0, 0, 1,
            ),
        )
        # Absolute retry replaces Smoker's earlier row.
        database.execute(
            versus_infected_class_insert,
            (
                stale_infected_segment_id, 1, 1, 125, 1, 100, 20, 1, 0, 1, 0,
                2, 1, 10, 4, 0, 0, 0, 0, 2,
            ),
        )
        database.execute(
            versus_infected_class_insert,
            (
                stale_infected_segment_id, 2, 1, 125, 1, 0, 0, 0, 0, 0, 0,
                0, 0, 0, 0, 2, 3, 0, 0, 2,
            ),
        )
        database.execute(
            versus_infected_class_insert,
            (
                stale_infected_segment_id, 4, 1, 125, 1, 200, 80, 0, 0, 0, 0,
                0, 0, 0, 0, 0, 0, 200, 80, 2,
            ),
        )
        database.execute(
            versus_infected_class_insert,
            (
                stale_infected_segment_id, 8, 1, 125, 1, 150, 20, 1, 1, 0, 0,
                0, 0, 0, 0, 0, 0, 0, 0, 2,
            ),
        )
        database.execute(
            versus_infected_insert,
            (
                stale_infected_segment_id, 1, 125, 4, 450, 120, 2, 1, 1, 0, 2,
            ),
        )

        current_boot_id = "test-01:2:b"
        database.execute(
            "UPDATE lps_player_segments SET status = 'abandoned', "
            "ended_at = last_saved_at WHERE status = 'active' AND session_id IN "
            "(SELECT session_id FROM lps_sessions WHERE server_key = ? "
            "AND boot_id <> ?)",
            ("test-01", current_boot_id),
        )
        database.execute(
            "UPDATE lps_rounds SET status = 'abandoned', ended_at = last_saved_at "
            "WHERE status = 'active' AND run_id IN "
            "(SELECT run_id FROM lps_runs WHERE server_key = ? AND boot_id <> ?)",
            ("test-01", current_boot_id),
        )
        database.execute(
            "UPDATE lps_versus_run_results SET result_status = 'abandoned', "
            "finalized_at = last_saved_at WHERE result_status = 'active' "
            "AND run_id IN (SELECT run_id FROM lps_runs WHERE server_key = ? "
            "AND boot_id <> ?)",
            ("test-01", current_boot_id),
        )
        database.execute(
            "UPDATE lps_sessions SET status = 'abandoned', ended_at = last_saved_at "
            "WHERE server_key = ? AND boot_id <> ? AND status = 'active'",
            ("test-01", current_boot_id),
        )
        database.execute(
            "UPDATE lps_runs SET status = 'abandoned', ended_at = last_saved_at "
            "WHERE server_key = ? AND boot_id <> ? AND status = 'active'",
            ("test-01", current_boot_id),
        )

        completed_versus_run_id = "test-01:1:a:run:3"
        database.execute(
            run_insert,
            (
                completed_versus_run_id, "test-01:1:a", "test-01", "versus",
                "versus", "c8", 200, 300, 300, "completed", 2, 2, 0, 4,
            ),
        )
        database.execute(
            versus_run_result_insert,
            (
                completed_versus_run_id, 1, 250, 500, 450, -1, -1, 1,
                "active", None, 1,
            ),
        )
        database.execute(
            versus_run_result_insert,
            (
                completed_versus_run_id, 1, 300, 500, 700, 1, 2, 1,
                "completed", 300, 2,
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

        assert len(tables) == 16, tables
        assert len(indexes) == 31, indexes
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
        run = database.execute(
            "SELECT status, round_count, completed_round_count, "
            "failed_round_count, revision FROM lps_runs WHERE run_id = ?",
            (run_id,),
        ).fetchone()
        assert run == ("completed", 2, 1, 1, 4), run
        round_row = database.execute(
            "SELECT map_seq, attempt_no, half_no, status, revision "
            "FROM lps_rounds WHERE round_id = ?", (round_id,)
        ).fetchone()
        assert round_row == (1, 1, 0, "failed", 2), round_row
        segment = database.execute(
            "SELECT side, active_play_seconds, status, revision "
            "FROM lps_player_segments WHERE segment_id = ?", (segment_id,)
        ).fetchone()
        assert segment == ("survivor", 31, "closed", 2), segment
        pve_stats = database.execute(
            f"SELECT {', '.join(PVE_STAT_COLUMNS)} "
            "FROM lps_pve_segment_stats WHERE segment_id = ?", (segment_id,)
        ).fetchone()
        assert pve_stats == (
            1, 45, 12, 3, 1, 1, 250, 400, 90, 35, 7, 8, 4,
            2, 1, 1, 1, 1, 3, 2, 1, 80, 55, 2, 1, 125, 1, 1, 0, 0,
            1, 0, 1, 0, 1, 0, 100, 20, 50, 10, 30, 40,
            1, 2, 3, 4, 5, 6, 7, 8, 2, 1, 0, 1,
            1, 2, 1, 1, 1, 1, 1, 1, 1, 2, 3, 2, 11, 4, 1, 2, 3,
        ), pve_stats

        equipment_stats = database.execute(
            f"SELECT {', '.join(EQUIPMENT_STAT_COLUMNS)} "
            "FROM lps_pve_segment_equipment_stats WHERE segment_id = ? "
            "ORDER BY equipment_id",
            (segment_id,),
        ).fetchall()
        assert equipment_stats == [
            (1, 1, 45, 0, 2, 1, 0, 0, 1, 70, 0, 0, 1),
            (12, 1, 45, 0, 7, 2, 1, 0, 4, 180, 400, 0, 2),
        ], equipment_stats

        stale_session = database.execute(
            "SELECT ended_at, status FROM lps_sessions WHERE session_id = ?",
            (stale_session_id,),
        ).fetchone()
        assert stale_session == (125, "abandoned"), stale_session
        stale_run = database.execute(
            "SELECT ended_at, status FROM lps_runs WHERE run_id = ?",
            (stale_run_id,),
        ).fetchone()
        assert stale_run == (125, "abandoned"), stale_run
        stale_round = database.execute(
            "SELECT ended_at, status FROM lps_rounds WHERE round_id = ?",
            (stale_round_id,),
        ).fetchone()
        assert stale_round == (125, "abandoned"), stale_round
        versus_round_result = database.execute(
            "SELECT scoring_team_slot, teams_flipped, team_0_map_score, "
            "team_1_map_score, team_0_campaign_score, team_1_campaign_score, "
            "score_available, result_status, revision "
            "FROM lps_versus_round_results WHERE round_id = ?",
            (stale_round_id,),
        ).fetchone()
        assert versus_round_result == (
            0, 0, 420, -1, 0, 0, 1, "abandoned", 2,
        ), versus_round_result
        versus_run_result = database.execute(
            "SELECT team_0_campaign_score, team_1_campaign_score, "
            "winner_team_slot, score_available, result_status, finalized_at, "
            "revision FROM lps_versus_run_results WHERE run_id = ?",
            (stale_run_id,),
        ).fetchone()
        assert versus_run_result == (
            0, 0, -1, 1, "abandoned", 125, 1,
        ), versus_run_result
        completed_versus_run_result = database.execute(
            "SELECT team_0_campaign_score, team_1_campaign_score, "
            "winner_team_slot, raw_winner_team, score_available, result_status, "
            "finalized_at, revision FROM lps_versus_run_results WHERE run_id = ?",
            (completed_versus_run_id,),
        ).fetchone()
        assert completed_versus_run_result == (
            500, 700, 1, 2, 1, "completed", 300, 2,
        ), completed_versus_run_result
        stale_segment = database.execute(
            "SELECT ended_at, status FROM lps_player_segments WHERE segment_id = ?",
            (stale_segment_id,),
        ).fetchone()
        assert stale_segment == (125, "abandoned"), stale_segment
        stale_infected_segment = database.execute(
            "SELECT ended_at, status FROM lps_player_segments WHERE segment_id = ?",
            (stale_infected_segment_id,),
        ).fetchone()
        assert stale_infected_segment == (125, "abandoned"), stale_infected_segment

        versus_survivor_stats = database.execute(
            f"SELECT {', '.join(VERSUS_SURVIVOR_STAT_COLUMNS)} "
            "FROM lps_versus_survivor_stats WHERE segment_id = ?",
            (stale_segment_id,),
        ).fetchone()
        assert versus_survivor_stats == (
            1, 125, 12, 3, 4, 1, 2, 300, 200, 500, 400, 70, 10, 11, 12,
            2, 1, 3, 1, 1, 4, 1, 2, 80, 90, 2, 1, 125, 2, 300, 2, 1,
            1, 1, 1, 1, 2, 1, 1, 2, 4,
        ), versus_survivor_stats

        versus_survivor_class_stats = database.execute(
            f"SELECT {', '.join(VERSUS_SURVIVOR_CLASS_STAT_COLUMNS)} "
            "FROM lps_versus_survivor_infected_class_stats "
            "WHERE segment_id = ? ORDER BY infected_class",
            (stale_segment_id,),
        ).fetchall()
        assert versus_survivor_class_stats == [
            (1, 1, 125, 1, 1, 100, 50, 2),
            (3, 1, 125, 1, 2, 100, 100, 2),
            (6, 1, 125, 1, 1, 100, 50, 2),
            (8, 1, 125, 1, 2, 500, 400, 2),
        ], versus_survivor_class_stats

        survivor_special_class_totals = database.execute(
            "SELECT SUM(human_controller_kills), SUM(bot_controller_kills), "
            "SUM(damage_to_human_controllers), "
            "SUM(damage_to_bot_controllers) "
            "FROM lps_versus_survivor_infected_class_stats "
            "WHERE segment_id = ? AND infected_class BETWEEN 1 AND 6",
            (stale_segment_id,),
        ).fetchone()
        assert survivor_special_class_totals == (3, 4, 300, 200), (
            survivor_special_class_totals
        )

        survivor_tank_class_totals = database.execute(
            "SELECT human_controller_kills, bot_controller_kills, "
            "damage_to_human_controllers, damage_to_bot_controllers "
            "FROM lps_versus_survivor_infected_class_stats "
            "WHERE segment_id = ? AND infected_class = 8",
            (stale_segment_id,),
        ).fetchone()
        assert survivor_tank_class_totals == (1, 2, 500, 400), (
            survivor_tank_class_totals
        )

        versus_infected_stats = database.execute(
            f"SELECT {', '.join(VERSUS_INFECTED_STAT_COLUMNS)} "
            "FROM lps_versus_infected_stats WHERE segment_id = ?",
            (stale_infected_segment_id,),
        ).fetchone()
        assert versus_infected_stats == (
            1, 125, 4, 450, 120, 2, 1, 1, 0, 2,
        ), versus_infected_stats

        versus_infected_class_stats = database.execute(
            f"SELECT {', '.join(VERSUS_INFECTED_CLASS_STAT_COLUMNS)} "
            "FROM lps_versus_infected_class_stats WHERE segment_id = ? "
            "ORDER BY infected_class",
            (stale_infected_segment_id,),
        ).fetchall()
        assert versus_infected_class_stats == [
            (1, 1, 125, 1, 100, 20, 1, 0, 1, 0, 2, 1, 10, 4, 0, 0, 0, 0, 2),
            (2, 1, 125, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2, 3, 0, 0, 2),
            (4, 1, 125, 1, 200, 80, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 200, 80, 2),
            (8, 1, 125, 1, 150, 20, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2),
        ], versus_infected_class_stats

        class_totals = database.execute(
            "SELECT SUM(spawn_count), SUM(damage_to_human_survivors), "
            "SUM(damage_to_bot_survivors), SUM(human_survivor_incaps), "
            "SUM(bot_survivor_incaps), SUM(human_survivor_kills), "
            "SUM(bot_survivor_kills) "
            "FROM lps_versus_infected_class_stats WHERE segment_id = ?",
            (stale_infected_segment_id,),
        ).fetchone()
        assert class_totals == (4, 450, 120, 2, 1, 1, 0), class_totals

        ability_totals = database.execute(
            "SELECT SUM(human_survivor_controls), SUM(bot_survivor_controls), "
            "SUM(human_survivor_control_seconds), "
            "SUM(bot_survivor_control_seconds), "
            "SUM(human_survivor_ability_hits), "
            "SUM(bot_survivor_ability_hits), "
            "SUM(human_survivor_ability_damage), "
            "SUM(bot_survivor_ability_damage) "
            "FROM lps_versus_infected_class_stats WHERE segment_id = ?",
            (stale_infected_segment_id,),
        ).fetchone()
        assert ability_totals == (2, 1, 10, 4, 2, 3, 200, 80), ability_totals

        versus_side_mismatches = database.execute(
            "SELECT COUNT(*) FROM ("
            "SELECT s.segment_id FROM lps_versus_survivor_stats s "
            "JOIN lps_player_segments g ON g.segment_id = s.segment_id "
            "WHERE g.side <> 'survivor' UNION ALL "
            "SELECT c.segment_id FROM lps_versus_survivor_infected_class_stats c "
            "JOIN lps_player_segments g ON g.segment_id = c.segment_id "
            "WHERE g.side <> 'survivor' UNION ALL "
            "SELECT i.segment_id FROM lps_versus_infected_stats i "
            "JOIN lps_player_segments g ON g.segment_id = i.segment_id "
            "WHERE g.side <> 'infected')"
        ).fetchone()[0]
        assert versus_side_mismatches == 0, versus_side_mismatches

        contract_checks = run_versus_contract_checks(database)
        assert len(contract_checks) == 10, contract_checks
        assert all(value == 0 for value in contract_checks.values()), contract_checks

        # Prove that the frozen checks reject the three most important forms
        # of drift instead of only accepting the happy-path fixture.
        database.execute(
            "UPDATE lps_versus_round_results SET scoring_team_slot = 1 "
            "WHERE round_id = ?",
            (stale_round_id,),
        )
        assert run_versus_contract_checks(database)[
            "versus_scoring_slot_mismatches"
        ] == 1
        database.execute(
            "UPDATE lps_versus_round_results SET scoring_team_slot = 0 "
            "WHERE round_id = ?",
            (stale_round_id,),
        )

        database.execute(
            "UPDATE lps_versus_survivor_stats SET human_special_kills = 4 "
            "WHERE segment_id = ?",
            (stale_segment_id,),
        )
        assert run_versus_contract_checks(database)[
            "versus_survivor_class_total_mismatches"
        ] == 1
        database.execute(
            "UPDATE lps_versus_survivor_stats SET human_special_kills = 3 "
            "WHERE segment_id = ?",
            (stale_segment_id,),
        )

        database.execute(
            "UPDATE lps_versus_infected_stats SET spawn_count = 5 "
            "WHERE segment_id = ?",
            (stale_infected_segment_id,),
        )
        assert run_versus_contract_checks(database)[
            "versus_infected_class_total_mismatches"
        ] == 1
        database.execute(
            "UPDATE lps_versus_infected_stats SET spawn_count = 4 "
            "WHERE segment_id = ?",
            (stale_infected_segment_id,),
        )
        assert all(
            value == 0
            for value in run_versus_contract_checks(database).values()
        )

        active_lifecycle_rows = sum(
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
        assert active_lifecycle_rows == 0, active_lifecycle_rows
    finally:
        database.close()

    print(
        f"SQLite integration passed: {len(statements) + len(incident_statements)} statements, "
        f"{len(tables)} tables, {len(indexes)} indexes."
    )


if __name__ == "__main__":
    main()
