from __future__ import annotations

import argparse
import sqlite3
from pathlib import Path

EQUIPMENT_NAMES = {
    1: "Other Firearm", 2: "Pistol", 3: "Dual Pistols", 4: "Magnum",
    5: "SMG", 6: "Silenced SMG", 7: "MP5", 8: "Pump Shotgun",
    9: "Chrome Shotgun", 10: "Auto Shotgun", 11: "SPAS", 12: "M16",
    13: "AK-47", 14: "SCAR", 15: "SG552", 16: "Hunting Rifle",
    17: "Military Sniper", 18: "Scout", 19: "AWP",
    20: "Grenade Launcher", 21: "M60", 22: "Chainsaw",
    23: "Mounted Gun", 24: "Minigun", 25: "Baseball Bat",
    26: "Cricket Bat", 27: "Crowbar", 28: "Electric Guitar",
    29: "Fire Axe", 30: "Frying Pan", 31: "Golf Club", 32: "Katana",
    33: "Knife", 34: "Machete", 35: "Pitchfork", 36: "Shovel",
    37: "Tonfa", 38: "Molotov", 39: "Pipe Bomb", 40: "Vomit Jar",
}


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
        pve_stats = database.execute(
            "SELECT COUNT(*) FROM lps_pve_segment_stats"
        ).fetchone()[0]
        equipment_stats = database.execute(
            "SELECT COUNT(*) FROM lps_pve_segment_equipment_stats"
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
        orphan_pve_stats = database.execute(
            "SELECT COUNT(*) FROM lps_pve_segment_stats p "
            "LEFT JOIN lps_player_segments g ON g.segment_id = p.segment_id "
            "WHERE g.segment_id IS NULL"
        ).fetchone()[0]
        orphan_equipment_stats = database.execute(
            "SELECT COUNT(*) FROM lps_pve_segment_equipment_stats e "
            "LEFT JOIN lps_player_segments g ON g.segment_id = e.segment_id "
            "WHERE g.segment_id IS NULL"
        ).fetchone()[0]
        invalid_pve_stats = database.execute(
            "SELECT COUNT(*) FROM lps_pve_segment_stats WHERE stats_version <> 1 "
            "OR common_kills < 0 OR special_kills < 0 OR tank_kills < 0 "
            "OR witch_kills < 0 OR damage_to_special < 0 OR damage_to_tank < 0 "
            "OR damage_to_witch < 0 OR damage_taken_infected < 0 "
            "OR friendly_fire_to_humans < 0 OR friendly_fire_to_bots < 0 "
            "OR friendly_fire_taken < 0 OR incapacitations < 0 OR deaths < 0 "
            "OR incap_revives < 0 OR ledge_rescues < 0 OR defib_revives < 0 "
            "OR rescues_received < 0 OR medkits_used_self < 0 "
            "OR medkits_used_on_others < 0 OR medkit_healing_self < 0 "
            "OR medkit_healing_others < 0 OR pills_used < 0 "
            "OR adrenaline_used < 0 OR temporary_health_received < 0 "
            "OR chapter_participations < 0 OR chapter_completions_alive < 0 "
            "OR chapter_completions_dead < 0 OR campaign_completions < 0 "
            "OR smoker_kills < 0 OR boomer_kills < 0 OR hunter_kills < 0 "
            "OR spitter_kills < 0 OR jockey_kills < 0 OR charger_kills < 0 "
            "OR damage_to_smoker < 0 OR damage_to_boomer < 0 "
            "OR damage_to_hunter < 0 OR damage_to_spitter < 0 "
            "OR damage_to_jockey < 0 OR damage_to_charger < 0 "
            "OR smoker_controls_received < 0 OR hunter_controls_received < 0 "
            "OR jockey_controls_received < 0 OR charger_controls_received < 0 "
            "OR smoker_controlled_seconds < 0 OR hunter_controlled_seconds < 0 "
            "OR jockey_controlled_seconds < 0 OR charger_controlled_seconds < 0 "
            "OR smoker_saves < 0 OR hunter_saves < 0 OR jockey_saves < 0 "
            "OR charger_saves < 0 OR melee_tongue_self_cuts < 0 "
            "OR tank_rocks_destroyed < 0 OR witch_oneshots < 0 "
            "OR witch_solo_kills < 0 OR tank_encounters < 0 "
            "OR tank_kill_participations < 0 OR witch_encounters < 0 "
            "OR witch_kill_participations < 0 OR incendiary_packs_deployed < 0 "
            "OR explosive_packs_deployed < 0 OR objective_interactions < 0 "
            "OR ammo_pile_uses < 0 OR incapacitated_seconds < 0 "
            "OR ledge_hanging_seconds < 0 "
            "OR black_white_teammates_restored < 0 "
            "OR smoker_kills + boomer_kills + hunter_kills + spitter_kills "
            "+ jockey_kills + charger_kills <> special_kills "
            "OR damage_to_smoker + damage_to_boomer + damage_to_hunter "
            "+ damage_to_spitter + damage_to_jockey + damage_to_charger "
            "<> damage_to_special "
            "OR chapter_completions_alive + chapter_completions_dead "
            "> chapter_participations OR campaign_completions > chapter_participations "
            "OR revision < 0"
        ).fetchone()[0]
        invalid_equipment_stats = database.execute(
            "SELECT COUNT(*) FROM lps_pve_segment_equipment_stats "
            "WHERE stats_version <> 1 OR equipment_id < 1 OR equipment_id > 40 "
            "OR actions < 0 OR common_kills < 0 OR special_kills < 0 "
            "OR tank_kills < 0 OR witch_kills < 0 OR headshot_kills < 0 "
            "OR damage_to_special < 0 OR damage_to_tank < 0 "
            "OR damage_to_witch < 0 OR revision < 0"
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
            f"runs={runs} rounds={rounds} segments={segments} pve_stats={pve_stats} "
            f"equipment_stats={equipment_stats}"
        )
        print(
            f"health integrity={integrity} orphan_rounds={orphan_rounds} "
            f"orphan_segments={orphan_segments} "
            f"segment_run_mismatches={segment_run_mismatches} "
            f"orphan_pve_stats={orphan_pve_stats} "
            f"orphan_equipment_stats={orphan_equipment_stats} "
            f"invalid_pve_stats={invalid_pve_stats} "
            f"invalid_equipment_stats={invalid_equipment_stats} "
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

        print("recent_pve_stats:")
        rows = database.execute(
            "SELECT r.map_name, u.game_mode, p.common_kills, p.special_kills, "
            "p.tank_kills, p.witch_kills, p.damage_to_special, p.damage_to_tank, "
            "p.damage_to_witch, p.damage_taken_infected, "
            "p.friendly_fire_to_humans, p.friendly_fire_to_bots, "
            "p.friendly_fire_taken, p.incapacitations, p.deaths, "
            "p.incap_revives, p.ledge_rescues, p.defib_revives, "
            "p.rescues_received, p.medkits_used_self, "
            "p.medkits_used_on_others, p.medkit_healing_self, "
            "p.medkit_healing_others, p.pills_used, p.adrenaline_used, "
            "p.temporary_health_received, p.chapter_participations, "
            "p.chapter_completions_alive, p.chapter_completions_dead, "
            "p.campaign_completions, p.smoker_kills, p.boomer_kills, "
            "p.hunter_kills, p.spitter_kills, p.jockey_kills, p.charger_kills, "
            "p.damage_to_smoker, p.damage_to_boomer, p.damage_to_hunter, "
            "p.damage_to_spitter, p.damage_to_jockey, p.damage_to_charger, "
            "p.smoker_controls_received, p.hunter_controls_received, "
            "p.jockey_controls_received, p.charger_controls_received, "
            "p.smoker_controlled_seconds, p.hunter_controlled_seconds, "
            "p.jockey_controlled_seconds, p.charger_controlled_seconds, "
            "p.smoker_saves, p.hunter_saves, p.jockey_saves, p.charger_saves, "
            "p.melee_tongue_self_cuts, p.tank_rocks_destroyed, "
            "p.witch_oneshots, p.witch_solo_kills, p.tank_encounters, "
            "p.tank_kill_participations, p.witch_encounters, "
            "p.witch_kill_participations, p.incendiary_packs_deployed, "
            "p.explosive_packs_deployed, p.objective_interactions, "
            "p.ammo_pile_uses, p.incapacitated_seconds, "
            "p.ledge_hanging_seconds, p.black_white_teammates_restored, "
            "p.revision "
            "FROM lps_pve_segment_stats p "
            "JOIN lps_player_segments g ON g.segment_id = p.segment_id "
            "JOIN lps_rounds r ON r.round_id = g.round_id "
            "JOIN lps_runs u ON u.run_id = g.run_id "
            "ORDER BY p.last_saved_at DESC LIMIT 10"
        ).fetchall()
        for row in rows:
            print(
                "  "
                f"map={row[0]!r} mode={row[1]} common={row[2]} special={row[3]} "
                f"tank={row[4]} witch={row[5]} damage_si={row[6]} "
                f"damage_tank={row[7]} damage_witch={row[8]} "
                f"damage_taken={row[9]} ff_human={row[10]} ff_bot={row[11]} "
                f"ff_taken={row[12]} incaps={row[13]} deaths={row[14]} "
                f"revives={row[15]} ledge={row[16]} defib={row[17]} "
                f"rescues_received={row[18]} medkits_self={row[19]} "
                f"medkits_others={row[20]} heal_self={row[21]} "
                f"heal_others={row[22]} pills={row[23]} adrenaline={row[24]} "
                f"temp_health={row[25]} chapters={row[26]} "
                f"chapters_alive={row[27]} chapters_dead={row[28]} "
                f"campaigns={row[29]} si_kills={row[30:36]} "
                f"si_damage={row[36:42]} controls={row[42:46]} "
                f"controlled_seconds={row[46:50]} saves={row[50:54]} "
                f"tongue_self_cuts={row[54]} rocks={row[55]} "
                f"witch_oneshots={row[56]} witch_solo={row[57]} "
                f"tank_encounters={row[58]} tank_participations={row[59]} "
                f"witch_encounters={row[60]} witch_participations={row[61]} "
                f"incendiary_packs={row[62]} explosive_packs={row[63]} "
                f"objectives={row[64]} ammo_pile={row[65]} "
                f"incap_seconds={row[66]} ledge_seconds={row[67]} "
                f"black_white_restored={row[68]} revision={row[69]}"
            )

        print("recent_equipment_stats:")
        rows = database.execute(
            "SELECT r.map_name, u.game_mode, e.equipment_id, e.actions, "
            "e.common_kills, e.special_kills, e.tank_kills, e.witch_kills, "
            "e.headshot_kills, e.damage_to_special, e.damage_to_tank, "
            "e.damage_to_witch, e.revision "
            "FROM lps_pve_segment_equipment_stats e "
            "JOIN lps_player_segments g ON g.segment_id = e.segment_id "
            "JOIN lps_rounds r ON r.round_id = g.round_id "
            "JOIN lps_runs u ON u.run_id = g.run_id "
            "ORDER BY e.last_saved_at DESC, e.segment_id DESC, e.equipment_id "
            "LIMIT 30"
        ).fetchall()
        for row in rows:
            print(
                "  "
                f"map={row[0]!r} mode={row[1]} equipment="
                f"{EQUIPMENT_NAMES.get(row[2], f'Unknown({row[2]})')!r} "
                f"actions={row[3]} common={row[4]} special={row[5]} "
                f"tank={row[6]} witch={row[7]} headshot={row[8]} "
                f"damage_si={row[9]} damage_tank={row[10]} "
                f"damage_witch={row[11]} revision={row[12]}"
            )
    finally:
        database.close()


if __name__ == "__main__":
    main()
