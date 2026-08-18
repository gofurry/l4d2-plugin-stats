package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	"go.uber.org/zap"
)

func TestBaiduGeoIPProviderSuccessAndErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"success", `{"status":0,"content":{"address_detail":{"country":"中国","country_code":0,"province":"上海市","city":"上海市","district":"浦东新区","adcode":"310115"},"point":{"x":"121.5","y":"31.2"}}}`, ""},
		{"provider quota", `{"status":302}`, "quota_exceeded"},
		{"provider IP whitelist", `{"status":210}`, "ip_whitelist_rejected"},
		{"malformed", `{`, "invalid_response"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("ip") != "8.8.8.8" || r.URL.Query().Get("ak") != "secret" || r.URL.Query().Get("coor") != "bd09ll" {
					t.Errorf("query=%v", r.URL.Query())
				}
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()
			provider := NewBaiduGeoIPProvider(server.Client())
			provider.Endpoint = server.URL
			entry, err := provider.Lookup(context.Background(), netip.MustParseAddr("8.8.8.8"), "secret")
			if test.want == "" {
				if err != nil || entry.City != "上海市" || entry.Longitude == nil || *entry.Longitude != 121.5 {
					t.Fatalf("entry=%+v err=%v", entry, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("err=%v want=%s", err, test.want)
			}
		})
	}
}

type countingGeoIPProvider struct {
	count int
	err   error
}

func (p *countingGeoIPProvider) Lookup(context.Context, netip.Addr, string) (store.GeoIPCacheEntry, error) {
	p.count++
	if p.err != nil {
		return store.GeoIPCacheEntry{}, p.err
	}
	return store.GeoIPCacheEntry{Country: "中国", Province: "上海市", City: "上海市", CoordinateSystem: "bd09ll", Precision: "city", Status: "resolved"}, nil
}

func TestGeoIPCacheHitExpiryAndTransientFailureTTL(t *testing.T) {
	ctx := context.Background()
	dashboard, err := store.OpenDashboard(ctx, filepath.Join(t.TempDir(), "dashboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer dashboard.Close()
	if err := dashboard.UpdateGeoIPSettings(ctx, true, "test-key", false); err != nil {
		t.Fatal(err)
	}
	config, err := dashboard.GeoIPRuntimeConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	addr := netip.MustParseAddr("8.8.8.8")
	hash := geoIPHash(config.CacheSecret, addr)
	provider := &countingGeoIPProvider{}
	service := NewGeoIPService(dashboard, nil, provider, zap.NewNop())
	service.resolveJob(ctx, geoIPJob{IP: addr, Hash: hash})
	entry, err := service.Cached(ctx, addr.String())
	if err != nil || entry == nil || entry.City != "上海市" || provider.count != 1 {
		t.Fatalf("cache hit=%+v count=%d err=%v", entry, provider.count, err)
	}
	if entry.ExpiresAt-entry.ResolvedAt != int64((30 * 24 * time.Hour).Seconds()) {
		t.Fatalf("success TTL=%d", entry.ExpiresAt-entry.ResolvedAt)
	}

	failedAddr := netip.MustParseAddr("1.1.1.1")
	failedHash := geoIPHash(config.CacheSecret, failedAddr)
	provider.err = &GeoIPProviderError{Code: "quota_exceeded"}
	service.resolveJob(ctx, geoIPJob{IP: failedAddr, Hash: failedHash})
	failed, err := dashboard.GeoIPCache(ctx, failedHash, "baidu")
	if err != nil || failed.Status != "unavailable" || failed.ErrorCode != "quota_exceeded" {
		t.Fatalf("failure cache=%+v err=%v", failed, err)
	}
	if ttl := failed.ExpiresAt - failed.ResolvedAt; ttl != int64(time.Hour.Seconds()) {
		t.Fatalf("failure TTL=%d", ttl)
	}

	expired := *entry
	expired.ExpiresAt = time.Now().Unix() - 1
	if err := dashboard.UpsertGeoIPCache(ctx, expired); err != nil {
		t.Fatal(err)
	}
	if cached, err := service.Cached(ctx, addr.String()); err != nil || cached != nil || service.count.Load() != 1 {
		t.Fatalf("expired cache was not queued: cached=%+v pending=%d err=%v", cached, service.count.Load(), err)
	}
}

func TestProviderErrorCodeDoesNotExposeDetails(t *testing.T) {
	if got := providerErrorCode(errors.New("https://api.map.baidu.com/location/ip?ak=secret&ip=8.8.8.8")); got != "provider_error" {
		t.Fatalf("provider error leaked details: %q", got)
	}
}

func TestBaiduGeoIPProviderTimeoutAndAddressPrivacy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { time.Sleep(100 * time.Millisecond) }))
	defer server.Close()
	provider := NewBaiduGeoIPProvider(&http.Client{Timeout: 10 * time.Millisecond})
	provider.Endpoint = server.URL
	if _, err := provider.Lookup(context.Background(), netip.MustParseAddr("8.8.8.8"), "secret"); err == nil || providerErrorCode(err) != "network_error" {
		t.Fatalf("timeout err=%v", err)
	}
	for _, value := range []string{"127.0.0.1", "10.0.0.1", "192.0.2.1", "2001:db8::1", "invalid"} {
		if _, ok := normalizePublicIP(value); ok {
			t.Fatalf("reserved address %s accepted", value)
		}
	}
	if addr, ok := normalizePublicIP("8.8.8.8"); !ok || addr.String() != "8.8.8.8" {
		t.Fatalf("public address rejected: %v %v", addr, ok)
	}
	if geoIPHash("one", netip.MustParseAddr("8.8.8.8")) == geoIPHash("two", netip.MustParseAddr("8.8.8.8")) {
		t.Fatal("HMAC cache keys do not depend on the stable secret")
	}
}
