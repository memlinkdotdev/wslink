# wslink installer (Windows)
# irm https://raw.githubusercontent.com/memlinkdotdev/wslink/main/install.ps1 | iex

$ErrorActionPreference = 'Stop'
$repo       = 'memlinkdotdev/wslink'
$binName    = 'wslink.exe'
$installDir = Join-Path $env:LOCALAPPDATA 'wslink'

# Detect latest release
$release = Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest"
$version  = $release.tag_name

if (-not $version) {
  Write-Error "Could not detect latest release. Check your internet connection."
  exit 1
}

$asset = $release.assets | Where-Object { $_.name -eq 'wslink-windows-amd64.zip' } | Select-Object -First 1
if (-not $asset) {
  Write-Error "No windows-amd64 asset in release $version."
  exit 1
}

Write-Host "Installing wslink $version to $installDir" -ForegroundColor Cyan

# Create install dir
New-Item -ItemType Directory -Force -Path $installDir | Out-Null

# Download + extract
$zipPath = Join-Path $env:TEMP "wslink-$version.zip"
Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $zipPath -UseBasicParsing
Expand-Archive -Path $zipPath -DestinationPath $installDir -Force
Remove-Item $zipPath -Force

# Verify
$binPath = Join-Path $installDir $binName
if (-not (Test-Path $binPath)) {
  Write-Error "Install failed: $binPath not found."
  exit 1
}

# Add to user PATH
$currentPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($currentPath -notlike "*$installDir*") {
  [Environment]::SetEnvironmentVariable('Path', "$currentPath;$installDir", 'User')
  $env:Path = "$env:Path;$installDir"
  Write-Host "Added $installDir to user PATH." -ForegroundColor Green
}

Write-Host ""
Write-Host "wslink $version installed." -ForegroundColor Green
Write-Host ""
Write-Host "Try it:"
Write-Host "  wslink forward 4444              # bridge WSL:4444 <-> Windows:4444"
Write-Host "  wslink forward 4444 --wsl-name Ubuntu"
Write-Host "  wslink --version"
Write-Host ""
