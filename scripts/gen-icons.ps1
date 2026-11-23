<#
.SYNOPSIS
  Regenerates every app icon / logo file from the real vector logo
  (ui/assets/logo.svg - the "A" + node cluster, transparent background),
  rasterized at whatever size each output needs via @resvg/resvg-js (true
  vector rendering, so nothing is ever blurry/upscaled). See docs/ICONS.md
  for what each output file is used for.

.EXAMPLE
  powershell -ExecutionPolicy Bypass -File scripts/gen-icons.ps1

.EXAMPLE
  # dry run into another folder to inspect the results first
  powershell -ExecutionPolicy Bypass -File scripts/gen-icons.ps1 -OutRoot $env:TEMP\icons-test
#>
param(
    # Repo root to write into. Defaults to the repo this script lives in.
    [string]$OutRoot = (Split-Path -Parent $PSScriptRoot),
    [string]$LogoSvg = (Join-Path (Split-Path -Parent $PSScriptRoot) 'ui\assets\logo.svg')
)

$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Drawing

$ScriptsDir = $PSScriptRoot
if (-not (Test-Path (Join-Path $ScriptsDir 'node_modules\@resvg'))) {
    Write-Host "Installing scripts/ tooling (npm install in $ScriptsDir)..."
    Push-Location $ScriptsDir
    npm install | Out-Null
    Pop-Location
}

# Light tile behind the glyph for places shown on dark backgrounds: browser
# tabs in dark mode, the portal's dark UI, iOS home screen (which fills
# transparency with black). Same color as AppLogo.vue's tile.
$TileColor = [System.Drawing.Color]::FromArgb(255, 241, 238, 252) # #f1eefc

# ---------- helpers ----------

# Rasterizes $LogoSvg at an exact pixel size via the Node/resvg helper
# (GDI+ has no SVG support of its own) and returns it as a Bitmap.
function Render-Glyph([int]$size) {
    $tmp = [IO.Path]::Combine([IO.Path]::GetTempPath(), "ancol-logo-$size-$([guid]::NewGuid().ToString('N')).png")
    node (Join-Path $ScriptsDir 'render-svg.js') $LogoSvg $tmp $size
    if ($LASTEXITCODE -ne 0 -or -not (Test-Path $tmp)) { throw "render-svg.js failed for size $size" }
    $bmp = [System.Drawing.Bitmap]::FromFile($tmp)
    $copy = New-Object System.Drawing.Bitmap($bmp) # detach from the file so it can be deleted
    $bmp.Dispose()
    Remove-Item $tmp -ErrorAction SilentlyContinue
    return $copy
}

function New-Png([int]$size, [bool]$tile, [double]$padRatio) {
    $bmp = New-Object System.Drawing.Bitmap($size, $size, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
    $g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
    $g.Clear([System.Drawing.Color]::Transparent)
    if ($tile) {
        # rounded-corner tile
        $r = [Math]::Max(2, [int]($size * 0.22)); $d = 2 * $r
        $p = New-Object System.Drawing.Drawing2D.GraphicsPath
        $p.AddArc(0, 0, $d, $d, 180, 90)
        $p.AddArc($size - $d - 1, 0, $d, $d, 270, 90)
        $p.AddArc($size - $d - 1, $size - $d - 1, $d, $d, 0, 90)
        $p.AddArc(0, $size - $d - 1, $d, $d, 90, 90)
        $p.CloseFigure()
        $g.FillPath((New-Object System.Drawing.SolidBrush($TileColor)), $p)
    }
    $pad = [int][Math]::Round($size * $padRatio)
    if ($tile -and $pad -lt 1) { $pad = 1 }
    $glyphSize = $size - 2 * $pad
    $glyph = Render-Glyph $glyphSize
    $g.DrawImage($glyph, $pad, $pad, $glyphSize, $glyphSize)
    $glyph.Dispose()
    $g.Dispose()
    $ms = New-Object IO.MemoryStream
    $bmp.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png)
    $bmp.Dispose()
    return , $ms.ToArray()
}

# .ico container holding PNG-compressed images (supported by Windows Vista+ and all browsers)
function Write-Ico([string]$path, [int[]]$sizes, [bool]$tile, [double]$padRatio) {
    $pngs = @{}
    foreach ($s in $sizes) { $pngs[$s] = New-Png $s $tile $padRatio }
    $ms = New-Object IO.MemoryStream
    $w = New-Object IO.BinaryWriter($ms)
    $w.Write([UInt16]0); $w.Write([UInt16]1); $w.Write([UInt16]$sizes.Count)
    $offset = 6 + 16 * $sizes.Count
    foreach ($s in $sizes) {
        $len = $pngs[$s].Length
        $w.Write([byte]($s % 256)); $w.Write([byte]($s % 256)) # 256 is stored as 0
        $w.Write([byte]0); $w.Write([byte]0)
        $w.Write([UInt16]1); $w.Write([UInt16]32)
        $w.Write([UInt32]$len); $w.Write([UInt32]$offset)
        $offset += $len
    }
    foreach ($s in $sizes) { $w.Write($pngs[$s]) }
    $w.Flush()
    Save-Bytes $path $ms.ToArray()
}

function Save-Bytes([string]$rel, [byte[]]$bytes) {
    $full = Join-Path $OutRoot $rel
    New-Item -ItemType Directory -Force (Split-Path -Parent $full) | Out-Null
    [IO.File]::WriteAllBytes($full, $bytes)
    Write-Host ("  {0,-48} {1,8:N0} bytes" -f $rel, $bytes.Length)
}

# ---------- outputs ----------
Write-Host "source: $LogoSvg"
Write-Host "writing into: $OutRoot"

# Desktop app (Wails). Transparent background, no tile.
$appIcoSizes = 16, 24, 32, 48, 64, 128, 256
Write-Ico 'build/windows/icon.ico' $appIcoSizes $false 0   # exe icon + NSIS installer
Write-Ico 'ancol.ico'              $appIcoSizes $false 0   # master copy at repo root
Save-Bytes 'build/appicon.png' (New-Png 1024 $false 0)     # Wails source for macOS .icns / regenerating icon.ico

# Browser tab / home-screen icons get the light tile. Same set for both
# web UIs: the server portal and the client's local settings page.
foreach ($site in 'cmd/syncserver/portal', 'web') {
    Write-Ico  "$site/public/favicon.ico"          (16, 32, 48, 64) $true 0.10
    Save-Bytes "$site/public/favicon-32.png"       (New-Png 32 $true 0.10)
    Save-Bytes "$site/public/apple-touch-icon.png" (New-Png 180 $true 0.12)
}

# In-page logo shared by both web UIs: transparent, ui/components/AppLogo.vue
# draws the light tile itself in CSS.
Save-Bytes 'ui/assets/logo.png' (New-Png 128 $false 0)

Write-Host ""
Write-Host "Done. Next steps:"
Write-Host "  1. Bump ?v= on the icon <link> tags in cmd/syncserver/portal/index.html and web/index.html so browsers drop the cached favicon."
Write-Host "  2. Rebuild: cd cmd/syncserver/portal && npm run build ; cd web && npm run build ; wails build"
