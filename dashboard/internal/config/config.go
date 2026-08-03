package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var serverKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

type Duration time.Duration

func (d Duration) Value() time.Duration { return time.Duration(d) }

func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	v, err := time.ParseDuration(node.Value)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", node.Value, err)
	}
	*d = Duration(v)
	return nil
}

type Config struct {
	Path              string                  `yaml:"-"`
	Directory         string                  `yaml:"-"`
	Server            ServerConfig            `yaml:"server"`
	DashboardDatabase DashboardDatabaseConfig `yaml:"dashboard_database"`
	StatsDatabase     StatsDatabaseConfig     `yaml:"stats_database"`
	Logging           LoggingConfig           `yaml:"logging"`
	Bootstrap         BootstrapConfig         `yaml:"bootstrap"`
	Admin             AdminConfig             `yaml:"admin"`
}

type ServerConfig struct {
	Listen        string   `yaml:"listen"`
	PublicBaseURL string   `yaml:"public_base_url"`
	ReadTimeout   Duration `yaml:"read_timeout"`
	WriteTimeout  Duration `yaml:"write_timeout"`
	IdleTimeout   Duration `yaml:"idle_timeout"`
}

type DashboardDatabaseConfig struct {
	Path string `yaml:"path"`
}

type StatsDatabaseConfig struct {
	Driver          string   `yaml:"driver"`
	DSN             string   `yaml:"dsn"`
	QueryTimeout    Duration `yaml:"query_timeout"`
	MaxOpenConns    int      `yaml:"max_open_conns"`
	MaxIdleConns    int      `yaml:"max_idle_conns"`
	ConnMaxLifetime Duration `yaml:"conn_max_lifetime"`
}

type LoggingConfig struct {
	Level       string `yaml:"level"`
	Format      string `yaml:"format"`
	File        string `yaml:"file"`
	MaxSizeMB   int    `yaml:"max_size_mb"`
	MaxBackups  int    `yaml:"max_backups"`
	MaxAgeDays  int    `yaml:"max_age_days"`
	Compress    bool   `yaml:"compress"`
	AlsoConsole bool   `yaml:"also_console"`
}

type BootstrapConfig struct {
	Site    BootstrapSiteConfig     `yaml:"site"`
	Servers []BootstrapServerConfig `yaml:"servers"`
}

type BootstrapSiteConfig struct {
	Title       string                `yaml:"title"`
	FooterText  string                `yaml:"footer_text"`
	FooterLinks []BootstrapLinkConfig `yaml:"footer_links"`
}

type BootstrapLinkConfig struct {
	Label      string `yaml:"label"`
	URL        string `yaml:"url"`
	OpenNewTab bool   `yaml:"open_in_new_tab"`
	Enabled    bool   `yaml:"enabled"`
}

type BootstrapServerConfig struct {
	ServerKey      string `yaml:"server_key"`
	DisplayName    string `yaml:"display_name"`
	ConnectAddress string `yaml:"connect_address"`
	QueryAddress   string `yaml:"query_address"`
	Primary        bool   `yaml:"primary"`
	Enabled        bool   `yaml:"enabled"`
	SortOrder      int    `yaml:"sort_order"`
}

type AdminConfig struct {
	Enabled      bool     `yaml:"enabled"`
	Username     string   `yaml:"username"`
	PasswordHash string   `yaml:"password_hash"`
	JWTSecret    string   `yaml:"jwt_secret"`
	TokenTTL     Duration `yaml:"token_ttl"`
	CookieSecure string   `yaml:"cookie_secure"`
}

func Default() Config {
	return Config{
		Server: ServerConfig{
			Listen:       "127.0.0.1:18848",
			ReadTimeout:  Duration(10 * time.Second),
			WriteTimeout: Duration(15 * time.Second),
			IdleTimeout:  Duration(60 * time.Second),
		},
		DashboardDatabase: DashboardDatabaseConfig{Path: "./dashboard.db"},
		StatsDatabase: StatsDatabaseConfig{
			Driver:          "sqlite",
			QueryTimeout:    Duration(5 * time.Second),
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: Duration(30 * time.Minute),
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			File:       "./logs/l4d2-stats.log",
			MaxSizeMB:  50,
			MaxBackups: 10,
			MaxAgeDays: 30,
			Compress:   true,
		},
		Bootstrap: BootstrapConfig{Site: BootstrapSiteConfig{Title: "L4D2 Stats"}},
		Admin: AdminConfig{
			Username:     "admin",
			TokenTTL:     Duration(8 * time.Hour),
			CookieSecure: "auto",
		},
	}
}

