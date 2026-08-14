#if defined _l4d2_player_stats_versus_abilities_included
	#endinput
#endif
#define _l4d2_player_stats_versus_abilities_included

LPSControlType g_LPSVersusControlType[MAXPLAYERS + 1];
float g_LPSVersusControlStartedAt[MAXPLAYERS + 1];
int g_LPSVersusControlControllerUserId[MAXPLAYERS + 1];
int g_LPSVersusControlZombieClass[MAXPLAYERS + 1];
bool g_LPSVersusControlVictimIsBot[MAXPLAYERS + 1];
bool g_LPSVersusChargerPummeling[MAXPLAYERS + 1];

// An acid pool can continue hurting survivors after its Spitter dies. Capture
// the original user ID while the owner entity is still available.
int g_LPSVersusSpitOwnerUserId[LPS_MAX_ENTITY_INDEX + 1];
int g_LPSVersusSpitOwnerSegmentGeneration[LPS_MAX_ENTITY_INDEX + 1];
int g_LPSVersusAbilitySegmentGeneration[MAXPLAYERS + 1];

void LPS_InitializeVersusAbilities()
{
	HookEvent("tongue_grab", LPS_Event_VersusTongueGrab, EventHookMode_Post);
	HookEvent("tongue_release", LPS_Event_VersusTongueRelease, EventHookMode_Post);
	HookEvent("tongue_pull_stopped", LPS_Event_VersusTongueStopped, EventHookMode_Post);
	HookEvent("choke_stopped", LPS_Event_VersusTongueStopped, EventHookMode_Post);
	HookEvent("lunge_pounce", LPS_Event_VersusLungePounce, EventHookMode_Post);
	HookEvent("pounce_end", LPS_Event_VersusPounceEnd, EventHookMode_Post);
	HookEvent("pounce_stopped", LPS_Event_VersusPounceStopped, EventHookMode_Post);
	HookEvent("jockey_ride", LPS_Event_VersusJockeyRide, EventHookMode_Post);
	HookEvent("jockey_ride_end", LPS_Event_VersusJockeyRideEnd, EventHookMode_Post);
	HookEvent("charger_carry_start", LPS_Event_VersusChargerCarryStart, EventHookMode_Post);
	HookEvent("charger_carry_end", LPS_Event_VersusChargerCarryEnd, EventHookMode_Post);
	HookEvent("charger_pummel_start", LPS_Event_VersusChargerPummelStart, EventHookMode_Post);
	HookEvent("charger_pummel_end", LPS_Event_VersusChargerPummelEnd, EventHookMode_Post);
	HookEvent("player_now_it", LPS_Event_VersusPlayerNowIt, EventHookMode_Post);
	HookEvent("spit_burst", LPS_Event_VersusSpitBurst, EventHookMode_Post);

	for (int client = 1; client <= MaxClients; client++)
	{
		LPS_ResetVersusControlVictim(client);
		g_LPSVersusAbilitySegmentGeneration[client] = 0;
	}
	for (int entity = 1; entity <= LPS_MAX_ENTITY_INDEX; entity++)
	{
		g_LPSVersusSpitOwnerUserId[entity] = 0;
		g_LPSVersusSpitOwnerSegmentGeneration[entity] = 0;
	}

	int entity = -1;
	while ((entity = FindEntityByClassname(entity, "insect_swarm")) != -1)
	{
		LPS_CaptureVersusSpitOwnerLater(entity);
	}
}

void LPS_OnVersusAbilityEntityCreated(int entity, const char[] classname)
{
	if (entity < 1 || entity > LPS_MAX_ENTITY_INDEX)
	{
		return;
	}

	g_LPSVersusSpitOwnerUserId[entity] = 0;
	g_LPSVersusSpitOwnerSegmentGeneration[entity] = 0;
	if (StrEqual(classname, "insect_swarm", true))
	{
		LPS_CaptureVersusSpitOwnerLater(entity);
	}
}

void LPS_OnVersusAbilityEntityDestroyed(int entity)
{
	if (entity >= 1 && entity <= LPS_MAX_ENTITY_INDEX)
	{
		g_LPSVersusSpitOwnerUserId[entity] = 0;
		g_LPSVersusSpitOwnerSegmentGeneration[entity] = 0;
	}
}

