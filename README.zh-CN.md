# drako

[English](README.md) · [简体中文](README.zh-CN.md)

[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/lucky7xz/drako?color=007D9C&label=version)](https://github.com/lucky7xz/drako/tags)
[![License](https://img.shields.io/github/license/lucky7xz/drako?color=orange)](https://github.com/lucky7xz/drako/blob/main/LICENCE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/lucky7xz/drako?color=00ADD8&logo=go&logoColor=white)](https://go.dev/)

https://github.com/user-attachments/assets/21fb2340-bc74-4886-a629-8e95d116e830

**可定制的命令面板启动器（Command-Deck Launcher）。**

把你常用的每一条命令放进终端里的一张网格，按一个键即可运行。

- **整合，而非取代。** 只要能在终端里运行，就能绑定到一个格子：工具、TUI、脚本。
- **先看再运行。** 按 `e` 显示格子背后的确切命令。按下 Enter 之前，什么都不会运行。
- **面板是纯文本。** 一个 profile 就是一个 TOML 文件：自己编写，用 Git 管理，与他人分享。
- **Summon（召唤）。** `drako summon <repo>` 从任意 Git 托管站点拉取面板；`--show` 先列出其中每一条命令，`--rev`/`--sha256` 用于校验。
- **一个格子，适配所有系统。** 同一条命令可按平台区分（apt、pacman、brew……），drako 会选出合适的那一条。
- **批量启动。** 标记多个格子，在 tmux 或 herdr 中并排启动。
- **脚本与 AI 代理。** 通过 `drako ls`、`explain` 和 `run`，脚本或代理可以使用你整理好的命令，而不是自行编造。
- **没有意外。** 单个静态二进制文件，无遥测，无更新检查。只有在 summon 时 drako 才会访问网络。

完整文档（英文）：**[chronyx.xyz/docs/drako](https://chronyx.xyz/docs/drako/)**

## 安装

```bash
go install github.com/lucky7xz/drako@latest
```

想尝鲜？`@dev` 安装 `dev` 分支上的最新改动：更新，但不那么稳定。

```bash
go install github.com/lucky7xz/drako@dev
```

需要 Go 1.25 或更高版本。常见的安装方式：

| 平台 | 命令 |
| --- | --- |
| Arch | `sudo pacman -S go` |
| Fedora | `sudo dnf install golang` |
| macOS | `brew install go` |
| Windows | `winget install GoLang.Go` |
| Debian、Ubuntu 及其他 | [go.dev/dl](https://go.dev/dl/)（发行版自带的版本常常低于 1.25） |

然后启动它。第一次请使用完整路径，因为 `~/go/bin` 通常还不在你的 `PATH` 中：

```bash
~/go/bin/drako
```

要一劳永逸地解决，请在 Core profile 的 **🐉 Settings** 格子中运行 **Add go/bin to PATH**。它会把 `~/go/bin` 加入 `~/.bashrc`（macOS 上为 `~/.zshrc`，Windows 上为 PowerShell 配置文件）。打开一个新的 shell，直接输入 `drako` 即可。

没有 Go？[Releases](https://github.com/lucky7xz/drako/releases) 页面提供适用于 Linux、macOS 和 FreeBSD 的预编译静态二进制文件，以及 Linux 安装包。安装你下载的那个：

| 发行版 | 命令 |
| --- | --- |
| Debian、Ubuntu | `sudo apt install ./drako_*.deb` |
| Fedora、RHEL | `sudo dnf install ./drako_*.rpm` |
| openSUSE | `sudo zypper install --allow-unsigned-rpm ./drako_*.rpm` |
| Arch | `sudo pacman -U ./drako_*.pkg.tar.zst` |
| Alpine | `doas apk add --allow-untrusted ./drako_*.apk` |

方向键移动，`Enter` 运行，`e` 查看说明，`?` 显示所有按键，`q` 退出。

**Termux（Android）：** 请关闭 cgo 构建，因为 Termux 默认不带 C 工具链：

```bash
CGO_ENABLED=0 go install github.com/lucky7xz/drako@latest
```

Termux 没有 `sudo`，所以首次启动后请运行一次 **🚨 Sudo Switch**（Core profile 的 **Settings** 格子）。推荐使用实体键盘，但不是必需的。

**Emoji：** 面板用 emoji 作为视觉标记。现代终端（Ghostty、WezTerm 等）开箱即可显示；较旧的终端可能需要 [Nerd Font](https://www.nerdfonts.com/) 或 emoji 字体包，例如 `fonts-noto-color-emoji`。

drako 以 Linux 为先。macOS 和 Windows 在设计上受支持，但测试较少。

## 可选依赖

drako 本身只是一个二进制文件。下列工具在某项功能需要时才会查找；缺少其中一个，只会失去那一项功能，绝不会崩溃。

| 工具 | 用途 | 缺少时 |
| --- | --- | --- |
| `bash` | 运行格子（默认 shell 为 `bash -lc`） | 格子运行失败，除非设置 `default_shell`（`sh`、`zsh`、`fish`、`pwsh`、`cmd`） |
| `git` | 从仓库 summon 面板 | 仍可 summon 单个文件或本地文件夹 |
| `tmux` 或 `herdr` | 批量启动 | 批量启动会提示缺少什么 |
| `wl-copy`、`xclip` 或 `xsel` | 复制命令（`y`） | drako 改用 OSC 52，许多终端都支持 |
| `xdg-open` | `drako open`、打开路径 | 该操作失败 |
| `$VISUAL` / `$EDITOR` | 在库存中编辑面板（`e`） | drako 依次尝试 `nano`、`vim`、`vi` |

面板会用到各自的工具（`btop`、`mpv`、`distrobox` 等）。Core 的 **📦 Install Tools** 格子可以安装常用的那些。

## 在线试用

```bash
ssh chronyx.xyz
```

通过 SSH 提供的 drako，运行于 Glassroot 模式，为访客锁定了界面。此功能为实验性质且多人共用：请把它当作公共终端。背后的面板是 [groot demo](https://github.com/lucky7xz/groot_demo)。

## 面板

面板是一份可运行的速查表：每个格子都带着一条命令和一段说明，知识随按键一起流转。

**随 drako 附带。** Core 在首次启动时装备；其余的在库存（`i`）中等待。

| 面板 | 内容 |
| --- | --- |
| **Core** | 按发行版更新系统、清理与病毒扫描、进程监视器、文件查找、带宽、系统状态、工具安装器和设置 |
| **ssh-101** | 运行并加固 SSH 服务器：服务、防火墙、配置与日志、密钥、仅密钥登录、传输、隧道 |
| **tmux-101** | 会话、窗格与复制模式：tmux 基础，也是批量启动所依赖的引擎 |
| **jukebox** | 终端收音机：SomaFM、世界新闻、精选电台、离线噪音（mpv），以及你自己的音乐（cliamp） |
| **distrobox** | 创建、进入、导出和删除容器；用完即弃的容器，适合测试有风险的操作 |

**Summon 更多。** 使用 `drako summon <repo>`，或通过 Core 的 **📦 Install Tools** 格子。

| 面板 | 内容 |
| --- | --- |
| [**101 Series**](https://github.com/lucky7xz/101-deck) | 四个入门面板：docker、git、网络和 unix |
| [**GGML**](https://github.com/lucky7xz/ggml-deck) | 用 llama.cpp 和 whisper.cpp 运行本地 AI：安装、模型、Web UI、语音转写 |
| [**groot demo**](https://github.com/lucky7xz/groot_demo) | `ssh chronyx.xyz` 背后的面板 |

你自己的面板：把一组 `.profile.toml` 文件放进 Git，在任何机器上 summon。

## 安全

面板是一组将在你的机器上运行的命令。请像对待可执行文件一样对待 summon 来的面板：先阅读（`--show`，然后按 `e`），只使用来自可信来源的面板，并固定（pin）你拉取的版本。

## 给 AI 代理的说明

`drako agents` 会输出一份简短说明，教 AI 代理如何使用你的面板。把它加入代理的指令文件：

```bash
drako agents >> AGENTS.md
```

## 参与贡献

欢迎通过 issue 提出想法和报告缺陷。目前不接受 pull request；详见 [CONTRIBUTING](docs/CONTRIBUTING.md)。drako 处于早期测试（beta）阶段。

## 开发

```bash
git clone https://github.com/lucky7xz/drako && cd drako
go vet ./... && go test ./...
```

drako 的开发大量借助 AI。代码库刻意保持小而模块化，使一个人就能完整审查。

## 构建所用

- [`charmbracelet/bubbletea`](https://github.com/charmbracelet/bubbletea)：TUI
- [`charmbracelet/lipgloss`](https://github.com/charmbracelet/lipgloss)：样式
- [`BurntSushi/toml`](https://github.com/BurntSushi/toml)：TOML
- [`shirou/gopsutil`](https://github.com/shirou/gopsutil)：系统信息
- [`fsnotify/fsnotify`](https://github.com/fsnotify/fsnotify)：文件监视
- [`yuin/gopher-lua`](https://github.com/yuin/gopher-lua)：Lua
- [`golang.org/x/term`](https://pkg.go.dev/golang.org/x/term)：终端处理

## 许可证

引擎采用 [AGPL-3.0](LICENCE) 许可。随附的 bootstrap 面板采用 MIT 许可。

<div align="center">

Tame the chaos.

</div>
