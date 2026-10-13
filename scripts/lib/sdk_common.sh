#!/usr/bin/env bash
# Shared Bash 3.2-compatible helpers. Callers enable strict mode and own cleanup.
SDK_REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
# Never let inherited environment values become cleanup paths.
SDK_WORK_DIR=''
SDK_LOCK=''
SDK_TARGET=''
# shellcheck source=../sdk_versions.sh
source "${SDK_REPO_ROOT}/scripts/sdk_versions.sh"

sdk_die() { printf 'error: %s\n' "$*" >&2; return 1; }
sdk_log() { printf '%s\n' "$*" >&2; }

sdk_host_platform() {
    local os arch
    os=$(uname -s)
    arch=$(uname -m)
    case "${os}/${arch}" in
        Linux/x86_64) printf 'linux-amd64\n' ;;
        Darwin/x86_64) printf 'darwin-amd64\n' ;;
        Darwin/arm64) printf 'darwin-arm64\n' ;;
        Linux/aarch64|Linux/arm64)
            sdk_die 'Linux arm64 is unsupported: upstream still selects a 4.4 SDK for this architecture, incompatible with these 4.6 bindings.' ;;
        *) sdk_die "Unsupported platform: ${os}/${arch}" ;;
    esac
}

sdk_select_platform() {
    SDK_PLATFORM=$1
    case "$SDK_PLATFORM" in
        linux-amd64)
            SDK_URL=$AGORA_LINUX_URL
            SDK_SHA256=$AGORA_LINUX_SHA256
            SDK_VERSION=$AGORA_LINUX_VERSION
            SDK_REQUIRED_FILES=(libagora_rtc_sdk.so libagora-fdkaac.so libagora-ffmpeg.so
                libagora-soundtouch.so libagora_uap_aed.so libaosl.so
                libagora_rtm_sdk.so libagora_rtm_sdk_c.so) ;;
        darwin-amd64|darwin-arm64)
            SDK_URL=$AGORA_MACOS_URL
            SDK_SHA256=$AGORA_MACOS_SHA256
            SDK_VERSION=$AGORA_MACOS_VERSION
            SDK_REQUIRED_FILES=(libAgoraRtcKit.dylib libAgorafdkaac.dylib
                libAgoraffmpeg.dylib libAgoraAiNoiseSuppressionExtension.dylib
                libAgoraVideoAv1EncoderExtension.dylib libuap_aed.dylib
                libaosl.dylib libAgoraRtmKit.dylib libagora_rtm_sdk_c.dylib) ;;
        *) sdk_die "Unsupported SDK platform: $SDK_PLATFORM"; return 1 ;;
    esac
}

sdk_sha256() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$1" | awk '{print $1}'
    else
        sdk_die 'Install sha256sum or shasum to verify SDK downloads.'
    fi
}

sdk_verify_checksum() {
    local actual
    [[ -f "$1" && -s "$1" ]] || { sdk_die "Archive is missing or empty: $1"; return 1; }
    actual=$(sdk_sha256 "$1") || return 1
    [[ "$actual" == "$SDK_SHA256" ]] || {
        sdk_die "SHA-256 mismatch for $1 (expected $SDK_SHA256; got $actual)."
        return 1
    }
}

sdk_validate_headers() {
    local directory=$1 relative
    for relative in include/c/api2/agora_service.h include/c/api2/agora_rtc_conn.h \
        include/c/api2/vad.h include/c/base/agora_base.h \
        agora_rtm_sdk_c/include/C_IAgoraService.h \
        agora_rtm_sdk_c/agora_rtm_sdk/high_level_api/include/AgoraRtmBase.h; do
        [[ -s "${directory}/${relative}" ]] || { sdk_die "Required SDK header missing: ${directory}/${relative}"; return 1; }
    done
}

sdk_validate_libraries() {
    local directory=$1 relative
    for relative in "${SDK_REQUIRED_FILES[@]}"; do
        [[ -s "${directory}/${relative}" ]] || { sdk_die "Required SDK library missing: ${directory}/${relative}"; return 1; }
    done
}

sdk_validate_installation() {
    local directory=$1
    [[ -f "${directory}/sdk_version" && -f "${directory}/sdk_sha256" && -f "${directory}/sdk_platform" ]] || return 1
    [[ "$(cat "${directory}/sdk_version")" == "$SDK_URL" && \
       "$(cat "${directory}/sdk_sha256")" == "$SDK_SHA256" && \
       "$(cat "${directory}/sdk_platform")" == "$SDK_PLATFORM" ]] || return 1
    sdk_validate_libraries "$directory"
}

