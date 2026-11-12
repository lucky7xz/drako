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
	only       []string // when set, the rule binds just these packages and allows none of them
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
		// Not tui: the screen code reaches the disk through lair and host.
		// summon owns its quarantine, multiplex its batch scripts.
		allowed: []string{"lair", "summon", "proc", "host", "multiplex", "cli", "app"},
	},
	{
		// lair reaches drako's files through its os.Root, so nothing can
		// leave the config directory. Opening a file by path would step
		// around that; the few lines that must are knownExceptions. cli
		// reaches drako's folder through lair, never by path.
		capability: "open a file by path instead of through the lair's root",
		calls: []string{
			"os.ReadFile", "os.WriteFile", "os.Open", "os.OpenFile", "os.Create", "os.CreateTemp",
			"os.Remove", "os.RemoveAll", "os.Rename", "os.Mkdir", "os.MkdirAll", "os.MkdirTemp",
			"os.ReadDir", "os.Stat", "os.Lstat", "os.Chmod", "os.Symlink", "os.Readlink", "os.Link",
			"os.Truncate", "os.DirFS", "github.com/BurntSushi/toml.DecodeFile",
		},
		only: []string{"lair", "cli"},
	},
	{
		// Only the shells talk to the user. Everything beneath them returns
		// values and errors, so what the user is told is decided in one layer.
		capability: "print to the user's terminal",
		calls:      []string{"fmt.Print", "fmt.Printf", "fmt.Println"},
		allowed:    []string{"cli", "app", "tui"},
	},
	{
		capability: "step around Go's memory safety",
		imports:    []string{"unsafe", "C"},
	},
}

// pure packages hold values and decisions only: none of them appears in a
// rule's allowed list, and of drako's own packages they may import only
// each other.
var pure = []string{"profile", "settings", "equip", "launch", "glyph"}

// effects are the packages that touch the world. Each is audited on its own,
// so of drako's packages they may import only pure ones, and proc: the one
// way to start a program.
var effects = []string{"lair", "proc", "summon", "host", "multiplex"}

// internal is the import path prefix of drako's own packages.
const internal = "github.com/lucky7xz/drako/internal/"

// knownExceptions is every place the code breaks a rule today, as
// "package/file.go what". The list may only shrink: a refactoring step that
// moves an effect to its home deletes the lines it has made untrue.
var knownExceptions = []string{
	// use the network

	// touch files

	// open a file by path instead of through the lair's root
	"lair/files.go os.ReadFile", // read: follows a file the user linked out of the folder
	"lair/files.go os.Stat",     // stat, read's twin; and explain, to recognise a linked subfolder
	"lair/lair.go os.MkdirAll",  // Create: the folder has to exist before it can be opened
	"lair/lair.go os.RemoveAll", // Destroy: a folder cannot remove itself through its own root

	// cli, bound since step 8a: only check, for files you name by path
	"cli/check.go os.ReadFile", // stays: a file you name by path, outside drako's folder
	"cli/check.go os.Stat",     // stays: same
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
		if slices.Contains(effects, pkg) {
			for _, imp := range imports {
				dep, ours := strings.CutPrefix(imp, internal)
				dep, _, _ = strings.Cut(dep, "/")
				if ours && dep != "proc" && !slices.Contains(pure, dep) {
					out = append(out, rel+" "+imp)
				}
			}
		}
		for _, r := range rules {
			if slices.Contains(r.allowed, pkg) || (r.only != nil && !slices.Contains(r.only, pkg)) {
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
