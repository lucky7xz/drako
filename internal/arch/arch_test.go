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

// pure packages hold values and decisions only: none of them appears in a
// rule's allowed list, and of drako's own packages they may import only
// each other.
var pure = []string{"profile", "settings", "equip", "launch"}

// internal is the import path prefix of drako's own packages.
const internal = "github.com/lucky7xz/drako/internal/"

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

	// a pure package importing one that is not
	"launch/plan.go github.com/lucky7xz/drako/internal/config",
	"launch/selection.go github.com/lucky7xz/drako/internal/config",
}

func TestPackagesKeepToTheirCapabilities(t *testing.T) {
	found := violations(t, "..")

	for _, v := range found {
		if !slices.Contains(knownExceptions, v) {
			t.Errorf("new violation: %q\n\tThe rules at the top of this file say which package this belongs in.", v)
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

		imports, calls := uses(t, path)
		if slices.Contains(pure, pkg) {
			for _, imp := range imports {
				dep, ours := strings.CutPrefix(imp, internal)
				dep, _, _ = strings.Cut(dep, "/")
				if ours && !slices.Contains(pure, dep) {
					out = append(out, rel+" "+imp)
				}
			}
		}
		for _, r := range rules {
			if slices.Contains(r.allowed, pkg) {
				continue
			}
			for _, imp := range imports {
				if slices.Contains(r.imports, imp) {
					out = append(out, rel+" "+imp)
				}
			}
			for _, call := range calls {
				if slices.Contains(r.calls, call) {
					out = append(out, rel+" "+call)
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

// uses is what one file reaches for: its import paths, and every
// "importpath.Name" it mentions. Build tags are ignored on purpose, so files
// for other operating systems are held to the same rules.
func uses(t *testing.T, path string) (imports, calls []string) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}

	local := map[string]string{} // name in this file -> import path
	for _, imp := range file.Imports {
		importPath, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			t.Fatal(err)
		}
		imports = append(imports, importPath)
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
			calls = append(calls, local[x.Name]+"."+sel.Sel.Name)
		}
		return true
	})
	return imports, calls
}
