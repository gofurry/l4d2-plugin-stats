package server

import (
	"testing"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

func TestValidateSiteSteamOpenIDProxyPort(t *testing.T) {
	validSettings := func(port int64) store.SiteSettings {
		return store.SiteSettings{
			Language: "zh-CN", BrowserTitle: "L4D2 Stats", Theme: "light",
			A2SRefreshSeconds: 30, A2SJitterSeconds: 2, A2SRetryCount: 1,
			SteamOpenIDProxyPort: port,
		}
	}
	for _, port := range []int64{0, 1, 7890, 65535} {
		settings := validSettings(port)
		if err := validateSite(&settings); err != nil {
			t.Fatalf("proxy port %d: %v", port, err)
		}
	}
	for _, port := range []int64{-1, 65536} {
		settings := validSettings(port)
		if err := validateSite(&settings); err == nil {
			t.Fatalf("proxy port %d should fail", port)
		}
	}
}
