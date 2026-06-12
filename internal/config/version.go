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
//   - `go install <module>@vX.Y.Z`: read from the embedded module info
//   - plain `go build` / `go run`: "dev"
func Version() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return "dev"
}
