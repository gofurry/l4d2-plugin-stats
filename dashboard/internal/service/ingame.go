package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

var (
	ErrIngameDisabled           = errors.New("in-game portal is disabled")
	ErrIngameUnknownServer      = errors.New("in-game server could not be resolved")
	ErrIngameContentUnavailable = errors.New("in-game content is unavailable")
	ErrIngamePlayerNotFound     = errors.New("player was not found")
)

type ingameDashboard interface {
	IngameSettings(context.Context) (store.IngameSettings, error)
	IngameServerSettings(context.Context, string) (store.IngameServerSettings, error)
	ListServers(context.Context) ([]store.GameServer, error)
	ListSiteDocuments(context.Context, bool) ([]store.SiteDocument, error)
	GetSiteDocument(context.Context, string, bool) (store.SiteDocument, error)
	ListServerDocuments(context.Context, string) ([]store.ServerDocument, error)
	GetServerDocument(context.Context, string, string) (store.ServerDocument, error)
	PlayerProfileVisibility(context.Context, string) (store.PlayerProfileVisibility, error)
	ListAnnouncements(context.Context, store.AnnouncementFilter) (store.AnnouncementPage, error)
	GetAnnouncement(context.Context, string) (store.Announcement, error)
	SiteSettings(context.Context) (store.SiteSettings, error)
}

type ingameStatusSource interface {
	CachedStatuses(context.Context) ([]store.ServerStatus, error)
}

type ingamePlayerSource interface {
	Summary(context.Context, string) (*store.PlayerSummary, error)
	PVE(context.Context, string, int64) (store.PlayerPVE, error)
	Versus(context.Context, string, int64) (store.PlayerVersus, error)
	Relationships(context.Context, string, store.PlayerRelationshipQuery) (store.PlayerRelationshipPage, error)
}

type ingameAchievementSource interface {
	Compact(context.Context, string) (CompactAchievementOverview, error)
}

type ingameRankingSource interface {
	List(context.Context, store.RankingQuery) (store.RankingPage, error)
	IngameHighlights(context.Context, string, []string, [3]string) ([]IngameHighlight, error)
}

type IngameService struct {
	dashboard    ingameDashboard
	statuses     ingameStatusSource
	players      ingamePlayerSource
	rankings     ingameRankingSource
	achievements ingameAchievementSource
	cache        *ingameViewCache
	now          func() time.Time
}

func NewIngameService(dashboard ingameDashboard, statuses ingameStatusSource, players ingamePlayerSource, rankings ingameRankingSource, achievements ingameAchievementSource) *IngameService {
	return &IngameService{
		dashboard: dashboard, statuses: statuses, players: players, rankings: rankings,
		achievements: achievements, cache: newIngameViewCache(), now: time.Now,
	}
}

type IngameServerOption struct {
	ServerKey string
	Title     string
}

type IngameDocumentLink struct {
	Key   string
	Label string
}

type IngameBaseView struct {
	ServerKey     string
	Server        store.GameServer
	Status        store.ServerStatus
	Config        ResolvedIngameConfig
	Documents     []IngameDocumentLink
	WebsiteHref   string
	ServerOptions []IngameServerOption
	SelectionOnly bool
}

type IngameAnnouncementSummary struct {
	ID        string
	Title     string
	Summary   string
	UpdatedAt int64
}

type IngameOnlinePlayer struct {
	Name            string
	SteamID         string
	DurationSeconds int64
}

type IngameHighlightView struct {
	Metric IngameMetricDefinition
	Name   string
	Value  float64
}

type IngameHomeView struct {
	IngameBaseView
	Announcement *IngameAnnouncementSummary
	Players      []IngameOnlinePlayer
	Bots         int
	Highlights   []IngameHighlightView
	StatsFailed  bool
}

type IngamePVEView struct {
	Available           bool
	CommonKills         int64
	SpecialKills        int64
	BossKills           int64
	Rescues             int64
	CampaignCompletions int64
	HeadshotKills       int64
}

