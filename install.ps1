# MUSE // One-Line Universal Installer for Windows
# Usage in PowerShell:
#   irm https://raw.githubusercontent.com/KARTHIKKJ369/Tmusic/main/install.ps1 | iex

$ErrorActionPreference = 'Stop'

$Repo = "KARTHIKKJ369/Tmusic"
Write-Host "==> Installing muse for Windows..." -ForegroundColor Cyan

# Detect Architecture
$Arch = $env:PROCESSOR_ARCHITECTURE
switch -Regex ($Arch) {
    "AMD64|x86_64" { $TargetArch = "x86_64"; $AltArch = "amd64" }
    "ARM64"        { $TargetArch = "arm64";  $AltArch = "arm64" }
    default        { $TargetArch = "x86_64"; $AltArch = "amd64" }
}

$InstallDir = Join-Path $env:LOCALAPPDATA "Programs\muse"
if (!(Test-Path -Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

$TempZip = Join-Path $env:TEMP "muse_install.zip"
$TempExtract = Join-Path $env:TEMP "muse_extract"

$Urls = @(
    "https://github.com/$Repo/releases/latest/download/muse_Windows_$TargetArch.zip",
    "https://github.com/$Repo/releases/latest/download/Tmusic_Windows_$TargetArch.zip",
    "https://github.com/$Repo/releases/latest/download/muse_Windows_$AltArch.zip",
    "https://github.com/$Repo/releases/latest/download/Tmusic_Windows_$AltArch.zip"
)

$Downloaded = $false
Write-Host "==> Downloading pre-built release binary (Windows $TargetArch)..." -ForegroundColor Cyan

foreach ($Url in $Urls) {
    try {
        Invoke-WebRequest -Uri $Url -OutFile $TempZip -UseBasicParsing -ErrorAction SilentlyContinue
        if (Test-Path $TempZip) {
            $Downloaded = $true
            Write-Host "==> Downloaded release from: $Url" -ForegroundColor Green
            break
        }
    } catch {}
}

if (-not $Downloaded) {
    # Fallback to go install if Go is installed
    if (Get-Command go -ErrorAction SilentlyContinue) {
        Write-Host "==> Pre-built release not found, building via 'go install'..." -ForegroundColor Yellow
        $env:GOBIN = $InstallDir
        go install "github.com/$Repo/cmd/muse@latest"
        $Downloaded = $true
    } else {
        Write-Error "Could not download pre-built release binary. Please ensure internet access or install Go (https://go.dev)."
        exit 1
    }
} else {
    if (Test-Path $TempExtract) { Remove-Item -Recurse -Force $TempExtract }
    Expand-Archive -Path $TempZip -DestinationPath $TempExtract -Force
    
    $ExeSource = Get-ChildItem -Path $TempExtract -Filter "muse.exe" -Recurse | Select-Object -First 1
    if ($ExeSource) {
        Copy-Item -Path $ExeSource.FullName -Destination (Join-Path $InstallDir "muse.exe") -Force
    } else {
        Write-Error "muse.exe not found in downloaded release."
        exit 1
    }
    
    # Clean up temp
    Remove-Item -Force $TempZip -ErrorAction SilentlyContinue
    Remove-Item -Recurse -Force $TempExtract -ErrorAction SilentlyContinue
}

# Ensure InstallDir is in User PATH
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    $env:Path += ";$InstallDir"
    Write-Host "==> Added $InstallDir to your User PATH environment variable." -ForegroundColor Yellow
}

Write-Host ""
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Green
Write-Host "  ✓ MUSE has been installed successfully to:" -ForegroundColor Green
Write-Host "    $InstallDir\muse.exe" -ForegroundColor White
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Green
Write-Host ""
Write-Host "To get started:" -ForegroundColor Cyan
Write-Host "  1. Set your music folder (only once):"
Write-Host "     muse dir C:\Users\$env:USERNAME\Music" -ForegroundColor Yellow
Write-Host ""
Write-Host "  2. Start listening:"
Write-Host "     muse" -ForegroundColor Yellow
Write-Host ""
