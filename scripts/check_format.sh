#!/usr/bin/env bash
set -euo pipefail
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
go_command=${1:-go}
goimports=${GOIMPORTS:-"${root}/.tools/bin/goimports"}
if [[ ! -x "$goimports" ]]; then
    printf 'Missing goimports; run make install-goimports first.\n' >&2
    exit 1
fi
cd "$root"
module_path=$("$go_command" list -m -f '{{.Path}}')
files=()
while IFS= read -r -d '' file; do files+=("$file"); done < <(
    find . -type d \( -name .git -o -name .cache -o -name .tools -o -name agora_libs -o -name bin -o -name vendor \) -prune \
        -o -type f -name '*.go' -print0
)
[[ ${#files[@]} -gt 0 ]] || exit 0
# Use parser-based grouping without adding/removing imports or resolving packages.
# This also covers tests and inactive build tags, and preserves cgo preambles.
options=(-format-only -local "${module_path}/")
if [[ "${2:-}" == --write ]]; then
    "$goimports" "${options[@]}" -w "${files[@]}"
    "$go_command" run ./internal/cmd/funcspacing -w "${files[@]}"
    # The new gaps end gofmt alignment groups between adjacent one-line functions.
    "$goimports" "${options[@]}" -w "${files[@]}"
    exit 0
fi
unformatted=$("$goimports" "${options[@]}" -l "${files[@]}")
if [[ -n "$unformatted" ]]; then
    printf 'Run make fmt; these Go files need formatting:\n%s\n' "$unformatted" >&2
    exit 1
fi
# Go's standard formatter permits adjacent one-line functions and methods.
# Parse declarations separately so their attached docs and directives stay intact.
if ! unspaced=$("$go_command" run ./internal/cmd/funcspacing "${files[@]}"); then
    printf 'Run make fmt; these Go files need function spacing:\n%s\n' "$unspaced" >&2
    exit 1
fi
