package systemd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/config"
)

const ServiceName = "l4d2-stats.service"

type Identity struct {
	Username string
	Group    string
	UID      string
	GID      string
	GIDs     []string
	Root     bool
}

type Options struct {
	BinaryPath string
	Config     *config.Config
	Identity   Identity
}

func GenerateUnit(options Options) string {
	writePaths := uniqueSorted([]string{
		options.Config.Directory,
		filepath.Dir(options.Config.DashboardDatabase.Path),
		filepath.Dir(options.Config.Logging.File),
	})
	var b strings.Builder
	fmt.Fprintln(&b, "[Unit]")
	fmt.Fprintln(&b, "Description=L4D2 Player Stats Dashboard")
	fmt.Fprintln(&b, "After=network-online.target")
	fmt.Fprintln(&b, "Wants=network-online.target")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "[Service]")
	fmt.Fprintln(&b, "Type=simple")
	fmt.Fprintf(&b, "User=%s\n", options.Identity.Username)
	fmt.Fprintf(&b, "Group=%s\n", options.Identity.Group)
	fmt.Fprintf(&b, "WorkingDirectory=%s\n", options.Config.Directory)
	fmt.Fprintf(&b, "ExecStart=%s serve --config %s\n", quote(options.BinaryPath), quote(options.Config.Path))
	fmt.Fprintln(&b, "Restart=on-failure")
	fmt.Fprintln(&b, "RestartSec=3")
	fmt.Fprintln(&b, "KillSignal=SIGTERM")
	fmt.Fprintln(&b, "TimeoutStopSec=20")
	fmt.Fprintln(&b, "NoNewPrivileges=true")
	fmt.Fprintln(&b, "PrivateTmp=true")
	fmt.Fprintln(&b, "ProtectSystem=full")
	fmt.Fprintln(&b, "ProtectHome=false")
	for _, path := range writePaths {
		fmt.Fprintf(&b, "ReadWritePaths=%s\n", path)
	}
	fmt.Fprintln(&b, "StandardOutput=journal")
	fmt.Fprintln(&b, "StandardError=journal")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "[Install]")
	fmt.Fprintln(&b, "WantedBy=multi-user.target")
	return b.String()
}

func quote(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = filepath.Clean(value)
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
