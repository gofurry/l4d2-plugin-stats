package service

import (
	"context"
	"testing"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

type activityStatsStore struct{ store.StatsStore }

type activityAggregateStore struct {
	store.DashboardAggregateStore
	filter store.AggregateFilter
	rows   []store.AggregateRow
}

func (s *activityAggregateStore) ListAggregateRows(_ context.Context, filter store.AggregateFilter) ([]store.AggregateRow, error) {
	s.filter = filter
	return s.rows, nil
}

func TestPlayerActivityUsesDailyBucketsAndCapsTimeline(t *testing.T) {
	rows := make([]store.AggregateRow, 0, 370)
	for day := int64(1); day <= 370; day++ {
		rows = append(rows, store.AggregateRow{
			Version: store.AggregateContractVersion, Kind: "activity", Day: day,
			ServerKey: "server-one", SteamID: "76561198000000000",
			Metrics: map[string]int64{"session_count": 1, "connected_seconds": 2, "active_play_seconds": 1},
		})
	}
	aggregates := &activityAggregateStore{rows: rows}
	service := NewPlayerService(&activityStatsStore{}, aggregates)

	result, err := service.ActivityFiltered(context.Background(), "76561198000000000", store.PlayerFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if aggregates.filter.Grain != store.AggregateGrainDaily {
		t.Fatalf("grain=%q, want daily", aggregates.filter.Grain)
	}
	if len(result.Timeline) != playerActivityTimelineLimit || result.Timeline[0].Day != 341 || result.Timeline[29].Day != 370 {
		t.Fatalf("unexpected timeline: len=%d first=%d last=%d", len(result.Timeline), result.Timeline[0].Day, result.Timeline[len(result.Timeline)-1].Day)
	}
	if len(result.Servers) != 1 || result.Servers[0].ActiveSeconds != 370 {
		t.Fatalf("server distribution should retain all-time totals: %+v", result.Servers)
	}
}
