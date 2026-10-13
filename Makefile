SHELL := /bin/bash
.DEFAULT_GOAL := help

GO ?= go
STATICCHECK_VERSION := v0.8.2
TOOLS_BIN := $(CURDIR)/.tools/bin
STATICCHECK_CACHE ?= $(CURDIR)/.cache/staticcheck
export STATICCHECK_CACHE
GOFLAGS ?= -mod=readonly
export GOFLAGS
AGORA_LIBS_DIR ?= $(CURDIR)/agora_libs
export AGORA_LIBS_DIR
OUTPUT_BIN_PATH ?= $(CURDIR)/bin
WITH_AGORA := ./scripts/with_agora_env.sh

.PHONY: help fmt fmt-check vet lint install-tools test race build-examples check native-smoke \
        download-agora-libs sync-agora-libs-headers test-tooling test-avcodec clean

help: ## Show available targets.
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "%-25s %s\n", $$1, $$2}'

fmt: ## Format all Go packages.
	./scripts/check_format.sh "$(GO)" --write

fmt-check: ## Fail if any Go source needs formatting.
	./scripts/check_format.sh "$(GO)"

install-tools: ## Install checksum-verified, pinned Staticcheck using Go modules.
	GOBIN="$(TOOLS_BIN)" $(GO) install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)

lint: install-tools ## Lint first-party APIs and examples; vet also checks native bindings.
	PATH="$$(dirname "$$(command -v "$(GO)")"):$$PATH" $(WITH_AGORA) "$(TOOLS_BIN)/staticcheck" . ./messaging/... ./mediaio/... ./token/... ./examples/...

vet: ## Vet every package with the pinned native SDK.
	$(WITH_AGORA) $(GO) vet ./...

test: ## Run offline tests; no Agora credentials or live channel needed.
	$(WITH_AGORA) $(GO) test -count=1 ./...

race: ## Run offline tests with Go's race detector (native code is not instrumented).
	$(WITH_AGORA) $(GO) test -race -count=1 ./...

build-examples: ## Build all examples from the root module.
	mkdir -p "$(OUTPUT_BIN_PATH)"
	@for package in $$($(WITH_AGORA) $(GO) list -f '{{if eq .Name "main"}}{{.ImportPath}}{{end}}' ./examples/...); do \
		$(WITH_AGORA) $(GO) build -o "$(OUTPUT_BIN_PATH)/$${package##*/}" "$$package" || exit $$?; \
	done

native-smoke: ## Check credential-free native lifecycle with strict cgo pointer checks.
	AGORA_NATIVE_SMOKE=1 GOEXPERIMENT=cgocheck2 $(WITH_AGORA) $(GO) test -race -count=1 -run '^TestNative' ./...

check: fmt-check test-tooling lint vet test race build-examples native-smoke ## Run all default offline checks after SDK installation.

download-agora-libs: ## Install the checksum-pinned SDK for this OS/architecture.
	./scripts/download_agora_libs.sh

sync-agora-libs-headers: ## Refresh vendored C headers from the pinned Linux SDK.
	./scripts/sync_agora_libs_headers.sh

test-tooling: ## Check shell syntax and non-destructive installer behavior offline.
	./scripts/test_tooling.sh

test-avcodec: ## Test optional FFmpeg decoders (requires FFmpeg development packages).
	$(WITH_AGORA) $(GO) test -tags avcodec -count=1 ./...

clean: ## Remove generated example binaries; preserve native SDK/cache.
	rm -f "$(OUTPUT_BIN_PATH)/send_pcm" "$(OUTPUT_BIN_PATH)/recv_pcm" "$(OUTPUT_BIN_PATH)/loopback" "$(OUTPUT_BIN_PATH)/media" "$(OUTPUT_BIN_PATH)/messages" "$(OUTPUT_BIN_PATH)/multi_sessions" "$(OUTPUT_BIN_PATH)/ai_audio" "$(OUTPUT_BIN_PATH)/process_audio" "$(OUTPUT_BIN_PATH)/messaging" "$(OUTPUT_BIN_PATH)/send_media"
