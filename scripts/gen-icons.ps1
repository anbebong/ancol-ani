<#
.SYNOPSIS
  Regenerates every app icon / logo file from one source image.
  See docs/ICONS.md for what each output file is used for.

.EXAMPLE
  # from a new high-res logo (recommended: square PNG >= 1024px, transparent background)
  powershell -ExecutionPolicy Bypass -File scripts/gen-icons.ps1 -Source path\to\logo.png

.EXAMPLE
  # rebuild from the current ancol.ico (only 64px inside, so large sizes come out blurry)
  powershell -ExecutionPolicy Bypass -File scripts/gen-icons.ps1 -Source ancol.ico

.EXAMPLE
  # dry run into another folder to inspect the results first
  powershell -ExecutionPolicy Bypass -File scripts/gen-icons.ps1 -Source logo.png -OutRoot $env:TEMP\icons-test
#>
param(
    [Parameter(Mandatory = $true)][string]$Source,
    # Repo root to write into. Defaults to the repo this script lives in.
    [string]$OutRoot = (Split-Path -Parent $PSScriptRoot)
)

$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Drawing

# Light tile behind the (dark) glyph for places shown on dark backgrounds:
# browser tabs in dark mode, the portal's dark UI, iOS home screen (which
# fills transparency with black). Same color as AppLogo.vue's tile.
$TileColor = [System.Drawing.Color]::FromArgb(255, 241, 238, 252) # #f1eefc

# ---------- load source ----------
$Source = (Resolve-Path $Source).Path
if ($Source -match '\.ico$') {
    $ico = New-Object System.Drawing.Icon($Source, 256, 256) # picks the largest image inside
    $src = $ico.ToBitmap()
} else {
    $src = [System.Drawing.Bitmap]::FromFile($Source)
}
Write-Host "source: $Source ($($src.Width)x$($src.Height))"
if ($src.Width -lt 256) {
    Write-Warning "Source is only $($src.Width)px - sizes above that will be upscaled and look blurry. Use a >= 1024px PNG if you have one."
}

# ---------- helpers ----------
function New-Png([int]$size, [bool]$tile, [double]$padRatio) {
    $bmp = New-Object System.Drawing.Bitmap($size, $size, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
    $g.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
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
    $g.DrawImage($src, $pad, $pad, $size - 2 * $pad, $size - 2 * $pad)
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

$src.Dispose()
Write-Host ""
Write-Host "Done. Next steps:"
Write-Host "  1. Bump ?v= on the icon <link> tags in cmd/syncserver/portal/index.html and web/index.html so browsers drop the cached favicon."
Write-Host "  2. Rebuild: cd $portal && npm run build ; cd web && npm run build ; wails build"