type IngameVersusView struct {
	Available               bool
	ShowSurvivor            bool
	ShowInfected            bool
	HumanSIKills            int64
	InfectedDamage          int64
	SurvivorControls        int64
	SurvivorIncapacitations int64
}

type IngamePlayerView struct {
	IngameBaseView
	SteamID       string
	PlayerName    string
	ShowOverview  bool
	Summary       *store.PlayerSummary
	PVE           *IngamePVEView
	Achievements  *CompactAchievementOverview
	Companions    []store.PlayerRelationship
	Versus        *IngameVersusView
	NothingPublic bool
}

type IngameRankingView struct {
	IngameBaseView
	Metric     IngameMetricDefinition
	Page       store.RankingPage
	PageNumber int
	PageCount  int
}

type IngameInfoView struct {
	IngameBaseView
	Key             string
	Title           string
	ContentMarkdown string
}

type IngameAnnouncementView struct {
	IngameBaseView
	Announcement store.Announcement
}

type ingamePortalContext struct {
	base     IngameBaseView
	settings store.IngameSettings
}

func (s *IngameService) Home(ctx context.Context, serverKey string) (IngameHomeView, error) {
	key := "home:" + strings.TrimSpace(serverKey)
	value, err := s.cache.get(ctx, key, func(buildCtx context.Context) (ingameBuildResult, error) {
		portal, err := s.portalContext(buildCtx, serverKey, true)
		if err != nil {
			return ingameBuildResult{}, err
		}
		view := IngameHomeView{IngameBaseView: portal.base}
		if view.SelectionOnly {
			return ingameBuildResult{value: view, ttl: time.Duration(portal.settings.HomeCacheSeconds) * time.Second}, nil
		}
		if portal.base.Config.Modules.ShowAnnouncements {
			if page, announcementErr := s.dashboard.ListAnnouncements(buildCtx, store.AnnouncementFilter{Limit: 1}); announcementErr == nil && len(page.Items) > 0 {
				item := page.Items[0]
				view.Announcement = &IngameAnnouncementSummary{ID: item.ID, Title: item.Title, Summary: markdownSummary(item.ContentMarkdown, 140), UpdatedAt: item.UpdatedAt}
			}
		}
		if portal.base.Status.Online && portal.base.Config.Modules.ShowPlayers {
			view.Players = onlinePlayers(portal.base.Status.PlayerList)
			view.Bots = portal.base.Status.Bots
		}
		if portal.base.Status.Online && portal.base.Config.Modules.ShowHighlights {
			ids, names := highlightPlayers(view.Players)
			if len(ids) >= 2 && s.rankings != nil {
				highlightCtx, cancel := context.WithTimeout(buildCtx, 750*time.Millisecond)
				highlights, highlightErr := s.rankings.IngameHighlights(highlightCtx, portal.base.ServerKey, ids, portal.base.Config.Metrics)
				cancel()
				if highlightErr == nil {
					for _, highlight := range highlights {
						view.Highlights = append(view.Highlights, IngameHighlightView{Metric: highlight.Metric, Name: names[highlight.SteamID], Value: highlight.Value})
					}
				} else {
					view.StatsFailed = true
				}
			}
		}
		return ingameBuildResult{value: view, ttl: time.Duration(portal.settings.HomeCacheSeconds) * time.Second}, nil
	})
	if err != nil {
		return IngameHomeView{}, err
	}
	return value.(IngameHomeView), nil
}

