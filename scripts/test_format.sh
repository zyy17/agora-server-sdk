#!/usr/bin/env bash
# Formatter regressions use isolated source fixtures, without native libraries.
set -euo pipefail
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
go_command=${1:-go}
export GOIMPORTS=${GOIMPORTS:-"${root}/.tools/bin/goimports"}
export GOPROXY=off GOSUMDB=off
sandbox=$(mktemp -d)
trap 'rm -rf "$sandbox"' EXIT
fail() { printf 'FAIL: %s\n' "$*" >&2; exit 1; }

mkdir -p "${sandbox}/scripts" "${sandbox}/agora" "${sandbox}/examples/with spaces"
cp "${root}/scripts/check_format.sh" "${sandbox}/scripts/"
mkdir -p "${sandbox}/internal/cmd/funcspacing"
cp "${root}/internal/cmd/funcspacing/main.go" "${sandbox}/internal/cmd/funcspacing/"
printf 'module example.org/sdk\n' > "${sandbox}/go.mod"
cat > "${sandbox}/source.in" <<'GO'
//go:build formatfixture
// +build formatfixture

package formatfixture

/*
#cgo CFLAGS: -DFORMAT_FIXTURE=1
#include <stdlib.h>
*/
import "C"
import (
	rtc "example.org/sdk/rtc"
	_ "example.net/driver"
	"os"
	third "example.net/library"
	"example.org/sdkextra"
	sdk "example.org/sdk"
	"fmt"
)

func number()int{return int(C.int(1))}
// next keeps its directive attached.
//
//go:noinline
func next()int{return 2}
func lastNumber()int{return 3}
GO
cat > "${sandbox}/expected.out" <<'GO'
//go:build formatfixture
// +build formatfixture

package formatfixture

/*
#cgo CFLAGS: -DFORMAT_FIXTURE=1
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"os"

	_ "example.net/driver"
	third "example.net/library"
	"example.org/sdkextra"

	sdk "example.org/sdk"
	rtc "example.org/sdk/rtc"
)

func number() int { return int(C.int(1)) }

// next keeps its directive attached.
//
//go:noinline
func next() int { return 2 }

func lastNumber() int { return 3 }
GO

# Include native wrappers, tests, inactive tags, and paths containing spaces.
files=(agora/native.go source_test.go 'examples/with spaces/main.go')
for file in "${files[@]}"; do cp "${sandbox}/source.in" "${sandbox}/${file}"; done
for dir in .git .cache .tools agora_libs bin vendor; do
    mkdir -p "${sandbox}/${dir}"
    printf 'Not Go source: this directory must be skipped.\n' > "${sandbox}/${dir}/ignored.go"
done
check=("${sandbox}/scripts/check_format.sh" "$go_command")
if "${check[@]}" >"${sandbox}/check.log" 2>&1; then
    fail 'Unformatted imports were accepted'
fi
grep -Fq 'Run make fmt' "${sandbox}/check.log" || fail 'Missing repair instructions'
for file in "${files[@]}"; do
    grep -Fq "$file" "${sandbox}/check.log" || fail "Missing format failure for $file"
    cmp -s "${sandbox}/source.in" "${sandbox}/${file}" || fail 'Check mode changed source'
done

"${check[@]}" --write
for file in "${files[@]}"; do
    diff -u "${sandbox}/expected.out" "${sandbox}/${file}" || fail "Unexpected formatting for $file"
done
"${check[@]}"
"${check[@]}" --write
for file in "${files[@]}"; do
    cmp -s "${sandbox}/expected.out" "${sandbox}/${file}" || fail 'Formatting is not idempotent'
done

# A gofmt-clean file must still fail when adjacent one-line functions lack a gap.
printf 'package spacing\n\nfunc one() {}\nfunc two() {}\n' > "${sandbox}/spacing.go"
cp "${sandbox}/spacing.go" "${sandbox}/spacing.in"
if "${check[@]}" >"${sandbox}/spacing.log" 2>&1; then
    fail 'Missing function spacing was accepted'
fi
grep -Fq 'spacing.go' "${sandbox}/spacing.log" || fail 'Missing spacing diagnostic'
cmp -s "${sandbox}/spacing.in" "${sandbox}/spacing.go" || fail 'Check mode changed spacing'
"${check[@]}" --write
"${check[@]}"

printf 'package broken\nfunc {\n' > "${sandbox}/broken.go"
if "${check[@]}" >"${sandbox}/syntax.log" 2>&1; then fail 'Invalid Go syntax was accepted'; fi
grep -Fq 'broken.go' "${sandbox}/syntax.log" || fail 'Syntax failure did not identify its file'
if GOIMPORTS="${sandbox}/missing" "${check[@]}" >"${sandbox}/missing.log" 2>&1; then
    fail 'Missing formatter was accepted'
fi
grep -Fq 'make install-goimports' "${sandbox}/missing.log" || fail 'Missing formatter setup instructions'
printf 'Formatter tooling tests passed (offline).\n'
