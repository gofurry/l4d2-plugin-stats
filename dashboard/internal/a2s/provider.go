package a2s

import (
	"context"
	"math/rand/v2"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	steama2s "github.com/gofurry/steam-go/addons/a2s"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"
)

type Info struct {
	Name       string
	Map        string
	Players    int
	MaxPlayers int
	Bots       int
}

type Player struct {
	Name            string
	Score           int32
	DurationSeconds int64
}

type Rule struct {
	Name  string
	Value string
}

type Client interface {
	Query(context.Context, string, time.Duration) (Info, []Player, []Rule, time.Duration, error)
}

type SteamClient struct{}

func (SteamClient) Query(ctx context.Context, address string, timeout time.Duration) (Info, []Player, []Rule, time.Duration, error) {
	client, err := steama2s.NewClient(address, steama2s.WithTimeout(timeout))
	if err != nil {
		return Info{}, nil, nil, 0, err
	}
	defer client.Close()
	started := time.Now()
	result, err := client.QueryInfo(ctx)
	latency := time.Since(started)
	if err != nil {
		return Info{}, nil, nil, latency, err
	}
	players := make([]Player, 0)
	rules := make([]Rule, 0)
	details, detailsCtx := errgroup.WithContext(ctx)
	details.Go(func() error {
		playerClient, clientErr := steama2s.NewClient(address, steama2s.WithTimeout(timeout))
		if clientErr != nil {
			return nil
		}
		defer playerClient.Close()
		if resultPlayers, playerErr := playerClient.QueryPlayers(detailsCtx); playerErr == nil {
			players = make([]Player, 0, len(resultPlayers.Players))
			for _, player := range resultPlayers.Players {
				players = append(players, Player{Name: player.Name, Score: player.Score, DurationSeconds: int64(player.Duration)})
			}
		}
		return nil
	})
	details.Go(func() error {
		rulesClient, clientErr := steama2s.NewClient(address, steama2s.WithTimeout(timeout))
		if clientErr != nil {
			return nil
		}
		defer rulesClient.Close()
		if resultRules, rulesErr := rulesClient.QueryRules(detailsCtx); rulesErr == nil {
			names := make([]string, 0, len(resultRules.Items))
			for name := range resultRules.Items {
				names = append(names, name)
			}
			sort.Strings(names)
			rules = make([]Rule, 0, len(names))
			for _, name := range names {
				rules = append(rules, Rule{Name: name, Value: resultRules.Items[name]})
			}
		}
		return nil
	})
	_ = details.Wait()
	return Info{Name: result.Name, Map: result.Map, Players: int(result.Players), MaxPlayers: int(result.MaxPlayers), Bots: int(result.Bots)}, players, rules, latency, nil
}

type Provider struct {
	dashboard store.DashboardStore
	snapshots store.ServerStatusSnapshotStore
	presence  store.StatsPresenceStore
	client    Client
	timeout   time.Duration
	ttl       time.Duration // test override; production uses Dashboard settings
	staleTTL  time.Duration
	semaphore chan struct{}
	mu        sync.RWMutex
	entries   map[string]cacheEntry
	group     singleflight.Group
	hydrateMu sync.Mutex
	hydrated  bool
}

type cacheEntry struct {
	status  store.ServerStatus
	expires time.Time
}

type queryPolicy struct {
	refresh time.Duration
	jitter  time.Duration
	retries int
}

func NewProvider(dashboard store.DashboardStore, client Client, presence ...store.StatsPresenceStore) *Provider {
	provider := &Provider{
		dashboard: dashboard, client: client, timeout: 2 * time.Second,
		staleTTL:  5 * time.Minute,
		semaphore: make(chan struct{}, 4), entries: make(map[string]cacheEntry),
	}
	if len(presence) > 0 {
		provider.presence = presence[0]
	}
	provider.snapshots, _ = dashboard.(store.ServerStatusSnapshotStore)
	return provider
}

