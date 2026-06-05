#!/bin/sh
# install.sh — download, verify, and install the chip binary.
#
#   curl -fsSL https://raw.githubusercontent.com/jackspirou/chip/master/install.sh | sh
#
# Every download is checksum-verified against the release's checksums.txt.
#
# Environment overrides:
#   CHIP_VERSION      release tag to install (default: latest release)
#   CHIP_INSTALL_DIR  install directory (default: first writable of
#                     /usr/local/bin, then $HOME/.local/bin)
set -eu

REPO="jackspirou/chip"
RELEASES="https://github.com/${REPO}/releases"
API="https://api.github.com/repos/${REPO}/releases/latest"

say() { printf 'chip-install: %s\n' "$1" >&2; }
die() { say "error: $1"; exit 1; }

if command -v curl >/dev/null 2>&1; then
	dl() { curl -fsSL -o "$2" "$1"; }
	fetch() { curl -fsSL "$1"; }
elif command -v wget >/dev/null 2>&1; then
	dl() { wget -qO "$2" "$1"; }
	fetch() { wget -qO- "$1"; }
else
	die "need curl or wget"
fi
command -v tar >/dev/null 2>&1 || die "need tar"

os=$(uname -s)
case "$os" in
Linux) os=linux ;;
Darwin) os=darwin ;;
*) die "unsupported OS: $os (try the prebuilt binaries at ${RELEASES})" ;;
esac

arch=$(uname -m)
case "$arch" in
x86_64 | amd64) arch=amd64 ;;
arm64 | aarch64) arch=arm64 ;;
*) die "unsupported architecture: $arch" ;;
esac

version="${CHIP_VERSION:-}"
if [ -z "$version" ]; then
	say "resolving latest release"
	version=$(fetch "$API" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1)
	[ -n "$version" ] || die "could not resolve latest version; set CHIP_VERSION"
fi
nov="${version#v}"

archive="chip_${nov}_${os}_${arch}.tar.gz"
url="${RELEASES}/download/${version}/${archive}"
sums="${RELEASES}/download/${version}/checksums.txt"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

say "downloading ${archive} (${version})"
dl "$url" "${tmp}/${archive}" || die "download failed: ${url}"
dl "$sums" "${tmp}/checksums.txt" || die "download failed: ${sums}"

want=$(awk -v f="$archive" '$2 == f {print $1}' "${tmp}/checksums.txt" | head -1)
[ -n "$want" ] || die "no checksum listed for ${archive}"
if command -v sha256sum >/dev/null 2>&1; then
	got=$(sha256sum "${tmp}/${archive}" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
	got=$(shasum -a 256 "${tmp}/${archive}" | awk '{print $1}')
else
	die "need sha256sum or shasum to verify the download"
fi
[ "$want" = "$got" ] || die "checksum mismatch for ${archive}: want ${want}, got ${got}"
say "checksum verified"

tar -xzf "${tmp}/${archive}" -C "$tmp"
[ -f "${tmp}/chip" ] || die "archive did not contain a chip binary"
chmod +x "${tmp}/chip"

dir="${CHIP_INSTALL_DIR:-}"
if [ -z "$dir" ]; then
	for cand in /usr/local/bin "$HOME/.local/bin"; do
		if [ -d "$cand" ] && [ -w "$cand" ]; then
			dir="$cand"
			break
		fi
	done
	[ -n "$dir" ] || dir="$HOME/.local/bin"
fi
mkdir -p "$dir" || die "cannot create install dir: ${dir}"
mv "${tmp}/chip" "${dir}/chip" || die "cannot install to ${dir} (set CHIP_INSTALL_DIR)"
say "installed chip ${version} to ${dir}/chip"

case ":${PATH}:" in
*":${dir}:"*) ;;
*) say "note: ${dir} is not on your PATH" ;;
esac
"${dir}/chip" version || true
