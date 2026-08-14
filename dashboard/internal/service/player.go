package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	"golang.org/x/sync/singleflight"
)

type playerCacheEntry struct {
	value   any
	expires time.Time
	owner   string
	usedAt  time.Time
}

type PlayerService struct {
	stats      store.StatsStore
	aggregates store.DashboardAggregateStore
	ttl        time.Duration
	capacity   int
	entries    int
	mu         sync.Mutex
	cache      map[string]playerCacheEntry
	players    map[string]time.Time
	group      singleflight.Group
}

const playerActivityTimelineLimit = 30

func NewPlayerService(stats store.StatsStore, aggregates ...store.DashboardAggregateStore) *PlayerService {
	var aggregateStore store.DashboardAggregateStore
	if len(aggregates) > 0 {
		aggregateStore = aggregates[0]
	}
	return &PlayerService{stats: stats, aggregates: aggregateStore, ttl: 60 * time.Second, capacity: 256, entries: 1024, cache: make(map[string]playerCacheEntry), players: make(map[string]time.Time)}
}

func (s *PlayerService) Summary(ctx context.Context, steamID string) (*store.PlayerSummary, error) {
	value, err := s.cached(ctx, steamID, "summary:"+steamID, func(ctx context.Context) (any, error) {
		summary, err := s.stats.PlayerSummary(ctx, steamID)
		if err != nil || summary == nil || s.aggregates == nil {
			return summary, err
		}
		rows, err := s.aggregates.ListAggregateRows(ctx, store.AggregateFilter{Grain: store.AggregateGrainLifetime, Kinds: []string{"activity"}, SteamID: steamID})
		if err != nil {
			return nil, err
		}
		summary.SessionCount, summary.ConnectedSeconds, summary.ActiveSeconds = 0, 0, 0
		for _, row := range rows {
			summary.SessionCount += row.Metrics["session_count"]
			summary.ConnectedSeconds += row.Metrics["connected_seconds"]
			summary.ActiveSeconds += row.Metrics["active_play_seconds"]
		}
		return summary, nil
	})
	if err != nil || value == nil {
		return nil, err
	}
	return value.(*store.PlayerSummary), nil
}

func (s *PlayerService) Preview(ctx context.Context, steamID string) (*store.PlayerPreview, error) {
	value, err := s.cached(ctx, steamID, "preview:"+steamID, func(ctx context.Context) (any, error) {
		summary, err := s.Summary(ctx, steamID)
		if err != nil {
			return nil, err
		}
		if summary == nil {
			return nil, nil
		}
		pve, err := s.PVE(ctx, steamID, 0)
		if err != nil {
			return nil, err
		}
		versus, err := s.Versus(ctx, steamID, 0)
		if err != nil {
			return nil, err
		}
		pveRescues := pve.IncapRevives + pve.LedgeRescues + pve.DefibRevives
		pveBossKills := pve.TankKills + pve.WitchKills
		versusHumanKills := versus.HumanSpecialKills + versus.HumanTankKills
		var headshotKills int64
		for _, equipment := range pve.Equipment {
			headshotKills += equipment.HeadshotKills
		}
		var companions []store.PlayerCompanion
		if analysis, ok := s.stats.(store.StatsAnalysisStore); ok {
			companions, err = analysis.PlayerCompanions(ctx, steamID)
			if err != nil {
				return nil, err
			}
		}
		return &store.PlayerPreview{
			SteamID: summary.SteamID, PlayerName: summary.LastName, SessionCount: summary.SessionCount,
			ActivePlaySeconds: summary.ActiveSeconds, LastSeenAt: summary.LastSeenAt,
			PVE: store.PlayerPreviewPVE{
				Available:   pve.CommonKills+pve.SpecialKills+pveBossKills+headshotKills+pveRescues+pve.CampaignCompletions > 0,
				CommonKills: pve.CommonKills, SpecialKills: pve.SpecialKills, BossKills: pveBossKills, HeadshotKills: headshotKills,
				Rescues: pveRescues, CampaignCompletions: pve.CampaignCompletions,
			},
			Versus: store.PlayerPreviewVersus{
				Available:               versusHumanKills+versus.DamageToHumanSurvivors+versus.HumanSurvivorControls+versus.HumanSurvivorIncaps > 0,
				HumanSIKills:            versusHumanKills,
				InfectedDamage:          versus.DamageToHumanSurvivors,
				SurvivorControls:        versus.HumanSurvivorControls,
				SurvivorIncapacitations: versus.HumanSurvivorIncaps,
			},
			Companions: companions,
		}, nil
	})
	if err != nil || value == nil {
		return nil, err
	}
	return value.(*store.PlayerPreview), nil
}

