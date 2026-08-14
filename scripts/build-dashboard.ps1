$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$dashboardRoot = Join-Path $repoRoot "dashboard"
$frontendRoot = Join-Path $dashboardRoot "frontend"
$outputDirectory = Join-Path $repoRoot "dist"
$outputFile = Join-Path $outputDirectory "l4d2-stats.exe"
$version = if ($env:L4D2_STATS_VERSION) { $env:L4D2_STATS_VERSION } else { "1.3.1" }

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
    New-Item -ItemType Directory -Path $outputDirectory -Force | Out-Null
    $distConfig = Join-Path $outputDirectory "config.yaml"
    if (-not (Test-Path $distConfig)) {
        Copy-Item (Join-Path $dashboardRoot "config.example.yaml") $distConfig
    }
    $previousCgo = $env:CGO_ENABLED
    $env:CGO_ENABLED = "0"
    try {
        go build `
            -trimpath `
            -ldflags "-s -w -X github.com/gofurry/l4d2-plugin-stats/dashboard/internal/cli.Version=$version" `
            -o $outputFile `
            ./cmd/l4d2-stats
    } finally {
        $env:CGO_ENABLED = $previousCgo
    }
} finally {
    Pop-Location
}

Write-Host "Dashboard build completed: $outputFile"
Write-Host "Dashboard test config: $(Join-Path $outputDirectory 'config.yaml')"
