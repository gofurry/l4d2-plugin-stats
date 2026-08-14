#if defined _l4d2_player_stats_pve_damage_included
	#endinput
#endif
#define _l4d2_player_stats_pve_damage_included

void LPS_HookStatsClient(int client)
{
	if (client < 1 || client > MaxClients || !IsClientInGame(client))
	{
		return;
	}

	LPS_HookStatsDamageEntity(client);
}

void LPS_HookStatsDamageEntity(int entity)
{
	if (entity < 1
		|| entity > LPS_MAX_ENTITY_INDEX
		|| !IsValidEntity(entity)
		|| g_LPSDamageHooked[entity])
	{
		return;
	}

	SDKHook(entity, SDKHook_OnTakeDamageAlive, LPS_OnTakeDamageAlivePre);
	SDKHook(entity, SDKHook_OnTakeDamageAlivePost, LPS_OnTakeDamageAlivePost);
	g_LPSDamageHooked[entity] = true;
}

void LPS_ResetStatsDamageEntity(int entity)
{
	if (entity < 1 || entity > LPS_MAX_ENTITY_INDEX)
	{
		return;
	}

	g_LPSDamageHooked[entity] = false;
	g_LPSDamageHealthCaptured[entity] = false;
	g_LPSDamageVictimWasIncapacitated[entity] = false;
	g_LPSDamageHealthBefore[entity] = 0.0;
	if (entity <= MaxClients)
	{
		g_LPSClientEffectiveHealthBeforeCommand[entity] = 0.0;
	}
}

public Action LPS_OnTakeDamageAlivePre(
	int victim,
	int &attacker,
	int &inflictor,
	float &damage,
	int &damageType
)
{
	if (!LPS_CanCaptureStatsDamage(victim, damage))
	{
		return Plugin_Continue;
	}

	float health = LPS_GetTrackedEffectiveHealth(victim);
	if (health <= 0.0)
	{
		return Plugin_Continue;
	}

	g_LPSDamageHealthBefore[victim] = health;
	g_LPSDamageVictimWasIncapacitated[victim] = LPS_IsSurvivorIncapacitated(victim);
	g_LPSDamageHealthCaptured[victim] = true;
	return Plugin_Continue;
}

public void LPS_OnTakeDamageAlivePost(
	int victim,
	int attacker,
	int inflictor,
	float damage,
	int damageType
)
{
	if (victim < 1
		|| victim > LPS_MAX_ENTITY_INDEX
		|| !g_LPSDamageHealthCaptured[victim])
	{
		return;
	}

	float before = g_LPSDamageHealthBefore[victim];
	bool wasIncapacitated = g_LPSDamageVictimWasIncapacitated[victim];
	g_LPSDamageHealthCaptured[victim] = false;
	g_LPSDamageVictimWasIncapacitated[victim] = false;
	g_LPSDamageHealthBefore[victim] = 0.0;

	if (!LPS_CanCaptureStatsDamage(victim, damage)
		|| LPS_IsExcludedCombatDamage(damageType)
		|| attacker == victim)
	{
		return;
	}

	float after = LPS_GetTrackedEffectiveHealth(victim);
	float lost = before - after;
	int actualDamage = lost > 0.25 ? RoundToNearest(lost) : 0;

	// Entering the incapacitated state replaces the standing health pool with
	// incap health. In that one transition, before-after is not meaningful, so
	// cap the engine's applied damage by the pre-hit effective health instead.
	if (actualDamage <= 0
		&& !wasIncapacitated
		&& LPS_IsSurvivorIncapacitated(victim))
	{
		actualDamage = RoundToCeil(damage < before ? damage : before);
	}

	int maximumLoss = RoundToCeil(before);
	if (actualDamage > maximumLoss)
	{
		actualDamage = maximumLoss;
	}
	if (actualDamage <= 0)
	{
		return;
	}

	if (g_LPSModeFamily == LPSModeFamily_PvE)
	{
		LPS_RecordPvEDamage(victim, attacker, inflictor, actualDamage);
	}
	else if (g_LPSModeFamily == LPSModeFamily_Versus)
	{
		LPS_RecordVersusDamage(victim, attacker, inflictor, actualDamage);
	}
}

bool LPS_IsSurvivorIncapacitated(int client)
{
	return client >= 1
		&& client <= MaxClients
		&& IsClientInGame(client)
		&& GetClientTeam(client) == LPS_TEAM_SURVIVOR
		&& HasEntProp(client, Prop_Send, "m_isIncapacitated")
		&& GetEntProp(client, Prop_Send, "m_isIncapacitated") != 0;
}

