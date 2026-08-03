#if defined _l4d2_player_stats_versus_stats_included
	#endinput
#endif
#define _l4d2_player_stats_versus_stats_included

enum struct LPSVersusSurvivorStatsSnapshot
{
	char segmentId[LPS_MAX_SEGMENT_ID];
	int commonKills;
	int humanSpecialKills;
	int botSpecialKills;
	int humanTankKills;
	int botTankKills;
	int damageToHumanSpecial;
	int damageToBotSpecial;
	int damageToHumanTank;
	int damageToBotTank;
	int damageTakenInfected;
	int friendlyFireToHumans;
	int friendlyFireToBots;
	int friendlyFireTaken;
	int incapacitations;
	int deaths;
	int incapRevives;
	int ledgeRescues;
	int defibRevives;
	int rescuesReceived;
	int medkitsUsedSelf;
	int medkitsUsedOnOthers;
	int medkitHealingSelf;
	int medkitHealingOthers;
	int pillsUsed;
	int adrenalineUsed;
	int temporaryHealthReceived;
	int witchKills;
	int damageToWitch;
	int molotovsThrown;
	int pipeBombsThrown;
	int vomitJarsThrown;
	int incendiaryPacksDeployed;
	int explosivePacksDeployed;
	int meleeTongueSelfCuts;
	int tankRocksDestroyed;
	int witchOneShots;
	int witchSoloKills;
	int classStats[
		view_as<int>(LPSVersusInfectedClass_Count)
		* view_as<int>(LPSVersusSurvivorClassStat_Count)
	];
	int revision;
}

enum struct LPSVersusInfectedStatsSnapshot
{
	char segmentId[LPS_MAX_SEGMENT_ID];
	int spawnCount;
	int damageToHumanSurvivors;
	int damageToBotSurvivors;
	int humanSurvivorIncaps;
	int botSurvivorIncaps;
	int humanSurvivorKills;
	int botSurvivorKills;
	int classStats[
		view_as<int>(LPSVersusInfectedClass_Count)
		* view_as<int>(LPSVersusInfectedStat_Count)
	];
	int classAbilityStats[
		view_as<int>(LPSVersusInfectedClass_Count)
		* view_as<int>(LPSVersusInfectedAbilityStat_Count)
	];
	int revision;
}

ArrayList g_LPSClosedVersusSurvivorStats;
ArrayList g_LPSClosedVersusInfectedStats;
bool g_LPSVersusStatsQueueOverflowReported;

bool g_LPSClientVersusSurvivorStatsActive[MAXPLAYERS + 1];
bool g_LPSClientVersusSurvivorStatsDirty[MAXPLAYERS + 1];
LPSVersusSurvivorStatsSnapshot g_LPSClientVersusSurvivorStats[MAXPLAYERS + 1];
int g_LPSClientVersusSurvivorStatsPersistedRevision[MAXPLAYERS + 1];

bool g_LPSClientVersusInfectedStatsActive[MAXPLAYERS + 1];
bool g_LPSClientVersusInfectedStatsDirty[MAXPLAYERS + 1];
bool g_LPSClientVersusInfectedSpawnPending[MAXPLAYERS + 1];
LPSVersusInfectedStatsSnapshot g_LPSClientVersusInfectedStats[MAXPLAYERS + 1];
int g_LPSClientVersusInfectedStatsPersistedRevision[MAXPLAYERS + 1];

bool g_LPSFlushCapturedVersusSurvivorStats[MAXPLAYERS + 1];
char g_LPSFlushCapturedVersusSurvivorSegmentId[MAXPLAYERS + 1][LPS_MAX_SEGMENT_ID];
int g_LPSFlushCapturedVersusSurvivorRevision[MAXPLAYERS + 1];

bool g_LPSFlushCapturedVersusInfectedStats[MAXPLAYERS + 1];
char g_LPSFlushCapturedVersusInfectedSegmentId[MAXPLAYERS + 1][LPS_MAX_SEGMENT_ID];
int g_LPSFlushCapturedVersusInfectedRevision[MAXPLAYERS + 1];

int g_LPSFlushClosedVersusSurvivorStatsCount;
int g_LPSFlushClosedVersusInfectedStatsCount;

void LPS_InitializeVersusStats()
{
	g_LPSClosedVersusSurvivorStats = new ArrayList(sizeof(LPSVersusSurvivorStatsSnapshot));
	g_LPSClosedVersusInfectedStats = new ArrayList(sizeof(LPSVersusInfectedStatsSnapshot));
	g_LPSVersusStatsQueueOverflowReported = false;
	g_LPSFlushClosedVersusSurvivorStatsCount = 0;
	g_LPSFlushClosedVersusInfectedStatsCount = 0;

	HookEvent("player_spawn", LPS_Event_VersusPlayerSpawnStats, EventHookMode_Post);
	HookEvent("infected_death", LPS_Event_VersusInfectedDeathStats, EventHookMode_Post);
	HookEvent("player_death", LPS_Event_VersusPlayerDeathStats, EventHookMode_Post);
	HookEvent("player_incapacitated", LPS_Event_VersusPlayerIncapacitatedStats, EventHookMode_Post);
	HookEvent("revive_success", LPS_Event_VersusReviveSuccessStats, EventHookMode_Post);
	HookEvent("defibrillator_used", LPS_Event_VersusDefibrillatorUsedStats, EventHookMode_Post);
	HookEvent("heal_success", LPS_Event_VersusHealSuccessStats, EventHookMode_Post);
	HookEvent("pills_used", LPS_Event_VersusPillsUsedStats, EventHookMode_Post);
	HookEvent("adrenaline_used", LPS_Event_VersusAdrenalineUsedStats, EventHookMode_Post);

	for (int client = 1; client <= MaxClients; client++)
	{
		LPS_ResetClientVersusStatsState(client);
	}
	LPS_ResetVersusStatsFlushCapture();
}

bool LPS_CanRecordVersusEvents()
{
	return g_LPSModeFamily == LPSModeFamily_Versus
		&& g_LPSVersusStatsEnabled.BoolValue
		&& g_LPSRunActive
		&& g_LPSRoundActive;
}

void LPS_OnClientVersusSegmentStarted(int client)
{
	LPS_OnClientVersusAbilitySegmentStarted(client);
	LPS_OnClientVersusSurvivorDetailSegmentStarted(client);

	if (!LPS_CanRecordVersusEvents() || !g_LPSClientSegmentActive[client])
	{
		return;
	}

	if (StrEqual(g_LPSClientSegmentSide[client], "survivor", true))
	{
		strcopy(
			g_LPSClientVersusSurvivorStats[client].segmentId,
			LPS_MAX_SEGMENT_ID,
			g_LPSClientSegmentId[client]
		);
		g_LPSClientVersusSurvivorStats[client].revision = 1;
		g_LPSClientVersusSurvivorStatsPersistedRevision[client] = 0;
		g_LPSClientVersusSurvivorStatsDirty[client] = true;
		g_LPSClientVersusSurvivorStatsActive[client] = true;
		g_LPSClientEffectiveHealthBeforeCommand[client] = LPS_GetTrackedEffectiveHealth(client);
		return;
	}

	if (StrEqual(g_LPSClientSegmentSide[client], "infected", true))
	{
		strcopy(
			g_LPSClientVersusInfectedStats[client].segmentId,
			LPS_MAX_SEGMENT_ID,
			g_LPSClientSegmentId[client]
		);
		g_LPSClientVersusInfectedStats[client].revision = 1;
		g_LPSClientVersusInfectedStatsPersistedRevision[client] = 0;
		g_LPSClientVersusInfectedStatsDirty[client] = true;
		g_LPSClientVersusInfectedStatsActive[client] = true;
		// If the Segment begins after player_spawn, the next non-ghost command
		// still records the currently controlled infected life exactly once.
		g_LPSClientVersusInfectedSpawnPending[client] = true;
	}
}

void LPS_OnClientVersusSegmentEnding(int client)
{
	// Close both sides of a possible pin before the infected snapshot is
	// captured. This covers death, rescue, side changes and Round shutdown.
	LPS_OnClientVersusAbilitySegmentEnding(client);
	LPS_OnClientVersusSurvivorDetailSegmentEnding(client);

	if (g_LPSClientVersusSurvivorStatsActive[client])
	{
		LPSVersusSurvivorStatsSnapshot snapshot;
		LPS_FillClientVersusSurvivorSnapshot(client, snapshot, true);
		LPS_QueueClosedVersusSurvivorStats(snapshot);
	}

	if (g_LPSClientVersusInfectedStatsActive[client])
	{
		LPSVersusInfectedStatsSnapshot snapshot;
		LPS_FillClientVersusInfectedSnapshot(client, snapshot, true);
		LPS_QueueClosedVersusInfectedStats(snapshot);
	}

	LPS_ResetClientVersusStatsState(client);
}

public void LPS_Event_VersusPlayerSpawnStats(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordVersusEvents())
	{
		return;
	}

	int client = GetClientOfUserId(event.GetInt("userid"));
	if (!LPS_IsHumanClientOnTeam(client, LPS_TEAM_INFECTED)
		|| !g_LPSClientVersusInfectedStatsActive[client])
	{
		return;
	}

	g_LPSClientVersusInfectedSpawnPending[client] = true;
	RequestFrame(LPS_Frame_RecordVersusSpawn, GetClientUserId(client));
}

public void LPS_Frame_RecordVersusSpawn(any userId)
{
	LPS_RecordVersusSpawnIfReady(GetClientOfUserId(userId));
}