func (s *IngameService) Player(ctx context.Context, serverKey, steamID string) (IngamePlayerView, error) {
	steamID = strings.TrimSpace(steamID)
	if !validSteamID64(steamID) {
		return IngamePlayerView{}, ErrIngamePlayerNotFound
	}
	key := "player:" + strings.TrimSpace(serverKey) + ":" + steamID
	value, err := s.cache.get(ctx, key, func(buildCtx context.Context) (ingameBuildResult, error) {
		portal, err := s.portalContext(buildCtx, serverKey, false)
		if err != nil {
			return ingameBuildResult{}, err
		}
		visibility, err := s.dashboard.PlayerProfileVisibility(buildCtx, steamID)
		if err != nil {
			return ingameBuildResult{}, err
		}
		view := IngamePlayerView{IngameBaseView: portal.base, SteamID: steamID}
		view.ShowOverview = visibility.Visible(store.PlayerProfileOverview)
		if view.ShowOverview {
			summary, summaryErr := s.players.Summary(buildCtx, steamID)
			if summaryErr != nil {
				return ingameBuildResult{}, summaryErr
			}
			if summary == nil {
				return ingameBuildResult{}, ErrIngamePlayerNotFound
			}
			view.Summary, view.PlayerName = summary, summary.LastName
			pve, pveErr := s.players.PVE(buildCtx, steamID, 0)
			if pveErr != nil {
				return ingameBuildResult{}, pveErr
			}
			view.PVE = compactPVE(pve)
		}
		if visibility.Visible(store.PlayerProfileAchievements) && s.achievements != nil {
			if achievements, achievementErr := s.achievements.Compact(buildCtx, steamID); achievementErr == nil {
				view.Achievements = &achievements
			}
		}
		if visibility.Visible(store.PlayerProfileRelationships) {
			page, relationshipErr := s.players.Relationships(buildCtx, steamID, store.PlayerRelationshipQuery{
				PlayerFilter: store.PlayerFilter{GameMode: "all"}, Page: 1, PageSize: 3,
				Sort: "shared_seconds", Order: "desc",
			})
			if relationshipErr == nil {
				view.Companions = page.Items
			}
		}
		showSurvivor := visibility.Visible(store.PlayerProfileVersusSurvivor)
		showInfected := visibility.Visible(store.PlayerProfileVersusInfected)
		if showSurvivor || showInfected {
			versus, versusErr := s.players.Versus(buildCtx, steamID, 0)
			if versusErr == nil {
				view.Versus = compactVersus(versus, showSurvivor, showInfected)
			}
		}
		if view.PlayerName == "" {
			view.PlayerName = "玩家档案"
		}
		view.NothingPublic = !view.ShowOverview && view.Achievements == nil && len(view.Companions) == 0 && view.Versus == nil
		return ingameBuildResult{value: view, ttl: time.Duration(portal.settings.PlayerCacheSeconds) * time.Second}, nil
	})
	if err != nil {
		return IngamePlayerView{}, err
	}
	return value.(IngamePlayerView), nil
}

func (s *IngameService) Rankings(ctx context.Context, serverKey, metricKey string, page int) (IngameRankingView, error) {
	if page < 1 {
		return IngameRankingView{}, fmt.Errorf("page must be positive")
	}
	if metricKey == "" {
		metricKey = "active_play_seconds"
	}
	metric, ok := IngameMetric(metricKey)
	if !ok {
		return IngameRankingView{}, fmt.Errorf("unsupported ranking metric")
	}
	key := fmt.Sprintf("ranking:%s:%s:%d", strings.TrimSpace(serverKey), metricKey, page)
	value, err := s.cache.get(ctx, key, func(buildCtx context.Context) (ingameBuildResult, error) {
		portal, err := s.portalContext(buildCtx, serverKey, false)
		if err != nil {
			return ingameBuildResult{}, err
		}
		ranking, err := s.rankings.List(buildCtx, store.RankingQuery{
			Mode: metric.Mode, Metric: metric.RankingMetric, ServerKey: portal.base.ServerKey,
			Limit: 10, Offset: (page - 1) * 10,
		})
		if err != nil {
			return ingameBuildResult{}, err
		}
		pageCount := int((ranking.Total + 9) / 10)
		view := IngameRankingView{IngameBaseView: portal.base, Metric: metric, Page: ranking, PageNumber: page, PageCount: pageCount}
		return ingameBuildResult{value: view, ttl: time.Duration(portal.settings.RankingCacheSeconds) * time.Second}, nil
	})
	if err != nil {
		return IngameRankingView{}, err
	}
	return value.(IngameRankingView), nil
}

