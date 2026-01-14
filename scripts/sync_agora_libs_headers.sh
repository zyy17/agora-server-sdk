#!/usr/bin/env bash

set -euo pipefail

# Base URL for downloading SDKs.
readonly BASE_URL="https://download.agora.io/sdk/release"

# SDK version file.
readonly SDK_VERSION_FILE="agora_sdk_version"

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

# Read the SDK version from the file for the given OS.
read_sdk_version() {
    local sdk_version_file=$1
    local os=$2
    if [[ ! -f "${sdk_version_file}" ]]; then
        echo "Unknown sdk version" >&2
        exit 1
    fi

    # Read the file and extract the filename for the given OS.
    # Format: os: filename
    local version_line
    version_line=$(grep "^${os}:" "${sdk_version_file}" || true)
    if [[ -z "${version_line}" ]]; then
        echo "Unknown sdk version for OS: ${os}" >&2
        exit 1
    fi
    
    # Extract filename (everything after "os: ")
    echo "${version_line#*: }"
}

main() {
    echo "Reading Linux SDK version from ${SDK_VERSION_FILE}..."
    local sdk_filename
    sdk_filename=$(read_sdk_version "${SDK_VERSION_FILE}" "linux")
    echo "Found SDK: ${sdk_filename}"
    
    local download_url="${BASE_URL}/${sdk_filename}"
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
