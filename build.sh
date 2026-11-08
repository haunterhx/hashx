#!/usr/bin/env bash
# hashx build helper (Linux / macOS / WSL / Git Bash)
#
#   ./build.sh            build for this machine        -> ./bin/
#   ./build.sh all        cross-compile release targets -> ./dist/ + SHA256SUMS
#   ./build.sh install    build and copy to ~/.local/bin (override with PREFIX=)
#   ./build.sh test       go vet + go test -race
#   ./build.sh clean      remove ./bin and ./dist
#
# Env overrides: VERSION=1.2.3  PREFIX=/usr/local/bin
set -euo pipefail

APP="hashx"
PKG="./cmd/hashx"
MIN_GO="1.22"
TARGETS=(linux/amd64 linux/arm64 windows/amd64 windows/arm64 darwin/amd64 darwin/arm64)

cd "$(dirname "${BASH_SOURCE[0]}")"

if [[ -t 1 ]]; then R=$'\033[1;31m'; G=$'\033[1;32m'; Y=$'\033[33m'; N=$'\033[0m'; else R='' G='' Y='' N=''; fi
info() { echo "${G}==>${N} $*"; }
warn() { echo "${Y}warning:${N} $*" >&2; }
die()  { echo "${R}error:${N} $*" >&2; exit 1; }

check_go() {
  if ! command -v go >/dev/null 2>&1; then
    echo "${R}Go is not installed or not on PATH.${N}" >&2
    case "$(uname -s)" in
      Linux)  echo "  Install: https://go.dev/dl/  (or: sudo apt install golang-go / sudo dnf install golang)" >&2 ;;
      Darwin) echo "  Install: brew install go   (or https://go.dev/dl/)" >&2 ;;
      *)      echo "  Install: https://go.dev/dl/" >&2 ;;
    esac
    exit 1
  fi
  local have; have="$(go env GOVERSION)"; have="${have#go}"; have="${have%%[!0-9.]*}"
  if [[ "$(printf '%s\n%s\n' "$MIN_GO" "$have" | sort -V | head -n1)" != "$MIN_GO" ]]; then
    die "Go $MIN_GO+ required, found $have"
  fi
}

version() {
  echo "${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
}

ldflags() {
  local commit date
  commit="$(git rev-parse --short HEAD 2>/dev/null || echo none)"
  date="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  # -X on a variable that doesn't exist is silently ignored, so commit/date are optional in main.go
  echo "-s -w -X main.version=$(version) -X main.commit=${commit} -X main.date=${date}"
}

build_one() { # $1=goos $2=goarch $3=outfile
  CGO_ENABLED=0 GOOS="$1" GOARCH="$2" \
    go build -trimpath -ldflags "$(ldflags)" -o "$3" "$PKG"
}

cmd_native() {
  check_go
  local ext=""; [[ "$(go env GOOS)" == "windows" ]] && ext=".exe"
  mkdir -p bin
  info "building $APP $(version) for $(go env GOOS)/$(go env GOARCH)"
  build_one "$(go env GOOS)" "$(go env GOARCH)" "bin/${APP}${ext}"
  info "done: bin/${APP}${ext}"
}

cmd_all() {
  check_go
  local v; v="$(version)"
  rm -rf dist && mkdir -p dist
  for t in "${TARGETS[@]}"; do
    local os="${t%/*}" arch="${t#*/}" out="dist/${APP}_${v}_${t%/*}_${t#*/}"
    [[ "$os" == "windows" ]] && out+=".exe"
    info "building $out"
    build_one "$os" "$arch" "$out"
  done
  if command -v sha256sum >/dev/null; then (cd dist && sha256sum -- * > SHA256SUMS)
  else (cd dist && shasum -a 256 -- * > SHA256SUMS); fi
  info "artifacts in ./dist"
  ls -lh dist
}

cmd_install() {
  cmd_native
  local prefix="${PREFIX:-$HOME/.local/bin}"
  mkdir -p "$prefix"
  install -m 0755 "bin/$APP" "$prefix/$APP"
  info "installed to $prefix/$APP"
  case ":$PATH:" in
    *":$prefix:"*) ;;
    *) warn "$prefix is not on your PATH. Add this to ~/.bashrc or ~/.zshrc:"
       echo "    export PATH=\"\$PATH:$prefix\"" ;;
  esac
}

cmd_test() {
  check_go
  info "go vet";  go vet ./...
  info "go test"; go test -race -count=1 ./...
}

cmd_clean() { rm -rf bin dist; info "cleaned"; }

case "${1:-native}" in
  native)  cmd_native ;;
  all)     cmd_all ;;
  install) cmd_install ;;
  test)    cmd_test ;;
  clean)   cmd_clean ;;
  -h|--help|help) sed -n '2,10p' "$0" | sed 's/^# \{0,1\}//' ;;
  *) die "unknown command '$1' (try: ./build.sh help)" ;;
esac