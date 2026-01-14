#!/usr/bin/env bash

set -euo pipefail

# Base URL for downloading SDKs.
readonly BASE_URL="https://download.agora.io/sdk/release"

# SDK version file.
readonly SDK_VERSION_FILE="agora_sdk_version"

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

# Download and extract SDK.
check_and_download() {
    local download_url=$1
    local download_file_name=$2
    local dst_dir=$3
    local os=$4
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

    # Determine the suffix of dynamic library.
    local dynamic_library_suffix
    if [[ "${os}" == "darwin" ]]; then
        dynamic_library_suffix="dylib"
    else
        dynamic_library_suffix="so"
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
    mv ${dst_dir}/agora_sdk/*."${dynamic_library_suffix}" ${dst_dir}
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
    
    # Read the SDK version for the given OS.
    SDK_VERSION=$(read_sdk_version "${SDK_VERSION_FILE}" "${os}")

    # Download the SDK.
    check_and_download "${BASE_URL}/${SDK_VERSION}" "agora_sdk_${architecture}_${os}" ${TARGET_LIBS_DIR} "${os}"
    
    echo "✓ All SDKs downloaded to ${TARGET_LIBS_DIR}"
}

# Entrypoint.
main "$@"
