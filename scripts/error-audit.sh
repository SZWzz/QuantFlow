#!/usr/bin/env bash
# scripts/error-audit.sh
# Scan Go codebase for error handling issues:
#   - Unchecked errors (ignored with _)
#   - Bare panic calls
#   - log.Fatal in library code
#   - Sprintf with %v for errors (should use %w for wrapping)
set -euo pipefail

echo "==> Error handling audit"
echo ""

# 1. Unchecked errors: assignments to '_'
echo "--- Unchecked errors (err assigned to _) ---"
grep -rn "_, err\s*:=" --include="*.go" . | grep -v "_test.go" | grep -v vendor/ | head -20 || echo "(none found)"

echo ""
# 2. Bare panic calls (outside main and test files)
echo "--- Bare panic calls (library code) ---"
grep -rn "\bpanic(" --include="*.go" . | grep -v "_test.go" | grep -v "vendor/" | grep -v "main.go" | grep -v "/main " | head -20 || echo "(none found)"

echo ""
# 3. log.Fatal in non-main packages
echo "--- log.Fatal calls in library packages ---"
grep -rn "log\.Fatal" --include="*.go" . | grep -v "vendor/" | grep -v "cmd/" | grep -v "main.go" | head -20 || echo "(none found)"

echo ""
# 4. %v instead of %w for error wrapping
echo "--- fmt.Errorf with %v for errors (should use %w) ---"
grep -rn 'fmt.Errorf.*%v.*err' --include="*.go" . | grep -v "_test.go" | grep -v vendor/ | head -20 || echo "(none found)"

echo ""
echo "==> Audit complete"
