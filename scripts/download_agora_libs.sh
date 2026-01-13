#!/usr/bin/env bash

set -euo pipefail

# Base URL for downloading SDKs.
readonly BASE_URL="https://download.agora.io/sdk/release"

# Latest SDK versions.
readonly LINUX_SDK_URL="${BASE_URL}/agora_rtc_sdk_x86_64-linux-gnu-v4.4.32.156_27122_SERVER_20251229_1956_996360_20251021_1427-3a.zip"
readonly DARWIN_SDK_URL="${BASE_URL}/agora_sdk_mac_v4.4.32.156_26548_FULL_20251230_1429_996508_20251021_1427-3a.zip"

# Target libraries directory.
readonly TARGET_LIBS_DIR="agora_libs"

# Detect OS.
detect_os() {
    local uname_s
    uname_s=$(uname -s)
    
    case "${uname_s}" in
        Linux)
            echo "linux"
            ;;
        Darwin)
            echo "darwin"
            ;;
        *)
            echo "Unsupported OS: ${uname_s}" >&2
            exit 1
            ;;
    esac
}

# Detect architecture.
detect_architecture() {
    local uname_m
    uname_m=$(uname -m)
    
    case "${uname_m}" in
        x86_64)
            echo "amd64"
            ;;
        aarch64)
            echo "arm64"
            ;;
        arm64)
            echo "arm64"
            ;;
        *)
            echo "Unsupported architecture: ${uname_m}" >&2
            exit 1
            ;;
    esac
}

# Download and extract SDK.
check_and_download() {
    local download_url=$1
    local download_file_name=$2
    local dst_dir=$3
    local version_file="${dst_dir}/sdk_version"
    local zip_file="${download_file_name}.zip"
    
    # Check if already downloaded with correct version.
    if [[ -f "${version_file}" ]]; then
        local sdk_version
        sdk_version=$(cat "${version_file}")
        if [[ "${sdk_version}" == "${download_url}" ]]; then
            echo "✓ ${dst_dir} already downloaded"
            return 0
        fi
    fi
    
    # Clean up old files.
    [[ -f "${zip_file}" ]] && rm -f "${zip_file}"
    [[ -d "${dst_dir}" ]] && rm -rf "${dst_dir}"
    
    # Download the zip file.
    echo "Downloading ${download_url} and saving as ${zip_file}..."
    if ! curl -f -L -o "${zip_file}" "${download_url}"; then
        echo "✗ Failed to download ${download_url}" >&2
        return 1
    fi
    
    # Extract the zip file. 
    echo "Extracting ${zip_file}..."
    unzip -q "${zip_file}" -d "${dst_dir}"
    rm -f "${zip_file}"

    # Move the agora_sdk directory to the target libraries directory.
    mv ${dst_dir}/agora_sdk/* ${dst_dir}
    rm -rf "${dst_dir}/agora_sdk"
    
    # Write download URL to version file.
    echo "${download_url}" > "${version_file}"
    echo "✓ ${dst_dir} downloaded successfully"
}

main() {    
    os=$(detect_os)
    echo "Detected OS: ${os}"

    architecture=$(detect_architecture)
    echo "Detected architecture: ${architecture}"
    
    # Create target libraries directory if it doesn't exist.
    mkdir -p "${TARGET_LIBS_DIR}"
    
    # Download SDK based on OS.
    if [[ "${os}" == "darwin" ]]; then
        SDK_DOWNLOAD_URL="${DARWIN_SDK_URL}"
    else
        SDK_DOWNLOAD_URL="${LINUX_SDK_URL}"
    fi
    
    # Always download Linux SDK (for cross-compilation or server use).
    check_and_download "${SDK_DOWNLOAD_URL}" "agora_sdk_${architecture}_${os}" ${TARGET_LIBS_DIR}
    
    echo "✓ All SDKs downloaded to ${TARGET_LIBS_DIR}"
}

# Entrypoint.
main "$@"
