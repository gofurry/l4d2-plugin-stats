#if defined _l4d2_player_stats_pve_interactions_included
	#endinput
#endif
#define _l4d2_player_stats_pve_interactions_included

// Objective outputs are deliberately allowlisted. Generic player_use and
// finale_start are not used because they cannot prove a successful action.
bool g_LPSObjectiveEntityCounted[LPS_MAX_ENTITY_INDEX + 1];

bool g_LPSIncapacitatedTimerActive[MAXPLAYERS + 1];
bool g_LPSLedgeTimerActive[MAXPLAYERS + 1];
float g_LPSIncapacitatedStartedAt[MAXPLAYERS + 1];
float g_LPSLedgeStartedAt[MAXPLAYERS + 1];

int g_LPSPendingBlackWhiteSubjectUserId[MAXPLAYERS + 1];

void LPS_InitializePvEInteractions()
{
	HookEvent("ammo_pickup", LPS_Event_AmmoPickupStats, EventHookMode_Post);
	HookEvent("player_incapacitated", LPS_Event_PlayerIncapacitatedDuration, EventHookMode_Post);
	HookEvent("player_ledge_grab", LPS_Event_PlayerLedgeGrabDuration, EventHookMode_Post);
	HookEvent("player_ledge_release", LPS_Event_PlayerLedgeReleaseDuration, EventHookMode_Post);
	HookEvent("revive_success", LPS_Event_ReviveSuccessDuration, EventHookMode_Post);
	HookEvent("player_death", LPS_Event_PlayerDeathDuration, EventHookMode_Post);
	HookEvent("heal_begin", LPS_Event_HealBeginBlackWhite, EventHookMode_Post);
	HookEvent("heal_interrupted", LPS_Event_HealInterruptedBlackWhite, EventHookMode_Post);
	HookEvent("heal_success", LPS_Event_HealSuccessBlackWhite, EventHookMode_Post);

	HookEntityOutput("func_button", "OnIn", LPS_OnObjectiveInteractionCompleted);
	HookEntityOutput("func_rot_button", "OnIn", LPS_OnObjectiveInteractionCompleted);
	HookEntityOutput("func_button_timed", "OnTimeUp", LPS_OnObjectiveInteractionCompleted);
	HookEntityOutput("func_buildable_button", "OnTimeUp", LPS_OnObjectiveInteractionCompleted);
	HookEntityOutput("momentary_rot_button", "OnFullyClosed", LPS_OnObjectiveInteractionCompleted);
	HookEntityOutput("point_script_use_target", "OnUseFinished", LPS_OnObjectiveInteractionCompleted);

	LPS_ResetObjectiveInteractionTracking();
	for (int client = 1; client <= MaxClients; client++)
	{
		LPS_ResetClientPvEInteractionState(client);
	}
}

void LPS_ShutdownPvEInteractions()
{
	UnhookEntityOutput("func_button", "OnIn", LPS_OnObjectiveInteractionCompleted);
	UnhookEntityOutput("func_rot_button", "OnIn", LPS_OnObjectiveInteractionCompleted);
	UnhookEntityOutput("func_button_timed", "OnTimeUp", LPS_OnObjectiveInteractionCompleted);
	UnhookEntityOutput("func_buildable_button", "OnTimeUp", LPS_OnObjectiveInteractionCompleted);
	UnhookEntityOutput("momentary_rot_button", "OnFullyClosed", LPS_OnObjectiveInteractionCompleted);
	UnhookEntityOutput("point_script_use_target", "OnUseFinished", LPS_OnObjectiveInteractionCompleted);
}

void LPS_ResetObjectiveInteractionTracking()
{
	for (int entity = 0; entity <= LPS_MAX_ENTITY_INDEX; entity++)
	{
		g_LPSObjectiveEntityCounted[entity] = false;
	}
}

void LPS_OnInteractionEntityCreated(int entity)
{
	if (entity >= 1 && entity <= LPS_MAX_ENTITY_INDEX)
	{
		g_LPSObjectiveEntityCounted[entity] = false;
	}
}

