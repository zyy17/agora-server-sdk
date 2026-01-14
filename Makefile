# Set the Agora libraries directory.
AGORA_LIBS_DIR := $(PWD)/agora_libs

# Set the Agora RTC base directory.
AGORA_RTC_BASE_DIR := $(PWD)/agora_rtc

# Set the output binary path.
OUTPUT_BIN_PATH := $(PWD)/bin

# Set the examples directory.
EXAMPLES_DIR := $(PWD)/examples

.PHONY: build-examples
build-examples: download-agora-libs ## Build the examples.
	cd $(EXAMPLES_DIR) && CGO_LDFLAGS="-L$(AGORA_LIBS_DIR)" go build -o $(OUTPUT_BIN_PATH)/send_pcm $(EXAMPLES_DIR)/send_pcm/main.go
	cd $(EXAMPLES_DIR) && CGO_LDFLAGS="-L$(AGORA_LIBS_DIR)" go build -o $(OUTPUT_BIN_PATH)/recv_pcm $(EXAMPLES_DIR)/recv_pcm/main.go

.PHONY: download-agora-libs
download-agora-libs: ## Download the official Agora libraries into agora_libs.
	./scripts/download_agora_libs.sh

.PHONY: add-trailing-newlines
add-trailing-newlines: ## Add trailing newlines to the Agora headers and source files.
	find agora -type f \( -name "*.h" -o -name "*.c" \) -exec sed -i -e '$$a\' {} +
