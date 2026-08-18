package service

import (
	"context"
	"errors"
	"fmt"
	"net"
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
	ListIngameServerSettings(context.Context) ([]store.IngameServerSettings, error)
	ListServers(context.Context) ([]store.GameServer, error)
	ListSiteDocuments(context.Context, bool) ([]store.SiteDocument, error)
	GetSiteDocument(context.Context, string, bool) (store.SiteDocument, error)
	ListServerDocuments(context.Context, string) ([]store.ServerDocument, error)
	GetServerDocument(context.Context, string, string) (store.ServerDocument, error)
	ListServerQuickLinks(context.Context, string) ([]store.IngameQuickLink, error)
	ListIngameMapNames(context.Context) ([]store.IngameMapName, error)
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
	IngameRecent24h(context.Context, string, time.Time) (store.ServerRecent24h, error)
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
	Instances []IngameServerOptionInstance
}

type IngameServerOptionInstance struct {
	DisplayName string
	Address     string
}

type IngameServerInstance struct {
	ServerID      string
	DisplayName   string
	Address       string
	SortOrder     int64
	Online        bool
	Stale         bool
	Checking      bool
	Map           string
	Players       int
	MaxPlayers    int
	Bots          int
	GameMode      string
	Difficulty    string
	LatencyMS     int64
	LastSuccessAt time.Time
	ActionID      string
}

type IngameDocumentLink struct {
	Key   string
	Label string
}

type IngameQuickLinkView struct {
	Label    string
	ActionID string
}

type IngameActionView struct {
	ID     string
	Title  string
	Prompt string
	Value  string
}

type IngameBaseView struct {
	ServerKey         string
	Config            ResolvedIngameConfig
	Documents         []IngameDocumentLink
	QuickLinks        []IngameQuickLinkView
	WebsiteActionID   string
	Actions           []IngameActionView
	Instances         []IngameServerInstance
	OnlineInstances   int
	TotalInstances    int
	OnlinePlayerCount int
	BotCount          int
	ServerOptions     []IngameServerOption
	SelectionOnly     bool
	ActivePage        string
}

type IngameAnnouncementSummary struct {
	ID        string
	Title     string
	Summary   string
	UpdatedAt int64
}

type IngameOnlinePlayer struct {
	Name            string
	InstanceName    string
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
	Highlights   []IngameHighlightView
	Recent24h    *store.ServerRecent24h
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
	CurrentPlay   *IngameCurrentPlay
	PVE           *IngamePVEView
	Achievements  *CompactAchievementOverview
	Companions    []store.PlayerRelationship
	Versus        *IngameVersusView
	NothingPublic bool
}

type IngameCurrentPlay struct {
	InstanceName    string
	MapName         string
	DurationSeconds int64
}

type IngameRankingView struct {
	IngameBaseView
	Metric     IngameMetricDefinition
	Page       store.RankingPage
	PageNumber int
	PageCount  int
	Catalog    []IngameMetricDefinition
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
	players  []ingameSourcedPlayer
}

type ingameSourcedPlayer struct {
	player       store.ServerPlayer
	instanceName string
	mapName      string
	stale        bool
}

