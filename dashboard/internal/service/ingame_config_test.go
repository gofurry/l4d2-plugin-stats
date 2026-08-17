package service

import (
	"strings"
	"testing"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

func TestValidateIngameURL(t *testing.T) {
	allowed := []string{"", "http://example.com/x", "https://example.com/x?y=1#z"}
	for _, value := range allowed {
		if err := ValidateIngameURL(value); err != nil {
			t.Errorf("allowed URL %q: %v", value, err)
		}
	}
	rejected := []string{
		"/banner.jpg", "../x.jpg", "file:///tmp/x", "data:image/png;base64,x",
		"javascript:alert(1)", "steam://x", "ftp://example.com/x",
		"https://user:pass@example.com/", "https:///missing-host", "https://" + strings.Repeat("a", 2049),
	}
	for _, value := range rejected {
		if err := ValidateIngameURL(value); err == nil {
			t.Errorf("rejected URL accepted: %q", value)
		}
	}
}

func TestResolveIngameConfig(t *testing.T) {
	global := store.IngameSettings{
		Title: "Global", Description: "Global description", BannerURL: "https://example.com/global.jpg",
		WebsiteURL: "https://example.com", ShowAnnouncements: true, ShowPlayers: true, ShowHighlights: true,
		HighlightMetrics: [3]string{"active_play_seconds", "special_kills", "rescues"},
	}
	server := store.IngameServerSettings{
		TitleMode: "override", Title: "Server", DescriptionMode: "hidden", BannerMode: "override",
		BannerURL: "https://example.com/server.jpg", WebsiteMode: "hidden", HighlightMode: "override",
		HighlightMetrics: [3]string{"common_kills", "boss_kills", "sessions"},
	}
	resolved := ResolveIngameConfig(global, server, "Fallback")
	if resolved.Appearance.Title != "Server" || resolved.Appearance.Description != "" || resolved.Appearance.WebsiteURL != "" || resolved.Appearance.BannerURL != server.BannerURL {
		t.Fatalf("resolved appearance=%+v", resolved.Appearance)
	}
	if resolved.Metrics != server.HighlightMetrics || !resolved.Modules.ShowPlayers {
		t.Fatalf("resolved config=%+v", resolved)
	}
	global.Title = ""
	server.TitleMode = "inherit"
	if title := ResolveIngameConfig(global, server, "Fallback").Appearance.Title; title != "Fallback" {
		t.Fatalf("fallback title=%q", title)
	}
}

func TestResolveIngameDocument(t *testing.T) {
	site := store.SiteDocument{Enabled: true, ContentMarkdown: "site"}
	if content, ok := ResolveIngameDocument(store.ServerDocument{Mode: "inherit"}, site); !ok || content != "site" {
		t.Fatalf("inherit content=%q ok=%v", content, ok)
	}
	site.Enabled = false
	if _, ok := ResolveIngameDocument(store.ServerDocument{Mode: "inherit"}, site); ok {
		t.Fatal("disabled site document inherited")
	}
	if content, ok := ResolveIngameDocument(store.ServerDocument{Mode: "override", ContentMarkdown: "server"}, site); !ok || content != "server" {
		t.Fatalf("override content=%q ok=%v", content, ok)
	}
	if _, ok := ResolveIngameDocument(store.ServerDocument{Mode: "hidden"}, site); ok {
		t.Fatal("hidden document resolved")
	}
}

func TestValidateIngameCachePresets(t *testing.T) {
	valid := store.IngameSettings{
		HighlightMetrics: [3]string{"active_play_seconds", "special_kills", "rescues"},
		HomeCacheSeconds: 30, PlayerCacheSeconds: 60, RankingCacheSeconds: 120, ContentCacheSeconds: 300,
	}
	if err := ValidateIngameSettings(valid); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]int64{"home": 5, "player": 10, "ranking": 10, "content": 10} {
		invalid := valid
		switch name {
		case "home":
			invalid.HomeCacheSeconds = value
		case "player":
			invalid.PlayerCacheSeconds = value
		case "ranking":
			invalid.RankingCacheSeconds = value
		case "content":
			invalid.ContentCacheSeconds = value
		}
		if err := ValidateIngameSettings(invalid); err == nil {
			t.Errorf("accepted invalid %s preset", name)
		}
	}
}
