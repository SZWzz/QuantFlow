# Release Pipeline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Automate cross-platform build, package, checksum, and release to GitHub Releases on `git tag v*`.

**Architecture:** GitHub Actions matrix build across 6 platform/arch combinations (macOS arm64/amd64, Linux arm64/amd64, Windows arm64/amd64). Each builds Go binary + frontend, packages into platform-specific format (.app+.dmg / .tar.gz / .zip), generates SHA256 checksums, and uploads to GitHub Releases via softprops/action-gh-release.

**Tech Stack:** GitHub Actions, Go 1.25+, Node 20, bash (macOS hdiutil, tar, zip), WiX Toolset (optional, .msi deferred)

## Global Constraints

- No new external build dependencies (everything available on GitHub Actions runners)
- Version must match today's date in format `YYYY.M.D` per CLAUDE.md rule 3
- Release triggered by `git tag v*` push only (not on every commit)
- Build matrix: 6 combinations, all built in parallel
- Code signing deferred (requires Apple Developer Certificate on self-hosted runner)
- .msi packaging deferred to phase 2 (WiX Toolset learning cost)

---

### Task 1: Release Workflow (release.yml)

**Files:**
- Create: `.github/workflows/release.yml`

**Interfaces:**
- Consumes: `git tag v*` push, GitHub Actions runners
- Produces: Release workflow with build matrix, packaging, checksums, upload

- [ ] **Step 1: Write the workflow file**

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  release:
    strategy:
      fail-fast: false
      matrix:
        include:
          - os: macos-latest
            goos: darwin
            goarch: arm64
            ext: zip
            label: macOS ARM64
          - os: macos-13
            goos: darwin
            goarch: amd64
            ext: zip
            label: macOS AMD64
          - os: ubuntu-latest
            goos: linux
            goarch: amd64
            ext: tar.gz
            label: Linux AMD64
          - os: ubuntu-latest
            goos: linux
            goarch: arm64
            ext: tar.gz
            label: Linux ARM64
          - os: windows-latest
            goos: windows
            goarch: amd64
            ext: zip
            label: Windows AMD64
          - os: windows-latest
            goos: windows
            goarch: arm64
            ext: zip
            label: Windows ARM64

    runs-on: ${{ matrix.os }}

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'
          check-latest: true

      - name: Setup Node
        uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'
          cache-dependency-path: frontend/package-lock.json

      - name: Install frontend dependencies
        run: npm ci
        working-directory: frontend

      - name: Build frontend
        run: npm run build -q
        working-directory: frontend

      - name: Build Go binary
        run: |
          GOOS=${{ matrix.goos }} GOARCH=${{ matrix.goarch }} \
          CGO_ENABLED=0 \
          go build -a -ldflags="-s -w -X quantflow/internal/crash.appVersion=${GITHUB_REF_NAME#v} -X quantflow/internal/crash.buildMode=release" \
            -o build/quantflow${{ matrix.goos == 'windows' && '.exe' || '' }} .

      - name: Package (macOS)
        if: matrix.goos == 'darwin'
        run: |
          VERSION="${GITHUB_REF_NAME#v}"
          mkdir -p "build/QuantFlow.app/Contents/MacOS"
          mkdir -p "build/QuantFlow.app/Contents/Resources"
          cp "build/quantflow${{ matrix.goos == 'windows' && '.exe' || '' }}" "build/QuantFlow.app/Contents/MacOS/"
          cp -r frontend/dist "build/QuantFlow.app/Contents/Resources/frontend"
          cp resources/icon.icns "build/QuantFlow.app/Contents/Resources/" 2>/dev/null || true
          cat > "build/QuantFlow.app/Contents/Info.plist" << 'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key><string>QuantFlow</string>
    <key>CFBundleDisplayName</key><string>QuantFlow Terminal</string>
    <key>CFBundleIdentifier</key><string>com.quantflow.terminal</string>
    <key>CFBundleVersion</key><string>${VERSION}</string>
    <key>CFBundleShortVersionString</key><string>${VERSION}</string>
    <key>CFBundleExecutable</key><string>quantflow</string>
    <key>CFBundleIconFile</key><string>icon.icns</string>
    <key>CFBundlePackageType</key><string>APPL</string>
    <key>NSHighResolutionCapable</key><true/>
    <key>LSMinimumSystemVersion</key><string>11.0</string>
