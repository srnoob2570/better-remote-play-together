GO ?= GOTOOLCHAIN=go1.27.2 go
WAILS3 ?= $(shell command -v wails3 2>/dev/null || printf '%s/bin/wails3' "$$(go env GOPATH)")
VERSION ?= 0.2.1
BINARY ?= dist/better-rpt
HELPER ?= dist/better-rpt-helper
ARGS ?=

.PHONY: dev build helper deb test test-race vet release release-plan

dev:
	PATH="$(dir $(WAILS3)):$$PATH" $(WAILS3) dev -config ./build/config.yml -port 9245

build:
	PATH="$(dir $(WAILS3)):$$PATH" VERSION=$(VERSION) $(WAILS3) build

helper:
	mkdir -p $(dir $(HELPER))
	CGO_ENABLED=0 $(GO) build -trimpath -buildvcs=false \
		-ldflags "-s -w -X main.version=$(VERSION)" \
		-o $(HELPER) ./cmd/better-rpt-helper

deb: build
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
.PHONY: all install uninstall

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

# Elimina los archivos que crea 'install'. No toca los datos del usuario ni la
# configuración de Steam. Usa los mismos BINDIR/HELPERDIR/ICONDIR que al
# instalar; DESTDIR permite probar sobre un árbol temporal.
uninstall:
	rm -f "$(DESTDIR)$(BINDIR)/better-rpt"
	rm -f "$(DESTDIR)$(HELPERDIR)/better-rpt-helper"
	-rmdir "$(DESTDIR)$(HELPERDIR)" 2>/dev/null
	rm -f "$(DESTDIR)$(APPSDIR)/better-rpt.desktop"
	for size in $(ICON_SIZES); do \
		rm -f "$(DESTDIR)$(ICONDIR)/$${size}x$${size}/apps/better-rpt.png"; \
	done
