$ErrorActionPreference = "Stop"

$projectRoot = Split-Path -Parent $PSScriptRoot
$migrationRoot = Join-Path $projectRoot "database\migrations"
$drivers = @("sqlite", "mysql", "pgsql")
$requiredTables = @(
    "lps_schema_migrations",
    "lps_servers",
    "lps_server_boots",
    "lps_players",
    "lps_sessions",
    "lps_runs",
    "lps_rounds",
    "lps_player_segments",
    "lps_pve_segment_stats",
    "lps_versus_survivor_stats",
    "lps_versus_infected_stats"
)
$requiredIndexes = @(
    "lps_idx_sessions_server_started",
    "lps_idx_sessions_steam_started",
    "lps_idx_runs_server_started",
    "lps_idx_rounds_run_sequence",
    "lps_idx_segments_round_steam",
    "lps_idx_segments_steam_started"
)

foreach ($driver in $drivers) {
    $migration = Join-Path $migrationRoot "$driver\0001_initial.sql"
    if (-not (Test-Path -LiteralPath $migration -PathType Leaf)) {
        throw "Missing $driver migration: $migration"
    }

    $sql = Get-Content -LiteralPath $migration -Raw
    $statements = [regex]::Split(
        $sql,
        '(?m)^\s*-- statement-breakpoint\s*$'
    ) | ForEach-Object { $_.Trim() } | Where-Object { $_ -ne "" }

    if ($statements.Count -lt $requiredTables.Count) {
        throw "$driver migration has too few statements: $($statements.Count)"
    }

    if ($statements[0] -notmatch '(?is)^CREATE TABLE IF NOT EXISTS\s+lps_schema_migrations') {
        throw "$driver migration must bootstrap lps_schema_migrations first."
    }

    foreach ($statement in $statements) {
        if (-not $statement.EndsWith(';')) {
            throw "$driver migration contains a statement without a terminating semicolon."
        }
    }

    foreach ($table in $requiredTables) {
        if ($sql -notmatch "(?i)CREATE TABLE IF NOT EXISTS\s+$table\b") {
            throw "$driver migration is missing table $table."
        }
    }

    foreach ($index in $requiredIndexes) {
        if ($sql -notmatch "(?i)\b$index\b") {
            throw "$driver migration is missing index $index."
        }
    }

    Write-Host "Validated $driver migration ($($statements.Count) statements)."
}