void LPS_RecordVersusSpawnIfReady(int client)
{
	if (!LPS_CanRecordVersusEvents()
		|| !LPS_IsHumanClientOnTeam(client, LPS_TEAM_INFECTED)
		|| !g_LPSClientVersusInfectedStatsActive[client]
		|| !g_LPSClientVersusInfectedSpawnPending[client]
		|| !IsPlayerAlive(client))
	{
		return;
	}

	if (HasEntProp(client, Prop_Send, "m_isGhost")
		&& GetEntProp(client, Prop_Send, "m_isGhost") != 0)
	{
		return;
	}

	int zombieClass = LPS_GetZombieClass(client);
	if (!((zombieClass >= 1 && zombieClass <= 6) || zombieClass == 8))
	{
		return;
	}

	g_LPSClientVersusInfectedSpawnPending[client] = false;
	LPS_AddVersusInfectedStat(
		client,
		zombieClass,
		LPSVersusInfectedStat_SpawnCount,
		1
	);
}

void LPS_RecordVersusDamage(int victim, int attacker, int inflictor, int actualDamage)
{
	if (!LPS_CanRecordVersusEvents()
		|| victim < 1
		|| victim > LPS_MAX_ENTITY_INDEX
		|| !IsValidEntity(victim)
		|| actualDamage <= 0)
	{
		return;
	}

	if (victim > MaxClients)
	{
		if (!LPS_IsWitchEntity(victim))
		{
			return;
		}

		int survivor = LPS_ResolveHumanVersusAttacker(
			attacker,
			inflictor,
			LPS_TEAM_SURVIVOR
		);
		if (survivor > 0)
		{
			LPS_AddVersusSurvivorStat(
				survivor,
				LPSVersusSurvivorStat_DamageToWitch,
				actualDamage
			);
			LPS_RecordVersusWitchContribution(victim, survivor);
		}
		return;
	}

	if (!IsClientInGame(victim))
	{
		return;
	}

	int victimTeam = GetClientTeam(victim);
	if (victimTeam == LPS_TEAM_INFECTED)
	{
		int survivor = LPS_ResolveHumanVersusAttacker(attacker, inflictor, LPS_TEAM_SURVIVOR);
		if (survivor <= 0)
		{
			return;
		}

		int zombieClass = LPS_GetZombieClass(victim);
		LPS_AddVersusSurvivorClassStat(
			survivor,
			zombieClass,
			IsFakeClient(victim)
				? LPSVersusSurvivorClassStat_DamageToBotControllers
				: LPSVersusSurvivorClassStat_DamageToHumanControllers,
			actualDamage
		);
		return;
	}

	if (victimTeam != LPS_TEAM_SURVIVOR)
	{
		return;
	}

	int survivor = LPS_ResolveHumanVersusAttacker(attacker, inflictor, LPS_TEAM_SURVIVOR);
	if (survivor > 0 && survivor != victim)
	{
		if (IsFakeClient(victim))
		{
			LPS_AddVersusSurvivorStat(
				survivor,
				LPSVersusSurvivorStat_FriendlyFireToBots,
				actualDamage
			);
		}
		else
		{
			LPS_AddVersusSurvivorStat(
				survivor,
				LPSVersusSurvivorStat_FriendlyFireToHumans,
				actualDamage
			);
			LPS_AddVersusSurvivorStat(
				victim,
				LPSVersusSurvivorStat_FriendlyFireTaken,
				actualDamage
			);
		}
		return;
	}

	// Spit pools can outlive their Spitter. Resolve their captured owner first
	// so late acid ticks remain attributed to the original human and class.
	int infected = LPS_ResolveHumanVersusSpitOwner(attacker, inflictor);
	int zombieClass = infected > 0 ? 4 : 0;
	if (infected <= 0)
	{
		infected = LPS_ResolveHumanVersusAttacker(attacker, inflictor, LPS_TEAM_INFECTED);
		zombieClass = infected > 0 ? LPS_GetZombieClass(infected) : 0;
	}
	if (infected > 0)
	{
		LPS_AddVersusInfectedStat(
			infected,
			zombieClass,
			IsFakeClient(victim)
				? LPSVersusInfectedStat_DamageToBotSurvivors
				: LPSVersusInfectedStat_DamageToHumanSurvivors,
			actualDamage
		);
		if (zombieClass == 4 && LPS_IsVersusSpitDamageSource(attacker, inflictor))
		{
			LPS_AddVersusInfectedAbilityStat(
				infected,
				4,
				IsFakeClient(victim)
					? LPSVersusInfectedAbilityStat_BotSurvivorAbilityDamage
					: LPSVersusInfectedAbilityStat_HumanSurvivorAbilityDamage,
				actualDamage
			);
		}
	}

	if (!IsFakeClient(victim)
		&& (infected > 0
			|| LPS_IsInfectedDamageSource(attacker)
			|| LPS_IsInfectedDamageSource(inflictor)))
	{
		LPS_AddVersusSurvivorStat(
			victim,
			LPSVersusSurvivorStat_DamageTakenInfected,
			actualDamage
		);
	}
}

public void LPS_Event_VersusInfectedDeathStats(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordVersusEvents())
	{
		return;
	}

	int attacker = GetClientOfUserId(event.GetInt("attacker"));
	LPS_AddVersusSurvivorStat(attacker, LPSVersusSurvivorStat_CommonKills, 1);
}

public void LPS_Event_VersusPlayerDeathStats(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordVersusEvents())
	{
		return;
	}

	int victim = GetClientOfUserId(event.GetInt("userid"));
	if (victim < 1 || victim > MaxClients || !IsClientInGame(victim))
	{
		return;
	}
	LPS_OnVersusAbilityPlayerDeath(victim);

	int victimTeam = GetClientTeam(victim);
	if (victimTeam == LPS_TEAM_INFECTED)
	{
		g_LPSClientVersusInfectedSpawnPending[victim] = false;
	}

	if (event.GetBool("abort"))
	{
		return;
	}

	int attacker = GetClientOfUserId(event.GetInt("attacker"));
	if (victimTeam == LPS_TEAM_SURVIVOR)
	{
		LPS_AddVersusSurvivorStat(victim, LPSVersusSurvivorStat_Deaths, 1);
		if (LPS_IsHumanClientOnTeam(attacker, LPS_TEAM_INFECTED))
		{
			LPS_AddVersusInfectedStat(
				attacker,
				LPS_GetZombieClass(attacker),
				IsFakeClient(victim)
					? LPSVersusInfectedStat_BotSurvivorKills
					: LPSVersusInfectedStat_HumanSurvivorKills,
				1
			);
		}
		return;
	}

	if (victimTeam != LPS_TEAM_INFECTED
		|| !LPS_IsHumanClientOnTeam(attacker, LPS_TEAM_SURVIVOR))
	{
		return;
	}

	int zombieClass = LPS_GetZombieClass(victim);
	LPS_AddVersusSurvivorClassStat(
		attacker,
		zombieClass,
		IsFakeClient(victim)
			? LPSVersusSurvivorClassStat_BotControllerKills
			: LPSVersusSurvivorClassStat_HumanControllerKills,
		1
	);
}

public void LPS_Event_VersusPlayerIncapacitatedStats(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordVersusEvents())
	{
		return;
	}

	int victim = GetClientOfUserId(event.GetInt("userid"));
	if (victim < 1
		|| victim > MaxClients
		|| !IsClientInGame(victim)
		|| GetClientTeam(victim) != LPS_TEAM_SURVIVOR)
	{
		return;
	}

	LPS_AddVersusSurvivorStat(victim, LPSVersusSurvivorStat_Incapacitations, 1);
	int attacker = GetClientOfUserId(event.GetInt("attacker"));
	if (LPS_IsHumanClientOnTeam(attacker, LPS_TEAM_INFECTED))
	{
		LPS_AddVersusInfectedStat(
			attacker,
			LPS_GetZombieClass(attacker),
			IsFakeClient(victim)
				? LPSVersusInfectedStat_BotSurvivorIncaps
				: LPSVersusInfectedStat_HumanSurvivorIncaps,
			1
		);
	}
}

public void LPS_Event_VersusReviveSuccessStats(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordVersusEvents())
	{
		return;
	}

	int rescuer = GetClientOfUserId(event.GetInt("userid"));
	int subject = GetClientOfUserId(event.GetInt("subject"));
	LPS_AddVersusSurvivorStat(
		rescuer,
		event.GetBool("ledge_hang")
			? LPSVersusSurvivorStat_LedgeRescues
			: LPSVersusSurvivorStat_IncapRevives,
		1
	);
	LPS_AddVersusSurvivorStat(subject, LPSVersusSurvivorStat_RescuesReceived, 1);
}

public void LPS_Event_VersusDefibrillatorUsedStats(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordVersusEvents())
	{
		return;
	}

	int rescuer = GetClientOfUserId(event.GetInt("userid"));
	int subject = GetClientOfUserId(event.GetInt("subject"));
	LPS_AddVersusSurvivorStat(rescuer, LPSVersusSurvivorStat_DefibRevives, 1);
	LPS_AddVersusSurvivorStat(subject, LPSVersusSurvivorStat_RescuesReceived, 1);
}

public void LPS_Event_VersusHealSuccessStats(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordVersusEvents())
	{
		return;
	}

	int healer = GetClientOfUserId(event.GetInt("userid"));
	int subject = GetClientOfUserId(event.GetInt("subject"));
	int restored = event.GetInt("health_restored");
	if (healer == subject)
	{
		LPS_AddVersusSurvivorStat(healer, LPSVersusSurvivorStat_MedkitsUsedSelf, 1);
		LPS_AddVersusSurvivorStat(healer, LPSVersusSurvivorStat_MedkitHealingSelf, restored);
	}
	else
	{
		LPS_AddVersusSurvivorStat(healer, LPSVersusSurvivorStat_MedkitsUsedOnOthers, 1);
		LPS_AddVersusSurvivorStat(healer, LPSVersusSurvivorStat_MedkitHealingOthers, restored);
	}
}

