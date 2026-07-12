# Windows Support

> drako is Linux-first — that's where it lives and gets tested daily. Windows support
> is designed in (native paths, PowerShell execution, per-platform deck variants) but
> has not been tested on real hardware yet. Treat this page as a map of intent, and
> please report what you find.

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
- **Shell Integration:** Commands run via `powershell.exe` by default. Users with PowerShell 7 can set `default_shell = "pwsh"` in `config.toml`.
- **Cross-Platform Decks:** Profile authors can give a cell a `windows` command variant alongside the Linux/macOS ones — see "Cross-Platform Decks" in the README. The generated Core profile already uses Windows-native commands.

## Scoop

Scoop commands (`scoop status`, `scoop update`, `scoop cleanup`, …) bind to grid cells like any other CLI tool — a Scoop deck makes a great first custom profile.

## Troubleshooting

### Colors looking wrong?
Make sure you are using a modern terminal like **Windows Terminal**. The legacy `conhost.exe` (classic cmd window) has limited color support.

### Clipboard
Drako attempts to use the system clipboard. If you have issues, ensure you are not running in a restricted environment.

## Status — Audit & Fix, July 2026

A July 2026 audit found Windows support broken at the executor level: the default shell
resolved to `pwsh` (PowerShell 7, not preinstalled on Windows — only `powershell.exe`
5.1 ships in the box), so every cell failed at exec. On top of that, the core profile's
windows variants mixed PowerShell and cmd syntax (`%APPDATA%` is a cmd-ism PowerShell
won't expand), referenced a nonexistent `drako internal open-url` subcommand, and listed
winget packages that don't exist.

**Fixed since:** the executor now runs `powershell.exe` for `default_shell =
"powershell"` and uses it as the Windows fallback (`pwsh` remains available as an
explicit setting); all core-profile windows variants were rewritten to PowerShell
5.1-safe syntax with real winget IDs (`junegunn.fzf`, `aristocratos.btop4win`,
`muesli.duf`, `Fastfetch-cli.Fastfetch`, `zyedidia.micro`); the post-command screen
clear uses `cls` on Windows.

**Known gaps:**

- **Batch launch** drives tmux and is not available on Windows — drako shows a friendly
  message instead of a raw exec error.
- **The ssh-utils inventory profile is POSIX-only** (no windows variants; `read`,
  `systemctl`, `ufw` throughout).
- **All of this is desk-checked, not tested on real Windows hardware.** Reports welcome.

