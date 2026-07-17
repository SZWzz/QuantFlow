#!/usr/bin/env bash
# scripts/linux-package.sh
# Linux tar.gz packaging script
# Usage: BUILD_VERSION=2026.7.20 ARCH=amd64 ./scripts/linux-package.sh
#
# Requires: go build output at build/quantflow, frontend dist at frontend/dist/

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
