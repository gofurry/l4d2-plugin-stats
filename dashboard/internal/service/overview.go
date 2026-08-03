package service

import (
	"context"
	"sync"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	"golang.org/x/sync/singleflight"
)

type OverviewService struct {
	store store.StatsStore
	ttl   time.Duration
	mu    sync.RWMutex
	value store.Overview
	until time.Time
	group singleflight.Group
}

func NewOverviewService(stats store.StatsStore, ttl time.Duration) *OverviewService {
	return &OverviewService{store: stats, ttl: ttl}
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
		value, err := s.store.Overview(context.Background(), time.Now().Add(-7*24*time.Hour))
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
