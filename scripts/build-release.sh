#!/bin/bash
# Builds the release archives and their SHA256SUMS into a directory.
#
# usage: scripts/build-release.sh [out dir] [version]
#
# The version defaults to the git tag (v1.3.0 builds 1.3.0). The archives are
# reproducible: the same source and Go toolchain give the same bytes, with the
# files dated to the last commit, or to SOURCE_DATE_EPOCH when it is set.
set -euo pipefail

cd "$(dirname "$0")/.."
OUT=$(mkdir -p "${1:-dist}" && cd "${1:-dist}" && pwd)
VERSION=${2:-$(git describe --tags --always --dirty)}
VERSION=${VERSION#v}
MTIME=@${SOURCE_DATE_EPOCH:-$(git log -1 --format=%ct)}
TARGETS="linux/386 linux/amd64 linux/arm/5 linux/arm/6 linux/arm/7 linux/arm64
  windows/386 windows/amd64 windows/arm64 darwin/amd64 darwin/arm64 freebsd/amd64 freebsd/arm64"
FILES="README.md LICENSE settings.toml"

export CGO_ENABLED=0 TZ=UTC
STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT

go version
for target in $TARGETS; do
  IFS=/ read -r os arch arm <<< "$target"
  name=speedtest-go_${VERSION}_${os}_${arch}${arm:+v$arm}
  bin=speedtest-backend
  [ "$os" = windows ] && bin=$bin.exe
  dir=$STAGE/$name
  mkdir "$dir"

  GOOS=$os GOARCH=$arch GOARM=$arm go build -trimpath -buildvcs=false \
    -ldflags "-w -s -X main.version=$VERSION" -o "$dir/$bin" .
  # shellcheck disable=SC2086
  cp $FILES "$dir/"
  # the same modes and dates whatever the umask and checkout time
  chmod 0644 "$dir"/*
  chmod 0755 "$dir/$bin"
  touch -d "$MTIME" "$dir"/*

  rm -f "$OUT/$name".{tar.gz,zip}
  if [ "$os" = windows ]; then
    # shellcheck disable=SC2086
    (cd "$dir" && zip -q -X -9 "$OUT/$name.zip" "$bin" $FILES)
  else
    # shellcheck disable=SC2086
    tar --sort=name --owner=0 --group=0 --numeric-owner --mtime="$MTIME" \
      -C "$dir" -cf - "$bin" $FILES | gzip -n -9 > "$OUT/$name.tar.gz"
  fi
  echo "$name"
done

cd "$OUT"
sha256sum speedtest-go_"${VERSION}"_* > SHA256SUMS
cat SHA256SUMS
if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "version=$VERSION" >> "$GITHUB_OUTPUT"
fi
