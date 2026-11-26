#!/bin/sh
# Install drako: download the release for this system, check its checksum,
# and put the binary in ~/.local/bin (/usr/local/bin when run as root).
#
#   curl -fsSL https://github.com/lucky7xz/drako/releases/latest/download/install.sh | sh
#
# Optional settings (put them after the pipe: ... | DRAKO_VERSION=v0.5.0 sh):
#   DRAKO_VERSION=v0.5.0      install this release instead of the latest
#   DRAKO_INSTALL_DIR=~/bin   install into this folder instead
#
# Linux, macOS and FreeBSD. Never uses sudo, never edits shell files.
# Everything runs from main, on the last line: a cut-off download runs nothing.

set -eu

REPO=lucky7xz/drako

say() { echo "drako-install: $*"; }
die() { echo "drako-install: $*" >&2; exit 1; }

# The release archive for this system, e.g. drako_Linux_x86_64.tar.gz.
asset_name() {
	os=$(uname -s)
	arch=$(uname -m)
	case "$os" in
	Linux) os=Linux ;;
	Darwin) os=Darwin ;;
	FreeBSD) os=Freebsd ;;
	*) die "no release for $os. Linux, macOS and FreeBSD only; with Go: go install github.com/$REPO@latest" ;;
	esac
	case "$arch" in
	x86_64 | amd64) arch=x86_64 ;;
	aarch64 | arm64) arch=arm64 ;;
	armv7*) arch=armv7 ;;
	esac
	# Must match the builds in .goreleaser.yaml.
	case "$os/$arch" in
	Linux/x86_64 | Linux/arm64 | Linux/armv7 | Darwin/x86_64 | Darwin/arm64 | Freebsd/x86_64) ;;
	*) die "no release for $(uname -s) on $(uname -m); with Go: go install github.com/$REPO@latest" ;;
	esac
	echo "drako_${os}_${arch}.tar.gz"
}

download() { # url file
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL --proto '=https' -o "$2" "$1"
	elif command -v wget >/dev/null 2>&1; then
		wget -q -O "$2" "$1"
	else
		die "need curl or wget"
	fi
}

sha256() { # file
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | awk '{ print $1 }'
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$1" | awk '{ print $1 }'
	elif command -v sha256 >/dev/null 2>&1; then
		sha256 -q "$1"
	else
		die "need sha256sum, shasum or sha256 to check the download"
	fi
}

main() {
	asset=$(asset_name)

	version=${DRAKO_VERSION:-}
	case "$version" in
	"") base="https://github.com/$REPO/releases/latest/download" ;;
	*/* | *[!A-Za-z0-9.+-]*) die "DRAKO_VERSION should look like v0.5.0, not $version" ;;
	v[0-9]*) base="https://github.com/$REPO/releases/download/$version" ;;
	[0-9]*) base="https://github.com/$REPO/releases/download/v$version" ;;
	*) die "DRAKO_VERSION should look like v0.5.0, not $version" ;;
	esac

	if [ -n "${DRAKO_INSTALL_DIR:-}" ]; then
		dir=$DRAKO_INSTALL_DIR
	elif [ "$(id -u)" = 0 ]; then
		dir=/usr/local/bin
	else
		dir=$HOME/.local/bin
	fi

	tmp=$(mktemp -d)
	trap 'rm -rf "$tmp"' EXIT
	trap 'exit 1' HUP INT TERM

	say "downloading $asset (${version:-latest})"
	download "$base/$asset" "$tmp/$asset" || die "download failed: $base/$asset"
	download "$base/checksums.txt" "$tmp/checksums.txt" || die "download failed: $base/checksums.txt"

	# Compare file names exactly, not as a pattern. If a release came out
	# between the two downloads, the sums differ and nothing is installed.
	want=$(awk -v f="$asset" '$2 == f { print $1 }' "$tmp/checksums.txt")
	[ -n "$want" ] || die "checksums.txt lists no $asset"
	got=$(sha256 "$tmp/$asset")
	[ "$want" = "$got" ] || die "checksum mismatch for $asset, nothing installed (expected $want, got $got). Run it again; if it repeats, please report it."

	tar -xzf "$tmp/$asset" -C "$tmp" drako || die "could not unpack $asset"

	# Copy next to the target, then rename: a running drako is replaced in
	# one step, never half-written.
	mkdir -p "$dir" || die "could not create $dir"
	cp "$tmp/drako" "$dir/.drako.new" || die "could not write to $dir"
	chmod 0755 "$dir/.drako.new"
	mv -f "$dir/.drako.new" "$dir/drako" || die "could not write to $dir"

	say "installed $dir/drako"
	"$dir/drako" version 2>/dev/null || true

	case ":$PATH:" in
	*":$dir:"*) ;;
	*) say "$dir is not on your PATH. Add it to your shell's rc file: export PATH=\"$dir:\$PATH\"" ;;
	esac
	found=$(command -v drako 2>/dev/null || true)
	if [ -n "$found" ] && [ "$found" != "$dir/drako" ]; then
		say "note: your shell runs $found first, not this one"
	fi
}

main "$@"
