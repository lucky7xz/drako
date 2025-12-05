# Windows Support

## Installation

### Via Go (Recommended)
If you have Go installed:

```powershell
go install github.com/lucky7xz/drako@latest
```

### Via Scoop (Coming Soon)
We plan to add a Scoop bucket for easy installation:
```powershell
# Future command
scoop bucket add drako https://github.com/lucky7xz/drako-bucket
scoop install drako
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
- **Scoop Integration:** Use the bundled `scoop` profile to manage your packages from a TUI.

## Scoop Profile

We include a "Starter Deck" for Scoop users. You can find it in your Inventory (`i`).

It includes commands for:
- Checking updates (`scoop status`, `scoop update`)
- Searching and installing apps
- Cleaning up old versions (`scoop cleanup`)

## Troubleshooting

### Colors looking wrong?
Make sure you are using a modern terminal like **Windows Terminal**. The legacy `conhost.exe` (classic cmd window) has limited color support.

### Clipboard
Drako attempts to use the system clipboard. If you have issues, ensure you are not running in a restricted environment.

