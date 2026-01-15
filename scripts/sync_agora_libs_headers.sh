#!/usr/bin/env bash

set -euo pipefail

# Base URL for downloading SDKs.
readonly BASE_URL="https://download.agora.io/sdk/release"

# Latest Linux SDK version.
readonly LINUX_SDK_URL="${BASE_URL}/agora_rtc_sdk_x86_64-linux-gnu-v4.4.32.156_27122_SERVER_20251229_1956_996360_20251021_1427-3a.zip"

# Target directory for headers.
readonly TARGET_DIR="agora/headers"

# Headers to sync.
readonly HEADERS_TO_SYNC=(
    "agora_rtm_sdk_c"
    "include"
)

# Temporary directory for download and extraction.
readonly TEMP_DIR=$(mktemp -d)
trap "rm -rf ${TEMP_DIR}" EXIT

main() {
    # Extract filename from URL (everything after the last /)
    local sdk_filename
    sdk_filename="${LINUX_SDK_URL##*/}"
    echo "Using SDK: ${sdk_filename}"
    
    local download_url="${LINUX_SDK_URL}"
    local zip_file="${TEMP_DIR}/${sdk_filename}"
    local extract_dir="${TEMP_DIR}/extracted"
    
    # Download the SDK.
    echo "Downloading ${download_url}..."
    if ! curl -f -L -o "${zip_file}" "${download_url}"; then
        echo "✗ Failed to download ${download_url}" >&2
        exit 1
    fi
    
    # Extract the zip file.
    echo "Extracting ${sdk_filename}..."
    mkdir -p "${extract_dir}"
    unzip -q "${zip_file}" -d "${extract_dir}"
    
    # Find the agora_sdk directory in the extracted files.
    local sdk_dir
    sdk_dir=$(find "${extract_dir}" -type d -name "agora_sdk" | head -n 1)
    if [[ -z "${sdk_dir}" ]]; then
        echo "✗ Could not find agora_sdk directory in extracted files" >&2
        exit 1
    fi
    
    echo "Found SDK directory: ${sdk_dir}"
    
    # Create target directory if it doesn't exist.
    mkdir -p "${TARGET_DIR}"
    
    # Copy directories from HEADERS_TO_SYNC list.
    for dir in "${HEADERS_TO_SYNC[@]}"; do
        if [[ -d "${sdk_dir}/${dir}" ]]; then
            echo "Copying ${dir}/ to ${TARGET_DIR}/..."
            rm -rf "${TARGET_DIR}/${dir}"
            cp -r "${sdk_dir}/${dir}" "${TARGET_DIR}/"
            echo "✓ Copied ${dir}/"
        else
            echo "⚠ Warning: ${dir}/ not found in SDK"
        fi
    done
    
    echo "✓ Successfully synced headers to ${TARGET_DIR}"

    # Remove the temporary directory.
    rm -rf "${TEMP_DIR}"
}

# Entrypoint.
main "$@"
