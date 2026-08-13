#if defined _l4d2_player_stats_round_context_included
	#endinput
#endif
#define _l4d2_player_stats_round_context_included

enum struct LPSRoundContextSnapshot
{
	char roundId[LPS_MAX_ROUND_ID];
	int capturedAt;
	char collectorVersion[33];
	char rulesetName[LPS_MAX_RULESET_NAME];
	char difficulty[LPS_MAX_DIFFICULTY];
	int survivorLimit;
	int maxPlayerZombies;
	int commonLimit;
	int tankHealth;
	int witchHealth;
	int changeMask;
	bool incidentCaptureEnabled;
	bool incidentCaptureComplete;
	int incidentExpectedCount;
	int incidentDroppedCount;
	int revision;
}

ArrayList g_LPSPendingRoundContexts;
LPSRoundContextSnapshot g_LPSActiveRoundContext;
bool g_LPSRoundContextActive;
bool g_LPSRoundContextDirty;
bool g_LPSRoundHadHumanSegment;
float g_LPSRoundStartedGameTime;

ConVar g_LPSContextDifficulty;
ConVar g_LPSContextSurvivorLimit;
ConVar g_LPSContextMaxPlayerZombies;
ConVar g_LPSContextCommonLimit;
ConVar g_LPSContextTankHealth;
ConVar g_LPSContextWitchHealth;

void LPS_InitializeRoundContext()
{
	g_LPSPendingRoundContexts = new ArrayList(sizeof(LPSRoundContextSnapshot));
	g_LPSContextDifficulty = FindConVar("z_difficulty");
	g_LPSContextSurvivorLimit = FindConVar("survivor_limit");
	g_LPSContextMaxPlayerZombies = FindConVar("z_max_player_zombies");
	g_LPSContextCommonLimit = FindConVar("z_common_limit");
	g_LPSContextTankHealth = FindConVar("z_tank_health");
	g_LPSContextWitchHealth = FindConVar("z_witch_health");

	g_LPSRulesetName.AddChangeHook(LPS_OnRulesetContextChanged);
	LPS_HookOptionalContextCvar(g_LPSContextDifficulty);
	LPS_HookOptionalContextCvar(g_LPSContextSurvivorLimit);
	LPS_HookOptionalContextCvar(g_LPSContextMaxPlayerZombies);
	LPS_HookOptionalContextCvar(g_LPSContextCommonLimit);
	LPS_HookOptionalContextCvar(g_LPSContextTankHealth);
	LPS_HookOptionalContextCvar(g_LPSContextWitchHealth);
	LPS_ResetActiveRoundContext();
}

void LPS_HookOptionalContextCvar(ConVar cvar)
{
	if (cvar != null)
	{
		cvar.AddChangeHook(LPS_OnEngineContextChanged);
	}
}

public void LPS_OnRulesetContextChanged(ConVar convar, const char[] oldValue, const char[] newValue)
{
	LPS_MarkRoundContextChanged(0);
}

public void LPS_OnEngineContextChanged(ConVar convar, const char[] oldValue, const char[] newValue)
{
	if (convar == g_LPSContextDifficulty) LPS_MarkRoundContextChanged(1);
	else if (convar == g_LPSContextSurvivorLimit) LPS_MarkRoundContextChanged(2);
	else if (convar == g_LPSContextMaxPlayerZombies) LPS_MarkRoundContextChanged(3);
	else if (convar == g_LPSContextCommonLimit) LPS_MarkRoundContextChanged(4);
	else if (convar == g_LPSContextTankHealth) LPS_MarkRoundContextChanged(5);
	else if (convar == g_LPSContextWitchHealth) LPS_MarkRoundContextChanged(6);
}

void LPS_MarkRoundContextChanged(int bit)
{
	if (!g_LPSRoundContextActive)
	{
		return;
	}
	int mask = 1 << bit;
	if ((g_LPSActiveRoundContext.changeMask & mask) == 0)
	{
		g_LPSActiveRoundContext.changeMask |= mask;
		g_LPSActiveRoundContext.revision++;
		g_LPSRoundContextDirty = true;
	}
}