public void LPS_Event_VersusPillsUsedStats(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordVersusEvents())
	{
		return;
	}

	int subject = GetClientOfUserId(event.GetInt("subject"));
	LPS_AddVersusSurvivorStat(subject, LPSVersusSurvivorStat_PillsUsed, 1);
	LPS_RecordVersusTemporaryHealthAfterUse(subject);
}

public void LPS_Event_VersusAdrenalineUsedStats(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordVersusEvents())
	{
		return;
	}

	int client = GetClientOfUserId(event.GetInt("userid"));
	LPS_AddVersusSurvivorStat(client, LPSVersusSurvivorStat_AdrenalineUsed, 1);
	LPS_RecordVersusTemporaryHealthAfterUse(client);
}

void LPS_RecordVersusTemporaryHealthAfterUse(int client)
{
	if (client < 1
		|| client > MaxClients
		|| !g_LPSClientVersusSurvivorStatsActive[client]
		|| !IsClientInGame(client))
	{
		return;
	}

	float before = g_LPSClientEffectiveHealthBeforeCommand[client];
	float current = LPS_GetTrackedEffectiveHealth(client);
	if (before <= 0.0)
	{
		return;
	}

	if (current - before > 0.25)
	{
		LPS_AddVersusSurvivorStat(
			client,
			LPSVersusSurvivorStat_TemporaryHealthReceived,
			RoundToNearest(current - before)
		);
		return;
	}

	DataPack pack;
	CreateDataTimer(0.0, LPS_Timer_RecordVersusTemporaryHealth, pack, TIMER_FLAG_NO_MAPCHANGE);
	pack.WriteCell(GetClientUserId(client));
	pack.WriteFloat(before);
	pack.WriteString(g_LPSClientVersusSurvivorStats[client].segmentId);
}

public Action LPS_Timer_RecordVersusTemporaryHealth(Handle timer, DataPack pack)
{
	pack.Reset();
	int client = GetClientOfUserId(pack.ReadCell());
	float before = pack.ReadFloat();
	char segmentId[LPS_MAX_SEGMENT_ID];
	pack.ReadString(segmentId, sizeof(segmentId));

	if (client < 1
		|| client > MaxClients
		|| !g_LPSClientVersusSurvivorStatsActive[client]
		|| !StrEqual(g_LPSClientVersusSurvivorStats[client].segmentId, segmentId, true))
	{
		return Plugin_Stop;
	}

	float gained = LPS_GetTrackedEffectiveHealth(client) - before;
	if (gained > 0.25)
	{
		LPS_AddVersusSurvivorStat(
			client,
			LPSVersusSurvivorStat_TemporaryHealthReceived,
			RoundToNearest(gained)
		);
	}
	return Plugin_Stop;
}

bool LPS_IsHumanClientOnTeam(int client, int team)
{
	return client >= 1
		&& client <= MaxClients
		&& IsClientInGame(client)
		&& !IsFakeClient(client)
		&& GetClientTeam(client) == team;
}

int LPS_ResolveHumanVersusAttacker(int attacker, int inflictor, int team)
{
	if (LPS_IsHumanClientOnTeam(attacker, team))
	{
		return attacker;
	}

	int owner = LPS_GetEntityPlayerOwner(attacker);
	if (LPS_IsHumanClientOnTeam(owner, team))
	{
		return owner;
	}

	owner = LPS_GetEntityPlayerOwner(inflictor);
	return LPS_IsHumanClientOnTeam(owner, team) ? owner : 0;
}

void LPS_AddVersusSurvivorStat(int client, LPSVersusSurvivorStat stat, int amount)
{
	if (amount <= 0
		|| client < 1
		|| client > MaxClients
		|| !g_LPSClientVersusSurvivorStatsActive[client]
		|| !g_LPSClientSegmentActive[client]
		|| !StrEqual(
			g_LPSClientVersusSurvivorStats[client].segmentId,
			g_LPSClientSegmentId[client],
			true
		))
	{
		return;
	}

	switch (stat)
	{
		case LPSVersusSurvivorStat_CommonKills: g_LPSClientVersusSurvivorStats[client].commonKills += amount;
		case LPSVersusSurvivorStat_HumanSpecialKills: g_LPSClientVersusSurvivorStats[client].humanSpecialKills += amount;
		case LPSVersusSurvivorStat_BotSpecialKills: g_LPSClientVersusSurvivorStats[client].botSpecialKills += amount;
		case LPSVersusSurvivorStat_HumanTankKills: g_LPSClientVersusSurvivorStats[client].humanTankKills += amount;
		case LPSVersusSurvivorStat_BotTankKills: g_LPSClientVersusSurvivorStats[client].botTankKills += amount;
		case LPSVersusSurvivorStat_DamageToHumanSpecial: g_LPSClientVersusSurvivorStats[client].damageToHumanSpecial += amount;
		case LPSVersusSurvivorStat_DamageToBotSpecial: g_LPSClientVersusSurvivorStats[client].damageToBotSpecial += amount;
		case LPSVersusSurvivorStat_DamageToHumanTank: g_LPSClientVersusSurvivorStats[client].damageToHumanTank += amount;
		case LPSVersusSurvivorStat_DamageToBotTank: g_LPSClientVersusSurvivorStats[client].damageToBotTank += amount;
		case LPSVersusSurvivorStat_DamageTakenInfected: g_LPSClientVersusSurvivorStats[client].damageTakenInfected += amount;
		case LPSVersusSurvivorStat_FriendlyFireToHumans: g_LPSClientVersusSurvivorStats[client].friendlyFireToHumans += amount;
		case LPSVersusSurvivorStat_FriendlyFireToBots: g_LPSClientVersusSurvivorStats[client].friendlyFireToBots += amount;
		case LPSVersusSurvivorStat_FriendlyFireTaken: g_LPSClientVersusSurvivorStats[client].friendlyFireTaken += amount;
		case LPSVersusSurvivorStat_Incapacitations: g_LPSClientVersusSurvivorStats[client].incapacitations += amount;
		case LPSVersusSurvivorStat_Deaths: g_LPSClientVersusSurvivorStats[client].deaths += amount;
		case LPSVersusSurvivorStat_IncapRevives: g_LPSClientVersusSurvivorStats[client].incapRevives += amount;
		case LPSVersusSurvivorStat_LedgeRescues: g_LPSClientVersusSurvivorStats[client].ledgeRescues += amount;
		case LPSVersusSurvivorStat_DefibRevives: g_LPSClientVersusSurvivorStats[client].defibRevives += amount;
		case LPSVersusSurvivorStat_RescuesReceived: g_LPSClientVersusSurvivorStats[client].rescuesReceived += amount;
		case LPSVersusSurvivorStat_MedkitsUsedSelf: g_LPSClientVersusSurvivorStats[client].medkitsUsedSelf += amount;
		case LPSVersusSurvivorStat_MedkitsUsedOnOthers: g_LPSClientVersusSurvivorStats[client].medkitsUsedOnOthers += amount;
		case LPSVersusSurvivorStat_MedkitHealingSelf: g_LPSClientVersusSurvivorStats[client].medkitHealingSelf += amount;
		case LPSVersusSurvivorStat_MedkitHealingOthers: g_LPSClientVersusSurvivorStats[client].medkitHealingOthers += amount;
		case LPSVersusSurvivorStat_PillsUsed: g_LPSClientVersusSurvivorStats[client].pillsUsed += amount;
		case LPSVersusSurvivorStat_AdrenalineUsed: g_LPSClientVersusSurvivorStats[client].adrenalineUsed += amount;
		case LPSVersusSurvivorStat_TemporaryHealthReceived: g_LPSClientVersusSurvivorStats[client].temporaryHealthReceived += amount;
		case LPSVersusSurvivorStat_WitchKills: g_LPSClientVersusSurvivorStats[client].witchKills += amount;
		case LPSVersusSurvivorStat_DamageToWitch: g_LPSClientVersusSurvivorStats[client].damageToWitch += amount;
		case LPSVersusSurvivorStat_MolotovsThrown: g_LPSClientVersusSurvivorStats[client].molotovsThrown += amount;
		case LPSVersusSurvivorStat_PipeBombsThrown: g_LPSClientVersusSurvivorStats[client].pipeBombsThrown += amount;
		case LPSVersusSurvivorStat_VomitJarsThrown: g_LPSClientVersusSurvivorStats[client].vomitJarsThrown += amount;
		case LPSVersusSurvivorStat_IncendiaryPacksDeployed: g_LPSClientVersusSurvivorStats[client].incendiaryPacksDeployed += amount;
		case LPSVersusSurvivorStat_ExplosivePacksDeployed: g_LPSClientVersusSurvivorStats[client].explosivePacksDeployed += amount;
		case LPSVersusSurvivorStat_MeleeTongueSelfCuts: g_LPSClientVersusSurvivorStats[client].meleeTongueSelfCuts += amount;
		case LPSVersusSurvivorStat_TankRocksDestroyed: g_LPSClientVersusSurvivorStats[client].tankRocksDestroyed += amount;
		case LPSVersusSurvivorStat_WitchOneShots: g_LPSClientVersusSurvivorStats[client].witchOneShots += amount;
		case LPSVersusSurvivorStat_WitchSoloKills: g_LPSClientVersusSurvivorStats[client].witchSoloKills += amount;
	}

	g_LPSClientVersusSurvivorStats[client].revision++;
	g_LPSClientVersusSurvivorStatsDirty[client] = true;
}

int LPS_GetVersusSurvivorClassStatOffset(
	int classIndex,
	LPSVersusSurvivorClassStat stat
)
{
	return (classIndex * view_as<int>(LPSVersusSurvivorClassStat_Count))
		+ view_as<int>(stat);
}

