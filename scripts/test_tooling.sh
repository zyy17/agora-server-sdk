#!/usr/bin/env bash
# Offline installer regressions. No native SDK download or RTC credentials.
set -euo pipefail
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
# shellcheck source=lib/sdk_common.sh
source "${root}/scripts/lib/sdk_common.sh"
for script in "${root}"/scripts/*.sh "${root}"/scripts/lib/*.sh; do bash -n "$script"; done
sandbox=$(mktemp -d)
trap 'rm -rf "$sandbox"' EXIT
fail() { printf 'FAIL: %s\n' "$*" >&2; exit 1; }
assert_equal() { [[ "$1" == "$2" ]] || fail "${3:-values differ}: $1 != $2"; }

sdk_select_platform linux-amd64
assert_equal "$SDK_SHA256" "$AGORA_LINUX_SHA256" 'Linux pin'
sdk_select_platform darwin-amd64
assert_equal "$SDK_SHA256" "$AGORA_MACOS_SHA256" 'macOS x64 pin'
sdk_select_platform darwin-arm64
assert_equal "$SDK_SHA256" "$AGORA_MACOS_SHA256" 'macOS arm64 pin'
if sdk_select_platform linux-arm64 2>/dev/null; then fail 'Unsupported Linux arm64 was accepted'; fi
sdk_select_platform linux-amd64

printf 'abc' > "${sandbox}/checksum"
assert_equal "$(sdk_sha256 "${sandbox}/checksum")" \
    'ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad' 'SHA-256 implementation'
if sdk_verify_checksum "${sandbox}/checksum" 2>/dev/null; then fail 'Bad checksum was accepted'; fi

# Old installation survives bad input and network errors; a stub makes any
# accidental network attempt fail immediately and records that it happened.
mkdir "${sandbox}/bin" "${sandbox}/installed"
printf 'keep me' > "${sandbox}/installed/original"
cat > "${sandbox}/bin/curl" <<'STUB'
#!/usr/bin/env bash
printf 'called\n' >> "${TEST_CURL_LOG}"
exit 22
STUB
cat > "${sandbox}/bin/uname" <<'STUB'
#!/usr/bin/env bash
case "$1" in -s) printf 'Linux\n' ;; -m) printf '%s\n' "${TEST_ARCH:-x86_64}" ;; esac
STUB
chmod +x "${sandbox}/bin/"*
export TEST_CURL_LOG="${sandbox}/curl.log"
export AGORA_LIBS_DIR="${sandbox}/installed"
export AGORA_SDK_CACHE_DIR="${sandbox}/cache"
export PATH="${sandbox}/bin:${PATH}"
if "${root}/scripts/download_agora_libs.sh" --archive "${sandbox}/checksum" >"${sandbox}/bad-archive.log" 2>&1; then
    fail 'Installer accepted bad archive'
fi
assert_equal "$(cat "${sandbox}/installed/original")" 'keep me' 'Bad archive changed old install'
[[ ! -e "$TEST_CURL_LOG" ]] || fail 'Local archive unexpectedly triggered network access'
if "${root}/scripts/download_agora_libs.sh" >"${sandbox}/failed-download.log" 2>&1; then
    fail 'Installer accepted failed download'
fi
assert_equal "$(cat "${sandbox}/installed/original")" 'keep me' 'Failed download changed old install'
[[ -f "$TEST_CURL_LOG" ]] || fail 'Network failure stub was not exercised'
[[ ! -e "${AGORA_LIBS_DIR}.install-lock" ]] || fail 'Failed install left a lock'
[[ ! -e "$AGORA_SDK_CACHE_DIR" ]] || fail 'Failed download was cached'
rm "$TEST_CURL_LOG"
if TEST_ARCH=aarch64 "${root}/scripts/download_agora_libs.sh" >"${sandbox}/arm64.log" 2>&1; then
    fail 'Installer accepted Linux arm64'
fi
[[ ! -e "$TEST_CURL_LOG" ]] || fail 'Unsupported architecture reached download'

# Exercise archive validation through the real installer. Only an isolated copy
# of its manifest is changed to trust the deliberately invalid test fixtures.
mkdir "${sandbox}/fixture-repo"
cp -R "${root}/scripts" "${sandbox}/fixture-repo/"
for fixture in incomplete-sdk unsafe-sdk symlink-sdk; do
    fixture_archive="${root}/scripts/testdata/${fixture}.zip"
    cp "${root}/scripts/sdk_versions.sh" "${sandbox}/fixture-repo/scripts/sdk_versions.sh"
    printf "\nAGORA_LINUX_SHA256='%s'\n" "$(sdk_sha256 "$fixture_archive")" >> "${sandbox}/fixture-repo/scripts/sdk_versions.sh"
    if "${sandbox}/fixture-repo/scripts/download_agora_libs.sh" --archive "$fixture_archive" >"${sandbox}/${fixture}.log" 2>&1; then
        fail "Installer accepted $fixture fixture"
    fi
    assert_equal "$(cat "${AGORA_LIBS_DIR}/original")" 'keep me' "Invalid fixture changed old install"
    [[ ! -e "$AGORA_SDK_CACHE_DIR" ]] || fail 'Invalid fixture was cached'
    [[ ! -e "${AGORA_LIBS_DIR}.install-lock" ]] || fail 'Invalid fixture left a lock'
done
grep -Fq 'Required SDK library missing' "${sandbox}/incomplete-sdk.log" || fail 'Incomplete fixture failed for the wrong reason'
grep -Fq 'Unsafe ZIP member' "${sandbox}/unsafe-sdk.log" || fail 'Traversal was not rejected before extraction'
grep -Fq 'unsupported symlinks or special files' "${sandbox}/symlink-sdk.log" || fail 'Symlink was not rejected before extraction'
[[ ! -e "${sandbox}/escaped.txt" ]] || fail 'Unsafe archive escaped extraction directory'

# A marker alone must never count as a complete installation.
printf '%s\n' "$SDK_URL" > "${AGORA_LIBS_DIR}/sdk_version"
printf '%s\n' "$SDK_SHA256" > "${AGORA_LIBS_DIR}/sdk_sha256"
printf '%s\n' "$SDK_PLATFORM" > "${AGORA_LIBS_DIR}/sdk_platform"
if sdk_validate_installation "$AGORA_LIBS_DIR" 2>/dev/null; then fail 'Partial installation was accepted'; fi
for library in "${SDK_REQUIRED_FILES[@]}"; do printf 'fixture' > "${AGORA_LIBS_DIR}/${library}"; done
sdk_validate_installation "$AGORA_LIBS_DIR" || fail 'Complete fixture installation was rejected'
printf 'old release\n' > "${AGORA_LIBS_DIR}/sdk_sha256"
if sdk_validate_installation "$AGORA_LIBS_DIR" 2>/dev/null; then fail 'Stale marker was accepted'; fi

# Verify rollback if replacement fails after the old directory was moved.
sdk_begin_transaction "$AGORA_LIBS_DIR"
if sdk_commit_directory "${sandbox}/missing-replacement" 2>/dev/null; then fail 'Missing replacement succeeded'; fi
sdk_cleanup_transaction
assert_equal "$(cat "${AGORA_LIBS_DIR}/original")" 'keep me' 'Rollback lost original installation'
[[ ! -e "${AGORA_LIBS_DIR}.install-lock" ]] || fail 'Rollback left a lock'

# Verify a successful swap and reject replacing a symlink target.
sdk_begin_transaction "$AGORA_LIBS_DIR"
mkdir "${SDK_WORK_DIR}/new"
printf 'new install' > "${SDK_WORK_DIR}/new/replacement"
sdk_commit_directory "${SDK_WORK_DIR}/new"
sdk_cleanup_transaction
assert_equal "$(cat "${AGORA_LIBS_DIR}/replacement")" 'new install' 'Replacement failed'
ln -s "$AGORA_LIBS_DIR" "${sandbox}/symlink"
if sdk_begin_transaction "${sandbox}/symlink" 2>/dev/null; then fail 'Symlink installation was accepted'; fi
printf 'Installer tooling tests passed (offline).\n'
