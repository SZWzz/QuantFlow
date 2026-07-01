.PHONY: build build-full build-frontend test lint vet clean bench coverage distclean

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
	ln -sfn "$(PWD)/python/.venv" build/python/.venv
	@echo "→ Build OK: build/quantflow"

test:
	go test ./... -v -count=1

test-race:
	go test -race ./... -v -count=1

bench:
	go test ./... -bench=. -benchmem

lint:
	go vet ./...

coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

clean:
	rm -f coverage.out

distclean: clean
	rm -rf build/
	rm -rf frontend/dist/
	rm -rf data/