void LPS_AddVersusSurvivorClassStat(
	int client,
	int zombieClass,
	LPSVersusSurvivorClassStat stat,
	int amount
)
{
	int classIndex = LPS_GetVersusInfectedClassIndex(zombieClass);
	if (amount <= 0
		|| classIndex < 0
		|| stat < LPSVersusSurvivorClassStat_HumanControllerKills
		|| stat >= LPSVersusSurvivorClassStat_Count
		|| client < 1
		|| client > MaxClients
		|| !g_LPSClientVersusSurvivorStatsActive[client]
		|| !g_LPSClientSegmentActive[client]
		|| !StrEqual(
			g_LPSClientVersusSurvivorStats[client].segmentId,
			g_LPSClientSegmentId[client],
			true
		))
	{
		return;
	}

	g_LPSClientVersusSurvivorStats[client].classStats[
		LPS_GetVersusSurvivorClassStatOffset(classIndex, stat)
	] += amount;

	bool tank = zombieClass == 8;
	switch (stat)
	{
		case LPSVersusSurvivorClassStat_HumanControllerKills:
		{
			if (tank) g_LPSClientVersusSurvivorStats[client].humanTankKills += amount;
			else g_LPSClientVersusSurvivorStats[client].humanSpecialKills += amount;
		}
		case LPSVersusSurvivorClassStat_BotControllerKills:
		{
			if (tank) g_LPSClientVersusSurvivorStats[client].botTankKills += amount;
			else g_LPSClientVersusSurvivorStats[client].botSpecialKills += amount;
		}
		case LPSVersusSurvivorClassStat_DamageToHumanControllers:
		{
			if (tank) g_LPSClientVersusSurvivorStats[client].damageToHumanTank += amount;
			else g_LPSClientVersusSurvivorStats[client].damageToHumanSpecial += amount;
		}
		case LPSVersusSurvivorClassStat_DamageToBotControllers:
		{
			if (tank) g_LPSClientVersusSurvivorStats[client].damageToBotTank += amount;
			else g_LPSClientVersusSurvivorStats[client].damageToBotSpecial += amount;
		}
	}

	g_LPSClientVersusSurvivorStats[client].revision++;
	g_LPSClientVersusSurvivorStatsDirty[client] = true;
}

int LPS_GetVersusInfectedClassIndex(int zombieClass)
{
	switch (zombieClass)
	{
		case 1: return view_as<int>(LPSVersusInfectedClass_Smoker);
		case 2: return view_as<int>(LPSVersusInfectedClass_Boomer);
		case 3: return view_as<int>(LPSVersusInfectedClass_Hunter);
		case 4: return view_as<int>(LPSVersusInfectedClass_Spitter);
		case 5: return view_as<int>(LPSVersusInfectedClass_Jockey);
		case 6: return view_as<int>(LPSVersusInfectedClass_Charger);
		case 8: return view_as<int>(LPSVersusInfectedClass_Tank);
	}
	return -1;
}

int LPS_GetVersusInfectedZombieClass(int classIndex)
{
	return classIndex == view_as<int>(LPSVersusInfectedClass_Tank)
		? 8
		: classIndex + 1;
}

int LPS_GetVersusInfectedClassStatOffset(int classIndex, LPSVersusInfectedStat stat)
{
	return (classIndex * view_as<int>(LPSVersusInfectedStat_Count)) + view_as<int>(stat);
}

void LPS_AddVersusInfectedStat(
	int client,
	int zombieClass,
	LPSVersusInfectedStat stat,
	int amount
)
{
	int classIndex = LPS_GetVersusInfectedClassIndex(zombieClass);
	if (amount <= 0
		|| classIndex < 0
		|| stat < LPSVersusInfectedStat_SpawnCount
		|| stat >= LPSVersusInfectedStat_Count
		|| client < 1
		|| client > MaxClients
		|| !g_LPSClientVersusInfectedStatsActive[client]
		|| !g_LPSClientSegmentActive[client]
		|| !StrEqual(
			g_LPSClientVersusInfectedStats[client].segmentId,
			g_LPSClientSegmentId[client],
			true
		))
	{
		return;
	}

	switch (stat)
	{
		case LPSVersusInfectedStat_SpawnCount: g_LPSClientVersusInfectedStats[client].spawnCount += amount;
		case LPSVersusInfectedStat_DamageToHumanSurvivors: g_LPSClientVersusInfectedStats[client].damageToHumanSurvivors += amount;
		case LPSVersusInfectedStat_DamageToBotSurvivors: g_LPSClientVersusInfectedStats[client].damageToBotSurvivors += amount;
		case LPSVersusInfectedStat_HumanSurvivorIncaps: g_LPSClientVersusInfectedStats[client].humanSurvivorIncaps += amount;
		case LPSVersusInfectedStat_BotSurvivorIncaps: g_LPSClientVersusInfectedStats[client].botSurvivorIncaps += amount;
		case LPSVersusInfectedStat_HumanSurvivorKills: g_LPSClientVersusInfectedStats[client].humanSurvivorKills += amount;
		case LPSVersusInfectedStat_BotSurvivorKills: g_LPSClientVersusInfectedStats[client].botSurvivorKills += amount;
	}
	g_LPSClientVersusInfectedStats[client].classStats[
		LPS_GetVersusInfectedClassStatOffset(classIndex, stat)
	] += amount;

	g_LPSClientVersusInfectedStats[client].revision++;
	g_LPSClientVersusInfectedStatsDirty[client] = true;
}

void LPS_FillClientVersusSurvivorSnapshot(
	int client,
	LPSVersusSurvivorStatsSnapshot snapshot,
	bool closed
)
{
	strcopy(snapshot.segmentId, sizeof(snapshot.segmentId), g_LPSClientVersusSurvivorStats[client].segmentId);
	snapshot.commonKills = g_LPSClientVersusSurvivorStats[client].commonKills;
	snapshot.humanSpecialKills = g_LPSClientVersusSurvivorStats[client].humanSpecialKills;
	snapshot.botSpecialKills = g_LPSClientVersusSurvivorStats[client].botSpecialKills;
	snapshot.humanTankKills = g_LPSClientVersusSurvivorStats[client].humanTankKills;
	snapshot.botTankKills = g_LPSClientVersusSurvivorStats[client].botTankKills;
	snapshot.damageToHumanSpecial = g_LPSClientVersusSurvivorStats[client].damageToHumanSpecial;
	snapshot.damageToBotSpecial = g_LPSClientVersusSurvivorStats[client].damageToBotSpecial;
	snapshot.damageToHumanTank = g_LPSClientVersusSurvivorStats[client].damageToHumanTank;
	snapshot.damageToBotTank = g_LPSClientVersusSurvivorStats[client].damageToBotTank;
	snapshot.damageTakenInfected = g_LPSClientVersusSurvivorStats[client].damageTakenInfected;
	snapshot.friendlyFireToHumans = g_LPSClientVersusSurvivorStats[client].friendlyFireToHumans;
	snapshot.friendlyFireToBots = g_LPSClientVersusSurvivorStats[client].friendlyFireToBots;
	snapshot.friendlyFireTaken = g_LPSClientVersusSurvivorStats[client].friendlyFireTaken;
	snapshot.incapacitations = g_LPSClientVersusSurvivorStats[client].incapacitations;
	snapshot.deaths = g_LPSClientVersusSurvivorStats[client].deaths;
	snapshot.incapRevives = g_LPSClientVersusSurvivorStats[client].incapRevives;
	snapshot.ledgeRescues = g_LPSClientVersusSurvivorStats[client].ledgeRescues;
	snapshot.defibRevives = g_LPSClientVersusSurvivorStats[client].defibRevives;
	snapshot.rescuesReceived = g_LPSClientVersusSurvivorStats[client].rescuesReceived;
	snapshot.medkitsUsedSelf = g_LPSClientVersusSurvivorStats[client].medkitsUsedSelf;
	snapshot.medkitsUsedOnOthers = g_LPSClientVersusSurvivorStats[client].medkitsUsedOnOthers;
	snapshot.medkitHealingSelf = g_LPSClientVersusSurvivorStats[client].medkitHealingSelf;
	snapshot.medkitHealingOthers = g_LPSClientVersusSurvivorStats[client].medkitHealingOthers;
	snapshot.pillsUsed = g_LPSClientVersusSurvivorStats[client].pillsUsed;
	snapshot.adrenalineUsed = g_LPSClientVersusSurvivorStats[client].adrenalineUsed;
	snapshot.temporaryHealthReceived = g_LPSClientVersusSurvivorStats[client].temporaryHealthReceived;
	snapshot.witchKills = g_LPSClientVersusSurvivorStats[client].witchKills;
	snapshot.damageToWitch = g_LPSClientVersusSurvivorStats[client].damageToWitch;
	snapshot.molotovsThrown = g_LPSClientVersusSurvivorStats[client].molotovsThrown;
	snapshot.pipeBombsThrown = g_LPSClientVersusSurvivorStats[client].pipeBombsThrown;
	snapshot.vomitJarsThrown = g_LPSClientVersusSurvivorStats[client].vomitJarsThrown;
	snapshot.incendiaryPacksDeployed = g_LPSClientVersusSurvivorStats[client].incendiaryPacksDeployed;
	snapshot.explosivePacksDeployed = g_LPSClientVersusSurvivorStats[client].explosivePacksDeployed;
	snapshot.meleeTongueSelfCuts = g_LPSClientVersusSurvivorStats[client].meleeTongueSelfCuts;
	snapshot.tankRocksDestroyed = g_LPSClientVersusSurvivorStats[client].tankRocksDestroyed;
	snapshot.witchOneShots = g_LPSClientVersusSurvivorStats[client].witchOneShots;
	snapshot.witchSoloKills = g_LPSClientVersusSurvivorStats[client].witchSoloKills;
	for (int index = 0; index < sizeof(snapshot.classStats); index++)
	{
		snapshot.classStats[index] = g_LPSClientVersusSurvivorStats[client].classStats[index];
	}
	snapshot.revision = g_LPSClientVersusSurvivorStats[client].revision + (closed ? 1 : 0);
}

