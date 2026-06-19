.PHONY: build test lint vet clean bench coverage

build:
	go build ./...

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
	rm -rf data/
