package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

type fakeIngameDashboard struct {
	settings        store.IngameSettings
	servers         []store.GameServer
	visibility      store.PlayerProfileVisibility
	overrides       []store.IngameServerSettings
	documents       []store.ServerDocument
	quickLinks      []store.IngameQuickLink
	mapNames        []store.IngameMapName
	lastDocumentKey string
	settingsCalls   int
	overrideCalls   int
	quickLinkCalls  int
	mapNameCalls    int
	visibilityCalls int
}

func (f *fakeIngameDashboard) IngameSettings(context.Context) (store.IngameSettings, error) {
	f.settingsCalls++
	return f.settings, nil
}
func (f *fakeIngameDashboard) IngameServerSettings(context.Context, string) (store.IngameServerSettings, error) {
	return store.IngameServerSettings{TitleMode: "inherit", DescriptionMode: "inherit", BannerMode: "inherit", BackgroundMode: "inherit", WebsiteMode: "inherit", HighlightMode: "inherit"}, nil
}
func (f *fakeIngameDashboard) ListIngameServerSettings(context.Context) ([]store.IngameServerSettings, error) {
	f.overrideCalls++
	return append([]store.IngameServerSettings(nil), f.overrides...), nil
}
func (f *fakeIngameDashboard) ListServers(context.Context) ([]store.GameServer, error) {
	return append([]store.GameServer(nil), f.servers...), nil
}
func (f *fakeIngameDashboard) ListSiteDocuments(context.Context, bool) ([]store.SiteDocument, error) {
	return []store.SiteDocument{}, nil
}
func (f *fakeIngameDashboard) GetSiteDocument(context.Context, string, bool) (store.SiteDocument, error) {
	return store.SiteDocument{}, nil
}
func (f *fakeIngameDashboard) ListServerDocuments(context.Context, string) ([]store.ServerDocument, error) {
	return append([]store.ServerDocument(nil), f.documents...), nil
}
func (f *fakeIngameDashboard) GetServerDocument(_ context.Context, serverKey, documentKey string) (store.ServerDocument, error) {
	f.lastDocumentKey = serverKey
	for _, document := range f.documents {
		if document.ServerKey == serverKey && document.Key == documentKey {
			return document, nil
		}
	}
	return store.ServerDocument{ServerKey: serverKey, Key: documentKey, Mode: "inherit"}, nil
}
func (f *fakeIngameDashboard) ListServerQuickLinks(_ context.Context, serverKey string) ([]store.IngameQuickLink, error) {
	f.quickLinkCalls++
	result := make([]store.IngameQuickLink, 0, len(f.quickLinks))
	for _, link := range f.quickLinks {
		if link.ServerKey == serverKey {
			result = append(result, link)
		}
	}
	return result, nil
}
func (f *fakeIngameDashboard) ListIngameMapNames(context.Context) ([]store.IngameMapName, error) {
	f.mapNameCalls++
	return append([]store.IngameMapName(nil), f.mapNames...), nil
}
func (f *fakeIngameDashboard) PlayerProfileVisibility(context.Context, string) (store.PlayerProfileVisibility, error) {
	f.visibilityCalls++
	return f.visibility, nil
}
func (f *fakeIngameDashboard) ListAnnouncements(context.Context, store.AnnouncementFilter) (store.AnnouncementPage, error) {
	return store.AnnouncementPage{Items: []store.Announcement{}}, nil
}
func (f *fakeIngameDashboard) GetAnnouncement(context.Context, string) (store.Announcement, error) {
	return store.Announcement{}, nil
}
func (f *fakeIngameDashboard) SiteSettings(context.Context) (store.SiteSettings, error) {
	return store.SiteSettings{A2SRefreshSeconds: 30, PublicOrigin: "https://stats.example.com"}, nil
}

type fakeIngameStatuses struct {
	statuses []store.ServerStatus
	calls    int
}

