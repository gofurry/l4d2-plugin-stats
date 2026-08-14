from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


def read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


def function_body(source: str, signature: str, next_signature: str) -> str:
    start = source.index(signature)
    end = source.index(next_signature, start)
    return source[start:end]


def require(body: str, fragments: list[str], name: str) -> None:
    for fragment in fragments:
        if fragment not in body:
            raise AssertionError(f"{name} is missing frozen fragment: {fragment}")


pve_source = read("collector/include/l4d2_player_stats/pve_combat.inc")
pve_death = function_body(
    pve_source,
    "public void LPS_Event_PlayerDeathStats",
    "public void LPS_Event_WitchKilledStats",
)
require(
    pve_death,
    [
        'if (!event.GetBool("abort"))',
        "LPS_AddPvEStat(victim, LPSPvEStat_Deaths, 1);",
        "LPS_IsHumanClientOnTeam(victim, LPS_TEAM_SURVIVOR)",
        '(event.GetInt("type") & DMG_FALL) != 0',
        "LPS_AddPvEStat(victim, LPSPvEStat_FallDeaths, 1);",
    ],
    "PvE fall-death path",
)
if pve_death.count("LPSPvEStat_FallDeaths") != 1:
    raise AssertionError("PvE fall death must have exactly one increment path")
if pve_death.index('if (!event.GetBool("abort"))') > pve_death.index("LPSPvEStat_FallDeaths"):
    raise AssertionError("PvE fall death must remain inside the non-abort branch")

versus_source = read("collector/include/l4d2_player_stats/versus_stats.inc")
versus_death = function_body(
    versus_source,
    "public void LPS_Event_VersusPlayerDeathStats",
    "public void LPS_Event_VersusPlayerIncapacitatedStats",
)
require(
    versus_death,
    [
        'bool aborted = event.GetBool("abort");',
        "if (aborted)",
        "LPS_AddVersusSurvivorStat(victim, LPSVersusSurvivorStat_Deaths, 1);",
        "LPS_IsHumanClientOnTeam(victim, LPS_TEAM_SURVIVOR)",
        '(event.GetInt("type") & DMG_FALL) != 0',
        "LPS_AddVersusSurvivorStat(victim, LPSVersusSurvivorStat_FallDeaths, 1);",
    ],
    "Versus fall-death path",
)
if versus_death.count("LPSVersusSurvivorStat_FallDeaths") != 1:
    raise AssertionError("Versus fall death must have exactly one increment path")
if versus_death.index("if (aborted)") > versus_death.index("LPSVersusSurvivorStat_FallDeaths"):
    raise AssertionError("Versus abort guard must precede fall-death accounting")

for relative in [
    "collector/include/l4d2_player_stats/pve_persistence.inc",
    "collector/include/l4d2_player_stats/versus_stats.inc",
]:
    if "fall_deaths" not in read(relative):
        raise AssertionError(f"{relative} does not persist fall_deaths")

incident_source = read("collector/include/l4d2_player_stats/incidents.inc").lower()
if "fall_death" in incident_source or "falldeath" in incident_source:
    raise AssertionError("fall deaths must not introduce a new Incident type")

route_source = read("dashboard/internal/server/routes_achievements.go").lower()
for forbidden in ["refresh", "rebuild", "claim"]:
    if forbidden in route_source:
        raise AssertionError(f"achievement routes must not expose {forbidden}")

print("Achievement Contract v1 static paths validated: automatic-only evaluation and matched PvE/Versus fall deaths.")
