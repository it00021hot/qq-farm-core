#!/usr/bin/env bash
# Generate farm protobuf Go code from internal/farm/proto/*.proto
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PROTO_DIR="$ROOT/internal/farm/proto"
MODULE_PREFIX="github.com/it00021hot/qq-farm-core/internal/farm/proto"

export PATH="${HOME}/.local/bin:$(go env GOPATH)/bin:${PATH}"

if ! command -v protoc >/dev/null 2>&1; then
  echo "protoc not found; install protobuf compiler first" >&2
  exit 1
fi
if ! command -v protoc-gen-go >/dev/null 2>&1; then
  echo "protoc-gen-go not found; run: go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11" >&2
  exit 1
fi

pkg_for() {
  case "$1" in
    game.proto) echo gatepb ;;
    notifypb.proto) echo itempb ;;
    *) echo "${1%.proto}" ;;
  esac
}

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

OPTS=(
  -I="$PROTO_DIR"
  --go_out="$TMP"
  --go_opt=paths=source_relative
)

while IFS= read -r -d '' f; do
  base="$(basename "$f")"
  pkg="$(pkg_for "$base")"
  OPTS+=(--go_opt="M${base}=${MODULE_PREFIX}/${pkg}")
done < <(find "$PROTO_DIR" -maxdepth 1 -name '*.proto' -print0 | sort -z)

protoc "${OPTS[@]}" "$PROTO_DIR"/*.proto

while IFS= read -r -d '' gen; do
  base="$(basename "$gen")"
  case "$base" in
    game.pb.go) pkg=gatepb ;;
    notifypb.pb.go) pkg=itempb ;;
    *.pb.go) pkg="${base%.pb.go}" ;;
    *) continue ;;
  esac
  mkdir -p "$PROTO_DIR/$pkg"
  mv "$gen" "$PROTO_DIR/$pkg/$base"
  echo "wrote $pkg/$base"
done < <(find "$TMP" -name '*.pb.go' -print0)

echo "farm proto generation complete"
