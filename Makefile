.PHONY: build build-full build-frontend test lint vet clean bench coverage distclean python-setup
.PHONY: release-darwin release-linux release-windows release-checksum release frontend-build
.PHONY: check

# Pre-push gate — mirrors AGENTS.md "full check before commit" and CI jobs.
# Fails fast on the first red step: Go vet+test → frontend typecheck → vitest → pytest.
check:
	go vet ./... && go test ./... -count=1
	cd frontend && npx vue-tsc --noEmit
	cd frontend && npx vitest run
	cd python && .venv/bin/python -m pytest tests/ -q

build:
	go build ./...

build-frontend:
	cd frontend && npm run build -q

# Full macOS build: frontend → Go binary (force rebuild for fresh embed) → Python sidecar
build-full: build-frontend
	go build -a -o build/quantflow .
	rsync -a --delete \
		--exclude='.venv/' \
		--exclude='__pycache__/' \
		--exclude='*.pyc' \
		--exclude='tests/' \
		--exclude='.DS_Store' \
		python/ build/python/
	ln -sfn "$(CURDIR)/python/.venv" build/python/.venv
	@echo "→ Build OK: build/quantflow"

test:
	go test ./... -v -count=1

test-race:
	go test -race ./... -v -count=1

bench:
	go test ./... -bench=. -benchmem

lint:
	go vet ./...
	golangci-lint run ./... --timeout 5m

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

clean:
	rm -f coverage.out

python-setup:
	cd python && python3 -m venv .venv && \
	.venv/bin/pip install --upgrade pip && \
	.venv/bin/pip install -e ".[dev,data]"

distclean: clean
	rm -rf build/
	rm -rf frontend/dist/
	rm -rf data/

# ── Release targets ──────────────────────────────────────────────

VERSION ?= $(shell date +%Y.%-m.%-d)

frontend-build:
	cd frontend && npm ci && npm run build -q

release-darwin:
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o build/quantflow .
	BUILD_VERSION=$(VERSION) ARCH=arm64 ./scripts/darwin-package.sh

release-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o build/quantflow .
	BUILD_VERSION=$(VERSION) ARCH=amd64 ./scripts/linux-package.sh

release-windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o build/quantflow.exe .
	BUILD_VERSION=$(VERSION) ARCH=amd64 pwsh ./scripts/windows-package.ps1

release-checksum:
	./scripts/checksum.sh

release: frontend-build release-darwin release-linux release-checksum
	@echo "Release artifacts ready in build/"

# ── Coverage gate ─────────────────────────────────────────────────
coverage-gate:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	THRESHOLD=60; \
	echo "Backend coverage: $$COVERAGE%"; \
	if [ "$$(echo "$$COVERAGE < $$THRESHOLD" | bc -l)" -eq 1 ]; then \
		echo "❌ Below threshold $$THRESHOLD%"; exit 1; \
	else \
		echo "✅ Meets threshold $$THRESHOLD%"; \
	fi
	cd frontend && npx vitest run --coverage

leak-check:
	go test -race -count=1 ./internal/ws/ ./internal/market/ ./internal/workflow/ ./internal/trading/
