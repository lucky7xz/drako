#!/bin/sh
# Generate THIRD-PARTY-NOTICES.md from the modules linked into the binary.
#
# MIT, BSD and ISC all require their copyright notice to travel with copies of
# the software, and a compiled binary is a copy. Generated at release time so it
# can never drift from go.mod.
#
# Deps of the main package only — `go list -m all` would also pull in test-only
# modules that never reach the binary.
set -eu

out="${1:-THIRD-PARTY-NOTICES.md}"
self="github.com/lucky7xz/drako"

{
	echo "# Third-party notices"
	echo
	echo "drako links the Go modules below into its release binaries. Each is listed"
	echo "with the copyright notice and licence its own terms require us to carry."
	echo
	echo "drako itself is AGPL-3.0 — see LICENCE. Bootstrap assets are MIT, except"
	echo "where a file's own notice says otherwise."
} >"$out"

go list -deps -f '{{with .Module}}{{.Path}}	{{.Version}}	{{.Dir}}{{end}}' . |
	sort -u |
	while IFS='	' read -r path version dir; do
		[ -n "${dir:-}" ] || continue        # stdlib has no module
		[ "$path" != "$self" ] || continue   # our own licence ships separately

		lic=$(ls "$dir" 2>/dev/null | grep -iE '^(licen[cs]e|copying)' | head -1) || true
		if [ -z "$lic" ]; then
			echo "warning: no licence file found for $path $version" >&2
			continue
		fi

		{
			echo
			echo "## $path $version"
			echo
			echo '```'
			cat "$dir/$lic"
			echo '```'
		} >>"$out"
	done

# Embedded assets are compiled into the binary by //go:embed, so their notices
# have to travel with it just as the module licences above do. Each file states
# its own in leading # lines.
scales="internal/config/bootstrap/scales"
if [ -d "$scales" ]; then
	{
		echo
		echo "## Embedded assets"
		echo
		echo "Carried inside the binary, and written into the config directory on first run."
	} >>"$out"

	for f in "$scales"/*; do
		[ -f "$f" ] || continue
		notice=$(grep '^#' "$f" | sed 's/^#[[:space:]]\{0,1\}//') || true
		if [ -z "$notice" ]; then
			echo "warning: no notice in $f" >&2
			continue
		fi
		{
			echo
			echo "### ${f#internal/config/bootstrap/}"
			echo
			echo '```'
			echo "$notice"
			echo '```'
		} >>"$out"
	done
fi

echo "wrote $out ($(grep -c '^## github\|^## golang' "$out") modules)"