</dict>
</plist>
PLIST
          hdiutil create -volname "QuantFlow ${VERSION}" \
            -srcfolder build/QuantFlow.app \
            -ov -format UDZO \
            "build/QuantFlow-${VERSION}-${{ matrix.goarch }}.dmg"

      - name: Package (Linux)
        if: matrix.goos == 'linux'
        run: |
          VERSION="${GITHUB_REF_NAME#v}"
          mkdir -p build/quantflow-${VERSION}-linux-${{ matrix.goarch }}
          cp build/quantflow build/quantflow-${VERSION}-linux-${{ matrix.goarch }}/
          cp -r resources build/quantflow-${VERSION}-linux-${{ matrix.goarch }}/ 2>/dev/null || true
          cp README.md build/quantflow-${VERSION}-linux-${{ matrix.goarch }}/ 2>/dev/null || true
          tar -czf "build/quantflow-${VERSION}-linux-${{ matrix.goarch }}.tar.gz" \
            -C build "quantflow-${VERSION}-linux-${{ matrix.goarch }}"

      - name: Package (Windows)
        if: matrix.goos == 'windows'
        run: |
          $version = "$env:GITHUB_REF_NAME" -replace '^v', ''
          Compress-Archive -Path build/quantflow.exe,frontend/dist,resources -DestinationPath "build/quantflow-${version}-windows-${{ matrix.goarch }}.zip"

      - name: Generate checksums
        run: |
          cd build
          if [[ "${{ matrix.os }}" == "windows-latest" ]]; then
            Get-ChildItem -File | Where-Object { $_.Extension -ne '.sha256' } | ForEach-Object {
              $hash = (Get-FileHash $_.FullName -Algorithm SHA256).Hash.ToLower()
              "$hash  $($_.Name)" | Out-File -Append checksums.txt -Encoding ascii
            }
          else
            for f in *; do
              [ -f "$f" ] && echo "$(sha256sum "$f" | cut -d' ' -f1)  $f"
            done > checksums.txt
          fi
          cat checksums.txt

      - name: Upload Release Assets
        uses: softprops/action-gh-release@v2
        with:
          files: build/*
          generate_release_notes: true
          fail_on_unmatched_files: false
```

- [ ] **Step 2: Verify workflow syntax**

```bash
# Validate YAML syntax
cd /Volumes/shenzy/vibe_coding/QuantFlow
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml')); print('YAML OK')"
```

Expected: `YAML OK`

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "feat(ci): add GitHub Actions release pipeline with cross-platform build matrix"
```

---

### Task 2: macOS Build Script (darwin-package.sh)

**Files:**
- Create: `build/darwin-package.sh`
- Create: `resources/Info.plist` (template)

**Interfaces:**
- Consumes: `BUILD_VERSION` env, compiled binary at `build/quantflow`, frontend dist at `frontend/dist/`
- Produces: `build/QuantFlow.app` bundle and `build/QuantFlow-${VERSION}-${ARCH}.dmg`

- [ ] **Step 1: Create the packaging script**

```bash
#!/usr/bin/env bash
# build/darwin-package.sh
# macOS .app bundle + DMG creation script
# Usage: BUILD_VERSION=2026.7.20 ARCH=arm64 ./build/darwin-package.sh

set -euo pipefail

VERSION="${BUILD_VERSION:?BUILD_VERSION is required}"
ARCH="${ARCH:-arm64}"
BUNDLE="build/QuantFlow.app"

echo "==> Creating .app bundle at ${BUNDLE}"
mkdir -p "${BUNDLE}/Contents/MacOS"
mkdir -p "${BUNDLE}/Contents/Resources"

cp "build/quantflow" "${BUNDLE}/Contents/MacOS/"
cp -r "frontend/dist" "${BUNDLE}/Contents/Resources/frontend"

# Copy icon if present
if [ -f "resources/icon.icns" ]; then
  cp "resources/icon.icns" "${BUNDLE}/Contents/Resources/"
fi

# Generate Info.plist
cat > "${BUNDLE}/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key><string>QuantFlow</string>
  <key>CFBundleDisplayName</key><string>QuantFlow Terminal</string>
  <key>CFBundleIdentifier</key><string>com.quantflow.terminal</string>
  <key>CFBundleVersion</key><string>${VERSION}</string>
  <key>CFBundleShortVersionString</key><string>${VERSION}</string>
  <key>CFBundleExecutable</key><string>quantflow</string>
  <key>CFBundleIconFile</key><string>icon.icns</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>NSHighResolutionCapable</key><true/>
  <key>LSMinimumSystemVersion</key><string>11.0</string>
</dict>
</plist>
EOF

echo "==> Creating DMG"
DMG="build/QuantFlow-${VERSION}-${ARCH}.dmg"
hdiutil create -volname "QuantFlow ${VERSION}" \
  -srcfolder "${BUNDLE}" \
  -ov -format UDZO \
  "${DMG}"

echo "==> Done: ${DMG}"
```

```bash
chmod +x /Volumes/shenzy/vibe_coding/QuantFlow/build/darwin-package.sh
```

- [ ] **Step 2: Commit**

```bash
git add build/darwin-package.sh
git commit -m "feat(build): add macOS packaging script for .app bundle and DMG"
```

---

### Task 3: Linux Build Script (linux-package.sh)

**Files:**
- Create: `build/linux-package.sh`

**Interfaces:**
- Consumes: `BUILD_VERSION` env, compiled binary at `build/quantflow`, frontend dist
- Produces: `build/quantflow-${VERSION}-linux-${ARCH}.tar.gz`

- [ ] **Step 1: Create the packaging script**

```bash
#!/usr/bin/env bash
# build/linux-package.sh
# Linux tar.gz packaging script
# Usage: BUILD_VERSION=2026.7.20 ARCH=amd64 ./build/linux-package.sh

set -euo pipefail

VERSION="${BUILD_VERSION:?BUILD_VERSION is required}"
ARCH="${ARCH:-amd64}"
STAGING="build/quantflow-${VERSION}-linux-${ARCH}"

echo "==> Creating staging directory: ${STAGING}"
mkdir -p "${STAGING}"

cp "build/quantflow" "${STAGING}/"

# Copy resources if present
if [ -d "resources" ]; then
  cp -r resources "${STAGING}/" 2>/dev/null || true
fi

# Copy docs
cp README.md "${STAGING}/" 2>/dev/null || true
cp LICENSE "${STAGING}/" 2>/dev/null || true

# Copy frontend dist
cp -r "frontend/dist" "${STAGING}/frontend" 2>/dev/null || true

echo "==> Creating tar.gz"
TARBALL="build/quantflow-${VERSION}-linux-${ARCH}.tar.gz"
tar -czf "${TARBALL}" -C build "quantflow-${VERSION}-linux-${ARCH}"

echo "==> Done: ${TARBALL}"
```

```bash
chmod +x /Volumes/shenzy/vibe_coding/QuantFlow/build/linux-package.sh
```

- [ ] **Step 2: Commit**

```bash
git add build/linux-package.sh
git commit -m "feat(build): add Linux packaging script for tar.gz"
```

---

### Task 4: Windows Build Script (windows-package.ps1)

**Files:**
- Create: `build/windows-package.ps1`

**Interfaces:**
- Consumes: `BUILD_VERSION` env, compiled binary `build/quantflow.exe`, frontend dist
- Produces: `build/quantflow-${VERSION}-windows-${ARCH}.zip`

- [ ] **Step 1: Create the packaging script**

```powershell
# build/windows-package.ps1
# Windows zip packaging script
# Usage: $env:BUILD_VERSION="2026.7.20"; $env:ARCH="amd64"; .\build\windows-package.ps1

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

Copy-Item "README.md" "$staging\" -Force 2>$null
Copy-Item "LICENSE" "$staging\" -Force 2>$null

$zipPath = "build\quantflow-$Version-windows-$Arch.zip"
Compress-Archive -Path "$staging\*" -DestinationPath $zipPath -Force

Write-Host "==> Done: $zipPath"
```

- [ ] **Step 2: Commit**

```bash
git add build/windows-package.ps1
git commit -m "feat(build): add Windows packaging script for zip"
```

---

### Task 5: Checksum Generation + Release Automation Script

**Files:**
- Create: `build/checksum.sh`
- Modify: `Makefile` (add release targets)

**Interfaces:**
- Consumes: built artifacts in `build/`
- Produces: `build/checksums.txt`

- [ ] **Step 1: Create checksum script**

```bash
#!/usr/bin/env bash
# build/checksum.sh
# Generate SHA256 checksums for all build artifacts.
# Usage: ./build/checksum.sh

set -euo pipefail

OUTPUT="build/checksums.txt"
echo "==> Generating SHA256 checksums"

rm -f "${OUTPUT}"
for f in build/*; do
  if [ -f "$f" ] && [ "$(basename "$f")" != "checksums.txt" ]; then
    echo "$(sha256sum "$f" | cut -d' ' -f1)  $(basename "$f")" >> "${OUTPUT}"
  fi
done

cat "${OUTPUT}"
echo "==> Done: ${OUTPUT}"
```

```bash
chmod +x /Volumes/shenzy/vibe_coding/QuantFlow/build/checksum.sh
```

Add to Makefile:

```makefile
# Add to existing Makefile

# Release targets
VERSION ?= $(shell date +%Y.%-m.%-d)

.PHONY: release-darwin release-linux release-windows release-checksum release

release-darwin:
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o build/quantflow .
	BUILD_VERSION=$(VERSION) ARCH=arm64 ./build/darwin-package.sh

release-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o build/quantflow .
	BUILD_VERSION=$(VERSION) ARCH=amd64 ./build/linux-package.sh

release-windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o build/quantflow.exe .
	BUILD_VERSION=$(VERSION) ARCH=amd64 ./build/windows-package.ps1

release-checksum:
	./build/checksum.sh

release: frontend-build release-darwin release-linux release-checksum
	@echo "Release artifacts ready in build/"

frontend-build:
	cd frontend && npm ci && npm run build -q
```

- [ ] **Step 2: Commit**

```bash
git add build/checksum.sh Makefile
git commit -m "feat(build): add checksum generation and Makefile release targets"
```

---

### Task 6: Homebrew Tap Formula (Optional)

**Files:**
- Create: `build/homebrew/quantflow.rb`

**Interfaces:**
- Consumes: GitHub Release assets
- Produces: Homebrew formula for `brew install quantflow`

- [ ] **Step 1: Create Homebrew formula**

```ruby
# build/homebrew/quantflow.rb
# Homebrew formula for QuantFlow Terminal
# Usage: brew tap SZWzz/homebrew-tap && brew install quantflow

class Quantflow < Formula
  desc "Dual-mode quantitative finance terminal (Bloomberg-style + workflow orchestration)"
  homepage "https://github.com/SZWzz/QuantFlow"
  license "AGPL-3.0"
  version "2026.7.20"

  if Hardware::CPU.arm?
    url "https://github.com/SZWzz/QuantFlow/releases/download/v2026.7.20/quantflow-darwin-arm64.zip"
    sha256 "0000000000000000000000000000000000000000000000000000000000000000" # REPLACE with actual
  else
    url "https://github.com/SZWzz/QuantFlow/releases/download/v2026.7.20/quantflow-darwin-amd64.zip"
    sha256 "0000000000000000000000000000000000000000000000000000000000000000" # REPLACE with actual
  end

  depends_on "go" => :optional

  def install
    app_path = "QuantFlow.app"
    prefix.install app_path
    bin.write_exec_script "#{prefix}/#{app_path}/Contents/MacOS/quantflow"
  end

  def caveats
    <<~EOS
      QuantFlow Terminal installed to #{prefix}/QuantFlow.app.
      Run with: open -a QuantFlow
      Or from command line: quantflow

      Python sidecar (optional, for ML/AI features):
        cd #{prefix}/QuantFlow.app/Contents/Resources/frontend/../python
        python3 -m venv venv
        source venv/bin/activate
        pip install -r requirements.txt
        python -m src.server
    EOS
  end
end
```

- [ ] **Step 2: Commit**

```bash
git add build/homebrew/quantflow.rb
git commit -m "feat(build): add Homebrew formula for macOS distribution"
```
