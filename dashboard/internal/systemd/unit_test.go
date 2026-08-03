package systemd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/config"
)

func TestGenerateUnitUsesInvokerAndAbsolutePaths(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	cfg.Path = filepath.Join(root, "config.yaml")
	cfg.Directory = root
	cfg.DashboardDatabase.Path = filepath.Join(root, "dashboard.db")
	cfg.Logging.File = filepath.Join(root, "logs", "dashboard.log")
	unit := GenerateUnit(Options{
		BinaryPath: filepath.Join(root, "l4d2-stats"), Config: &cfg,
		Identity: Identity{Username: "alice", Group: "games"},
	})
	for _, expected := range []string{
		"User=alice", "Group=games", "Restart=on-failure", "NoNewPrivileges=true", "PrivateTmp=true",
		"ExecStart=" + quote(filepath.Join(root, "l4d2-stats")) + " serve --config " + quote(cfg.Path),
	} {
		if !strings.Contains(unit, expected) {
			t.Fatalf("unit missing %q:\n%s", expected, unit)
		}
	}
}