void LPS_OnInteractionEntityDestroyed(int entity)
{
	if (entity >= 1 && entity <= LPS_MAX_ENTITY_INDEX)
	{
		g_LPSObjectiveEntityCounted[entity] = false;
	}
}

public void LPS_OnObjectiveInteractionCompleted(
	const char[] output,
	int caller,
	int activator,
	float delay
)
{
	if (caller < 1
		|| caller > LPS_MAX_ENTITY_INDEX
		|| g_LPSObjectiveEntityCounted[caller]
		|| !LPS_IsHumanSurvivorClient(activator))
	{
		return;
	}

	if (LPS_CanRecordPvEEvents())
	{
		g_LPSObjectiveEntityCounted[caller] = true;
		LPS_AddPvEStat(activator, LPSPvEStat_ObjectiveInteractions, 1);
	}
	else if (LPS_CanRecordVersusEvents())
	{
		g_LPSObjectiveEntityCounted[caller] = true;
		LPS_AddVersusSurvivorStat(activator, LPSVersusSurvivorStat_ObjectiveInteractions, 1);
	}

	if (g_LPSObjectiveEntityCounted[caller])
	{
		LPS_RecordObjectiveCompleteIncident(activator, caller);
	}
}

public void LPS_Event_AmmoPickupStats(Event event, const char[] name, bool dontBroadcast)
{
	if (!LPS_CanRecordPvEEvents())
	{
		return;
	}

	int client = GetClientOfUserId(event.GetInt("userid"));
	if (LPS_IsHumanSurvivorClient(client))
	{
		LPS_AddPvEStat(client, LPSPvEStat_AmmoPileUses, 1);
	}
}

public void LPS_Event_PlayerIncapacitatedDuration(Event event, const char[] name, bool dontBroadcast)
{
	int client = GetClientOfUserId(event.GetInt("userid"));
	if (!LPS_CanTrackPvEInteractionDuration(client))
	{
		return;
	}

	if (LPS_IsSurvivorHangingFromLedge(client))
	{
		LPS_EndIncapacitatedDuration(client);
		LPS_StartLedgeDuration(client);
	}
	else
	{
		LPS_EndLedgeDuration(client);
		LPS_StartIncapacitatedDuration(client);
	}
}

public void LPS_Event_PlayerLedgeGrabDuration(Event event, const char[] name, bool dontBroadcast)
{
	int client = GetClientOfUserId(event.GetInt("userid"));
	if (!LPS_CanTrackPvEInteractionDuration(client))
	{
		return;
	}

	LPS_EndIncapacitatedDuration(client);
	LPS_StartLedgeDuration(client);
}

public void LPS_Event_PlayerLedgeReleaseDuration(Event event, const char[] name, bool dontBroadcast)
{
	int client = GetClientOfUserId(event.GetInt("userid"));
	LPS_EndLedgeDuration(client);
}

public void LPS_Event_ReviveSuccessDuration(Event event, const char[] name, bool dontBroadcast)
{
	int subject = GetClientOfUserId(event.GetInt("subject"));
	LPS_EndIncapacitatedDuration(subject);
	LPS_EndLedgeDuration(subject);
}

public void LPS_Event_PlayerDeathDuration(Event event, const char[] name, bool dontBroadcast)
{
	int client = GetClientOfUserId(event.GetInt("userid"));
	LPS_EndIncapacitatedDuration(client);
	LPS_EndLedgeDuration(client);
	if (client >= 1 && client <= MaxClients)
	{
		g_LPSPendingBlackWhiteSubjectUserId[client] = 0;
	}
}

