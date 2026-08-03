#if defined _l4d2_player_stats_versus_results_included
	#endinput
#endif
#define _l4d2_player_stats_versus_results_included

enum struct LPSVersusRoundResultSnapshot
{
	char roundId[LPS_MAX_ROUND_ID];
	int scoringTeamSlot;
	int teamsFlipped;
	int team0MapScore;
	int team1MapScore;
	int team0CampaignScore;
	int team1CampaignScore;
	int rawWinnerTeam;
	int scoreAvailable;
	char resultStatus[LPS_MAX_LIFECYCLE_STATUS];
	int finalizedAt;
	int revision;
}

enum struct LPSVersusRunResultSnapshot
{
	char runId[LPS_MAX_RUN_ID];
	int team0CampaignScore;
	int team1CampaignScore;
	int winnerTeamSlot;
	int rawWinnerTeam;
	int scoreAvailable;
	char resultStatus[LPS_MAX_LIFECYCLE_STATUS];
	int finalizedAt;
	int revision;
}

ArrayList g_LPSClosedVersusRoundResults;
ArrayList g_LPSClosedVersusRunResults;
int g_LPSFlushClosedVersusRoundResultCount;
int g_LPSFlushClosedVersusRunResultCount;

bool g_LPSVersusRoundResultActive;
char g_LPSVersusResultRoundId[LPS_MAX_ROUND_ID];
int g_LPSVersusTeamsFlipped;
int g_LPSVersusRoundStartMapScore[2];
int g_LPSVersusRawRoundWinnerTeam;
int g_LPSVersusRawMatchWinnerTeam;
int g_LPSVersusRunResultRevision;
int g_LPSVersusLastCampaignScore[2];

bool g_LPSHasLastVersusRoundResult;
LPSVersusRoundResultSnapshot g_LPSLastVersusRoundResult;

void LPS_InitializeVersusResults()
{
	g_LPSClosedVersusRoundResults = new ArrayList(sizeof(LPSVersusRoundResultSnapshot));
	g_LPSClosedVersusRunResults = new ArrayList(sizeof(LPSVersusRunResultSnapshot));
	g_LPSFlushClosedVersusRoundResultCount = 0;
	g_LPSFlushClosedVersusRunResultCount = 0;
	g_LPSVersusRawMatchWinnerTeam = -1;
	g_LPSVersusRunResultRevision = 0;
	g_LPSVersusLastCampaignScore[0] = -1;
	g_LPSVersusLastCampaignScore[1] = -1;
	g_LPSHasLastVersusRoundResult = false;
	LPS_ResetVersusRoundResultState();
}

void LPS_OnVersusRunStarted()
{
	g_LPSVersusRawMatchWinnerTeam = -1;
	g_LPSVersusRunResultRevision = 0;
	g_LPSVersusLastCampaignScore[0] = -1;
	g_LPSVersusLastCampaignScore[1] = -1;
	g_LPSHasLastVersusRoundResult = false;
}

void LPS_OnVersusRoundStarted()
{
	if (g_LPSModeFamily != LPSModeFamily_Versus || !g_LPSRoundActive)
	{
		return;
	}

	strcopy(g_LPSVersusResultRoundId, sizeof(g_LPSVersusResultRoundId), g_LPSRoundId);
	g_LPSVersusTeamsFlipped = GameRules_GetProp("m_bAreTeamsFlipped", 1) != 0 ? 1 : 0;
	int unusedCampaignScore0;
	int unusedCampaignScore1;
	LPS_ReadVersusScores(
		g_LPSVersusRoundStartMapScore[0],
		g_LPSVersusRoundStartMapScore[1],
		unusedCampaignScore0,
		unusedCampaignScore1
	);
	// m_bAreTeamsFlipped is retained for diagnostics, but listen servers can
	// leave it false in both halves. The final slot is resolved from the actual
	// map-score change at round end, with the half number as a fallback.
	g_LPSVersusRawRoundWinnerTeam = -1;
	g_LPSVersusRoundResultActive = true;
}

void LPS_SetVersusRoundWinner(int rawWinnerTeam)
{
	if (g_LPSVersusRoundResultActive)
	{
		g_LPSVersusRawRoundWinnerTeam = rawWinnerTeam;
	}
}

void LPS_SetVersusMatchWinner(int rawWinnerTeam)
{
	g_LPSVersusRawMatchWinnerTeam = rawWinnerTeam;
}

