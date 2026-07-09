package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/lucky7xz/drako/internal/config"
	"github.com/lucky7xz/drako/internal/paths"
	"github.com/lucky7xz/drako/internal/profiles"
)

// HandleCheckCommand processes 'drako check [path ...]': lint profile files
// for authoring mistakes. With no arguments it checks every equipped and
// inventory profile. Returns exit code 1 when any error-level finding
// exists, so deck repositories can run it in CI.
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
// arguments means the equipped dir + inventory.
func checkTargets(args []string) ([]string, error) {
	if len(args) == 0 {
		configDir, err := paths.ConfigDir()
		if err != nil {
			return nil, fmt.Errorf("could not get config dir: %v", err)
		}
		args = []string{configDir, paths.InventoryDir(configDir)}
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
	for _, f := range files {
		src, err := os.ReadFile(f)
		var findings []config.Finding
		if err != nil {
			findings = []config.Finding{{Level: "error", Msg: fmt.Sprintf("cannot read file: %v", err)}}
		} else {
			findings = config.CheckProfile(src)
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