func (s *PlayerService) PVE(ctx context.Context, steamID string, cutoff int64) (store.PlayerPVE, error) {
	return s.PVEFiltered(ctx, steamID, store.PlayerFilter{Cutoff: cutoff})
}

func (s *PlayerService) PVEFiltered(ctx context.Context, steamID string, filter store.PlayerFilter) (store.PlayerPVE, error) {
	filtered, ok := s.stats.(store.StatsFilteredStore)
	if !ok {
		return s.stats.PlayerPVE(ctx, steamID, filter.Cutoff)
	}
	key := fmt.Sprintf("pve:%s:%d:%s:%s", steamID, filter.Cutoff, filter.ServerKey, filter.GameMode)
	value, err := s.cached(ctx, steamID, key, func(ctx context.Context) (any, error) { return filtered.PlayerPVEFiltered(ctx, steamID, filter) })
	if err != nil {
		return store.PlayerPVE{}, err
	}
	result := value.(store.PlayerPVE)
	if s.aggregates != nil {
		rows, err := s.aggregateRows(ctx, steamID, filter, []string{"pve_equipment"})
		if err != nil {
			return store.PlayerPVE{}, err
		}
		byID := make(map[int64]*store.PVEEquipment)
		for _, row := range rows {
			id, err := strconv.ParseInt(row.Dimension, 10, 64)
			if err != nil {
				continue
			}
			entry := byID[id]
			if entry == nil {
				entry = &store.PVEEquipment{EquipmentID: id}
				byID[id] = entry
			}
			entry.Actions += row.Metrics["actions"]
			entry.CommonKills += row.Metrics["common_kills"]
			entry.SpecialKills += row.Metrics["special_kills"]
			entry.TankKills += row.Metrics["tank_kills"]
			entry.WitchKills += row.Metrics["witch_kills"]
			entry.HeadshotKills += row.Metrics["headshot_kills"]
			entry.DamageToSpecial += row.Metrics["damage_to_special"]
			entry.DamageToTank += row.Metrics["damage_to_tank"]
			entry.DamageToWitch += row.Metrics["damage_to_witch"]
		}
		result.Equipment = result.Equipment[:0]
		for _, entry := range byID {
			result.Equipment = append(result.Equipment, *entry)
		}
		sort.Slice(result.Equipment, func(i, j int) bool { return result.Equipment[i].EquipmentID < result.Equipment[j].EquipmentID })
	}
	return result, nil
}

func (s *PlayerService) Versus(ctx context.Context, steamID string, cutoff int64) (store.PlayerVersus, error) {
	return s.VersusFiltered(ctx, steamID, store.PlayerFilter{Cutoff: cutoff})
}