void LPS_OnVersusRoundEnding(const char[] status)
{
	if (!g_LPSVersusRoundResultActive
		|| !StrEqual(g_LPSVersusResultRoundId, g_LPSRoundId, true))
	{
		return;
	}

	LPSVersusRoundResultSnapshot snapshot;
	strcopy(snapshot.roundId, sizeof(snapshot.roundId), g_LPSVersusResultRoundId);
	snapshot.teamsFlipped = g_LPSVersusTeamsFlipped;
	LPS_ReadVersusScores(
		snapshot.team0MapScore,
		snapshot.team1MapScore,
		snapshot.team0CampaignScore,
		snapshot.team1CampaignScore
	);
	snapshot.scoringTeamSlot = LPS_DetermineVersusScoringTeamSlot(
		snapshot.team0MapScore,
		snapshot.team1MapScore
	);
	snapshot.rawWinnerTeam = g_LPSVersusRawRoundWinnerTeam;
	snapshot.scoreAvailable = LPS_IsVersusRoundScoreAvailable(snapshot) ? 1 : 0;
	LPS_RememberVersusCampaignScores(
		snapshot.team0CampaignScore,
		snapshot.team1CampaignScore
	);
	strcopy(snapshot.resultStatus, sizeof(snapshot.resultStatus), status);
	snapshot.finalizedAt = GetTime();
	snapshot.revision = 1;

	LPS_QueueVersusRoundResult(snapshot);
	g_LPSLastVersusRoundResult = snapshot;
	g_LPSHasLastVersusRoundResult = true;
	if (StrEqual(status, "completed", true))
	{
		LPS_QueueCurrentVersusRunResult("active", false);
	}
	LPS_ResetVersusRoundResultState();
}

void LPS_ReclassifyLastVersusResultAsAbandoned(const char[] roundId)
{
	if (!g_LPSHasLastVersusRoundResult
		|| !StrEqual(g_LPSLastVersusRoundResult.roundId, roundId, true)
		|| !StrEqual(g_LPSLastVersusRoundResult.resultStatus, "completed", true))
	{
		return;
	}

	strcopy(
		g_LPSLastVersusRoundResult.resultStatus,
		sizeof(g_LPSLastVersusRoundResult.resultStatus),
		"abandoned"
	);
	g_LPSLastVersusRoundResult.revision++;
	LPS_QueueVersusRoundResult(g_LPSLastVersusRoundResult);
}

void LPS_OnVersusRunEnding(const char[] status)
{
	if (!StrEqual(g_LPSRunModeFamily, "versus", true))
	{
		return;
	}

	LPS_QueueCurrentVersusRunResult(status, true);
}

void LPS_QueueCurrentVersusRunResult(const char[] status, bool finalized)
{
	if (!g_LPSRunActive || !StrEqual(g_LPSRunModeFamily, "versus", true))
	{
		return;
	}

	LPSVersusRunResultSnapshot snapshot;
	strcopy(snapshot.runId, sizeof(snapshot.runId), g_LPSRunId);
	int unusedMapScore0;
	int unusedMapScore1;
	LPS_ReadVersusScores(
		unusedMapScore0,
		unusedMapScore1,
		snapshot.team0CampaignScore,
		snapshot.team1CampaignScore
	);
	if (finalized && StrEqual(status, "abandoned", true))
	{
		LPS_KeepHighestKnownVersusCampaignScores(
			snapshot.team0CampaignScore,
			snapshot.team1CampaignScore
		);
	}
	LPS_RememberVersusCampaignScores(
		snapshot.team0CampaignScore,
		snapshot.team1CampaignScore
	);
	snapshot.scoreAvailable = snapshot.team0CampaignScore >= 0
		&& snapshot.team1CampaignScore >= 0 ? 1 : 0;
	snapshot.winnerTeamSlot = -1;
	if (finalized && StrEqual(status, "completed", true) && snapshot.scoreAvailable != 0)
	{
		if (snapshot.team0CampaignScore > snapshot.team1CampaignScore)
		{
			snapshot.winnerTeamSlot = 0;
		}
		else if (snapshot.team1CampaignScore > snapshot.team0CampaignScore)
		{
			snapshot.winnerTeamSlot = 1;
		}
		else
		{
			snapshot.winnerTeamSlot = 2;
		}
	}
	snapshot.rawWinnerTeam = g_LPSVersusRawMatchWinnerTeam;
	strcopy(snapshot.resultStatus, sizeof(snapshot.resultStatus), status);
	snapshot.finalizedAt = finalized ? GetTime() : 0;
	snapshot.revision = ++g_LPSVersusRunResultRevision;
	LPS_QueueVersusRunResult(snapshot);
}

