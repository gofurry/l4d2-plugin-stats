#if defined _l4d2_player_stats_incidents_included
	#endinput
#endif
#define _l4d2_player_stats_incidents_included

enum struct LPSIncidentSnapshot
{
	char roundId[LPS_MAX_ROUND_ID];
	int sequence;
	int type;
	int occurredAt;
	int roundOffsetMs;
	int durationMs;
	int actorKind;
	char actorSteamId[LPS_MAX_STEAM_ID];
	int targetKind;
	char targetSteamId[LPS_MAX_STEAM_ID];
	int helperKind;
	char helperSteamId[LPS_MAX_STEAM_ID];
	int infectedClass;
	int endReason;
	int detailFlags;
	int relatedSequence;
	bool hasPosition;
	int posX;
	int posY;
	int posZ;
	bool hasEndPosition;
	int endPosX;
	int endPosY;
	int endPosZ;
}

ArrayList g_LPSPendingIncidents;
int g_LPSIncidentSequence;
int g_LPSIncidentQueuePeak;
int g_LPSIncidentDroppedTotal;
bool g_LPSIncidentOverflowReported;
int g_LPSBossSpawnIncidentSequence[LPS_MAX_ENTITY_INDEX + 1];
int g_LPSBossSpawnIncidentOffsetMs[LPS_MAX_ENTITY_INDEX + 1];
bool g_LPSWitchStartleIncidentRecorded[LPS_MAX_ENTITY_INDEX + 1];
bool g_LPSControlIncidentActive[MAXPLAYERS + 1];
char g_LPSControlIncidentRoundId[MAXPLAYERS + 1][LPS_MAX_ROUND_ID];
float g_LPSControlIncidentStartedGameTime[MAXPLAYERS + 1];
int g_LPSControlIncidentOccurredAt[MAXPLAYERS + 1];
int g_LPSControlIncidentOffsetMs[MAXPLAYERS + 1];
LPSControlType g_LPSControlIncidentType[MAXPLAYERS + 1];
int g_LPSControlIncidentActorKind[MAXPLAYERS + 1];
char g_LPSControlIncidentActorSteamId[MAXPLAYERS + 1][LPS_MAX_STEAM_ID];
int g_LPSControlIncidentTargetKind[MAXPLAYERS + 1];
char g_LPSControlIncidentTargetSteamId[MAXPLAYERS + 1][LPS_MAX_STEAM_ID];
int g_LPSControlIncidentInfectedClass[MAXPLAYERS + 1];
bool g_LPSControlIncidentHasPosition[MAXPLAYERS + 1];
int g_LPSControlIncidentPosition[MAXPLAYERS + 1][3];

void LPS_InitializeIncidents()
{
	g_LPSPendingIncidents = new ArrayList(sizeof(LPSIncidentSnapshot));
	g_LPSIncidentSequence = 0;
	g_LPSIncidentQueuePeak = 0;
	g_LPSIncidentDroppedTotal = 0;
	g_LPSIncidentOverflowReported = false;
	for (int entity = 1; entity <= LPS_MAX_ENTITY_INDEX; entity++)
	{
		g_LPSBossSpawnIncidentSequence[entity] = 0;
		g_LPSBossSpawnIncidentOffsetMs[entity] = 0;
		g_LPSWitchStartleIncidentRecorded[entity] = false;
	}
	for (int client = 1; client <= MaxClients; client++)
	{
		LPS_ResetControlIncident(client);
	}

	HookEvent("player_incapacitated", LPS_Event_IncidentIncap, EventHookMode_Post);
	HookEvent("player_death", LPS_Event_IncidentDeath, EventHookMode_Post);
	HookEvent("revive_success", LPS_Event_IncidentRevive, EventHookMode_Post);
	HookEvent("defibrillator_used", LPS_Event_IncidentDefib, EventHookMode_Post);
	HookEvent("triggered_car_alarm", LPS_Event_IncidentCarAlarm, EventHookMode_Post);
	HookEvent("witch_harasser_set", LPS_Event_IncidentWitchStartle, EventHookMode_Post);
	HookEvent("heal_success", LPS_Event_IncidentMedkitHeal, EventHookMode_Post);
}

