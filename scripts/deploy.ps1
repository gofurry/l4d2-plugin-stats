$ErrorActionPreference = "Stop"

$configPath = Join-Path $PSScriptRoot "config.local.ps1"
if (-not (Test-Path -LiteralPath $configPath -PathType Leaf)) {
    throw "Missing scripts\config.local.ps1."
}

. $configPath

$projectRoot = Split-Path -Parent $PSScriptRoot
$compiledPlugin = Join-Path $projectRoot "dist\l4d2_player_stats.smx"
$deployedPlugin = Join-Path $PluginDirectory "l4d2_player_stats.smx"
$migrationSource = Join-Path $projectRoot "database\migrations"
$migrationDestination = Join-Path $RuntimeConfigDirectory "migrations"

& (Join-Path $PSScriptRoot "build.ps1")

foreach ($requiredDirectory in @($PluginDirectory, (Split-Path -Parent $RuntimeConfigDirectory))) {
    if (-not (Test-Path -LiteralPath $requiredDirectory -PathType Container)) {
        throw "SourceMod deployment directory not found: $requiredDirectory"
    }
}

Copy-Item -LiteralPath $compiledPlugin -Destination $deployedPlugin -Force
New-Item -ItemType Directory -Path $migrationDestination -Force | Out-Null

foreach ($driver in @("sqlite", "mysql", "pgsql")) {
    $sourceFile = Join-Path $migrationSource "$driver\0001_initial.sql"
    $driverDestination = Join-Path $migrationDestination $driver
    New-Item -ItemType Directory -Path $driverDestination -Force | Out-Null
    Copy-Item -LiteralPath $sourceFile -Destination (Join-Path $driverDestination "0001_initial.sql") -Force
}

$sourceHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $compiledPlugin).Hash
$deployedHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $deployedPlugin).Hash
if ($sourceHash -ne $deployedHash) {
    throw "Deployed plugin checksum failed: $deployedPlugin"
}

Write-Host "Deploy succeeded: $deployedPlugin"
Write-Host "Migrations deployed: $migrationDestination"
Write-Host "Reload with: sm plugins reload l4d2_player_stats"
