package ingame

import (
	"bytes"
	"strings"
	"testing"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/service"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

func TestIngameAssetsMeetLegacyBudgets(t *testing.T) {
	css, err := CSS()
	if err != nil {
		t.Fatal(err)
	}
	if len(css) >= 20*1024 {
		t.Fatalf("CSS size=%d, budget < 20 KiB", len(css))
	}
	atlas, err := AchievementAtlas()
	if err != nil {
		t.Fatal(err)
	}
	if len(atlas) >= 250*1024 {
		t.Fatalf("atlas size=%d, budget < 250 KiB", len(atlas))
	}
	if len(atlas) < 8 || !bytes.Equal(atlas[:8], []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}) {
		t.Fatal("achievement atlas is not PNG")
	}
}

func TestIngameTemplatesContainNoClientApplication(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatal(err)
	}
	base := service.IngameBaseView{
		ServerKey: "main", Config: service.ResolvedIngameConfig{Appearance: service.ResolvedIngameAppearance{Title: "Main"}},
		WebsiteHref: "steam://openurl_external/https://example.com/full",
		Status:      store.ServerStatus{Online: true, Map: "c1m1_hotel", Players: 2, MaxPlayers: 8},
	}
	views := []struct {
		name string
		data any
	}{
		{name: "home.html", data: service.IngameHomeView{IngameBaseView: base}},
		{name: "player.html", data: service.IngamePlayerView{IngameBaseView: base, PlayerName: "Player", Achievements: &service.CompactAchievementOverview{Badges: []service.AchievementBadge{{ArtworkKey: "career.veteran", Title: "Veteran"}}}}},
		{name: "rankings.html", data: service.IngameRankingView{IngameBaseView: base, Metric: service.IngameMetricCatalog()[0], Catalog: service.IngameMetricCatalog(), Page: store.RankingPage{Items: []store.RankingEntry{}}, PageNumber: 1}},
		{name: "info.html", data: service.IngameInfoView{IngameBaseView: base, Title: "Info", ContentMarkdown: "# Heading\n<script>alert(1)</script>\n![remote](https://example.com/a.png)\n[bad](javascript:alert(1))\n[good](https://example.com/)"}},
		{name: "announcement.html", data: service.IngameAnnouncementView{IngameBaseView: base, Announcement: store.Announcement{Title: "News", ContentMarkdown: "**Safe**"}}},
	}
	for _, test := range views {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := renderer.Render(&output, test.name, test.data); err != nil {
				t.Fatal(err)
			}
			html := output.String()
			if strings.Contains(html, "#ZgotmplZ") || !strings.Contains(html, `href="steam://openurl_external/https://example.com/full"`) {
				t.Fatalf("external browser URL was not rendered safely: %s", html)
			}
			if len(html) >= 40*1024 {
				t.Fatalf("HTML size=%d, budget < 40 KiB", len(html))
			}
			for _, forbidden := range []string{"<script", "type=module", "fetch(", "XMLHttpRequest", "react", "<img src=\"https://example.com/a.png", "javascript:"} {
				if strings.Contains(strings.ToLower(html), strings.ToLower(forbidden)) {
					t.Fatalf("rendered forbidden client content %q: %s", forbidden, html)
				}
			}
		})
	}
}

func TestMarkdownRendererSanitizesLinksAndRawHTML(t *testing.T) {
	html := string(renderMarkdown("<iframe src=x></iframe>\n- **safe**\n[bad](file:///tmp/x)\n[good](https://example.com/x)"))
	if strings.Contains(html, "iframe") || strings.Contains(html, "file:") || !strings.Contains(html, `<a href="https://example.com/x">good</a>`) {
		t.Fatalf("sanitized markdown=%s", html)
	}
}
