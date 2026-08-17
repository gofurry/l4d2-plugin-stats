package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

type fakeIngameDashboard struct {
	settings        store.IngameSettings
	servers         []store.GameServer
	visibility      store.PlayerProfileVisibility
	settingsCalls   int
	visibilityCalls int
}

func (f *fakeIngameDashboard) IngameSettings(context.Context) (store.IngameSettings, error) {
	f.settingsCalls++
	return f.settings, nil
}
func (f *fakeIngameDashboard) IngameServerSettings(context.Context, string) (store.IngameServerSettings, error) {
	return store.IngameServerSettings{TitleMode: "inherit", DescriptionMode: "inherit", BannerMode: "inherit", BackgroundMode: "inherit", WebsiteMode: "inherit", HighlightMode: "inherit"}, nil
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
	return []store.ServerDocument{}, nil
}
func (f *fakeIngameDashboard) GetServerDocument(context.Context, string, string) (store.ServerDocument, error) {
	return store.ServerDocument{Mode: "inherit"}, nil
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
	return store.SiteSettings{A2SRefreshSeconds: 30}, nil
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
	ids            []string
	metrics        [3]string
}

func (f *fakeIngameRankings) List(context.Context, store.RankingQuery) (store.RankingPage, error) {
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
	if dashboard.settingsCalls != 1 || statuses.calls != 1 || rankings.highlightCalls != 1 {
		t.Fatalf("cache miss counts settings=%d status=%d highlights=%d", dashboard.settingsCalls, statuses.calls, rankings.highlightCalls)
	}
}

func TestIngamePlayerChecksAnonymousVisibilityBeforeQueries(t *testing.T) {
	dashboard := &fakeIngameDashboard{
		settings: defaultIngameTestSettings(), servers: []store.GameServer{{ID: "server", DisplayName: "Main", Enabled: true}},
		visibility: store.PlayerProfileVisibility{VisibleSections: []store.PlayerProfileSection{store.PlayerProfileAchievements}},
	}
	statuses := &fakeIngameStatuses{statuses: []store.ServerStatus{{ServerID: "server", ServerKey: "main", Online: true, LastSuccessAt: time.Now()}}}
	players := &fakeIngamePlayers{}
	achievements := &fakeIngameAchievements{}
	service := NewIngameService(dashboard, statuses, players, &fakeIngameRankings{}, achievements)
	view, err := service.Player(context.Background(), "main", "76561198000000001")
	if err != nil {
		t.Fatal(err)
	}
	if view.Achievements == nil || players.summaryCalls != 0 || players.pveCalls != 0 || players.relationshipCalls != 0 || players.versusCalls != 0 {
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

func TestIngameHomeSelectsAmongResolvedServers(t *testing.T) {
	dashboard := &fakeIngameDashboard{settings: defaultIngameTestSettings(), servers: []store.GameServer{
		{ID: "one", DisplayName: "One", Enabled: true}, {ID: "two", DisplayName: "Two", Enabled: true},
	}}
	statuses := &fakeIngameStatuses{statuses: []store.ServerStatus{{ServerID: "one", ServerKey: "one"}, {ServerID: "two", ServerKey: "two"}}}
	service := NewIngameService(dashboard, statuses, &fakeIngamePlayers{}, &fakeIngameRankings{}, &fakeIngameAchievements{})
	view, err := service.Home(context.Background(), "")
	if err != nil || !view.SelectionOnly || len(view.ServerOptions) != 2 {
		t.Fatalf("selection view=%+v err=%v", view, err)
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
