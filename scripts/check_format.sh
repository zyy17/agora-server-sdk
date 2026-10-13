#!/usr/bin/env bash
set -euo pipefail
root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
go_command=${1:-go}
gofmt="$("$go_command" env GOROOT)/bin/gofmt"
cd "$root"
files=()
while IFS= read -r -d '' file; do files+=("$file"); done < <(
    find . -type d \( -name .git -o -name agora_libs -o -name bin -o -name vendor \) -prune \
        -o -type f -name '*.go' -print0
)
[[ ${#files[@]} -gt 0 ]] || exit 0
if [[ "${2:-}" == --write ]]; then
    "$gofmt" -w "${files[@]}"
    exit 0
fi
unformatted=$("$gofmt" -l "${files[@]}")
if [[ -n "$unformatted" ]]; then
    printf 'Run make fmt; these Go files need formatting:\n%s\n' "$unformatted" >&2
    exit 1
fi
