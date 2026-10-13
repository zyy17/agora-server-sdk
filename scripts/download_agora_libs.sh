#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=lib/sdk_common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/sdk_common.sh"

archive=''
case "${1:-}" in
    '') ;;
    --archive)
        [[ $# -eq 2 ]] || { sdk_die 'Usage: download_agora_libs.sh [--archive PATH]'; exit 2; }
        archive=$2 ;;
    --help|-h)
        printf 'Usage: %s [--archive PATH]\n\nInstalls the pinned, SHA-256-verified native SDK for this host.\nEnvironment: AGORA_LIBS_DIR (default: repository/agora_libs),\n             AGORA_SDK_CACHE_DIR (default: user cache/agora-server-sdk).\n' "$0"
        exit 0 ;;
    *) sdk_die "Unknown argument: $1"; exit 2 ;;
esac
sdk_select_platform "$(sdk_host_platform)"
target=${AGORA_LIBS_DIR:-${SDK_REPO_ROOT}/agora_libs}
# A local archive is always checked, even when the installation is current.
if [[ -n "$archive" ]]; then sdk_verify_checksum "$archive"; fi
if sdk_validate_installation "$target"; then
    sdk_log "Agora SDK ${SDK_VERSION} already installed in $target"
    exit 0
fi
trap sdk_cleanup_transaction EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
sdk_begin_transaction "$target"
sdk_prepare_archive "$archive" "$SDK_WORK_DIR"
sdk_extract_archive "$SDK_ARCHIVE" "${SDK_WORK_DIR}/extracted"
sdk_validate_libraries "${SDK_WORK_DIR}/extracted/agora_sdk"
if [[ "$SDK_PLATFORM" == linux-amd64 ]]; then
    sdk_validate_headers "${SDK_WORK_DIR}/extracted/agora_sdk"
fi
sdk_cache_archive "$SDK_ARCHIVE"
printf '%s\n' "$SDK_URL" > "${SDK_WORK_DIR}/extracted/agora_sdk/sdk_version"
printf '%s\n' "$SDK_SHA256" > "${SDK_WORK_DIR}/extracted/agora_sdk/sdk_sha256"
printf '%s\n' "$SDK_PLATFORM" > "${SDK_WORK_DIR}/extracted/agora_sdk/sdk_platform"
sdk_commit_directory "${SDK_WORK_DIR}/extracted/agora_sdk"
sdk_log "Installed verified Agora SDK ${SDK_VERSION} in $SDK_TARGET"
