from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent


def require(text: str, needle: str, label: str) -> None:
    if needle not in text:
        raise AssertionError(f"missing v1.3.5 contract path: {label}")


def main() -> None:
    definitions = (ROOT / "collector/include/l4d2_player_stats/definitions.inc").read_text(encoding="utf-8")
    config = (ROOT / "collector/include/l4d2_player_stats/config.inc").read_text(encoding="utf-8")
    techniques = (ROOT / "collector/include/l4d2_player_stats/survivor_techniques.inc").read_text(encoding="utf-8")
    chat = (ROOT / "collector/include/l4d2_player_stats/chat_audit.inc").read_text(encoding="utf-8")
    extended = (ROOT / "collector/include/l4d2_player_stats/pve_extended.inc").read_text(encoding="utf-8")
    damage = (ROOT / "collector/include/l4d2_player_stats/pve_damage.inc").read_text(encoding="utf-8")
    equipment = (ROOT / "collector/include/l4d2_player_stats/equipment_stats.inc").read_text(encoding="utf-8")
    pve_state = (ROOT / "collector/include/l4d2_player_stats/pve_stats_state.inc").read_text(encoding="utf-8")
    versus = (ROOT / "collector/include/l4d2_player_stats/versus_stats.inc").read_text(encoding="utf-8")

    for needle, label in (
        ('#define LPS_SCHEMA_VERSION 7', "Stats schema 7"),
        ('#define LPS_CHAT_QUEUE_LIMIT 1024', "bounded chat queue"),
        ('#define LPS_CHAT_FLUSH_BATCH_LIMIT 64', "bounded chat batch"),
        ('#define LPS_CHAT_TRANSPORT_RETENTION_SECONDS 259200', "72-hour outbox"),
    ):
        require(definitions, needle, label)

    for metric in ("teammateProtections", "ledgeGrabs", "tankRockHitsReceived", "hunterSkeets", "chargerLevels"):
        require(pve_state, metric, f"PvE snapshot {metric}")
        require(versus, metric, f"Versus survivor snapshot {metric}")

    for needle, label in (
        ('event.GetInt("award") != 67', "engine protection award 67"),
        ('g_LPSLedgeTimerActive[client]', "PvE ledge transition reuse"),
        ('HookEvent("charger_impact"', "Charger impact invalidation"),
        ('LPS_ResolveKillEquipment(', "last-damage-aware equipment classification"),
        ('LPS_IsMeleeEquipment(equipment)', "official melee gate"),
        ('LPS_InvalidateHunterEpisode', "Hunter episode invalidation"),
        ('LPS_InvalidateChargerEpisode', "Charger episode invalidation"),
        ('LPS_OnSurvivorTechniqueGameFrame', "bounded pre-death technique sampling"),
        ('g_LPSHunterAirborneConfirmed', "latched Hunter airborne proof"),
        ('g_LPSHunterLastAirborneAt', "recent Hunter airborne proof"),
        ('m_hGroundEntity', "Hunter airborne engine state"),
        ('g_LPSChargerChargingConfirmed', "latched Charger charge proof"),
        ('g_LPSChargerLastChargingAt', "recent Charger charge proof"),
        ('LPS_OnSurvivorTechniqueDamagePre', "pre-damage Charger state capture"),
        ('g_LPSChargerMeleeDamagePending', "bounded Charger melee hit candidate"),
        ('g_LPSChargerMeleeEquipment', "hit-time Charger melee classification"),
        ('m_customAbility', "Charger ability entity lookup"),
        ('HookEvent("player_hurt"', "pre-death lethal damage latch"),
        ('g_LPSHunterLethalCandidate', "Hunter lethal damage proof"),
        ('g_LPSChargerLethalCandidate', "Charger lethal damage proof"),
        ('m_isCharging', "Charger charge engine state"),
        ('LPS_TECHNIQUE_CONFIRM_GRACE', "bounded death-state grace"),
        ('PrintToServer("[LPS technique]', "administrator-visible technique diagnostics"),
    ):
        require(techniques, needle, label)
    if 'HasEntProp(client, Prop_Send, "m_isCharging")' in techniques:
        raise AssertionError("Charger charge state must be read from m_customAbility, not the player entity")
    require(techniques, 'HasEntProp(ability, Prop_Send, "m_isCharging")', "ability-owned Charger charge state")
    require(damage, "LPS_OnSurvivorTechniqueDamagePre(victim, attacker, inflictor)", "SDK pre-damage technique hook")
    require(damage, "LPS_OnSurvivorTechniqueDamagePost(victim)", "SDK post-damage candidate cleanup")
    require(config, '"sm_lps_technique_debug"', "opt-in bounded validation diagnostics")
    require(equipment, "equipment >= LPSEquipment_BaseballBat && equipment <= LPSEquipment_Tonfa", "fixed official melee range")

    for needle, label in (
        ('AddCommandListener(LPS_Command_ChatAudit, "say")', "say capture"),
        ('AddCommandListener(LPS_Command_ChatAudit, "say_team")', "say_team capture"),
        ('IsFakeClient(client)', "Bot author rejection"),
        ("g_LPSChatSeq++;", "sequence before queue admission"),
        ('g_LPSChatQueue.Length >= LPS_CHAT_QUEUE_LIMIT', "queue overflow accounting"),
        ("message.commandLike = content[0] == '!' || content[0] == '/'", "command-like classification"),
        ('ON CONFLICT(message_id) DO NOTHING', "idempotent outbox insert"),
        ('DBPrio_Low', "failure-isolated low-priority chat transaction"),
    ):
        require(chat, needle, label)
    if "LPS_CanRecordPvEEvents" in chat or "LPS_CanRecordVersusEvents" in chat:
        raise AssertionError("chat capture must remain independent of gameplay whitelist")

    for needle, label in (
        ("actualDamage <= 0", "positive effective rock damage"),
        ("LPS_MarkTankRockVictim(rock, victim)", "rock/victim deduplication"),
        ("LPSPvEStat_TankRockHitsReceived", "PvE rock hit snapshot"),
        ("LPSVersusSurvivorStat_TankRockHitsReceived", "Versus rock hit snapshot"),
    ):
        require(extended, needle, label)
    require(damage, "LPS_RecordTankRockHitReceived(victim, attacker, inflictor, actualDamage)", "effective damage hook")

    print("v1.3.5 collector contract validated: telemetry state machines, bounded chat outbox, and rock deduplication.")


if __name__ == "__main__":
    main()
