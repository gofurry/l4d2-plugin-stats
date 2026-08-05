package a2s

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

type fakeDashboard struct {
	servers  []store.GameServer
	settings store.SiteSettings
}

func (f *fakeDashboard) Ping(context.Context) error                      { return nil }
func (f *fakeDashboard) MigrationVersion(context.Context) (int64, error) { return 3, nil }
func (f *fakeDashboard) Site(context.Context) (store.Site, error)        { return store.Site{}, nil }
func (f *fakeDashboard) SiteSettings(context.Context) (store.SiteSettings, error) {
	return f.settings, nil
}
func (f *fakeDashboard) UpdateSite(context.Context, store.SiteSettings) error { return nil }
func (f *fakeDashboard) ListSiteDocuments(context.Context, bool) ([]store.SiteDocument, error) {
	return []store.SiteDocument{}, nil
}
func (f *fakeDashboard) GetSiteDocument(context.Context, string, bool) (store.SiteDocument, error) {
	return store.SiteDocument{}, nil
}
func (f *fakeDashboard) UpdateSiteDocument(_ context.Context, document store.SiteDocument) (store.SiteDocument, error) {
	return document, nil
}
func (f *fakeDashboard) ListServers(context.Context) ([]store.GameServer, error) {
	return f.servers, nil
}
func (f *fakeDashboard) CreateServer(_ context.Context, s store.GameServer) (store.GameServer, error) {
	return s, nil
}
func (f *fakeDashboard) UpdateServer(context.Context, store.GameServer) error      { return nil }
func (f *fakeDashboard) SetServerEnabled(context.Context, string, bool) error      { return nil }
func (f *fakeDashboard) MoveServer(context.Context, string, string) error          { return nil }
func (f *fakeDashboard) DeleteServer(context.Context, string) error                { return nil }
func (f *fakeDashboard) AdminConfigured(context.Context) (bool, error)             { return false, nil }
func (f *fakeDashboard) CreateAdmin(context.Context, string, string, string) error { return nil }
func (f *fakeDashboard) Admin(context.Context) (*store.AdminAccount, error)        { return nil, nil }
func (f *fakeDashboard) UpdateAdminUsername(context.Context, string) error         { return nil }
func (f *fakeDashboard) UpdateAdminPassword(context.Context, string) error         { return nil }
func (f *fakeDashboard) ListAnnouncements(context.Context, store.AnnouncementFilter) (store.AnnouncementPage, error) {
	return store.AnnouncementPage{Items: []store.Announcement{}}, nil
}
func (f *fakeDashboard) ListAnnouncementYears(context.Context) ([]int, error) { return []int{}, nil }
func (f *fakeDashboard) GetAnnouncement(context.Context, string) (store.Announcement, error) {
	return store.Announcement{}, nil
}
func (f *fakeDashboard) CreateAnnouncement(_ context.Context, value store.Announcement) (store.Announcement, error) {
	return value, nil
}
func (f *fakeDashboard) UpdateAnnouncement(_ context.Context, value store.Announcement) (store.Announcement, error) {
	return value, nil
}
func (f *fakeDashboard) DeleteAnnouncement(context.Context, string) error { return nil }
func (f *fakeDashboard) Close() error                                     { return nil }

type fakeClient struct {
	mu    sync.Mutex
	calls int
	fail  bool
}

func (f *fakeClient) Query(context.Context, string, time.Duration) (Info, []Player, []Rule, time.Duration, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.fail {
		return Info{}, nil, nil, 0, errors.New("offline")
	}
	return Info{Name: "Test Server", Map: "c1m1_hotel", Players: 3, MaxPlayers: 8, Bots: 1}, []Player{{Name: "Coach", Score: 8, DurationSeconds: 90}}, []Rule{{Name: "sv_tags", Value: "coop"}}, 12 * time.Millisecond, nil
}

func TestProviderCachesAndFallsBackToRecentSuccess(t *testing.T) {
	dashboard := &fakeDashboard{servers: []store.GameServer{{
		ID: "main", DisplayName: "Configured", Address: "127.0.0.1:27015", Enabled: true,
	}}}
	client := &fakeClient{}
	provider := NewProvider(dashboard, client)
	provider.ttl = time.Hour

	firstList, err := provider.Statuses(context.Background())
	first := firstList[0]
	if err != nil || !first.Online || first.Name != "Test Server" || first.ServerID != "main" {
		t.Fatalf("first status = %#v, %v", first, err)
	}
	secondList, err := provider.Statuses(context.Background())
	second := secondList[0]
	if err != nil || second.Name != first.Name || client.calls != 1 {
		t.Fatalf("cached status = %#v, %v; calls = %d", second, err, client.calls)
	}

	provider.ttl = time.Nanosecond
	provider.mu.Lock()
	key := serverCacheKey(dashboard.servers[0])
	entry := provider.entries[key]
	entry.expires = time.Now().Add(-time.Second)
	provider.entries[key] = entry
	provider.mu.Unlock()
	client.fail = true
	staleList, err := provider.Statuses(context.Background())
	stale := staleList[0]
	if err != nil || stale.Online || !stale.Stale || stale.Name != "Test Server" {
		t.Fatalf("stale status = %#v, %v", stale, err)
	}
}

