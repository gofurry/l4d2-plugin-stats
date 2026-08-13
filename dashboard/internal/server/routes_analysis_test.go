package server

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

func TestAnalysisRangeCutoffSupportsPlayerYearRange(t *testing.T) {
	before := time.Now().AddDate(-1, 0, 0).Unix()
	cutoff, err := analysisRangeCutoff("365d")
	after := time.Now().AddDate(-1, 0, 0).Unix()
	if err != nil {
		t.Fatalf("analysisRangeCutoff(365d): %v", err)
	}
	if cutoff < before || cutoff > after {
		t.Fatalf("analysisRangeCutoff(365d) = %d, want between %d and %d", cutoff, before, after)
	}
}

func TestAnalysisPageFilterValidatesAndNormalizesParameters(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		filter, ok := analysisPageFilter(c, store.AnalysisFilter{}, mapAnalysisSorts, "map_name", "asc")
		if !ok {
			return nil
		}
		return c.JSON(filter)
	})

	for _, test := range []struct {
		query string
		code  int
	}{
		{"?page=2&page_size=50&sort=eligible_rounds&order=desc", 200},
		{"?page=0", 400}, {"?page_size=101", 400}, {"?sort=drop_table", 400}, {"?order=sideways", 400},
	} {
		response, err := app.Test(httptest.NewRequest("GET", "http://example.test/"+test.query, nil))
		if err != nil || response.StatusCode != test.code {
			t.Fatalf("analysis page filter %q = %d, %v; want %d", test.query, response.StatusCode, err, test.code)
		}
		if test.code == 200 {
			var filter store.AnalysisFilter
			if err := json.NewDecoder(response.Body).Decode(&filter); err != nil {
				t.Fatal(err)
			}
			if filter.Page != 2 || filter.PageSize != 50 || filter.Sort != "eligible_rounds" || filter.Order != "desc" {
				t.Fatalf("normalized analysis filter = %#v", filter)
			}
		}
	}
}
