MODULE  := github.com/blsssss/mamori
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null)
LDFLAGS := -X $(MODULE)/internal/version.Version=$(VERSION) -X $(MODULE)/internal/version.Commit=$(COMMIT)

.PHONY: check test lint frontend build installer dev bindings clean

check:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }
	GOOS=windows go vet ./...
	GOOS=linux go vet ./internal/...
	go test -short ./...
	cd frontend && npm run lint && npm run check && npm test

test:
	go test -race -count=1 ./...

lint:
	golangci-lint run

frontend:
	cd frontend && npm ci && npm run build

build:
	wails build -clean -trimpath -ldflags "$(LDFLAGS)"

installer:
	wails build -clean -trimpath -webview2 embed -nsis -ldflags "$(LDFLAGS)"

dev:
	wails dev

bindings:
	wails generate module

clean:
	rm -rf build/bin frontend/dist/assets
