#!/usr/bin/env bash
# Build the release .deb in Ubuntu 22.04 so it links against glibc 2.35.
set -euo pipefail

root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

version="${VERSION:?VERSION is required}"
image="${DEB_IMAGE:-better-rpt-build:22.04}"
cache_prefix="${DEB_CACHE:-better-rpt-deb}"

if ! command -v docker >/dev/null 2>&1; then
	echo "error: docker is required to build the release .deb" >&2
	exit 1
fi

if [ ! -f cmd/better-rpt/dist/index.html ]; then
	echo "error: missing cmd/better-rpt/dist/index.html; run 'make frontend' first" >&2
	exit 1
fi

docker build --build-arg GO_VERSION=1.27.3 -t "$image" \
	-f packaging/Dockerfile.build packaging

container="better-rpt-build-$$"
cleanup() { docker rm -f "$container" >/dev/null 2>&1 || true; }
trap cleanup EXIT

# Reuse compiled packages and downloaded modules across release builds. Set
# DEB_NO_CACHE=1 for a clean build; DEB_CACHE changes the volume prefix.
cache_args=()
if [ "${DEB_NO_CACHE:-0}" != "1" ]; then
	cache_args=(
		--env GOCACHE=/cache/go-build
		--env GOMODCACHE=/cache/go-mod
		--volume "${cache_prefix}-go-build:/cache/go-build"
		--volume "${cache_prefix}-go-mod:/cache/go-mod"
	)
fi

docker run --name "$container" \
	"${cache_args[@]}" \
	-e VERSION="$version" \
	-v "$root:/src:ro" -w /src "$image" \
	sh -c 'set -e
		version=$1
		mkdir -p /tmp/dist
		CGO_ENABLED=1 go build -tags "gtk3 production" -trimpath -buildvcs=false \
			-ldflags "-s -w -X main.version=$version" \
			-o /tmp/dist/better-rpt ./cmd/better-rpt
		CGO_ENABLED=0 go build -trimpath -buildvcs=false \
			-ldflags "-s -w -X main.version=$version" \
			-o /tmp/dist/better-rpt-helper ./cmd/better-rpt-helper' _ "$version"

mkdir -p dist
docker cp "$container:/tmp/dist/better-rpt" dist/better-rpt
docker cp "$container:/tmp/dist/better-rpt-helper" dist/better-rpt-helper

nfpm="${NFPM:-$HOME/go/bin/nfpm}"
if [ ! -x "$nfpm" ]; then
	if command -v nfpm >/dev/null 2>&1; then
		nfpm="$(command -v nfpm)"
	else
		echo "error: nfpm v2.48.0 not found at $nfpm or in PATH" >&2
		echo "install it with: go install github.com/goreleaser/nfpm/v2/cmd/nfpm@v2.48.0" >&2
		exit 1
	fi
fi

if ! "$nfpm" --version | grep -q 'v2\.48\.0'; then
	echo "error: nfpm v2.48.0 is required" >&2
	"$nfpm" --version >&2 || true
	exit 1
fi

VERSION="$version" SOURCE_DATE_EPOCH=1700000000 \
	"$nfpm" package --config packaging/nfpm.yaml --packager deb --target dist/

echo "built dist/better-rpt_${version}-1_amd64.deb"
