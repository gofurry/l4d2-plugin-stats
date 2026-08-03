$ErrorActionPreference = "Stop"

$configPath = Join-Path $PSScriptRoot "config.local.ps1"
if (-not (Test-Path -LiteralPath $configPath -PathType Leaf)) {
    throw "Missing scripts\config.local.ps1. Copy config.example.ps1 and configure local paths."
}

. $configPath

$projectRoot = Split-Path -Parent $PSScriptRoot
$sourceFile = Join-Path $projectRoot "collector\src\l4d2_player_stats.sp"
$projectInclude = Join-Path $projectRoot "collector\include"
$distDirectory = Join-Path $projectRoot "dist"
$outputFile = Join-Path $distDirectory "l4d2_player_stats.smx"
$migrationValidator = Join-Path $PSScriptRoot "validate-migrations.ps1"
$versusContractValidator = Join-Path $PSScriptRoot "validate_versus_contract.py"

foreach ($requiredPath in @($CompilerPath, $SourceModInclude, $sourceFile)) {
    if (-not (Test-Path -LiteralPath $requiredPath)) {
        throw "Required build path not found: $requiredPath"
    }
}

New-Item -ItemType Directory -Path $distDirectory -Force | Out-Null
& $migrationValidator
python $versusContractValidator
if ($LASTEXITCODE -ne 0) {
    throw "Versus contract validation failed. Exit code: $LASTEXITCODE"
}

Write-Host "Building L4D2 Player Stats..."
& $CompilerPath `
    $sourceFile `
    "-i$SourceModInclude" `
    "-i$projectInclude" `
    "-o$outputFile"

if ($LASTEXITCODE -ne 0) {
    throw "Compilation failed. spcomp exit code: $LASTEXITCODE"
}

if (-not (Test-Path -LiteralPath $outputFile -PathType Leaf)) {
    throw "Compiler did not create output: $outputFile"
}

Write-Host "Build succeeded: $outputFile"
