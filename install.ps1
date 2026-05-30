param(
    [string]$Version = ""
)

$ErrorActionPreference = "Stop"
$Repo = "memlinkdotdev/wslink"

if (-not $Version) {
    # Default to latest known version; API override below as a bonus
    $Version = "v0.1.3"
    try {
        $api = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -UseBasicParsing
        $Version = $api.tag_name
    } catch {
        # fallback to hardcoded default
    }
}

$DestDir = "$env:LOCALAPPDATA\wslink"
$ExePath = "$DestDir\wslink.exe"

Write-Host "wslink installer" -ForegroundColor Cyan
Write-Host "===============" -ForegroundColor Cyan
Write-Host "Repo:  $Repo"
Write-Host "Tag:   $Version"
Write-Host "Dest:  $DestDir"
Write-Host ""

# 1. Create destination
New-Item -ItemType Directory -Force -Path $DestDir | Out-Null

# 2. Download release
$Url = "https://github.com/$Repo/releases/download/$Version/wslink.exe"
Write-Host "Downloading $Url ..." -ForegroundColor Yellow
try {
    Invoke-WebRequest -Uri $Url -OutFile "$DestDir\wslink.tmp" -UseBasicParsing
    Move-Item -Force "$DestDir\wslink.tmp" $ExePath
} catch {
    Write-Host "Download failed: $_" -ForegroundColor Red
    exit 1
}
Write-Host "  Saved to $ExePath" -ForegroundColor Green

# 3. Add to user PATH
$RegPath = "HKCU:\Environment"
$CurrentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($CurrentPath -split ";" -notcontains $DestDir) {
    $NewPath = if ($CurrentPath) { "$CurrentPath;$DestDir" } else { $DestDir }
    [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
    Write-Host "  Added to user PATH" -ForegroundColor Green
} else {
    Write-Host "  Already in PATH" -ForegroundColor Gray
}

# 4. Done
Write-Host ""
Write-Host "Installed! Restart your terminal and run:" -ForegroundColor Cyan
Write-Host "  wslink forward 4444" -ForegroundColor White
Write-Host ""