void LPS_CaptureVersusSpitOwnerLater(int entity)
{
	if (entity > MaxClients && entity <= LPS_MAX_ENTITY_INDEX && IsValidEntity(entity))
	{
		RequestFrame(LPS_Frame_CaptureVersusSpitOwner, EntIndexToEntRef(entity));
	}
}

public void LPS_Frame_CaptureVersusSpitOwner(any entityReference)
{
	int entity = EntRefToEntIndex(entityReference);
	if (entity <= MaxClients || entity > LPS_MAX_ENTITY_INDEX || !IsValidEntity(entity))
	{
		return;
	}

	int owner = LPS_GetEntityPlayerOwner(entity);
	if (LPS_IsHumanClientOnTeam(owner, LPS_TEAM_INFECTED)
		&& LPS_GetZombieClass(owner) == 4
		&& LPS_CanOwnVersusAbilityStats(owner))
	{
		g_LPSVersusSpitOwnerUserId[entity] = GetClientUserId(owner);
		g_LPSVersusSpitOwnerSegmentGeneration[entity] =
			g_LPSVersusAbilitySegmentGeneration[owner];
	}
}

public void LPS_Event_VersusTongueGrab(Event event, const char[] name, bool dontBroadcast)
{
	LPS_StartVersusControl(
		GetClientOfUserId(event.GetInt("victim")),
		GetClientOfUserId(event.GetInt("userid")),
		LPSControlType_Smoker,
		1
	);
}

public void LPS_Event_VersusTongueRelease(Event event, const char[] name, bool dontBroadcast)
{
	LPS_EndVersusControl(GetClientOfUserId(event.GetInt("victim")), LPSControlType_Smoker);
}

public void LPS_Event_VersusTongueStopped(Event event, const char[] name, bool dontBroadcast)
{
	int victim = GetClientOfUserId(event.GetInt("victim"));
	int rescuer = GetClientOfUserId(event.GetInt("userid"));
	LPSIncidentEndReason reason = rescuer == victim
		? LPSIncidentEnd_SelfFreed
		: (rescuer > 0 ? LPSIncidentEnd_Rescued : LPSIncidentEnd_Released);
	LPS_EndVersusControl(victim, LPSControlType_Smoker, reason, rescuer);
}

public void LPS_Event_VersusLungePounce(Event event, const char[] name, bool dontBroadcast)
{
	LPS_StartVersusControl(
		GetClientOfUserId(event.GetInt("victim")),
		GetClientOfUserId(event.GetInt("userid")),
		LPSControlType_Hunter,
		3
	);
}

public void LPS_Event_VersusPounceEnd(Event event, const char[] name, bool dontBroadcast)
{
	LPS_EndVersusControl(GetClientOfUserId(event.GetInt("victim")), LPSControlType_Hunter);
}

public void LPS_Event_VersusPounceStopped(Event event, const char[] name, bool dontBroadcast)
{
	int rescuer = GetClientOfUserId(event.GetInt("userid"));
	LPS_EndVersusControl(GetClientOfUserId(event.GetInt("victim")), LPSControlType_Hunter, rescuer > 0 ? LPSIncidentEnd_Rescued : LPSIncidentEnd_Released, rescuer);
}

public void LPS_Event_VersusJockeyRide(Event event, const char[] name, bool dontBroadcast)
{
	LPS_StartVersusControl(
		GetClientOfUserId(event.GetInt("victim")),
		GetClientOfUserId(event.GetInt("userid")),
		LPSControlType_Jockey,
		5
	);
}

public void LPS_Event_VersusJockeyRideEnd(Event event, const char[] name, bool dontBroadcast)
{
	int rescuer = GetClientOfUserId(event.GetInt("rescuer"));
	LPS_EndVersusControl(GetClientOfUserId(event.GetInt("victim")), LPSControlType_Jockey, rescuer > 0 ? LPSIncidentEnd_Rescued : LPSIncidentEnd_Released, rescuer);
}

public void LPS_Event_VersusChargerCarryStart(Event event, const char[] name, bool dontBroadcast)
{
	int victim = GetClientOfUserId(event.GetInt("victim"));
	if (victim >= 1 && victim <= MaxClients)
	{
		g_LPSVersusChargerPummeling[victim] = false;
	}
	LPS_StartVersusControl(
		victim,
		GetClientOfUserId(event.GetInt("userid")),
		LPSControlType_Charger,
		6
	);
}

