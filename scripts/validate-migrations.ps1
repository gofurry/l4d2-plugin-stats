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
    "lps_pve_segment_equipment_stats",
    "lps_versus_survivor_stats",
    "lps_versus_infected_stats",
    "lps_versus_infected_class_stats"
)
$requiredIndexes = @(
    "lps_idx_sessions_server_started",
    "lps_idx_sessions_steam_started",
    "lps_idx_runs_server_started",
    "lps_idx_rounds_run_sequence",
    "lps_idx_segments_round_steam",
    "lps_idx_segments_steam_started",
    "lps_idx_pve_equipment_id",
    "lps_idx_versus_infected_class_id"
)
$requiredPvEColumns = @(
    "smoker_kills",
    "charger_kills",
    "damage_to_smoker",
    "damage_to_charger",
    "smoker_controls_received",
    "charger_controls_received",
    "smoker_controlled_seconds",
    "charger_controlled_seconds",
    "smoker_saves",
    "charger_saves",
    "melee_tongue_self_cuts",
    "tank_rocks_destroyed",
    "witch_oneshots",
    "witch_solo_kills",
    "tank_encounters",
    "tank_kill_participations",
    "witch_encounters",
    "witch_kill_participations",
    "incendiary_packs_deployed",
    "explosive_packs_deployed",
    "objective_interactions",
    "ammo_pile_uses",
    "incapacitated_seconds",
    "ledge_hanging_seconds",
    "black_white_teammates_restored"
)
$requiredEquipmentColumns = @(
    "equipment_id",
    "actions",
    "common_kills",
    "special_kills",
    "tank_kills",
    "witch_kills",
    "headshot_kills",
    "damage_to_special",
    "damage_to_tank",
    "damage_to_witch"
)
$requiredVersusInfectedClassColumns = @(
    "infected_class",
    "spawn_count",
    "damage_to_human_survivors",
    "damage_to_bot_survivors",
    "human_survivor_incaps",
    "bot_survivor_incaps",
    "human_survivor_kills",
    "bot_survivor_kills"
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

    foreach ($column in $requiredPvEColumns) {
        if ($sql -notmatch "(?i)\b$column\b") {
            throw "$driver migration is missing PvE column $column."
        }
    }

    foreach ($column in $requiredEquipmentColumns) {
        if ($sql -notmatch "(?i)\b$column\b") {
            throw "$driver migration is missing equipment column $column."
        }
    }

    foreach ($column in $requiredVersusInfectedClassColumns) {
        if ($sql -notmatch "(?i)\b$column\b") {
            throw "$driver migration is missing Versus infected class column $column."
        }
    }

    Write-Host "Validated $driver migration ($($statements.Count) statements)."
}