void LPS_RememberVersusCampaignScores(int team0CampaignScore, int team1CampaignScore)
{
	if (team0CampaignScore >= 0
		&& (g_LPSVersusLastCampaignScore[0] < 0
			|| team0CampaignScore > g_LPSVersusLastCampaignScore[0]))
	{
		g_LPSVersusLastCampaignScore[0] = team0CampaignScore;
	}
	if (team1CampaignScore >= 0
		&& (g_LPSVersusLastCampaignScore[1] < 0
			|| team1CampaignScore > g_LPSVersusLastCampaignScore[1]))
	{
		g_LPSVersusLastCampaignScore[1] = team1CampaignScore;
	}
}

void LPS_KeepHighestKnownVersusCampaignScores(int &team0CampaignScore, int &team1CampaignScore)
{
	if (g_LPSVersusLastCampaignScore[0] > team0CampaignScore)
	{
		team0CampaignScore = g_LPSVersusLastCampaignScore[0];
	}
	if (g_LPSVersusLastCampaignScore[1] > team1CampaignScore)
	{
		team1CampaignScore = g_LPSVersusLastCampaignScore[1];
	}
}

void LPS_ReadVersusScores(
	int &team0MapScore,
	int &team1MapScore,
	int &team0CampaignScore,
	int &team1CampaignScore
)
{
	team0MapScore = GameRules_GetProp("m_iSurvivorScore", 4, 0);
	team1MapScore = GameRules_GetProp("m_iSurvivorScore", 4, 1);
	team0CampaignScore = GameRules_GetProp("m_iCampaignScore", 4, 0);
	team1CampaignScore = GameRules_GetProp("m_iCampaignScore", 4, 1);
}

int LPS_DetermineVersusScoringTeamSlot(int team0MapScore, int team1MapScore)
{
	if (g_LPSVersusRoundStartMapScore[0] >= 0
		&& g_LPSVersusRoundStartMapScore[1] >= 0
		&& team0MapScore >= 0
		&& team1MapScore >= 0)
	{
		int team0Delta = team0MapScore - g_LPSVersusRoundStartMapScore[0];
		int team1Delta = team1MapScore - g_LPSVersusRoundStartMapScore[1];
		if (team0Delta > 0 && team0Delta > team1Delta)
		{
			return 0;
		}
		if (team1Delta > 0 && team1Delta > team0Delta)
		{
			return 1;
		}
	}

	// Some round-start events expose the previous half's map score. The final
	// one-sided score remains useful when the delta is therefore inconclusive.
	if (team0MapScore > 0 && team1MapScore <= 0)
	{
		return 0;
	}
	if (team1MapScore > 0 && team0MapScore <= 0)
	{
		return 1;
	}

	// A survivor team can legitimately score zero. In that case the first and
	// second halves still map to logical team slots 0 and 1 respectively.
	return LPS_GetVersusHalfFallbackTeamSlot();
}

int LPS_GetVersusHalfFallbackTeamSlot()
{
	if (g_LPSRoundHalfNo == 2)
	{
		return 1;
	}
	return 0;
}

bool LPS_IsVersusRoundScoreAvailable(LPSVersusRoundResultSnapshot snapshot)
{
	if (snapshot.scoringTeamSlot == 0)
	{
		return snapshot.team0MapScore >= 0;
	}
	if (snapshot.scoringTeamSlot == 1)
	{
		return snapshot.team1MapScore >= 0;
	}
	return false;
}

void LPS_QueueVersusRoundResult(LPSVersusRoundResultSnapshot snapshot)
{
	if (LPS_HasLifecycleQueueCapacity())
	{
		g_LPSClosedVersusRoundResults.PushArray(snapshot, sizeof(snapshot));
	}
}

void LPS_QueueVersusRunResult(LPSVersusRunResultSnapshot snapshot)
{
	if (LPS_HasLifecycleQueueCapacity())
	{
		g_LPSClosedVersusRunResults.PushArray(snapshot, sizeof(snapshot));
	}
}

int LPS_GetPendingVersusResultCount()
{
	int roundCount = g_LPSClosedVersusRoundResults == null ? 0 : g_LPSClosedVersusRoundResults.Length;
	int runCount = g_LPSClosedVersusRunResults == null ? 0 : g_LPSClosedVersusRunResults.Length;
	return roundCount + runCount;
}

