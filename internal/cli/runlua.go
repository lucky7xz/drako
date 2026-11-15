package cli

// drako run-lua runs a Lua script with the interpreter built into drako
// (GopherLua, Lua 5.1), so a script runs wherever drako does. The interpreter
// is stock: its standard libraries are open and drako adds nothing to them. A
// script has the user's full rights, like any command.

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

// HandleRunLuaCommand processes 'drako run-lua <script.lua> [args...]'. The
// path is taken as written. Like lua(1) it exits 0 when the script ends, 1
// when it fails or cannot be read; os.exit(n) in the script exits with n.
func HandleRunLuaCommand(args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "drako run-lua: no script given. Usage: drako run-lua <script.lua> [args...]")
		return 1
	}
	script, rest := args[2], args[3:]

	L := lua.NewState()
	defer L.Close()

	// The script's arguments, as lua(1) hands them over: the arg table
	// (arg[0] is the script) and the chunk's own "...".
	argTable := L.NewTable()
	argTable.RawSetInt(0, lua.LString(script))
	for i, a := range rest {
		argTable.RawSetInt(i+1, lua.LString(a))
	}
	L.SetGlobal("arg", argTable)

	chunk, err := L.LoadFile(script)
	if err != nil {
		fmt.Fprintf(os.Stderr, "drako run-lua: %s\n", loadErrorText(script, err))
		return 1
	}
	L.Push(chunk)
	for _, a := range rest {
		L.Push(lua.LString(a))
	}
	if err := L.PCall(len(rest), 0, nil); err != nil {
		fmt.Fprintf(os.Stderr, "drako run-lua: %s\n", strings.TrimSpace(err.Error()))
		return 1
	}
	return 0
}

// loadErrorText says why script did not load: that it cannot be read, with
// the system's reason, or the interpreter's own words for a syntax error.
func loadErrorText(script string, err error) string {
	var apiErr *lua.ApiError
	var pathErr *fs.PathError
	if errors.As(err, &apiErr) && errors.As(apiErr.Cause, &pathErr) {
		return fmt.Sprintf("cannot read %s: %v", script, pathErr.Err)
	}
	return strings.TrimSpace(err.Error())
}