func (s *PlayerService) VersusFiltered(ctx context.Context, steamID string, filter store.PlayerFilter) (store.PlayerVersus, error) {
	filtered, ok := s.stats.(store.StatsFilteredStore)
	if !ok {
		value, err := s.stats.PlayerVersus(ctx, steamID, filter.Cutoff)
		return value, err
	}
	key := fmt.Sprintf("versus:%s:%d:%s", steamID, filter.Cutoff, filter.ServerKey)
	value, err := s.cached(ctx, steamID, key, func(ctx context.Context) (any, error) { return filtered.PlayerVersusFiltered(ctx, steamID, filter) })
	if err != nil {
		return store.PlayerVersus{}, err
	}
	result := value.(store.PlayerVersus)
	if s.aggregates != nil {
		rawAssistByClass := make(map[int64][2]*int64, len(result.SurvivorClasses))
		for _, entry := range result.SurvivorClasses {
			rawAssistByClass[entry.ClassID] = [2]*int64{entry.HumanControllerAssists, entry.BotControllerAssists}
		}
		rows, err := s.aggregateRows(ctx, steamID, filter, []string{"versus_survivor_class", "versus_infected_class"})
		if err != nil {
			return store.PlayerVersus{}, err
		}
		survivor := make(map[int64]*store.VersusSurvivorClass)
		infected := make(map[int64]*store.VersusInfectedClass)
		for _, row := range rows {
			id, err := strconv.ParseInt(row.Dimension, 10, 64)
			if err != nil {
				continue
			}
			if row.Kind == "versus_survivor_class" {
				entry := survivor[id]
				if entry == nil {
					assists := rawAssistByClass[id]
					entry = &store.VersusSurvivorClass{ClassID: id, HumanControllerAssists: assists[0], BotControllerAssists: assists[1]}
					survivor[id] = entry
				}
				entry.HumanControllerKills += row.Metrics["human_controller_kills"]
				entry.BotControllerKills += row.Metrics["bot_controller_kills"]
				entry.DamageToHumanControllers += row.Metrics["damage_to_human_controllers"]
				entry.DamageToBotControllers += row.Metrics["damage_to_bot_controllers"]
			} else {
				entry := infected[id]
				if entry == nil {
					entry = &store.VersusInfectedClass{ClassID: id}
					infected[id] = entry
				}
				addVersusInfectedClass(entry, row.Metrics)
			}
		}
		result.SurvivorClasses = result.SurvivorClasses[:0]
		result.InfectedClasses = result.InfectedClasses[:0]
		for _, entry := range survivor {
			result.SurvivorClasses = append(result.SurvivorClasses, *entry)
		}
		for _, entry := range infected {
			result.InfectedClasses = append(result.InfectedClasses, *entry)
		}
		sort.Slice(result.SurvivorClasses, func(i, j int) bool { return result.SurvivorClasses[i].ClassID < result.SurvivorClasses[j].ClassID })
		sort.Slice(result.InfectedClasses, func(i, j int) bool { return result.InfectedClasses[i].ClassID < result.InfectedClasses[j].ClassID })
	}
	return result, nil
}

func (s *PlayerService) Relationships(ctx context.Context, steamID string, query store.PlayerRelationshipQuery) (store.PlayerRelationshipPage, error) {
	relationships, ok := s.stats.(store.StatsRelationshipStore)
	if !ok {
		return store.PlayerRelationshipPage{}, fmt.Errorf("player relationships are unavailable")
	}
	key := fmt.Sprintf("relationships:%s:%d:%s:%s:%d:%d:%s:%s", steamID, query.Cutoff, query.ServerKey, query.GameMode, query.Page, query.PageSize, query.Sort, query.Order)
	value, err := s.cached(ctx, steamID, key, func(ctx context.Context) (any, error) {
		return relationships.PlayerRelationships(ctx, steamID, query)
	})
	if err != nil {
		return store.PlayerRelationshipPage{}, err
	}
	return value.(store.PlayerRelationshipPage), nil
}

func (s *PlayerService) Activity(ctx context.Context, steamID string, cutoff int64) (store.PlayerActivity, error) {
	return s.ActivityFiltered(ctx, steamID, store.PlayerFilter{Cutoff: cutoff})
}

