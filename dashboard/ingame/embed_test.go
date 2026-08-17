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
	if len(css) >= 25*1024 {
		t.Fatalf("CSS size=%d, budget < 25 KiB", len(css))
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

func TestVisualV2CSSRemainsLegacySafe(t *testing.T) {
	cssBytes, err := CSS()
	if err != nil {
		t.Fatal(err)
	}
	css := strings.ToLower(string(cssBytes))
	for _, required := range []string{"position:fixed", "background-position:center center", "background-size:cover", ".page{position:relative", ".home-banner img{display:block;width:100%;height:auto", "::-webkit-scrollbar", "-webkit-border-radius", "-webkit-box-shadow"} {
		if !strings.Contains(css, required) {
			t.Errorf("Visual v2 CSS missing %q", required)
		}
	}
	for _, forbidden := range []string{"display:flex", "display:grid", "gap:", "var(", "backdrop-filter", "position:sticky", "min-width:700", "transition:", "animation:"} {
		if strings.Contains(css, forbidden) {
			t.Errorf("legacy CSS contains forbidden dependency %q", forbidden)
		}
	}
}

func TestIngameTemplatesContainNoClientApplication(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatal(err)
	}
	base := service.IngameBaseView{
		ServerKey: "main", Config: service.ResolvedIngameConfig{Appearance: service.ResolvedIngameAppearance{Title: "Main", BackgroundURL: "https://example.com/background.jpg?ver=2"}},
		WebsiteHref:     "steam://openurl_external/https://example.com/full",
		OnlineInstances: 1, TotalInstances: 1, OnlinePlayerCount: 2,
	}
	views := []struct {
		name string
		data any
	}{
		{name: "home.html", data: service.IngameHomeView{IngameBaseView: withActivePage(base, "home")}},
		{name: "player.html", data: service.IngamePlayerView{IngameBaseView: withActivePage(base, "player"), PlayerName: "Player", Achievements: &service.CompactAchievementOverview{Badges: []service.AchievementBadge{{ArtworkKey: "career.veteran", Title: "Veteran"}}}}},
		{name: "rankings.html", data: service.IngameRankingView{IngameBaseView: withActivePage(base, "rankings"), Metric: service.IngameMetricCatalog()[0], Catalog: service.IngameMetricCatalog(), Page: store.RankingPage{Items: []store.RankingEntry{}}, PageNumber: 1}},
		{name: "info.html", data: service.IngameInfoView{IngameBaseView: withActivePage(base, "introduction"), Title: "Info", ContentMarkdown: "# Heading\n<script>alert(1)</script>\n![remote](https://example.com/a.png)\n[bad](javascript:alert(1))\n[good](https://example.com/)"}},
		{name: "announcement.html", data: service.IngameAnnouncementView{IngameBaseView: withActivePage(base, "announcement"), Announcement: store.Announcement{Title: "News", ContentMarkdown: "**Safe**"}}},
	}
	for _, test := range views {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := renderer.Render(&output, test.name, test.data); err != nil {
				t.Fatal(err)
			}
			html := output.String()
			t.Logf("rendered HTML bytes=%d", len(html))
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
	connect := renderTemplate(t, renderer, "connect.html", service.IngameConnectView{ConnectHref: "steam://connect/127.0.0.1:27015"})
	if strings.Contains(strings.ToLower(connect), "<script") || !strings.Contains(connect, `http-equiv="refresh"`) || !strings.Contains(connect, "steam://connect/127.0.0.1:27015") {
		t.Fatalf("connect redirect is not a zero-JS Steam redirect: %s", connect)
	}
}

func TestMarkdownRendererSanitizesLinksAndRawHTML(t *testing.T) {
	html := string(renderMarkdown("<iframe src=x></iframe>\n- **safe**\n[bad](file:///tmp/x)\n[steam](steam://run/550)\n[good](https://example.com/x)"))
	if strings.Contains(html, "iframe") || strings.Contains(html, "file:") || strings.Contains(html, "steam://run") || !strings.Contains(html, `<a href="steam://openurl_external/https://example.com/x">good</a>`) {
		t.Fatalf("sanitized markdown=%s", html)
	}
}

func TestAssetFingerprintTracksEmbeddedContent(t *testing.T) {
	css, err := CSS()
	if err != nil {
		t.Fatal(err)
	}
	atlas, err := AchievementAtlas()
	if err != nil {
		t.Fatal(err)
	}
	fingerprint := AssetFingerprint()
	if len(fingerprint) != 12 || fingerprint == "v1.3.4" {
		t.Fatalf("asset fingerprint=%q", fingerprint)
	}
	if fingerprint != fingerprintAssets(css, atlas) {
		t.Fatal("asset fingerprint does not match embedded content")
	}
	if fingerprintAssets([]byte("same")) != fingerprintAssets([]byte("same")) {
		t.Fatal("identical content produced unstable fingerprints")
	}
	if fingerprintAssets([]byte("before")) == fingerprintAssets([]byte("after")) {
		t.Fatal("changed content kept the same fingerprint")
	}
}

func TestVisualV2ShellsNavigationAndBackground(t *testing.T) {
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatal(err)
	}
	base := service.IngameBaseView{
		ServerKey: "main-key", ActivePage: "home",
		Config: service.ResolvedIngameConfig{
			Appearance: service.ResolvedIngameAppearance{Title: "Main", Description: "Description", BannerURL: "https://example.com/banner.jpg", BackgroundURL: "https://example.com/background.jpg?ver=2"},
			Modules:    service.ResolvedIngameModules{ShowPlayers: true, ShowServerIntro: true, ShowServerStatus: true},
		},
		OnlineInstances: 1, TotalInstances: 1, OnlinePlayerCount: 2,
		Instances: []service.IngameServerInstance{{DisplayName: "Main #1", Address: "127.0.0.1:27015", Online: true, Map: "c1m1_hotel", Players: 2, MaxPlayers: 8, JoinHref: "steam://openurl_external/https://stats.example.com/ingame/connect?instance=server-id&server=main-key"}},
		Documents: []service.IngameDocumentLink{{Key: "commands", Label: "常用命令"}},
	}
	home := renderTemplate(t, renderer, "home.html", service.IngameHomeView{IngameBaseView: base})
	for _, expected := range []string{`class="home-banner"`, `class="home-intro"`, `class="panel server-navigation-card home-navigation-card"`, `href="#players"`, `class="page-background"`, `background-image:url(&#34;https://example.com/background.jpg?ver=2&#34;)`, `href="steam://openurl_external/https://stats.example.com/ingame/connect?instance=server-id&amp;server=main-key"`, "/ingame/assets/" + AssetFingerprint() + "/ingame.css"} {
		if !strings.Contains(home, expected) {
			t.Fatalf("home missing %q: %s", expected, home)
		}
	}
	if strings.Contains(home, `>服务器首页</a>`) {
		t.Fatalf("home contains redundant server-home navigation: %s", home)
	}
	if strings.Contains(home, "hero-overlay") {
		t.Fatalf("title card is still overlaid on Banner: %s", home)
	}

	noBanner := base
	noBanner.Config.Appearance.BannerURL = ""
	home = renderTemplate(t, renderer, "home.html", service.IngameHomeView{IngameBaseView: noBanner})
	if strings.Contains(home, `class="home-banner"`) || !strings.Contains(home, `class="home-intro"`) {
		t.Fatalf("no-banner home left an empty banner region: %s", home)
	}

	hiddenIntro := base
	hiddenIntro.Config.Modules.ShowServerIntro = false
	home = renderTemplate(t, renderer, "home.html", service.IngameHomeView{IngameBaseView: hiddenIntro})
	if !strings.Contains(home, `class="home-banner"`) || strings.Contains(home, `class="home-intro"`) || !strings.Contains(home, `class="nav"`) || strings.Contains(home, `class="panel-divider"`) {
		t.Fatalf("intro switch affected unrelated Home content: %s", home)
	}

	playerBase := withActivePage(base, "player")
	player := renderTemplate(t, renderer, "player.html", service.IngamePlayerView{IngameBaseView: playerBase, PlayerName: "Player", Achievements: &service.CompactAchievementOverview{Badges: []service.AchievementBadge{{ArtworkKey: "career.veteran", Title: "Veteran"}}}})
	if !strings.Contains(player, `class="panel server-navigation-card subpage-navigation-card"`) || !strings.Contains(player, `class="compact-server-header"`) || !strings.Contains(player, `class="panel player-panel"`) || strings.Contains(player, `class="home-hero`) || !strings.Contains(player, "/ingame/assets/"+AssetFingerprint()+"/achievements.png") {
		t.Fatalf("player shell or badge asset is invalid: %s", player)
	}
	statusHidden := playerBase
	statusHidden.Config.Modules.ShowServerStatus = false
	player = renderTemplate(t, renderer, "player.html", service.IngamePlayerView{IngameBaseView: statusHidden, PlayerName: "Player"})
	if strings.Contains(player, `class="compact-status`) || !strings.Contains(player, `class="compact-title"`) {
		t.Fatalf("status switch did not hide only compact summary: %s", player)
	}
	home = renderTemplate(t, renderer, "home.html", service.IngameHomeView{IngameBaseView: statusHidden})
	if strings.Contains(home, `class="panel instance-status"`) {
		t.Fatalf("status switch left Home status panel: %s", home)
	}

	rankingBase := withActivePage(base, "rankings")
	rankings := renderTemplate(t, renderer, "rankings.html", service.IngameRankingView{IngameBaseView: rankingBase, Metric: service.IngameMetricCatalog()[0], Catalog: service.IngameMetricCatalog(), PageNumber: 1})
	if !strings.Contains(rankings, `class="compact-server-header"`) || !strings.Contains(rankings, `class="active" href="/ingame/rankings?server=main-key"`) {
		t.Fatalf("rankings compact header or active navigation is invalid: %s", rankings)
	}
	infoBase := withActivePage(base, "commands")
	info := renderTemplate(t, renderer, "info.html", service.IngameInfoView{IngameBaseView: infoBase, Title: "常用命令", ContentMarkdown: "[外部文档](https://example.com/docs)"})
	if !strings.Contains(info, `class="active" href="/ingame/info/commands?server=main-key"`) || !strings.Contains(info, `href="steam://openurl_external/https://example.com/docs"`) {
		t.Fatalf("info active navigation or external Markdown link is invalid: %s", info)
	}

	unsafe := base
	unsafe.Config.Appearance.BackgroundURL = `javascript:alert(1)`
	home = renderTemplate(t, renderer, "home.html", service.IngameHomeView{IngameBaseView: unsafe})
	if strings.Contains(strings.ToLower(home), "javascript:") || strings.Contains(home, "background-image") {
		t.Fatalf("unsafe background URL was rendered: %s", home)
	}
}

func withActivePage(base service.IngameBaseView, page string) service.IngameBaseView {
	base.ActivePage = page
	return base
}

func renderTemplate(t *testing.T, renderer *Renderer, name string, value any) string {
	t.Helper()
	var output bytes.Buffer
	if err := renderer.Render(&output, name, value); err != nil {
		t.Fatal(err)
	}
	return output.String()
}
