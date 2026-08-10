#if defined _l4d2_player_stats_survivor_incidents_included
	#endinput
#endif
#define _l4d2_player_stats_survivor_incidents_included

// Incident signals live in one module so future survivor-triggered events can
// share strict attribution rules without mixing PvE and Versus storage.
void LPS_InitializeSurvivorIncidents()
{
	HookEvent("triggered_car_alarm", LPS_Event_CarAlarmTriggered, EventHookMode_Post);
}

void LPS_ShutdownSurvivorIncidents()
{
	UnhookEvent("triggered_car_alarm", LPS_Event_CarAlarmTriggered, EventHookMode_Post);
}

public void LPS_Event_CarAlarmTriggered(Event event, const char[] name, bool dontBroadcast)
{
	int client = GetClientOfUserId(event.GetInt("userid"));
	if (!LPS_IsHumanSurvivorClient(client))
	{
		return;
	}

	if (LPS_CanRecordPvEEvents())
	{
		LPS_AddPvEStat(client, LPSPvEStat_CarAlarmsTriggered, 1);
		return;
	}

	if (LPS_CanRecordVersusEvents())
	{
		LPS_AddVersusSurvivorStat(client, LPSVersusSurvivorStat_CarAlarmsTriggered, 1);
	}
}
