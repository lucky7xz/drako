# Windows Support

## Installation

### Via Go (Recommended)
If you have Go installed:

```powershell
go install github.com/lucky7xz/drako@latest
```

## Configuration

On Windows, your configuration lives in the standard AppData location:

```
%APPDATA%\drako\config.toml
```

(Usually `C:\Users\YourName\AppData\Roaming\drako\config.toml`)

## Features

- **Native Path Handling:** Drako understands drive letters (`C:\`) and backslashes.
- **Shell Integration:** Commands run via your default shell (PowerShell by default if detected, or configured in `config.toml`).
- **Cross-Platform Decks:** Profile authors can give a cell a `windows` command variant alongside the Linux/macOS ones — see "Cross-Platform Decks" in the README. The generated Core profile already uses Windows-native commands.

## Scoop

Scoop commands (`scoop status`, `scoop update`, `scoop cleanup`, …) bind to grid cells like any other CLI tool — a Scoop deck makes a great first custom profile.

## Troubleshooting

### Colors looking wrong?
Make sure you are using a modern terminal like **Windows Terminal**. The legacy `conhost.exe` (classic cmd window) has limited color support.

### Clipboard
Drako attempts to use the system clipboard. If you have issues, ensure you are not running in a restricted environment.

## Known Broken — Audit 2026-07

Windows support has never been tested end-to-end. A code audit (July 2026) found it is
currently broken at the executor level, before any profile TOML matters:

- **Executor default shell is bash on every platform.** `buildShellCmd()`
  (`internal/core/commands.go`) has no GOOS awareness; with `default_shell` unset (the
  shipped template comments it out), every cell runs via `bash -lc`, which does not exist
  on stock Windows. Setting `default_shell = "powershell"` doesn't rescue it either: that
  case execs `pwsh` (PowerShell 7, not preinstalled) — stock Windows only ships
  `powershell.exe` 5.1.
- **Core profile windows variants mix two shells.** Some are PowerShell-only (`gci`,
  `Where-Object`, `Out-GridView`, `$PROFILE`, `Read-Host`, `Add-Content`), others are
  cmd-style (`%APPDATA%\drako` — PowerShell won't expand that, and neither does
  `drako open`). No single shell choice runs the current set.
- **Per-command issues:** `curl wttr.in` hits the PowerShell 5.1 `Invoke-WebRequest`
  alias (needs `curl.exe`); `winget install` with five package names in one call is
  dubious; `btop`/`bmon`/`neofetch` aren't real winget package IDs; Defender's
  `MpCmdRun.exe` path varies by version (the stable interface is the `Start-MpScan`
  cmdlet); the "Reload Shell" windows variant is an empty string.

**Fix plan (future project):** make the shell default GOOS-aware (windows →
`powershell.exe`), then rewrite all windows variants to consistent PowerShell 5.1-safe
syntax. Needs a Windows test machine.

