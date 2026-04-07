#!/usr/bin/env sh
set -eu

REPO="${REPO:-nanoinfluencer/nano-cli}"
BIN_DIR="${BIN_DIR:-/usr/local/bin}"
VERSION="${VERSION:-latest}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"

case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *)
    echo "unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

case "$os" in
  darwin|linux) ;;
  *)
    echo "unsupported operating system: $os" >&2
    exit 1
    ;;
esac

if [ "$VERSION" = "latest" ]; then
  VERSION="$(
    curl -fsSL -o /dev/null -w '%{url_effective}' "https://github.com/$REPO/releases/latest" \
      | sed -n 's#.*/tag/\([^/]*\)$#\1#p'
  )"
  if [ -z "$VERSION" ]; then
    echo "failed to resolve latest release tag for $REPO" >&2
    exit 1
  fi
fi

release_url="https://github.com/$REPO/releases/download/$VERSION"
version_no_v="${VERSION#v}"
archive="nanoinf_${version_no_v}_${os}_${arch}.tar.gz"
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT INT TERM

echo "Downloading $archive from $release_url"
curl -fsSL "$release_url/$archive" -o "$tmpdir/$archive"
tar -xzf "$tmpdir/$archive" -C "$tmpdir"

mkdir -p "$BIN_DIR"
install "$tmpdir/nanoinf" "$BIN_DIR/nanoinf"

echo "Installed nanoinf to $BIN_DIR/nanoinf"
