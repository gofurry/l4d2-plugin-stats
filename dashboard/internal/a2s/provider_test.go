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

type snapshotDashboard struct {
	*fakeDashboard
	mu        sync.Mutex
	snapshots []store.ServerStatus
}

func (f *snapshotDashboard) ListServerStatusSnapshots(context.Context) ([]store.ServerStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]store.ServerStatus(nil), f.snapshots...), nil
}

func (f *snapshotDashboard) UpsertServerStatusSnapshot(_ context.Context, status store.ServerStatus) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.snapshots {
		if f.snapshots[i].ServerID == status.ServerID {
			f.snapshots[i] = status
			return nil
		}
	}
	f.snapshots = append(f.snapshots, status)
	return nil
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
	mu      sync.Mutex
	calls   int
	fail    bool
	players []Player
	rules   []Rule
}

type blockingClient struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
	mu      sync.Mutex
	calls   int
}

type instanceClient struct {
	mu      sync.Mutex
	players map[string][]Player
	rules   []Rule
}

func (f *instanceClient) Query(_ context.Context, address string, _ time.Duration) (Info, []Player, []Rule, time.Duration, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	players := append([]Player(nil), f.players[address]...)
	return Info{Name: address, Map: "c1m1_hotel", Players: len(players), MaxPlayers: 8}, players, append([]Rule(nil), f.rules...), time.Millisecond, nil
}

func (f *blockingClient) Query(ctx context.Context, _ string, _ time.Duration) (Info, []Player, []Rule, time.Duration, error) {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	f.once.Do(func() { close(f.started) })
	select {
	case <-ctx.Done():
		return Info{}, nil, nil, 0, ctx.Err()
	case <-f.release:
		return Info{Name: "Slow Server", Map: "c2m1_highway", Players: 1, MaxPlayers: 8}, nil, nil, time.Second, nil
	}
}

func (f *blockingClient) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func (f *fakeClient) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func (f *fakeClient) setFail(fail bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fail = fail
}

func (f *fakeClient) Query(context.Context, string, time.Duration) (Info, []Player, []Rule, time.Duration, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.fail {
		return Info{}, nil, nil, 0, errors.New("offline")
	}
	players := f.players
	if players == nil {
		players = []Player{{Name: "Coach", Score: 8, DurationSeconds: 90}}
	}
	rules := f.rules
	if rules == nil {
		rules = []Rule{{Name: "sv_tags", Value: "coop"}}
	}
	return Info{Name: "Test Server", Map: "c1m1_hotel", Players: 3, MaxPlayers: 8, Bots: 1}, players, rules, 12 * time.Millisecond, nil
}

type fakePresence struct {
	mu        sync.Mutex
	players   []store.ActivePlayer
	err       error
	serverKey string
	calls     int
}

func (f *fakePresence) ActivePlayers(_ context.Context, serverKey string, _ int64) ([]store.ActivePlayer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.serverKey = serverKey
	return f.players, f.err
}

