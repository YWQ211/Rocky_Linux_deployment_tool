#!/bin/sh
# Build the fixed release, refusing to publish bytes that differ from its pins.
set -eu
cd "$(dirname "$0")/.."
version=$(cat release/VERSION)
printf '%s\n' "$version" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'
grep -Fqx "const Version = \"$version\"" internal/cli/cli.go
export GOTOOLCHAIN=local
go test ./... -count=1
go vet ./...
python3 scripts/test-launcher.py
mkdir -p dist/release
for arch in amd64 arm64; do
    CGO_ENABLED=0 GOOS=linux GOARCH=$arch go build -buildvcs=false -trimpath -ldflags '-s -w' \
        -o "dist/release/rockdev-linux-$arch" ./cmd/rockdev
done
(cd dist/release && sha256sum --strict -c ../../release/SHA256SUMS)
cp release/SHA256SUMS LICENSE THIRD_PARTY_NOTICES dist/release/
printf 'Release %s built and checksum-pinned.\n' "$version"
