package server

import (
	"testing"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

func TestValidateSiteSteamOpenIDProxyURL(t *testing.T) {
	validSettings := func(proxyURL string) store.SiteSettings {
		return store.SiteSettings{
			Language: "zh-CN", BrowserTitle: "L4D2 Stats", Theme: "light",
			A2SRefreshSeconds: 30, A2SJitterSeconds: 2, A2SRetryCount: 1,
			SteamOpenIDProxyURL: proxyURL,
		}
	}
	for input, want := range map[string]string{"": "", "127.0.0.1:7890": "http://127.0.0.1:7890", "socks5://10.0.0.8:1080": "socks5://10.0.0.8:1080"} {
		settings := validSettings(input)
		if err := validateSite(&settings); err != nil {
			t.Fatalf("proxy URL %q: %v", input, err)
		}
		if settings.SteamOpenIDProxyURL != want {
			t.Fatalf("normalized proxy URL=%q, want %q", settings.SteamOpenIDProxyURL, want)
		}
	}
	for _, proxyURL := range []string{"ftp://proxy.example.com", "http://proxy.example.com/path"} {
		settings := validSettings(proxyURL)
		if err := validateSite(&settings); err == nil {
			t.Fatalf("proxy URL %q should fail", proxyURL)
		}
	}
}
