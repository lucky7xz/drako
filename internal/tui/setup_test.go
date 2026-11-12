package tui

import (
	"os"

	"github.com/lucky7xz/drako/internal/host"
	"github.com/lucky7xz/drako/internal/lair"
)

// setupFromEnv is the Setup app.Run would build: the config folder and the
// wanted profile from the environment, the platform from this machine.
func setupFromEnv(glassroot bool) Setup {
	configDir, err := lair.ConfigDir()
	if err != nil {
		panic(err)
	}
	return Setup{
		ConfigDir: configDir,
		Targets:   host.Targets(),
		Profile:   os.Getenv("DRAKO_PROFILE"),
		Glassroot: glassroot,
	}
}