void LPS_OnIncidentRoundStarted()
{
	g_LPSIncidentSequence = 0;
	g_LPSIncidentOverflowReported = false;
	for (int entity = 1; entity <= LPS_MAX_ENTITY_INDEX; entity++)
	{
		g_LPSBossSpawnIncidentSequence[entity] = 0;
		g_LPSBossSpawnIncidentOffsetMs[entity] = 0;
		g_LPSWitchStartleIncidentRecorded[entity] = false;
	}
	for (int client = 1; client <= MaxClients; client++)
	{
		LPS_ResetControlIncident(client);
	}
}

bool LPS_CanCaptureIncident()
{
	return g_LPSRoundContextActive
		&& g_LPSActiveRoundContext.incidentCaptureEnabled
		&& g_LPSRoundActive;
}

int LPS_BeginIncident(LPSIncidentSnapshot incident, LPSIncidentType type)
{
	if (!LPS_CanCaptureIncident())
	{
		return 0;
	}
	g_LPSIncidentSequence++;
	g_LPSActiveRoundContext.incidentExpectedCount++;
	g_LPSActiveRoundContext.revision++;
	g_LPSRoundContextDirty = true;

	strcopy(incident.roundId, sizeof(incident.roundId), g_LPSRoundId);
	incident.sequence = g_LPSIncidentSequence;
	incident.type = view_as<int>(type);
	incident.occurredAt = GetTime();
	incident.roundOffsetMs = LPS_CurrentRoundOffsetMs();
	incident.durationMs = 0;
	incident.actorKind = view_as<int>(LPSParticipant_None);
	incident.actorSteamId[0] = '\0';
	incident.targetKind = view_as<int>(LPSParticipant_None);
	incident.targetSteamId[0] = '\0';
	incident.helperKind = view_as<int>(LPSParticipant_None);
	incident.helperSteamId[0] = '\0';
	incident.infectedClass = 0;
	incident.endReason = view_as<int>(LPSIncidentEnd_Unknown);
	incident.detailFlags = 0;
	incident.relatedSequence = 0;
	incident.hasPosition = false;
	incident.hasEndPosition = false;
	return incident.sequence;
}

bool LPS_QueueIncident(LPSIncidentSnapshot incident)
{
	if (g_LPSPendingIncidents.Length >= g_LPSIncidentQueueLimit.IntValue)
	{
		g_LPSActiveRoundContext.incidentDroppedCount++;
		g_LPSActiveRoundContext.revision++;
		g_LPSRoundContextDirty = true;
		g_LPSIncidentDroppedTotal++;
		if (!g_LPSIncidentOverflowReported)
		{
			LogError("Incident analysis queue reached %d rows; dropping new analysis facts without blocking core stats.", g_LPSIncidentQueueLimit.IntValue);
			g_LPSIncidentOverflowReported = true;
		}
		return false;
	}
	g_LPSPendingIncidents.PushArray(incident, sizeof(incident));
	if (g_LPSPendingIncidents.Length > g_LPSIncidentQueuePeak)
	{
		g_LPSIncidentQueuePeak = g_LPSPendingIncidents.Length;
	}
	return true;
}

int LPS_CurrentRoundOffsetMs()
{
	if (g_LPSRoundStartedGameTime <= 0.0)
	{
		return 0;
	}
	float elapsed = GetGameTime() - g_LPSRoundStartedGameTime;
	return elapsed > 0.0 ? RoundToNearest(elapsed * 1000.0) : 0;
}

LPSParticipantKind LPS_ResolveParticipant(int client, char[] steamId, int maxLength, bool required)
{
	steamId[0] = '\0';
	if (client < 1 || client > MaxClients || !IsClientInGame(client))
	{
		return required ? LPSParticipant_Unknown : LPSParticipant_None;
	}
	int team = GetClientTeam(client);
	if (IsFakeClient(client))
	{
		if (team == LPS_TEAM_SURVIVOR) return LPSParticipant_BotSurvivor;
		if (team == LPS_TEAM_INFECTED) return LPSParticipant_BotInfected;
		return LPSParticipant_Other;
	}
	if (!g_LPSClientAuthorized[client] || g_LPSClientSteamId[client][0] == '\0')
	{
		return required ? LPSParticipant_Unknown : LPSParticipant_Other;
	}
	strcopy(steamId, maxLength, g_LPSClientSteamId[client]);
	if (team == LPS_TEAM_SURVIVOR) return LPSParticipant_HumanSurvivor;
	if (team == LPS_TEAM_INFECTED) return LPSParticipant_HumanInfected;
	return LPSParticipant_Other;
}

