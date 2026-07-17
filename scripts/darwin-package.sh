#!/usr/bin/env bash
# scripts/darwin-package.sh
# macOS .app bundle + DMG creation script
# Usage: BUILD_VERSION=2026.7.20 ARCH=arm64 ./scripts/darwin-package.sh
#
# Requires: go build output at build/quantflow, frontend dist at frontend/dist/

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

# Generate Info.plist from template
sed "s/__VERSION__/${VERSION}/g" resources/Info.plist.template > "${BUNDLE}/Contents/Info.plist"

echo "==> Creating DMG"
DMG="build/QuantFlow-${VERSION}-${ARCH}.dmg"
hdiutil create -volname "QuantFlow ${VERSION}" \
  -srcfolder "${BUNDLE}" \
  -ov -format UDZO \
  "${DMG}"

echo "==> Done: ${DMG}"
