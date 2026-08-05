package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

func TestRankingCacheHasHardEntryLimit(t *testing.T) {
	service := &RankingService{cache: make(map[string]rankingCacheEntry)}
	now := time.Now()
	for index := 0; index < rankingCacheCapacity+20; index++ {
		service.storeCache(fmt.Sprintf("key-%d", index), store.RankingPage{}, now.Add(time.Duration(index)*time.Millisecond))
	}
	if got := len(service.cache); got != rankingCacheCapacity {
		t.Fatalf("ranking cache size = %d, want %d", got, rankingCacheCapacity)
	}
}

func TestPlayerCacheHasHardEntryLimit(t *testing.T) {
	service := &PlayerService{
		ttl: time.Minute, capacity: 256, entries: 3,
		cache: make(map[string]playerCacheEntry), players: make(map[string]time.Time),
	}
	for index := 0; index < 8; index++ {
		service.put(fmt.Sprintf("key-%d", index), "76561198000000001", index)
	}
	if got := len(service.cache); got != service.entries {
		t.Fatalf("player cache size = %d, want %d", got, service.entries)
	}
}
