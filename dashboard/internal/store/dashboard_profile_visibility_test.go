package store

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPlayerProfileVisibilityDefaultsAndPersistence(t *testing.T) {
	ctx := context.Background()
	dashboard, err := OpenDashboard(ctx, filepath.Join(t.TempDir(), "dashboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer dashboard.Close()

	visibility, err := dashboard.PlayerProfileVisibility(ctx, "76561198000000001")
	if err != nil || !reflect.DeepEqual(visibility.VisibleSections, DefaultPlayerProfileSections) || visibility.UpdatedAt != 0 {
		t.Fatalf("default visibility=%#v err=%v", visibility, err)
	}

	visibility, err = dashboard.ReplacePlayerProfileVisibility(ctx, "76561198000000001", []PlayerProfileSection{PlayerProfileHistory, PlayerProfileOverview}, 123)
	if err != nil {
		t.Fatal(err)
	}
	want := []PlayerProfileSection{PlayerProfileOverview, PlayerProfileHistory}
	if !reflect.DeepEqual(visibility.VisibleSections, want) || visibility.UpdatedAt != 123 {
		t.Fatalf("saved visibility=%#v", visibility)
	}
	loaded, err := dashboard.PlayerProfileVisibility(ctx, "76561198000000001")
	if err != nil || !reflect.DeepEqual(loaded, visibility) {
		t.Fatalf("loaded visibility=%#v err=%v", loaded, err)
	}

	empty, err := dashboard.ReplacePlayerProfileVisibility(ctx, "76561198000000001", nil, 124)
	if err != nil || len(empty.VisibleSections) != 0 {
		t.Fatalf("empty visibility=%#v err=%v", empty, err)
	}
	loaded, err = dashboard.PlayerProfileVisibility(ctx, "76561198000000001")
	if err != nil || len(loaded.VisibleSections) != 0 || loaded.UpdatedAt != 124 {
		t.Fatalf("loaded empty visibility=%#v err=%v", loaded, err)
	}
}

func TestPlayerProfileVisibilityRejectsUnknownAndDuplicateSections(t *testing.T) {
	ctx := context.Background()
	dashboard, err := OpenDashboard(ctx, filepath.Join(t.TempDir(), "dashboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer dashboard.Close()

	if _, err := dashboard.ReplacePlayerProfileVisibility(ctx, "76561198000000001", []PlayerProfileSection{"future"}, 1); err == nil {
		t.Fatal("unknown section was accepted")
	}
	if _, err := dashboard.ReplacePlayerProfileVisibility(ctx, "76561198000000001", []PlayerProfileSection{PlayerProfileOverview, PlayerProfileOverview}, 1); err == nil {
		t.Fatal("duplicate section was accepted")
	}
}
