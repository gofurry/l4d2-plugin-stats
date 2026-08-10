package service

import (
	"testing"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

func TestRetentionPlanIDIncludesAggregateContractVersion(t *testing.T) {
	plan := store.RetentionPlan{AggregateVersion: store.AggregateContractVersion, DetailCutoff: 1, SessionCutoff: 2, ResultCutoff: 3, SourceWatermark: 4}
	other := plan
	other.AggregateVersion++
	if retentionPlanID(plan) == retentionPlanID(other) {
		t.Fatal("retention preview ID does not include aggregate contract version")
	}
}
