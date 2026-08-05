package service

import (
	"context"
	"fmt"
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
	stats    store.StatsStore
	ttl      time.Duration
	capacity int
	entries  int
	mu       sync.Mutex
	cache    map[string]playerCacheEntry
	players  map[string]time.Time
	group    singleflight.Group
}

func NewPlayerService(stats store.StatsStore) *PlayerService {
	return &PlayerService{stats: stats, ttl: 60 * time.Second, capacity: 256, entries: 1024, cache: make(map[string]playerCacheEntry), players: make(map[string]time.Time)}
}

func (s *PlayerService) Summary(ctx context.Context, steamID string) (*store.PlayerSummary, error) {
	value, err := s.cached(ctx, steamID, "summary:"+steamID, func(ctx context.Context) (any, error) { return s.stats.PlayerSummary(ctx, steamID) })
	if err != nil || value == nil {
		return nil, err
	}
	return value.(*store.PlayerSummary), nil
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
	return value.(store.PlayerPVE), nil
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
	return value.(store.PlayerVersus), nil
}

func (s *PlayerService) Activity(ctx context.Context, steamID string, cutoff int64) (store.PlayerActivity, error) {
	return s.ActivityFiltered(ctx, steamID, store.PlayerFilter{Cutoff: cutoff})
}

func (s *PlayerService) ActivityFiltered(ctx context.Context, steamID string, filter store.PlayerFilter) (store.PlayerActivity, error) {
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
