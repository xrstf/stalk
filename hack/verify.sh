#!/usr/bin/env bash

set -euo pipefail
cd $(dirname $0)/..

EXIT_CODE=0

errcho() {
  echo $@ >&2
}

try() {
  local title="$1"
  shift

  echo "===================================="
  echo "$title"
  echo "===================================="
  echo

  startTime=$(date +%s)

  set +e
  $@
  exitCode=$?
  set -e

  elapsed=$(($(date +%s) - $startTime))

  if [[ $exitCode -eq 0 ]]; then
    echo -e "\n[${elapsed}s] SUCCESS :)"
  else
    echo -e "\n[${elapsed}s] FAILED."
    EXIT_CODE=1
  fi

  git reset --hard --quiet
  git clean --force

  echo
}

function verify_go_mod_tidy() (
  set -e

  # bad formatting in go.sum is not automatically fixed by go, for some reason
  (set -x; rm go.sum; go mod tidy)

  if ! git diff --exit-code; then
    echo "::error::Please run go mod tidy."
    return 1
  fi

  echo "go.mod is tidy."
)

function ensure_gimps() {
  location=gimps

  if ! [ -x "$(command -v gimps)" ]; then
    version=0.6.0
    arch="$(go env GOARCH)"
    os="$(go env GOOS)"
    url="https://github.com/xrstf/gimps/releases/download/v${version}/gimps_${version}_${os}_${arch}.tar.gz"
    location=/tmp/gimps

    errcho "Downloading gimps v$version..."
    wget -qO- "$url" | tar xzOf - "gimps_${version}_${os}_${arch}/gimps" > $location
    chmod +x $location
  fi

  echo "$location"
}

function verify_go_imports() (
  set -e

  gimps="$(ensure_gimps)"

  (set -x; $gimps .)

  if ! git diff --exit-code; then
    echo "::error::Some import statements are not properly grouped. Please run https://github.com/xrstf/gimps or sort them manually."
    return 1
  fi

  echo "Go import statements are properly sorted."
)

function verify_go_build() (
  set -e

  if ! (set -x; make build); then
    echo "::error::Code does not compile."
    return 1
  fi

  echo "Code compiles."
)

function verify_go_lint() (
  set -e

  if ! (set -x; golangci-lint run ./...); then
    echo "::error::Computer says no."
    return 1
  fi

  echo "Code looks sane."
)

try "go.mod tidy?" verify_go_mod_tidy
try "gimpsed?" verify_go_imports
try "Go code builds?" verify_go_build

exit $EXIT_CODE