void LPS_CaptureIncidentPosition(int entity, LPSIncidentSnapshot incident, bool ending)
{
	if (entity < 1 || entity > LPS_MAX_ENTITY_INDEX || !IsValidEntity(entity))
	{
		return;
	}
	float origin[3];
	GetEntPropVector(entity, Prop_Send, "m_vecOrigin", origin);
	if (ending)
	{
		incident.hasEndPosition = true;
		incident.endPosX = RoundToNearest(origin[0]);
		incident.endPosY = RoundToNearest(origin[1]);
		incident.endPosZ = RoundToNearest(origin[2]);
	}
	else
	{
		incident.hasPosition = true;
		incident.posX = RoundToNearest(origin[0]);
		incident.posY = RoundToNearest(origin[1]);
		incident.posZ = RoundToNearest(origin[2]);
	}
}

void LPS_RecordPointIncident(LPSIncidentType type, int actor, int target, bool actorRequired, bool targetRequired, int infectedClass = 0, int detailFlags = 0)
{
	char actorSteamId[LPS_MAX_STEAM_ID];
	char targetSteamId[LPS_MAX_STEAM_ID];
	int actorKind = view_as<int>(LPS_ResolveParticipant(actor, actorSteamId, sizeof(actorSteamId), actorRequired));
	int targetKind = view_as<int>(LPS_ResolveParticipant(target, targetSteamId, sizeof(targetSteamId), targetRequired));
	bool hasHuman = actorKind == view_as<int>(LPSParticipant_HumanSurvivor)
		|| actorKind == view_as<int>(LPSParticipant_HumanInfected)
		|| targetKind == view_as<int>(LPSParticipant_HumanSurvivor)
		|| targetKind == view_as<int>(LPSParticipant_HumanInfected);
	if (!hasHuman)
	{
		return;
	}

	LPSIncidentSnapshot incident;
	if (LPS_BeginIncident(incident, type) == 0) return;
	incident.actorKind = actorKind;
	strcopy(incident.actorSteamId, sizeof(incident.actorSteamId), actorSteamId);
	incident.targetKind = targetKind;
	strcopy(incident.targetSteamId, sizeof(incident.targetSteamId), targetSteamId);
	incident.infectedClass = infectedClass;
	incident.detailFlags = detailFlags;
	LPS_CaptureIncidentPosition(target, incident, false);
	LPS_QueueIncident(incident);
}

void LPS_DiscardPendingIncidentsForRound(const char[] roundId)
{
	LPSIncidentSnapshot incident;
	for (int index = g_LPSPendingIncidents.Length - 1; index >= 0; index--)
	{
		g_LPSPendingIncidents.GetArray(index, incident, sizeof(incident));
		if (StrEqual(incident.roundId, roundId, true))
		{
			g_LPSPendingIncidents.Erase(index);
			g_LPSIncidentDroppedTotal++;
		}
	}
}

public void LPS_Event_IncidentIncap(Event event, const char[] name, bool dontBroadcast)
{
	int target = GetClientOfUserId(event.GetInt("userid"));
	if (target >= 1 && target <= MaxClients && IsClientInGame(target) && GetClientTeam(target) == LPS_TEAM_SURVIVOR)
	{
		int actor = GetClientOfUserId(event.GetInt("attacker"));
		int infectedClass = actor >= 1 && actor <= MaxClients && IsClientInGame(actor) ? LPS_GetZombieClass(actor) : 0;
		LPS_RecordPointIncident(LPSIncident_SurvivorIncap, actor, target, true, true, infectedClass);
	}
}

public void LPS_Event_IncidentDeath(Event event, const char[] name, bool dontBroadcast)
{
	if (event.GetBool("abort")) return;
	int target = GetClientOfUserId(event.GetInt("userid"));
	if (target >= 1 && target <= MaxClients && IsClientInGame(target) && GetClientTeam(target) == LPS_TEAM_SURVIVOR)
	{
		int actor = GetClientOfUserId(event.GetInt("attacker"));
		LPS_RecordPointIncident(LPSIncident_SurvivorDeath, actor, target, true, true);
	}
}

public void LPS_Event_IncidentRevive(Event event, const char[] name, bool dontBroadcast)
{
	LPS_RecordPointIncident(event.GetBool("ledge_hang") ? LPSIncident_LedgeRescue : LPSIncident_SurvivorRevive, GetClientOfUserId(event.GetInt("userid")), GetClientOfUserId(event.GetInt("subject")), true, true);
}

