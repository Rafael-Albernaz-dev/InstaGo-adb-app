SHELL := /bin/bash
export GOTOOLCHAIN := local
export GOFLAGS := -mod=readonly

CORE_GO_PKGS := ./internal/domain/... ./internal/adapters/sqlite/... ./internal/adapters/assets/... ./internal/adapters/codex/... ./internal/usecase/...

.PHONY: hygiene-check gofmt-check front-build front-test check-core check-native check-gtk3 native-deps gtk3-deps build build-gtk3 install-desktop run check hooks

hygiene-check:
	bash scripts/ci/check-repo-hygiene.sh

gofmt-check:
	@test -z "$$(gofmt -l assets.go cmd internal)" || (gofmt -l assets.go cmd internal; exit 1)

front-build:
	npm --prefix frontend run build

front-test:
	npm --prefix frontend run test

check-core: hygiene-check gofmt-check front-build front-test
	go vet $(CORE_GO_PKGS)
	go test -race $(CORE_GO_PKGS)

native-deps:
	@pkg-config --exists gtk4 webkitgtk-6.0 || ( \
		echo "Missing native Wails 3 headers: gtk4 / webkitgtk-6.0." >&2; \
		echo "Install with: sudo apt-get update && sudo apt-get install --no-install-recommends libgtk-4-dev libwebkitgtk-6.0-dev" >&2; \
		exit 1; \
	)

check-native: native-deps front-build
	go vet ./...
	go test -race ./...

gtk3-deps:
	@pkg-config --exists gtk+-3.0 webkit2gtk-4.1 || ( \
		echo "Missing GTK3/WebKit2GTK fallback headers: gtk+-3.0 / webkit2gtk-4.1." >&2; \
		exit 1; \
	)

check-gtk3: gtk3-deps front-build
	go vet -tags gtk3 ./...
	go test -race -tags gtk3 ./...

build-gtk3: gtk3-deps front-build
	mkdir -p build/bin
	go build -tags gtk3 -trimpath -o build/bin/instago-gtk3 ./cmd/instago

install-desktop: build-gtk3
	bash scripts/linux/install-desktop.sh "$(CURDIR)/build/bin/instago-gtk3"

build: native-deps front-build
	mkdir -p build/bin
	go build -trimpath -o build/bin/instago ./cmd/instago

run: build
	./build/bin/instago

check: check-core

hooks:
	git config core.hooksPath .githooks
