BINARY   := kallie
PKG      := ./cmd/kallie
GUI_BINARY := kallie-gui
GUI_PKG  := ./cmd/kallie-gui
BUILD_DIR := bin
DIST_DIR := dist
PREFIX   ?= /usr/local
DESTDIR  ?=
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.DEFAULT_GOAL := help
.PHONY: help build build-gui all run run-gui test vet fmt lint tidy check clean release install install-cli install-gui uninstall

help: ## Show this help
	@grep -E '^[a-z_-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-10s %s\n", $$1, $$2}'

build: ## Build a development binary into bin/
	go build -o $(BUILD_DIR)/$(BINARY) $(PKG)

build-gui: ## Build the Fyne GUI into bin/ (needs cgo + GL/X11/Wayland headers on Linux)
	CGO_ENABLED=1 go build -o $(BUILD_DIR)/$(GUI_BINARY) $(GUI_PKG)

all: build build-gui ## Build the CLI and the GUI

run: ## Run kallie (pass args with ARGS="--json")
	go run $(PKG) $(ARGS)

run-gui: ## Run the GUI (pass args with ARGS="--days 7")
	go run $(GUI_PKG) $(ARGS)

test: ## Run tests with the race detector
	go test -race ./...

vet: ## Run go vet
	go vet ./...

fmt: ## Format code
	gofmt -w .

lint: ## Check formatting and vet
	@test -z "$$(gofmt -l .)" || (gofmt -l .; echo "gofmt needed"; exit 1)
	go vet ./...

tidy: ## Tidy go.mod
	go mod tidy

check: lint test ## Run lint and tests

release: ## Cross-compile stripped production binaries into dist/
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; ext=""; \
		[ $$os = windows ] && ext=.exe; \
		out=$(DIST_DIR)/$(BINARY)-$(VERSION)-$$os-$$arch$$ext; \
		echo "building $$out"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags "$(LDFLAGS)" -o $$out $(PKG) || exit 1; \
	done

install: install-cli install-gui ## Install the CLI and GUI to $(PREFIX)/bin (override PREFIX or DESTDIR)

install-cli: ## Install only the (static) CLI
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) $(PKG)
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 0755 $(BUILD_DIR)/$(BINARY) $(DESTDIR)$(PREFIX)/bin/$(BINARY)

install-gui: ## Install only the GUI (needs cgo)
	CGO_ENABLED=1 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(GUI_BINARY) $(GUI_PKG)
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 0755 $(BUILD_DIR)/$(GUI_BINARY) $(DESTDIR)$(PREFIX)/bin/$(GUI_BINARY)

uninstall: ## Remove the installed binaries
	rm -f $(DESTDIR)$(PREFIX)/bin/$(BINARY) $(DESTDIR)$(PREFIX)/bin/$(GUI_BINARY)

clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR) $(DIST_DIR)
