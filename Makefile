SHELL := /bin/bash
.DEFAULT_GOAL := help
.NOTPARALLEL:

GO ?= go
NPM ?= npm
APP_ARGS ?=
MACOSX_DEPLOYMENT_TARGET ?= 13.0
CGO_CFLAGS ?= -mmacosx-version-min=$(MACOSX_DEPLOYMENT_TARGET)
CGO_LDFLAGS ?= -mmacosx-version-min=$(MACOSX_DEPLOYMENT_TARGET)

export MACOSX_DEPLOYMENT_TARGET CGO_CFLAGS CGO_LDFLAGS

BUILD_TOOL := $(GO) run ./cmd/build
SPECKIT_INSTALL := scripts/install-speckit.sh
SPECKIT_UPDATE_CHECK := scripts/check-speckit-updates.py

SPECKIT_VERSION := 0.16.5
SPECKIT_COMPANION_VERSION := 0.20.1
SPECKIT_BROWNFIELD_VERSION := 1.0.0
SPECKIT_BUGFIX_VERSION := 1.1.0
SPECKIT_FEATURE_NUMBERING_VERSION := 1.0.0

SPECKIT_VERSION_ENV = \
	SPECKIT_VERSION="$(SPECKIT_VERSION)" \
	SPECKIT_COMPANION_VERSION="$(SPECKIT_COMPANION_VERSION)" \
	SPECKIT_BROWNFIELD_VERSION="$(SPECKIT_BROWNFIELD_VERSION)" \
	SPECKIT_BUGFIX_VERSION="$(SPECKIT_BUGFIX_VERSION)" \
	SPECKIT_FEATURE_NUMBERING_VERSION="$(SPECKIT_FEATURE_NUMBERING_VERSION)"

.PHONY: help dev run prepare build package \
	deps deps-client deps-frontend deps-browser \
	speckit-install speckit-update-check \
	fmt fmt-check vet test test-race check \
	proto-generate proto-check proto-breaking bindings-check browser-test \
	release-preflight release

help: ## Show available commands.
	@awk 'BEGIN {FS = ":.*## "; printf "Usage: make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_-]+:.*## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

dev: ## Build and run the complete application in development mode.
	$(BUILD_TOOL) dev $(APP_ARGS)

run: ## Build and run the application, forwarding APP_ARGS.
	$(BUILD_TOOL) run $(APP_ARGS)

prepare: ## Verify and build protobuf, player, bindings, and master assets.
	$(BUILD_TOOL) prepare

build: ## Build the macOS arm64 application binary.
	$(BUILD_TOOL) build

package: ## Build the ad-hoc signed macOS application bundle.
	$(BUILD_TOOL) package

deps: deps-client deps-frontend deps-browser ## Install all locked Node.js dependencies.

deps-client: ## Install locked player dependencies.
	$(NPM) ci --prefix client

deps-frontend: ## Install locked master frontend dependencies.
	$(NPM) ci --prefix frontend

deps-browser: ## Install locked browser-test dependencies.
	$(NPM) ci --prefix tests/browser

speckit-install: ## Install pinned Spec Kit and all project extensions.
	$(SPECKIT_VERSION_ENV) $(SPECKIT_INSTALL)

speckit-update-check: ## Check for Spec Kit and extension updates without installing them.
	$(SPECKIT_VERSION_ENV) $(SPECKIT_UPDATE_CHECK)

fmt: ## Format all Go sources.
	gofmt -w .

fmt-check: ## Fail when a Go source is not formatted.
	@unformatted="$$(gofmt -l .)"; \
	if [[ -n "$$unformatted" ]]; then \
		printf 'Go files need formatting:\n%s\n' "$$unformatted" >&2; \
		exit 1; \
	fi

vet: ## Run Go static analysis.
	$(GO) vet ./...

test: ## Run the Go test suite.
	$(GO) test ./...

test-race: ## Run the Go test suite with the race detector.
	$(GO) test -race ./...

proto-generate: deps-client ## Regenerate protobuf code and update the reviewed revision.
	scripts/proto-generate.sh --sync-revision

proto-check: deps-client ## Verify protobuf formatting, generation, and generated clients.
	scripts/proto-check.sh

proto-breaking: ## Verify protobuf compatibility and negative fixtures.
	scripts/proto-breaking.sh --all-fixtures

bindings-check: ## Verify deterministic Wails bindings and their public surface.
	scripts/wails-bindings-check.sh

browser-test: deps-client deps-browser ## Install Chromium and run Playwright journeys.
	$(NPM) exec --prefix tests/browser -- playwright install chromium
	$(NPM) test --prefix tests/browser

check: fmt-check vet test-race proto-check proto-breaking bindings-check ## Run the main local quality gates.

release-preflight: ## Validate Developer ID and notarization prerequisites.
	scripts/build-macos.sh --preflight

release: ## Build, sign, notarize, and verify the release DMG.
	scripts/build-macos.sh
