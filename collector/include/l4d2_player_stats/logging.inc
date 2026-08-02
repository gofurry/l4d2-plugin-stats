#if defined _l4d2_player_stats_logging_included
	#endinput
#endif
#define _l4d2_player_stats_logging_included

char g_LPSLastLoggedError[LPS_MAX_DATABASE_ERROR];
int g_LPSLastErrorLogAt;
int g_LPSSuppressedErrorCount;

void LPS_ReportDatabaseError(const char[] context, const char[] error)
{
	Format(g_LPSLastDatabaseError, sizeof(g_LPSLastDatabaseError), "%s: %s", context, error);

	int now = GetTime();
	int window = g_LPSLogSuppressionWindow.IntValue;
	bool sameError = StrEqual(g_LPSLastLoggedError, g_LPSLastDatabaseError, true);

	if (sameError && g_LPSLastErrorLogAt > 0 && now - g_LPSLastErrorLogAt < window)
	{
		g_LPSSuppressedErrorCount++;
		return;
	}

	if (g_LPSSuppressedErrorCount > 0)
	{
		LogError(
			"Suppressed %d repeated database error(s) during the previous logging window.",
			g_LPSSuppressedErrorCount
		);
	}

	LogError("Database %s", g_LPSLastDatabaseError);
	strcopy(g_LPSLastLoggedError, sizeof(g_LPSLastLoggedError), g_LPSLastDatabaseError);
	g_LPSLastErrorLogAt = now;
	g_LPSSuppressedErrorCount = 0;
}

void LPS_ReportDatabaseRecovery()
{
	if (g_LPSLastLoggedError[0] == '\0')
	{
		return;
	}

	if (g_LPSSuppressedErrorCount > 0)
	{
		LogMessage(
			"Database recovered after suppressing %d repeated error(s).",
			g_LPSSuppressedErrorCount
		);
	}
	else
	{
		LogMessage("Database connection recovered.");
	}

	g_LPSLastLoggedError[0] = '\0';
	g_LPSLastDatabaseError[0] = '\0';
	g_LPSLastErrorLogAt = 0;
	g_LPSSuppressedErrorCount = 0;
}
