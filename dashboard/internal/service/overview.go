package service

import (
	"context"
	"sync"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	"golang.org/x/sync/singleflight"
)

type OverviewService struct {
	store      store.StatsStore
	aggregates store.DashboardAggregateStore
	ttl        time.Duration
	mu         sync.RWMutex
	value      store.Overview
	until      time.Time
	group      singleflight.Group
}

func NewOverviewService(stats store.StatsStore, ttl time.Duration, aggregates ...store.DashboardAggregateStore) *OverviewService {
	var aggregateStore store.DashboardAggregateStore
	if len(aggregates) > 0 {
		aggregateStore = aggregates[0]
	}
	return &OverviewService{store: stats, aggregates: aggregateStore, ttl: ttl}
}

func (s *OverviewService) Get(ctx context.Context) (store.Overview, error) {
	now := time.Now()
	s.mu.RLock()
	if now.Before(s.until) {
		value := s.value
		s.mu.RUnlock()
		return value, nil
	}
	s.mu.RUnlock()
	ch := s.group.DoChan("overview", func() (any, error) {
		value, err := s.load(context.Background())
		if err != nil {
			return store.Overview{}, err
		}
		s.mu.Lock()
		s.value = value
		s.until = time.Now().Add(s.ttl)
		s.mu.Unlock()
		return value, nil
	})
	select {
	case <-ctx.Done():
		return store.Overview{}, ctx.Err()
	case result := <-ch:
		if result.Err != nil {
			return store.Overview{}, result.Err
		}
		return result.Val.(store.Overview), nil
	}
}

func (s *OverviewService) load(ctx context.Context) (store.Overview, error) {
	if s.aggregates == nil {
		return s.store.Overview(ctx, time.Now().Add(-7*24*time.Hour))
	}
	status, err := s.aggregates.AggregateStatus(ctx)
	if err != nil || status.State != "ready" {
		return s.store.Overview(ctx, time.Now().Add(-7*24*time.Hour))
	}
	rows, err := s.aggregates.ListAggregateRows(ctx, store.AggregateFilter{Grain: store.AggregateGrainLifetime})
	if err != nil {
		return store.Overview{}, err
	}
	cutoffDay := time.Now().UTC().Add(-7*24*time.Hour).Unix() / 86400
	recentActivity, err := s.aggregates.ListAggregateRows(ctx, store.AggregateFilter{Grain: store.AggregateGrainDaily, Kinds: []string{"activity"}, CutoffDay: cutoffDay})
	if err != nil {
		return store.Overview{}, err
	}
	result := store.Overview{Generated: time.Now().UTC()}
	players := make(map[string]struct{})
	active := make(map[string]struct{})
	for _, row := range rows {
		switch row.Kind {
		case "activity":
			if row.SteamID != "" {
				players[row.SteamID] = struct{}{}
			}
			result.Core.TotalActivePlaySeconds += row.Metrics["active_play_seconds"]
		case "run_result":
			if row.Dimension == "pve" {
				result.Core.CompletedPVERuns += row.Metrics["completed_runs"]
			}
		case "versus_result":
			if row.Dimension == "run" {
				result.Core.CompletedVersusRuns += row.Metrics["completed_results"]
				result.Versus.CompletedMatches += row.Metrics["completed_results"]
			}
			if row.Dimension == "round" {
				result.Versus.CompletedHalves += row.Metrics["completed_results"]
			}
		case "pve_combat":
			result.PVE.CommonKills += row.Metrics["common_kills"]
			result.PVE.SpecialKills += row.Metrics["special_kills"]
			result.PVE.TankKills += row.Metrics["tank_kills"]
			result.PVE.WitchKills += row.Metrics["witch_kills"]
			result.PVE.Rescues += row.Metrics["incap_revives"] + row.Metrics["ledge_rescues"] + row.Metrics["defib_revives"]
		case "versus_survivor":
			result.Versus.HumanControlledKills += row.Metrics["human_special_kills"] + row.Metrics["human_tank_kills"]
		case "versus_infected_class":
			result.Versus.HumanSurvivorControls += row.Metrics["human_survivor_controls"]
		}
	}
	for _, row := range recentActivity {
		if row.SteamID != "" && row.Metrics["active_play_seconds"] > 0 {
			active[row.SteamID] = struct{}{}
		}
	}
	result.Core.TotalPlayers = int64(len(players))
	result.Core.ActivePlayers7Days = int64(len(active))
	return result, nil
}
