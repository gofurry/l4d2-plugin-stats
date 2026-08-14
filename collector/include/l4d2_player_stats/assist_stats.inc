#if defined _l4d2_player_stats_assist_stats_included
	#endinput
#endif
#define _l4d2_player_stats_assist_stats_included

// Per-life contributor membership only. Damage amounts remain in the existing
// aggregate counters; this tracker deliberately creates no per-hit rows.
int g_LPSAssistContributorsLow[LPS_MAX_ENTITY_INDEX + 1];
int g_LPSAssistContributorsHigh[LPS_MAX_ENTITY_INDEX + 1];

void LPS_InitializeAssistStats()
{
	LPS_ResetAllAssistContributors();
}

void LPS_ResetAllAssistContributors()
{
	for (int entity = 0; entity <= LPS_MAX_ENTITY_INDEX; entity++)
	{
		g_LPSAssistContributorsLow[entity] = 0;
		g_LPSAssistContributorsHigh[entity] = 0;
	}
}

void LPS_ResetAssistTarget(int target)
{
	if (target < 1 || target > LPS_MAX_ENTITY_INDEX)
	{
		return;
	}

	g_LPSAssistContributorsLow[target] = 0;
	g_LPSAssistContributorsHigh[target] = 0;
}

void LPS_ClearAssistContributor(int client)
{
	if (client < 1 || client > MaxClients)
	{
		return;
	}

	int mask = 1 << ((client - 1) % 32);
	for (int target = 1; target <= LPS_MAX_ENTITY_INDEX; target++)
	{
		if (client <= 32)
		{
			g_LPSAssistContributorsLow[target] &= ~mask;
		}
		else
		{
			g_LPSAssistContributorsHigh[target] &= ~mask;
		}
	}
}

void LPS_RecordAssistContribution(int target, int contributor)
{
	if (target < 1
		|| target > MaxClients
		|| contributor < 1
		|| contributor > MaxClients
		|| target == contributor
		|| !IsClientInGame(target)
		|| !IsClientInGame(contributor)
		|| IsFakeClient(contributor)
		|| GetClientTeam(target) != LPS_TEAM_INFECTED
		|| GetClientTeam(contributor) != LPS_TEAM_SURVIVOR
		|| !g_LPSClientSegmentActive[contributor])
	{
		return;
	}

	int mask = 1 << ((contributor - 1) % 32);
	if (contributor <= 32)
	{
		g_LPSAssistContributorsLow[target] |= mask;
	}
	else
	{
		g_LPSAssistContributorsHigh[target] |= mask;
	}
}

bool LPS_IsAssistContributor(int target, int client)
{
	if (target < 1 || target > LPS_MAX_ENTITY_INDEX || client < 1 || client > MaxClients)
	{
		return false;
	}

	int mask = 1 << ((client - 1) % 32);
	return client <= 32
		? (g_LPSAssistContributorsLow[target] & mask) != 0
		: (g_LPSAssistContributorsHigh[target] & mask) != 0;
}

void LPS_ResolvePvESpecialAssists(int victim, int killer, int zombieClass, bool aborted)
{
	if (victim < 1 || victim > MaxClients)
	{
		return;
	}

	if (!aborted && zombieClass >= 1 && zombieClass <= 6)
	{
		for (int client = 1; client <= MaxClients; client++)
		{
			if (client == killer || !LPS_IsAssistContributor(victim, client))
			{
				continue;
			}

			LPS_AddPvEStat(client, LPSPvEStat_SpecialAssists, 1);
			LPS_AddPvESpecialClassAssist(client, zombieClass);
		}
	}

	LPS_ResetAssistTarget(victim);
}

void LPS_AddPvESpecialClassAssist(int client, int zombieClass)
{
	switch (zombieClass)
	{
		case 1: LPS_AddPvEStat(client, LPSPvEStat_SmokerAssists, 1);
		case 2: LPS_AddPvEStat(client, LPSPvEStat_BoomerAssists, 1);
		case 3: LPS_AddPvEStat(client, LPSPvEStat_HunterAssists, 1);
		case 4: LPS_AddPvEStat(client, LPSPvEStat_SpitterAssists, 1);
		case 5: LPS_AddPvEStat(client, LPSPvEStat_JockeyAssists, 1);
		case 6: LPS_AddPvEStat(client, LPSPvEStat_ChargerAssists, 1);
	}
}

void LPS_ResolveVersusInfectedAssists(
	int victim,
	int killer,
	int zombieClass,
	bool humanController,
	bool aborted
)
{
	if (victim < 1 || victim > MaxClients)
	{
		return;
	}

	if (!aborted && ((zombieClass >= 1 && zombieClass <= 6) || zombieClass == 8))
	{
		for (int client = 1; client <= MaxClients; client++)
		{
			if (client == killer || !LPS_IsAssistContributor(victim, client))
			{
				continue;
			}

			LPS_AddVersusSurvivorClassStat(
				client,
				zombieClass,
				humanController
					? LPSVersusSurvivorClassStat_HumanControllerAssists
					: LPSVersusSurvivorClassStat_BotControllerAssists,
				1
			);
		}
	}

	LPS_ResetAssistTarget(victim);
}