func (s *IngameService) Info(ctx context.Context, serverKey, documentKey string) (IngameInfoView, error) {
	if !validIngameDocumentKey(documentKey) {
		return IngameInfoView{}, ErrIngameContentUnavailable
	}
	key := "info:" + strings.TrimSpace(serverKey) + ":" + documentKey
	value, err := s.cache.get(ctx, key, func(buildCtx context.Context) (ingameBuildResult, error) {
		portal, err := s.portalContext(buildCtx, serverKey, false)
		if err != nil {
			return ingameBuildResult{}, err
		}
		serverDocument, err := s.dashboard.GetServerDocument(buildCtx, portal.base.Server.ID, documentKey)
		if err != nil {
			return ingameBuildResult{}, err
		}
		siteDocument, siteErr := s.dashboard.GetSiteDocument(buildCtx, documentKey, false)
		if siteErr != nil {
			siteDocument = store.SiteDocument{Key: documentKey}
		}
		content, available := ResolveIngameDocument(serverDocument, siteDocument)
		if !available {
			return ingameBuildResult{}, ErrIngameContentUnavailable
		}
		view := IngameInfoView{IngameBaseView: portal.base, Key: documentKey, Title: ingameDocumentLabel(documentKey), ContentMarkdown: content}
		return ingameBuildResult{value: view, ttl: time.Duration(portal.settings.ContentCacheSeconds) * time.Second}, nil
	})
	if err != nil {
		return IngameInfoView{}, err
	}
	return value.(IngameInfoView), nil
}

func (s *IngameService) Announcement(ctx context.Context, serverKey, id string) (IngameAnnouncementView, error) {
	key := "announcement:" + strings.TrimSpace(serverKey) + ":" + strings.TrimSpace(id)
	value, err := s.cache.get(ctx, key, func(buildCtx context.Context) (ingameBuildResult, error) {
		portal, err := s.portalContext(buildCtx, serverKey, false)
		if err != nil {
			return ingameBuildResult{}, err
		}
		if !portal.base.Config.Modules.ShowAnnouncements {
			return ingameBuildResult{}, ErrIngameContentUnavailable
		}
		announcement, err := s.dashboard.GetAnnouncement(buildCtx, id)
		if err != nil {
			return ingameBuildResult{}, ErrIngameContentUnavailable
		}
		view := IngameAnnouncementView{IngameBaseView: portal.base, Announcement: announcement}
		return ingameBuildResult{value: view, ttl: time.Duration(portal.settings.ContentCacheSeconds) * time.Second}, nil
	})
	if err != nil {
		return IngameAnnouncementView{}, err
	}
	return value.(IngameAnnouncementView), nil
}