public void LPS_Event_IncidentDefib(Event event, const char[] name, bool dontBroadcast)
{
	LPS_RecordPointIncident(LPSIncident_DefibRevive, GetClientOfUserId(event.GetInt("userid")), GetClientOfUserId(event.GetInt("subject")), true, true);
}

public void LPS_Event_IncidentCarAlarm(Event event, const char[] name, bool dontBroadcast)
{
	int client = GetClientOfUserId(event.GetInt("userid"));
	if (LPS_IsHumanSurvivorClient(client))
	{
		LPS_RecordPointIncident(LPSIncident_CarAlarm, client, client, true, true);
	}
}

public void LPS_Event_IncidentWitchStartle(Event event, const char[] name, bool dontBroadcast)
{
	int witch = event.GetInt("witchid");
	int actor = GetClientOfUserId(event.GetInt("userid"));
	if (witch < 1
		|| witch > LPS_MAX_ENTITY_INDEX
		|| g_LPSWitchStartleIncidentRecorded[witch]
		|| !LPS_IsHumanSurvivorClient(actor))
	{
		return;
	}

	LPSIncidentSnapshot incident;
	if (LPS_BeginIncident(incident, LPSIncident_WitchStartle) == 0)
	{
		return;
	}
	incident.actorKind = view_as<int>(LPS_ResolveParticipant(
		actor,
		incident.actorSteamId,
		sizeof(incident.actorSteamId),
		true
	));
	incident.targetKind = view_as<int>(LPSParticipant_Witch);
	incident.relatedSequence = g_LPSBossSpawnIncidentSequence[witch];
	if (incident.relatedSequence > 0
		&& incident.roundOffsetMs >= g_LPSBossSpawnIncidentOffsetMs[witch])
	{
		incident.durationMs = incident.roundOffsetMs - g_LPSBossSpawnIncidentOffsetMs[witch];
	}
	LPS_CaptureIncidentPosition(witch, incident, false);
	if (LPS_QueueIncident(incident))
	{
		g_LPSWitchStartleIncidentRecorded[witch] = true;
	}
}

public void LPS_Event_IncidentMedkitHeal(Event event, const char[] name, bool dontBroadcast)
{
	LPS_RecordPointIncident(
		LPSIncident_MedkitHeal,
		GetClientOfUserId(event.GetInt("userid")),
		GetClientOfUserId(event.GetInt("subject")),
		true,
		true
	);
}

void LPS_RecordObjectiveCompleteIncident(int actor, int objective)
{
	if (!LPS_IsHumanSurvivorClient(actor))
	{
		return;
	}

	LPSIncidentSnapshot incident;
	if (LPS_BeginIncident(incident, LPSIncident_ObjectiveComplete) == 0)
	{
		return;
	}
	incident.actorKind = view_as<int>(LPS_ResolveParticipant(
		actor,
		incident.actorSteamId,
		sizeof(incident.actorSteamId),
		true
	));
	incident.targetKind = view_as<int>(LPSParticipant_Other);
	LPS_CaptureIncidentPosition(objective, incident, false);
	if (!incident.hasPosition)
	{
		LPS_CaptureIncidentPosition(actor, incident, false);
	}
	LPS_QueueIncident(incident);
}

int LPS_GetPendingIncidentCount()
{
	return g_LPSPendingIncidents == null ? 0 : g_LPSPendingIncidents.Length;
}

