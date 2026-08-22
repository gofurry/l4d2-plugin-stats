package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

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
	ChatAudit         ChatAuditConfig         `yaml:"chat_audit"`
	Logging           LoggingConfig           `yaml:"logging"`
	Monitor           MonitorConfig           `yaml:"monitor"`
}

type ServerConfig struct {
	Listen       string   `yaml:"listen"`
	ReadTimeout  Duration `yaml:"read_timeout"`
	WriteTimeout Duration `yaml:"write_timeout"`
	IdleTimeout  Duration `yaml:"idle_timeout"`
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

type ChatAuditConfig struct {
	DatabasePath string `yaml:"database_path"`
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

type MonitorConfig struct {
	Enabled   bool     `yaml:"enabled"`
	Refresh   Duration `yaml:"refresh"`
	DiskPaths []string `yaml:"disk_paths"`
}

func Default() Config {
	return Config{
		Server: ServerConfig{
			Listen:       "0.0.0.0:18848",
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
		ChatAudit: ChatAuditConfig{DatabasePath: "./chat-audit.db"},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			File:       "./logs/l4d2-stats.log",
			MaxSizeMB:  50,
			MaxBackups: 10,
			MaxAgeDays: 30,
			Compress:   true,
		},
		Monitor: MonitorConfig{Refresh: Duration(2 * time.Second)},
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
	cfg.ChatAudit.DatabasePath = resolvePath(cfg.Directory, cfg.ChatAudit.DatabasePath)
	cfg.Logging.File = resolvePath(cfg.Directory, cfg.Logging.File)
	for i, path := range cfg.Monitor.DiskPaths {
		cfg.Monitor.DiskPaths[i] = resolvePath(cfg.Directory, path)
	}
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
	if c.DashboardDatabase.Path == "" {
		problems = append(problems, errors.New("dashboard_database.path is required"))
	}
	if c.ChatAudit.DatabasePath == "" {
		problems = append(problems, errors.New("chat_audit.database_path is required"))
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
	if c.Monitor.Enabled && c.Monitor.Refresh.Value() <= 0 {
		problems = append(problems, errors.New("monitor.refresh must be positive when monitor is enabled"))
	}
	return errors.Join(problems...)
}
