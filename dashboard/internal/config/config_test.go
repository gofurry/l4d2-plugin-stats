package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLoadUsesStrictYAMLAndResolvesRelativePaths(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
server:
  listen: "127.0.0.1:18848"
dashboard_database:
  path: "./data/dashboard.db"
stats_database:
  driver: "sqlite"
  dsn: "../stats.sq3"
logging:
  file: "./logs/dashboard.log"
  format: "json"
  max_size_mb: 25
  max_backups: 4
  max_age_days: 7
bootstrap:
  site:
    title: "Test Stats"
  servers: []
admin:
  enabled: false
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got, want := cfg.DashboardDatabase.Path, filepath.Join(dir, "data", "dashboard.db"); got != want {
		t.Fatalf("dashboard path = %q, want %q", got, want)
	}
	if got, want := cfg.StatsDatabase.DSN, filepath.Clean(filepath.Join(dir, "..", "stats.sq3")); got != want {
		t.Fatalf("stats path = %q, want %q", got, want)
	}
	if got, want := cfg.Logging.File, filepath.Join(dir, "logs", "dashboard.log"); got != want {
		t.Fatalf("log path = %q, want %q", got, want)
	}
	if got := cfg.StatsDatabase.QueryTimeout.Value().String(); got != "5s" {
		t.Fatalf("default query timeout = %q", got)
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := `
server:
  listen: "127.0.0.1:18848"
  surprise: true
stats_database:
  driver: "sqlite"
  dsn: "./stats.sq3"
bootstrap:
  site:
    title: "Test"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "field surprise not found") {
		t.Fatalf("Load() error = %v, want unknown-field failure", err)
	}
}

func TestValidateRejectsUnsafeFooterURL(t *testing.T) {
	cfg := Default()
	cfg.StatsDatabase.DSN = "stats.sq3"
	cfg.Bootstrap.Site.FooterLinks = []BootstrapLinkConfig{{Label: "bad", URL: "javascript:alert(1)"}}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "url must use http or https") {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestExampleConfigRemainsStrictlyLoadable(t *testing.T) {
	_, current, _, _ := runtime.Caller(0)
	path := filepath.Clean(filepath.Join(filepath.Dir(current), "..", "..", "config.example.yaml"))
	if _, err := Load(path); err != nil {
		t.Fatalf("config.example.yaml is invalid: %v", err)
	}
}
