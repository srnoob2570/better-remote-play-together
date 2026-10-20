GO ?= GOTOOLCHAIN=go1.27.2 go
VERSION ?= 0.1.2
BINARY ?= dist/better-rpt
HELPER ?= dist/better-rpt-helper
ARGS ?=

.PHONY: dev build helper deb test test-race vet release release-plan

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

deb:
	VERSION="$(VERSION)" ./scripts/build-deb.sh

test:
	$(GO) test ./...

release:
	./scripts/release.sh $(ARGS)

release-plan:
	./scripts/release.sh --dry-run $(ARGS)

test-race:
	$(GO) test -race ./internal/...

vet:
	$(GO) vet ./...

INSTALL ?= install
DESTDIR ?=
BINDIR ?= /usr/bin
HELPERDIR ?= /usr/lib/better-rpt
APPSDIR ?= /usr/share/applications
ICONDIR ?= /usr/share/icons/hicolor
ICON_SIZES ?= 16 24 32 48 64 128 256 512

# Instala los binarios ya compilados con el mismo layout que el .deb. La app
# busca el helper del sistema en /usr/lib/better-rpt; mantén HELPERDIR en
# sintonía con internal/helper/install.go. Usa DESTDIR para un árbol temporal.
.PHONY: all install

all: build helper

install:
	@if [ ! -x "$(BINARY)" ] || [ ! -x "$(HELPER)" ]; then \
		echo "error: faltan $(BINARY) o $(HELPER); ejecuta 'make all' antes de 'sudo make install'" >&2; \
		exit 1; \
	fi
	$(INSTALL) -Dm755 $(BINARY) $(DESTDIR)$(BINDIR)/better-rpt
	$(INSTALL) -Dm755 $(HELPER) $(DESTDIR)$(HELPERDIR)/better-rpt-helper
	$(INSTALL) -Dm644 packaging/better-rpt.desktop \
		$(DESTDIR)$(APPSDIR)/better-rpt.desktop
	for size in $(ICON_SIZES); do \
		$(INSTALL) -Dm644 "packaging/icons/hicolor/$${size}x$${size}/apps/better-rpt.png" \
			"$(DESTDIR)$(ICONDIR)/$${size}x$${size}/apps/better-rpt.png"; \
	done
