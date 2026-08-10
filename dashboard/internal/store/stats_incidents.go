package store

import (
	"context"
	"fmt"
)

// CarAlarmRanking reads the additive incident counter directly from Stats DB.
// It intentionally stays outside Aggregate Contract v1 so that adding this
// field does not silently change the frozen aggregate metric set.
func (s *statsStore) CarAlarmRanking(ctx context.Context, query RankingQuery) ([]RankingEntry, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	table := "lps_pve_segment_stats"
	modeCondition := "r.mode_family='pve' AND r.game_mode IN ('coop','realism')"
	if query.Mode == "versus_survivor" {
		table = "lps_versus_survivor_stats"
		modeCondition = "r.mode_family='versus' AND r.game_mode='versus'"
	} else if query.Mode != "pve" {
		return nil, fmt.Errorf("unsupported car alarm ranking mode %q", query.Mode)
	}

	where := " WHERE s.side='survivor' AND p.stats_version=1 AND " + modeCondition
	args := make([]any, 0, 2)
	if query.Cutoff > 0 {
		args = append(args, query.Cutoff)
		where += " AND s.started_at >= " + s.bind(len(args))
	}
	if query.ServerKey != "" {
		args = append(args, query.ServerKey)
		where += " AND s.server_key = " + s.bind(len(args))
	}

	statement := fmt.Sprintf(`SELECT s.steam_id, MAX(pl.last_name),
COALESCE(SUM(COALESCE(p.car_alarms_triggered,0)),0), COALESCE(SUM(s.active_play_seconds),0)
FROM %s p
JOIN lps_player_segments s ON s.segment_id=p.segment_id
JOIN lps_runs r ON r.run_id=s.run_id
JOIN lps_players pl ON pl.steam_id=s.steam_id%s
GROUP BY s.steam_id
HAVING SUM(COALESCE(p.car_alarms_triggered,0)) > 0
ORDER BY SUM(COALESCE(p.car_alarms_triggered,0)) DESC, s.steam_id ASC`, table, where)
	rows, err := s.db.QueryContext(queryCtx, statement, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]RankingEntry, 0)
	for rows.Next() {
		var entry RankingEntry
		var value any
		if err := rows.Scan(&entry.SteamID, &entry.PlayerName, &value, &entry.ActiveSeconds); err != nil {
			return nil, err
		}
		entry.Value = float64(integerValue(value))
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for index := range entries {
		entries[index].Rank = int64(index + 1)
	}
	return entries, nil
}
