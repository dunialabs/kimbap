#!/usr/bin/env bash

set -euo pipefail

TARGETS=(
  "darwin/amd64"
  "linux/amd64"
  "windows/amd64"
)

echo "== kimbap build exclusion census =="
echo "module: $(go list -m)"

for target in "${TARGETS[@]}"; do
  goos="${target%/*}"
  goarch="${target#*/}"
  echo
  echo "-- ${goos}/${goarch} --"

  output="$(GOOS="${goos}" GOARCH="${goarch}" go list -f '{{if .IgnoredGoFiles}}{{.ImportPath}}:{{.IgnoredGoFiles}}{{end}}' ./...)"
  if [[ -z "${output}" ]]; then
    echo "(no ignored files)"
    continue
  fi
  printf '%s\n' "${output}"
done
