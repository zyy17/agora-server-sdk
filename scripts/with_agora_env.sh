#!/usr/bin/env bash
# Run a command with the linker and runtime paths needed by the native SDK.
set -euo pipefail
# shellcheck source=lib/sdk_common.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/sdk_common.sh"
[[ $# -gt 0 ]] || { sdk_die 'Usage: with_agora_env.sh COMMAND [ARG ...]'; exit 2; }
sdk_select_platform "$(sdk_host_platform)"
libs=${AGORA_LIBS_DIR:-${SDK_REPO_ROOT}/agora_libs}
if ! sdk_validate_installation "$libs"; then
    sdk_die 'Native SDK is missing, incomplete, or outdated. Run make download-agora-libs first.'
    exit 1
fi
libs=$(cd "$libs" && pwd)
export AGORA_LIBS_DIR="$libs"
export CGO_ENABLED=1
# Quoted paths are interpreted by Go's cgo flag parser, not by eval.
case "$SDK_PLATFORM" in
    linux-amd64)
        export CGO_LDFLAGS="\"-L${libs}\" \"-Wl,-rpath-link,${libs}\" ${CGO_LDFLAGS:-}"
        export LD_LIBRARY_PATH="${libs}${LD_LIBRARY_PATH:+:${LD_LIBRARY_PATH}}" ;;
    darwin-*)
        export CGO_LDFLAGS="\"-L${libs}\" ${CGO_LDFLAGS:-}"
        export DYLD_LIBRARY_PATH="${libs}${DYLD_LIBRARY_PATH:+:${DYLD_LIBRARY_PATH}}" ;;
esac
exec "$@"