func (s *PlayerService) ActivityFiltered(ctx context.Context, steamID string, filter store.PlayerFilter) (store.PlayerActivity, error) {
	if s.aggregates != nil {
		key := fmt.Sprintf("activity:%s:%d:%s", steamID, filter.Cutoff, filter.ServerKey)
		value, err := s.cached(ctx, steamID, key, func(ctx context.Context) (any, error) {
			cutoffDay := int64(0)
			if filter.Cutoff > 0 {
				cutoffDay = filter.Cutoff / 86400
			}
			rows, err := s.aggregates.ListAggregateRows(ctx, store.AggregateFilter{
				Grain: store.AggregateGrainDaily, Kinds: []string{"activity"}, SteamID: steamID,
				ServerKey: filter.ServerKey, CutoffDay: cutoffDay,
			})
			if err != nil {
				return store.PlayerActivity{}, err
			}
			byDay := make(map[int64]*store.PlayerActivityPoint)
			byServer := make(map[string]*store.PlayerServerActivity)
			for _, row := range rows {
				point := byDay[row.Day]
				if point == nil {
					point = &store.PlayerActivityPoint{Day: row.Day}
					byDay[row.Day] = point
				}
				point.SessionCount += row.Metrics["session_count"]
				point.ConnectedSeconds += row.Metrics["connected_seconds"]
				point.ActiveSeconds += row.Metrics["active_play_seconds"]
				server := byServer[row.ServerKey]
				if server == nil {
					server = &store.PlayerServerActivity{ServerKey: row.ServerKey}
					byServer[row.ServerKey] = server
				}
				server.SessionCount += row.Metrics["session_count"]
				server.ActiveSeconds += row.Metrics["active_play_seconds"]
			}
			result := store.PlayerActivity{}
			for _, point := range byDay {
				result.Timeline = append(result.Timeline, *point)
			}
			for _, server := range byServer {
				result.Servers = append(result.Servers, *server)
			}
			sort.Slice(result.Timeline, func(i, j int) bool { return result.Timeline[i].Day < result.Timeline[j].Day })
			if len(result.Timeline) > playerActivityTimelineLimit {
				result.Timeline = result.Timeline[len(result.Timeline)-playerActivityTimelineLimit:]
			}
			sort.Slice(result.Servers, func(i, j int) bool { return result.Servers[i].ServerKey < result.Servers[j].ServerKey })
			return result, nil
		})
		if err != nil {
			return store.PlayerActivity{}, err
		}
		return value.(store.PlayerActivity), nil
	}
	filtered, ok := s.stats.(store.StatsFilteredStore)
	if !ok {
		value, err := s.stats.PlayerActivity(ctx, steamID, filter.Cutoff)
		return value, err
	}
	key := fmt.Sprintf("activity:%s:%d:%s", steamID, filter.Cutoff, filter.ServerKey)
	value, err := s.cached(ctx, steamID, key, func(ctx context.Context) (any, error) { return filtered.PlayerActivityFiltered(ctx, steamID, filter) })
	if err != nil {
		return store.PlayerActivity{}, err
	}
	return value.(store.PlayerActivity), nil
}

func (s *PlayerService) aggregateRows(ctx context.Context, steamID string, filter store.PlayerFilter, kinds []string) ([]store.AggregateRow, error) {
	cutoffDay := int64(0)
	grain := store.AggregateGrainDaily
	if filter.Cutoff > 0 {
		cutoffDay = filter.Cutoff / 86400
	} else if len(kinds) == 1 && kinds[0] == "activity" {
		grain = store.AggregateGrainMonthly
	} else {
		grain = store.AggregateGrainLifetime
	}
	rows, err := s.aggregates.ListAggregateRows(ctx, store.AggregateFilter{Grain: grain, Kinds: kinds, SteamID: steamID, ServerKey: filter.ServerKey, CutoffDay: cutoffDay})
	if err != nil {
		return nil, err
	}
	if filter.GameMode == "" {
		return rows, nil
	}
	filtered := rows[:0]
	for _, row := range rows {
		if row.Mode == filter.GameMode {
			filtered = append(filtered, row)
		}
	}
	return filtered, nil
}

