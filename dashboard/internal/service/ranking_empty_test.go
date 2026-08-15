package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

func TestEmptyRankingUsesJSONArray(t *testing.T) {
	service := &RankingService{}
	page := service.finishRanking(context.Background(), store.RankingQuery{
		Mode:   "pve",
		Metric: "common_kills",
		Limit:  20,
	}, nil)
	if page.Items == nil {
		t.Fatal("empty ranking items must be a non-nil slice")
	}
	encoded, err := json.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"items":[]`) {
		t.Fatalf("empty ranking must encode items as []: %s", encoded)
	}
}
