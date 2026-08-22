#if defined _l4d2_player_stats_modes_included
	#endinput
#endif
#define _l4d2_player_stats_modes_included

ConVar g_LPSGameModeConVar;
LPSModeFamily g_LPSModeFamily;
bool g_LPSModeInitialized;
char g_LPSGameMode[LPS_MAX_GAME_MODE];

void LPS_InitializeModes()
{
	g_LPSGameModeConVar = FindConVar("mp_gamemode");
	if (g_LPSGameModeConVar == null)
	{
		SetFailState("Required ConVar mp_gamemode was not found.");
		return;
	}

	g_LPSGameModeConVar.AddChangeHook(LPS_OnGameModeChanged);
	LPS_RefreshGameMode(false);
}

public void OnMapStart()
{
	LPS_OnSessionsMapStart();
	LPS_RefreshGameMode(true);
	LPS_OnLifecycleMapStart();
}

public void OnMapEnd()
{
	LPS_OnSessionsMapEnd();
	LPS_OnLifecycleMapEnd();
}

public void LPS_OnGameModeChanged(ConVar convar, const char[] oldValue, const char[] newValue)
{
	LPS_RefreshGameMode(true);
}

void LPS_RefreshGameMode(bool notifySessions)
{
	char nextMode[LPS_MAX_GAME_MODE];
	g_LPSGameModeConVar.GetString(nextMode, sizeof(nextMode));
	TrimString(nextMode);

	LPSModeFamily nextFamily = LPS_GetModeFamily(nextMode);
	bool changed = !g_LPSModeInitialized
		|| nextFamily != g_LPSModeFamily
		|| !StrEqual(nextMode, g_LPSGameMode, true);

	if (!changed)
	{
		return;
	}
	LPS_ResetAllSurvivorTechniqueState();

	bool previouslySupported = g_LPSModeFamily != LPSModeFamily_Unsupported;
	bool nowSupported = nextFamily != LPSModeFamily_Unsupported;
	LPSModeFamily previousFamily = g_LPSModeFamily;
	char previousMode[LPS_MAX_GAME_MODE];
	strcopy(previousMode, sizeof(previousMode), g_LPSGameMode);

	if (notifySessions && g_LPSModeInitialized)
	{
		LPS_OnLifecycleModeChanging(previousFamily, previousMode, nextFamily, nextMode);
	}

	g_LPSModeFamily = nextFamily;
	strcopy(g_LPSGameMode, sizeof(g_LPSGameMode), nextMode);
	g_LPSModeInitialized = true;

	if (!notifySessions)
	{
		return;
	}

	if (previouslySupported && !nowSupported)
	{
		LPS_EndAllActiveSessions("unsupported_mode");
	}
	else if (!previouslySupported && nowSupported)
	{
		LPS_StartEligibleSessions();
	}

	LPS_OnLifecycleModeChanged(previousFamily, previousMode, nextFamily, nextMode);
}

LPSModeFamily LPS_GetModeFamily(const char[] gameMode)
{
	if (StrEqual(gameMode, "coop", true) || StrEqual(gameMode, "realism", true))
	{
		return LPSModeFamily_PvE;
	}

	if (StrEqual(gameMode, "versus", true))
	{
		return LPSModeFamily_Versus;
	}

	return LPSModeFamily_Unsupported;
}

bool LPS_IsSupportedMode()
{
	return g_LPSModeFamily != LPSModeFamily_Unsupported;
}

void LPS_GetModeFamilyName(char[] buffer, int maxLength)
{
	switch (g_LPSModeFamily)
	{
		case LPSModeFamily_PvE: strcopy(buffer, maxLength, "pve");
		case LPSModeFamily_Versus: strcopy(buffer, maxLength, "versus");
		default: strcopy(buffer, maxLength, "unsupported");
	}
}