// Start keeps the in-memory status snapshot warm independently of homepage
// traffic. The first pass runs immediately; later passes follow the refresh
// interval from the current site settings.
func (p *Provider) Start(ctx context.Context) {
	go func() {
		first := true
		for {
			policy := p.refreshEnabled(ctx, first)
			first = false
			timer := time.NewTimer(policy.refresh)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				return
			case <-timer.C:
			}
		}
	}()
}

func (p *Provider) Statuses(ctx context.Context) ([]store.ServerStatus, error) {
	p.hydrate(ctx)
	settings, err := p.dashboard.SiteSettings(ctx)
	if err != nil {
		return nil, err
	}
	policy := normalizePolicy(settings)
	servers, err := p.dashboard.ListServers(ctx)
	if err != nil {
		return nil, err
	}
	p.removeUnknownServers(servers)
	enabled := make([]store.GameServer, 0, len(servers))
	for _, server := range servers {
		if server.Enabled {
			enabled = append(enabled, server)
		}
	}
	statuses := make([]store.ServerStatus, len(enabled))
	for i := range enabled {
		statuses[i] = p.status(enabled[i], policy)
	}
	return statuses, nil
}

// CachedStatuses returns the in-memory/persisted snapshot without scheduling
// or executing an A2S query. It is the only status path used by the MOTD portal.
func (p *Provider) CachedStatuses(ctx context.Context) ([]store.ServerStatus, error) {
	p.hydrate(ctx)
	p.mu.RLock()
	defer p.mu.RUnlock()
	statuses := make([]store.ServerStatus, 0, len(p.entries))
	for _, entry := range p.entries {
		status := entry.status
		status.PlayerList = append([]store.ServerPlayer(nil), status.PlayerList...)
		status.Rules = append([]store.ServerRule(nil), status.Rules...)
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func (p *Provider) refreshEnabled(ctx context.Context, immediate bool) queryPolicy {
	p.hydrate(ctx)
	settings, err := p.dashboard.SiteSettings(ctx)
	if err != nil {
		return normalizePolicy(store.SiteSettings{})
	}
	policy := normalizePolicy(settings)
	refreshPolicy := policy
	if immediate {
		refreshPolicy.jitter = 0
	}
	servers, err := p.dashboard.ListServers(ctx)
	if err != nil {
		return policy
	}
	p.removeUnknownServers(servers)
	group, groupCtx := errgroup.WithContext(ctx)
	for _, server := range servers {
		if !server.Enabled {
			continue
		}
		server := server
		group.Go(func() error {
			_, _, _ = p.group.Do(serverCacheKey(server), func() (any, error) {
				return p.refresh(groupCtx, server, refreshPolicy)
			})
			return nil
		})
	}
	_ = group.Wait()
	return policy
}

func (p *Provider) InvalidateServer(serverID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for key, entry := range p.entries {
		if entry.status.ServerID == serverID {
			delete(p.entries, key)
		}
	}
}

func (p *Provider) removeUnknownServers(servers []store.GameServer) {
	known := make(map[string]struct{}, len(servers))
	for _, server := range servers {
		known[serverCacheKey(server)] = struct{}{}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for key := range p.entries {
		if _, ok := known[key]; !ok {
			delete(p.entries, key)
		}
	}
}

func (p *Provider) LastStatus(ctx context.Context, serverID string) (store.ServerStatus, bool, error) {
	p.hydrate(ctx)
	server, err := p.serverByID(ctx, serverID)
	if err != nil {
		return store.ServerStatus{}, false, err
	}
	p.mu.RLock()
	entry, exists := p.entries[serverCacheKey(server)]
	p.mu.RUnlock()
	return entry.status, exists, nil
}

func (p *Provider) RefreshStatus(ctx context.Context, serverID string) (store.ServerStatus, error) {
	settings, err := p.dashboard.SiteSettings(ctx)
	if err != nil {
		return store.ServerStatus{}, err
	}
	server, err := p.serverByID(ctx, serverID)
	if err != nil {
		return store.ServerStatus{}, err
	}
	key := serverCacheKey(server)
	policy := normalizePolicy(settings)
	policy.jitter = 0
	result := p.group.DoChan(key, func() (any, error) { return p.refresh(ctx, server, policy) })
	select {
	case <-ctx.Done():
		return store.ServerStatus{}, ctx.Err()
	case value := <-result:
		if value.Err != nil {
			return store.ServerStatus{}, value.Err
		}
		return value.Val.(store.ServerStatus), nil
	}
}

func (p *Provider) serverByID(ctx context.Context, serverID string) (store.GameServer, error) {
	servers, err := p.dashboard.ListServers(ctx)
	if err != nil {
		return store.GameServer{}, err
	}
	for _, server := range servers {
		if server.ID == serverID {
			return server, nil
		}
	}
	return store.GameServer{}, store.ErrServerNotFound
}

func (p *Provider) status(server store.GameServer, policy queryPolicy) store.ServerStatus {
	now := time.Now()
	key := serverCacheKey(server)
	p.mu.RLock()
	entry, exists := p.entries[key]
	p.mu.RUnlock()
	if exists && now.Before(entry.expires) {
		return entry.status
	}
	backgroundPolicy := policy
	backgroundPolicy.jitter = 0
	p.group.DoChan(key, func() (any, error) {
		return p.refresh(context.Background(), server, backgroundPolicy)
	})
	if exists {
		status := entry.status
		status.Stale = true
		status.Checking = true
		return status
	}
	return store.ServerStatus{
		ServerID: server.ID, DisplayName: server.DisplayName, Address: server.Address,
		Checking: true,
	}
}

func (p *Provider) refresh(ctx context.Context, server store.GameServer, policy queryPolicy) (store.ServerStatus, error) {
	if policy.jitter > 0 {
		timer := time.NewTimer(time.Duration(rand.Int64N(int64(policy.jitter) + 1)))
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return store.ServerStatus{}, ctx.Err()
		case <-timer.C:
		}
	}
	select {
	case <-ctx.Done():
		return store.ServerStatus{}, ctx.Err()
	case p.semaphore <- struct{}{}:
	}
	defer func() { <-p.semaphore }()
	var info Info
	var players []Player
	var rules []Rule
	var latency time.Duration
	var err error
	for attempt := 0; attempt <= policy.retries; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, p.timeout)
		info, players, rules, latency, err = p.client.Query(attemptCtx, server.Address, p.timeout)
		cancel()
		if err == nil {
			break
		}
	}
	now := time.Now().UTC()
	key := serverCacheKey(server)
	cacheTTL := policy.refresh
	if p.ttl > 0 {
		cacheTTL = p.ttl
	}
	if err != nil {
		p.mu.RLock()
		previous, exists := p.entries[key]
		p.mu.RUnlock()
		status := store.ServerStatus{
			ServerID: server.ID, DisplayName: server.DisplayName, Address: server.Address,
			Online: false, Checking: false, CheckedAt: now,
		}
		if exists && !previous.status.LastSuccessAt.IsZero() && now.Sub(previous.status.LastSuccessAt) <= p.staleTTL {
			status = previous.status
			status.Online = false
			status.Stale = true
			status.Checking = false
			status.CheckedAt = now
		}
		p.mu.Lock()
		p.entries[key] = cacheEntry{status: status, expires: now.Add(cacheTTL)}
		p.mu.Unlock()
		return status, nil
	}
	status := store.ServerStatus{
		ServerID: server.ID, DisplayName: server.DisplayName, Address: server.Address,
		Online: true, Name: info.Name, Map: info.Map, Players: info.Players,
		MaxPlayers: info.MaxPlayers, Bots: info.Bots, LatencyMS: latency.Milliseconds(),
		LastSuccessAt: now, CheckedAt: now, Checking: false,
	}
	status.Rules = make([]store.ServerRule, 0, len(rules))
	for _, rule := range rules {
		status.Rules = append(status.Rules, store.ServerRule{Name: rule.Name, Value: rule.Value})
	}
	status.ServerKey = serverKeyFromRules(rules)
	status.PlayerList = p.linkPlayers(status.ServerKey, players, now)
	p.mu.Lock()
	p.entries[key] = cacheEntry{status: status, expires: now.Add(cacheTTL)}
	p.mu.Unlock()
	p.persist(status)
	return status, nil
}