void LPS_StartControlIncident(int victim, int controller, LPSControlType type, int infectedClass)
{
	if (!LPS_CanCaptureIncident()
		|| victim < 1 || victim > MaxClients || !IsClientInGame(victim)
		|| controller < 1 || controller > MaxClients || !IsClientInGame(controller)
		|| GetClientTeam(victim) != LPS_TEAM_SURVIVOR
		|| GetClientTeam(controller) != LPS_TEAM_INFECTED)
	{
		return;
	}
	if (g_LPSControlIncidentActive[victim]
		&& g_LPSControlIncidentType[victim] == type
		&& g_LPSControlIncidentInfectedClass[victim] == infectedClass)
	{
		return;
	}
	LPS_EndControlIncident(victim, LPSControlType_None, LPSIncidentEnd_Released, 0);
	char actorSteamId[LPS_MAX_STEAM_ID];
	char targetSteamId[LPS_MAX_STEAM_ID];
	LPSParticipantKind actorKind = LPS_ResolveParticipant(controller, actorSteamId, sizeof(actorSteamId), true);
	LPSParticipantKind targetKind = LPS_ResolveParticipant(victim, targetSteamId, sizeof(targetSteamId), true);
	bool hasHuman = actorKind == LPSParticipant_HumanInfected
		|| targetKind == LPSParticipant_HumanSurvivor;
	if (!hasHuman)
	{
		return;
	}
	g_LPSControlIncidentActive[victim] = true;
	strcopy(g_LPSControlIncidentRoundId[victim], sizeof(g_LPSControlIncidentRoundId[]), g_LPSRoundId);
	g_LPSControlIncidentStartedGameTime[victim] = GetGameTime();
	g_LPSControlIncidentOccurredAt[victim] = GetTime();
	g_LPSControlIncidentOffsetMs[victim] = LPS_CurrentRoundOffsetMs();
	g_LPSControlIncidentType[victim] = type;
	g_LPSControlIncidentActorKind[victim] = view_as<int>(actorKind);
	strcopy(g_LPSControlIncidentActorSteamId[victim], sizeof(g_LPSControlIncidentActorSteamId[]), actorSteamId);
	g_LPSControlIncidentTargetKind[victim] = view_as<int>(targetKind);
	strcopy(g_LPSControlIncidentTargetSteamId[victim], sizeof(g_LPSControlIncidentTargetSteamId[]), targetSteamId);
	g_LPSControlIncidentInfectedClass[victim] = infectedClass;
	if (IsValidEntity(victim))
	{
		float origin[3];
		GetEntPropVector(victim, Prop_Send, "m_vecOrigin", origin);
		g_LPSControlIncidentHasPosition[victim] = true;
		for (int axis = 0; axis < 3; axis++)
		{
			g_LPSControlIncidentPosition[victim][axis] = RoundToNearest(origin[axis]);
		}
	}
}

void LPS_EndControlIncident(int victim, LPSControlType expectedType, LPSIncidentEndReason reason, int helper)
{
	if (victim < 1 || victim > MaxClients
		|| !g_LPSControlIncidentActive[victim]
		|| (expectedType != LPSControlType_None && g_LPSControlIncidentType[victim] != expectedType))
	{
		return;
	}
	if (!g_LPSRoundContextActive
		|| !StrEqual(g_LPSControlIncidentRoundId[victim], g_LPSRoundId, true))
	{
		LPS_ResetControlIncident(victim);
		return;
	}
	LPSIncidentSnapshot incident;
	if (LPS_BeginIncident(incident, LPSIncident_SurvivorControl) == 0)
	{
		LPS_ResetControlIncident(victim);
		return;
	}
	incident.occurredAt = g_LPSControlIncidentOccurredAt[victim];
	incident.roundOffsetMs = g_LPSControlIncidentOffsetMs[victim];
	float elapsed = GetGameTime() - g_LPSControlIncidentStartedGameTime[victim];
	incident.durationMs = elapsed > 0.0 ? RoundToNearest(elapsed * 1000.0) : 0;
	incident.actorKind = g_LPSControlIncidentActorKind[victim];
	strcopy(incident.actorSteamId, sizeof(incident.actorSteamId), g_LPSControlIncidentActorSteamId[victim]);
	incident.targetKind = g_LPSControlIncidentTargetKind[victim];
	strcopy(incident.targetSteamId, sizeof(incident.targetSteamId), g_LPSControlIncidentTargetSteamId[victim]);
	incident.infectedClass = g_LPSControlIncidentInfectedClass[victim];
	incident.endReason = view_as<int>(reason);
	incident.helperKind = view_as<int>(LPS_ResolveParticipant(helper, incident.helperSteamId, sizeof(incident.helperSteamId), false));
	incident.hasPosition = g_LPSControlIncidentHasPosition[victim];
	if (incident.hasPosition)
	{
		incident.posX = g_LPSControlIncidentPosition[victim][0];
		incident.posY = g_LPSControlIncidentPosition[victim][1];
		incident.posZ = g_LPSControlIncidentPosition[victim][2];
	}
	LPS_CaptureIncidentPosition(victim, incident, true);
	LPS_QueueIncident(incident);
	LPS_ResetControlIncident(victim);
}

