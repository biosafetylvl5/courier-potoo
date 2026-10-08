#!/usr/bin/env bash
# Cross-compile potoo-server and package release archives into dist/.
#
# Usage: scripts/build-release.sh VERSION
#
# Each archive holds the binary, README.md, docker-compose.yml and the
# Courier service YAML. SHA256SUMS covers every archive.
set -euo pipefail

VERSION="${1:?usage: $0 VERSION}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="$ROOT/dist"
TARGETS=(
  darwin/arm64
  darwin/amd64
  linux/arm64
  linux/amd64
  windows/arm64
  windows/amd64
)

rm -rf "$DIST"
mkdir -p "$DIST"

for target in "${TARGETS[@]}"; do
  os="${target%/*}"
  arch="${target#*/}"
  name="potoo-server_${VERSION}_${os}_${arch}"
  stage="$DIST/$name"
  ext=""
  [ "$os" = windows ] && ext=".exe"

  echo "building $name"
  mkdir -p "$stage/services/courier"
  (
    cd "$ROOT/server"
    CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build \
      -trimpath \
      -ldflags "-s -w -X main.version=$VERSION" \
      -o "$stage/potoo-server$ext" .
  )
  cp "$ROOT/README.md" "$ROOT/docker-compose.yml" "$stage/"
  cp "$ROOT"/services/courier/*.yaml "$stage/services/courier/"

  (
    cd "$DIST"
    if [ "$os" = windows ]; then
      zip -qr "$name.zip" "$name"
    else
      tar -czf "$name.tar.gz" "$name"
    fi
  )
  rm -rf "$stage"
done

cd "$DIST"
if command -v sha256sum >/dev/null; then
  sha256sum -- *.tar.gz *.zip >SHA256SUMS
else
  shasum -a 256 -- *.tar.gz *.zip >SHA256SUMS
fi
cat SHA256SUMS