public void LPS_Event_VersusChargerCarryEnd(Event event, const char[] name, bool dontBroadcast)
{
	int victim = GetClientOfUserId(event.GetInt("victim"));
	if (victim < 1 || victim > MaxClients)
	{
		return;
	}

	DataPack pack;
	CreateDataTimer(0.1, LPS_Timer_EndVersusChargerCarry, pack, TIMER_FLAG_NO_MAPCHANGE);
	pack.WriteCell(GetClientUserId(victim));
}

public Action LPS_Timer_EndVersusChargerCarry(Handle timer, DataPack pack)
{
	pack.Reset();
	int victim = GetClientOfUserId(pack.ReadCell());
	if (victim >= 1 && victim <= MaxClients && !g_LPSVersusChargerPummeling[victim])
	{
		LPS_EndVersusControl(victim, LPSControlType_Charger);
	}
	return Plugin_Stop;
}

public void LPS_Event_VersusChargerPummelStart(Event event, const char[] name, bool dontBroadcast)
{
	int victim = GetClientOfUserId(event.GetInt("victim"));
	if (victim >= 1 && victim <= MaxClients)
	{
		g_LPSVersusChargerPummeling[victim] = true;
	}
	LPS_StartVersusControl(
		victim,
		GetClientOfUserId(event.GetInt("userid")),
		LPSControlType_Charger,
		6
	);
}

public void LPS_Event_VersusChargerPummelEnd(Event event, const char[] name, bool dontBroadcast)
{
	int victim = GetClientOfUserId(event.GetInt("victim"));
	if (victim >= 1 && victim <= MaxClients)
	{
		g_LPSVersusChargerPummeling[victim] = false;
	}
	LPS_EndVersusControl(victim, LPSControlType_Charger);
}

public void LPS_Event_VersusPlayerNowIt(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordVersusEvents() || !event.GetBool("by_boomer"))
	{
		return;
	}

	int victim = GetClientOfUserId(event.GetInt("userid"));
	int boomer = GetClientOfUserId(event.GetInt("attacker"));
	if (victim < 1
		|| victim > MaxClients
		|| !IsClientInGame(victim)
		|| GetClientTeam(victim) != LPS_TEAM_SURVIVOR
		|| !LPS_IsHumanClientOnTeam(boomer, LPS_TEAM_INFECTED)
		|| LPS_GetZombieClass(boomer) != 2)
	{
		return;
	}

	LPS_AddVersusInfectedAbilityStat(
		boomer,
		2,
		IsFakeClient(victim)
			? LPSVersusInfectedAbilityStat_BotSurvivorAbilityHits
			: LPSVersusInfectedAbilityStat_HumanSurvivorAbilityHits,
		1
	);
}

public void LPS_Event_VersusSpitBurst(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordVersusEvents())
	{
		return;
	}

	int spitter = GetClientOfUserId(event.GetInt("userid"));
	int entity = event.GetInt("subject");
	if (entity > MaxClients
		&& entity <= LPS_MAX_ENTITY_INDEX
		&& LPS_IsVersusSpitEntity(entity)
		&& LPS_CanOwnVersusAbilityStats(spitter))
	{
		g_LPSVersusSpitOwnerUserId[entity] = GetClientUserId(spitter);
		g_LPSVersusSpitOwnerSegmentGeneration[entity] =
			g_LPSVersusAbilitySegmentGeneration[spitter];
	}
}

void LPS_StartVersusControl(
	int victim,
	int controller,
	LPSControlType type,
	int zombieClass
)
{
	if (!LPS_CanRecordVersusEvents()
		|| victim < 1
		|| victim > MaxClients
		|| !IsClientInGame(victim)
		|| GetClientTeam(victim) != LPS_TEAM_SURVIVOR
		|| controller < 1
		|| controller > MaxClients
		|| !IsClientInGame(controller)
		|| GetClientTeam(controller) != LPS_TEAM_INFECTED
		|| LPS_GetZombieClass(controller) != zombieClass
		)
	{
		return;
	}

	int controllerUserId = GetClientUserId(controller);
	if (g_LPSVersusControlType[victim] == type
		&& g_LPSVersusControlControllerUserId[victim] == controllerUserId)
	{
		return;
	}

	LPS_EndVersusControl(victim, LPSControlType_None);
	g_LPSVersusControlType[victim] = type;
	g_LPSVersusControlStartedAt[victim] = GetGameTime();
	g_LPSVersusControlControllerUserId[victim] = controllerUserId;
	g_LPSVersusControlZombieClass[victim] = zombieClass;
	g_LPSVersusControlVictimIsBot[victim] = IsFakeClient(victim);
	LPS_StartControlIncident(victim, controller, type, zombieClass);

	if (LPS_CanOwnVersusAbilityStats(controller))
	{
		LPS_AddVersusInfectedAbilityStat(
			controller,
			zombieClass,
			g_LPSVersusControlVictimIsBot[victim]
				? LPSVersusInfectedAbilityStat_BotSurvivorControls
				: LPSVersusInfectedAbilityStat_HumanSurvivorControls,
			1
		);
	}
}

