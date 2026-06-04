# update-icon-svg.ps1 — Rebuild Windows icon from jpaste-logo.svg
# Usage: .\scripts\update-icon-svg.ps1 [-WailsExe wails3]
param(
    [string]$WailsExe = "wails3"
)

$ErrorActionPreference = "Stop"
$Root = Split-Path $PSScriptRoot -Parent

$SvgPath = Join-Path $Root "jpaste-logo.svg"
$PngPath = Join-Path $Root "paste.png"
$IcoPath = Join-Path $Root "build\windows\icon.ico"
$GoDir = Join-Path $PSScriptRoot "rasterize-logo"

if (-not (Test-Path $SvgPath)) {
    Write-Error "jpaste-logo.svg not found at: $SvgPath"
    exit 1
}
if (-not (Test-Path $GoDir)) {
    Write-Error "rasterize-logo/ not found at: $GoDir"
    exit 1
}

# ── 1. Go: SVG → PNG + multi-res .ico ──
Write-Host "=== 1/3 Rasterize SVG → PNG + .ico (Go/oksvg) ===" -ForegroundColor Cyan
Write-Host "  Running: go run ./scripts/rasterize-logo/"

Push-Location $GoDir
try {
    go run .
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Go rasterization failed (exit=$LASTEXITCODE)."
        exit 1
    }
} finally {
    Pop-Location
}

# ── 2. Wails CLI: refine .ico ──
Write-Host ""
Write-Host "=== 2/3 Wails CLI icons ===" -ForegroundColor Cyan

try {
    $null = & $WailsExe version 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  $WailsExe detected, generating icons..."
        & $WailsExe generate icons `
            -input $PngPath `
            -windowsfilename $IcoPath
        if ($LASTEXITCODE -eq 0) {
            Write-Host "  Done." -ForegroundColor Green
        }
    }
} catch {
    Write-Host "  $WailsExe not available — Python .ico is sufficient." -ForegroundColor Yellow
}

# ── 3. Regenerate .syso ──
Write-Host ""
Write-Host "=== 3/3 Regenerate .syso ===" -ForegroundColor Cyan

$sysoPath = Join-Path $Root "wails_windows_amd64.syso"
if (Test-Path $sysoPath) {
    Remove-Item $sysoPath -Force
    Write-Host "  Removed old $sysoPath"
}

Write-Host "  Running: $WailsExe generate syso ..."
Push-Location $Root
try {
    & $WailsExe generate syso `
        -arch amd64 `
        -icon "build\windows\icon.ico" `
        -manifest "build\windows\wails.exe.manifest" `
        -info "build\windows\info.json" `
        -out "wails_windows_amd64.syso"
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  Done." -ForegroundColor Green
    } else {
        Write-Host "  syso generation failed (exit=$LASTEXITCODE). Will regenerate during build." -ForegroundColor Yellow
    }
} finally {
    Pop-Location
}

Write-Host ""
Write-Host "=== Complete ===" -ForegroundColor Green
Write-Host "  Source: jpaste-logo.svg"
Write-Host "  PNG:    paste.png (1024×1024, tray icon)"
Write-Host "  ICO:    build/windows/icon.ico (16/24/32/48/64/128/256)"
Write-Host "  Build:  task build"