func Load(path string) (*Config, error) {
	if strings.TrimSpace(path) == "" {
		path = "./config.yaml"
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve config path: %w", err)
	}
	f, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("open config %s: %w", absPath, err)
	}
	defer f.Close()

	cfg := Default()
	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config %s: %w", absPath, err)
	}
	cfg.Path = absPath
	cfg.Directory = filepath.Dir(absPath)
	cfg.DashboardDatabase.Path = resolvePath(cfg.Directory, cfg.DashboardDatabase.Path)
	cfg.Logging.File = resolvePath(cfg.Directory, cfg.Logging.File)
	if cfg.StatsDatabase.Driver == "sqlite" && !strings.HasPrefix(cfg.StatsDatabase.DSN, "file:") {
		cfg.StatsDatabase.DSN = resolvePath(cfg.Directory, cfg.StatsDatabase.DSN)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func resolvePath(base, value string) string {
	if value == "" || filepath.IsAbs(value) {
		return value
	}
	return filepath.Clean(filepath.Join(base, value))
}

func (c *Config) Validate() error {
	var problems []error
	if _, _, err := net.SplitHostPort(c.Server.Listen); err != nil {
		problems = append(problems, fmt.Errorf("server.listen must be host:port: %w", err))
	}
	if c.Server.PublicBaseURL != "" {
		u, err := url.ParseRequestURI(c.Server.PublicBaseURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			problems = append(problems, errors.New("server.public_base_url must be an http or https URL"))
		}
	}
	if c.DashboardDatabase.Path == "" {
		problems = append(problems, errors.New("dashboard_database.path is required"))
	}
	switch c.StatsDatabase.Driver {
	case "sqlite", "mysql", "pgsql", "postgres", "postgresql":
	default:
		problems = append(problems, fmt.Errorf("unsupported stats_database.driver %q", c.StatsDatabase.Driver))
	}
	if strings.TrimSpace(c.StatsDatabase.DSN) == "" {
		problems = append(problems, errors.New("stats_database.dsn is required"))
	}
	if c.StatsDatabase.QueryTimeout.Value() <= 0 {
		problems = append(problems, errors.New("stats_database.query_timeout must be positive"))
	}
	if c.Logging.File == "" || c.Logging.MaxSizeMB <= 0 || c.Logging.MaxBackups < 0 || c.Logging.MaxAgeDays < 0 {
		problems = append(problems, errors.New("logging file and rotation limits are invalid"))
	}
	if c.Logging.Format != "json" && c.Logging.Format != "console" {
		problems = append(problems, errors.New("logging.format must be json or console"))
	}
	if strings.TrimSpace(c.Bootstrap.Site.Title) == "" {
		problems = append(problems, errors.New("bootstrap.site.title is required"))
	}
	for i, link := range c.Bootstrap.Site.FooterLinks {
		if strings.TrimSpace(link.Label) == "" {
			problems = append(problems, fmt.Errorf("bootstrap.site.footer_links[%d].label is required", i))
		}
		if err := validateHTTPURL(link.URL); err != nil {
			problems = append(problems, fmt.Errorf("bootstrap.site.footer_links[%d]: %w", i, err))
		}
	}
	primaryCount := 0
	seenServerKeys := make(map[string]struct{}, len(c.Bootstrap.Servers))
	for i, server := range c.Bootstrap.Servers {
		if !serverKeyPattern.MatchString(server.ServerKey) {
			problems = append(problems, fmt.Errorf("bootstrap.servers[%d].server_key is invalid", i))
		}
		if _, exists := seenServerKeys[server.ServerKey]; exists {
			problems = append(problems, fmt.Errorf("bootstrap.servers[%d].server_key is duplicated", i))
		}
		seenServerKeys[server.ServerKey] = struct{}{}
		if strings.TrimSpace(server.DisplayName) == "" {
			problems = append(problems, fmt.Errorf("bootstrap.servers[%d].display_name is required", i))
		}
		if _, _, err := net.SplitHostPort(server.ConnectAddress); err != nil {
			problems = append(problems, fmt.Errorf("bootstrap.servers[%d].connect_address must be host:port", i))
		}
		if _, _, err := net.SplitHostPort(server.QueryAddress); err != nil {
			problems = append(problems, fmt.Errorf("bootstrap.servers[%d].query_address must be host:port", i))
		}
		if server.Primary {
			primaryCount++
			if !server.Enabled {
				problems = append(problems, fmt.Errorf("bootstrap.servers[%d] primary server must be enabled", i))
			}
		}
	}
	if primaryCount > 1 {
		problems = append(problems, errors.New("bootstrap.servers can contain at most one primary server"))
	}
	if c.Admin.CookieSecure != "auto" && c.Admin.CookieSecure != "true" && c.Admin.CookieSecure != "false" {
		problems = append(problems, errors.New("admin.cookie_secure must be auto, true, or false"))
	}
	if c.Admin.Enabled {
		if strings.TrimSpace(c.Admin.Username) == "" || strings.TrimSpace(c.Admin.PasswordHash) == "" {
			problems = append(problems, errors.New("admin username and password_hash are required when enabled"))
		}
		if len(c.Admin.JWTSecret) < 32 {
			problems = append(problems, errors.New("admin.jwt_secret must contain at least 32 characters when enabled"))
		}
	}
	return errors.Join(problems...)
}

func validateHTTPURL(raw string) error {
	u, err := url.ParseRequestURI(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return errors.New("url must use http or https")
	}
	return nil
}