void LPS_EndVersusControl(int victim, LPSControlType expectedType, LPSIncidentEndReason reason = LPSIncidentEnd_Released, int helper = 0)
{
	if (victim < 1
		|| victim > MaxClients
		|| g_LPSVersusControlType[victim] == LPSControlType_None
		|| (expectedType != LPSControlType_None
			&& g_LPSVersusControlType[victim] != expectedType))
	{
		return;
	}

	int controller = GetClientOfUserId(g_LPSVersusControlControllerUserId[victim]);
	float elapsed = GetGameTime() - g_LPSVersusControlStartedAt[victim];
	int durationMs = elapsed > 0.0 ? RoundToNearest(elapsed * 1000.0) : 0;
	if ((reason == LPSIncidentEnd_Rescued || reason == LPSIncidentEnd_ControllerKilled)
		&& helper > 0)
	{
		LPS_RecordRelationshipControlRescue(
			helper,
			victim,
			g_LPSVersusControlType[victim],
			durationMs
		);
	}
	int seconds = elapsed > 0.0 ? RoundToNearest(elapsed) : 0;
	if (seconds > 0 && LPS_CanOwnVersusAbilityStats(controller))
	{
		LPS_AddVersusInfectedAbilityStat(
			controller,
			g_LPSVersusControlZombieClass[victim],
			g_LPSVersusControlVictimIsBot[victim]
				? LPSVersusInfectedAbilityStat_BotSurvivorControlSeconds
				: LPSVersusInfectedAbilityStat_HumanSurvivorControlSeconds,
			seconds
		);
	}
	LPS_EndControlIncident(victim, g_LPSVersusControlType[victim], reason, helper);
	LPS_ResetVersusControlVictim(victim);
}

void LPS_OnVersusAbilityPlayerDeath(int client, int attacker, bool aborted = false)
{
	if (client < 1 || client > MaxClients)
	{
		return;
	}

	LPS_EndVersusControl(client, LPSControlType_None, aborted ? LPSIncidentEnd_Released : LPSIncidentEnd_TargetDied);
	LPS_EndVersusControlsByControllerUserId(GetClientUserId(client), aborted ? LPSIncidentEnd_Released : LPSIncidentEnd_ControllerKilled, aborted ? 0 : attacker);
}

void LPS_OnClientVersusAbilitySegmentStarted(int client)
{
	if (client < 1 || client > MaxClients)
	{
		return;
	}

	LPS_ResetVersusControlVictim(client);
	g_LPSVersusChargerPummeling[client] = false;
	g_LPSVersusAbilitySegmentGeneration[client]++;
	if (g_LPSVersusAbilitySegmentGeneration[client] <= 0)
	{
		g_LPSVersusAbilitySegmentGeneration[client] = 1;
	}
}

void LPS_OnClientVersusAbilitySegmentEnding(int client)
{
	if (client < 1 || client > MaxClients)
	{
		return;
	}

	LPS_EndVersusControl(client, LPSControlType_None, LPSIncidentEnd_LifecycleEnd);
	LPS_EndVersusControlsByControllerUserId(GetClientUserId(client), LPSIncidentEnd_LifecycleEnd, 0);
}

void LPS_EndVersusControlsByControllerUserId(int controllerUserId, LPSIncidentEndReason reason = LPSIncidentEnd_Released, int helper = 0)
{
	if (controllerUserId <= 0)
	{
		return;
	}

	for (int victim = 1; victim <= MaxClients; victim++)
	{
		if (g_LPSVersusControlControllerUserId[victim] == controllerUserId)
		{
			LPS_EndVersusControl(victim, LPSControlType_None, reason, helper);
		}
	}
}

