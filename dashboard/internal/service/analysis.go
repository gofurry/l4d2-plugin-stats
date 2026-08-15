package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	"golang.org/x/sync/singleflight"
)

type analysisCacheEntry struct {
	value   any
	expires time.Time
}

type AnalysisService struct {
	stats   store.StatsAnalysisStore
	players *PlayerService
	ttl     time.Duration
	mu      sync.Mutex
	cache   map[string]analysisCacheEntry
	group   singleflight.Group
}

func NewAnalysisService(stats store.StatsAnalysisStore, players ...*PlayerService) *AnalysisService {
	var playerService *PlayerService
	if len(players) > 0 {
		playerService = players[0]
	}
	return &AnalysisService{stats: stats, players: playerService, ttl: 60 * time.Second, cache: make(map[string]analysisCacheEntry)}
}

func (s *AnalysisService) Options(ctx context.Context, filter store.AnalysisFilter) (store.AnalysisOptions, error) {
	value, err := s.cached(ctx, fmt.Sprintf("options:%+v", filter), func(ctx context.Context) (any, error) {
		return s.stats.AnalysisOptions(ctx, filter)
	})
	if err != nil {
		return store.AnalysisOptions{}, err
	}
	return value.(store.AnalysisOptions), nil
}

func (s *AnalysisService) Maps(ctx context.Context, filter store.AnalysisFilter) (store.AnalysisMaps, error) {
	value, err := s.cached(ctx, fmt.Sprintf("maps:%+v", filter), func(ctx context.Context) (any, error) {
		return s.stats.AnalysisMaps(ctx, filter)
	})
	if err != nil {
		return store.AnalysisMaps{}, err
	}
	return value.(store.AnalysisMaps), nil
}

func (s *AnalysisService) MapDetail(ctx context.Context, filter store.AnalysisFilter, mapName string) (store.AnalysisMapDetail, error) {
	value, err := s.cached(ctx, fmt.Sprintf("map:%+v:%s", filter, mapName), func(ctx context.Context) (any, error) {
		return s.stats.AnalysisMapDetail(ctx, filter, mapName)
	})
	if err != nil {
		return store.AnalysisMapDetail{}, err
	}
	return value.(store.AnalysisMapDetail), nil
}

func (s *AnalysisService) Contexts(ctx context.Context, filter store.AnalysisFilter) (store.AnalysisContexts, error) {
	value, err := s.cached(ctx, fmt.Sprintf("contexts:%+v", filter), func(ctx context.Context) (any, error) {
		return s.stats.AnalysisContexts(ctx, filter)
	})
	if err != nil {
		return store.AnalysisContexts{}, err
	}
	return value.(store.AnalysisContexts), nil
}

func (s *AnalysisService) Player(ctx context.Context, steamID string, filter store.PlayerFilter, view string) (store.PlayerAnalysis, error) {
	value, err := s.cached(ctx, fmt.Sprintf("player:%s:%+v:%s", steamID, filter, view), func(ctx context.Context) (any, error) {
		totals, err := s.stats.PlayerAnalysisTotals(ctx, steamID, filter, view)
		if err != nil {
			return nil, err
		}
		incidents, err := s.stats.PlayerIncidentAnalysis(ctx, steamID, filter)
		if err != nil {
			return nil, err
		}
		result := store.PlayerAnalysis{View: view, ActiveSeconds: totals.ActiveSeconds, Metrics: make(map[string]*float64), Samples: make(map[string]int64), Incidents: incidents}
		perHour := func(value int64) *float64 {
			if totals.ActiveSeconds <= 0 {
				return nil
			}
			metric := float64(value) * 3600 / float64(totals.ActiveSeconds)
			return &metric
		}
		ratio := func(value, sample int64) *float64 {
			if sample <= 0 {
				return nil
			}
			metric := float64(value) / float64(sample)
			return &metric
		}
		switch view {
		case "pve":
			if s.players != nil {
				pve, err := s.players.PVEFiltered(ctx, steamID, filter)
				if err != nil {
					return nil, err
				}
				var firearmKills, headshotKills int64
				for _, equipment := range pve.Equipment {
					if !isFirearmEquipment(equipment.EquipmentID) {
						continue
					}
					firearmKills += equipment.CommonKills + equipment.SpecialKills + equipment.TankKills + equipment.WitchKills
					headshotKills += equipment.HeadshotKills
				}
				result.Metrics["common_kills_per_hour"] = perHour(pve.CommonKills)
				result.Metrics["headshot_rate"] = ratio(headshotKills, firearmKills)
				result.Samples["firearm_kills"] = firearmKills
			}
			result.Metrics["special_kills_per_hour"] = perHour(totals.SpecialKills)
			result.Metrics["rescues_per_hour"] = perHour(totals.Rescues)
			result.Metrics["incaps_per_hour"] = perHour(totals.Incaps)
			result.Metrics["deaths_per_hour"] = perHour(totals.Deaths)
			result.Metrics["friendly_fire_per_hour"] = perHour(totals.FriendlyFire)
			result.Metrics["tank_kills_per_hour"] = perHour(totals.TankKills)
			result.Metrics["witch_kills_per_hour"] = perHour(totals.WitchKills)
		case "versus_survivor":
			result.Metrics["human_si_tank_kills_per_hour"] = perHour(totals.SpecialKills)
			result.Metrics["rescues_per_hour"] = perHour(totals.Rescues)
			result.Metrics["incaps_per_hour"] = perHour(totals.Incaps)
			result.Metrics["damage_per_hour"] = perHour(totals.Damage)
		case "versus_infected":
			result.Metrics["damage_per_hour"] = perHour(totals.Damage)
			result.Metrics["incaps_per_spawn"] = ratio(totals.Incaps, totals.Spawns)
			result.Metrics["controls_per_spawn"] = ratio(totals.Controls, totals.Spawns)
			result.Metrics["kills_per_spawn"] = ratio(totals.Kills, totals.Spawns)
			if totals.Controls >= 10 {
				result.Metrics["average_control_seconds"] = ratio(totals.ControlSeconds, totals.Controls)
			} else {
				result.Metrics["average_control_seconds"] = nil
			}
			result.Samples["spawns"] = totals.Spawns
			result.Samples["controls"] = totals.Controls
		}
		return result, nil
	})
	if err != nil {
		return store.PlayerAnalysis{}, err
	}
	return value.(store.PlayerAnalysis), nil
}

func isFirearmEquipment(equipmentID int64) bool {
	return equipmentID >= 1 && equipmentID <= 20 || equipmentID == 22 || equipmentID == 23
}

func (s *AnalysisService) cached(ctx context.Context, key string, load func(context.Context) (any, error)) (any, error) {
	now := time.Now()
	s.mu.Lock()
	if entry, ok := s.cache[key]; ok && now.Before(entry.expires) {
		s.mu.Unlock()
		return entry.value, nil
	}
	s.mu.Unlock()
	value, err, _ := s.group.Do(key, func() (any, error) { return load(ctx) })
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	if len(s.cache) >= 256 {
		for cacheKey := range s.cache {
			delete(s.cache, cacheKey)
			break
		}
	}
	s.cache[key] = analysisCacheEntry{value: value, expires: now.Add(s.ttl)}
	s.mu.Unlock()
	return value, nil
}
