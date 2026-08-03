package a2s

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/config"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

type fakeDashboard struct{ primary *store.GameServer }

func (f *fakeDashboard) Ping(context.Context) error                      { return nil }
func (f *fakeDashboard) MigrationVersion(context.Context) (int64, error) { return 1, nil }
func (f *fakeDashboard) Bootstrap(context.Context, config.BootstrapConfig, bool) (bool, error) {
	return false, nil
}
func (f *fakeDashboard) Site(context.Context) (store.Site, error) { return store.Site{}, nil }
func (f *fakeDashboard) PrimaryServer(context.Context) (*store.GameServer, error) {
	return f.primary, nil
}
func (f *fakeDashboard) Close() error { return nil }

type fakeClient struct {
	mu    sync.Mutex
	calls int
	fail  bool
}

func (f *fakeClient) QueryInfo(context.Context, string, time.Duration) (Info, time.Duration, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.fail {
		return Info{}, 0, errors.New("offline")
	}
	return Info{Name: "Test Server", Map: "c1m1_hotel", Players: 3, MaxPlayers: 8, Bots: 1}, 12 * time.Millisecond, nil
}

func TestProviderCachesAndFallsBackToRecentSuccess(t *testing.T) {
	dashboard := &fakeDashboard{primary: &store.GameServer{
		ServerKey: "main", DisplayName: "Configured", ConnectAddress: "127.0.0.1:27015",
		QueryAddress: "127.0.0.1:27015", Primary: true, Enabled: true,
	}}
	client := &fakeClient{}
	provider := NewProvider(dashboard, client)
	provider.ttl = time.Hour

	first, err := provider.PrimaryStatus(context.Background())
	if err != nil || !first.Online || first.Name != "Test Server" {
		t.Fatalf("first status = %#v, %v", first, err)
	}
	second, err := provider.PrimaryStatus(context.Background())
	if err != nil || second.Name != first.Name || client.calls != 1 {
		t.Fatalf("cached status = %#v, %v; calls = %d", second, err, client.calls)
	}

	provider.ttl = time.Nanosecond
	provider.mu.Lock()
	entry := provider.entries[dashboard.primary.QueryAddress]
	entry.expires = time.Now().Add(-time.Second)
	provider.entries[dashboard.primary.QueryAddress] = entry
	provider.mu.Unlock()
	client.fail = true
	stale, err := provider.PrimaryStatus(context.Background())
	if err != nil || stale.Online || !stale.Stale || stale.Name != "Test Server" {
		t.Fatalf("stale status = %#v, %v", stale, err)
	}
}

func TestProviderDoesNotAcceptUnregisteredAddresses(t *testing.T) {
	provider := NewProvider(&fakeDashboard{}, &fakeClient{})
	_, err := provider.PrimaryStatus(context.Background())
	if !errors.Is(err, ErrNoPrimaryServer) {
		t.Fatalf("PrimaryStatus() error = %v", err)
	}
}