void LPS_FillClientVersusInfectedSnapshot(
	int client,
	LPSVersusInfectedStatsSnapshot snapshot,
	bool closed
)
{
	strcopy(snapshot.segmentId, sizeof(snapshot.segmentId), g_LPSClientVersusInfectedStats[client].segmentId);
	snapshot.spawnCount = g_LPSClientVersusInfectedStats[client].spawnCount;
	snapshot.damageToHumanSurvivors = g_LPSClientVersusInfectedStats[client].damageToHumanSurvivors;
	snapshot.damageToBotSurvivors = g_LPSClientVersusInfectedStats[client].damageToBotSurvivors;
	snapshot.humanSurvivorIncaps = g_LPSClientVersusInfectedStats[client].humanSurvivorIncaps;
	snapshot.botSurvivorIncaps = g_LPSClientVersusInfectedStats[client].botSurvivorIncaps;
	snapshot.humanSurvivorKills = g_LPSClientVersusInfectedStats[client].humanSurvivorKills;
	snapshot.botSurvivorKills = g_LPSClientVersusInfectedStats[client].botSurvivorKills;
	for (int index = 0; index < sizeof(snapshot.classStats); index++)
	{
		snapshot.classStats[index] = g_LPSClientVersusInfectedStats[client].classStats[index];
	}
	for (int index = 0; index < sizeof(snapshot.classAbilityStats); index++)
	{
		snapshot.classAbilityStats[index] = g_LPSClientVersusInfectedStats[client].classAbilityStats[index];
	}
	snapshot.revision = g_LPSClientVersusInfectedStats[client].revision + (closed ? 1 : 0);
}

void LPS_QueueClosedVersusSurvivorStats(LPSVersusSurvivorStatsSnapshot snapshot)
{
	if (g_LPSClosedVersusSurvivorStats.Length >= g_LPSClosedLifecycleQueueLimit.IntValue)
	{
		LPS_ReportVersusStatsQueueOverflow();
		return;
	}
	g_LPSClosedVersusSurvivorStats.PushArray(snapshot, sizeof(snapshot));
}

void LPS_QueueClosedVersusInfectedStats(LPSVersusInfectedStatsSnapshot snapshot)
{
	if (g_LPSClosedVersusInfectedStats.Length >= g_LPSClosedLifecycleQueueLimit.IntValue)
	{
		LPS_ReportVersusStatsQueueOverflow();
		return;
	}
	g_LPSClosedVersusInfectedStats.PushArray(snapshot, sizeof(snapshot));
}

void LPS_ReportVersusStatsQueueOverflow()
{
	if (g_LPSVersusStatsQueueOverflowReported)
	{
		return;
	}
	g_LPSVersusStatsQueueOverflowReported = true;
	LogError(
		"Closed Versus stats memory queue reached its limit (%d); additional snapshots will be dropped until a flush succeeds.",
		g_LPSClosedLifecycleQueueLimit.IntValue
	);
}

bool LPS_AddVersusStatsFlushQueries(Transaction transaction, int &queryCount, char[] error, int errorLength)
{
	LPS_ResetVersusStatsFlushCapture();
	g_LPSFlushClosedVersusSurvivorStatsCount = g_LPSClosedVersusSurvivorStats.Length;
	g_LPSFlushClosedVersusInfectedStatsCount = g_LPSClosedVersusInfectedStats.Length;

	LPSVersusSurvivorStatsSnapshot survivorSnapshot;
	for (int index = 0; index < g_LPSFlushClosedVersusSurvivorStatsCount; index++)
	{
		g_LPSClosedVersusSurvivorStats.GetArray(index, survivorSnapshot, sizeof(survivorSnapshot));
		if (!LPS_AddVersusSurvivorSnapshotQuery(transaction, survivorSnapshot, queryCount, error, errorLength))
		{
			return false;
		}
	}

	LPSVersusInfectedStatsSnapshot infectedSnapshot;
	for (int index = 0; index < g_LPSFlushClosedVersusInfectedStatsCount; index++)
	{
		g_LPSClosedVersusInfectedStats.GetArray(index, infectedSnapshot, sizeof(infectedSnapshot));
		if (!LPS_AddVersusInfectedSnapshotQuery(transaction, infectedSnapshot, queryCount, error, errorLength))
		{
			return false;
		}
	}

	for (int client = 1; client <= MaxClients; client++)
	{
		if (g_LPSClientVersusSurvivorStatsActive[client]
			&& g_LPSClientVersusSurvivorStatsDirty[client])
		{
			LPS_FillClientVersusSurvivorSnapshot(client, survivorSnapshot, false);
			if (!LPS_AddVersusSurvivorSnapshotQuery(transaction, survivorSnapshot, queryCount, error, errorLength))
			{
				return false;
			}
			g_LPSFlushCapturedVersusSurvivorStats[client] = true;
			g_LPSFlushCapturedVersusSurvivorRevision[client] = survivorSnapshot.revision;
			strcopy(
				g_LPSFlushCapturedVersusSurvivorSegmentId[client],
				sizeof(g_LPSFlushCapturedVersusSurvivorSegmentId[]),
				survivorSnapshot.segmentId
			);
		}

		if (g_LPSClientVersusInfectedStatsActive[client]
			&& g_LPSClientVersusInfectedStatsDirty[client])
		{
			LPS_FillClientVersusInfectedSnapshot(client, infectedSnapshot, false);
			if (!LPS_AddVersusInfectedSnapshotQuery(transaction, infectedSnapshot, queryCount, error, errorLength))
			{
				return false;
			}
			g_LPSFlushCapturedVersusInfectedStats[client] = true;
			g_LPSFlushCapturedVersusInfectedRevision[client] = infectedSnapshot.revision;
			strcopy(
				g_LPSFlushCapturedVersusInfectedSegmentId[client],
				sizeof(g_LPSFlushCapturedVersusInfectedSegmentId[]),
				infectedSnapshot.segmentId
			);
		}
	}
	return true;
}

bool LPS_AddVersusSurvivorSnapshotQuery(
	Transaction transaction,
	LPSVersusSurvivorStatsSnapshot snapshot,
	int &queryCount,
	char[] error,
	int errorLength
)
{
	char query[LPS_MAX_QUERY_LENGTH];
	if (!LPS_BuildVersusSurvivorUpsertQuery(snapshot, query, sizeof(query)))
	{
		strcopy(error, errorLength, "a Versus survivor stats value exceeded its SQL escape buffer");
		return false;
	}
	transaction.AddQuery(query);
	queryCount++;

	if (!LPS_BuildVersusSurvivorClassUpsertQuery(snapshot, query, sizeof(query)))
	{
		strcopy(error, errorLength, "a Versus survivor class stats value exceeded its SQL buffer");
		return false;
	}
	if (query[0] != '\0')
	{
		transaction.AddQuery(query);
		queryCount++;
	}
	return true;
}

bool LPS_AddVersusInfectedSnapshotQuery(
	Transaction transaction,
	LPSVersusInfectedStatsSnapshot snapshot,
	int &queryCount,
	char[] error,
	int errorLength
)
{
	char query[LPS_MAX_QUERY_LENGTH];
	if (!LPS_BuildVersusInfectedUpsertQuery(snapshot, query, sizeof(query)))
	{
		strcopy(error, errorLength, "a Versus infected stats value exceeded its SQL escape buffer");
		return false;
	}
	transaction.AddQuery(query);
	queryCount++;

	if (!LPS_BuildVersusInfectedClassUpsertQuery(snapshot, query, sizeof(query)))
	{
		strcopy(error, errorLength, "a Versus infected class stats value exceeded its SQL buffer");
		return false;
	}
	if (query[0] != '\0')
	{
		transaction.AddQuery(query);
		queryCount++;
	}
	return true;
}

