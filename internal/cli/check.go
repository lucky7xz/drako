package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/paths"
	"github.com/lucky7xz/drako/internal/profiles"
)

// HandleCheckCommand processes 'drako check [path ...]': lint profile files
// for authoring mistakes. With no arguments it checks config.toml and every
// equipped and inventory profile. Returns exit code 1 when any error-level
// finding exists, so deck repositories can run it in CI.
func HandleCheckCommand(args []string) int {
	targets, err := checkTargets(args[2:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	if len(targets) == 0 {
		fmt.Println("No profile files to check.")
		return 0
	}

	if runChecks(os.Stdout, targets) > 0 {
		return 1
	}
	return 0
}

// checkTargets resolves the argument list to profile files. Arguments may be
// files or directories (scanned non-recursively for *.profile.toml); no
// arguments means config.toml, the equipped dir and the inventory.
func checkTargets(args []string) ([]string, error) {
	if len(args) == 0 {
		configDir, err := paths.ConfigDir()
		if err != nil {
			return nil, fmt.Errorf("could not get config dir: %v", err)
		}
		if _, err := os.Stat(paths.ConfigFile(configDir)); err == nil {
			args = append(args, paths.ConfigFile(configDir))
		}
		// A default dir that doesn't exist yet has nothing to check.
		for _, d := range []string{configDir, paths.InventoryDir(configDir)} {
			if _, err := os.Stat(d); !errors.Is(err, fs.ErrNotExist) {
				args = append(args, d)
			}
		}
	}

	var files []string
	for _, a := range args {
		info, err := os.Stat(a)
		switch {
		case err != nil:
			return nil, fmt.Errorf("cannot read %s: %v", a, err)
		case info.IsDir():
			matches, _ := filepath.Glob(filepath.Join(a, "*"+profiles.ProfileSuffix))
			files = append(files, matches...)
		default:
			files = append(files, a)
		}
	}
	sort.Strings(files)
	return files, nil
}

// runChecks lints every file, prints findings as a table, and returns the
// number of files with error-level findings.
func runChecks(out io.Writer, files []string) int {
	filesWithErrors := 0
	total := 0
	var rows [][]string
	twins := caseTwins(files)
	for _, f := range files {
		src, err := os.ReadFile(f)
		var findings []config.Finding
		switch {
		case err != nil:
			findings = []config.Finding{{Level: "error", Msg: fmt.Sprintf("cannot read file: %v", err)}}
		case filepath.Base(f) == paths.ConfigFileName:
			findings = config.CheckSettings(src)
		default:
			findings = config.CheckProfile(src)
		}
		if len(twins[f]) > 0 {
			findings = append(findings, config.Finding{Level: "warning", Msg: fmt.Sprintf(
				"clashes with %s: drako ignores case in deck names", strings.Join(twins[f], ", "))})
		}

		if len(findings) == 0 {
			rows = append(rows, []string{filepath.Base(f), "ok", ""})
			continue
		}
		hasError := false
		for _, fd := range findings {
			rows = append(rows, []string{filepath.Base(f), fd.Level, fd.Msg})
			if fd.Level == "error" {
				hasError = true
			}
			total++
		}
		if hasError {
			filesWithErrors++
		}
	}

	table(out, []string{"File", "Level", "Finding"}, rows)
	if total > 0 {
		fmt.Fprintf(out, "%d finding(s) in %d file(s); %d file(s) with errors.\n",
			total, len(files), filesWithErrors)
	}
	return filesWithErrors
}

// caseTwins maps each deck file to the other deck files whose names differ
// from its own only by case, which drako's name matching cannot tell apart.
func caseTwins(files []string) map[string][]string {
	byNorm := map[string][]string{}
	for _, f := range files {
		if base := filepath.Base(f); base != paths.ConfigFileName {
			n := profiles.NormalizeName(base)
			byNorm[n] = append(byNorm[n], base)
		}
	}
	twins := map[string][]string{}
	for _, f := range files {
		base := filepath.Base(f)
		for _, other := range byNorm[profiles.NormalizeName(base)] {
			if other != base && !slices.Contains(twins[f], other) {
				twins[f] = append(twins[f], other)
			}
		}
	}
	return twins
}
