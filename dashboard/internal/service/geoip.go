package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	"go.uber.org/zap"
)

const (
	baiduGeoIPEndpoint          = "https://api.map.baidu.com/location/ip"
	connectionLocationBatchSize = 200
	connectionLocationScanLimit = 2000
)

type GeoIPProvider interface {
	Lookup(context.Context, netip.Addr, string) (store.GeoIPCacheEntry, error)
}

type GeoIPProviderError struct {
	Code string
}

func (e *GeoIPProviderError) Error() string { return e.Code }

type BaiduGeoIPProvider struct {
	Client   *http.Client
	Endpoint string
}

func NewBaiduGeoIPProvider(client *http.Client) *BaiduGeoIPProvider {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &BaiduGeoIPProvider{Client: client, Endpoint: baiduGeoIPEndpoint}
}

func (p *BaiduGeoIPProvider) Lookup(ctx context.Context, addr netip.Addr, apiKey string) (store.GeoIPCacheEntry, error) {
	endpoint := p.Endpoint
	if endpoint == "" {
		endpoint = baiduGeoIPEndpoint
	}
	values := url.Values{"ip": {addr.String()}, "ak": {apiKey}, "coor": {"bd09ll"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+values.Encode(), nil)
	if err != nil {
		return store.GeoIPCacheEntry{}, &GeoIPProviderError{Code: "request_invalid"}
	}
	response, err := p.Client.Do(req)
	if err != nil {
		return store.GeoIPCacheEntry{}, &GeoIPProviderError{Code: "network_error"}
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return store.GeoIPCacheEntry{}, &GeoIPProviderError{Code: "http_" + strconv.Itoa(response.StatusCode)}
	}
	var body struct {
		Status  int `json:"status"`
		Content struct {
			AddressDetail struct {
				Country     string `json:"country"`
				CountryCode any    `json:"country_code"`
				Province    string `json:"province"`
				City        string `json:"city"`
				District    string `json:"district"`
				Adcode      any    `json:"adcode"`
			} `json:"address_detail"`
			Point struct {
				X any `json:"x"`
				Y any `json:"y"`
			} `json:"point"`
		} `json:"content"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(nil, response.Body, 1<<20))
	if err := decoder.Decode(&body); err != nil {
		return store.GeoIPCacheEntry{}, &GeoIPProviderError{Code: "invalid_response"}
	}
	if body.Status != 0 {
		return store.GeoIPCacheEntry{}, &GeoIPProviderError{Code: baiduStatusCode(body.Status)}
	}
	if strings.TrimSpace(body.Content.AddressDetail.City) == "" && strings.TrimSpace(body.Content.AddressDetail.Country) == "" {
		return store.GeoIPCacheEntry{}, &GeoIPProviderError{Code: "unsupported_location"}
	}
	return store.GeoIPCacheEntry{
		Provider: "baidu", Country: strings.TrimSpace(body.Content.AddressDetail.Country),
		CountryCode: scalarString(body.Content.AddressDetail.CountryCode), Province: strings.TrimSpace(body.Content.AddressDetail.Province),
		City: strings.TrimSpace(body.Content.AddressDetail.City), District: strings.TrimSpace(body.Content.AddressDetail.District),
		Adcode: scalarString(body.Content.AddressDetail.Adcode), Longitude: scalarFloat(body.Content.Point.X), Latitude: scalarFloat(body.Content.Point.Y),
		CoordinateSystem: "bd09ll", Precision: "city", Status: "resolved",
	}, nil
}

func baiduStatusCode(status int) string {
	switch status {
	case 3, 5, 101, 102, 200:
		return "credentials_or_permission_rejected"
	case 210:
		return "ip_whitelist_rejected"
	case 4, 301, 302:
		return "quota_exceeded"
	default:
		return "provider_status_" + strconv.Itoa(status)
	}
}

func scalarString(value any) string {
	switch value := value.(type) {
	case string:
		return strings.TrimSpace(value)
	case float64:
		return strconv.FormatInt(int64(value), 10)
	default:
		return ""
	}
}

func scalarFloat(value any) *float64 {
	var result float64
	switch value := value.(type) {
	case string:
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil
		}
		result = parsed
	case float64:
		result = value
	default:
		return nil
	}
	return &result
}

type geoIPJob struct {
	IP   netip.Addr
	Hash string
}

type GeoIPService struct {
	dashboard store.DashboardAuditStore
	stats     store.StatsChatAuditStore
	provider  GeoIPProvider
	logger    *zap.Logger
	queue     chan geoIPJob
	pending   sync.Map
	count     atomic.Int64
	rateMu    sync.Mutex
	nextCall  time.Time
}

func NewGeoIPService(dashboard store.DashboardAuditStore, stats store.StatsChatAuditStore, provider GeoIPProvider, logger *zap.Logger) *GeoIPService {
	if provider == nil {
		provider = NewBaiduGeoIPProvider(nil)
	}
	return &GeoIPService{dashboard: dashboard, stats: stats, provider: provider, logger: logger, queue: make(chan geoIPJob, 256)}
}

func (s *GeoIPService) Run(ctx context.Context) {
	go s.worker(ctx)
	ticker := time.NewTicker(5 * time.Minute)
	cleanupTicker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	defer cleanupTicker.Stop()
	s.enqueueRecent(ctx)
	s.deleteExpiredCache(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.enqueueRecent(ctx)
		case <-cleanupTicker.C:
			s.deleteExpiredCache(ctx)
		}
	}
}

func (s *GeoIPService) enqueueRecent(ctx context.Context) {
	config, err := s.dashboard.GeoIPRuntimeConfig(ctx)
	if err != nil || config.APIKey == "" {
		return
	}
	page, err := s.stats.ConnectionAudit(ctx, store.ConnectionAuditFilter{From: time.Now().Add(-24 * time.Hour).Unix(), Limit: 100})
	if err != nil {
		return
	}
	for _, row := range page.Items {
		s.Enqueue(ctx, row.IPAddress)
	}
}

func (s *GeoIPService) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-s.queue:
			s.resolveJob(ctx, job)
			s.pending.Delete(job.Hash)
			s.count.Add(-1)
		}
	}
}

func (s *GeoIPService) resolveJob(ctx context.Context, job geoIPJob) {
	config, err := s.dashboard.GeoIPRuntimeConfig(ctx)
	if err != nil || config.APIKey == "" {
		return
	}
	lookupCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	entry, err := s.lookup(lookupCtx, job.IP, config)
	now := time.Now().Unix()
	if err != nil {
		code := providerErrorCode(err)
		if job.IP.Is6() && (strings.HasPrefix(code, "provider_status_") || code == "credentials_or_permission_rejected" || code == "ip_whitelist_rejected") {
			code = "ipv6_not_authorized_or_unsupported"
		}
		entry = store.GeoIPCacheEntry{
			IPHash: job.Hash, Provider: "baidu", CoordinateSystem: "bd09ll", Precision: "city",
			Status: "unavailable", ErrorCode: code, ResolvedAt: now, ExpiresAt: now + int64(time.Hour.Seconds()),
		}
		_ = s.dashboard.UpsertGeoIPCache(ctx, entry)
		status := store.GeoIPRuntimeStatus{LastSuccessAt: config.LastSuccessAt, LastErrorAt: now, LastErrorCode: code, IPv4Status: config.IPv4Status, IPv6Status: config.IPv6Status}
		if job.IP.Is4() {
			status.IPv4Status = code
		} else {
			status.IPv6Status = code
		}
		_ = s.dashboard.UpdateGeoIPRuntimeStatus(ctx, status)
		s.logger.Warn("GeoIP provider lookup unavailable", zap.String("error_code", code))
		return
	}
	entry.IPHash, entry.Provider = job.Hash, "baidu"
	entry.ResolvedAt, entry.ExpiresAt = now, now+int64((30*24*time.Hour).Seconds())
	if err := s.dashboard.UpsertGeoIPCache(ctx, entry); err != nil {
		s.logger.Warn("GeoIP cache update unavailable", zap.String("error_code", "cache_write_failed"))
		return
	}
	status := store.GeoIPRuntimeStatus{LastSuccessAt: now, LastErrorAt: config.LastErrorAt, LastErrorCode: config.LastErrorCode, IPv4Status: config.IPv4Status, IPv6Status: config.IPv6Status}
	if job.IP.Is4() {
		status.IPv4Status = "working"
	} else {
		status.IPv6Status = "working"
	}
	_ = s.dashboard.UpdateGeoIPRuntimeStatus(ctx, status)
}

func (s *GeoIPService) lookup(ctx context.Context, addr netip.Addr, config store.GeoIPRuntimeConfig) (store.GeoIPCacheEntry, error) {
	if err := s.waitForProvider(ctx, config.QPSLimit); err != nil {
		return store.GeoIPCacheEntry{}, err
	}
	return s.provider.Lookup(ctx, addr, config.APIKey)
}

func (s *GeoIPService) waitForProvider(ctx context.Context, qpsLimit int64) error {
	if qpsLimit < 1 || qpsLimit > 3 {
		qpsLimit = 2
	}
	interval := time.Second / time.Duration(qpsLimit)
	now := time.Now()
	s.rateMu.Lock()
	permitAt := now
	if s.nextCall.After(now) {
		permitAt = s.nextCall
	}
	s.nextCall = permitAt.Add(interval)
	s.rateMu.Unlock()
	if wait := time.Until(permitAt); wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	return nil
}

func (s *GeoIPService) deleteExpiredCache(ctx context.Context) {
	const batchSize int64 = 500
	for batch := 0; batch < 8; batch++ {
		deleted, err := s.dashboard.DeleteExpiredGeoIPCache(ctx, time.Now().Unix(), batchSize)
		if err != nil || deleted < batchSize {
			return
		}
	}
}

func providerErrorCode(err error) string {
	var providerErr *GeoIPProviderError
	if errors.As(err, &providerErr) {
		return providerErr.Code
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	return "provider_error"
}

func (s *GeoIPService) Enqueue(ctx context.Context, rawIP string) {
	config, err := s.dashboard.GeoIPRuntimeConfig(ctx)
	if err != nil || config.APIKey == "" {
		return
	}
	addr, ok := normalizePublicIP(rawIP)
	if !ok {
		return
	}
	hash := geoIPHash(config.CacheSecret, addr)
	if cached, err := s.dashboard.GeoIPCache(ctx, hash, "baidu"); err == nil && cached.ExpiresAt > time.Now().Unix() {
		return
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return
	}
	if _, loaded := s.pending.LoadOrStore(hash, struct{}{}); loaded {
		return
	}
	select {
	case s.queue <- geoIPJob{IP: addr, Hash: hash}:
		s.count.Add(1)
	default:
		s.pending.Delete(hash)
	}
}

func (s *GeoIPService) Cached(ctx context.Context, rawIP string) (*store.GeoIPCacheEntry, error) {
	config, err := s.dashboard.GeoIPRuntimeConfig(ctx)
	if err != nil {
		return nil, err
	}
	addr, ok := normalizePublicIP(rawIP)
	if !ok {
		return &store.GeoIPCacheEntry{Provider: "local", Status: "private_or_reserved"}, nil
	}
	hash := geoIPHash(config.CacheSecret, addr)
	entry, err := s.dashboard.GeoIPCache(ctx, hash, "baidu")
	if errors.Is(err, sql.ErrNoRows) || (err == nil && entry.ExpiresAt <= time.Now().Unix()) {
		s.Enqueue(ctx, rawIP)
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (s *GeoIPService) Settings(ctx context.Context) (store.GeoIPSettings, error) {
	return s.dashboard.GeoIPSettings(ctx, s.count.Load())
}

func (s *GeoIPService) UpdateSettings(ctx context.Context, newKey string, clearKey bool, qpsLimit int64) error {
	return s.dashboard.UpdateGeoIPSettings(ctx, newKey, clearKey, qpsLimit)
}

func (s *GeoIPService) Test(ctx context.Context, rawIP string) (store.GeoIPCacheEntry, error) {
	config, err := s.dashboard.GeoIPRuntimeConfig(ctx)
	if err != nil {
		return store.GeoIPCacheEntry{}, err
	}
	if config.APIKey == "" {
		return store.GeoIPCacheEntry{}, errors.New("Baidu API key is not configured")
	}
	addr, ok := normalizePublicIP(rawIP)
	if !ok {
		return store.GeoIPCacheEntry{}, errors.New("test IP must be a public IPv4 or IPv6 address")
	}
	lookupCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	return s.lookup(lookupCtx, addr, config)
}

func (s *GeoIPService) Connections(ctx context.Context, filter store.ConnectionAuditFilter) (store.ConnectionAuditPage, error) {
	if strings.TrimSpace(filter.Location) != "" {
		return s.connectionsByLocation(ctx, filter)
	}
	page, err := s.stats.ConnectionAudit(ctx, filter)
	if err != nil {
		return page, err
	}
	filtered := make([]store.ConnectionAuditRow, 0, len(page.Items))
	for index := range page.Items {
		entry, cacheErr := s.Cached(ctx, page.Items[index].IPAddress)
		if cacheErr == nil {
			page.Items[index].GeoIP = entry
		}
		if filter.Location == "" || entryMatchesLocation(entry, filter.Location) {
			filtered = append(filtered, page.Items[index])
		}
	}
	page.Items = filtered
	return page, nil
}

func (s *GeoIPService) connectionsByLocation(ctx context.Context, filter store.ConnectionAuditFilter) (store.ConnectionAuditPage, error) {
	if filter.Limit < 1 || filter.Limit > 200 {
		filter.Limit = 100
	}
	location := strings.TrimSpace(filter.Location)
	scan := filter
	scan.Location = ""
	scan.Limit = connectionLocationBatchSize
	result := store.ConnectionAuditPage{Items: make([]store.ConnectionAuditRow, 0, filter.Limit)}
	scanned := 0
	for scanned < connectionLocationScanLimit {
		page, err := s.stats.ConnectionAudit(ctx, scan)
		if err != nil {
			return result, err
		}
		if len(page.Items) == 0 {
			result.NextCursorAt, result.NextCursorID = page.NextCursorAt, page.NextCursorID
			return result, nil
		}
		for index := range page.Items {
			row := page.Items[index]
			entry, cacheErr := s.Cached(ctx, row.IPAddress)
			if cacheErr == nil {
				row.GeoIP = entry
				if entry == nil {
					result.LocationPending = true
				}
			}
			scanned++
			if entryMatchesLocation(entry, location) {
				result.Items = append(result.Items, row)
			}
			moreInBatch := index+1 < len(page.Items)
			moreRawRows := moreInBatch || page.NextCursorID != ""
			if len(result.Items) == filter.Limit {
				if moreRawRows {
					result.NextCursorAt, result.NextCursorID = row.StartedAt, row.SessionID
				}
				return result, nil
			}
			if scanned == connectionLocationScanLimit {
				if moreRawRows {
					result.NextCursorAt, result.NextCursorID = row.StartedAt, row.SessionID
				}
				return result, nil
			}
		}
		if page.NextCursorID == "" {
			return result, nil
		}
		scan.CursorAt, scan.CursorID = page.NextCursorAt, page.NextCursorID
	}
	return result, nil
}

func entryMatchesLocation(entry *store.GeoIPCacheEntry, query string) bool {
	if entry == nil {
		return false
	}
	haystack := strings.ToLower(strings.Join([]string{entry.Country, entry.Province, entry.City, entry.District, entry.Adcode}, " "))
	return strings.Contains(haystack, strings.ToLower(strings.TrimSpace(query)))
}

func normalizePublicIP(raw string) (netip.Addr, bool) {
	addr, err := netip.ParseAddr(strings.TrimSpace(raw))
	if err != nil {
		return netip.Addr{}, false
	}
	addr = addr.Unmap()
	if !addr.IsGlobalUnicast() || addr.IsPrivate() || addr.IsLoopback() || addr.IsLinkLocalUnicast() {
		return netip.Addr{}, false
	}
	for _, prefix := range reservedIPPrefixes {
		if prefix.Contains(addr) {
			return netip.Addr{}, false
		}
	}
	return addr, true
}

var reservedIPPrefixes = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"), netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"), netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("2001:db8::/32"),
}

func geoIPHash(secret string, addr netip.Addr) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(addr.String()))
	return hex.EncodeToString(mac.Sum(nil))
}
