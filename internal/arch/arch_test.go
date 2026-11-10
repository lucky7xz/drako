package arch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// A rule ties one thing code can do to the world to the packages allowed to
// do it. Importing any of imports, or calling any of calls, counts as doing it.
type rule struct {
	capability string
	imports    []string
	calls      []string // "importpath.Func", for effects hiding in a broad package
	allowed    []string // packages directly under internal/
}

var rules = []rule{
	{
		capability: "start processes",
		imports:    []string{"os/exec", "syscall"},
		calls:      []string{"os.StartProcess"},
		allowed:    []string{"proc"},
	},
	{
		capability: "use the network",
		imports:    []string{"net", "net/http"},
		allowed:    []string{"summon", "host"},
	},
	{
		capability: "touch files",
		imports:    []string{"os"},
		// ui is today's name for tui.
		allowed: []string{"lair", "summon", "proc", "host", "tui", "ui", "cli", "app"},
	},
	{
		capability: "step around Go's memory safety",
		imports:    []string{"unsafe", "C"},
	},
}

// knownExceptions is every place the code breaks a rule today, as
// "package/file.go what". The list may only shrink: a refactoring step that
// moves an effect to its home deletes the lines it has made untrue.
var knownExceptions = []string{
	// start processes
	"app/app.go os/exec",
	"cli/open.go os/exec",
	"cli/summon_transport.go os/exec",
	"core/cmdline_other.go os/exec",
	"core/cmdline_windows.go os/exec",
	"core/cmdline_windows.go syscall",
	"core/commands.go os/exec",
	"core/commands.go syscall",
	"core/execplan.go os/exec",
	"core/runforexit.go os/exec",
	"core/runforexit.go syscall",
	"multiplex/herdr.go os/exec",
	"multiplex/resolve.go os/exec",
	"multiplex/tmux.go os/exec",
	"ui/clipboard.go os/exec",
	"ui/editor.go os/exec",

	// use the network
	"cli/summon_transport.go net/http",
	"core/network.go net",

	// touch files
	"config/bootstrap.go os",
	"config/discover.go os",
	"config/load.go os",
	"config/pivot.go os",
	"config/platform.go os",
	"config/themes.go os",
	"core/commands.go os",
	"core/logging.go os",
	"core/runforexit.go os",
	"multiplex/backend.go os",
	"multiplex/herdr.go os",
	"multiplex/resolve.go os",
	"multiplex/tmux.go os",
	"paths/paths.go os",
	"profiles/move.go os",
	"profiles/profiles.go os",
	"profiles/trash.go os",
}

func TestPackagesKeepToTheirCapabilities(t *testing.T) {
	found := violations(t, "..")

	for _, v := range found {
		if !slices.Contains(knownExceptions, v) {
			t.Errorf("new violation: %q\n\tThis effect belongs in one of the packages its rule allows; move it there.", v)
		}
	}
	for _, e := range knownExceptions {
		if !slices.Contains(found, e) {
			t.Errorf("stale exception: %q\n\tThe code no longer does this. Delete the line from knownExceptions.", e)
		}
	}
	t.Logf("%d known exceptions left", len(knownExceptions))
}

// violations lists every rule broken by a non-test Go file under root, sorted.
func violations(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		pkg, _, _ := strings.Cut(rel, "/")

		for _, use := range uses(t, path) {
			for _, r := range rules {
				if !slices.Contains(r.imports, use) && !slices.Contains(r.calls, use) {
					continue
				}
				if !slices.Contains(r.allowed, pkg) {
					out = append(out, rel+" "+use)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// uses is what one file reaches for: each import path, and each
// "importpath.Name" it mentions. Build tags are ignored on purpose, so
// files for other operating systems are held to the same rules.
func uses(t *testing.T, path string) []string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}

	var out []string
	local := map[string]string{} // name in this file -> import path
	for _, imp := range file.Imports {
		importPath, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, importPath)
		name := importPath[strings.LastIndex(importPath, "/")+1:]
		if imp.Name != nil {
			name = imp.Name.Name
		}
		local[name] = importPath
	}
	ast.Inspect(file, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if x, ok := sel.X.(*ast.Ident); ok && local[x.Name] != "" {
			out = append(out, local[x.Name]+"."+sel.Sel.Name)
		}
		return true
	})
	return out
}
