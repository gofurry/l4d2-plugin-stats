package dashboarddb

import "embed"

// Migrations contains the dashboard-owned SQLite migrations.
//
//go:embed migrations/*.sql
var Migrations embed.FS
