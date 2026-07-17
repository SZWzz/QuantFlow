# scripts/windows-package.ps1
# Windows zip packaging script
# Usage: $env:BUILD_VERSION="2026.7.20"; $env:ARCH="amd64"; .\build\windows-package.ps1
#
# Requires: go build output at build/quantflow.exe, frontend dist at frontend/dist/

param(
  [string]$Version = $env:BUILD_VERSION,
  [string]$Arch = $env:ARCH
)

if (-not $Version) {
  Write-Error "BUILD_VERSION is required"
  exit 1
}

if (-not $Arch) {
  $Arch = "amd64"
}

Write-Host "==> Creating Windows package for QuantFlow $Version ($Arch)"

$staging = "build\quantflow-$Version-windows-$Arch"
New-Item -ItemType Directory -Force -Path $staging | Out-Null

Copy-Item "build\quantflow.exe" "$staging\"
Copy-Item "frontend\dist" "$staging\frontend\" -Recurse -Force

if (Test-Path "resources") {
  Copy-Item "resources\*" "$staging\" -Recurse -Force
}

Copy-Item "README.md" "$staging\" -Force -ErrorAction SilentlyContinue
Copy-Item "LICENSE" "$staging\" -Force -ErrorAction SilentlyContinue

$zipPath = "build\quantflow-$Version-windows-$Arch.zip"
Compress-Archive -Path "$staging\*" -DestinationPath $zipPath -Force

Write-Host "==> Done: $zipPath"