func (s *IngameService) portalContext(ctx context.Context, requestedKey string, allowSelection bool) (ingamePortalContext, error) {
	settings, err := s.dashboard.IngameSettings(ctx)
	if err != nil {
		return ingamePortalContext{}, err
	}
	if !settings.Enabled {
		return ingamePortalContext{}, ErrIngameDisabled
	}
	servers, err := s.dashboard.ListServers(ctx)
	if err != nil {
		return ingamePortalContext{}, err
	}
	statuses, err := s.statuses.CachedStatuses(ctx)
	if err != nil {
		return ingamePortalContext{}, err
	}
	statusByServer := make(map[string]store.ServerStatus, len(statuses))
	for _, status := range statuses {
		statusByServer[status.ServerID] = status
	}
	options := make([]IngameServerOption, 0, len(servers))
	var selectedServer store.GameServer
	var selectedStatus store.ServerStatus
	requestedKey = strings.TrimSpace(requestedKey)
	for _, server := range servers {
		if !server.Enabled {
			continue
		}
		status, ok := statusByServer[server.ID]
		if !ok || status.ServerKey == "" {
			continue
		}
		options = append(options, IngameServerOption{ServerKey: status.ServerKey, Title: server.DisplayName})
		if requestedKey != "" && status.ServerKey == requestedKey {
			selectedServer, selectedStatus = server, status
		}
	}
	if requestedKey == "" && len(options) == 1 {
		for _, server := range servers {
			status := statusByServer[server.ID]
			if server.Enabled && status.ServerKey == options[0].ServerKey {
				selectedServer, selectedStatus = server, status
				break
			}
		}
	}
	if selectedServer.ID == "" {
		if requestedKey == "" && allowSelection && len(options) > 1 {
			return ingamePortalContext{settings: settings, base: IngameBaseView{ServerOptions: options, SelectionOnly: true}}, nil
		}
		return ingamePortalContext{}, ErrIngameUnknownServer
	}
	if site, siteErr := s.dashboard.SiteSettings(ctx); siteErr == nil && selectedStatus.Online && !selectedStatus.LastSuccessAt.IsZero() {
		refresh := site.A2SRefreshSeconds
		if refresh <= 0 {
			refresh = 30
		}
		if s.now().Sub(selectedStatus.LastSuccessAt) > time.Duration(refresh*2)*time.Second {
			selectedStatus.Stale = true
		}
	}
	serverSettings, err := s.dashboard.IngameServerSettings(ctx, selectedServer.ID)
	if err != nil {
		serverSettings = store.IngameServerSettings{ServerID: selectedServer.ID, TitleMode: "inherit", DescriptionMode: "inherit", BannerMode: "inherit", WebsiteMode: "inherit", HighlightMode: "inherit"}
	}
	config := ResolveIngameConfig(settings, serverSettings, selectedServer.DisplayName)
	base := IngameBaseView{
		ServerKey: selectedStatus.ServerKey, Server: selectedServer, Status: selectedStatus,
		Config: config, WebsiteHref: BuildExternalBrowserHref(config.Appearance.WebsiteURL),
	}
	base.Documents = s.documentLinks(ctx, selectedServer.ID)
	return ingamePortalContext{settings: settings, base: base}, nil
}

func (s *IngameService) documentLinks(ctx context.Context, serverID string) []IngameDocumentLink {
	siteDocuments, siteErr := s.dashboard.ListSiteDocuments(ctx, false)
	serverDocuments, serverErr := s.dashboard.ListServerDocuments(ctx, serverID)
	if siteErr != nil || serverErr != nil {
		return nil
	}
	siteByKey := make(map[string]store.SiteDocument, len(siteDocuments))
	serverByKey := make(map[string]store.ServerDocument, len(serverDocuments))
	for _, document := range siteDocuments {
		siteByKey[document.Key] = document
	}
	for _, document := range serverDocuments {
		serverByKey[document.Key] = document
	}
	links := make([]IngameDocumentLink, 0, 3)
	for _, key := range []string{store.IngameDocumentIntroduction, store.IngameDocumentCommands, store.IngameDocumentResources} {
		serverDocument := serverByKey[key]
		if serverDocument.Mode == "" {
			serverDocument = store.ServerDocument{ServerID: serverID, Key: key, Mode: "inherit"}
		}
		if _, available := ResolveIngameDocument(serverDocument, siteByKey[key]); available {
			links = append(links, IngameDocumentLink{Key: key, Label: ingameDocumentLabel(key)})
		}
	}
	return links
}

func (s *IngameService) InvalidateAll()                    { s.cache.clear() }
func (s *IngameService) InvalidateServer(serverKey string) { s.cache.clearServer(serverKey) }
func (s *IngameService) InvalidatePlayer(steamID string)   { s.cache.clearPlayer(steamID) }

