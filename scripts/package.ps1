param(
    [ValidatePattern('^[0-9A-Za-z][0-9A-Za-z._-]*$')]
    [string]$Version = "1.3.4"
)

$ErrorActionPreference = "Stop"

$projectRoot = Split-Path -Parent $PSScriptRoot
$dashboardRoot = Join-Path $projectRoot "dashboard"
$frontendRoot = Join-Path $dashboardRoot "frontend"
$distRoot = Join-Path $projectRoot "dist"
$stagingRoot = Join-Path $distRoot "release-staging"
$serverRoot = Join-Path $stagingRoot "left4dead2"
$pluginOutput = Join-Path $distRoot "l4d2_player_stats.smx"
$archiveName = "l4d2-plugin-stats-v$Version.zip"
$archivePath = Join-Path $distRoot $archiveName

& (Join-Path $PSScriptRoot "build.ps1")

Push-Location $frontendRoot
try {
    pnpm install --frozen-lockfile
    pnpm test
    pnpm typecheck
    pnpm lint
    pnpm build
} finally {
    Pop-Location
}

Push-Location $dashboardRoot
try {
    go tool sqlc generate
    go test ./...
} finally {
    Pop-Location
}

$resolvedStagingRoot = [System.IO.Path]::GetFullPath($stagingRoot)
$expectedParent = [System.IO.Path]::GetFullPath($distRoot) + [System.IO.Path]::DirectorySeparatorChar
if (-not $resolvedStagingRoot.StartsWith($expectedParent, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "Unsafe package staging path: $resolvedStagingRoot"
}
if (Test-Path -LiteralPath $resolvedStagingRoot) {
    Remove-Item -LiteralPath $resolvedStagingRoot -Recurse -Force
}

$pluginDestination = Join-Path $serverRoot "addons\sourcemod\plugins\l4d2_player_stats.smx"
$migrationDestination = Join-Path $serverRoot "addons\sourcemod\configs\l4d2_player_stats\migrations"
New-Item -ItemType Directory -Path (Split-Path -Parent $pluginDestination) -Force | Out-Null
Copy-Item -LiteralPath $pluginOutput -Destination $pluginDestination -Force

foreach ($driver in @("sqlite", "mysql", "pgsql")) {
    $driverDestination = Join-Path $migrationDestination $driver
    New-Item -ItemType Directory -Path $driverDestination -Force | Out-Null
    Get-ChildItem -LiteralPath (Join-Path $projectRoot "database\migrations\$driver") -Filter "*.sql" -File |
        Sort-Object Name |
        ForEach-Object { Copy-Item -LiteralPath $_.FullName -Destination (Join-Path $driverDestination $_.Name) -Force }
}

$dashboardDestination = Join-Path $stagingRoot "dashboard"
$windowsDestination = Join-Path $dashboardDestination "windows-amd64"
$linuxDestination = Join-Path $dashboardDestination "linux-amd64"
New-Item -ItemType Directory -Path $windowsDestination -Force | Out-Null
New-Item -ItemType Directory -Path $linuxDestination -Force | Out-Null

$previousCgo = $env:CGO_ENABLED
$previousGoos = $env:GOOS
$previousGoarch = $env:GOARCH
$env:CGO_ENABLED = "0"
$env:GOARCH = "amd64"
try {
    Push-Location $dashboardRoot
    try {
        $env:GOOS = "windows"
        go build -trimpath -ldflags "-s -w -X github.com/gofurry/l4d2-plugin-stats/dashboard/internal/cli.Version=$Version" -o (Join-Path $windowsDestination "l4d2-stats.exe") ./cmd/l4d2-stats
        if ($LASTEXITCODE -ne 0) { throw "Windows Dashboard build failed." }

        $env:GOOS = "linux"
        go build -trimpath -ldflags "-s -w -X github.com/gofurry/l4d2-plugin-stats/dashboard/internal/cli.Version=$Version" -o (Join-Path $linuxDestination "l4d2-stats") ./cmd/l4d2-stats
        if ($LASTEXITCODE -ne 0) { throw "Linux Dashboard build failed." }
    } finally {
        Pop-Location
    }
} finally {
    $env:CGO_ENABLED = $previousCgo
    $env:GOOS = $previousGoos
    $env:GOARCH = $previousGoarch
}

Copy-Item -LiteralPath (Join-Path $dashboardRoot "config.example.yaml") -Destination (Join-Path $windowsDestination "config.example.yaml") -Force
Copy-Item -LiteralPath (Join-Path $dashboardRoot "config.example.yaml") -Destination (Join-Path $linuxDestination "config.example.yaml") -Force

$examplesDestination = Join-Path $stagingRoot "examples"
New-Item -ItemType Directory -Path $examplesDestination -Force | Out-Null
Copy-Item -LiteralPath (Join-Path $projectRoot "config\databases.cfg.example") -Destination $examplesDestination -Force
Copy-Item -LiteralPath (Join-Path $projectRoot "config\dashboard-sqlite.example.yaml") -Destination (Join-Path $examplesDestination "dashboard-sqlite.yaml") -Force
Copy-Item -LiteralPath (Join-Path $projectRoot "config\dashboard-mysql.example.yaml") -Destination (Join-Path $examplesDestination "dashboard-mysql.yaml") -Force
Copy-Item -LiteralPath (Join-Path $projectRoot "config\dashboard-postgresql.example.yaml") -Destination (Join-Path $examplesDestination "dashboard-postgresql.yaml") -Force
Copy-Item -LiteralPath (Join-Path $projectRoot "config\nginx.conf.example") -Destination (Join-Path $examplesDestination "nginx.conf.example") -Force

$packageReadme = Join-Path $stagingRoot "README.md"
Copy-Item -LiteralPath (Join-Path $projectRoot "README.md") -Destination $packageReadme -Force
$utf8NoBom = [System.Text.UTF8Encoding]::new($false)
$readmeContent = [System.IO.File]::ReadAllText($packageReadme, $utf8NoBom)
$readmeContent = $readmeContent.Replace(
    "(docs/assets/",
    "(https://raw.githubusercontent.com/gofurry/l4d2-plugin-stats/main/docs/assets/"
).Replace(
    "(contracts/",
    "(https://github.com/gofurry/l4d2-plugin-stats/blob/main/contracts/"
).Replace(
    "(docs/",
    "(https://github.com/gofurry/l4d2-plugin-stats/blob/main/docs/"
)
[System.IO.File]::WriteAllText($packageReadme, $readmeContent, $utf8NoBom)
Copy-Item -LiteralPath (Join-Path $projectRoot "INSTALL.zh-CN.md") -Destination $stagingRoot -Force
Copy-Item -LiteralPath (Join-Path $projectRoot "UPGRADE.zh-CN.md") -Destination $stagingRoot -Force
Copy-Item -LiteralPath (Join-Path $projectRoot "CHANGELOG.md") -Destination $stagingRoot -Force
Copy-Item -LiteralPath (Join-Path $projectRoot "LICENSE") -Destination $stagingRoot -Force

$resolvedArchivePath = [System.IO.Path]::GetFullPath($archivePath)
if (-not $resolvedArchivePath.StartsWith($expectedParent, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "Unsafe release archive path: $resolvedArchivePath"
}
if (Test-Path -LiteralPath $resolvedArchivePath) {
    Remove-Item -LiteralPath $resolvedArchivePath -Force
}

Add-Type -AssemblyName System.IO.Compression
Add-Type -AssemblyName System.IO.Compression.FileSystem
$archive = [System.IO.Compression.ZipFile]::Open(
    $resolvedArchivePath,
    [System.IO.Compression.ZipArchiveMode]::Create
)
try {
    Get-ChildItem -LiteralPath $resolvedStagingRoot -Recurse -File |
        Sort-Object FullName |
        ForEach-Object {
            $entryName = $_.FullName.Substring($resolvedStagingRoot.Length).TrimStart(
                [System.IO.Path]::DirectorySeparatorChar,
                [System.IO.Path]::AltDirectorySeparatorChar
            ).Replace('\', '/')
            [System.IO.Compression.ZipFileExtensions]::CreateEntryFromFile(
                $archive,
                $_.FullName,
                $entryName,
                [System.IO.Compression.CompressionLevel]::Optimal
            ) | Out-Null
        }
} finally {
    $archive.Dispose()
}

$archiveHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $archivePath).Hash.ToLowerInvariant()
$hashPath = "$archivePath.sha256"
Set-Content -LiteralPath $hashPath -Value "$archiveHash  $archiveName" -Encoding ascii

Write-Host "Release package created: $archivePath"
Write-Host "SHA256: $archiveHash"