func addVersusInfectedClass(entry *store.VersusInfectedClass, metrics map[string]int64) {
	entry.Spawns += metrics["spawn_count"]
	entry.DamageToHumanSurvivors += metrics["damage_to_human_survivors"]
	entry.DamageToBotSurvivors += metrics["damage_to_bot_survivors"]
	entry.HumanSurvivorIncaps += metrics["human_survivor_incaps"]
	entry.BotSurvivorIncaps += metrics["bot_survivor_incaps"]
	entry.HumanSurvivorKills += metrics["human_survivor_kills"]
	entry.BotSurvivorKills += metrics["bot_survivor_kills"]
	entry.HumanSurvivorControls += metrics["human_survivor_controls"]
	entry.BotSurvivorControls += metrics["bot_survivor_controls"]
	entry.HumanSurvivorControlSeconds += metrics["human_survivor_control_seconds"]
	entry.BotSurvivorControlSeconds += metrics["bot_survivor_control_seconds"]
	entry.HumanSurvivorAbilityHits += metrics["human_survivor_ability_hits"]
	entry.BotSurvivorAbilityHits += metrics["bot_survivor_ability_hits"]
	entry.HumanSurvivorAbilityDamage += metrics["human_survivor_ability_damage"]
	entry.BotSurvivorAbilityDamage += metrics["bot_survivor_ability_damage"]
}

func (s *PlayerService) Sessions(ctx context.Context, steamID string, at int64, id string, limit int32) ([]store.PlayerSession, error) {
	return s.stats.PlayerSessions(ctx, steamID, at, id, limit)
}
func (s *PlayerService) Chapters(ctx context.Context, steamID string, at int64, id string, limit int32) ([]store.PlayerChapter, error) {
	return s.stats.PlayerChapters(ctx, steamID, at, id, limit)
}

func (s *PlayerService) cached(ctx context.Context, steamID, key string, load func(context.Context) (any, error)) (any, error) {
	now := time.Now()
	s.mu.Lock()
	entry, ok := s.cache[key]
	if ok && now.Before(entry.expires) {
		entry.usedAt = now
		s.cache[key] = entry
		s.players[steamID] = now
	}
	s.mu.Unlock()
	if ok && now.Before(entry.expires) {
		return entry.value, nil
	}
	ch := s.group.DoChan(key, func() (any, error) { return load(context.Background()) })
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-ch:
		if result.Err != nil {
			return nil, result.Err
		}
		s.put(key, steamID, result.Val)
		return result.Val, nil
	}
}

func (s *PlayerService) put(key, steamID string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for k, v := range s.cache {
		if now.After(v.expires) {
			delete(s.cache, k)
		}
	}
	present := make(map[string]bool, len(s.players))
	for _, entry := range s.cache {
		present[entry.owner] = true
	}
	for player := range s.players {
		if !present[player] {
			delete(s.players, player)
		}
	}
	if _, exists := s.players[steamID]; !exists && len(s.players) >= s.capacity {
		oldestPlayer := ""
		oldestTime := now
		for player, usedAt := range s.players {
			if oldestPlayer == "" || usedAt.Before(oldestTime) {
				oldestPlayer, oldestTime = player, usedAt
			}
		}
		for cacheKey, entry := range s.cache {
			if entry.owner == oldestPlayer {
				delete(s.cache, cacheKey)
			}
		}
		delete(s.players, oldestPlayer)
	}
	for len(s.cache) >= s.entries {
		oldestKey := ""
		oldestAt := now
		for cacheKey, entry := range s.cache {
			if oldestKey == "" || entry.usedAt.Before(oldestAt) {
				oldestKey, oldestAt = cacheKey, entry.usedAt
			}
		}
		delete(s.cache, oldestKey)
	}
	s.players[steamID] = now
	s.cache[key] = playerCacheEntry{value: value, expires: now.Add(s.ttl), owner: steamID, usedAt: now}
}
