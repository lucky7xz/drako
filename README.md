<div align="center">

# drako

**The customizable Command-Deck Launcher.**

Every command you rely on, one keypress away, on a grid in your terminal.

[![Docs](https://img.shields.io/badge/docs-chronyx.xyz-ff5555)](https://chronyx.xyz/docs/drako/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/lucky7xz/drako?color=007D9C&label=version)](https://github.com/lucky7xz/drako/tags)
[![Coverage](https://img.shields.io/coverallsCoverage/github/lucky7xz/drako?branch=dev)](https://coveralls.io/github/lucky7xz/drako?branch=dev)
[![License](https://img.shields.io/github/license/lucky7xz/drako?color=orange)](https://github.com/lucky7xz/drako/blob/main/LICENCE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/lucky7xz/drako?color=00ADD8&logo=go&logoColor=white)](https://go.dev/)

**[Docs](https://chronyx.xyz/docs/drako/)** · [English](README.md) · [简体中文](README.zh-CN.md)

<img width="100%" alt="drako batch launch in tmux" src="https://github.com/user-attachments/assets/274abaed-1cd8-4c35-8b96-4b37b85556e6" />

</div>

- **Harness, don't replace.** If it runs in a terminal, it can be bound to a cell: tools, TUIs, scripts.
- **Look before you run.** `e` shows the exact command behind a cell. Nothing runs until you press Enter.
- **Decks are plain text.** A profile is one TOML file: write your own, keep it in Git, share it.
- **Summon.** `drako summon <repo>` pulls a deck from any Git host; `--show` lists every command first, `--rev`/`--sha256` verify it.
- **One cell, every OS.** A command can differ per platform (apt, pacman, brew, …) and drako picks the right one.
- **Batch launch.** Mark cells and launch them side by side in tmux or herdr.
- **Scripts and AI agents.** `drako ls`, `explain` and `run` let a script or an agent use the commands you curated instead of making them up.
- **No surprises.** One static binary, no telemetry, no update check. drako uses the network only when you summon.

Full docs: **[chronyx.xyz/docs/drako](https://chronyx.xyz/docs/drako/)**

https://github.com/user-attachments/assets/9be2906d-2389-4e4e-98c8-cbded653ff79

## Install

Linux, macOS, FreeBSD:

```bash
curl -fsSL https://github.com/lucky7xz/drako/releases/latest/download/install.sh | sh
```

It downloads the release for your system, checks its checksum and puts `drako` in `~/.local/bin`. No root needed.

With Go 1.25+:

```bash
go install github.com/lucky7xz/drako@latest
```

Living on the edge? `@dev` installs the newest changes from the `dev` branch: newer, less settled.

```bash
go install github.com/lucky7xz/drako@dev
```

Then start it. The first time, by its full path, since `~/go/bin` is usually not on your `PATH` yet:

```bash
~/go/bin/drako
```

To fix that for good, run **Add drako to PATH** from the **🐉 Settings** cell in the Core profile. It adds `~/.local/bin` and `~/go/bin` to `~/.bashrc` (`~/.zshrc` on macOS, your PowerShell profile on Windows). Open a new shell and plain `drako` works.

Binaries and Linux packages (.deb, .rpm, .apk, Arch) are on the [Releases](https://github.com/lucky7xz/drako/releases) page. [Install](https://chronyx.xyz/docs/drako/install/) covers all of these, Termux included.

**Emoji:** profiles use emoji as visual markers. Modern terminals (Ghostty, WezTerm, …) show them out of the box; older ones may need a [Nerd Font](https://www.nerdfonts.com/) or an emoji font package such as `fonts-noto-color-emoji`.

drako is Linux-first. macOS and FreeBSD work and are less tested. Windows is not supported yet: `go install` builds it, but expect rough edges.

## Optional dependencies

drako itself is one binary. These are looked up when a feature needs them; a missing one costs you that feature, never a crash.

| Tool | Used for | Without it |
| --- | --- | --- |
| `bash` | running cells (`bash -lc` is the default; `powershell` on Windows) | cells fail; set `default_shell = "sh"` or `"zsh"`, which the shipped profiles also run under |
| `git` | summoning a deck from a repository | summoning a single file or a local folder still works |
| `tmux` or `herdr` | batch launch | batch launch tells you what is missing |
| `wl-copy`, `xclip` or `xsel` | copying a command (`y`) | drako falls back to OSC 52, which many terminals understand |
| `xdg-open` | `drako open`, opening paths | that action fails |
| `$VISUAL` / `$EDITOR` | editing a profile from the inventory (`e`) | drako tries `nano`, `vim`, `vi` |

Decks bring their own tools (`btop`, `mpv`, `distrobox`, …). Core's **📦 Install Tools** cell installs the common ones.

## Try it live

```bash
ssh chronyx.xyz
```

drako served over SSH in Glassroot Mode, which locks the interface down for guests. Experimental and shared: treat it as a public terminal. The deck behind it is [groot demo](https://github.com/lucky7xz/groot_demo).

## Decks

A profile is one grid you load into drako. A deck is what you share: a folder or repo of profiles, plus the scripts and tools they bring. (Decks and profiles can be used interchangeably at times.)

**Shipped with drako.** Core is equipped on first launch; the rest wait in the inventory (`i`).

| Profile | What you get |
| --- | --- |
| **Core** | System update for your distro, cleanup and virus scan, process monitor, file finder, bandwidth, system status, a tool installer, and settings |
| **ssh-101** | Run and secure an SSH server: service, firewall, config and logs, keys, key-only login, transfers, tunnels |
| **tmux-101** | Sessions, panes and copy mode: the tmux basics, and the engine batch launch runs on |
| **jukebox** | Terminal radio: SomaFM, world news, hand-picked stations, offline noise (mpv), and your own music (cliamp) |
| **distrobox** | Boxes running other distros: create, enter, stop, upgrade, export and remove them, or start a throwaway one that is removed on exit. Boxes share your home folder: what you delete inside is gone outside too (!) |

https://github.com/user-attachments/assets/7fa86d77-3345-4103-be3e-26b545a11e4d

*One Core, two distros: the same cells run `pacman` in an Arch box and `dnf` in a Fedora box.*

**Summon more.** `drako summon <repo>`, or from Core's **📦 Install Tools** cell.

| Deck | What you get |
| --- | --- |
| [**101 Series**](https://github.com/lucky7xz/101-deck) | Four starter profiles: docker, git, network and unix |
| [**GGML**](https://github.com/lucky7xz/ggml-deck) | Local AI with llama.cpp and whisper.cpp: setup, models, web UI, transcription |
| [**groot demo**](https://github.com/lucky7xz/groot_demo) | The deck behind `ssh chronyx.xyz` |

Your own: a folder of `.profile.toml` files in Git, summoned onto any machine.

## Safety

A deck is a set of commands that will run on your machine. Treat a summoned deck like an executable: only download and run what you trust! Read it first (`--show`). Unsure what a command does? Ask someone who knows before you run it.

## Agent instructions

`drako agents` prints a short brief that teaches an AI agent to drive your decks. Add it to the agent's instructions file. Commands are logged, but not sandboxed. Make sure you understand what is equipped.

```bash
drako agents >> AGENTS.md
```

## Contributing

Ideas and bug reports are welcome as issues. Pull requests are not accepted for now; see [CONTRIBUTING](docs/CONTRIBUTING.md). drako is in beta.

## Development

```bash
git clone https://github.com/lucky7xz/drako && cd drako
go vet ./... && go test ./...
```

drako is developed with heavy AI assistance. The codebase is kept small and modular so one person can audit it end to end.

## Built with

- [`charmbracelet/bubbletea`](https://github.com/charmbracelet/bubbletea): the TUI
- [`charmbracelet/lipgloss`](https://github.com/charmbracelet/lipgloss): styling
- [`BurntSushi/toml`](https://github.com/BurntSushi/toml): TOML
- [`shirou/gopsutil`](https://github.com/shirou/gopsutil): system info
- [`fsnotify/fsnotify`](https://github.com/fsnotify/fsnotify): file watching
- [`yuin/gopher-lua`](https://github.com/yuin/gopher-lua): Lua
- [`golang.org/x/term`](https://pkg.go.dev/golang.org/x/term): terminal handling

## License

The engine is [AGPL-3.0](LICENCE). The bootstrap profiles are MIT.

<div align="center">

Tame the chaos.

</div>
