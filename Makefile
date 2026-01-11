# Set the Agora SDK path.
AGORA_SDK_PATH := $(PWD)/third_party/Agora-Golang-Server-SDK

# Set the Agora RTC base directory.
AGORA_RTC_BASE_DIR := $(PWD)/agora_rtc

# Set the output binary path.
OUTPUT_BIN_PATH := $(PWD)/bin

# Set the examples directory.
EXAMPLES_DIR := $(PWD)/examples

# Get the kind of build machine.
OS := $(shell uname -s)

# Set the SDK path based on the build machine.
ifeq ($(OS),Darwin)
    AGORA_LIB_LINK_PATH := DYLD_LIBRARY_PATH=$(AGORA_SDK_PATH)/agora_sdk_mac
else ifeq ($(OS),Linux)
    AGORA_LIB_LINK_PATH := LD_LIBRARY_PATH=$(AGORA_SDK_PATH)/agora_sdk
else
    $(error Unsupported OS: $(OS))
endif

##@ Development

.PHONY: build-examples
build-examples: install-agora-sdk ## Build the examples
	cd $(EXAMPLES_DIR) && $(AGORA_LIB_LINK_PATH) go build -o $(OUTPUT_BIN_PATH)/send_pcm ./send_pcm/main.go
	cd $(EXAMPLES_DIR) && $(AGORA_LIB_LINK_PATH) go build -o $(OUTPUT_BIN_PATH)/recv_pcm ./recv_pcm/main.go

.PHONY: install-agora-sdk
install-agora-sdk: init-submodules ## Build and install the Agora SDK
	make -C $(AGORA_SDK_PATH) install

.PHONY: init-submodules
init-submodules: ## Initialize submodules and checkout to the target revision
	git submodule update --init --recursive

.PHONY: clean
clean: ## Clean the project
	rm -rf $(OUTPUT_BIN_PATH)
	rm -rf $(AGORA_RTC_BASE_DIR)
	make -C $(AGORA_SDK_PATH) clean

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
