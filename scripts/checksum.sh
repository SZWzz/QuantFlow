#!/usr/bin/env bash
# scripts/checksum.sh
# Generate SHA256 checksums for all build artifacts.
# Usage: ./scripts/checksum.sh

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
