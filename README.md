# drako

[English](README.md) · [简体中文](README.zh-CN.md)

[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/lucky7xz/drako?color=007D9C&label=version)](https://github.com/lucky7xz/drako/tags)
[![License](https://img.shields.io/github/license/lucky7xz/drako?color=orange)](https://github.com/lucky7xz/drako/blob/main/LICENCE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/lucky7xz/drako?color=00ADD8&logo=go&logoColor=white)](https://go.dev/)

https://github.com/user-attachments/assets/21fb2340-bc74-4886-a629-8e95d116e830

**The customizable Command-Deck Launcher.**

Every command you rely on, one keypress away, on a grid in your terminal.

- **Harness, don't replace.** If it runs in a terminal, it can be bound to a cell: tools, TUIs, scripts.
- **Look before you run.** `e` shows the exact command behind a cell. Nothing runs until you press Enter.
- **Decks are plain text.** A profile is one TOML file: write your own, keep it in Git, share it.
- **Summon.** `drako summon <repo>` pulls a deck from any Git host; `--show` lists every command first, `--rev`/`--sha256` verify it.
- **One cell, every OS.** A command can differ per platform (apt, pacman, brew, …) and drako picks the right one.
- **Batch launch.** Mark cells and launch them side by side in tmux or herdr.
- **Scripts and AI agents.** `drako ls`, `explain` and `run` let a script or an agent use the commands you curated instead of making them up.
- **No surprises.** One static binary, no telemetry, no update check. drako uses the network only when you summon.

Full docs: **[chronyx.xyz/docs/drako](https://chronyx.xyz/docs/drako/)**

## install

```bash
go install github.com/lucky7xz/drako@latest
```

Living on the edge? `@dev` installs the newest changes from the `dev` branch: newer, less settled.

```bash
go install github.com/lucky7xz/drako@dev
```

Requires Go 1.25+. Common ways to get it:

| Platform | Command |
| --- | --- |
| Arch | `sudo pacman -S go` |
| Fedora | `sudo dnf install golang` |
| macOS | `brew install go` |
| Windows | `winget install GoLang.Go` |
| Debian, Ubuntu, others | [go.dev/dl](https://go.dev/dl/) (distro packages are often older than 1.25) |

Then start it. The first time, by its full path, since `~/go/bin` is usually not on your `PATH` yet:

```bash
~/go/bin/drako
```

To fix that for good, run **Add go/bin to PATH** from the **🐉 Settings** cell in the Core profile. It adds `~/go/bin` to `~/.bashrc` (`~/.zshrc` on macOS, your PowerShell profile on Windows). Open a new shell and plain `drako` works.

No Go? Prebuilt static binaries for Linux, macOS and FreeBSD are on the [Releases](https://github.com/lucky7xz/drako/releases) page, and so are Linux packages. Install the one you downloaded:

| Distro | Command |
| --- | --- |
| Debian, Ubuntu | `sudo apt install ./drako_*.deb` |
| Fedora, RHEL | `sudo dnf install ./drako_*.rpm` |
| openSUSE | `sudo zypper install --allow-unsigned-rpm ./drako_*.rpm` |
| Arch | `sudo pacman -U ./drako_*.pkg.tar.zst` |
| Alpine | `doas apk add --allow-untrusted ./drako_*.apk` |

Arrows move, `Enter` runs, `e` explains, `?` shows every key, `q` quits.

**Termux (Android):** build without cgo, since Termux ships no C toolchain by default:

```bash
CGO_ENABLED=0 go install github.com/lucky7xz/drako@latest
```

Termux has no `sudo`, so run the **🚨 Sudo Switch** (Core profile, **Settings** cell) once after first launch. A physical keyboard is recommended but not required.

**Emoji:** decks use emoji as visual markers. Modern terminals (Ghostty, WezTerm, …) show them out of the box; older ones may need a [Nerd Font](https://www.nerdfonts.com/) or an emoji font package such as `fonts-noto-color-emoji`.

drako is Linux-first. macOS and Windows are supported by design and less tested.

## optional dependencies

drako itself is one binary. These are looked up when a feature needs them; a missing one costs you that feature, never a crash.

| Tool | Used for | Without it |
| --- | --- | --- |
| `bash` | running cells (`bash -lc` is the default shell) | cells fail, unless you set `default_shell` (`sh`, `zsh`, `fish`, `pwsh`, `cmd`) |
| `git` | summoning a deck from a repository | summoning a single file or a local folder still works |
| `tmux` or `herdr` | batch launch | batch launch tells you what is missing |
| `wl-copy`, `xclip` or `xsel` | copying a command (`y`) | drako falls back to OSC 52, which many terminals understand |
| `xdg-open` | `drako open`, opening paths | that action fails |
| `$VISUAL` / `$EDITOR` | editing a deck from the inventory (`e`) | drako tries `nano`, `vim`, `vi` |

Decks bring their own tools (`btop`, `mpv`, `distrobox`, …). Core's **📦 Install Tools** cell installs the common ones.

## try it live

```bash
ssh chronyx.xyz
```

drako served over SSH in Glassroot Mode, which locks the interface down for guests. Experimental and shared: treat it as a public terminal. The deck behind it is [groot demo](https://github.com/lucky7xz/groot_demo).

## decks

A deck is a runnable cheat sheet: every cell carries a command and a note on what it does, so the know-how travels with the keys.

**Shipped with drako.** Core is equipped on first launch; the rest wait in the inventory (`i`).

| Deck | What you get |
| --- | --- |
| **Core** | System update for your distro, cleanup and virus scan, process monitor, file finder, bandwidth, system status, a tool installer, and settings |
| **ssh-101** | Run and secure an SSH server: service, firewall, config and logs, keys, key-only login, transfers, tunnels |
| **tmux-101** | Sessions, panes and copy mode: the tmux basics, and the engine batch launch runs on |
| **jukebox** | Terminal radio: SomaFM, world news, hand-picked stations, offline noise (mpv), and your own music (cliamp) |
| **distrobox** | Create, enter, export and remove containers; throwaway boxes for testing something risky |

**Summon more.** `drako summon <repo>`, or from Core's **📦 Install Tools** cell.

| Deck | What you get |
| --- | --- |
| [**101 Series**](https://github.com/lucky7xz/101-deck) | Four starter decks: docker, git, network and unix |
| [**GGML**](https://github.com/lucky7xz/ggml-deck) | Local AI with llama.cpp and whisper.cpp: setup, models, web UI, transcription |
| [**groot demo**](https://github.com/lucky7xz/groot_demo) | The deck behind `ssh chronyx.xyz` |

Your own: a folder of `.profile.toml` files in Git, summoned onto any machine.

## safety

A deck is a set of commands that will run on your machine. Treat a summoned deck like an executable: read it first (`--show`, then `e`), take decks only from sources you trust, and pin what you pull.

## agent instructions

`drako agents` prints a short brief that teaches an AI agent to drive your decks. Add it to the agent's instructions file:

```bash
drako agents >> AGENTS.md
```

## contributing

Ideas and bug reports are welcome as issues. Pull requests are not accepted for now; see [CONTRIBUTING](docs/CONTRIBUTING.md). drako is in early beta.

## development

```bash
git clone https://github.com/lucky7xz/drako && cd drako
go vet ./... && go test ./...
```

drako is developed with heavy AI assistance. The codebase is kept small and modular so one person can audit it end to end.

## built with

- [`charmbracelet/bubbletea`](https://github.com/charmbracelet/bubbletea): the TUI
- [`charmbracelet/lipgloss`](https://github.com/charmbracelet/lipgloss): styling
- [`BurntSushi/toml`](https://github.com/BurntSushi/toml): TOML
- [`shirou/gopsutil`](https://github.com/shirou/gopsutil): system info
- [`fsnotify/fsnotify`](https://github.com/fsnotify/fsnotify): file watching
- [`yuin/gopher-lua`](https://github.com/yuin/gopher-lua): Lua
- [`golang.org/x/term`](https://pkg.go.dev/golang.org/x/term): terminal handling

## license

The engine is [AGPL-3.0](LICENCE). The bootstrap decks are MIT.

<div align="center">

Tame the chaos.

</div>