func onlinePlayers(players []store.ServerPlayer) []IngameOnlinePlayer {
	result := make([]IngameOnlinePlayer, 0, len(players))
	for _, player := range players {
		if strings.TrimSpace(player.Name) == "" {
			continue
		}
		result = append(result, IngameOnlinePlayer{Name: player.Name, SteamID: player.SteamID, DurationSeconds: player.DurationSeconds})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].DurationSeconds == result[j].DurationSeconds {
			return result[i].Name < result[j].Name
		}
		return result[i].DurationSeconds > result[j].DurationSeconds
	})
	if len(result) > 32 {
		result = result[:32]
	}
	return result
}

func highlightPlayers(players []IngameOnlinePlayer) ([]string, map[string]string) {
	ids := make([]string, 0, len(players))
	names := make(map[string]string, len(players))
	for _, player := range players {
		if !validSteamID64(player.SteamID) {
			continue
		}
		if _, exists := names[player.SteamID]; exists {
			continue
		}
		ids = append(ids, player.SteamID)
		names[player.SteamID] = player.Name
	}
	return ids, names
}

func compactPVE(pve store.PlayerPVE) *IngamePVEView {
	var headshots int64
	for _, equipment := range pve.Equipment {
		headshots += equipment.HeadshotKills
	}
	result := &IngamePVEView{
		CommonKills: pve.CommonKills, SpecialKills: pve.SpecialKills,
		BossKills:           pve.TankKills + pve.WitchKills,
		Rescues:             pve.IncapRevives + pve.LedgeRescues + pve.DefibRevives,
		CampaignCompletions: pve.CampaignCompletions, HeadshotKills: headshots,
	}
	result.Available = result.CommonKills+result.SpecialKills+result.BossKills+result.Rescues+result.CampaignCompletions+result.HeadshotKills > 0
	return result
}

func compactVersus(versus store.PlayerVersus, survivor, infected bool) *IngameVersusView {
	result := &IngameVersusView{ShowSurvivor: survivor, ShowInfected: infected}
	if survivor {
		result.HumanSIKills = versus.HumanSpecialKills + versus.HumanTankKills
	}
	if infected {
		result.InfectedDamage = versus.DamageToHumanSurvivors
		result.SurvivorControls = versus.HumanSurvivorControls
		result.SurvivorIncapacitations = versus.HumanSurvivorIncaps
	}
	result.Available = result.HumanSIKills+result.InfectedDamage+result.SurvivorControls+result.SurvivorIncapacitations > 0
	return result
}

func validSteamID64(value string) bool {
	if len(value) != 17 {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func validIngameDocumentKey(value string) bool {
	return value == store.IngameDocumentIntroduction || value == store.IngameDocumentCommands || value == store.IngameDocumentResources
}

func ingameDocumentLabel(key string) string {
	switch key {
	case store.IngameDocumentCommands:
		return "常用命令"
	case store.IngameDocumentResources:
		return "服务器资源"
	default:
		return "服务器介绍"
	}
}

func markdownSummary(value string, limit int) string {
	var builder strings.Builder
	space := false
	for _, character := range value {
		if unicode.IsLetter(character) || unicode.IsNumber(character) || unicode.IsPunct(character) && !strings.ContainsRune("#*_`[]>!~-", character) {
			builder.WriteRune(character)
			space = false
		} else if unicode.IsSpace(character) && !space && builder.Len() > 0 {
			builder.WriteByte(' ')
			space = true
		}
	}
	plain := strings.TrimSpace(builder.String())
	runes := []rune(plain)
	if len(runes) > limit {
		return strings.TrimSpace(string(runes[:limit])) + "…"
	}
	return plain
}

func FormatIngameValue(metric IngameMetricDefinition, value float64) string {
	if metric.Format == "duration" {
		hours := int64(value) / 3600
		minutes := (int64(value) % 3600) / 60
		if hours > 0 {
			return fmt.Sprintf("%dh %02dm", hours, minutes)
		}
		return fmt.Sprintf("%dm", minutes)
	}
	return strconv.FormatInt(int64(value), 10)
}
