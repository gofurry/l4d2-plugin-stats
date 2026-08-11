package server

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

var steamID64Pattern = regexp.MustCompile(`^[0-9]{17}$`)

func validSteamID64(value string) bool { return steamID64Pattern.MatchString(value) }

func rangeCutoff(value string) (int64, error) {
	switch value {
	case "", "all":
		return 0, nil
	case "30d":
		return time.Now().AddDate(0, 0, -30).Unix(), nil
	case "90d":
		return time.Now().AddDate(0, 0, -90).Unix(), nil
	case "365d":
		return time.Now().AddDate(-1, 0, 0).Unix(), nil
	default:
		return 0, errors.New("range must be all, 30d, 90d, or 365d")
	}
}

func encodeCursor(at int64, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%d\n%s", at, id)))
}
func decodeCursor(raw string) (int64, string, error) {
	if raw == "" {
		return 0, "", nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return 0, "", errors.New("invalid cursor")
	}
	parts := strings.SplitN(string(decoded), "\n", 2)
	if len(parts) != 2 || len(parts[1]) > 128 {
		return 0, "", errors.New("invalid cursor")
	}
	at, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || at <= 0 {
		return 0, "", errors.New("invalid cursor")
	}
	return at, parts[1], nil
}

func pageLimit(raw string) (int32, error) {
	if raw == "" {
		return 20, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > 100 {
		return 0, errors.New("limit must be between 1 and 100")
	}
	return int32(n), nil
}

func validateUsername(value string) error {
	n := utf8.RuneCountInString(strings.TrimSpace(value))
	if n < 3 || n > 64 {
		return errors.New("username must contain 3 to 64 characters")
	}
	return nil
}
func validatePassword(value string) error {
	if len(value) < 12 || len(value) > 72 {
		return errors.New("password must contain 12 to 72 bytes")
	}
	return nil
}

func normalizePublicOrigin(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return "", errors.New("public_origin must be an http/https origin without a path")
	}
	u.Path = ""
	return strings.TrimSuffix(u.String(), "/"), nil
}

func normalizeHTTPURL(raw, field string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if len(raw) > 2048 {
		return "", fmt.Errorf("%s is too long", field)
	}
	u, err := url.ParseRequestURI(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
		return "", fmt.Errorf("%s must be an http or https URL", field)
	}
	return u.String(), nil
}

func validateSite(settings *store.SiteSettings) error {
	settings.Language = strings.TrimSpace(settings.Language)
	if settings.Language != "zh-CN" && settings.Language != "en" {
		return errors.New("language must be zh-CN or en")
	}
	settings.Theme = strings.TrimSpace(settings.Theme)
	if settings.Theme != "light" && settings.Theme != "dark" {
		return errors.New("theme must be light or dark")
	}
	settings.BrowserTitle = strings.TrimSpace(settings.BrowserTitle)
	if n := utf8.RuneCountInString(settings.BrowserTitle); n < 1 || n > 80 {
		return errors.New("browser_title must contain 1 to 80 characters")
	}
	if settings.A2SRefreshSeconds != 5 && settings.A2SRefreshSeconds != 10 && settings.A2SRefreshSeconds != 15 && settings.A2SRefreshSeconds != 30 && settings.A2SRefreshSeconds != 45 && settings.A2SRefreshSeconds != 60 {
		return errors.New("a2s_refresh_seconds is invalid")
	}
	if settings.A2SJitterSeconds != 2 && settings.A2SJitterSeconds != 5 {
		return errors.New("a2s_jitter_seconds is invalid")
	}
	if settings.A2SRetryCount < 1 || settings.A2SRetryCount > 3 {
		return errors.New("a2s_retry_count is invalid")
	}
	if len(settings.Links) > 20 {
		return errors.New("at most 20 footer links are allowed")
	}
	origin, err := normalizePublicOrigin(settings.PublicOrigin)
	if err != nil {
		return err
	}
	settings.PublicOrigin = origin
	if settings.SteamOpenIDProxyPort < 0 || settings.SteamOpenIDProxyPort > 65535 {
		return errors.New("steam_openid_proxy_port must be empty or between 1 and 65535")
	}
	backgroundURL, err := normalizeHTTPURL(settings.BackgroundImageURL, "background_image_url")
	if err != nil {
		return err
	}
	settings.BackgroundImageURL = backgroundURL
	settings.SEODescription = strings.TrimSpace(settings.SEODescription)
	if utf8.RuneCountInString(settings.SEODescription) > 200 {
		return errors.New("seo_description must not exceed 200 characters")
	}
	seoImageURL, err := normalizeHTTPURL(settings.SEOImageURL, "seo_image_url")
	if err != nil {
		return err
	}
	settings.SEOImageURL = seoImageURL
	if (settings.SteamOpenIDEnabled || settings.SEOEnabled) && origin == "" {
		return errors.New("public_origin is required when Steam OpenID or SEO is enabled")
	}
	if settings.SEOEnabled && settings.SEODescription == "" {
		return errors.New("seo_description is required when SEO is enabled")
	}
	for i := range settings.Links {
		link := &settings.Links[i]
		link.Label = strings.TrimSpace(link.Label)
		link.URL = strings.TrimSpace(link.URL)
		if n := utf8.RuneCountInString(link.Label); n < 1 || n > 64 {
			return fmt.Errorf("footer link %d label is invalid", i+1)
		}
		u, err := url.ParseRequestURI(link.URL)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return fmt.Errorf("footer link %d URL must use http or https", i+1)
		}
	}
	return nil
}

func validateSiteDocument(document *store.SiteDocument) error {
	document.Key = strings.TrimSpace(document.Key)
	if document.Key != "introduction" && document.Key != "commands" && document.Key != "resources" {
		return errors.New("site document key is invalid")
	}
	document.ContentMarkdown = strings.TrimSpace(document.ContentMarkdown)
	if len(document.ContentMarkdown) > 100*1024 {
		return errors.New("content_markdown must not exceed 102400 bytes")
	}
	if document.Enabled && document.ContentMarkdown == "" {
		return errors.New("content_markdown is required when the site document is enabled")
	}
	return nil
}

func validateGameServer(server *store.GameServer) error {
	server.DisplayName = strings.TrimSpace(server.DisplayName)
	server.Address = strings.TrimSpace(server.Address)
	if n := utf8.RuneCountInString(server.DisplayName); n < 1 || n > 128 {
		return errors.New("display_name must contain 1 to 128 characters")
	}
	host, port, err := net.SplitHostPort(server.Address)
	if err != nil || host == "" || port == "" {
		return errors.New("address must be host:port")
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 {
		return errors.New("address port is invalid")
	}
	return nil
}

func validateAnnouncement(value store.Announcement) error {
	if n := utf8.RuneCountInString(value.Title); n < 1 || n > 120 {
		return errors.New("title must contain 1 to 120 characters")
	}
	if n := len(value.ContentMarkdown); n < 1 || n > 100*1024 {
		return errors.New("content_markdown must contain 1 to 102400 bytes")
	}
	return nil
}