void LPS_ResetVersusControlVictim(int victim)
{
	if (victim < 1 || victim > MaxClients)
	{
		return;
	}

	g_LPSVersusControlType[victim] = LPSControlType_None;
	g_LPSVersusControlStartedAt[victim] = 0.0;
	g_LPSVersusControlControllerUserId[victim] = 0;
	g_LPSVersusControlZombieClass[victim] = 0;
	g_LPSVersusControlVictimIsBot[victim] = false;
	g_LPSVersusChargerPummeling[victim] = false;
}

int LPS_GetVersusInfectedAbilityStatOffset(
	int classIndex,
	LPSVersusInfectedAbilityStat stat
)
{
	return (classIndex * view_as<int>(LPSVersusInfectedAbilityStat_Count))
		+ view_as<int>(stat);
}

void LPS_AddVersusInfectedAbilityStat(
	int client,
	int zombieClass,
	LPSVersusInfectedAbilityStat stat,
	int amount
)
{
	int classIndex = LPS_GetVersusInfectedClassIndex(zombieClass);
	if (amount <= 0
		|| classIndex < 0
		|| stat < LPSVersusInfectedAbilityStat_HumanSurvivorControls
		|| stat >= LPSVersusInfectedAbilityStat_Count
		|| !LPS_CanOwnVersusAbilityStats(client))
	{
		return;
	}

	g_LPSClientVersusInfectedStats[client].classAbilityStats[
		LPS_GetVersusInfectedAbilityStatOffset(classIndex, stat)
	] += amount;
	g_LPSClientVersusInfectedStats[client].revision++;
	g_LPSClientVersusInfectedStatsDirty[client] = true;
}

bool LPS_CanOwnVersusAbilityStats(int client)
{
	return client >= 1
		&& client <= MaxClients
		&& IsClientInGame(client)
		&& !IsFakeClient(client)
		&& g_LPSClientVersusInfectedStatsActive[client]
		&& g_LPSClientSegmentActive[client]
		&& StrEqual(
			g_LPSClientVersusInfectedStats[client].segmentId,
			g_LPSClientSegmentId[client],
			true
		);
}

bool LPS_IsVersusSpitDamageSource(int attacker, int inflictor)
{
	return LPS_IsVersusSpitEntity(attacker) || LPS_IsVersusSpitEntity(inflictor);
}

int LPS_ResolveHumanVersusSpitOwner(int attacker, int inflictor)
{
	int owner = LPS_GetHumanVersusSpitOwner(inflictor);
	if (owner > 0)
	{
		return owner;
	}
	return LPS_GetHumanVersusSpitOwner(attacker);
}

int LPS_GetHumanVersusSpitOwner(int entity)
{
	if (!LPS_IsVersusSpitEntity(entity))
	{
		return 0;
	}

	int owner = GetClientOfUserId(g_LPSVersusSpitOwnerUserId[entity]);
	if (LPS_CanOwnVersusAbilityStats(owner)
		&& g_LPSVersusSpitOwnerSegmentGeneration[entity]
			== g_LPSVersusAbilitySegmentGeneration[owner])
	{
		return owner;
	}

	owner = LPS_GetEntityPlayerOwner(entity);
	if (LPS_IsHumanClientOnTeam(owner, LPS_TEAM_INFECTED)
		&& LPS_GetZombieClass(owner) == 4
		&& LPS_CanOwnVersusAbilityStats(owner))
	{
		g_LPSVersusSpitOwnerUserId[entity] = GetClientUserId(owner);
		g_LPSVersusSpitOwnerSegmentGeneration[entity] =
			g_LPSVersusAbilitySegmentGeneration[owner];
		return owner;
	}
	return 0;
}

bool LPS_IsVersusSpitEntity(int entity)
{
	if (entity <= MaxClients || entity > LPS_MAX_ENTITY_INDEX || !IsValidEntity(entity))
	{
		return false;
	}

	char classname[32];
	GetEntityClassname(entity, classname, sizeof(classname));
	return StrEqual(classname, "insect_swarm", true);
}

void LPS_ShutdownVersusAbilities()
{
	for (int client = 1; client <= MaxClients; client++)
	{
		LPS_ResetVersusControlVictim(client);
	}
	for (int entity = 1; entity <= LPS_MAX_ENTITY_INDEX; entity++)
	{
		g_LPSVersusSpitOwnerUserId[entity] = 0;
		g_LPSVersusSpitOwnerSegmentGeneration[entity] = 0;
	}
}