public void LPS_Event_HealBeginBlackWhite(Event event, const char[] name, bool dontBroadcast)
{
	int healer = GetClientOfUserId(event.GetInt("userid"));
	int subjectUserId = event.GetInt("subject");
	int subject = GetClientOfUserId(subjectUserId);
	if (!(LPS_CanRecordPvEEvents() || LPS_CanRecordVersusEvents())
		|| !LPS_IsHumanSurvivorClient(healer)
		|| healer == subject
		|| subject < 1
		|| subject > MaxClients
		|| !IsClientInGame(subject)
		|| GetClientTeam(subject) != LPS_TEAM_SURVIVOR
		|| !LPS_IsBlackWhiteSurvivor(subject))
	{
		if (healer >= 1 && healer <= MaxClients)
		{
			g_LPSPendingBlackWhiteSubjectUserId[healer] = 0;
		}
		return;
	}

	g_LPSPendingBlackWhiteSubjectUserId[healer] = subjectUserId;
}

public void LPS_Event_HealInterruptedBlackWhite(Event event, const char[] name, bool dontBroadcast)
{
	int healer = GetClientOfUserId(event.GetInt("userid"));
	if (healer >= 1 && healer <= MaxClients)
	{
		g_LPSPendingBlackWhiteSubjectUserId[healer] = 0;
	}
}

public void LPS_Event_HealSuccessBlackWhite(Event event, const char[] name, bool dontBroadcast)
{
	int healer = GetClientOfUserId(event.GetInt("userid"));
	int subjectUserId = event.GetInt("subject");
	if (healer < 1
		|| healer > MaxClients
		|| g_LPSPendingBlackWhiteSubjectUserId[healer] != subjectUserId)
	{
		return;
	}

	g_LPSPendingBlackWhiteSubjectUserId[healer] = 0;
	DataPack pack;
	CreateDataTimer(0.0, LPS_Timer_VerifyBlackWhiteRestore, pack, TIMER_FLAG_NO_MAPCHANGE);
	pack.WriteCell(GetClientUserId(healer));
	pack.WriteCell(subjectUserId);
	pack.WriteString(g_LPSClientSegmentId[healer]);
}

public Action LPS_Timer_VerifyBlackWhiteRestore(Handle timer, DataPack pack)
{
	pack.Reset();
	int healer = GetClientOfUserId(pack.ReadCell());
	int subject = GetClientOfUserId(pack.ReadCell());
	char segmentId[LPS_MAX_SEGMENT_ID];
	pack.ReadString(segmentId, sizeof(segmentId));

	if (!LPS_IsHumanSurvivorClient(healer)
		|| !g_LPSClientSegmentActive[healer]
		|| !StrEqual(g_LPSClientSegmentId[healer], segmentId, true)
		|| subject < 1
		|| subject > MaxClients
		|| !IsClientInGame(subject)
		|| GetClientTeam(subject) != LPS_TEAM_SURVIVOR
		|| LPS_IsBlackWhiteSurvivor(subject))
	{
		return Plugin_Stop;
	}

	if (g_LPSModeFamily == LPSModeFamily_PvE)
	{
		LPS_AddPvEStat(healer, LPSPvEStat_BlackWhiteTeammatesRestored, 1);
	}
	else if (g_LPSModeFamily == LPSModeFamily_Versus)
	{
		LPS_AddVersusSurvivorStat(
			healer,
			LPSVersusSurvivorStat_BlackWhiteTeammatesRestored,
			1
		);
	}
	LPS_RecordRelationshipBlackWhiteRestore(healer, subject);
	return Plugin_Stop;
}

bool LPS_IsBlackWhiteSurvivor(int client)
{
	return client >= 1
		&& client <= MaxClients
		&& IsClientInGame(client)
		&& HasEntProp(client, Prop_Send, "m_bIsOnThirdStrike")
		&& GetEntProp(client, Prop_Send, "m_bIsOnThirdStrike") != 0;
}

bool LPS_IsSurvivorHangingFromLedge(int client)
{
	return client >= 1
		&& client <= MaxClients
		&& IsClientInGame(client)
		&& GetClientTeam(client) == LPS_TEAM_SURVIVOR
		&& HasEntProp(client, Prop_Send, "m_isHangingFromLedge")
		&& GetEntProp(client, Prop_Send, "m_isHangingFromLedge") != 0;
}

bool LPS_CanTrackPvEInteractionDuration(int client)
{
	return LPS_CanRecordPvEEvents()
		&& LPS_IsHumanSurvivorClient(client)
		&& g_LPSClientPvEStatsActive[client]
		&& g_LPSClientSegmentActive[client];
}

