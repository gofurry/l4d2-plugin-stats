package server

import (
	"testing"
	"time"
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
