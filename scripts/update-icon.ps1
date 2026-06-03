# update-icon.ps1 — Rebuild Windows icon from paste.png
# Usage: .\scripts\update-icon.ps1 [-WailsExe wails3]
param(
    [string]$WailsExe = "wails3"
)

$ErrorActionPreference = "Stop"
$Root = Split-Path $PSScriptRoot -Parent

$PngPath    = Join-Path $Root "paste.png"
$IcoPath    = Join-Path $Root "build\windows\icon.ico"

if (-not (Test-Path $PngPath)) {
    Write-Error "paste.png not found at: $PngPath"
    exit 1
}

Write-Host "=== 1/3 Generate .ico from paste.png ===" -ForegroundColor Cyan

# Try $WailsExe first (requires wails3 CLI installed)
$wailsOk = $false
try {
    $wailsVersion = & $WailsExe version 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  Using $WailsExe to generate icons..."
        & $WailsExe generate icons `
            -input $PngPath `
            -windowsfilename $IcoPath
        if ($LASTEXITCODE -eq 0) {
            Write-Host "  Done." -ForegroundColor Green
            $wailsOk = $true
        } else {
            Write-Host "  $WailsExe failed, falling back to .NET..." -ForegroundColor Yellow
        }
    }
} catch {
    Write-Host "  $WailsExe not found, using .NET fallback..." -ForegroundColor Yellow
}

if (-not $wailsOk) {
    # Fallback: generate .ico with .NET System.Drawing
    Write-Host "  Generating .ico via System.Drawing..."
    Add-Type -AssemblyName System.Drawing

    $bmp = [System.Drawing.Bitmap]::FromFile($PngPath)
    $sizes = @(16, 24, 32, 48, 64, 128, 256)

    # Build multi-resolution .ico
    $mem = New-Object System.IO.MemoryStream
    $writer = New-Object System.IO.BinaryWriter($mem)

    # ICO header
    $writer.Write([uint16]0)           # Reserved
    $writer.Write([uint16]1)           # ICO type
    $writer.Write([uint16]$sizes.Count) # Number of images

    $images = @()
    $dataOffset = 6 + 16 * $sizes.Count  # Header + directory entries

    foreach ($size in $sizes) {
        $resized = New-Object System.Drawing.Bitmap($bmp, $size, $size)
        $pngMem = New-Object System.IO.MemoryStream
        $resized.Save($pngMem, [System.Drawing.Imaging.ImageFormat]::Png)
        $pngBytes = $pngMem.ToArray()
        $pngMem.Close()
        $resized.Dispose()

        $entrySize = if ($size -ge 256) { 0 } else { $size }
        $writer.Write([byte]$entrySize)     # Width
        $writer.Write([byte]$entrySize)     # Height
        $writer.Write([byte]0)              # Palette size
        $writer.Write([byte]0)              # Reserved
        $writer.Write([uint16]1)            # Color planes
        $writer.Write([uint16]32)           # Bits per pixel
        $writer.Write([uint32]$pngBytes.Length)  # Image size
        $writer.Write([uint32]$dataOffset)  # Offset

        $images += , $pngBytes
        $dataOffset += $pngBytes.Length
    }

    foreach ($img in $images) {
        $writer.Write($img)
    }

    $writer.Flush()
    [System.IO.File]::WriteAllBytes($IcoPath, $mem.ToArray())
    $writer.Close()
    $mem.Close()
    $bmp.Dispose()

    Write-Host "  Done (System.Drawing fallback)." -ForegroundColor Green
}

Write-Host ""
Write-Host "=== 2/3 Replace tray icon (paste.png is embedded, no action needed) ===" -ForegroundColor Cyan
Write-Host "  paste.png is embedded via //go:embed in main.go — just replace the file."

Write-Host ""
Write-Host "=== 3/3 Regenerate .syso (Windows build resource) ===" -ForegroundColor Cyan

$sysoPath = Join-Path $Root "wails_windows_amd64.syso"
if (Test-Path $sysoPath) {
    Remove-Item $sysoPath -Force
    Write-Host "  Removed old $sysoPath"
}

Write-Host "  Running: $WailsExe generate syso -arch amd64 -icon build/windows/icon.ico ..."
Push-Location $Root
try {
    & $WailsExe generate syso `
        -arch amd64 `
        -icon "build\windows\icon.ico" `
        -manifest "build\windows\wails.exe.manifest" `
        -info "build\windows\info.json" `
        -out "wails_windows_amd64.syso"
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  Done. $sysoPath regenerated." -ForegroundColor Green
    } else {
        Write-Host "  syso generation failed (exit=$LASTEXITCODE). It will be regenerated during `task build`." -ForegroundColor Yellow
    }
} finally {
    Pop-Location
}

Write-Host ""
Write-Host "=== Complete ===" -ForegroundColor Green
Write-Host "  - build/windows/icon.ico  — updated from paste.png"
Write-Host "  - paste.png               — tray icon (embedded, already in place)"
Write-Host "  - Rebuild with: task build"
