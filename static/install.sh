#!/bin/sh
# fabrik installer: downloads the latest prebuilt fabrik CLI and puts it on PATH.
#
#   curl -fsSL https://gofabrik.dev/install.sh | sh
#
# Override the target directory with FABRIK_INSTALL_DIR=/path.
# Prefer building from source? See https://gofabrik.dev/docs/#install-source
set -eu

repo="gofabrik/fabrik"
bin="fabrik"
alt="https://gofabrik.dev/docs/#install-source"

info() { printf '%s\n' "$*"; }
die()  { printf 'fabrik install: %s\n' "$*" >&2; exit 1; }

os="$(uname -s)"
case "$os" in
  Darwin) os="darwin" ;;
  Linux)  os="linux" ;;
  *) die "unsupported OS '$os'; install from source instead: $alt" ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64 | amd64) arch="amd64" ;;
  arm64 | aarch64) arch="arm64" ;;
  *) die "unsupported architecture '$arch'; install from source instead: $alt" ;;
esac

asset="${bin}_${os}_${arch}.tar.gz"
base="https://github.com/${repo}/releases/latest/download"

if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL "$1" -o "$2"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -qO "$2" "$1"; }
else
  die "need curl or wget"
fi

sha256() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then shasum -a 256 "$1" | awk '{print $1}'
  else return 1
  fi
}

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

info "Downloading ${asset} (latest release)..."
fetch "${base}/${asset}" "${tmp}/${asset}" \
  || die "no prebuilt binary for ${os}/${arch}; install from source instead: $alt"

fetch "${base}/checksums.txt" "${tmp}/checksums.txt" 2>/dev/null \
  || die "could not download checksums.txt to verify ${asset}"
want="$(awk -v f="$asset" '$2 == f || $2 == "*"f {print $1}' "${tmp}/checksums.txt")"
[ -n "$want" ] || die "no checksum entry for ${asset} in checksums.txt"
got="$(sha256 "${tmp}/${asset}")" || die "need sha256sum or shasum to verify ${asset}"
[ "$want" = "$got" ] || die "checksum mismatch for ${asset}"
info "Checksum verified."

tar -xzf "${tmp}/${asset}" -C "$tmp" || die "could not extract ${asset}"
[ -f "${tmp}/${bin}" ] || die "archive did not contain '${bin}'"
chmod +x "${tmp}/${bin}"

if [ -n "${FABRIK_INSTALL_DIR:-}" ]; then
  dir="$FABRIK_INSTALL_DIR"
elif [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
  dir="/usr/local/bin"
else
  dir="${HOME}/.local/bin"
fi

mkdir -p "$dir" || die "cannot create $dir"
mv "${tmp}/${bin}" "${dir}/${bin}" || die "cannot write to $dir (set FABRIK_INSTALL_DIR to a writable dir)"
info "Installed ${bin} to ${dir}/${bin}"

case ":${PATH}:" in
  *":${dir}:"*)
    info "Run 'fabrik' to get started." ;;
  *)
    info ""
    info "${dir} is not on your PATH. Add it, then restart your shell:"
    info "  export PATH=\"\$PATH:${dir}\"" ;;
esac
