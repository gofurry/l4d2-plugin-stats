#if defined _l4d2_player_stats_versus_survivor_detail_included
	#endinput
#endif
#define _l4d2_player_stats_versus_survivor_detail_included

void LPS_OnClientVersusSurvivorDetailSegmentStarted(int client)
{
	if (client < 1 || client > MaxClients)
	{
		return;
	}
	g_LPSTongueStopHandled[client] = false;
	g_LPSPendingUpgradePackType[client] = 0;
}

void LPS_OnClientVersusSurvivorDetailSegmentEnding(int client)
{
	LPS_OnClientVersusSurvivorDetailSegmentStarted(client);
}

void LPS_RecordVersusWitchContribution(int witch, int client)
{
	if (!LPS_CanRecordVersusEvents()
		|| !LPS_IsWitchEntity(witch)
		|| !LPS_IsHumanClientOnTeam(client, LPS_TEAM_SURVIVOR))
	{
		return;
	}

	// The shared bounded contributor slots are safe because exactly one mode
	// family can be active at a time. Versus does not award PvE encounters.
	LPS_RecordBossContribution(witch, client);
}

void LPS_RecordVersusWitchKilled(Event event)
{
	if (!LPS_CanRecordVersusEvents())
	{
		return;
	}

	int killer = GetClientOfUserId(event.GetInt("userid"));
	int witch = event.GetInt("witchid");
	if (!LPS_IsHumanClientOnTeam(killer, LPS_TEAM_SURVIVOR))
	{
		LPS_ResetBossTracking(witch);
		return;
	}

	LPS_AddVersusSurvivorStat(killer, LPSVersusSurvivorStat_WitchKills, 1);
	if (event.GetBool("oneshot"))
	{
		LPS_AddVersusSurvivorStat(killer, LPSVersusSurvivorStat_WitchOneShots, 1);
	}

	int contributorCount = 0;
	int soleContributorUserId = 0;
	if (witch >= 1 && witch <= LPS_MAX_ENTITY_INDEX)
	{
		for (int slot = 1; slot <= MaxClients; slot++)
		{
			int userId = g_LPSBossContributorUserId[witch][slot];
			if (userId == 0)
			{
				continue;
			}
			contributorCount++;
			soleContributorUserId = userId;
		}
	}

	if (contributorCount == 1
		&& soleContributorUserId == GetClientUserId(killer))
	{
		LPS_AddVersusSurvivorStat(killer, LPSVersusSurvivorStat_WitchSoloKills, 1);
	}
	LPS_ResetBossTracking(witch);
}

void LPS_RecordVersusThrowableAction(int client, LPSEquipment equipment)
{
	if (!LPS_CanRecordVersusEvents()
		|| !LPS_IsHumanClientOnTeam(client, LPS_TEAM_SURVIVOR))
	{
		return;
	}

	switch (equipment)
	{
		case LPSEquipment_Molotov:
		{
			LPS_AddVersusSurvivorStat(
				client,
				LPSVersusSurvivorStat_MolotovsThrown,
				1
			);
		}
		case LPSEquipment_PipeBomb:
		{
			LPS_AddVersusSurvivorStat(
				client,
				LPSVersusSurvivorStat_PipeBombsThrown,
				1
			);
		}
		case LPSEquipment_VomitJar:
		{
			LPS_AddVersusSurvivorStat(
				client,
				LPSVersusSurvivorStat_VomitJarsThrown,
				1
			);
		}
	}
}
