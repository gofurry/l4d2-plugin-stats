package service

import (
	"strings"
	"testing"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

func TestValidateIngameURL(t *testing.T) {
	allowed := []string{"", "http://example.com/a.jpg", "https://example.com/a.jpg?ver=2", "https://example.com/x?y=1#z"}
	for _, value := range allowed {
		if err := ValidateIngameURL(value); err != nil {
			t.Errorf("allowed URL %q: %v", value, err)
		}
	}
	rejected := []string{
		"/a.jpg", "../a.jpg", "file:///tmp/x", "data:image/png;base64,x",
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
		Title: "Global", Description: "Global description", BannerURL: "https://example.com/global.jpg", BackgroundURL: "https://example.com/global-bg.jpg",
		WebsiteURL: "https://example.com", ShowAnnouncements: true, ShowPlayers: true, ShowHighlights: true, ShowServerIntro: true, ShowServerStatus: true,
		HighlightMetrics: [3]string{"active_play_seconds", "special_kills", "rescues"},
	}
	server := store.IngameServerSettings{
		TitleMode: "override", Title: "Server", DescriptionMode: "hidden", BannerMode: "override",
		BannerURL: "https://example.com/server.jpg", BackgroundMode: "override", BackgroundURL: "https://example.com/server-bg.jpg", WebsiteMode: "hidden", HighlightMode: "override",
		HighlightMetrics: [3]string{"common_kills", "boss_kills", "sessions"},
	}
	resolved := ResolveIngameConfig(global, server, "Fallback")
	if resolved.Appearance.Title != "Server" || resolved.Appearance.Description != "" || resolved.Appearance.WebsiteURL != "" || resolved.Appearance.BannerURL != server.BannerURL || resolved.Appearance.BackgroundURL != server.BackgroundURL {
		t.Fatalf("resolved appearance=%+v", resolved.Appearance)
	}
	if resolved.Metrics != server.HighlightMetrics || !resolved.Modules.ShowPlayers || !resolved.Modules.ShowServerIntro || !resolved.Modules.ShowServerStatus {
		t.Fatalf("resolved config=%+v", resolved)
	}
	global.Title = ""
	server.TitleMode = "inherit"
	if title := ResolveIngameConfig(global, server, "Fallback").Appearance.Title; title != "Fallback" {
		t.Fatalf("fallback title=%q", title)
	}
	server.BackgroundMode = "inherit"
	if background := ResolveIngameConfig(global, server, "Fallback").Appearance.BackgroundURL; background != global.BackgroundURL {
		t.Fatalf("inherited background=%q", background)
	}
	server.BackgroundMode = "hidden"
	if background := ResolveIngameConfig(global, server, "Fallback").Appearance.BackgroundURL; background != "" {
		t.Fatalf("hidden background=%q", background)
	}
}

func TestValidateIngameServerKey(t *testing.T) {
	for _, value := range []string{"one", "community.one", "group_01-east"} {
		if err := ValidateIngameServerKey(value); err != nil {
			t.Errorf("valid key %q: %v", value, err)
		}
	}
	for _, value := range []string{"", "change-me", "space key", "slash/key", strings.Repeat("a", 65)} {
		if err := ValidateIngameServerKey(value); err == nil {
			t.Errorf("invalid key accepted: %q", value)
		}
	}
}

func TestValidateServerQuickLinks(t *testing.T) {
	valid := []store.IngameQuickLink{
		{ServerKey: "community.one", Label: "地图合集", URL: "https://example.com/maps", SortOrder: 0, Enabled: true},
		{ServerKey: "community.one", Label: "问题反馈", URL: "http://example.com/issues", SortOrder: 1},
	}
	if err := ValidateServerQuickLinks(valid); err != nil {
		t.Fatal(err)
	}
	invalid := [][]store.IngameQuickLink{
		append(append([]store.IngameQuickLink{}, valid...), store.IngameQuickLink{ServerKey: "community.one", Label: "重复排序", URL: "https://example.com", SortOrder: 1}),
		{{ServerKey: "community.one", Label: "", URL: "https://example.com", SortOrder: 0}},
		{{ServerKey: "community.one", Label: strings.Repeat("长", 33), URL: "https://example.com", SortOrder: 0}},
		{{ServerKey: "community.one", Label: "Steam", URL: "steam://connect/127.0.0.1", SortOrder: 0}},
		{{ServerKey: "community.one", Label: "凭据", URL: "https://user:pass@example.com", SortOrder: 0}},
	}
	tooMany := make([]store.IngameQuickLink, 9)
	for index := range tooMany {
		tooMany[index] = store.IngameQuickLink{ServerKey: "community.one", Label: "Link", URL: "https://example.com", SortOrder: int64(index)}
	}
	invalid = append(invalid, tooMany)
	for index, links := range invalid {
		if err := ValidateServerQuickLinks(links); err == nil {
			t.Errorf("invalid quick-link case %d accepted: %+v", index, links)
		}
	}
}

func TestValidateIngameBackgroundSettings(t *testing.T) {
	global := store.IngameSettings{
		BackgroundURL:    "https://example.com/background.jpg?ver=2",
		HighlightMetrics: [3]string{"active_play_seconds", "special_kills", "rescues"},
		HomeCacheSeconds: 30, PlayerCacheSeconds: 60, RankingCacheSeconds: 120, ContentCacheSeconds: 300,
	}
	if err := ValidateIngameSettings(global); err != nil {
		t.Fatal(err)
	}
	global.BackgroundURL = "data:image/png;base64,x"
	if err := ValidateIngameSettings(global); err == nil {
		t.Fatal("unsafe global background URL accepted")
	}
	server := store.IngameServerSettings{
		ServerKey: "community.one",
		TitleMode: "inherit", DescriptionMode: "inherit", BannerMode: "inherit", BackgroundMode: "override",
		BackgroundURL: "https://example.com/background.jpg", WebsiteMode: "inherit", HighlightMode: "inherit",
	}
	if err := ValidateIngameServerSettings(server); err != nil {
		t.Fatal(err)
	}
	server.BackgroundURL = ""
	if err := ValidateIngameServerSettings(server); err == nil {
		t.Fatal("empty override background URL accepted")
	}
	server.BackgroundMode = "hidden"
	server.BackgroundURL = "file:///tmp/background.jpg"
	if err := ValidateIngameServerSettings(server); err == nil {
		t.Fatal("unsafe hidden background URL accepted")
	}
	server.BackgroundMode = "invalid"
	if err := ValidateIngameServerSettings(server); err == nil {
		t.Fatal("invalid background mode accepted")
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
