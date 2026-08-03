package a2s

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	steama2s "github.com/gofurry/steam-go/addons/a2s"
	"golang.org/x/sync/singleflight"
)

var ErrNoPrimaryServer = errors.New("no enabled primary server configured")

type Info struct {
	Name       string
	Map        string
	Players    int
	MaxPlayers int
	Bots       int
}

type Client interface {
	QueryInfo(context.Context, string, time.Duration) (Info, time.Duration, error)
}

type SteamClient struct{}

func (SteamClient) QueryInfo(ctx context.Context, address string, timeout time.Duration) (Info, time.Duration, error) {
	client, err := steama2s.NewClient(address, steama2s.WithTimeout(timeout))
	if err != nil {
		return Info{}, 0, err
	}
	defer client.Close()
	started := time.Now()
	result, err := client.QueryInfo(ctx)
	latency := time.Since(started)
	if err != nil {
		return Info{}, latency, err
	}
	return Info{Name: result.Name, Map: result.Map, Players: int(result.Players), MaxPlayers: int(result.MaxPlayers), Bots: int(result.Bots)}, latency, nil
}

type Provider struct {
	dashboard store.DashboardStore
	client    Client
	timeout   time.Duration
	ttl       time.Duration
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

func NewProvider(dashboard store.DashboardStore, client Client) *Provider {
	return &Provider{
		dashboard: dashboard, client: client, timeout: 2 * time.Second,
		ttl: 30 * time.Second, staleTTL: 5 * time.Minute,
		semaphore: make(chan struct{}, 4), entries: make(map[string]cacheEntry),
	}
}

func (p *Provider) PrimaryStatus(ctx context.Context) (*store.ServerStatus, error) {
	server, err := p.dashboard.PrimaryServer(ctx)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, ErrNoPrimaryServer
	}
	now := time.Now()
	p.mu.RLock()
	entry, exists := p.entries[server.QueryAddress]
	p.mu.RUnlock()
	if exists && now.Before(entry.expires) {
		status := entry.status
		return &status, nil
	}
	ch := p.group.DoChan(server.QueryAddress, func() (any, error) {
		return p.refresh(*server)
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-ch:
		if result.Err != nil {
			return nil, result.Err
		}
		status := result.Val.(store.ServerStatus)
		return &status, nil
	}
}

func (p *Provider) refresh(server store.GameServer) (store.ServerStatus, error) {
	p.semaphore <- struct{}{}
	defer func() { <-p.semaphore }()
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	info, latency, err := p.client.QueryInfo(ctx, server.QueryAddress, p.timeout)
	now := time.Now().UTC()
	if err != nil {
		p.mu.RLock()
		previous, exists := p.entries[server.QueryAddress]
		p.mu.RUnlock()
		status := store.ServerStatus{
			ConfiguredName: server.DisplayName, ConnectAddress: server.ConnectAddress,
			Online: false, CheckedAt: now,
		}
		if exists && !previous.status.LastSuccessAt.IsZero() && now.Sub(previous.status.LastSuccessAt) <= p.staleTTL {
			status = previous.status
			status.Online = false
			status.Stale = true
			status.CheckedAt = now
		}
		p.mu.Lock()
		p.entries[server.QueryAddress] = cacheEntry{status: status, expires: now.Add(p.ttl)}
		p.mu.Unlock()
		return status, nil
	}
	status := store.ServerStatus{
		ConfiguredName: server.DisplayName, ConnectAddress: server.ConnectAddress,
		Online: true, Name: info.Name, Map: info.Map, Players: info.Players,
		MaxPlayers: info.MaxPlayers, Bots: info.Bots, LatencyMS: latency.Milliseconds(),
		LastSuccessAt: now, CheckedAt: now,
	}
	p.mu.Lock()
	p.entries[server.QueryAddress] = cacheEntry{status: status, expires: now.Add(p.ttl)}
	p.mu.Unlock()
	return status, nil
}
