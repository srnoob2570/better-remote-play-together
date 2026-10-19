GO ?= GOTOOLCHAIN=go1.27.2 go
VERSION ?= 0.1.0
BINARY ?= dist/better-rpt
HELPER ?= dist/better-rpt-helper
ARGS ?=

.PHONY: dev build helper test test-race vet

dev:
	$(GO) run ./cmd/better-rpt $(ARGS)

build:
	mkdir -p $(dir $(BINARY))
	CGO_ENABLED=1 $(GO) build -trimpath -buildvcs=false \
		-ldflags "-s -w -X main.version=$(VERSION)" \
		-o $(BINARY) ./cmd/better-rpt

helper:
	mkdir -p $(dir $(HELPER))
	CGO_ENABLED=0 $(GO) build -trimpath -buildvcs=false \
		-ldflags "-s -w -X main.version=$(VERSION)" \
		-o $(HELPER) ./cmd/better-rpt-helper

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./internal/...

vet:
	$(GO) vet ./...