void LPS_StartRoundContext()
{
	LPS_ResetActiveRoundContext();
	g_LPSRoundStartedGameTime = GetGameTime();
	g_LPSRoundHadHumanSegment = false;
	strcopy(g_LPSActiveRoundContext.roundId, sizeof(g_LPSActiveRoundContext.roundId), g_LPSRoundId);
	g_LPSActiveRoundContext.capturedAt = GetTime();
	strcopy(g_LPSActiveRoundContext.collectorVersion, sizeof(g_LPSActiveRoundContext.collectorVersion), LPS_VERSION);
	g_LPSRulesetName.GetString(g_LPSActiveRoundContext.rulesetName, sizeof(g_LPSActiveRoundContext.rulesetName));
	TrimString(g_LPSActiveRoundContext.rulesetName);
	LPS_GetOptionalContextString(g_LPSContextDifficulty, g_LPSActiveRoundContext.difficulty, sizeof(g_LPSActiveRoundContext.difficulty));
	g_LPSActiveRoundContext.survivorLimit = LPS_GetOptionalContextInt(g_LPSContextSurvivorLimit);
	g_LPSActiveRoundContext.maxPlayerZombies = LPS_GetOptionalContextInt(g_LPSContextMaxPlayerZombies);
	g_LPSActiveRoundContext.commonLimit = LPS_GetOptionalContextInt(g_LPSContextCommonLimit);
	g_LPSActiveRoundContext.tankHealth = LPS_GetOptionalContextInt(g_LPSContextTankHealth);
	g_LPSActiveRoundContext.witchHealth = LPS_GetOptionalContextInt(g_LPSContextWitchHealth);
	g_LPSActiveRoundContext.incidentCaptureEnabled = g_LPSIncidentsEnabled.BoolValue;
	g_LPSActiveRoundContext.revision = 1;
	g_LPSRoundContextActive = true;
	g_LPSRoundContextDirty = true;
}

void LPS_FinalizeRoundContext()
{
	if (!g_LPSRoundContextActive)
	{
		return;
	}
	g_LPSActiveRoundContext.incidentCaptureComplete = false;
	g_LPSActiveRoundContext.revision++;
	if (g_LPSPendingRoundContexts.Length < g_LPSClosedLifecycleQueueLimit.IntValue)
	{
		g_LPSPendingRoundContexts.PushArray(g_LPSActiveRoundContext, sizeof(g_LPSActiveRoundContext));
	}
	else
	{
		LogError("Pending Round Context queue reached %d rows; this Round remains analysis-incomplete while core stats continue.", g_LPSClosedLifecycleQueueLimit.IntValue);
		LPS_DiscardPendingIncidentsForRound(g_LPSActiveRoundContext.roundId);
	}
	LPS_ResetActiveRoundContext();
}

void LPS_OnHumanSegmentStartedForAnalysis()
{
	if (g_LPSRoundContextActive)
	{
		g_LPSRoundHadHumanSegment = true;
	}
}

void LPS_GetOptionalContextString(ConVar cvar, char[] value, int maxLength)
{
	value[0] = '\0';
	if (cvar != null)
	{
		cvar.GetString(value, maxLength);
		TrimString(value);
	}
}

int LPS_GetOptionalContextInt(ConVar cvar)
{
	return cvar == null ? -1 : cvar.IntValue;
}

void LPS_ResetActiveRoundContext()
{
	g_LPSRoundContextActive = false;
	g_LPSRoundContextDirty = false;
	g_LPSRoundHadHumanSegment = false;
	g_LPSRoundStartedGameTime = 0.0;
	g_LPSActiveRoundContext.roundId[0] = '\0';
	g_LPSActiveRoundContext.capturedAt = 0;
	g_LPSActiveRoundContext.collectorVersion[0] = '\0';
	g_LPSActiveRoundContext.rulesetName[0] = '\0';
	g_LPSActiveRoundContext.difficulty[0] = '\0';
	g_LPSActiveRoundContext.survivorLimit = -1;
	g_LPSActiveRoundContext.maxPlayerZombies = -1;
	g_LPSActiveRoundContext.commonLimit = -1;
	g_LPSActiveRoundContext.tankHealth = -1;
	g_LPSActiveRoundContext.witchHealth = -1;
	g_LPSActiveRoundContext.changeMask = 0;
	g_LPSActiveRoundContext.incidentCaptureEnabled = false;
	g_LPSActiveRoundContext.incidentCaptureComplete = false;
	g_LPSActiveRoundContext.incidentExpectedCount = 0;
	g_LPSActiveRoundContext.incidentDroppedCount = 0;
	g_LPSActiveRoundContext.revision = 0;
}

int LPS_GetPendingRoundContextCount()
{
	return g_LPSPendingRoundContexts == null ? 0 : g_LPSPendingRoundContexts.Length;
}