func (p *Provider) hydrate(ctx context.Context) {
	if p.snapshots == nil {
		return
	}
	p.hydrateMu.Lock()
	defer p.hydrateMu.Unlock()
	if p.hydrated {
		return
	}
	statuses, err := p.snapshots.ListServerStatusSnapshots(ctx)
	if err != nil {
		return
	}
	p.mu.Lock()
	for _, status := range statuses {
		if status.ServerID == "" || status.Address == "" || status.LastSuccessAt.IsZero() {
			continue
		}
		status.Stale = true
		status.Checking = false
		p.entries[status.ServerID+"\x00"+status.Address] = cacheEntry{status: status}
	}
	p.mu.Unlock()
	p.hydrated = true
}

func (p *Provider) persist(status store.ServerStatus) {
	if p.snapshots == nil || !status.Online || status.LastSuccessAt.IsZero() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = p.snapshots.UpsertServerStatusSnapshot(ctx, status)
}

func (p *Provider) linkPlayers(serverKey string, a2sPlayers []Player, now time.Time) []store.ServerPlayer {
	fallback := make([]store.ServerPlayer, 0, len(a2sPlayers))
	for _, player := range a2sPlayers {
		fallback = append(fallback, store.ServerPlayer{Name: player.Name, Score: player.Score, DurationSeconds: player.DurationSeconds})
	}
	if serverKey == "" || p.presence == nil {
		return fallback
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	active, err := p.presence.ActivePlayers(ctx, serverKey, now.Add(-10*time.Minute).Unix())
	cancel()
	if err != nil || len(active) == 0 {
		return fallback
	}

	byName := make(map[string][]Player, len(a2sPlayers))
	for _, player := range a2sPlayers {
		name := strings.TrimSpace(player.Name)
		byName[name] = append(byName[name], player)
	}
	linked := make([]store.ServerPlayer, 0, len(active))
	for _, player := range active {
		duration := player.ConnectedSeconds
		if player.LastSavedAt > 0 && now.Unix() > player.LastSavedAt {
			duration += now.Unix() - player.LastSavedAt
		}
		entry := store.ServerPlayer{Name: player.Name, SteamID: player.SteamID, DurationSeconds: duration}
		if matches := byName[strings.TrimSpace(player.Name)]; len(matches) == 1 {
			entry.Score = matches[0].Score
			entry.DurationSeconds = matches[0].DurationSeconds
		}
		linked = append(linked, entry)
	}
	return linked
}

func serverKeyFromRules(rules []Rule) string {
	for _, rule := range rules {
		if strings.EqualFold(strings.TrimSpace(rule.Name), "sm_lps_server_key") {
			value := strings.TrimSpace(rule.Value)
			if validServerKey(value) {
				return value
			}
			return ""
		}
	}
	return ""
}

func validServerKey(value string) bool {
	if len(value) < 1 || len(value) > 64 || strings.EqualFold(value, "change-me") {
		return false
	}
	for _, char := range value {
		if (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '.' || char == '_' || char == '-' {
			continue
		}
		return false
	}
	return true
}

func normalizePolicy(settings store.SiteSettings) queryPolicy {
	refresh := settings.A2SRefreshSeconds
	if refresh != 5 && refresh != 10 && refresh != 15 && refresh != 30 && refresh != 45 && refresh != 60 {
		refresh = 30
	}
	jitter := settings.A2SJitterSeconds
	if jitter != 0 && jitter != 2 && jitter != 5 {
		jitter = 2
	}
	retries := settings.A2SRetryCount
	if retries < 1 || retries > 3 {
		retries = 1
	}
	return queryPolicy{refresh: time.Duration(refresh) * time.Second, jitter: time.Duration(jitter) * time.Second, retries: int(retries)}
}

func serverCacheKey(server store.GameServer) string {
	return server.ID + "\x00" + server.Address
}