void LPS_OnClientPvEInteractionSegmentStarted(int client)
{
	LPS_ResetClientPvEInteractionState(client);
	if (!LPS_CanTrackPvEInteractionDuration(client) || !IsPlayerAlive(client))
	{
		return;
	}

	if (LPS_IsSurvivorHangingFromLedge(client))
	{
		LPS_StartLedgeDuration(client);
	}
	else if (LPS_IsSurvivorIncapacitated(client))
	{
		LPS_StartIncapacitatedDuration(client);
	}
}

void LPS_OnClientPvEInteractionSegmentEnding(int client)
{
	LPS_EndIncapacitatedDuration(client);
	LPS_EndLedgeDuration(client);
	g_LPSPendingBlackWhiteSubjectUserId[client] = 0;
}

void LPS_StartIncapacitatedDuration(int client)
{
	if (!LPS_CanTrackPvEInteractionDuration(client) || g_LPSIncapacitatedTimerActive[client])
	{
		return;
	}

	g_LPSIncapacitatedTimerActive[client] = true;
	g_LPSIncapacitatedStartedAt[client] = GetGameTime();
}

void LPS_StartLedgeDuration(int client)
{
	if (!LPS_CanTrackPvEInteractionDuration(client) || g_LPSLedgeTimerActive[client])
	{
		return;
	}

	g_LPSLedgeTimerActive[client] = true;
	g_LPSLedgeStartedAt[client] = GetGameTime();
}

void LPS_EndIncapacitatedDuration(int client)
{
	if (client < 1 || client > MaxClients || !g_LPSIncapacitatedTimerActive[client])
	{
		return;
	}

	LPS_AccumulateIncapacitatedDuration(client);
	g_LPSIncapacitatedTimerActive[client] = false;
	g_LPSIncapacitatedStartedAt[client] = 0.0;
}

void LPS_EndLedgeDuration(int client)
{
	if (client < 1 || client > MaxClients || !g_LPSLedgeTimerActive[client])
	{
		return;
	}

	LPS_AccumulateLedgeDuration(client);
	g_LPSLedgeTimerActive[client] = false;
	g_LPSLedgeStartedAt[client] = 0.0;
}

void LPS_AccumulateAllPvEInteractionDurations()
{
	for (int client = 1; client <= MaxClients; client++)
	{
		LPS_AccumulateIncapacitatedDuration(client);
		LPS_AccumulateLedgeDuration(client);
	}
}

void LPS_AccumulateIncapacitatedDuration(int client)
{
	if (client < 1 || client > MaxClients || !g_LPSIncapacitatedTimerActive[client])
	{
		return;
	}

	int seconds = RoundToFloor(GetGameTime() - g_LPSIncapacitatedStartedAt[client]);
	if (seconds > 0)
	{
		LPS_AddPvEStat(client, LPSPvEStat_IncapacitatedSeconds, seconds);
		g_LPSIncapacitatedStartedAt[client] += float(seconds);
	}
}

void LPS_AccumulateLedgeDuration(int client)
{
	if (client < 1 || client > MaxClients || !g_LPSLedgeTimerActive[client])
	{
		return;
	}

	int seconds = RoundToFloor(GetGameTime() - g_LPSLedgeStartedAt[client]);
	if (seconds > 0)
	{
		LPS_AddPvEStat(client, LPSPvEStat_LedgeHangingSeconds, seconds);
		g_LPSLedgeStartedAt[client] += float(seconds);
	}
}

void LPS_ResetClientPvEInteractionState(int client)
{
	if (client < 1 || client > MaxClients)
	{
		return;
	}

	g_LPSIncapacitatedTimerActive[client] = false;
	g_LPSLedgeTimerActive[client] = false;
	g_LPSIncapacitatedStartedAt[client] = 0.0;
	g_LPSLedgeStartedAt[client] = 0.0;
	g_LPSPendingBlackWhiteSubjectUserId[client] = 0;
}
