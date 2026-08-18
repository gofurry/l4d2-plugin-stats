package service

import (
	"testing"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

func TestMapNameResolverPrecedence(t *testing.T) {
	resolver := NewMapNameResolver([]store.IngameMapName{
		{MapName: "c1m1_hotel", DisplayName: "自定义大厅"},
		{MapName: "CUSTOM_MAP", DisplayName: "三方图第一章"},
	})
	for raw, expected := range map[string]string{
		"c1m1_hotel":      "自定义大厅",
		"C5M1_WATERFRONT": "教区 1/5",
		"c8m1_apartments": "毫不留情 1/5",
		"c8m5_rooftops":   "毫不留情 5/5",
		"custom_map":      "三方图第一章",
		"unknown_map":     "unknown_map",
	} {
		if actual := resolver.DisplayName(raw); actual != expected {
			t.Errorf("DisplayName(%q)=%q, want %q", raw, actual, expected)
		}
	}
}

func TestOfficialIngameMapCatalogIncludesEveryCampaignChapter(t *testing.T) {
	want := map[string]int{"c1": 4, "c2": 5, "c3": 4, "c4": 5, "c5": 5, "c6": 3, "c7": 3, "c8": 5, "c9": 2, "c10": 5, "c11": 5, "c12": 5, "c13": 4, "c14": 2}
	counts := make(map[string]int)
	for key := range officialIngameMapNames {
		for campaign := range want {
			if len(key) > len(campaign) && key[:len(campaign)] == campaign && key[len(campaign)] == 'm' {
				counts[campaign]++
				break
			}
		}
	}
	for campaign, expected := range want {
		if counts[campaign] != expected {
			t.Errorf("%s chapters=%d, want %d", campaign, counts[campaign], expected)
		}
	}
}