# Fetch to the caller's temporary directory. A cache entry is accepted only after
# checksum verification. New entries are cached only after extraction validation.
sdk_prepare_archive() {
    local supplied_archive=$1 temporary_directory=$2 cache_file
    SDK_CACHE_DIR=${AGORA_SDK_CACHE_DIR:-${XDG_CACHE_HOME:-${HOME:?HOME must be set}}/agora-server-sdk}
    cache_file="${SDK_CACHE_DIR}/${SDK_SHA256}.zip"
    if [[ -n "$supplied_archive" ]]; then
        SDK_ARCHIVE=$supplied_archive
        sdk_verify_checksum "$SDK_ARCHIVE"
    elif [[ -f "$cache_file" ]] && sdk_verify_checksum "$cache_file"; then
        SDK_ARCHIVE=$cache_file
        sdk_log "Using verified SDK cache: $cache_file"
    else
        command -v curl >/dev/null 2>&1 || { sdk_die 'curl is required to download the native SDK.'; return 1; }
        SDK_ARCHIVE="${temporary_directory}/sdk.zip"
        sdk_log "Downloading Agora SDK ${SDK_VERSION} (${SDK_PLATFORM})"
        curl --fail --show-error --location --retry 3 --connect-timeout 30 \
            --proto '=https' --proto-redir '=https' --tlsv1.2 \
            --output "$SDK_ARCHIVE" "$SDK_URL" || return 1
        sdk_verify_checksum "$SDK_ARCHIVE"
    fi
}

sdk_extract_archive() {
    local archive=$1 destination=$2 entry listing attributes
    command -v unzip >/dev/null 2>&1 || { sdk_die 'unzip is required to extract the native SDK.'; return 1; }
    attributes=$(unzip -Z -l "$archive") || return 1
    if printf '%s\n' "$attributes" | grep -Eq '^[lbcps]'; then
        sdk_die 'SDK archive contains unsupported symlinks or special files.'
        return 1
    fi
    listing=$(unzip -Z1 "$archive") || return 1
    while IFS= read -r entry; do
        case "$entry" in agora_sdk/*) ;; *) sdk_die "Unexpected ZIP member: $entry"; return 1 ;; esac
        case "/${entry}/" in
            *'/../'*|*'/./'*|*'\'*) sdk_die "Unsafe ZIP member: $entry"; return 1 ;;
        esac
    done <<< "$listing"
    mkdir -p "$destination"
    unzip -q "$archive" -d "$destination"
}

sdk_cache_archive() {
    local archive=$1 temporary_cache cache_file
    cache_file="${SDK_CACHE_DIR}/${SDK_SHA256}.zip"
    [[ "$archive" != "$cache_file" ]] || return 0
    if ! mkdir -p "$SDK_CACHE_DIR" 2>/dev/null; then
        sdk_log "SDK verified; skipping unwritable cache: $SDK_CACHE_DIR"
        return 0
    fi
    if ! temporary_cache=$(mktemp "${SDK_CACHE_DIR}/.sdk.XXXXXX"); then
        sdk_log "SDK verified; skipping unwritable cache: $SDK_CACHE_DIR"
        return 0
    fi
    if cp "$archive" "$temporary_cache" && mv -f "$temporary_cache" "$cache_file"; then
        return 0
    fi
    rm -f "$temporary_cache"
    sdk_log "SDK verified; could not update optional cache: $cache_file"
}

# The previous directory is kept until the replacement is ready. The caller's
# EXIT trap must invoke sdk_cleanup_transaction to restore it on interruption.
sdk_begin_transaction() {
    local requested_target=$1 parent basename
    [[ -n "$requested_target" ]] || { sdk_die 'Empty installation path.'; return 1; }
    [[ ! -L "$requested_target" ]] || { sdk_die 'Refusing to replace a symlinked installation directory.'; return 1; }
    parent=$(dirname "$requested_target")
    basename=$(basename "$requested_target")
    [[ "$basename" != '.' && "$basename" != '..' && "$basename" != '/' ]] || { sdk_die 'Unsafe installation path.'; return 1; }
    mkdir -p "$parent"
    parent=$(cd "$parent" && pwd)
    SDK_TARGET="${parent}/${basename}"
    [[ "$SDK_TARGET" != "$SDK_REPO_ROOT" && "$SDK_TARGET" != / ]] || { sdk_die 'Refusing to replace the repository or filesystem root.'; return 1; }
    SDK_LOCK="${SDK_TARGET}.install-lock"
    if ! mkdir "$SDK_LOCK"; then
        SDK_LOCK=''
        sdk_die "Another install may be running for ${SDK_TARGET}; inspect its .install-lock directory."
        return 1
    fi
    SDK_WORK_DIR=$(mktemp -d "${parent}/.agora-install.XXXXXX")
}

sdk_commit_directory() {
    local replacement=$1
    if [[ -e "$SDK_TARGET" ]]; then
        mv "$SDK_TARGET" "${SDK_WORK_DIR}/previous" || return 1
    fi
    mv "$replacement" "$SDK_TARGET"
}

sdk_cleanup_transaction() {
    if [[ -n "${SDK_WORK_DIR:-}" && -d "$SDK_WORK_DIR" ]]; then
        if [[ -e "${SDK_WORK_DIR}/previous" && ! -e "$SDK_TARGET" ]]; then
            if ! mv "${SDK_WORK_DIR}/previous" "$SDK_TARGET"; then
                sdk_log "Previous installation retained at ${SDK_WORK_DIR}/previous; automatic restore failed."
                return 1
            fi
        fi
        rm -rf "$SDK_WORK_DIR"
    fi
    if [[ -n "${SDK_LOCK:-}" && -d "$SDK_LOCK" ]]; then
        rmdir "$SDK_LOCK"
    fi
}
