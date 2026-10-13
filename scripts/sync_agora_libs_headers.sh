#!/usr/bin/env bash
set -euo pipefail
# The macOS artifact contains only dylibs. The checked-in, platform-independent
# C headers always come from the pinned Linux SDK.
# shellcheck source=lib/sdk_common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/sdk_common.sh"
archive=''
case "${1:-}" in
    '') ;;
    --archive)
        [[ $# -eq 2 ]] || { sdk_die 'Usage: sync_agora_libs_headers.sh [--archive PATH]'; exit 2; }
        archive=$2 ;;
    --help|-h)
        printf 'Usage: %s [--archive PATH]\n\nRefreshes agora/headers from the pinned Linux SDK with checksum validation.\nEnvironment: AGORA_SDK_CACHE_DIR.\n' "$0"
        exit 0 ;;
    *) sdk_die "Unknown argument: $1"; exit 2 ;;
esac
sdk_select_platform linux-amd64
trap sdk_cleanup_transaction EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
sdk_begin_transaction "${SDK_REPO_ROOT}/agora/headers"
sdk_prepare_archive "$archive" "$SDK_WORK_DIR"
sdk_extract_archive "$SDK_ARCHIVE" "${SDK_WORK_DIR}/extracted"
sdk_validate_headers "${SDK_WORK_DIR}/extracted/agora_sdk"
mkdir "${SDK_WORK_DIR}/headers"
cp -R "${SDK_WORK_DIR}/extracted/agora_sdk/include" "${SDK_WORK_DIR}/headers/"
cp -R "${SDK_WORK_DIR}/extracted/agora_sdk/agora_rtm_sdk_c" "${SDK_WORK_DIR}/headers/"
sdk_cache_archive "$SDK_ARCHIVE"
sdk_commit_directory "${SDK_WORK_DIR}/headers"
sdk_log "Updated agora/headers from verified Agora SDK ${SDK_VERSION}"