bool LPS_BuildVersusSurvivorUpsertQuery(
	LPSVersusSurvivorStatsSnapshot snapshot,
	char[] query,
	int maxLength
)
{
	char segmentId[(LPS_MAX_SEGMENT_ID * 2) + 1];
	if (!g_LPSDatabase.Escape(snapshot.segmentId, segmentId, sizeof(segmentId)))
	{
		return false;
	}

	char values[2048];
	int written = FormatEx(values, sizeof(values), "'%s', %d, %d", segmentId, LPS_STATS_VERSION, GetTime());
	written += FormatEx(
		values[written],
		sizeof(values) - written,
		", %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d",
		snapshot.commonKills,
		snapshot.humanSpecialKills,
		snapshot.botSpecialKills,
		snapshot.humanTankKills,
		snapshot.botTankKills,
		snapshot.damageToHumanSpecial,
		snapshot.damageToBotSpecial,
		snapshot.damageToHumanTank,
		snapshot.damageToBotTank,
		snapshot.damageTakenInfected,
		snapshot.friendlyFireToHumans,
		snapshot.friendlyFireToBots,
		snapshot.friendlyFireTaken
	);
	written += FormatEx(
		values[written],
		sizeof(values) - written,
		", %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d",
		snapshot.incapacitations,
		snapshot.deaths,
		snapshot.incapRevives,
		snapshot.ledgeRescues,
		snapshot.defibRevives,
		snapshot.rescuesReceived,
		snapshot.medkitsUsedSelf,
		snapshot.medkitsUsedOnOthers,
		snapshot.medkitHealingSelf,
		snapshot.medkitHealingOthers,
		snapshot.pillsUsed,
		snapshot.adrenalineUsed,
		snapshot.temporaryHealthReceived
	);
	written += FormatEx(
		values[written],
		sizeof(values) - written,
		", %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d",
		snapshot.witchKills,
		snapshot.damageToWitch,
		snapshot.molotovsThrown,
		snapshot.pipeBombsThrown,
		snapshot.vomitJarsThrown,
		snapshot.incendiaryPacksDeployed,
		snapshot.explosivePacksDeployed,
		snapshot.meleeTongueSelfCuts,
		snapshot.tankRocksDestroyed,
		snapshot.witchOneShots,
		snapshot.witchSoloKills,
		snapshot.revision
	);
	if (written >= sizeof(values) - 1)
	{
		return false;
	}

	char columns[] = "segment_id, stats_version, last_saved_at, common_kills, human_special_kills, bot_special_kills, human_tank_kills, bot_tank_kills, damage_to_human_special, damage_to_bot_special, damage_to_human_tank, damage_to_bot_tank, damage_taken_infected, friendly_fire_to_humans, friendly_fire_to_bots, friendly_fire_taken, incapacitations, deaths, incap_revives, ledge_rescues, defib_revives, rescues_received, medkits_used_self, medkits_used_on_others, medkit_healing_self, medkit_healing_others, pills_used, adrenaline_used, temporary_health_received, witch_kills, damage_to_witch, molotovs_thrown, pipe_bombs_thrown, vomit_jars_thrown, incendiary_packs_deployed, explosive_packs_deployed, melee_tongue_self_cuts, tank_rocks_destroyed, witch_oneshots, witch_solo_kills, revision";
	if (g_LPSDatabaseDialect == LPSDatabaseDialect_MySQL)
	{
		FormatEx(
			query,
			maxLength,
			"INSERT INTO lps_versus_survivor_stats (%s) VALUES (%s) ON DUPLICATE KEY UPDATE stats_version = VALUES(stats_version), last_saved_at = VALUES(last_saved_at), common_kills = VALUES(common_kills), human_special_kills = VALUES(human_special_kills), bot_special_kills = VALUES(bot_special_kills), human_tank_kills = VALUES(human_tank_kills), bot_tank_kills = VALUES(bot_tank_kills), damage_to_human_special = VALUES(damage_to_human_special), damage_to_bot_special = VALUES(damage_to_bot_special), damage_to_human_tank = VALUES(damage_to_human_tank), damage_to_bot_tank = VALUES(damage_to_bot_tank), damage_taken_infected = VALUES(damage_taken_infected), friendly_fire_to_humans = VALUES(friendly_fire_to_humans), friendly_fire_to_bots = VALUES(friendly_fire_to_bots), friendly_fire_taken = VALUES(friendly_fire_taken), incapacitations = VALUES(incapacitations), deaths = VALUES(deaths), incap_revives = VALUES(incap_revives), ledge_rescues = VALUES(ledge_rescues), defib_revives = VALUES(defib_revives), rescues_received = VALUES(rescues_received), medkits_used_self = VALUES(medkits_used_self), medkits_used_on_others = VALUES(medkits_used_on_others), medkit_healing_self = VALUES(medkit_healing_self), medkit_healing_others = VALUES(medkit_healing_others), pills_used = VALUES(pills_used), adrenaline_used = VALUES(adrenaline_used), temporary_health_received = VALUES(temporary_health_received), witch_kills = VALUES(witch_kills), damage_to_witch = VALUES(damage_to_witch), molotovs_thrown = VALUES(molotovs_thrown), pipe_bombs_thrown = VALUES(pipe_bombs_thrown), vomit_jars_thrown = VALUES(vomit_jars_thrown), incendiary_packs_deployed = VALUES(incendiary_packs_deployed), explosive_packs_deployed = VALUES(explosive_packs_deployed), melee_tongue_self_cuts = VALUES(melee_tongue_self_cuts), tank_rocks_destroyed = VALUES(tank_rocks_destroyed), witch_oneshots = VALUES(witch_oneshots), witch_solo_kills = VALUES(witch_solo_kills), revision = VALUES(revision)",
			columns,
			values
		);
	}
	else
	{
		FormatEx(
			query,
			maxLength,
			"INSERT INTO lps_versus_survivor_stats (%s) VALUES (%s) ON CONFLICT(segment_id) DO UPDATE SET stats_version = excluded.stats_version, last_saved_at = excluded.last_saved_at, common_kills = excluded.common_kills, human_special_kills = excluded.human_special_kills, bot_special_kills = excluded.bot_special_kills, human_tank_kills = excluded.human_tank_kills, bot_tank_kills = excluded.bot_tank_kills, damage_to_human_special = excluded.damage_to_human_special, damage_to_bot_special = excluded.damage_to_bot_special, damage_to_human_tank = excluded.damage_to_human_tank, damage_to_bot_tank = excluded.damage_to_bot_tank, damage_taken_infected = excluded.damage_taken_infected, friendly_fire_to_humans = excluded.friendly_fire_to_humans, friendly_fire_to_bots = excluded.friendly_fire_to_bots, friendly_fire_taken = excluded.friendly_fire_taken, incapacitations = excluded.incapacitations, deaths = excluded.deaths, incap_revives = excluded.incap_revives, ledge_rescues = excluded.ledge_rescues, defib_revives = excluded.defib_revives, rescues_received = excluded.rescues_received, medkits_used_self = excluded.medkits_used_self, medkits_used_on_others = excluded.medkits_used_on_others, medkit_healing_self = excluded.medkit_healing_self, medkit_healing_others = excluded.medkit_healing_others, pills_used = excluded.pills_used, adrenaline_used = excluded.adrenaline_used, temporary_health_received = excluded.temporary_health_received, witch_kills = excluded.witch_kills, damage_to_witch = excluded.damage_to_witch, molotovs_thrown = excluded.molotovs_thrown, pipe_bombs_thrown = excluded.pipe_bombs_thrown, vomit_jars_thrown = excluded.vomit_jars_thrown, incendiary_packs_deployed = excluded.incendiary_packs_deployed, explosive_packs_deployed = excluded.explosive_packs_deployed, melee_tongue_self_cuts = excluded.melee_tongue_self_cuts, tank_rocks_destroyed = excluded.tank_rocks_destroyed, witch_oneshots = excluded.witch_oneshots, witch_solo_kills = excluded.witch_solo_kills, revision = excluded.revision",
			columns,
			values
		);
	}
	return true;
}

bool LPS_BuildVersusSurvivorClassUpsertQuery(
	LPSVersusSurvivorStatsSnapshot snapshot,
	char[] query,
	int maxLength
)
{
	query[0] = '\0';
	char segmentId[(LPS_MAX_SEGMENT_ID * 2) + 1];
	if (!g_LPSDatabase.Escape(snapshot.segmentId, segmentId, sizeof(segmentId)))
	{
		return false;
	}

	char values[2048];
	values[0] = '\0';
	int written = 0;
	int rowCount = 0;
	int savedAt = GetTime();
	for (int classIndex = 0;
		classIndex < view_as<int>(LPSVersusInfectedClass_Count);
		classIndex++)
	{
		int base = classIndex * view_as<int>(LPSVersusSurvivorClassStat_Count);
		bool hasValues = false;
		for (int statIndex = 0;
			statIndex < view_as<int>(LPSVersusSurvivorClassStat_Count);
			statIndex++)
		{
			if (snapshot.classStats[base + statIndex] > 0)
			{
				hasValues = true;
				break;
			}
		}
		if (!hasValues)
		{
			continue;
		}

		written += FormatEx(
			values[written],
			sizeof(values) - written,
			"%s('%s', %d, %d, %d, %d, %d, %d, %d, %d)",
			rowCount > 0 ? ", " : "",
			segmentId,
			LPS_GetVersusInfectedZombieClass(classIndex),
			LPS_STATS_VERSION,
			savedAt,
			snapshot.classStats[base + view_as<int>(LPSVersusSurvivorClassStat_HumanControllerKills)],
			snapshot.classStats[base + view_as<int>(LPSVersusSurvivorClassStat_BotControllerKills)],
			snapshot.classStats[base + view_as<int>(LPSVersusSurvivorClassStat_DamageToHumanControllers)],
			snapshot.classStats[base + view_as<int>(LPSVersusSurvivorClassStat_DamageToBotControllers)],
			snapshot.revision
		);
		if (written >= sizeof(values) - 1)
		{
			return false;
		}
		rowCount++;
	}

	if (rowCount == 0)
	{
		return true;
	}

	char columns[] = "segment_id, infected_class, stats_version, last_saved_at, human_controller_kills, bot_controller_kills, damage_to_human_controllers, damage_to_bot_controllers, revision";
	if (g_LPSDatabaseDialect == LPSDatabaseDialect_MySQL)
	{
		FormatEx(
			query,
			maxLength,
			"INSERT INTO lps_versus_survivor_infected_class_stats (%s) VALUES %s ON DUPLICATE KEY UPDATE stats_version = VALUES(stats_version), last_saved_at = VALUES(last_saved_at), human_controller_kills = VALUES(human_controller_kills), bot_controller_kills = VALUES(bot_controller_kills), damage_to_human_controllers = VALUES(damage_to_human_controllers), damage_to_bot_controllers = VALUES(damage_to_bot_controllers), revision = VALUES(revision)",
			columns,
			values
		);
	}
	else
	{
		FormatEx(
			query,
			maxLength,
			"INSERT INTO lps_versus_survivor_infected_class_stats (%s) VALUES %s ON CONFLICT(segment_id, infected_class) DO UPDATE SET stats_version = excluded.stats_version, last_saved_at = excluded.last_saved_at, human_controller_kills = excluded.human_controller_kills, bot_controller_kills = excluded.bot_controller_kills, damage_to_human_controllers = excluded.damage_to_human_controllers, damage_to_bot_controllers = excluded.damage_to_bot_controllers, revision = excluded.revision",
			columns,
			values
		);
	}
	return query[0] != '\0';
}

