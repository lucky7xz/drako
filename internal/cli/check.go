package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/lucky7xz/drako/internal/equip"
	"github.com/lucky7xz/drako/internal/lair"
	"github.com/lucky7xz/drako/internal/launch"
	"github.com/lucky7xz/drako/internal/profile"
	"github.com/lucky7xz/drako/internal/settings"
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
// arguments means config.toml, the equipped dir and the inventory, as lair
// lists them.
func checkTargets(args []string) ([]string, error) {
	if len(args) == 0 {
		configDir, err := lair.ConfigDir()
		if err != nil {
			return nil, fmt.Errorf("could not get config dir: %v", err)
		}
		names, err := lair.DeckFiles(configDir)
		if err != nil {
			return nil, err
		}
		var files []string
		for _, n := range names {
			files = append(files, filepath.Join(configDir, n))
		}
		sort.Strings(files)
		return files, nil
	}

	var files []string
	for _, a := range args {
		info, err := os.Stat(a)
		switch {
		case err != nil:
			return nil, fmt.Errorf("cannot read %s: %v", a, err)
		case info.IsDir():
			matches, _ := filepath.Glob(filepath.Join(a, "*"+equip.ProfileSuffix))
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
	clashes := nameClashes(files)
	themes := knownThemes()
	for _, f := range files {
		src, err := readForCheck(f)
		var findings []profile.Finding
		switch {
		case err != nil:
			findings = []profile.Finding{{Level: "error", Msg: fmt.Sprintf("cannot read file: %v", err)}}
		case filepath.Base(f) == lair.ConfigFileName:
			findings = settings.Lint(src)
			if base, _, err := settings.Decode(src); err == nil {
				findings = append(findings, nameFindings("default_shell", base.DefaultShell, "theme", base.Theme, themes)...)
			}
		default:
			findings = profile.Lint(src)
			if pf, _, err := profile.Decode(src, nil); err == nil {
				shell := ""
				if pf.Shell != nil {
					shell = *pf.Shell
				}
				findings = append(findings, nameFindings("shell", shell, "theme", pf.Theme, themes)...)
			}
		}
		if err := lair.CheckFileName(filepath.Base(f)); err != nil {
			findings = append(findings, profile.Finding{Level: "error", Msg: err.Error()})
		}
		for _, msg := range clashes[f] {
			findings = append(findings, profile.Finding{Level: "warning", Msg: msg})
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

// knownThemes is the theme names drako can draw: the built-in ones and those
// in the user's themes.toml.
func knownThemes() []string {
	dir, _ := lair.ConfigDir() // "" leaves the built-in themes
	return lair.ThemeNames(dir)
}

// nameFindings warns about a shell or theme setting that names nothing drako
// knows. Both fall back silently when drako runs (I-05): a typo in the shell
// runs every cell in another shell, and an unknown theme draws as dracula.
func nameFindings(shellKey, shell, themeKey, theme string, themes []string) []profile.Finding {
	var fs []profile.Finding
	warn := func(format string, args ...any) {
		fs = append(fs, profile.Finding{Level: "warning", Msg: fmt.Sprintf(format, args...)})
	}
	if s := strings.TrimSpace(shell); s != "" && !launch.KnownShell(s) {
		warn("%s = %q is not a shell drako knows (%s); cells run in %s instead",
			shellKey, s, strings.Join(launch.ShellNames(), ", "), launch.FallbackShell())
	}
	if t := strings.TrimSpace(theme); t != "" && !slices.Contains(themes, t) {
		warn("%s = %q is not a theme drako has; it draws as dracula (themes: %s)", themeKey, t, strings.Join(themes, ", "))
	}
	return fs
}

// readForCheck reads a file to lint. One in drako's folder is read through
// lair, the way the loader reads it; one named by path elsewhere, as it is.
func readForCheck(path string) ([]byte, error) {
	if dir, err := lair.ConfigDir(); err == nil {
		if rel, err := filepath.Rel(dir, path); err == nil && filepath.IsLocal(rel) {
			return lair.ReadFile(dir, rel)
		}
	}
	return os.ReadFile(path)
}

// nameClashes warns about deck files drako cannot tell apart: the same name
// in two places, or names that differ only by case.
func nameClashes(files []string) map[string][]string {
	byNorm := map[string][]string{}
	for _, f := range files {
		if base := filepath.Base(f); base != lair.ConfigFileName {
			n := equip.NormalizeName(base)
			byNorm[n] = append(byNorm[n], f)
		}
	}
	clashes := map[string][]string{}
	for _, f := range files {
		base := filepath.Base(f)
		for _, other := range byNorm[equip.NormalizeName(base)] {
			var msg string
			switch {
			case other == f:
				continue
			case filepath.Base(other) == base:
				msg = "also in " + filepath.Base(filepath.Dir(other)) + "/; remove one copy"
			default:
				msg = "clashes with " + filepath.Base(other) + ": drako ignores case in deck names"
			}
			if !slices.Contains(clashes[f], msg) {
				clashes[f] = append(clashes[f], msg)
			}
		}
	}
	return clashes
}
