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
    "lps_versus_round_results",
    "lps_versus_run_results",
    "lps_player_segments",
    "lps_pve_segment_stats",
    "lps_pve_segment_equipment_stats",
    "lps_versus_survivor_stats",
    "lps_versus_survivor_infected_class_stats",
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
    "lps_idx_versus_survivor_infected_class_id",
    "lps_idx_versus_infected_class_id",
    "lps_idx_server_boots_server_status",
    "lps_idx_sessions_started",
    "lps_idx_sessions_ended",
    "lps_idx_sessions_saved",
    "lps_idx_runs_started",
    "lps_idx_runs_saved",
    "lps_idx_rounds_started",
    "lps_idx_rounds_saved",
    "lps_idx_versus_round_results_saved",
    "lps_idx_versus_round_results_finalized",
    "lps_idx_versus_run_results_saved",
    "lps_idx_versus_run_results_finalized",
    "lps_idx_segments_session_status",
    "lps_idx_segments_started",
    "lps_idx_segments_ended",
    "lps_idx_segments_saved",
    "lps_idx_pve_stats_saved",
    "lps_idx_pve_equipment_saved",
    "lps_idx_versus_survivor_saved",
    "lps_idx_versus_survivor_class_saved",
    "lps_idx_versus_infected_saved",
    "lps_idx_versus_infected_class_saved"
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
$requiredVersusSurvivorColumns = @(
    "witch_kills",
    "damage_to_witch",
    "molotovs_thrown",
    "pipe_bombs_thrown",
    "vomit_jars_thrown",
    "incendiary_packs_deployed",
    "explosive_packs_deployed",
    "melee_tongue_self_cuts",
    "tank_rocks_destroyed",
    "witch_oneshots",
    "witch_solo_kills"
)
$requiredVersusSurvivorClassColumns = @(
    "infected_class",
    "human_controller_kills",
    "bot_controller_kills",
    "damage_to_human_controllers",
    "damage_to_bot_controllers"
)
$requiredVersusRoundResultColumns = @(
    "scoring_team_slot",
    "teams_flipped",
    "team_0_map_score",
    "team_1_map_score",
    "team_0_campaign_score",
    "team_1_campaign_score",
    "raw_winner_team",
    "score_available",
    "result_status",
    "finalized_at"
)
$requiredVersusRunResultColumns = @(
    "team_0_campaign_score",
    "team_1_campaign_score",
    "winner_team_slot",
    "raw_winner_team",
    "score_available",
    "result_status",
    "finalized_at"
)
$requiredAnalysisIndexes = @(
    "lps_idx_round_contexts_saved",
    "lps_idx_incidents_occurred",
    "lps_idx_incidents_actor_time",
    "lps_idx_incidents_target_time",
    "lps_idx_rounds_server_started_map"
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

    foreach ($column in $requiredVersusSurvivorColumns) {
        if ($sql -notmatch "(?i)\b$column\b") {
            throw "$driver migration is missing Versus survivor column $column."
        }
    }

    foreach ($column in $requiredVersusSurvivorClassColumns) {
        if ($sql -notmatch "(?i)\b$column\b") {
            throw "$driver migration is missing Versus survivor class column $column."
        }
    }

    foreach ($column in $requiredVersusRoundResultColumns) {
        if ($sql -notmatch "(?i)\b$column\b") {
            throw "$driver migration is missing Versus round result column $column."
        }
    }

    foreach ($column in $requiredVersusRunResultColumns) {
        if ($sql -notmatch "(?i)\b$column\b") {
            throw "$driver migration is missing Versus run result column $column."
        }
    }

    $secondMigration = Join-Path $migrationRoot "$driver\0002_car_alarms_triggered.sql"
    if (-not (Test-Path -LiteralPath $secondMigration -PathType Leaf)) {
        throw "Missing $driver migration: $secondMigration"
    }
    $secondSQL = Get-Content -LiteralPath $secondMigration -Raw
    $secondStatements = [regex]::Split(
        $secondSQL,
        '(?m)^\s*-- statement-breakpoint\s*$'
    ) | ForEach-Object { $_.Trim() } | Where-Object { $_ -ne "" }
    if ($secondStatements.Count -ne 2) {
        throw "$driver migration 0002 must contain exactly two statements."
    }
    foreach ($statement in $secondStatements) {
        if (-not $statement.EndsWith(';') -or $statement -notmatch '(?i)ADD COLUMN\s+car_alarms_triggered\b') {
            throw "$driver migration 0002 must add car_alarms_triggered with terminated statements."
        }
    }

    $thirdMigration = Join-Path $migrationRoot "$driver\0003_versus_objective_interactions.sql"
    if (-not (Test-Path -LiteralPath $thirdMigration -PathType Leaf)) {
        throw "Missing $driver migration: $thirdMigration"
    }
    $thirdSQL = Get-Content -LiteralPath $thirdMigration -Raw
    $thirdStatements = [regex]::Split(
        $thirdSQL,
        '(?m)^\s*-- statement-breakpoint\s*$'
    ) | ForEach-Object { $_.Trim() } | Where-Object { $_ -ne "" }
    if ($thirdStatements.Count -ne 1) {
        throw "$driver migration 0003 must contain exactly one statement."
    }
    $thirdStatement = [string]$thirdStatements
    if (-not $thirdStatement.EndsWith(';') -or $thirdStatement -notmatch '(?i)ALTER TABLE\s+lps_versus_survivor_stats\s+ADD COLUMN\s+objective_interactions\b') {
        throw "$driver migration 0003 must add objective_interactions to lps_versus_survivor_stats."
    }

    $fourthMigration = Join-Path $migrationRoot "$driver\0004_analysis_foundation.sql"
    if (-not (Test-Path -LiteralPath $fourthMigration -PathType Leaf)) {
        throw "Missing $driver migration: $fourthMigration"
    }
    $fourthSQL = Get-Content -LiteralPath $fourthMigration -Raw
    $fourthStatements = [regex]::Split(
        $fourthSQL,
        '(?m)^\s*-- statement-breakpoint\s*$'
    ) | ForEach-Object { $_.Trim() } | Where-Object { $_ -ne "" }
    if ($fourthStatements.Count -ne 7) {
        throw "$driver migration 0004 must contain exactly seven statements."
    }
    foreach ($statement in $fourthStatements) {
        if (-not $statement.EndsWith(';')) {
            throw "$driver migration 0004 contains a statement without a terminating semicolon."
        }
    }
    foreach ($table in @("lps_round_contexts", "lps_incidents")) {
        if ($fourthSQL -notmatch "(?i)CREATE TABLE IF NOT EXISTS\s+$table\b") {
            throw "$driver migration 0004 is missing table $table."
        }
    }
    foreach ($index in $requiredAnalysisIndexes) {
        if ($fourthSQL -notmatch "(?i)\b$index\b") {
            throw "$driver migration 0004 is missing index $index."
        }
    }

    $fifthMigration = Join-Path $migrationRoot "$driver\0005_relationships_and_assists.sql"
    if (-not (Test-Path -LiteralPath $fifthMigration -PathType Leaf)) {
        throw "Missing $driver migration: $fifthMigration"
    }
    $fifthSQL = Get-Content -LiteralPath $fifthMigration -Raw
    $fifthStatements = [regex]::Split(
        $fifthSQL,
        '(?m)^\s*-- statement-breakpoint\s*$'
    ) | ForEach-Object { $_.Trim() } | Where-Object { $_ -ne "" }
    $expectedFifthStatements = if ($driver -eq "mysql") { 17 } else { 19 }
    if ($fifthStatements.Count -ne $expectedFifthStatements) {
        throw "$driver migration 0005 must contain exactly $expectedFifthStatements statements."
    }
    foreach ($statement in $fifthStatements) {
        if (-not $statement.EndsWith(';')) {
            throw "$driver migration 0005 contains a statement without a terminating semicolon."
        }
    }
    $nullableAssistColumns = @(
        "special_assists", "smoker_assists", "boomer_assists",
        "hunter_assists", "spitter_assists", "jockey_assists",
        "charger_assists", "human_special_assists", "bot_special_assists",
        "human_tank_assists", "bot_tank_assists", "witch_encounters",
        "witch_kill_participations", "black_white_teammates_restored",
        "human_controller_assists", "bot_controller_assists"
    )
    foreach ($column in $nullableAssistColumns) {
        if ($fifthSQL -notmatch "(?i)ADD COLUMN\s+$column\s+BIGINT\s+NULL") {
            throw "$driver migration 0005 must add nullable historical column $column."
        }
    }
    if ($fifthSQL -notmatch '(?i)CREATE TABLE\s+lps_player_round_relationship_stats\b') {
        throw "$driver migration 0005 is missing lps_player_round_relationship_stats."
    }
    foreach ($index in @("lps_idx_relationship_actor_round", "lps_idx_relationship_target_round")) {
        if ($fifthSQL -notmatch "(?i)\b$index\b") {
            throw "$driver migration 0005 is missing index $index."
        }
    }

    $sixthMigration = Join-Path $migrationRoot "$driver\0006_fall_deaths.sql"
    if (-not (Test-Path -LiteralPath $sixthMigration -PathType Leaf)) {
        throw "Missing $driver migration: $sixthMigration"
    }
    $sixthSQL = Get-Content -LiteralPath $sixthMigration -Raw
    $sixthStatements = [regex]::Split(
        $sixthSQL,
        '(?m)^\s*-- statement-breakpoint\s*$'
    ) | ForEach-Object { $_.Trim() } | Where-Object { $_ -ne "" }
    if ($sixthStatements.Count -ne 2) {
        throw "$driver migration 0006 must contain exactly two statements."
    }
    foreach ($statement in $sixthStatements) {
        if (-not $statement.EndsWith(';')) {
            throw "$driver migration 0006 contains a statement without a terminating semicolon."
        }
    }
    foreach ($table in @("lps_pve_segment_stats", "lps_versus_survivor_stats")) {
        if ($sixthSQL -notmatch "(?i)ALTER TABLE\s+$table\s+ADD COLUMN\s+fall_deaths\s+BIGINT\s+NULL") {
            throw "$driver migration 0006 must add nullable fall_deaths to $table."
        }
    }

    Write-Host "Validated $driver migrations ($($statements.Count + $secondStatements.Count + $thirdStatements.Count + $fourthStatements.Count + $fifthStatements.Count + $sixthStatements.Count) statements)."
}