bool LPS_BuildVersusInfectedUpsertQuery(
	LPSVersusInfectedStatsSnapshot snapshot,
	char[] query,
	int maxLength
)
{
	char segmentId[(LPS_MAX_SEGMENT_ID * 2) + 1];
	if (!g_LPSDatabase.Escape(snapshot.segmentId, segmentId, sizeof(segmentId)))
	{
		return false;
	}

	char columns[] = "segment_id, stats_version, last_saved_at, spawn_count, damage_to_human_survivors, damage_to_bot_survivors, human_survivor_incaps, bot_survivor_incaps, human_survivor_kills, bot_survivor_kills, revision";
	char values[1024];
	FormatEx(
		values,
		sizeof(values),
		"'%s', %d, %d, %d, %d, %d, %d, %d, %d, %d, %d",
		segmentId,
		LPS_STATS_VERSION,
		GetTime(),
		snapshot.spawnCount,
		snapshot.damageToHumanSurvivors,
		snapshot.damageToBotSurvivors,
		snapshot.humanSurvivorIncaps,
		snapshot.botSurvivorIncaps,
		snapshot.humanSurvivorKills,
		snapshot.botSurvivorKills,
		snapshot.revision
	);

	if (g_LPSDatabaseDialect == LPSDatabaseDialect_MySQL)
	{
		FormatEx(
			query,
			maxLength,
			"INSERT INTO lps_versus_infected_stats (%s) VALUES (%s) ON DUPLICATE KEY UPDATE stats_version = VALUES(stats_version), last_saved_at = VALUES(last_saved_at), spawn_count = VALUES(spawn_count), damage_to_human_survivors = VALUES(damage_to_human_survivors), damage_to_bot_survivors = VALUES(damage_to_bot_survivors), human_survivor_incaps = VALUES(human_survivor_incaps), bot_survivor_incaps = VALUES(bot_survivor_incaps), human_survivor_kills = VALUES(human_survivor_kills), bot_survivor_kills = VALUES(bot_survivor_kills), revision = VALUES(revision)",
			columns,
			values
		);
	}
	else
	{
		FormatEx(
			query,
			maxLength,
			"INSERT INTO lps_versus_infected_stats (%s) VALUES (%s) ON CONFLICT(segment_id) DO UPDATE SET stats_version = excluded.stats_version, last_saved_at = excluded.last_saved_at, spawn_count = excluded.spawn_count, damage_to_human_survivors = excluded.damage_to_human_survivors, damage_to_bot_survivors = excluded.damage_to_bot_survivors, human_survivor_incaps = excluded.human_survivor_incaps, bot_survivor_incaps = excluded.bot_survivor_incaps, human_survivor_kills = excluded.human_survivor_kills, bot_survivor_kills = excluded.bot_survivor_kills, revision = excluded.revision",
			columns,
			values
		);
	}
	return true;
}

bool LPS_BuildVersusInfectedClassUpsertQuery(
	LPSVersusInfectedStatsSnapshot snapshot,
	char[] query,
	int maxLength
)
{
	query[0] = '\0';
	char segmentId[(LPS_MAX_SEGMENT_ID * 2) + 1];
	if (!g_LPSDatabase.Escape(snapshot.segmentId, segmentId, sizeof(segmentId)))
	{
		return false;
	}

	char values[4096];
	values[0] = '\0';
	int written = 0;
	int rowCount = 0;
	int savedAt = GetTime();
	for (int classIndex = 0;
		classIndex < view_as<int>(LPSVersusInfectedClass_Count);
		classIndex++)
	{
		int base = classIndex * view_as<int>(LPSVersusInfectedStat_Count);
		int abilityBase = classIndex * view_as<int>(LPSVersusInfectedAbilityStat_Count);
		bool hasValues = false;
		for (int statIndex = 0;
			statIndex < view_as<int>(LPSVersusInfectedStat_Count);
			statIndex++)
		{
			if (snapshot.classStats[base + statIndex] > 0)
			{
				hasValues = true;
				break;
			}
		}
		if (!hasValues)
		{
			for (int statIndex = 0;
				statIndex < view_as<int>(LPSVersusInfectedAbilityStat_Count);
				statIndex++)
			{
				if (snapshot.classAbilityStats[abilityBase + statIndex] > 0)
				{
					hasValues = true;
					break;
				}
			}
		}
		if (!hasValues)
		{
			continue;
		}

		written += FormatEx(
			values[written],
			sizeof(values) - written,
			"%s('%s', %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d)",
			rowCount > 0 ? ", " : "",
			segmentId,
			LPS_GetVersusInfectedZombieClass(classIndex),
			LPS_STATS_VERSION,
			savedAt,
			snapshot.classStats[base + view_as<int>(LPSVersusInfectedStat_SpawnCount)],
			snapshot.classStats[base + view_as<int>(LPSVersusInfectedStat_DamageToHumanSurvivors)],
			snapshot.classStats[base + view_as<int>(LPSVersusInfectedStat_DamageToBotSurvivors)],
			snapshot.classStats[base + view_as<int>(LPSVersusInfectedStat_HumanSurvivorIncaps)],
			snapshot.classStats[base + view_as<int>(LPSVersusInfectedStat_BotSurvivorIncaps)],
			snapshot.classStats[base + view_as<int>(LPSVersusInfectedStat_HumanSurvivorKills)],
			snapshot.classStats[base + view_as<int>(LPSVersusInfectedStat_BotSurvivorKills)],
			snapshot.classAbilityStats[abilityBase + view_as<int>(LPSVersusInfectedAbilityStat_HumanSurvivorControls)],
			snapshot.classAbilityStats[abilityBase + view_as<int>(LPSVersusInfectedAbilityStat_BotSurvivorControls)],
			snapshot.classAbilityStats[abilityBase + view_as<int>(LPSVersusInfectedAbilityStat_HumanSurvivorControlSeconds)],
			snapshot.classAbilityStats[abilityBase + view_as<int>(LPSVersusInfectedAbilityStat_BotSurvivorControlSeconds)],
			snapshot.classAbilityStats[abilityBase + view_as<int>(LPSVersusInfectedAbilityStat_HumanSurvivorAbilityHits)],
			snapshot.classAbilityStats[abilityBase + view_as<int>(LPSVersusInfectedAbilityStat_BotSurvivorAbilityHits)],
			snapshot.classAbilityStats[abilityBase + view_as<int>(LPSVersusInfectedAbilityStat_HumanSurvivorAbilityDamage)],
			snapshot.classAbilityStats[abilityBase + view_as<int>(LPSVersusInfectedAbilityStat_BotSurvivorAbilityDamage)],
			snapshot.revision
		);
		if (written >= sizeof(values) - 1)
		{
			return false;
		}
		rowCount++;
	}

	if (rowCount == 0)
	{
		return true;
	}

	char columns[] = "segment_id, infected_class, stats_version, last_saved_at, spawn_count, damage_to_human_survivors, damage_to_bot_survivors, human_survivor_incaps, bot_survivor_incaps, human_survivor_kills, bot_survivor_kills, human_survivor_controls, bot_survivor_controls, human_survivor_control_seconds, bot_survivor_control_seconds, human_survivor_ability_hits, bot_survivor_ability_hits, human_survivor_ability_damage, bot_survivor_ability_damage, revision";
	if (g_LPSDatabaseDialect == LPSDatabaseDialect_MySQL)
	{
		FormatEx(
			query,
			maxLength,
			"INSERT INTO lps_versus_infected_class_stats (%s) VALUES %s ON DUPLICATE KEY UPDATE stats_version = VALUES(stats_version), last_saved_at = VALUES(last_saved_at), spawn_count = VALUES(spawn_count), damage_to_human_survivors = VALUES(damage_to_human_survivors), damage_to_bot_survivors = VALUES(damage_to_bot_survivors), human_survivor_incaps = VALUES(human_survivor_incaps), bot_survivor_incaps = VALUES(bot_survivor_incaps), human_survivor_kills = VALUES(human_survivor_kills), bot_survivor_kills = VALUES(bot_survivor_kills), human_survivor_controls = VALUES(human_survivor_controls), bot_survivor_controls = VALUES(bot_survivor_controls), human_survivor_control_seconds = VALUES(human_survivor_control_seconds), bot_survivor_control_seconds = VALUES(bot_survivor_control_seconds), human_survivor_ability_hits = VALUES(human_survivor_ability_hits), bot_survivor_ability_hits = VALUES(bot_survivor_ability_hits), human_survivor_ability_damage = VALUES(human_survivor_ability_damage), bot_survivor_ability_damage = VALUES(bot_survivor_ability_damage), revision = VALUES(revision)",
			columns,
			values
		);
	}
	else
	{
		FormatEx(
			query,
			maxLength,
			"INSERT INTO lps_versus_infected_class_stats (%s) VALUES %s ON CONFLICT(segment_id, infected_class) DO UPDATE SET stats_version = excluded.stats_version, last_saved_at = excluded.last_saved_at, spawn_count = excluded.spawn_count, damage_to_human_survivors = excluded.damage_to_human_survivors, damage_to_bot_survivors = excluded.damage_to_bot_survivors, human_survivor_incaps = excluded.human_survivor_incaps, bot_survivor_incaps = excluded.bot_survivor_incaps, human_survivor_kills = excluded.human_survivor_kills, bot_survivor_kills = excluded.bot_survivor_kills, human_survivor_controls = excluded.human_survivor_controls, bot_survivor_controls = excluded.bot_survivor_controls, human_survivor_control_seconds = excluded.human_survivor_control_seconds, bot_survivor_control_seconds = excluded.bot_survivor_control_seconds, human_survivor_ability_hits = excluded.human_survivor_ability_hits, bot_survivor_ability_hits = excluded.bot_survivor_ability_hits, human_survivor_ability_damage = excluded.human_survivor_ability_damage, bot_survivor_ability_damage = excluded.bot_survivor_ability_damage, revision = excluded.revision",
			columns,
			values
		);
	}
	return query[0] != '\0';
}

