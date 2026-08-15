package auth

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

func TestSetupLoginAndTokenRevocation(t *testing.T) {
	ctx := context.Background()
	dashboard, err := store.OpenDashboard(ctx, filepath.Join(t.TempDir(), "dashboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer dashboard.Close()

	service, setupToken, err := New(ctx, dashboard)
	if err != nil || setupToken == "" {
		t.Fatalf("New() token=%q err=%v", setupToken, err)
	}
	if err := service.Setup(ctx, "wrong", "admin", "correct horse battery staple"); !errors.Is(err, ErrSetupToken) {
		t.Fatalf("wrong setup token error=%v", err)
	}
	if err := service.Setup(ctx, setupToken, "admin", "correct horse battery staple"); err != nil {
		t.Fatal(err)
	}
	if err := service.Setup(ctx, setupToken, "other", "correct horse battery staple"); !errors.Is(err, ErrSetupToken) {
		t.Fatalf("replayed setup token error=%v", err)
	}

	adminToken, err := service.Login(ctx, "admin", "correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ValidateAdmin(ctx, adminToken); err != nil {
		t.Fatalf("valid admin token: %v", err)
	}
	if _, err := service.Login(ctx, "admin", "wrong"); !errors.Is(err, ErrInvalidLogin) {
		t.Fatalf("wrong password error=%v", err)
	}

	steamToken, err := service.SignSteamIdentity(ctx, "76561198000000000")
	if err != nil {
		t.Fatal(err)
	}
	steamID, err := service.ValidateSteamIdentity(ctx, steamToken)
	if err != nil || steamID != "76561198000000000" {
		t.Fatalf("steam identity=%q err=%v", steamID, err)
	}
	badgeEditToken, err := service.SignSteamBadgeEdit(ctx, steamID)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ValidateSteamBadgeEdit(ctx, badgeEditToken, steamID); err != nil {
		t.Fatalf("valid badge edit token: %v", err)
	}
	if err := service.ValidateSteamBadgeEdit(ctx, badgeEditToken, "76561198000000001"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("cross-player badge edit token error=%v", err)
	}
	if _, err := service.ValidateSteamIdentity(ctx, badgeEditToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("badge edit token accepted as identity: %v", err)
	}
	intentToken, err := service.SignSteamOpenIDIntent(ctx, "badge_edit", "/player?tab=achievements")
	if err != nil {
		t.Fatal(err)
	}
	intent, err := service.ValidateSteamOpenIDIntent(ctx, intentToken)
	if err != nil || intent.Purpose != "badge_edit" || intent.ReturnTo != "/player?tab=achievements" {
		t.Fatalf("Steam intent=%+v err=%v", intent, err)
	}

	newToken, err := service.ChangePassword(ctx, "correct horse battery staple", "a different strong password")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ValidateAdmin(ctx, adminToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("old token should be revoked, error=%v", err)
	}
	if _, err := service.ValidateAdmin(ctx, newToken); err != nil {
		t.Fatalf("new token invalid: %v", err)
	}
}

func TestConcurrentSetupOnlyCreatesOneAdministrator(t *testing.T) {
	ctx := context.Background()
	dashboard, err := store.OpenDashboard(ctx, filepath.Join(t.TempDir(), "dashboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer dashboard.Close()
	service, token, err := New(ctx, dashboard)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := range 2 {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results <- service.Setup(ctx, token, "admin", "correct horse battery staple")
		}(i)
	}
	wg.Wait()
	close(results)
	succeeded := 0
	for err := range results {
		if err == nil {
			succeeded++
		}
	}
	if succeeded != 1 {
		t.Fatalf("successful setup attempts=%d", succeeded)
	}
}