func (f *fakeIngameStatuses) CachedStatuses(context.Context) ([]store.ServerStatus, error) {
	f.calls++
	return append([]store.ServerStatus(nil), f.statuses...), nil
}

type fakeIngamePlayers struct {
	summaryCalls, pveCalls, versusCalls, relationshipCalls int
}

func (f *fakeIngamePlayers) Summary(context.Context, string) (*store.PlayerSummary, error) {
	f.summaryCalls++
	return &store.PlayerSummary{SteamID: "76561198000000001", LastName: "Player", ActiveSeconds: 3600}, nil
}
func (f *fakeIngamePlayers) PVE(context.Context, string, int64) (store.PlayerPVE, error) {
	f.pveCalls++
	return store.PlayerPVE{CommonKills: 10}, nil
}
func (f *fakeIngamePlayers) Versus(context.Context, string, int64) (store.PlayerVersus, error) {
	f.versusCalls++
	return store.PlayerVersus{DamageToHumanSurvivors: 10}, nil
}
func (f *fakeIngamePlayers) Relationships(context.Context, string, store.PlayerRelationshipQuery) (store.PlayerRelationshipPage, error) {
	f.relationshipCalls++
	return store.PlayerRelationshipPage{}, nil
}

type fakeIngameAchievements struct{ calls int }

func (f *fakeIngameAchievements) Compact(context.Context, string) (CompactAchievementOverview, error) {
	f.calls++
	return CompactAchievementOverview{Unlocked: 1, Total: 10}, nil
}

type fakeIngameRankings struct {
	highlightCalls int
	recentCalls    int
	ids            []string
	metrics        [3]string
	lastQuery      store.RankingQuery
	recentErr      error
	recentServer   string
	recentCutoff   time.Time
}

func (f *fakeIngameRankings) IngameRecent24h(_ context.Context, serverKey string, cutoff time.Time) (store.ServerRecent24h, error) {
	f.recentCalls++
	f.recentServer, f.recentCutoff = serverKey, cutoff
	return store.ServerRecent24h{ActivePlayers: 4, CommonKills: 100, SpecialKills: 20, CompletedRuns: 2}, f.recentErr
}

func (f *fakeIngameRankings) List(_ context.Context, query store.RankingQuery) (store.RankingPage, error) {
	f.lastQuery = query
	return store.RankingPage{Items: []store.RankingEntry{}}, nil
}
func (f *fakeIngameRankings) IngameHighlights(_ context.Context, _ string, ids []string, metrics [3]string) ([]IngameHighlight, error) {
	f.highlightCalls++
	f.ids = append([]string(nil), ids...)
	f.metrics = metrics
	result := make([]IngameHighlight, 0, 3)
	for _, key := range metrics {
		metric, _ := IngameMetric(key)
		result = append(result, IngameHighlight{Metric: metric, SteamID: ids[0], Value: 10})
	}
	return result, nil
}

func defaultIngameTestSettings() store.IngameSettings {
	return store.IngameSettings{
		Enabled: true, ShowPlayers: true, ShowHighlights: true, ShowAnnouncements: true,
		HighlightMetrics: [3]string{"active_play_seconds", "special_kills", "rescues"},
		HomeCacheSeconds: 30, PlayerCacheSeconds: 60, RankingCacheSeconds: 120, ContentCacheSeconds: 300,
	}
}

