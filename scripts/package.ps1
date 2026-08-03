$ErrorActionPreference = "Stop"

$projectRoot = Split-Path -Parent $PSScriptRoot
$definitionsFile = Join-Path $projectRoot "collector\include\l4d2_player_stats\definitions.inc"
$compiledPlugin = Join-Path $projectRoot "dist\l4d2_player_stats.smx"
$migrationSource = Join-Path $projectRoot "database\migrations"
$stagingRoot = Join-Path $projectRoot "dist\package"
$serverRoot = Join-Path $stagingRoot "left4dead2"

& (Join-Path $PSScriptRoot "build.ps1")

$versionLine = Select-String -LiteralPath $definitionsFile -Pattern '#define LPS_VERSION "([^"]+)"'
if (-not $versionLine) {
    throw "Could not read LPS_VERSION from $definitionsFile"
}

$version = $versionLine.Matches[0].Groups[1].Value
$archivePath = Join-Path $projectRoot "dist\l4d2-player-stats-v$version.zip"
$resolvedStagingRoot = [System.IO.Path]::GetFullPath($stagingRoot)
$expectedParent = [System.IO.Path]::GetFullPath((Join-Path $projectRoot "dist")) + [System.IO.Path]::DirectorySeparatorChar

if (-not $resolvedStagingRoot.StartsWith($expectedParent, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "Unsafe package staging path: $resolvedStagingRoot"
}

if (Test-Path -LiteralPath $resolvedStagingRoot) {
    Remove-Item -LiteralPath $resolvedStagingRoot -Recurse -Force
}

$pluginDestination = Join-Path $serverRoot "addons\sourcemod\plugins\l4d2_player_stats.smx"
$migrationDestination = Join-Path $serverRoot "addons\sourcemod\configs\l4d2_player_stats\migrations"
New-Item -ItemType Directory -Path (Split-Path -Parent $pluginDestination) -Force | Out-Null
Copy-Item -LiteralPath $compiledPlugin -Destination $pluginDestination -Force

foreach ($driver in @("sqlite", "mysql", "pgsql")) {
    $driverDestination = Join-Path $migrationDestination $driver
    New-Item -ItemType Directory -Path $driverDestination -Force | Out-Null
    Copy-Item `
        -LiteralPath (Join-Path $migrationSource "$driver\0001_initial.sql") `
        -Destination (Join-Path $driverDestination "0001_initial.sql") `
        -Force
}

$examplesDestination = Join-Path $stagingRoot "examples"
New-Item -ItemType Directory -Path $examplesDestination -Force | Out-Null
Copy-Item -LiteralPath (Join-Path $projectRoot "config\databases.cfg.example") -Destination $examplesDestination -Force
Copy-Item -LiteralPath (Join-Path $projectRoot "README.md") -Destination $stagingRoot -Force
Copy-Item -LiteralPath (Join-Path $projectRoot "LICENSE") -Destination $stagingRoot -Force
Copy-Item -LiteralPath (Join-Path $projectRoot "CHANGELOG.md") -Destination $stagingRoot -Force
$contractsDestination = Join-Path $stagingRoot "contracts"
New-Item -ItemType Directory -Path $contractsDestination -Force | Out-Null
Copy-Item -LiteralPath (Join-Path $projectRoot "contracts\modes.md") -Destination $contractsDestination -Force
Copy-Item -LiteralPath (Join-Path $projectRoot "contracts\statistics.md") -Destination $contractsDestination -Force
Copy-Item -LiteralPath (Join-Path $projectRoot "contracts\versus-v1.md") -Destination $contractsDestination -Force
Copy-Item -LiteralPath (Join-Path $projectRoot "contracts\versus-schema-v1.json") -Destination $contractsDestination -Force
$databaseDocsDestination = Join-Path $stagingRoot "database"
New-Item -ItemType Directory -Path $databaseDocsDestination -Force | Out-Null
Copy-Item -LiteralPath (Join-Path $projectRoot "database\schema.md") -Destination $databaseDocsDestination -Force
$queryDestination = Join-Path $databaseDocsDestination "queries"
New-Item -ItemType Directory -Path $queryDestination -Force | Out-Null
Copy-Item -LiteralPath (Join-Path $projectRoot "database\queries\versus_contract_checks.sql") -Destination $queryDestination -Force
$docsDestination = Join-Path $stagingRoot "docs"
New-Item -ItemType Directory -Path $docsDestination -Force | Out-Null
Copy-Item -LiteralPath (Join-Path $projectRoot "docs\database-foundation.md") -Destination $docsDestination -Force
Copy-Item -LiteralPath (Join-Path $projectRoot "docs\v0.2-test-checklist.md") -Destination $docsDestination -Force
Copy-Item -LiteralPath (Join-Path $projectRoot "docs\v0.3-test-checklist.md") -Destination $docsDestination -Force
Copy-Item -LiteralPath (Join-Path $projectRoot "docs\v0.4-test-checklist.md") -Destination $docsDestination -Force
Copy-Item -LiteralPath (Join-Path $projectRoot "docs\v0.5-test-checklist.md") -Destination $docsDestination -Force
Copy-Item -LiteralPath (Join-Path $projectRoot "docs\v0.5.1-test-checklist.md") -Destination $docsDestination -Force
Copy-Item -LiteralPath (Join-Path $projectRoot "docs\v0.5.2-test-checklist.md") -Destination $docsDestination -Force
Copy-Item -LiteralPath (Join-Path $projectRoot "docs\v0.6-test-checklist.md") -Destination $docsDestination -Force
Copy-Item -LiteralPath (Join-Path $projectRoot "docs\v0.6.2-test-checklist.md") -Destination $docsDestination -Force
Copy-Item -LiteralPath (Join-Path $projectRoot "docs\v0.6.3-test-checklist.md") -Destination $docsDestination -Force
Copy-Item -LiteralPath (Join-Path $projectRoot "docs\v0.6.4-test-checklist.md") -Destination $docsDestination -Force
Copy-Item -LiteralPath (Join-Path $projectRoot "docs\v0.6.5-test-checklist.md") -Destination $docsDestination -Force
Copy-Item -LiteralPath (Join-Path $projectRoot "docs\v0.6.6-contract-freeze.md") -Destination $docsDestination -Force
Copy-Item -LiteralPath (Join-Path $projectRoot "docs\query-examples.md") -Destination $docsDestination -Force

Compress-Archive -Path (Join-Path $stagingRoot "*") -DestinationPath $archivePath -Force
Write-Host "Release package created: $archivePath"