bool LPS_AddVersusResultFlushQueries(Transaction transaction, int &queryCount, char[] error, int errorLength)
{
	g_LPSFlushClosedVersusRoundResultCount = g_LPSClosedVersusRoundResults.Length;
	g_LPSFlushClosedVersusRunResultCount = g_LPSClosedVersusRunResults.Length;

	LPSVersusRoundResultSnapshot roundSnapshot;
	for (int index = 0; index < g_LPSFlushClosedVersusRoundResultCount; index++)
	{
		g_LPSClosedVersusRoundResults.GetArray(index, roundSnapshot, sizeof(roundSnapshot));
		if (!LPS_AddVersusRoundResultQuery(transaction, roundSnapshot, queryCount, error, errorLength))
		{
			return false;
		}
	}

	LPSVersusRunResultSnapshot runSnapshot;
	for (int index = 0; index < g_LPSFlushClosedVersusRunResultCount; index++)
	{
		g_LPSClosedVersusRunResults.GetArray(index, runSnapshot, sizeof(runSnapshot));
		if (!LPS_AddVersusRunResultQuery(transaction, runSnapshot, queryCount, error, errorLength))
		{
			return false;
		}
	}

	return true;
}

bool LPS_AddVersusRoundResultQuery(
	Transaction transaction,
	LPSVersusRoundResultSnapshot snapshot,
	int &queryCount,
	char[] error,
	int errorLength
)
{
	char query[4096];
	char roundId[(LPS_MAX_ROUND_ID * 2) + 1];
	char resultStatus[(LPS_MAX_LIFECYCLE_STATUS * 2) + 1];
	if (!g_LPSDatabase.Escape(snapshot.roundId, roundId, sizeof(roundId))
		|| !g_LPSDatabase.Escape(snapshot.resultStatus, resultStatus, sizeof(resultStatus)))
	{
		strcopy(error, errorLength, "a Versus round result value exceeded its SQL escape buffer");
		return false;
	}

	int savedAt = GetTime();
	if (g_LPSDatabaseDialect == LPSDatabaseDialect_MySQL)
	{
		Format(query, sizeof(query), "INSERT INTO lps_versus_round_results (round_id, stats_version, last_saved_at, scoring_team_slot, teams_flipped, team_0_map_score, team_1_map_score, team_0_campaign_score, team_1_campaign_score, raw_winner_team, score_available, result_status, finalized_at, revision) VALUES ('%s', %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, '%s', %d, %d) ON DUPLICATE KEY UPDATE last_saved_at = VALUES(last_saved_at), scoring_team_slot = VALUES(scoring_team_slot), teams_flipped = VALUES(teams_flipped), team_0_map_score = VALUES(team_0_map_score), team_1_map_score = VALUES(team_1_map_score), team_0_campaign_score = VALUES(team_0_campaign_score), team_1_campaign_score = VALUES(team_1_campaign_score), raw_winner_team = VALUES(raw_winner_team), score_available = VALUES(score_available), result_status = VALUES(result_status), finalized_at = VALUES(finalized_at), revision = VALUES(revision)", roundId, LPS_STATS_VERSION, savedAt, snapshot.scoringTeamSlot, snapshot.teamsFlipped, snapshot.team0MapScore, snapshot.team1MapScore, snapshot.team0CampaignScore, snapshot.team1CampaignScore, snapshot.rawWinnerTeam, snapshot.scoreAvailable, resultStatus, snapshot.finalizedAt, snapshot.revision);
	}
	else
	{
		Format(query, sizeof(query), "INSERT INTO lps_versus_round_results (round_id, stats_version, last_saved_at, scoring_team_slot, teams_flipped, team_0_map_score, team_1_map_score, team_0_campaign_score, team_1_campaign_score, raw_winner_team, score_available, result_status, finalized_at, revision) VALUES ('%s', %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, '%s', %d, %d) ON CONFLICT(round_id) DO UPDATE SET last_saved_at = excluded.last_saved_at, scoring_team_slot = excluded.scoring_team_slot, teams_flipped = excluded.teams_flipped, team_0_map_score = excluded.team_0_map_score, team_1_map_score = excluded.team_1_map_score, team_0_campaign_score = excluded.team_0_campaign_score, team_1_campaign_score = excluded.team_1_campaign_score, raw_winner_team = excluded.raw_winner_team, score_available = excluded.score_available, result_status = excluded.result_status, finalized_at = excluded.finalized_at, revision = excluded.revision", roundId, LPS_STATS_VERSION, savedAt, snapshot.scoringTeamSlot, snapshot.teamsFlipped, snapshot.team0MapScore, snapshot.team1MapScore, snapshot.team0CampaignScore, snapshot.team1CampaignScore, snapshot.rawWinnerTeam, snapshot.scoreAvailable, resultStatus, snapshot.finalizedAt, snapshot.revision);
	}

	transaction.AddQuery(query);
	queryCount++;
	return true;
}