func waitForLastStatus(t *testing.T, provider *Provider, serverID string, accept func(store.ServerStatus) bool) store.ServerStatus {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		status, available, err := provider.LastStatus(context.Background(), serverID)
		if err != nil {
			t.Fatal(err)
		}
		if available && accept(status) {
			return status
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for server %q status", serverID)
	return store.ServerStatus{}
}

func TestProviderReturnsColdAndExpiredSnapshotsWithoutBlocking(t *testing.T) {
	dashboard := &fakeDashboard{servers: []store.GameServer{{
		ID: "main", DisplayName: "Configured", Address: "127.0.0.1:27015", Enabled: true,
	}}}
	client := &fakeClient{}
	provider := NewProvider(dashboard, client)
	provider.ttl = time.Hour

	firstList, err := provider.Statuses(context.Background())
	first := firstList[0]
	if err != nil || !first.Checking || first.Online || first.ServerID != "main" {
		t.Fatalf("first status = %#v, %v", first, err)
	}
	warmed := waitForLastStatus(t, provider, "main", func(status store.ServerStatus) bool { return status.Online && !status.Checking })
	if warmed.Name != "Test Server" || client.callCount() != 1 {
		t.Fatalf("warmed status = %#v; calls = %d", warmed, client.callCount())
	}
	secondList, err := provider.Statuses(context.Background())
	second := secondList[0]
	if err != nil || second.Name != warmed.Name || client.callCount() != 1 {
		t.Fatalf("cached status = %#v, %v; calls = %d", second, err, client.callCount())
	}

	provider.ttl = time.Nanosecond
	provider.mu.Lock()
	key := serverCacheKey(dashboard.servers[0])
	entry := provider.entries[key]
	entry.expires = time.Now().Add(-time.Second)
	provider.entries[key] = entry
	provider.mu.Unlock()
	client.setFail(true)
	staleList, err := provider.Statuses(context.Background())
	stale := staleList[0]
	if err != nil || !stale.Online || !stale.Stale || !stale.Checking || stale.Name != "Test Server" {
		t.Fatalf("stale status = %#v, %v", stale, err)
	}
	failed := waitForLastStatus(t, provider, "main", func(status store.ServerStatus) bool { return !status.Checking && !status.Online })
	if !failed.Stale || failed.Name != "Test Server" {
		t.Fatalf("failed refresh status = %#v", failed)
	}
}

func TestProviderStartWarmsEnabledServersWithoutHomepageTraffic(t *testing.T) {
	dashboard := &fakeDashboard{
		servers:  []store.GameServer{{ID: "main", DisplayName: "Main", Address: "127.0.0.1:27015", Enabled: true}},
		settings: store.SiteSettings{A2SRefreshSeconds: 5, A2SJitterSeconds: 5, A2SRetryCount: 1},
	}
	client := &fakeClient{}
	provider := NewProvider(dashboard, client)
	ctx, cancel := context.WithCancel(context.Background())
	provider.Start(ctx)
	warmed := waitForLastStatus(t, provider, "main", func(status store.ServerStatus) bool { return status.Online })
	cancel()
	if warmed.Name != "Test Server" || client.callCount() != 1 {
		t.Fatalf("warmed status = %#v; calls = %d", warmed, client.callCount())
	}
}

func TestProviderColdRequestDoesNotWaitAndSharesBackgroundRefresh(t *testing.T) {
	dashboard := &fakeDashboard{servers: []store.GameServer{{
		ID: "slow", DisplayName: "Slow", Address: "127.0.0.1:27015", Enabled: true,
	}}}
	client := &blockingClient{started: make(chan struct{}), release: make(chan struct{})}
	provider := NewProvider(dashboard, client)

	startedAt := time.Now()
	first, err := provider.Statuses(context.Background())
	if err != nil || len(first) != 1 || !first[0].Checking {
		t.Fatalf("cold Statuses() = %#v, %v", first, err)
	}
	if elapsed := time.Since(startedAt); elapsed > 100*time.Millisecond {
		t.Fatalf("cold Statuses() blocked for %s", elapsed)
	}
	select {
	case <-client.started:
	case <-time.After(time.Second):
		t.Fatal("background refresh did not start")
	}
	second, err := provider.Statuses(context.Background())
	if err != nil || !second[0].Checking || client.callCount() != 1 {
		t.Fatalf("second Statuses() = %#v, %v; calls = %d", second, err, client.callCount())
	}

	close(client.release)
	warmed := waitForLastStatus(t, provider, "slow", func(status store.ServerStatus) bool { return status.Online })
	if warmed.Name != "Slow Server" || client.callCount() != 1 {
		t.Fatalf("warmed status = %#v; calls = %d", warmed, client.callCount())
	}
}

func TestProviderReturnsPersistedSnapshotWhileRefreshingAfterRestart(t *testing.T) {
	server := store.GameServer{ID: "main", DisplayName: "Main", Address: "127.0.0.1:27015", Enabled: true}
	dashboard := &snapshotDashboard{
		fakeDashboard: &fakeDashboard{servers: []store.GameServer{server}},
		snapshots: []store.ServerStatus{{
			ServerID: server.ID, DisplayName: server.DisplayName, Address: server.Address,
			Online: true, Name: "Persisted Server", Map: "c1m1_hotel",
			LastSuccessAt: time.Now().Add(-time.Minute), CheckedAt: time.Now().Add(-time.Minute),
		}},
	}
	client := &blockingClient{started: make(chan struct{}), release: make(chan struct{})}
	provider := NewProvider(dashboard, client)

	statuses, err := provider.Statuses(context.Background())
	if err != nil || len(statuses) != 1 {
		t.Fatalf("Statuses() = %#v, %v", statuses, err)
	}
	status := statuses[0]
	if status.Name != "Persisted Server" || !status.Online || !status.Stale || !status.Checking {
		t.Fatalf("persisted status = %#v", status)
	}
	close(client.release)
	waitForLastStatus(t, provider, "main", func(status store.ServerStatus) bool { return status.Name == "Slow Server" })
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
	if statuses[0].ServerID != "first" || statuses[1].ServerID != "second" || !statuses[0].Checking || !statuses[1].Checking {
		t.Fatalf("ordered statuses = %#v", statuses)
	}
}

func TestProviderRetriesFailuresUsingConfiguredCount(t *testing.T) {
	dashboard := &fakeDashboard{
		servers:  []store.GameServer{{ID: "main", DisplayName: "Main", Address: "127.0.0.1:27015", Enabled: true}},
		settings: store.SiteSettings{A2SRefreshSeconds: 5, A2SJitterSeconds: 0, A2SRetryCount: 3},
	}
	client := &fakeClient{fail: true}
	provider := NewProvider(dashboard, client)
	status, err := provider.RefreshStatus(context.Background(), "main")
	if err != nil || status.Online || client.callCount() != 4 {
		t.Fatalf("RefreshStatus()=%#v err=%v calls=%d", status, err, client.callCount())
	}
}

func TestProviderPurgesRemovedAndChangedServerCacheEntries(t *testing.T) {
	dashboard := &fakeDashboard{servers: []store.GameServer{{
		ID: "main", DisplayName: "Main", Address: "127.0.0.1:27015", Enabled: true,
	}}}
	provider := NewProvider(dashboard, &fakeClient{})
	if _, err := provider.RefreshStatus(context.Background(), "main"); err != nil {
		t.Fatal(err)
	}
	dashboard.servers[0].Address = "127.0.0.1:27016"
	if _, err := provider.Statuses(context.Background()); err != nil {
		t.Fatal(err)
	}
	waitForLastStatus(t, provider, "main", func(status store.ServerStatus) bool { return status.Address == "127.0.0.1:27016" })
	provider.mu.RLock()
	defer provider.mu.RUnlock()
	if len(provider.entries) != 1 {
		t.Fatalf("A2S cache entries = %d, want 1", len(provider.entries))
	}
	if _, exists := provider.entries["main\x00127.0.0.1:27015"]; exists {
		t.Fatal("stale A2S cache entry was retained")
	}
}

func TestProviderLinksA2SPlayersThroughPublishedServerKey(t *testing.T) {
	dashboard := &fakeDashboard{servers: []store.GameServer{{ID: "main", DisplayName: "Main", Address: "127.0.0.1:27015", Enabled: true}}}
	client := &fakeClient{
		players: []Player{{Name: "Alice", Score: 42, DurationSeconds: 180}},
		rules:   []Rule{{Name: "sm_lps_server_key", Value: "community.one"}},
	}
	presence := &fakePresence{players: []store.ActivePlayer{{SteamID: "76561198000000001", Name: "Alice", LastSavedAt: time.Now().Unix(), ConnectedSeconds: 170}}}
	status, err := NewProvider(dashboard, client, presence).RefreshStatus(context.Background(), "main")
	if err != nil {
		t.Fatalf("RefreshStatus() = %#v, %v", status, err)
	}
	if status.ServerKey != "community.one" || presence.serverKey != "community.one" || len(status.PlayerList) != 1 {
		t.Fatalf("linked status = %#v", status)
	}
	if status.PlayerList[0].SteamID != "76561198000000001" || status.PlayerList[0].Score != 42 || status.PlayerList[0].DurationSeconds != 180 {
		t.Fatalf("linked player = %#v", status.PlayerList[0])
	}
}

func TestProviderFallsBackToAnonymousA2SPlayersWhenPresenceFails(t *testing.T) {
	dashboard := &fakeDashboard{servers: []store.GameServer{{ID: "main", DisplayName: "Main", Address: "127.0.0.1:27015", Enabled: true}}}
	client := &fakeClient{rules: []Rule{{Name: "sm_lps_server_key", Value: "community.one"}}}
	presence := &fakePresence{err: errors.New("stats unavailable")}
	status, err := NewProvider(dashboard, client, presence).RefreshStatus(context.Background(), "main")
	if err != nil || len(status.PlayerList) != 1 || status.PlayerList[0].SteamID != "" {
		t.Fatalf("fallback status = %#v, %v", status, err)
	}
}

func TestProviderDoesNotUseStatsPlayersWhenA2SPlayerListIsUnavailable(t *testing.T) {
	dashboard := &fakeDashboard{servers: []store.GameServer{{ID: "main", DisplayName: "Main", Address: "127.0.0.1:27015", Enabled: true}}}
	client := &fakeClient{players: []Player{}, rules: []Rule{{Name: "sm_lps_server_key", Value: "community.one"}}}
	presence := &fakePresence{players: []store.ActivePlayer{{SteamID: "76561198000000001", Name: "Alice"}}}
	status, err := NewProvider(dashboard, client, presence).RefreshStatus(context.Background(), "main")
	if err != nil {
		t.Fatal(err)
	}
	presence.mu.Lock()
	calls := presence.calls
	presence.mu.Unlock()
	if len(status.PlayerList) != 0 || calls != 0 {
		t.Fatalf("A2S-empty players=%#v presence calls=%d", status.PlayerList, calls)
	}
}

func TestProviderDoesNotSynthesizeGroupPlayersIntoAnInstance(t *testing.T) {
	dashboard := &fakeDashboard{servers: []store.GameServer{{ID: "main", DisplayName: "Main", Address: "127.0.0.1:27015", Enabled: true}}}
	client := &fakeClient{
		players: []Player{{Name: "Alice", Score: 7, DurationSeconds: 30}},
		rules:   []Rule{{Name: "sm_lps_server_key", Value: "community.one"}},
	}
	presence := &fakePresence{players: []store.ActivePlayer{
		{SteamID: "76561198000000001", Name: "Alice"},
		{SteamID: "76561198000000002", Name: "Bob"},
	}}
	status, err := NewProvider(dashboard, client, presence).RefreshStatus(context.Background(), "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(status.PlayerList) != 1 || status.PlayerList[0].Name != "Alice" || status.PlayerList[0].SteamID == "" {
		t.Fatalf("instance players = %#v", status.PlayerList)
	}
}

func TestProviderLeavesAmbiguousNamesAnonymous(t *testing.T) {
	dashboard := &fakeDashboard{servers: []store.GameServer{{ID: "main", DisplayName: "Main", Address: "127.0.0.1:27015", Enabled: true}}}
	client := &fakeClient{
		players: []Player{{Name: "Same"}, {Name: "Same"}},
		rules:   []Rule{{Name: "sm_lps_server_key", Value: "community.one"}},
	}
	presence := &fakePresence{players: []store.ActivePlayer{{SteamID: "76561198000000001", Name: "Same"}}}
	status, err := NewProvider(dashboard, client, presence).RefreshStatus(context.Background(), "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(status.PlayerList) != 2 || status.PlayerList[0].SteamID != "" || status.PlayerList[1].SteamID != "" {
		t.Fatalf("ambiguous players = %#v", status.PlayerList)
	}
}

func TestProviderLeavesAmbiguousStatsPresenceAnonymous(t *testing.T) {
	dashboard := &fakeDashboard{servers: []store.GameServer{{ID: "main", DisplayName: "Main", Address: "127.0.0.1:27015", Enabled: true}}}
	client := &fakeClient{players: []Player{{Name: "Same"}}, rules: []Rule{{Name: "sm_lps_server_key", Value: "community.one"}}}
	presence := &fakePresence{players: []store.ActivePlayer{
		{SteamID: "76561198000000001", Name: "Same"}, {SteamID: "76561198000000002", Name: "Same"},
	}}
	status, err := NewProvider(dashboard, client, presence).RefreshStatus(context.Background(), "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(status.PlayerList) != 1 || status.PlayerList[0].SteamID != "" {
		t.Fatalf("ambiguous Stats players = %#v", status.PlayerList)
	}
}

func TestProviderKeepsA2SPlayerOwnershipAcrossSharedKeyInstances(t *testing.T) {
	dashboard := &fakeDashboard{servers: []store.GameServer{
		{ID: "one", DisplayName: "One", Address: "127.0.0.1:27015", Enabled: true},
		{ID: "two", DisplayName: "Two", Address: "127.0.0.1:27016", Enabled: true},
	}}
	client := &instanceClient{
		players: map[string][]Player{
			"127.0.0.1:27015": {{Name: "Alice", Score: 1}},
			"127.0.0.1:27016": {{Name: "Bob", Score: 2}},
		},
		rules: []Rule{{Name: "sm_lps_server_key", Value: "shared"}},
	}
	presence := &fakePresence{players: []store.ActivePlayer{
		{SteamID: "76561198000000001", Name: "Alice"}, {SteamID: "76561198000000002", Name: "Bob"},
	}}
	provider := NewProvider(dashboard, client, presence)
	provider.refreshEnabled(context.Background(), true)
	one, available, err := provider.LastStatus(context.Background(), "one")
	if err != nil || !available {
		t.Fatalf("one status available=%v err=%v", available, err)
	}
	two, available, err := provider.LastStatus(context.Background(), "two")
	if err != nil || !available {
		t.Fatalf("two status available=%v err=%v", available, err)
	}
	if len(one.PlayerList) != 1 || one.PlayerList[0].Name != "Alice" || one.PlayerList[0].SteamID != "76561198000000001" {
		t.Fatalf("instance one players=%#v", one.PlayerList)
	}
	if len(two.PlayerList) != 1 || two.PlayerList[0].Name != "Bob" || two.PlayerList[0].SteamID != "76561198000000002" {
		t.Fatalf("instance two players=%#v", two.PlayerList)
	}
}

func TestProviderDeduplicatesPresenceQueriesForSharedServerKey(t *testing.T) {
	dashboard := &fakeDashboard{servers: []store.GameServer{
		{ID: "one", DisplayName: "One", Address: "127.0.0.1:27015", Enabled: true},
		{ID: "two", DisplayName: "Two", Address: "127.0.0.1:27016", Enabled: true},
	}}
	client := &fakeClient{players: []Player{{Name: "Alice"}}, rules: []Rule{{Name: "sm_lps_server_key", Value: "community.one"}}}
	presence := &fakePresence{players: []store.ActivePlayer{{SteamID: "76561198000000001", Name: "Alice"}}}
	provider := NewProvider(dashboard, client, presence)
	provider.refreshEnabled(context.Background(), true)
	presence.mu.Lock()
	calls := presence.calls
	presence.mu.Unlock()
	if calls != 1 {
		t.Fatalf("presence queries = %d, want 1", calls)
	}
}

func TestProviderPreservesOnlyGroupIdentityAfterStaleOutage(t *testing.T) {
	dashboard := &fakeDashboard{servers: []store.GameServer{{ID: "main", DisplayName: "Main", Address: "127.0.0.1:27015", Enabled: true}}}
	client := &fakeClient{players: []Player{{Name: "Alice"}}, rules: []Rule{{Name: "sm_lps_server_key", Value: "community.one"}}}
	provider := NewProvider(dashboard, client)
	provider.staleTTL = time.Nanosecond
	if _, err := provider.RefreshStatus(context.Background(), "main"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond)
	client.setFail(true)
	status, err := provider.RefreshStatus(context.Background(), "main")
	if err != nil {
		t.Fatal(err)
	}
	if status.ServerKey != "community.one" || status.Map != "" || status.Players != 0 || len(status.PlayerList) != 0 || status.Online {
		t.Fatalf("offline status = %#v", status)
	}
}
