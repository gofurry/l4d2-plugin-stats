package store

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	dashsql "github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store/sqlcgen/dashboard"
)

func (s *dashboardStore) ChatAuditSettings(ctx context.Context) (ChatAuditSettings, error) {
	row, err := s.q.GetChatAuditSettings(ctx)
	if err != nil {
		return ChatAuditSettings{}, fmt.Errorf("get chat audit settings: %w", err)
	}
	return ChatAuditSettings{
		Enabled: row.Enabled != 0, RetentionDays: row.RetentionDays,
		LastCleanupAt: row.LastCleanupAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

func (s *dashboardStore) UpdateChatAuditSettings(ctx context.Context, settings ChatAuditSettings) error {
	if !validChatRetention(settings.RetentionDays) {
		return fmt.Errorf("unsupported chat retention: %d days", settings.RetentionDays)
	}
	return s.q.UpdateChatAuditSettings(ctx, dashsql.UpdateChatAuditSettingsParams{
		Enabled: boolInt64(settings.Enabled), RetentionDays: settings.RetentionDays,
		UpdatedAt: time.Now().Unix(),
	})
}

func validChatRetention(days int64) bool {
	switch days {
	case 0, 7, 14, 30, 60, 90, 180, 365:
		return true
	default:
		return false
	}
}

func (s *dashboardStore) MarkChatAuditCleanup(ctx context.Context, at int64) error {
	return s.q.MarkChatAuditCleanup(ctx, at)
}

func (s *dashboardStore) RecordChatExport(ctx context.Context, entry ChatExportAuditEntry) error {
	if entry.OutputFormat != "csv" && entry.OutputFormat != "jsonl" {
		return fmt.Errorf("unsupported chat export format %q", entry.OutputFormat)
	}
	var rowCount any
	if entry.RowCount != nil {
		rowCount = *entry.RowCount
	}
	return s.q.CreateChatExportAudit(ctx, dashsql.CreateChatExportAuditParams{
		ExportID: entry.ExportID, ExportedAt: entry.ExportedAt, AdminIdentity: entry.AdminIdentity,
		OutputFormat: entry.OutputFormat, FilterSummary: entry.FilterSummary, RowCount: rowCount,
		Completed: boolInt64(entry.Completed),
	})
}

func (s *dashboardStore) GeoIPRuntimeConfig(ctx context.Context) (GeoIPRuntimeConfig, error) {
	row, err := s.q.GetGeoIPSettings(ctx)
	if err != nil {
		return GeoIPRuntimeConfig{}, fmt.Errorf("get GeoIP settings: %w", err)
	}
	return GeoIPRuntimeConfig{
		Enabled: row.Enabled != 0, Provider: row.Provider, APIKey: row.ApiKey,
		CacheSecret: row.CacheSecret, LastSuccessAt: row.LastSuccessAt,
		LastErrorAt: row.LastErrorAt, LastErrorCode: row.LastErrorCode,
		IPv4Status: row.Ipv4Status, IPv6Status: row.Ipv6Status, UpdatedAt: row.UpdatedAt,
	}, nil
}

// ensureGeoIPCacheSecret creates the installation-local HMAC secret during
// database initialization. Read-only paths such as doctor never mutate state.
func (s *dashboardStore) ensureGeoIPCacheSecret(ctx context.Context) error {
	row, err := s.q.GetGeoIPSettings(ctx)
	if err != nil {
		return fmt.Errorf("get GeoIP settings: %w", err)
	}
	if row.CacheSecret != "" {
		return nil
	}
	secret, err := randomSecret(32)
	if err != nil {
		return fmt.Errorf("generate GeoIP cache secret: %w", err)
	}
	if err := s.q.UpdateGeoIPSettings(ctx, dashsql.UpdateGeoIPSettingsParams{
		Enabled: row.Enabled, Provider: row.Provider, ApiKey: row.ApiKey,
		CacheSecret: secret, UpdatedAt: time.Now().Unix(),
	}); err != nil {
		return fmt.Errorf("save GeoIP cache secret: %w", err)
	}
	return nil
}

func (s *dashboardStore) GeoIPSettings(ctx context.Context, pending int64) (GeoIPSettings, error) {
	config, err := s.GeoIPRuntimeConfig(ctx)
	if err != nil {
		return GeoIPSettings{}, err
	}
	count, err := s.GeoIPCacheCount(ctx)
	if err != nil {
		return GeoIPSettings{}, fmt.Errorf("count GeoIP cache: %w", err)
	}
	return GeoIPSettings{
		Enabled: config.Enabled, Provider: config.Provider,
		APIKeySet: config.APIKey != "", APIKeyMasked: maskSecret(config.APIKey),
		LastSuccessAt: config.LastSuccessAt, LastErrorAt: config.LastErrorAt,
		LastErrorCode: config.LastErrorCode, IPv4Status: config.IPv4Status,
		IPv6Status: config.IPv6Status, CacheCount: count, PendingCount: pending,
		UpdatedAt: config.UpdatedAt,
	}, nil
}

// UpdateGeoIPSettings preserves the existing key when apiKey is empty. A key
// can only be removed through the explicit clearKey flag.
func (s *dashboardStore) UpdateGeoIPSettings(ctx context.Context, enabled bool, apiKey string, clearKey bool) error {
	current, err := s.GeoIPRuntimeConfig(ctx)
	if err != nil {
		return err
	}
	apiKey = strings.TrimSpace(apiKey)
	key := current.APIKey
	if clearKey {
		key = ""
	} else if apiKey != "" {
		if len(apiKey) > 256 {
			return errors.New("GeoIP API key is too long")
		}
		key = apiKey
	}
	if enabled && key == "" {
		return errors.New("GeoIP cannot be enabled without a Baidu API key")
	}
	return s.q.UpdateGeoIPSettings(ctx, dashsql.UpdateGeoIPSettingsParams{
		Enabled: boolInt64(enabled), Provider: "baidu", ApiKey: key,
		CacheSecret: current.CacheSecret, UpdatedAt: time.Now().Unix(),
	})
}

func (s *dashboardStore) UpdateGeoIPRuntimeStatus(ctx context.Context, status GeoIPRuntimeStatus) error {
	return s.q.UpdateGeoIPRuntimeStatus(ctx, dashsql.UpdateGeoIPRuntimeStatusParams{
		LastSuccessAt: status.LastSuccessAt, LastErrorAt: status.LastErrorAt,
		LastErrorCode: status.LastErrorCode, Ipv4Status: status.IPv4Status, Ipv6Status: status.IPv6Status,
	})
}

func (s *dashboardStore) GeoIPCache(ctx context.Context, hash, provider string) (GeoIPCacheEntry, error) {
	row, err := s.q.GetGeoIPCache(ctx, dashsql.GetGeoIPCacheParams{IpHash: hash, Provider: provider})
	if err != nil {
		return GeoIPCacheEntry{}, err
	}
	return GeoIPCacheEntry{
		IPHash: row.IpHash, Provider: row.Provider, Country: row.Country, CountryCode: row.CountryCode,
		Province: row.Province, City: row.City, District: row.District, Adcode: row.Adcode,
		Longitude: interfaceFloat(row.Longitude), Latitude: interfaceFloat(row.Latitude),
		CoordinateSystem: row.CoordinateSystem, Precision: row.Precision, Status: row.Status,
		ErrorCode: row.ErrorCode, ResolvedAt: row.ResolvedAt, ExpiresAt: row.ExpiresAt,
	}, nil
}

func (s *dashboardStore) UpsertGeoIPCache(ctx context.Context, entry GeoIPCacheEntry) error {
	return s.q.UpsertGeoIPCache(ctx, dashsql.UpsertGeoIPCacheParams{
		IpHash: entry.IPHash, Provider: entry.Provider, Country: entry.Country,
		CountryCode: entry.CountryCode, Province: entry.Province, City: entry.City,
		District: entry.District, Adcode: entry.Adcode, Longitude: pointerValue(entry.Longitude),
		Latitude: pointerValue(entry.Latitude), CoordinateSystem: entry.CoordinateSystem,
		Precision: entry.Precision, Status: entry.Status, ErrorCode: entry.ErrorCode,
		ResolvedAt: entry.ResolvedAt, ExpiresAt: entry.ExpiresAt,
	})
}

func (s *dashboardStore) GeoIPCacheCount(ctx context.Context) (int64, error) {
	return s.q.CountGeoIPCache(ctx)
}

func randomSecret(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
	for i := range value {
		value[i] = alphabet[int(value[i])%len(alphabet)]
	}
	return string(value), nil
}

func maskSecret(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 4 {
		return "****"
	}
	return "****" + value[len(value)-4:]
}

func interfaceFloat(value any) *float64 {
	switch value := value.(type) {
	case float64:
		return &value
	case int64:
		result := float64(value)
		return &result
	case []byte:
		var result float64
		if _, err := fmt.Sscan(string(value), &result); err == nil {
			return &result
		}
	}
	return nil
}

func pointerValue(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}