func (s *IngameService) Home(ctx context.Context, serverKey string) (IngameHomeView, error) {
	key := "home:" + strings.TrimSpace(serverKey)
	value, err := s.cache.get(ctx, key, func(buildCtx context.Context) (ingameBuildResult, error) {
		portal, err := s.portalContext(buildCtx, serverKey, true)
		if err != nil {
			return ingameBuildResult{}, err
		}
		view := IngameHomeView{IngameBaseView: portal.base}
		view.ActivePage = "home"
		if view.SelectionOnly {
			return ingameBuildResult{value: view, ttl: time.Duration(portal.settings.HomeCacheSeconds) * time.Second}, nil
		}
		if portal.base.Config.Modules.ShowAnnouncements {
			if page, announcementErr := s.dashboard.ListAnnouncements(buildCtx, store.AnnouncementFilter{Limit: 1}); announcementErr == nil && len(page.Items) > 0 {
				item := page.Items[0]
				view.Announcement = &IngameAnnouncementSummary{ID: item.ID, Title: item.Title, Summary: markdownSummary(item.ContentMarkdown, 140), UpdatedAt: item.UpdatedAt}
			}
		}
		if portal.base.OnlineInstances > 0 && portal.base.Config.Modules.ShowPlayers {
			view.Players = onlinePlayers(portal.players)
		}
		if s.rankings != nil {
			recentCtx, cancel := context.WithTimeout(buildCtx, time.Second)
			recent, recentErr := s.rankings.IngameRecent24h(recentCtx, portal.base.ServerKey, s.now().Add(-24*time.Hour))
			cancel()
			if recentErr == nil {
				view.Recent24h = &recent
			}
		}
		if portal.base.OnlineInstances > 0 && portal.base.Config.Modules.ShowHighlights {
			ids, names := highlightPlayers(view.Players)
			if len(ids) >= 1 && s.rankings != nil {
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
		view.ActivePage = "player"
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
			view.CurrentPlay = currentIngamePlay(portal.players, steamID)
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
		view := IngameRankingView{IngameBaseView: portal.base, Metric: metric, Page: ranking, PageNumber: page, PageCount: pageCount, Catalog: IngameMetricCatalog()}
		view.ActivePage = "rankings"
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
		serverDocument, err := s.dashboard.GetServerDocument(buildCtx, portal.base.ServerKey, documentKey)
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
		view.ActivePage = documentKey
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
		view.ActivePage = "announcement"
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
	serverSettings, err := s.dashboard.ListIngameServerSettings(ctx)
	if err != nil {
		return ingamePortalContext{}, err
	}
	customMapNames, err := s.dashboard.ListIngameMapNames(ctx)
	if err != nil {
		return ingamePortalContext{}, err
	}
	mapNames := NewMapNameResolver(customMapNames)
	settingsByKey := make(map[string]store.IngameServerSettings, len(serverSettings))
	for _, value := range serverSettings {
		settingsByKey[value.ServerKey] = value
	}
	statusByServer := make(map[string]store.ServerStatus, len(statuses))
	for _, status := range statuses {
		statusByServer[status.ServerID] = status
	}
	sort.SliceStable(servers, func(i, j int) bool {
		if servers[i].SortOrder != servers[j].SortOrder {
			return servers[i].SortOrder < servers[j].SortOrder
		}
		if servers[i].DisplayName != servers[j].DisplayName {
			return servers[i].DisplayName < servers[j].DisplayName
		}
		return servers[i].Address < servers[j].Address
	})
	type groupMember struct {
		server store.GameServer
		status store.ServerStatus
	}
	type groupValue struct {
		key     string
		title   string
		config  ResolvedIngameConfig
		members []groupMember
	}
	groups := make([]*groupValue, 0, len(servers))
	groupByKey := make(map[string]*groupValue, len(servers))
	for _, server := range servers {
		if !server.Enabled {
			continue
		}
		status, ok := statusByServer[server.ID]
		if !ok || ValidateIngameServerKey(status.ServerKey) != nil {
			continue
		}
		group := groupByKey[status.ServerKey]
		if group == nil {
			group = &groupValue{key: status.ServerKey}
			groupByKey[status.ServerKey] = group
			groups = append(groups, group)
		}
		group.members = append(group.members, groupMember{server: server, status: status})
	}
	options := make([]IngameServerOption, 0, len(groups))
	for _, group := range groups {
		fallbackTitle := group.key
		if len(group.members) > 0 && strings.TrimSpace(group.members[0].server.DisplayName) != "" {
			fallbackTitle = group.members[0].server.DisplayName
		}
		group.config = ResolveIngameConfig(settings, settingsByKey[group.key], fallbackTitle)
		group.title = group.config.Appearance.Title
		instances := make([]IngameServerOptionInstance, 0, len(group.members))
		for _, member := range group.members {
			instances = append(instances, IngameServerOptionInstance{DisplayName: member.server.DisplayName, Address: member.server.Address})
		}
		options = append(options, IngameServerOption{ServerKey: group.key, Title: group.title, Instances: instances})
	}
	requestedKey = strings.TrimSpace(requestedKey)
	var selected *groupValue
	if requestedKey != "" {
		selected = groupByKey[requestedKey]
	} else if len(groups) == 1 {
		selected = groups[0]
	}
	if selected == nil {
		if requestedKey == "" && allowSelection && len(options) > 1 {
			selectionConfig := ResolveIngameConfig(settings, store.IngameServerSettings{}, "选择服务器")
			return ingamePortalContext{settings: settings, base: IngameBaseView{Config: selectionConfig, ServerOptions: options, SelectionOnly: true, ActivePage: "home"}}, nil
		}
		return ingamePortalContext{}, ErrIngameUnknownServer
	}
	quickLinks, err := s.dashboard.ListServerQuickLinks(ctx, selected.key)
	if err != nil {
		return ingamePortalContext{}, err
	}
	refreshSeconds := int64(30)
	if loadedSite, siteErr := s.dashboard.SiteSettings(ctx); siteErr == nil {
		refresh := loadedSite.A2SRefreshSeconds
		if refresh <= 0 {
			refresh = 30
		}
		refreshSeconds = refresh
	}
	base := IngameBaseView{ServerKey: selected.key, Config: selected.config, TotalInstances: len(selected.members)}
	addAction := func(title, prompt, value string) string {
		id := fmt.Sprintf("action-%02d", len(base.Actions)+1)
		base.Actions = append(base.Actions, IngameActionView{ID: id, Title: title, Prompt: prompt, Value: value})
		return id
	}
	websiteURL := strings.TrimSpace(selected.config.Appearance.WebsiteURL)
	if websiteURL != "" && ValidateIngameURL(websiteURL) == nil {
		base.WebsiteActionID = addAction("完整网站", "请使用普通浏览器访问：", websiteURL)
	}
	for _, link := range quickLinks {
		if !link.Enabled || ValidateServerQuickLinks([]store.IngameQuickLink{link}) != nil {
			continue
		}
		base.QuickLinks = append(base.QuickLinks, IngameQuickLinkView{
			Label: link.Label, ActionID: addAction(link.Label, "请使用普通浏览器访问：", link.URL),
		})
	}
	players := make([]ingameSourcedPlayer, 0)
	for _, member := range selected.members {
		status := member.status
		if status.Online && !status.LastSuccessAt.IsZero() && s.now().Sub(status.LastSuccessAt) > time.Duration(refreshSeconds*2)*time.Second {
			status.Stale = true
		}
		instanceName := strings.TrimSpace(member.server.DisplayName)
		if instanceName == "" {
			instanceName = strings.TrimSpace(status.Name)
		}
		instance := IngameServerInstance{
			ServerID: member.server.ID, DisplayName: instanceName,
			Address: member.server.Address, SortOrder: member.server.SortOrder, Online: status.Online, Stale: status.Stale,
			Checking: status.Checking, Map: mapNames.DisplayName(status.Map), MaxPlayers: status.MaxPlayers,
			Players: status.Players, Bots: status.Bots, GameMode: serverRuleValue(status.Rules, "mp_gamemode"),
			Difficulty: serverRuleValue(status.Rules, "z_difficulty"), LatencyMS: status.LatencyMS,
			LastSuccessAt: status.LastSuccessAt,
		}
		if status.Online {
			base.OnlineInstances++
			base.OnlinePlayerCount += len(status.PlayerList)
			base.BotCount += status.Bots
			if address := BuildIngameConnectAddress(member.server.Address); address != "" {
				instance.ActionID = addAction("加入游戏", "请在游戏控制台输入：", "connect "+address)
			}
			for _, player := range status.PlayerList {
				players = append(players, ingameSourcedPlayer{player: player, instanceName: instanceName, mapName: instance.Map, stale: status.Stale})
			}
		}
		base.Instances = append(base.Instances, instance)
	}
	base.Documents = s.documentLinks(ctx, selected.key)
	return ingamePortalContext{settings: settings, base: base, players: players}, nil
}

func currentIngamePlay(players []ingameSourcedPlayer, steamID string) *IngameCurrentPlay {
	for _, sourced := range players {
		if sourced.stale || sourced.player.SteamID != steamID {
			continue
		}
		return &IngameCurrentPlay{InstanceName: sourced.instanceName, MapName: sourced.mapName, DurationSeconds: sourced.player.DurationSeconds}
	}
	return nil
}

func (s *IngameService) documentLinks(ctx context.Context, serverKey string) []IngameDocumentLink {
	siteDocuments, siteErr := s.dashboard.ListSiteDocuments(ctx, false)
	serverDocuments, serverErr := s.dashboard.ListServerDocuments(ctx, serverKey)
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
			serverDocument = store.ServerDocument{ServerKey: serverKey, Key: key, Mode: "inherit"}
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

func (s *IngameService) ErrorBackground(ctx context.Context) string {
	settings, err := s.dashboard.IngameSettings(ctx)
	if err != nil {
		return ""
	}
	background := strings.TrimSpace(settings.BackgroundURL)
	if ValidateIngameURL(background) != nil {
		return ""
	}
	return background
}

func onlinePlayers(players []ingameSourcedPlayer) []IngameOnlinePlayer {
	result := make([]IngameOnlinePlayer, 0, len(players))
	for _, sourced := range players {
		if strings.TrimSpace(sourced.player.Name) == "" {
			continue
		}
		result = append(result, IngameOnlinePlayer{
			Name: sourced.player.Name, InstanceName: sourced.instanceName,
			SteamID: sourced.player.SteamID, DurationSeconds: sourced.player.DurationSeconds,
		})
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

func BuildIngameConnectAddress(address string) string {
	host, port, err := net.SplitHostPort(strings.TrimSpace(address))
	if err != nil || !validConnectHost(host) {
		return ""
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return ""
	}
	return net.JoinHostPort(host, strconv.Itoa(portNumber))
}

func serverRuleValue(rules []store.ServerRule, name string) string {
	for _, rule := range rules {
		if !strings.EqualFold(strings.TrimSpace(rule.Name), name) {
			continue
		}
		value := []rune(strings.TrimSpace(rule.Value))
		if len(value) > 32 {
			value = value[:32]
		}
		return string(value)
	}
	return ""
}

func validConnectHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" || len(host) > 253 {
		return false
	}
	if net.ParseIP(host) != nil {
		return true
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) < 1 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, character := range label {
			if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || character == '-' {
				continue
			}
			return false
		}
	}
	return true
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
