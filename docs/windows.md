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

