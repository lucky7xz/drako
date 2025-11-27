![Demo v0.1.8](docs/demo.gif)

The terminal is a realm of immense power, but also of high entropy. Commands are forgotten, workflows fracture, and focus is lost to the noise. **Drako** is a **TUI-Deck launcher** that enables structure, transforming your terminal into a disciplined, grid-based command center. 


## 🚀 Quick Start

> Requires Go **1.24** or newer.

If Go is installed on your system, installing `drako` is a single command.

```bash
go install github.com/lucky7xz/drako@latest 
```

### Installing Go

- macOS: `brew install go`
- Arch: `sudo pacman -S go`
- Debian/Ubuntu: `sudo apt install golang`
- Windows is not **yet** supported.


Run `drako`. On its first execution, it will construct your configuration file at `~/.config/drako/config.toml`. This is the foundation. Modify it to begin bending your workflow into shape. We also provide a handful of profiles by default, to give you some inspiration. 

### Update

To update `drako` to the latest version, simply run the installation command again.

If you are not getting the latest version, use this command instead:

```bash
GOPROXY=direct go install github.com/lucky/drako@latest
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



## ✨ Philosophy

`drako` is built on a few core principles:

<<<<<<< HEAD
-   **The Grid is Your Command Deck.** Your most vital commands are laid out on a visual grid for immediate, single-keypress access. No more searching shell history or forgotten aliases.

-   **Profiles are Shifting Forms.** A profile is a complete reconfiguration of the grid for a different context. Switch from a "DevOps" (`docker`, `ufw`) to a "Network Sentinel" deck (`nmap`, `mtr`) instantly.

-   **Your Deck is Portable.** The true power of profiles is their portability. By keeping your `profile` file directory in a Git repository, you can deploy your entire command center to a new server with a single command. This transforms `drako` into a declarative, repeatable control panel for any machine you manage.

-   **Harness, Don't Replace.** `drako` integrates with the tools you already use. If it runs in the terminal, it can be bound to the grid.

-   **The Power of TUI Decks:** For those who wish to build true terminal cathedrals, `drako` serves as the gateway to `para13`, a `TUI-Deck` build with seamless integration into `drako`. More will be revealed in time. Stay tuned


=======
-   **The Grid is Your Command Deck:** Commands are mapped to a visual grid for immediate, single-keypress access. It beats searching shell history or remembering aliases.
-   **Profiles are Contexts:** A profile is a complete reconfiguration of the grid. Switch from a "Dev" deck (`go build`, `test`) to an "Ops" deck (`nmap`, `ssh`) instantly.
-   **Portable Configuration:** Your entire setup lives in `~/.config/drako`. Git-manage your own profile folder and `summon` it with `drako summon`. You can deploy your exact control panel to any new machine in an instant.
-   **Harness, Don't Replace:** It integrates with the tools you already use. If it runs in the terminal, it can be bound to the grid.
---
>>>>>>> 01310d8 (moved config and exported all)

## 🪄 Summoning Profiles

Share and reuse command decks across machines and teams. Instead of manually copying profiles, summon them directly from remote sources:

```bash

# Clones the repo and looks for .profile.toml files.
# Discards the temporary repo

drako summon git@github.com:user/my_profile_collection.git
```

Works with any Git host (GitHub, GitLab, self-hosted). Summoned profiles land in your inventory, validated for syntax before copying.

If a profile needs extra files (scripts, configs), declare it under `assets = [" relative path / or / dir",...]`. `drako` will offer to copy those files alongside the profile, keeping their paths intact.

## ⚠️ Safety First

- Only run commands you understand. Some entries perform system changes (e.g., package updates, Docker operations).
- Review commands: press `e` to open the command description and read every command.
- When unsure, consult documentation or ask a trusted friend/colleague.

## Roadmap 

 - [x] Update Bootstrap collection
 - [x] Summon profiles incl assets
 - [ ] DRY and Refactor  
 - [ ] Full unit test suite
 - [ ] Windows support (limited)
 - [ ] Steamdeck support (limited)
 - [ ] ARM Support
 - [ ] CI/CD



## 🤝 Contribution

Ideas are welcome. Bugs will be hunted.
-   **Issues:** Report defects or propose architectural changes.
-   **Pull Requests:** Fork the repository and submit your work.
-   **Alpha State:** `drako` is currently in ALPHA. It is stable but evolving. This is your opportunity to influence its development.



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