bool LPS_CanCaptureStatsDamage(int victim, float damage)
{
	bool supportedStatsMode = g_LPSModeFamily == LPSModeFamily_PvE
		|| (g_LPSModeFamily == LPSModeFamily_Versus
			&& g_LPSVersusStatsEnabled.BoolValue);
	if (!supportedStatsMode
		|| !g_LPSRunActive
		|| !g_LPSRoundActive
		|| damage <= 0.0
		|| victim < 1
		|| victim > LPS_MAX_ENTITY_INDEX)
	{
		return false;
	}

	if (victim <= MaxClients)
	{
		return IsClientInGame(victim)
			&& (GetClientTeam(victim) == LPS_TEAM_SURVIVOR
				|| GetClientTeam(victim) == LPS_TEAM_INFECTED);
	}

	return (g_LPSModeFamily == LPSModeFamily_PvE
			|| (g_LPSModeFamily == LPSModeFamily_Versus
				&& g_LPSVersusStatsEnabled.BoolValue))
		&& LPS_IsWitchEntity(victim);
}

float LPS_GetTrackedEffectiveHealth(int entity)
{
	if (entity < 1 || entity > LPS_MAX_ENTITY_INDEX || !IsValidEntity(entity))
	{
		return 0.0;
	}

	if (entity <= MaxClients)
	{
		if (!IsClientInGame(entity))
		{
			return 0.0;
		}

		float health = float(GetClientHealth(entity));
		if (GetClientTeam(entity) != LPS_TEAM_SURVIVOR
			|| !HasEntProp(entity, Prop_Send, "m_healthBuffer"))
		{
			return health;
		}

		float buffer = GetEntPropFloat(entity, Prop_Send, "m_healthBuffer");
		if (buffer <= 0.0)
		{
			return health;
		}

		if (g_LPSPainPillsDecayRate != null
			&& HasEntProp(entity, Prop_Send, "m_healthBufferTime"))
		{
			float elapsed = GetGameTime() - GetEntPropFloat(entity, Prop_Send, "m_healthBufferTime");
			if (elapsed > 0.0)
			{
				buffer -= elapsed * g_LPSPainPillsDecayRate.FloatValue;
			}
		}

		if (buffer < 0.0)
		{
			buffer = 0.0;
		}
		return health + buffer;
	}

	if (HasEntProp(entity, Prop_Data, "m_iHealth"))
	{
		return float(GetEntProp(entity, Prop_Data, "m_iHealth"));
	}
	return 0.0;
}

bool LPS_IsExcludedCombatDamage(int damageType)
{
	int excluded = DMG_CRUSH
		| DMG_VEHICLE
		| DMG_FALL
		| DMG_DROWN
		| DMG_DROWNRECOVER
		| DMG_DISSOLVE;
	return (damageType & excluded) != 0;
}

void LPS_RecordPvEDamage(int victim, int attacker, int inflictor, int actualDamage)
{
	int humanAttacker = LPS_ResolveHumanSurvivorAttacker(attacker, inflictor);

	if (victim <= MaxClients && IsClientInGame(victim))
	{
		int victimTeam = GetClientTeam(victim);
		if (victimTeam == LPS_TEAM_INFECTED && humanAttacker > 0)
		{
			int zombieClass = LPS_GetZombieClass(victim);
			if (zombieClass == 8)
			{
				LPS_AddPvEStat(humanAttacker, LPSPvEStat_DamageToTank, actualDamage);
				LPS_RecordEquipmentDamage(
					victim,
					humanAttacker,
					attacker,
					inflictor,
					LPSEquipmentStat_DamageToTank,
					actualDamage
				);
				LPS_RecordBossContribution(victim, humanAttacker);
			}
			else if (zombieClass >= 1 && zombieClass <= 6)
			{
				LPS_RecordAssistContribution(victim, humanAttacker);
				LPS_AddPvEStat(humanAttacker, LPSPvEStat_DamageToSpecial, actualDamage);
				LPS_AddSpecialClassDamage(humanAttacker, zombieClass, actualDamage);
				LPS_RecordEquipmentDamage(
					victim,
					humanAttacker,
					attacker,
					inflictor,
					LPSEquipmentStat_DamageToSpecial,
					actualDamage
				);
			}
			return;
		}

		if (victimTeam != LPS_TEAM_SURVIVOR)
		{
			return;
		}

		if (humanAttacker > 0 && humanAttacker != victim)
		{
			if (IsFakeClient(victim))
			{
				LPS_AddPvEStat(humanAttacker, LPSPvEStat_FriendlyFireToBots, actualDamage);
			}
			else
			{
				LPS_RecordRelationshipFriendlyFire(humanAttacker, victim, actualDamage);
				LPS_AddPvEStat(humanAttacker, LPSPvEStat_FriendlyFireToHumans, actualDamage);
				LPS_AddPvEStat(victim, LPSPvEStat_FriendlyFireTaken, actualDamage);
			}
			return;
		}

		if (LPS_IsInfectedDamageSource(attacker) || LPS_IsInfectedDamageSource(inflictor))
		{
			LPS_AddPvEStat(victim, LPSPvEStat_DamageTakenInfected, actualDamage);
		}
		return;
	}

	if (humanAttacker > 0 && LPS_IsWitchEntity(victim))
	{
		LPS_AddPvEStat(humanAttacker, LPSPvEStat_DamageToWitch, actualDamage);
		LPS_RecordEquipmentDamage(
			victim,
			humanAttacker,
			attacker,
			inflictor,
			LPSEquipmentStat_DamageToWitch,
			actualDamage
		);
		LPS_RecordBossContribution(victim, humanAttacker);
	}
}

