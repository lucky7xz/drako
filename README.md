![Demo v0.1.8](docs/demo.gif)

The terminal is a realm of immense power, but also of high entropy. Commands are forgotten, workflows fracture, and focus is lost to the noise. **Drako** is a **TUI-Deck launcher** that enables structure, transforming your terminal into a disciplined, grid-based command center. 


## 🚀 Quick Start

`If go is installed, installing `drako` is a single command.

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
GOPROXY=direct go install github.com/lucky/drako@latest  # install drako
```
If newly added profiles do not appear after the update, note that `drako` only creates the bootstrap folder under .config/drako if there is none present alreay. As such, the new profiles will no be created. 


## Navigation

- **Grid Navigation:** Use w/a/s/d/arrow keys or vim keys (h, j, k, l) to move around the grid. You can also use number keys for col/row if pressed in sequence. Eg. pressing 3 and 4 in quick sequence, will move the cursor to the 3rd column - 4th row.  
- **Switch Profile:** Use `Alt` + number keys (`1`-`9`) to switch directly to a profile. The modifier can be changed in the configuration. 
- **Prifile Inventory:** Use `i` to open the profile inventory to add/remove profiles from your rotation.
- **Lock Current Profile:** Press `r` to lock or unlock the current profile.
- **Tab:** Press `tab` to switch from grid mode to directory mode, or vice versa.
- **Quit:** Press `q` to exit `drako`. Note that TUIs opened with drako will be closed via their own quit command.




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

## ✨ The Core Philosophy

`drako` is built on a few core principles:

-   **The Grid is Your Command Deck.** Your most vital commands are laid out on a visual grid for immediate, single-keypress access. No more searching shell history or forgotten aliases.

-   **Profiles are Shifting Forms.** A profile is a complete reconfiguration of the grid for a different context. Switch from a "Go Developer" deck (`go build`, `go test`) to a "Network Sentinel" deck (`nmap`, `mtr`) instantly.

-   **Your Deck is Portable.** The true power of profiles is their portability. By keeping your `~/.config/drako` directory in a Git repository, you can deploy your entire command center to a new server with a single command. This transforms `drako` into a declarative, repeatable control panel for any machine you manage.

-   **Harness, Don't Replace.** `drako` integrates with the tools you already use. If it runs in the terminal, it can be bound to the grid.

-   **The Power of TUI Decks:** For those who wish to build true terminal cathedrals, `drako` serves as the gateway to `para13`, a `TUI-Deck` build with seamless integration into `drako`. More will be revealed in time. Stay tuned

---

## 🪄 Summoning Profiles

Share and reuse command decks across machines and teams. Instead of manually copying profiles, summon them directly from remote sources:

```bash

# Clones the repo and looks for .profile.toml files.
# Discards the temporary repo

drako summon git@github.com:user/my_profile_collection.git
```

Works with any git server (GitHub, GitLab, Gitea, Bitbucket, self-hosted). Pick and choose the files you need from your repo. Summoned profiles land in `~/.config/drako/inventory/`, ready to equip via the inventory (`i` key). Each profile is validated for safety (size limits, TOML format, profile structure) and requires your confirmation before copying. Private repos require SSH keys

For git repos, profiles can declare `assets = ["path/or/dir", ...]` (relative), which drako copies into `~/.config/drako/` preserving paths, with a pre-copy plan shown and sensible size limits. This way you can copy over a handful of small files with their corresponding profile in one go.

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

---

## 🤝 Contribution

Ideas are welcome. Bugs will be hunted.
-   **Issues:** Report defects or propose architectural changes.
-   **Pull Requests:** Fork the repository and submit your work.
-   **Alpha State:** `drako` is currently in ALPHA. It is stable but evolving. This is your opportunity to influence its development.


## 📜 License

The core Drako engine is released under the [GNU Affero General Public License v3.0](LICENSE). Bootstrap assets in the `bootstrap/` directory are released under either [MIT](bootstrap/LICENSE-MIT) or [Apache-2.0](bootstrap/LICENSE-Apache) licenses.

---
<div align="center">

Tame the chaos.

</div>