func TestIngameHomeUsesBoundedCachedView(t *testing.T) {
	now := time.Now()
	dashboard := &fakeIngameDashboard{settings: defaultIngameTestSettings(), servers: []store.GameServer{{ID: "server", DisplayName: "Main", Enabled: true}}}
	status := store.ServerStatus{ServerID: "server", ServerKey: "main", Online: true, LastSuccessAt: now, Bots: 3}
	for index := 0; index < 35; index++ {
		status.PlayerList = append(status.PlayerList, store.ServerPlayer{
			Name: fmt.Sprintf("Player %02d", index), SteamID: fmt.Sprintf("76561198%09d", index), DurationSeconds: int64(1000 - index),
		})
	}
	statuses := &fakeIngameStatuses{statuses: []store.ServerStatus{status}}
	players := &fakeIngamePlayers{}
	rankings := &fakeIngameRankings{}
	service := NewIngameService(dashboard, statuses, players, rankings, &fakeIngameAchievements{})
	service.now = func() time.Time { return now }
	first, err := service.Home(context.Background(), "main")
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Home(context.Background(), "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Players) != 32 || len(first.Highlights) != 3 || second.ServerKey != "main" {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
	if len(rankings.ids) != 32 || rankings.metrics != dashboard.settings.HighlightMetrics {
		t.Fatalf("bounded highlight ids=%d metrics=%v", len(rankings.ids), rankings.metrics)
	}
	if dashboard.settingsCalls != 1 || dashboard.overrideCalls != 1 || dashboard.quickLinkCalls != 1 || dashboard.mapNameCalls != 1 || statuses.calls != 1 || rankings.highlightCalls != 1 || rankings.recentCalls != 1 {
		t.Fatalf("cache miss counts settings=%d overrides=%d quick_links=%d maps=%d status=%d highlights=%d recent=%d", dashboard.settingsCalls, dashboard.overrideCalls, dashboard.quickLinkCalls, dashboard.mapNameCalls, statuses.calls, rankings.highlightCalls, rankings.recentCalls)
	}
	if first.Recent24h == nil || first.Recent24h.ActivePlayers != 4 || rankings.recentServer != "main" || rankings.recentCutoff.Unix() != now.Add(-24*time.Hour).Unix() {
		t.Fatalf("recent view=%+v server=%q cutoff=%v", first.Recent24h, rankings.recentServer, rankings.recentCutoff)
	}
}

func TestIngameHomeOmitsRecent24hWhenStatsFail(t *testing.T) {
	now := time.Now()
	dashboard := &fakeIngameDashboard{settings: defaultIngameTestSettings(), servers: []store.GameServer{{ID: "server", DisplayName: "Main", Enabled: true}}}
	statuses := &fakeIngameStatuses{statuses: []store.ServerStatus{{ServerID: "server", ServerKey: "main", LastSuccessAt: now}}}
	rankings := &fakeIngameRankings{recentErr: errors.New("stats unavailable")}
	service := NewIngameService(dashboard, statuses, &fakeIngamePlayers{}, rankings, &fakeIngameAchievements{})
	service.now = func() time.Time { return now }
	view, err := service.Home(context.Background(), "main")
	if err != nil || view.Recent24h != nil || rankings.recentCalls != 1 {
		t.Fatalf("home=%+v recent calls=%d err=%v", view, rankings.recentCalls, err)
	}
}

func TestIngamePlayerChecksAnonymousVisibilityBeforeQueries(t *testing.T) {
	dashboard := &fakeIngameDashboard{
		settings: defaultIngameTestSettings(), servers: []store.GameServer{{ID: "server", DisplayName: "Main", Enabled: true}},
		visibility: store.PlayerProfileVisibility{VisibleSections: []store.PlayerProfileSection{store.PlayerProfileAchievements}},
	}
	statuses := &fakeIngameStatuses{statuses: []store.ServerStatus{{ServerID: "server", ServerKey: "main", Online: true, LastSuccessAt: time.Now(), PlayerList: []store.ServerPlayer{{Name: "Player", SteamID: "76561198000000001", DurationSeconds: 60}}}}}
	players := &fakeIngamePlayers{}
	achievements := &fakeIngameAchievements{}
	service := NewIngameService(dashboard, statuses, players, &fakeIngameRankings{}, achievements)
	view, err := service.Player(context.Background(), "main", "76561198000000001")
	if err != nil {
		t.Fatal(err)
	}
	if view.Achievements == nil || view.CurrentPlay != nil || players.summaryCalls != 0 || players.pveCalls != 0 || players.relationshipCalls != 0 || players.versusCalls != 0 {
		t.Fatalf("visibility leaked queries view=%+v players=%+v", view, players)
	}
	if achievements.calls != 1 {
		t.Fatalf("achievement calls=%d", achievements.calls)
	}
	if _, err := service.Player(context.Background(), "main", "76561198000000001"); err != nil {
		t.Fatal(err)
	}
	if dashboard.visibilityCalls != 1 {
		t.Fatalf("player cache did not hit: visibility calls=%d", dashboard.visibilityCalls)
	}
	service.InvalidatePlayer("76561198000000001")
	if _, err := service.Player(context.Background(), "main", "76561198000000001"); err != nil {
		t.Fatal(err)
	}
	if dashboard.visibilityCalls != 2 {
		t.Fatalf("privacy invalidation did not rebuild: visibility calls=%d", dashboard.visibilityCalls)
	}
}

func TestIngamePlayerShowsExactFreshCurrentPlayOnlyForPublicOverview(t *testing.T) {
	now := time.Now()
	steamID := "76561198000000001"
	dashboard := &fakeIngameDashboard{
		settings: defaultIngameTestSettings(), servers: []store.GameServer{{ID: "server", DisplayName: "官图 #1", Enabled: true}},
		visibility: store.PlayerProfileVisibility{VisibleSections: []store.PlayerProfileSection{store.PlayerProfileOverview}},
	}
	statuses := &fakeIngameStatuses{statuses: []store.ServerStatus{{
		ServerID: "server", ServerKey: "main", Online: true, Map: "c5m1_waterfront", LastSuccessAt: now,
		PlayerList: []store.ServerPlayer{{Name: "Player", SteamID: steamID, DurationSeconds: 1080}},
	}}}
	service := NewIngameService(dashboard, statuses, &fakeIngamePlayers{}, &fakeIngameRankings{}, &fakeIngameAchievements{})
	service.now = func() time.Time { return now }
	view, err := service.Player(context.Background(), "main", steamID)
	if err != nil || view.CurrentPlay == nil || view.CurrentPlay.InstanceName != "官图 #1" || view.CurrentPlay.MapName != "教区 1/5" || view.CurrentPlay.DurationSeconds != 1080 {
		t.Fatalf("current play=%+v err=%v", view.CurrentPlay, err)
	}

	statuses.statuses[0].LastSuccessAt = now.Add(-2 * time.Minute)
	staleService := NewIngameService(dashboard, statuses, &fakeIngamePlayers{}, &fakeIngameRankings{}, &fakeIngameAchievements{})
	staleService.now = func() time.Time { return now }
	stale, err := staleService.Player(context.Background(), "main", steamID)
	if err != nil || stale.CurrentPlay != nil {
		t.Fatalf("stale current play=%+v err=%v", stale.CurrentPlay, err)
	}

	statuses.statuses[0].LastSuccessAt = now
	statuses.statuses[0].PlayerList[0].SteamID = "76561198000000002"
	otherService := NewIngameService(dashboard, statuses, &fakeIngamePlayers{}, &fakeIngameRankings{}, &fakeIngameAchievements{})
	otherService.now = func() time.Time { return now }
	other, err := otherService.Player(context.Background(), "main", steamID)
	if err != nil || other.CurrentPlay != nil {
		t.Fatalf("non-matching current play=%+v err=%v", other.CurrentPlay, err)
	}
}

func TestIngameHomeSelectsAmongResolvedServers(t *testing.T) {
	settings := defaultIngameTestSettings()
	settings.BackgroundURL = "https://example.com/background.jpg"
	dashboard := &fakeIngameDashboard{settings: settings, servers: []store.GameServer{
		{ID: "one", DisplayName: "One", Enabled: true}, {ID: "two", DisplayName: "Two", Enabled: true},
	}, overrides: []store.IngameServerSettings{{ServerKey: "one", TitleMode: "inherit", DescriptionMode: "inherit", BannerMode: "inherit", BackgroundMode: "inherit", WebsiteMode: "inherit", HighlightMode: "inherit"}}}
	statuses := &fakeIngameStatuses{statuses: []store.ServerStatus{{ServerID: "one", ServerKey: "one"}, {ServerID: "two", ServerKey: "two"}}}
	service := NewIngameService(dashboard, statuses, &fakeIngamePlayers{}, &fakeIngameRankings{}, &fakeIngameAchievements{})
	view, err := service.Home(context.Background(), "")
	if err != nil || !view.SelectionOnly || len(view.ServerOptions) != 2 || view.ActivePage != "home" || view.Config.Appearance.BackgroundURL != settings.BackgroundURL {
		t.Fatalf("selection view=%+v err=%v", view, err)
	}
	if len(view.ServerOptions[0].Instances) != 1 || view.ServerOptions[0].Instances[0].DisplayName != "One" {
		t.Fatalf("selection option=%+v", view.ServerOptions[0])
	}
}

func TestIngameHomeAggregatesInstancesInOneServerGroup(t *testing.T) {
	now := time.Now()
	dashboard := &fakeIngameDashboard{
		settings: defaultIngameTestSettings(),
		quickLinks: []store.IngameQuickLink{
			{ServerKey: "shared", Label: "地图合集", URL: "https://example.com/maps", SortOrder: 0, Enabled: true},
			{ServerKey: "shared", Label: "隐藏链接", URL: "https://example.com/hidden", SortOrder: 1, Enabled: false},
		},
		servers: []store.GameServer{
			{ID: "one", DisplayName: "Group #1", Address: "127.0.0.1:27015", Enabled: true, SortOrder: 1},
			{ID: "two", DisplayName: "Group #2", Address: "127.0.0.1:27016", Enabled: true, SortOrder: 2},
			{ID: "three", DisplayName: "Group #3", Address: "127.0.0.1:27017", Enabled: true, SortOrder: 3},
			{ID: "other", DisplayName: "Other", Address: "127.0.0.1:27018", Enabled: true, SortOrder: 4},
		},
		overrides: []store.IngameServerSettings{{ServerKey: "shared", TitleMode: "override", Title: "Shared Group", DescriptionMode: "inherit", BannerMode: "inherit", BackgroundMode: "inherit", WebsiteMode: "inherit", HighlightMode: "inherit"}},
		mapNames:  []store.IngameMapName{{MapName: "c1m1_hotel", DisplayName: "自定义第一章"}},
	}
	statuses := &fakeIngameStatuses{statuses: []store.ServerStatus{
		{ServerID: "one", ServerKey: "shared", Online: true, Map: "c1m1_hotel", Players: 3, Bots: 1, LatencyMS: 24, LastSuccessAt: now, Rules: []store.ServerRule{{Name: "mp_gamemode", Value: "coop"}, {Name: "z_difficulty", Value: "Hard"}}, PlayerList: []store.ServerPlayer{{Name: "Alice", SteamID: "76561198000000001"}}},
		{ServerID: "two", ServerKey: "shared", Online: true, Players: 1, LastSuccessAt: now, PlayerList: []store.ServerPlayer{{Name: "Bob", SteamID: "76561198000000002"}}},
		{ServerID: "three", ServerKey: "shared", Online: false, LastSuccessAt: now.Add(-time.Hour)},
		{ServerID: "other", ServerKey: "other", Online: true, Players: 8, LastSuccessAt: now},
	}}
	service := NewIngameService(dashboard, statuses, &fakeIngamePlayers{}, &fakeIngameRankings{}, &fakeIngameAchievements{})
	service.now = func() time.Time { return now }
	view, err := service.Home(context.Background(), "shared")
	if err != nil {
		t.Fatal(err)
	}
	if view.Config.Appearance.Title != "Shared Group" || view.OnlineInstances != 2 || view.TotalInstances != 3 || view.OnlinePlayerCount != 2 || view.BotCount != 1 || len(view.Players) != 2 || len(view.Instances) != 3 {
		t.Fatalf("group view=%+v", view)
	}
	if view.Instances[0].ActionID == "" || view.Instances[2].ActionID != "" || view.Instances[0].Map != "自定义第一章" || view.Instances[0].GameMode != "coop" || view.Instances[0].Difficulty != "Hard" || view.Instances[0].LatencyMS != 24 {
		t.Fatalf("instance actions/details=%+v", view.Instances)
	}
	if view.Players[0].InstanceName != "Group #1" || len(view.QuickLinks) != 1 || view.QuickLinks[0].Label != "地图合集" {
		t.Fatalf("player sources/quick links=%+v/%+v", view.Players, view.QuickLinks)
	}
	actionText := ""
	for _, action := range view.Actions {
		actionText += action.Value + "\n"
	}
	if !strings.Contains(actionText, "https://example.com/maps") || !strings.Contains(actionText, "connect 127.0.0.1:27015") || strings.Contains(actionText, "steam://") {
		t.Fatalf("action cards=%+v", view.Actions)
	}
	if dashboard.settingsCalls != 1 || dashboard.overrideCalls != 1 || statuses.calls != 1 {
		t.Fatalf("group build calls settings=%d overrides=%d statuses=%d", dashboard.settingsCalls, dashboard.overrideCalls, statuses.calls)
	}
}

func TestIngameSelectionDeduplicatesSharedServerKey(t *testing.T) {
	dashboard := &fakeIngameDashboard{settings: defaultIngameTestSettings(), servers: []store.GameServer{
		{ID: "one", DisplayName: "Shared #1", Address: "127.0.0.1:27015", Enabled: true},
		{ID: "two", DisplayName: "Shared #2", Address: "127.0.0.1:27016", Enabled: true},
		{ID: "other", DisplayName: "Other", Address: "127.0.0.1:27017", Enabled: true},
	}}
	statuses := &fakeIngameStatuses{statuses: []store.ServerStatus{
		{ServerID: "one", ServerKey: "shared"}, {ServerID: "two", ServerKey: "shared"}, {ServerID: "other", ServerKey: "other"},
	}}
	service := NewIngameService(dashboard, statuses, &fakeIngamePlayers{}, &fakeIngameRankings{}, &fakeIngameAchievements{})
	view, err := service.Home(context.Background(), "")
	sharedInstances := 0
	for _, option := range view.ServerOptions {
		if option.ServerKey == "shared" {
			sharedInstances = len(option.Instances)
		}
	}
	if err != nil || !view.SelectionOnly || len(view.ServerOptions) != 2 || sharedInstances != 2 {
		t.Fatalf("selection=%+v err=%v", view, err)
	}
}

func TestIngameOfflineSnapshotStillResolvesGroup(t *testing.T) {
	dashboard := &fakeIngameDashboard{settings: defaultIngameTestSettings(), servers: []store.GameServer{{ID: "one", DisplayName: "Offline", Address: "127.0.0.1:27015", Enabled: true}}}
	statuses := &fakeIngameStatuses{statuses: []store.ServerStatus{{ServerID: "one", ServerKey: "offline-group", Online: false}}}
	service := NewIngameService(dashboard, statuses, &fakeIngamePlayers{}, &fakeIngameRankings{}, &fakeIngameAchievements{})
	view, err := service.Home(context.Background(), "offline-group")
	if err != nil || view.ServerKey != "offline-group" || view.TotalInstances != 1 || view.OnlineInstances != 0 {
		t.Fatalf("offline group=%+v err=%v", view, err)
	}
}

func TestBuildIngameConnectAddressValidatesAddress(t *testing.T) {
	for input, expected := range map[string]string{
		"example.com:27015":   "example.com:27015",
		"[2001:db8::1]:27015": "[2001:db8::1]:27015",
		"bad host:27015":      "",
		"example.com:70000":   "",
		"evil.com:27015/path": "",
	} {
		if actual := BuildIngameConnectAddress(input); actual != expected {
			t.Errorf("BuildIngameConnectAddress(%q)=%q, want %q", input, actual, expected)
		}
	}
}

func TestIngameRankingsAndDocumentsRemainGroupScoped(t *testing.T) {
	dashboard := &fakeIngameDashboard{
		settings: defaultIngameTestSettings(),
		servers: []store.GameServer{
			{ID: "one", DisplayName: "One", Address: "127.0.0.1:27015", Enabled: true},
			{ID: "two", DisplayName: "Two", Address: "127.0.0.1:27016", Enabled: true},
		},
		documents: []store.ServerDocument{{ServerKey: "shared", Key: store.IngameDocumentCommands, Mode: "override", ContentMarkdown: "shared commands"}},
	}
	statuses := &fakeIngameStatuses{statuses: []store.ServerStatus{{ServerID: "one", ServerKey: "shared"}, {ServerID: "two", ServerKey: "shared"}}}
	rankings := &fakeIngameRankings{}
	service := NewIngameService(dashboard, statuses, &fakeIngamePlayers{}, rankings, &fakeIngameAchievements{})
	if _, err := service.Rankings(context.Background(), "shared", "common_kills", 1); err != nil {
		t.Fatal(err)
	}
	if rankings.lastQuery.ServerKey != "shared" || rankings.lastQuery.Limit != 10 {
		t.Fatalf("ranking query=%+v", rankings.lastQuery)
	}
	view, err := service.Info(context.Background(), "shared", store.IngameDocumentCommands)
	if err != nil || view.ContentMarkdown != "shared commands" || dashboard.lastDocumentKey != "shared" {
		t.Fatalf("document view=%+v key=%q err=%v", view, dashboard.lastDocumentKey, err)
	}
}

func TestIngameErrorBackgroundRejectsUnsafeStoredValue(t *testing.T) {
	dashboard := &fakeIngameDashboard{settings: defaultIngameTestSettings()}
	dashboard.settings.BackgroundURL = "https://example.com/background.jpg?ver=2"
	service := NewIngameService(dashboard, &fakeIngameStatuses{}, nil, nil, nil)
	if background := service.ErrorBackground(context.Background()); background != dashboard.settings.BackgroundURL {
		t.Fatalf("safe error background=%q", background)
	}
	dashboard.settings.BackgroundURL = "javascript:alert(1)"
	if background := service.ErrorBackground(context.Background()); background != "" {
		t.Fatalf("unsafe error background=%q", background)
	}
}

type summaryCountingStats struct {
	store.StatsStore
	calls int
}

func (s *summaryCountingStats) PlayerSummary(context.Context, string) (*store.PlayerSummary, error) {
	s.calls++
	return &store.PlayerSummary{LastName: "Name"}, nil
}

func TestRankingSkipPlayerNamesAvoidsPerPlayerQueries(t *testing.T) {
	stats := &summaryCountingStats{}
	service := &RankingService{stats: stats}
	entries := []store.RankingEntry{{SteamID: "1"}, {SteamID: "2"}}
	service.finishRanking(context.Background(), store.RankingQuery{Mode: "pve", Metric: "common_kills", Limit: 2, SkipPlayerNames: true}, entries)
	if stats.calls != 0 {
		t.Fatalf("skip names made %d player queries", stats.calls)
	}
	service.finishRanking(context.Background(), store.RankingQuery{Mode: "pve", Metric: "common_kills", Limit: 2}, entries)
	if stats.calls != 2 {
		t.Fatalf("normal ranking name queries=%d", stats.calls)
	}
}