void LPS_OnVersusStatsFlushSucceeded()
{
	for (int index = 0;
		index < g_LPSFlushClosedVersusSurvivorStatsCount
			&& g_LPSClosedVersusSurvivorStats.Length > 0;
		index++)
	{
		g_LPSClosedVersusSurvivorStats.Erase(0);
	}

	for (int index = 0;
		index < g_LPSFlushClosedVersusInfectedStatsCount
			&& g_LPSClosedVersusInfectedStats.Length > 0;
		index++)
	{
		g_LPSClosedVersusInfectedStats.Erase(0);
	}

	if (g_LPSClosedVersusSurvivorStats.Length < g_LPSClosedLifecycleQueueLimit.IntValue
		&& g_LPSClosedVersusInfectedStats.Length < g_LPSClosedLifecycleQueueLimit.IntValue)
	{
		g_LPSVersusStatsQueueOverflowReported = false;
	}

	for (int client = 1; client <= MaxClients; client++)
	{
		if (g_LPSFlushCapturedVersusSurvivorStats[client]
			&& g_LPSClientVersusSurvivorStatsActive[client]
			&& StrEqual(
				g_LPSClientVersusSurvivorStats[client].segmentId,
				g_LPSFlushCapturedVersusSurvivorSegmentId[client],
				true
			))
		{
			g_LPSClientVersusSurvivorStatsPersistedRevision[client] = g_LPSFlushCapturedVersusSurvivorRevision[client];
			if (g_LPSClientVersusSurvivorStats[client].revision
				<= g_LPSFlushCapturedVersusSurvivorRevision[client])
			{
				g_LPSClientVersusSurvivorStatsDirty[client] = false;
			}
		}

		if (g_LPSFlushCapturedVersusInfectedStats[client]
			&& g_LPSClientVersusInfectedStatsActive[client]
			&& StrEqual(
				g_LPSClientVersusInfectedStats[client].segmentId,
				g_LPSFlushCapturedVersusInfectedSegmentId[client],
				true
			))
		{
			g_LPSClientVersusInfectedStatsPersistedRevision[client] = g_LPSFlushCapturedVersusInfectedRevision[client];
			if (g_LPSClientVersusInfectedStats[client].revision
				<= g_LPSFlushCapturedVersusInfectedRevision[client])
			{
				g_LPSClientVersusInfectedStatsDirty[client] = false;
			}
		}
	}
	LPS_ResetVersusStatsFlushCapture();
}

void LPS_ResetVersusStatsFlushCapture()
{
	g_LPSFlushClosedVersusSurvivorStatsCount = 0;
	g_LPSFlushClosedVersusInfectedStatsCount = 0;
	for (int client = 1; client <= MaxClients; client++)
	{
		g_LPSFlushCapturedVersusSurvivorStats[client] = false;
		g_LPSFlushCapturedVersusSurvivorSegmentId[client][0] = '\0';
		g_LPSFlushCapturedVersusSurvivorRevision[client] = 0;
		g_LPSFlushCapturedVersusInfectedStats[client] = false;
		g_LPSFlushCapturedVersusInfectedSegmentId[client][0] = '\0';
		g_LPSFlushCapturedVersusInfectedRevision[client] = 0;
	}
}

void LPS_ResetClientVersusStatsState(int client)
{
	if (client < 1 || client > MaxClients)
	{
		return;
	}

	g_LPSClientVersusSurvivorStatsActive[client] = false;
	g_LPSClientVersusSurvivorStatsDirty[client] = false;
	g_LPSClientVersusSurvivorStatsPersistedRevision[client] = 0;
	g_LPSClientVersusSurvivorStats[client].segmentId[0] = '\0';
	g_LPSClientVersusSurvivorStats[client].commonKills = 0;
	g_LPSClientVersusSurvivorStats[client].humanSpecialKills = 0;
	g_LPSClientVersusSurvivorStats[client].botSpecialKills = 0;
	g_LPSClientVersusSurvivorStats[client].humanTankKills = 0;
	g_LPSClientVersusSurvivorStats[client].botTankKills = 0;
	g_LPSClientVersusSurvivorStats[client].damageToHumanSpecial = 0;
	g_LPSClientVersusSurvivorStats[client].damageToBotSpecial = 0;
	g_LPSClientVersusSurvivorStats[client].damageToHumanTank = 0;
	g_LPSClientVersusSurvivorStats[client].damageToBotTank = 0;
	g_LPSClientVersusSurvivorStats[client].damageTakenInfected = 0;
	g_LPSClientVersusSurvivorStats[client].friendlyFireToHumans = 0;
	g_LPSClientVersusSurvivorStats[client].friendlyFireToBots = 0;
	g_LPSClientVersusSurvivorStats[client].friendlyFireTaken = 0;
	g_LPSClientVersusSurvivorStats[client].incapacitations = 0;
	g_LPSClientVersusSurvivorStats[client].deaths = 0;
	g_LPSClientVersusSurvivorStats[client].incapRevives = 0;
	g_LPSClientVersusSurvivorStats[client].ledgeRescues = 0;
	g_LPSClientVersusSurvivorStats[client].defibRevives = 0;
	g_LPSClientVersusSurvivorStats[client].rescuesReceived = 0;
	g_LPSClientVersusSurvivorStats[client].medkitsUsedSelf = 0;
	g_LPSClientVersusSurvivorStats[client].medkitsUsedOnOthers = 0;
	g_LPSClientVersusSurvivorStats[client].medkitHealingSelf = 0;
	g_LPSClientVersusSurvivorStats[client].medkitHealingOthers = 0;
	g_LPSClientVersusSurvivorStats[client].pillsUsed = 0;
	g_LPSClientVersusSurvivorStats[client].adrenalineUsed = 0;
	g_LPSClientVersusSurvivorStats[client].temporaryHealthReceived = 0;
	g_LPSClientVersusSurvivorStats[client].witchKills = 0;
	g_LPSClientVersusSurvivorStats[client].damageToWitch = 0;
	g_LPSClientVersusSurvivorStats[client].molotovsThrown = 0;
	g_LPSClientVersusSurvivorStats[client].pipeBombsThrown = 0;
	g_LPSClientVersusSurvivorStats[client].vomitJarsThrown = 0;
	g_LPSClientVersusSurvivorStats[client].incendiaryPacksDeployed = 0;
	g_LPSClientVersusSurvivorStats[client].explosivePacksDeployed = 0;
	g_LPSClientVersusSurvivorStats[client].meleeTongueSelfCuts = 0;
	g_LPSClientVersusSurvivorStats[client].tankRocksDestroyed = 0;
	g_LPSClientVersusSurvivorStats[client].witchOneShots = 0;
	g_LPSClientVersusSurvivorStats[client].witchSoloKills = 0;
	for (int index = 0; index < sizeof(g_LPSClientVersusSurvivorStats[].classStats); index++)
	{
		g_LPSClientVersusSurvivorStats[client].classStats[index] = 0;
	}
	g_LPSClientVersusSurvivorStats[client].revision = 0;

	g_LPSClientVersusInfectedStatsActive[client] = false;
	g_LPSClientVersusInfectedStatsDirty[client] = false;
	g_LPSClientVersusInfectedSpawnPending[client] = false;
	g_LPSClientVersusInfectedStatsPersistedRevision[client] = 0;
	g_LPSClientVersusInfectedStats[client].segmentId[0] = '\0';
	g_LPSClientVersusInfectedStats[client].spawnCount = 0;
	g_LPSClientVersusInfectedStats[client].damageToHumanSurvivors = 0;
	g_LPSClientVersusInfectedStats[client].damageToBotSurvivors = 0;
	g_LPSClientVersusInfectedStats[client].humanSurvivorIncaps = 0;
	g_LPSClientVersusInfectedStats[client].botSurvivorIncaps = 0;
	g_LPSClientVersusInfectedStats[client].humanSurvivorKills = 0;
	g_LPSClientVersusInfectedStats[client].botSurvivorKills = 0;
	for (int index = 0; index < sizeof(g_LPSClientVersusInfectedStats[].classStats); index++)
	{
		g_LPSClientVersusInfectedStats[client].classStats[index] = 0;
	}
	for (int index = 0; index < sizeof(g_LPSClientVersusInfectedStats[].classAbilityStats); index++)
	{
		g_LPSClientVersusInfectedStats[client].classAbilityStats[index] = 0;
	}
	g_LPSClientVersusInfectedStats[client].revision = 0;
}

int LPS_GetActiveVersusSurvivorStatsCount()
{
	int count = 0;
	for (int client = 1; client <= MaxClients; client++)
	{
		if (g_LPSClientVersusSurvivorStatsActive[client]) count++;
	}
	return count;
}

int LPS_GetActiveVersusInfectedStatsCount()
{
	int count = 0;
	for (int client = 1; client <= MaxClients; client++)
	{
		if (g_LPSClientVersusInfectedStatsActive[client]) count++;
	}
	return count;
}

int LPS_GetPendingClosedVersusSurvivorStatsCount()
{
	return g_LPSClosedVersusSurvivorStats == null ? 0 : g_LPSClosedVersusSurvivorStats.Length;
}

int LPS_GetPendingClosedVersusInfectedStatsCount()
{
	return g_LPSClosedVersusInfectedStats == null ? 0 : g_LPSClosedVersusInfectedStats.Length;
}

void LPS_ShutdownVersusStats()
{
	for (int client = 1; client <= MaxClients; client++)
	{
		g_LPSClientVersusInfectedSpawnPending[client] = false;
	}
	LPS_ResetVersusStatsFlushCapture();
}
