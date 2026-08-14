#if defined _l4d2_player_stats_pve_combat_included
	#endinput
#endif
#define _l4d2_player_stats_pve_combat_included

public void LPS_Event_InfectedDeathStats(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordPvEEvents())
	{
		return;
	}

	int attacker = GetClientOfUserId(event.GetInt("attacker"));
	LPS_AddPvEStat(attacker, LPSPvEStat_CommonKills, 1);
	LPS_RecordEquipmentKill(
		event.GetInt("infected_id"),
		attacker,
		event.GetInt("weapon_id"),
		"",
		event.GetBool("minigun"),
		LPSEquipmentStat_CommonKills,
		event.GetBool("headshot")
	);
}

public void LPS_Event_PlayerDeathStats(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordPvEEvents())
	{
		return;
	}

	int victim = GetClientOfUserId(event.GetInt("userid"));
	if (victim < 1 || victim > MaxClients || !IsClientInGame(victim))
	{
		return;
	}

	int victimTeam = GetClientTeam(victim);
	if (victimTeam == LPS_TEAM_SURVIVOR)
	{
		LPS_EndControl(victim, event.GetBool("abort") ? LPSIncidentEnd_Released : LPSIncidentEnd_TargetDied);
		if (!event.GetBool("abort"))
		{
			LPS_AddPvEStat(victim, LPSPvEStat_Deaths, 1);
		}
		return;
	}

	if (victimTeam != LPS_TEAM_INFECTED)
	{
		return;
	}

	int attacker = GetClientOfUserId(event.GetInt("attacker"));
	int zombieClass = LPS_GetZombieClass(victim);
	bool aborted = event.GetBool("abort");
	LPS_OnSpecialInfectedDeathForExtendedStats(victim, attacker, aborted);
	LPS_ResolvePvESpecialAssists(victim, attacker, zombieClass, aborted);
	if (aborted)
	{
		if (zombieClass == 8)
		{
			LPS_ResetBossTracking(victim);
		}
		return;
	}

	char weaponName[64];
	event.GetString("weapon", weaponName, sizeof(weaponName));
	if (zombieClass == 8)
	{
		LPS_RecordBossDeathIncident(victim, attacker, true);
		LPS_AddPvEStat(attacker, LPSPvEStat_TankKills, 1);
		LPS_RecordEquipmentKill(
			victim,
			attacker,
			event.GetInt("weaponid"),
			weaponName,
			false,
			LPSEquipmentStat_TankKills,
			event.GetBool("headshot")
		);
		LPS_ResolveBossParticipation(victim, true, attacker, false);
	}
	else if (zombieClass >= 1 && zombieClass <= 6)
	{
		LPS_AddPvEStat(attacker, LPSPvEStat_SpecialKills, 1);
		LPS_AddSpecialClassKill(attacker, zombieClass);
		LPS_RecordEquipmentKill(
			victim,
			attacker,
			event.GetInt("weaponid"),
			weaponName,
			false,
			LPSEquipmentStat_SpecialKills,
			event.GetBool("headshot")
		);
	}
}

public void LPS_Event_WitchKilledStats(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordPvEEvents())
	{
		LPS_RecordVersusWitchKilled(event);
		return;
	}

	int attacker = GetClientOfUserId(event.GetInt("userid"));
	LPS_AddPvEStat(attacker, LPSPvEStat_WitchKills, 1);
	int witch = event.GetInt("witchid");
	LPS_RecordBossDeathIncident(witch, attacker, false, event.GetBool("oneshot"));
	LPS_RecordEquipmentKill(
		witch,
		attacker,
		0,
		"",
		false,
		LPSEquipmentStat_WitchKills,
		false
	);
	LPS_ResolveBossParticipation(witch, false, attacker, event.GetBool("oneshot"));
}

public void LPS_Event_PlayerIncapacitatedStats(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordPvEEvents())
	{
		return;
	}

	int victim = GetClientOfUserId(event.GetInt("userid"));
	LPS_AddPvEStat(victim, LPSPvEStat_Incapacitations, 1);
}