void LPS_ResetControlIncident(int victim)
{
	if (victim < 1 || victim > MaxClients) return;
	g_LPSControlIncidentActive[victim] = false;
	g_LPSControlIncidentRoundId[victim][0] = '\0';
	g_LPSControlIncidentStartedGameTime[victim] = 0.0;
	g_LPSControlIncidentOccurredAt[victim] = 0;
	g_LPSControlIncidentOffsetMs[victim] = 0;
	g_LPSControlIncidentType[victim] = LPSControlType_None;
	g_LPSControlIncidentActorKind[victim] = view_as<int>(LPSParticipant_None);
	g_LPSControlIncidentActorSteamId[victim][0] = '\0';
	g_LPSControlIncidentTargetKind[victim] = view_as<int>(LPSParticipant_None);
	g_LPSControlIncidentTargetSteamId[victim][0] = '\0';
	g_LPSControlIncidentInfectedClass[victim] = 0;
	g_LPSControlIncidentHasPosition[victim] = false;
}

void LPS_RecordBossSpawnIncident(int entity, bool tank)
{
	if (!LPS_CanCaptureIncident() || !g_LPSRoundHadHumanSegment
		|| entity < 1 || entity > LPS_MAX_ENTITY_INDEX || !IsValidEntity(entity))
	{
		return;
	}
	LPSIncidentSnapshot incident;
	if (LPS_BeginIncident(incident, tank ? LPSIncident_TankSpawn : LPSIncident_WitchSpawn) == 0) return;
	if (tank)
	{
		incident.targetKind = view_as<int>(LPS_ResolveParticipant(entity, incident.targetSteamId, sizeof(incident.targetSteamId), true));
		incident.infectedClass = 8;
	}
	else
	{
		incident.targetKind = view_as<int>(LPSParticipant_Witch);
	}
	LPS_CaptureIncidentPosition(entity, incident, false);
	if (LPS_QueueIncident(incident))
	{
		g_LPSBossSpawnIncidentSequence[entity] = incident.sequence;
		g_LPSBossSpawnIncidentOffsetMs[entity] = incident.roundOffsetMs;
	}
}

void LPS_RecordBossDeathIncident(int entity, int attacker, bool tank, bool oneShot = false)
{
	if (!LPS_CanCaptureIncident() || !g_LPSRoundHadHumanSegment
		|| entity < 1 || entity > LPS_MAX_ENTITY_INDEX)
	{
		return;
	}
	LPSIncidentSnapshot incident;
	if (LPS_BeginIncident(incident, tank ? LPSIncident_TankDeath : LPSIncident_WitchDeath) == 0) return;
	incident.actorKind = view_as<int>(LPS_ResolveParticipant(attacker, incident.actorSteamId, sizeof(incident.actorSteamId), true));
	if (tank)
	{
		incident.targetKind = view_as<int>(LPS_ResolveParticipant(entity, incident.targetSteamId, sizeof(incident.targetSteamId), true));
		incident.infectedClass = 8;
	}
	else
	{
		incident.targetKind = view_as<int>(LPSParticipant_Witch);
		incident.detailFlags = oneShot ? 1 : 0;
	}
	incident.relatedSequence = g_LPSBossSpawnIncidentSequence[entity];
	if (incident.relatedSequence > 0 && incident.roundOffsetMs >= g_LPSBossSpawnIncidentOffsetMs[entity])
	{
		incident.durationMs = incident.roundOffsetMs - g_LPSBossSpawnIncidentOffsetMs[entity];
	}
	LPS_CaptureIncidentPosition(entity, incident, false);
	LPS_QueueIncident(incident);
	g_LPSBossSpawnIncidentSequence[entity] = 0;
	g_LPSBossSpawnIncidentOffsetMs[entity] = 0;
}

void LPS_OnIncidentEntityDestroyed(int entity)
{
	if (entity >= 1 && entity <= LPS_MAX_ENTITY_INDEX)
	{
		g_LPSBossSpawnIncidentSequence[entity] = 0;
		g_LPSBossSpawnIncidentOffsetMs[entity] = 0;
		g_LPSWitchStartleIncidentRecorded[entity] = false;
	}
}

void LPS_EndAllIncidentControls()
{
	for (int victim = 1; victim <= MaxClients; victim++)
	{
		if (g_LPSModeFamily == LPSModeFamily_PvE)
		{
			LPS_EndControl(victim, LPSIncidentEnd_LifecycleEnd);
		}
		else if (g_LPSModeFamily == LPSModeFamily_Versus)
		{
			LPS_EndVersusControl(victim, LPSControlType_None, LPSIncidentEnd_LifecycleEnd);
		}
		else if (g_LPSControlIncidentActive[victim])
		{
			LPS_EndControlIncident(victim, LPSControlType_None, LPSIncidentEnd_LifecycleEnd, 0);
		}
	}
}
