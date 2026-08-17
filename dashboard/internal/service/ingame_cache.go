package service

import (
	"context"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type ingameCacheEntry struct {
	value   any
	expires time.Time
	usedAt  time.Time
}

type ingameBuildResult struct {
	value any
	ttl   time.Duration
}

type ingameViewCache struct {
	mu       sync.Mutex
	entries  map[string]ingameCacheEntry
	capacity int
	group    singleflight.Group
}

func newIngameViewCache() *ingameViewCache {
	return &ingameViewCache{entries: make(map[string]ingameCacheEntry), capacity: 512}
}

func (c *ingameViewCache) get(ctx context.Context, key string, build func(context.Context) (ingameBuildResult, error)) (any, error) {
	now := time.Now()
	c.mu.Lock()
	stale, exists := c.entries[key]
	if exists && now.Before(stale.expires) {
		stale.usedAt = now
		c.entries[key] = stale
		c.mu.Unlock()
		return stale.value, nil
	}
	c.mu.Unlock()

	result := c.group.DoChan(key, func() (any, error) {
		buildCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		built, err := build(buildCtx)
		if err != nil {
			return nil, err
		}
		if built.ttl <= 0 {
			built.ttl = 30 * time.Second
		}
		c.put(key, built.value, built.ttl)
		return built.value, nil
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case loaded := <-result:
		if loaded.Err != nil {
			if exists {
				return stale.value, nil
			}
			return nil, loaded.Err
		}
		return loaded.Val, nil
	}
}

func (c *ingameViewCache) put(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for cacheKey, entry := range c.entries {
		if now.After(entry.expires) && len(c.entries) >= c.capacity/2 {
			delete(c.entries, cacheKey)
		}
	}
	for len(c.entries) >= c.capacity {
		oldestKey := ""
		oldestAt := now
		for cacheKey, entry := range c.entries {
			if oldestKey == "" || entry.usedAt.Before(oldestAt) {
				oldestKey, oldestAt = cacheKey, entry.usedAt
			}
		}
		delete(c.entries, oldestKey)
	}
	c.entries[key] = ingameCacheEntry{value: value, expires: now.Add(ttl), usedAt: now}
}

func (c *ingameViewCache) clear() {
	c.mu.Lock()
	c.entries = make(map[string]ingameCacheEntry)
	c.mu.Unlock()
}

func (c *ingameViewCache) clearServer(serverKey string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.entries {
		parts := strings.SplitN(key, ":", 3)
		if len(parts) >= 2 && parts[1] == serverKey {
			delete(c.entries, key)
		}
	}
}

func (c *ingameViewCache) clearPlayer(steamID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.entries {
		if strings.HasPrefix(key, "player:") && strings.HasSuffix(key, ":"+steamID) {
			delete(c.entries, key)
		}
	}
}
