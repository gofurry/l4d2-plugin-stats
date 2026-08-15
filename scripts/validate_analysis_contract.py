from __future__ import annotations

import re
import sqlite3
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
DEFINITIONS = ROOT / "collector" / "include" / "l4d2_player_stats" / "definitions.inc"
MIGRATION = ROOT / "database" / "migrations" / "sqlite" / "0004_analysis_foundation.sql"

INCIDENT_IDS = {
    "LPSIncident_SurvivorControl": 1,
    "LPSIncident_SurvivorIncap": 2,
    "LPSIncident_SurvivorDeath": 3,
    "LPSIncident_SurvivorRevive": 4,
    "LPSIncident_LedgeRescue": 5,
    "LPSIncident_DefibRevive": 6,
    "LPSIncident_CarAlarm": 7,
    "LPSIncident_TankSpawn": 8,
    "LPSIncident_TankDeath": 9,
    "LPSIncident_WitchSpawn": 10,
    "LPSIncident_WitchDeath": 11,
    "LPSIncident_WitchStartle": 12,
    "LPSIncident_MedkitHeal": 13,
    "LPSIncident_ObjectiveComplete": 14,
}


def require_define(source: str, name: str, value: int) -> None:
    if not re.search(rf"^#define\s+{re.escape(name)}\s+{value}\s*$", source, re.MULTILINE):
        raise AssertionError(f"{name} must remain {value}")


def main() -> None:
    definitions = DEFINITIONS.read_text(encoding="utf-8")
    require_define(definitions, "LPS_SCHEMA_VERSION", 6)
    require_define(definitions, "LPS_INCIDENT_VERSION", 1)
    require_define(definitions, "LPS_CONTEXT_VERSION", 1)
    require_define(definitions, "LPS_ANALYSIS_FLUSH_INCIDENT_LIMIT", 256)
    for symbol, value in INCIDENT_IDS.items():
        if not re.search(rf"\b{symbol}\s*=\s*{value}\b", definitions):
            raise AssertionError(f"{symbol} must remain {value}")

    database = sqlite3.connect(":memory:")
    try:
        database.execute("CREATE TABLE lps_rounds (round_id TEXT PRIMARY KEY, server_key TEXT, started_at INTEGER, map_name TEXT)")
        statements = [
            statement.strip()
            for statement in MIGRATION.read_text(encoding="utf-8").split("-- statement-breakpoint")
            if statement.strip()
        ]
        for statement in statements:
            database.execute(statement)

        tables = {row[0] for row in database.execute("SELECT name FROM sqlite_master WHERE type='table'")}
        indexes = {row[0] for row in database.execute("SELECT name FROM sqlite_master WHERE type='index'")}
        assert {"lps_round_contexts", "lps_incidents"} <= tables
        assert {
            "lps_idx_round_contexts_saved",
            "lps_idx_incidents_occurred",
            "lps_idx_incidents_actor_time",
            "lps_idx_incidents_target_time",
            "lps_idx_rounds_server_started_map",
        } <= indexes

        database.execute(
            "INSERT INTO lps_round_contexts "
            "(round_id,context_version,captured_at,last_saved_at,incident_capture_enabled,"
            "incident_capture_complete,incident_expected_count,incident_dropped_count) "
            "VALUES ('r',1,1,2,1,1,1,0)"
        )
        incident = (
            "INSERT INTO lps_incidents "
            "(round_id,incident_seq,incident_version,incident_type,occurred_at,round_offset_ms) "
            "VALUES ('r',1,1,7,2,100) ON CONFLICT(round_id,incident_seq) DO NOTHING"
        )
        database.execute(incident)
        database.execute(incident)
        assert database.execute("SELECT COUNT(*) FROM lps_incidents").fetchone() == (1,)
    finally:
        database.close()

    print("Analysis contracts validated: context v1, incident v1, schema 6, capture limit 256.")


if __name__ == "__main__":
    main()
