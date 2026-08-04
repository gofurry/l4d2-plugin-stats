package a2s

import (
	"context"
	"math/rand/v2"
	"sort"
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
	client    Client
	timeout   time.Duration
	ttl       time.Duration // test override; production uses Dashboard settings
	staleTTL  time.Duration
	semaphore chan struct{}
	mu        sync.RWMutex
	entries   map[string]cacheEntry
	group     singleflight.Group
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

func NewProvider(dashboard store.DashboardStore, client Client) *Provider {
	return &Provider{
		dashboard: dashboard, client: client, timeout: 2 * time.Second,
		staleTTL:  5 * time.Minute,
		semaphore: make(chan struct{}, 4), entries: make(map[string]cacheEntry),
	}
}

func (p *Provider) Statuses(ctx context.Context) ([]store.ServerStatus, error) {
	settings, err := p.dashboard.SiteSettings(ctx)
	if err != nil {
		return nil, err
	}
	policy := normalizePolicy(settings)
	servers, err := p.dashboard.ListServers(ctx)
	if err != nil {
		return nil, err
	}
	enabled := make([]store.GameServer, 0, len(servers))
	for _, server := range servers {
		if server.Enabled {
			enabled = append(enabled, server)
		}
	}
	statuses := make([]store.ServerStatus, len(enabled))
	group, groupCtx := errgroup.WithContext(ctx)
	for i := range enabled {
		i := i
		group.Go(func() error {
			status, err := p.status(groupCtx, enabled[i], policy)
			if err == nil {
				statuses[i] = status
			}
			return err
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	return statuses, nil
}

func (p *Provider) LastStatus(ctx context.Context, serverID string) (store.ServerStatus, bool, error) {
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
	key := "refresh\x00" + serverCacheKey(server)
	policy := normalizePolicy(settings)
	policy.jitter = 0
	result := p.group.DoChan(key, func() (any, error) { return p.refresh(server, policy) })
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

func (p *Provider) status(ctx context.Context, server store.GameServer, policy queryPolicy) (store.ServerStatus, error) {
	now := time.Now()
	key := serverCacheKey(server)
	p.mu.RLock()
	entry, exists := p.entries[key]
	p.mu.RUnlock()
	if exists && now.Before(entry.expires) {
		return entry.status, nil
	}
	ch := p.group.DoChan(key, func() (any, error) {
		return p.refresh(server, policy)
	})
	select {
	case <-ctx.Done():
		return store.ServerStatus{}, ctx.Err()
	case result := <-ch:
		if result.Err != nil {
			return store.ServerStatus{}, result.Err
		}
		return result.Val.(store.ServerStatus), nil
	}
}

func (p *Provider) refresh(server store.GameServer, policy queryPolicy) (store.ServerStatus, error) {
	if policy.jitter > 0 {
		time.Sleep(time.Duration(rand.Int64N(int64(policy.jitter) + 1)))
	}
	p.semaphore <- struct{}{}
	defer func() { <-p.semaphore }()
	var info Info
	var players []Player
	var rules []Rule
	var latency time.Duration
	var err error
	for attempt := 0; attempt <= policy.retries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
		info, players, rules, latency, err = p.client.Query(ctx, server.Address, p.timeout)
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
			Online: false, CheckedAt: now,
		}
		if exists && !previous.status.LastSuccessAt.IsZero() && now.Sub(previous.status.LastSuccessAt) <= p.staleTTL {
			status = previous.status
			status.Online = false
			status.Stale = true
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
		LastSuccessAt: now, CheckedAt: now,
	}
	status.PlayerList = make([]store.ServerPlayer, 0, len(players))
	for _, player := range players {
		status.PlayerList = append(status.PlayerList, store.ServerPlayer{Name: player.Name, Score: player.Score, DurationSeconds: player.DurationSeconds})
	}
	status.Rules = make([]store.ServerRule, 0, len(rules))
	for _, rule := range rules {
		status.Rules = append(status.Rules, store.ServerRule{Name: rule.Name, Value: rule.Value})
	}
	p.mu.Lock()
	p.entries[key] = cacheEntry{status: status, expires: now.Add(cacheTTL)}
	p.mu.Unlock()
	return status, nil
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