int LPS_ResolveHumanSurvivorAttacker(int attacker, int inflictor)
{
	if (LPS_IsHumanSurvivorClient(attacker))
	{
		return attacker;
	}

	int owner = LPS_GetEntityPlayerOwner(attacker);
	if (LPS_IsHumanSurvivorClient(owner))
	{
		return owner;
	}

	owner = LPS_GetEntityPlayerOwner(inflictor);
	return LPS_IsHumanSurvivorClient(owner) ? owner : 0;
}

int LPS_GetEntityPlayerOwner(int entity)
{
	if (entity < 1 || entity > LPS_MAX_ENTITY_INDEX || !IsValidEntity(entity))
	{
		return 0;
	}

	int owner = 0;
	if (HasEntProp(entity, Prop_Send, "m_hOwnerEntity"))
	{
		owner = GetEntPropEnt(entity, Prop_Send, "m_hOwnerEntity");
	}
	else if (HasEntProp(entity, Prop_Data, "m_hOwnerEntity"))
	{
		owner = GetEntPropEnt(entity, Prop_Data, "m_hOwnerEntity");
	}

	if (owner >= 1 && owner <= MaxClients)
	{
		return owner;
	}

	if (HasEntProp(entity, Prop_Send, "m_hThrower"))
	{
		owner = GetEntPropEnt(entity, Prop_Send, "m_hThrower");
	}
	else if (HasEntProp(entity, Prop_Data, "m_hThrower"))
	{
		owner = GetEntPropEnt(entity, Prop_Data, "m_hThrower");
	}

	return owner >= 1 && owner <= MaxClients ? owner : 0;
}

bool LPS_IsHumanSurvivorClient(int client)
{
	return client >= 1
		&& client <= MaxClients
		&& IsClientInGame(client)
		&& !IsFakeClient(client)
		&& GetClientTeam(client) == LPS_TEAM_SURVIVOR;
}

bool LPS_IsInfectedDamageSource(int entity)
{
	if (entity < 1 || entity > LPS_MAX_ENTITY_INDEX || !IsValidEntity(entity))
	{
		return false;
	}

	if (entity <= MaxClients)
	{
		return IsClientInGame(entity) && GetClientTeam(entity) == LPS_TEAM_INFECTED;
	}

	char classname[64];
	GetEntityClassname(entity, classname, sizeof(classname));
	return StrEqual(classname, "infected", true)
		|| StrEqual(classname, "witch", true)
		|| StrEqual(classname, "witch_bride", true)
		|| StrEqual(classname, "tank_rock", true)
		|| StrEqual(classname, "insect_swarm", true);
}

bool LPS_IsWitchEntity(int entity)
{
	if (entity <= MaxClients
		|| entity < 1
		|| entity > LPS_MAX_ENTITY_INDEX
		|| !IsValidEntity(entity))
	{
		return false;
	}

	char classname[32];
	GetEntityClassname(entity, classname, sizeof(classname));
	return StrEqual(classname, "witch", true) || StrEqual(classname, "witch_bride", true);
}

int LPS_GetZombieClass(int client)
{
	if (client < 1
		|| client > MaxClients
		|| !IsClientInGame(client)
		|| !HasEntProp(client, Prop_Send, "m_zombieClass"))
	{
		return 0;
	}
	return GetEntProp(client, Prop_Send, "m_zombieClass");
}

void LPS_AddSpecialClassDamage(int client, int zombieClass, int amount)
{
	switch (zombieClass)
	{
		case 1: LPS_AddPvEStat(client, LPSPvEStat_DamageToSmoker, amount);
		case 2: LPS_AddPvEStat(client, LPSPvEStat_DamageToBoomer, amount);
		case 3: LPS_AddPvEStat(client, LPSPvEStat_DamageToHunter, amount);
		case 4: LPS_AddPvEStat(client, LPSPvEStat_DamageToSpitter, amount);
		case 5: LPS_AddPvEStat(client, LPSPvEStat_DamageToJockey, amount);
		case 6: LPS_AddPvEStat(client, LPSPvEStat_DamageToCharger, amount);
	}
}

void LPS_AddSpecialClassKill(int client, int zombieClass)
{
	switch (zombieClass)
	{
		case 1: LPS_AddPvEStat(client, LPSPvEStat_SmokerKills, 1);
		case 2: LPS_AddPvEStat(client, LPSPvEStat_BoomerKills, 1);
		case 3: LPS_AddPvEStat(client, LPSPvEStat_HunterKills, 1);
		case 4: LPS_AddPvEStat(client, LPSPvEStat_SpitterKills, 1);
		case 5: LPS_AddPvEStat(client, LPSPvEStat_JockeyKills, 1);
		case 6: LPS_AddPvEStat(client, LPSPvEStat_ChargerKills, 1);
	}
}