func TestProviderDoesNotAcceptUnregisteredAddresses(t *testing.T) {
	provider := NewProvider(&fakeDashboard{}, &fakeClient{})
	statuses, err := provider.Statuses(context.Background())
	if err != nil || len(statuses) != 0 {
		t.Fatalf("Statuses() = %#v, %v", statuses, err)
	}
}

func TestProviderAdminRefreshUsesRegisteredServerAndExposesLastResult(t *testing.T) {
	dashboard := &fakeDashboard{servers: []store.GameServer{{
		ID: "hidden", DisplayName: "Hidden", Address: "127.0.0.1:27017", Enabled: false,
	}}}
	client := &fakeClient{}
	provider := NewProvider(dashboard, client)

	if _, available, err := provider.LastStatus(context.Background(), "hidden"); err != nil || available {
		t.Fatalf("initial LastStatus() available=%v err=%v", available, err)
	}
	refreshed, err := provider.RefreshStatus(context.Background(), "hidden")
	if err != nil || !refreshed.Online || refreshed.ServerID != "hidden" || client.calls != 1 {
		t.Fatalf("RefreshStatus()=%#v err=%v calls=%d", refreshed, err, client.calls)
	}
	last, available, err := provider.LastStatus(context.Background(), "hidden")
	if err != nil || !available || last.CheckedAt.IsZero() || last.Name != "Test Server" {
		t.Fatalf("LastStatus()=%#v available=%v err=%v", last, available, err)
	}
	if _, err := provider.RefreshStatus(context.Background(), "unknown"); !errors.Is(err, store.ErrServerNotFound) {
		t.Fatalf("unknown RefreshStatus() error=%v", err)
	}
}

func TestProviderListsEveryEnabledServerInConfiguredOrder(t *testing.T) {
	dashboard := &fakeDashboard{servers: []store.GameServer{
		{ID: "first", DisplayName: "First", Address: "127.0.0.1:27015", Enabled: true},
		{ID: "second", DisplayName: "Second", Address: "127.0.0.1:27016", Enabled: true},
		{ID: "hidden", DisplayName: "Hidden", Address: "127.0.0.1:27017", Enabled: false},
	}}
	client := &fakeClient{}
	statuses, err := NewProvider(dashboard, client).Statuses(context.Background())
	if err != nil || len(statuses) != 2 {
		t.Fatalf("Statuses() = %#v, %v", statuses, err)
	}
	if statuses[0].ServerID != "first" || statuses[1].ServerID != "second" || client.calls != 2 {
		t.Fatalf("ordered statuses = %#v; calls = %d", statuses, client.calls)
	}
}

func TestProviderRetriesFailuresUsingConfiguredCount(t *testing.T) {
	dashboard := &fakeDashboard{
		servers:  []store.GameServer{{ID: "main", DisplayName: "Main", Address: "127.0.0.1:27015", Enabled: true}},
		settings: store.SiteSettings{A2SRefreshSeconds: 5, A2SJitterSeconds: 0, A2SRetryCount: 3},
	}
	client := &fakeClient{fail: true}
	statuses, err := NewProvider(dashboard, client).Statuses(context.Background())
	if err != nil || len(statuses) != 1 || statuses[0].Online || client.calls != 4 {
		t.Fatalf("Statuses()=%#v err=%v calls=%d", statuses, err, client.calls)
	}
}

func TestProviderPurgesRemovedAndChangedServerCacheEntries(t *testing.T) {
	dashboard := &fakeDashboard{servers: []store.GameServer{{
		ID: "main", DisplayName: "Main", Address: "127.0.0.1:27015", Enabled: true,
	}}}
	provider := NewProvider(dashboard, &fakeClient{})
	if _, err := provider.Statuses(context.Background()); err != nil {
		t.Fatal(err)
	}
	dashboard.servers[0].Address = "127.0.0.1:27016"
	if _, err := provider.Statuses(context.Background()); err != nil {
		t.Fatal(err)
	}
	provider.mu.RLock()
	defer provider.mu.RUnlock()
	if len(provider.entries) != 1 {
		t.Fatalf("A2S cache entries = %d, want 1", len(provider.entries))
	}
	if _, exists := provider.entries["main\x00127.0.0.1:27015"]; exists {
		t.Fatal("stale A2S cache entry was retained")
	}
}
