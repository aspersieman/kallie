BINARY   := kallie
PKG      := ./cmd/kallie
BUILD_DIR := bin
DIST_DIR := dist
PREFIX   ?= /usr/local
DESTDIR  ?=
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -s -w
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.DEFAULT_GOAL := help
.PHONY: help build run test vet fmt lint tidy check clean release install uninstall

help: ## Show this help
	@grep -E '^[a-z_-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-10s %s\n", $$1, $$2}'

build: ## Build a development binary into bin/
	go build -o $(BUILD_DIR)/$(BINARY) $(PKG)

run: ## Run kallie (pass args with ARGS="--json")
	go run $(PKG) $(ARGS)

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

install: ## Install to $(PREFIX)/bin (override PREFIX or DESTDIR)
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) $(PKG)
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 0755 $(BUILD_DIR)/$(BINARY) $(DESTDIR)$(PREFIX)/bin/$(BINARY)

uninstall: ## Remove the installed binary
	rm -f $(DESTDIR)$(PREFIX)/bin/$(BINARY)

clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR) $(DIST_DIR)
