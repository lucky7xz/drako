package config

import "runtime/debug"

// AppName is the full name of the application, used for display.
const AppName = "lucky7xz/drako"

// version is stamped into release binaries at build time via -ldflags:
//
//	-ldflags "-X github.com/lucky7xz/drako/internal/config.version=v1.2.3"
//
// It is empty for other builds; Version() then derives the value from the
// embedded module info or falls back to "dev".
var version = ""

// Version returns the running build's version string:
//   - release binaries: the tag stamped in at build time (ldflags)
//   - `go install <module>@vX.Y.Z`: the tag, from the embedded module info
//   - local builds from the repo: "dev+<short-commit>[-dirty]"
//   - anything else: "dev"
func Version() string {
	if version != "" {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}

	// A local VCS build carries vcs.* settings and an unhelpfully long
	// pseudo-version; collapse it to a short dev id instead.
	var rev string
	var dirty, fromVCS bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			fromVCS = true
			if len(s.Value) >= 7 {
				rev = s.Value[:7]
			}
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if fromVCS {
		v := "dev"
		if rev != "" {
			v += "+" + rev
		}
		if dirty {
			v += "-dirty"
		}
		return v
	}

	// Installed at a tag: the embedded module version is the tag.
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	return "dev"
}
