#!/bin/sh
# gayle's CI — DF-1011. Ported from .github/workflows/ci.yml, which is deleted.
#
# Reproduce any of it locally; the commands are the ones CI runs:
#
#   make fmt-check
#   make vet
#   make build
#   go test -race -shuffle=on ./...
#   go run golang.org/x/vuln/cmd/govulncheck@latest ./...
#
# Every check runs even when an earlier one fails. The workflow stopped at the
# first, so a change with two problems took two round trips to find out.
#
# -race and -shuffle are CI's, not the Makefile's: `make test` is the fast local
# loop. -shuffle catches tests that only pass in declaration order, and -race
# catches what both of those hide.
set -eu

echo "NODE: ${K8S_NODE:-<unset>}"

FAILED=""

check() {
  _name=$1
  shift

  echo "--- :test_tube: ${_name}"
  _s=$(date +%s)
  if "$@"; then
    _status=ok
  else
    _status=FAILED
    FAILED="${FAILED} ${_name}"
  fi
  _e=$(date +%s)
  echo "CHECK[${_name}]: $((_e - _s))s ${_status}"
}

check "fmt-check" make fmt-check
check "vet" make vet
check "build" make build
check "test" go test -race -shuffle=on ./...

# Symbol-level vulnerability reachability: fails only when our code actually
# calls (or is compiled against) vulnerable code — including stdlib and
# toolchain vulns Dependabot cannot see — and stays quiet on module-level
# advisories whose code we never use.
#
# A failure here with NO DIFF usually means a new advisory landed in the vuln
# database rather than that this change broke something. Triage with
# `govulncheck -show verbose ./...`.
#
# Last, because it is the check most likely to fail for reasons unrelated to the
# change under test, and the others are what the author is waiting on.
check "govulncheck" go run golang.org/x/vuln/cmd/govulncheck@latest ./...

if [ -n "$FAILED" ]; then
  echo "--- :boom: failed:${FAILED}"
  exit 1
fi

echo "--- :white_check_mark: all checks passed"
