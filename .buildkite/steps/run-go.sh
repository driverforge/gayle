#!/bin/sh
# One CI task, named by the first argument.
#
# Every task reproduces locally — that is the point of keeping them one line
# each rather than hiding them behind a script-specific abstraction:
#
#   lint   make fmt-check && make vet
#   build  make build
#   test   go test -race -shuffle=on ./...
#   vuln   go run golang.org/x/vuln/cmd/govulncheck@latest ./...
#
# -race and -shuffle belong to CI rather than the Makefile: `make test` stays
# the fast local loop. -shuffle catches tests that only pass in declaration
# order, and -race catches what both of those hide.
set -eu

TASK="${1:?usage: run-go.sh lint|build|test|vuln}"

echo "NODE: ${K8S_NODE:-<unset>}"

case "$TASK" in
  lint)
    make fmt-check
    make vet
    ;;

  build)
    make build
    ;;

  test)
    go test -race -shuffle=on ./...
    ;;

  vuln)
    # Symbol-level reachability: fails only when our code actually calls (or is
    # compiled against) vulnerable code — including standard library and
    # toolchain advisories a dependency scanner cannot see — and stays quiet on
    # advisories whose code is never reached.
    #
    # A failure here with NO DIFF usually means a new advisory landed in the
    # vulnerability database rather than that this change broke something.
    # Triage with `govulncheck -show verbose ./...`.
    #
    # Its own step, so that a new advisory does not mask a real test failure and
    # a real test failure does not hide a new advisory. They answer different
    # questions and neither should wait on the other.
    go run golang.org/x/vuln/cmd/govulncheck@latest ./...
    ;;

  *)
    echo "unknown task: ${TASK} (expected lint, build, test or vuln)" >&2
    exit 1
    ;;
esac
