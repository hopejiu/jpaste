$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path $PSScriptRoot -Parent
$BinDir = Join-Path $ProjectRoot "bin"
$BinaryPath = Join-Path $BinDir "jpaste.exe"

Write-Host "=== Step 1: Build ==="
Push-Location $ProjectRoot
try {
    wails3 task windows:build:native
    if ($LASTEXITCODE -ne 0) {
        throw "Build failed (exit code: $LASTEXITCODE)"
    }
} finally {
    Pop-Location
}

# 检查 UPX 是否可用
$upx = Get-Command upx -ErrorAction SilentlyContinue
if (-not $upx) {
    Write-Warning "UPX not found in PATH. Skipping compression."
} else {
    Write-Host "`n=== Step 2: UPX Compress ==="
    if (-not (Test-Path $BinaryPath)) {
        throw "Binary not found: $BinaryPath"
    }

    $beforeSize = (Get-Item $BinaryPath).Length
    Write-Host "Before: $("{0:N0}" -f $beforeSize) bytes"

    & $upx.Source --best --no-color $BinaryPath
    if ($LASTEXITCODE -ne 0) {
        throw "UPX compression failed (exit code: $LASTEXITCODE)"
    }

    $afterSize = (Get-Item $BinaryPath).Length
    $saved = $beforeSize - $afterSize
    $pct = if ($beforeSize -gt 0) { [math]::Round(($saved / $beforeSize) * 100, 1) } else { 0 }

    Write-Host "After:  $("{0:N0}" -f $afterSize) bytes"
    Write-Host "Saved:  $("{0:N0}" -f $saved) bytes ($pct% reduction)"
}

Write-Host "`n=== Step 3: Open Explorer ==="
explorer (Resolve-Path $BinDir)