bool LPS_AddVersusRunResultQuery(
	Transaction transaction,
	LPSVersusRunResultSnapshot snapshot,
	int &queryCount,
	char[] error,
	int errorLength
)
{
	char query[3072];
	char runId[(LPS_MAX_RUN_ID * 2) + 1];
	char resultStatus[(LPS_MAX_LIFECYCLE_STATUS * 2) + 1];
	if (!g_LPSDatabase.Escape(snapshot.runId, runId, sizeof(runId))
		|| !g_LPSDatabase.Escape(snapshot.resultStatus, resultStatus, sizeof(resultStatus)))
	{
		strcopy(error, errorLength, "a Versus run result value exceeded its SQL escape buffer");
		return false;
	}

	char finalizedAt[32];
	if (snapshot.finalizedAt > 0) IntToString(snapshot.finalizedAt, finalizedAt, sizeof(finalizedAt));
	else strcopy(finalizedAt, sizeof(finalizedAt), "NULL");
	int savedAt = GetTime();
	if (g_LPSDatabaseDialect == LPSDatabaseDialect_MySQL)
	{
		Format(query, sizeof(query), "INSERT INTO lps_versus_run_results (run_id, stats_version, last_saved_at, team_0_campaign_score, team_1_campaign_score, winner_team_slot, raw_winner_team, score_available, result_status, finalized_at, revision) VALUES ('%s', %d, %d, %d, %d, %d, %d, %d, '%s', %s, %d) ON DUPLICATE KEY UPDATE last_saved_at = VALUES(last_saved_at), team_0_campaign_score = VALUES(team_0_campaign_score), team_1_campaign_score = VALUES(team_1_campaign_score), winner_team_slot = VALUES(winner_team_slot), raw_winner_team = VALUES(raw_winner_team), score_available = VALUES(score_available), result_status = VALUES(result_status), finalized_at = VALUES(finalized_at), revision = VALUES(revision)", runId, LPS_STATS_VERSION, savedAt, snapshot.team0CampaignScore, snapshot.team1CampaignScore, snapshot.winnerTeamSlot, snapshot.rawWinnerTeam, snapshot.scoreAvailable, resultStatus, finalizedAt, snapshot.revision);
	}
	else
	{
		Format(query, sizeof(query), "INSERT INTO lps_versus_run_results (run_id, stats_version, last_saved_at, team_0_campaign_score, team_1_campaign_score, winner_team_slot, raw_winner_team, score_available, result_status, finalized_at, revision) VALUES ('%s', %d, %d, %d, %d, %d, %d, %d, '%s', %s, %d) ON CONFLICT(run_id) DO UPDATE SET last_saved_at = excluded.last_saved_at, team_0_campaign_score = excluded.team_0_campaign_score, team_1_campaign_score = excluded.team_1_campaign_score, winner_team_slot = excluded.winner_team_slot, raw_winner_team = excluded.raw_winner_team, score_available = excluded.score_available, result_status = excluded.result_status, finalized_at = excluded.finalized_at, revision = excluded.revision", runId, LPS_STATS_VERSION, savedAt, snapshot.team0CampaignScore, snapshot.team1CampaignScore, snapshot.winnerTeamSlot, snapshot.rawWinnerTeam, snapshot.scoreAvailable, resultStatus, finalizedAt, snapshot.revision);
	}

	transaction.AddQuery(query);
	queryCount++;
	return true;
}

void LPS_OnVersusResultFlushSucceeded()
{
	for (int index = 0; index < g_LPSFlushClosedVersusRoundResultCount && g_LPSClosedVersusRoundResults.Length > 0; index++)
	{
		g_LPSClosedVersusRoundResults.Erase(0);
	}
	for (int index = 0; index < g_LPSFlushClosedVersusRunResultCount && g_LPSClosedVersusRunResults.Length > 0; index++)
	{
		g_LPSClosedVersusRunResults.Erase(0);
	}
	g_LPSFlushClosedVersusRoundResultCount = 0;
	g_LPSFlushClosedVersusRunResultCount = 0;
}

void LPS_ResetVersusRoundResultState()
{
	g_LPSVersusRoundResultActive = false;
	g_LPSVersusResultRoundId[0] = '\0';
	g_LPSVersusTeamsFlipped = 0;
	g_LPSVersusRoundStartMapScore[0] = -1;
	g_LPSVersusRoundStartMapScore[1] = -1;
	g_LPSVersusRawRoundWinnerTeam = -1;
}

void LPS_ShutdownVersusResults()
{
	delete g_LPSClosedVersusRoundResults;
	delete g_LPSClosedVersusRunResults;
	g_LPSClosedVersusRoundResults = null;
	g_LPSClosedVersusRunResults = null;
}
