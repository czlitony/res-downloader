# Download ffmpeg binaries for all platforms
# Usage: .\scripts\download-ffmpeg.ps1

$ErrorActionPreference = "Stop"

$ffmpegDir = Join-Path (Join-Path $PSScriptRoot "..") "ffmpeg-bin"
New-Item -ItemType Directory -Force -Path $ffmpegDir | Out-Null

Write-Host "FFmpeg binaries will be downloaded to: $ffmpegDir" -ForegroundColor Cyan

# Windows amd64
$winAmd64Url = "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip"
$winAmd64Zip = Join-Path $ffmpegDir "ffmpeg-win-amd64.zip"
$winAmd64Dir = Join-Path $ffmpegDir "windows-amd64"

Write-Host "`n[1/4] Downloading Windows amd64..." -ForegroundColor Yellow
if (!(Test-Path (Join-Path $winAmd64Dir "ffmpeg.exe"))) {
    Invoke-WebRequest -Uri $winAmd64Url -OutFile $winAmd64Zip
    Expand-Archive -Path $winAmd64Zip -DestinationPath $ffmpegDir -Force
    # Find extracted dir and rename
    $extractedDir = Get-ChildItem -Path $ffmpegDir -Directory | Where-Object { $_.Name -like "ffmpeg-*-essentials*" } | Select-Object -First 1
    if ($extractedDir) {
        New-Item -ItemType Directory -Force -Path $winAmd64Dir | Out-Null
        Copy-Item (Join-Path (Join-Path $extractedDir.FullName "bin") "ffmpeg.exe") -Destination $winAmd64Dir
        Remove-Item $extractedDir.FullName -Recurse -Force
    }
    Remove-Item $winAmd64Zip -Force
    Write-Host "  Done: $(Join-Path $winAmd64Dir 'ffmpeg.exe')" -ForegroundColor Green
} else {
    Write-Host "  Already exists, skipping" -ForegroundColor Gray
}

# macOS (universal from evermeet.cx)
$macUrl = "https://evermeet.cx/ffmpeg/getrelease/ffmpeg/zip"
$macZip = Join-Path $ffmpegDir "ffmpeg-mac.zip"
$macDir = Join-Path $ffmpegDir "darwin-universal"

Write-Host "`n[2/4] Downloading macOS universal..." -ForegroundColor Yellow
if (!(Test-Path (Join-Path $macDir "ffmpeg"))) {
    try {
        Invoke-WebRequest -Uri $macUrl -OutFile $macZip
        New-Item -ItemType Directory -Force -Path $macDir | Out-Null
        Expand-Archive -Path $macZip -DestinationPath $macDir -Force
        Remove-Item $macZip -Force
        Write-Host "  Done: $(Join-Path $macDir 'ffmpeg')" -ForegroundColor Green
    } catch {
        Write-Host "  Download failed, please download manually from https://evermeet.cx/ffmpeg/" -ForegroundColor Red
    }
} else {
    Write-Host "  Already exists, skipping" -ForegroundColor Gray
}

# Linux amd64
$linuxAmd64Url = "https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz"
$linuxAmd64Dir = Join-Path $ffmpegDir "linux-amd64"

Write-Host "`n[3/4] Downloading Linux amd64..." -ForegroundColor Yellow
if (!(Test-Path (Join-Path $linuxAmd64Dir "ffmpeg"))) {
    Write-Host "  Linux version requires manual download (PowerShell doesn't support .tar.xz)" -ForegroundColor Yellow
    Write-Host "  Please download from $linuxAmd64Url and extract to $linuxAmd64Dir" -ForegroundColor Yellow
} else {
    Write-Host "  Already exists, skipping" -ForegroundColor Gray
}

# Linux arm64
$linuxArm64Url = "https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-arm64-static.tar.xz"
$linuxArm64Dir = Join-Path $ffmpegDir "linux-arm64"

Write-Host "`n[4/4] Downloading Linux arm64..." -ForegroundColor Yellow
if (!(Test-Path (Join-Path $linuxArm64Dir "ffmpeg"))) {
    Write-Host "  Linux version requires manual download (PowerShell doesn't support .tar.xz)" -ForegroundColor Yellow
    Write-Host "  Please download from $linuxArm64Url and extract to $linuxArm64Dir" -ForegroundColor Yellow
} else {
    Write-Host "  Already exists, skipping" -ForegroundColor Gray
}

Write-Host "`nDownload complete! Directory structure:" -ForegroundColor Cyan
Write-Host @"
ffmpeg-bin/
+-- windows-amd64/
|   +-- ffmpeg.exe
+-- darwin-universal/
|   +-- ffmpeg
+-- linux-amd64/
|   +-- ffmpeg
+-- linux-arm64/
    +-- ffmpeg
"@

Write-Host "`nCopy the corresponding ffmpeg to build output directory when packaging" -ForegroundColor Cyan
