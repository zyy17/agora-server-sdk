# Set the Agora libraries directory.
AGORA_LIBS_DIR := $(PWD)/agora_libs

# Set the Agora RTC base directory.
AGORA_RTC_BASE_DIR := $(PWD)/agora_rtc

# Set the output binary path.
OUTPUT_BIN_PATH := $(PWD)/bin

# Set the examples directory.
EXAMPLES_DIR := $(PWD)/examples

# Get the kind of build machine.
OS := $(shell uname -s)

# Set the CGO_LDFLAGS based on the build machine.
ifeq ($(OS),Darwin)
    CGO_LDFLAGS := -L$(AGORA_LIBS_DIR)
else ifeq ($(OS),Linux)
    CGO_LDFLAGS := -L$(AGORA_LIBS_DIR) -Wl,-rpath-link=$(AGORA_LIBS_DIR)
else
    $(error Unsupported OS: $(OS))
endif

.PHONY: build-examples
build-examples: download-agora-libs ## Build the examples.
	cd $(EXAMPLES_DIR) && CGO_LDFLAGS="$(CGO_LDFLAGS)" go build -o $(OUTPUT_BIN_PATH)/send_pcm $(EXAMPLES_DIR)/send_pcm/main.go
	cd $(EXAMPLES_DIR) && CGO_LDFLAGS="$(CGO_LDFLAGS)" go build -o $(OUTPUT_BIN_PATH)/recv_pcm $(EXAMPLES_DIR)/recv_pcm/main.go

.PHONY: download-agora-libs
download-agora-libs: ## Download the official Agora libraries into agora_libs.
	./scripts/download_agora_libs.sh

.PHONY: sync-agora-libs-headers
sync-agora-libs-headers: ## Sync the official Agora headers into agora/headers.
	./scripts/sync_agora_libs_headers.sh
	$(MAKE) add-trailing-newlines

.PHONY: add-trailing-newlines
add-trailing-newlines: ## Add trailing newlines to the Agora headers and source files.
	find agora -type f \( -name "*.h" -o -name "*.c" \) -exec sed -i -e '$$a\' {} +

.PHONY: clean
clean: ## Clean the build files.
	rm -rf $(AGORA_LIBS_DIR)
	rm -rf $(OUTPUT_BIN_PATH)
