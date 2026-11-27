<div align="center">

# drako

**可定制的命令面板启动器（Command-Deck Launcher）。**

把你常用的每一条命令放进终端里的一张网格，按一个键即可运行。

[![Docs](https://img.shields.io/badge/docs-chronyx.xyz-ff5555)](https://chronyx.xyz/docs/drako/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/lucky7xz/drako?color=007D9C&label=version)](https://github.com/lucky7xz/drako/tags)
[![Coverage](https://img.shields.io/coverallsCoverage/github/lucky7xz/drako?branch=dev)](https://coveralls.io/github/lucky7xz/drako?branch=dev)
[![License](https://img.shields.io/github/license/lucky7xz/drako?color=orange)](https://github.com/lucky7xz/drako/blob/main/LICENCE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/lucky7xz/drako?color=00ADD8&logo=go&logoColor=white)](https://go.dev/)

**[文档](https://chronyx.xyz/docs/drako/)** · [English](README.md) · [简体中文](README.zh-CN.md)

<img width="100%" alt="drako 在 tmux 中批量启动" src="https://github.com/user-attachments/assets/274abaed-1cd8-4c35-8b96-4b37b85556e6" />

</div>

- **整合，而非取代。** 只要能在终端里运行，就能绑定到一个格子：工具、TUI、脚本。
- **先看再运行。** 按 `e` 显示格子背后的确切命令。按下 Enter 之前，什么都不会运行。
- **面板是纯文本。** 一个 profile 就是一个 TOML 文件：自己编写，用 Git 管理，与他人分享。
- **Summon（召唤）。** `drako summon <repo>` 从任意 Git 托管站点拉取面板；`--show` 先列出其中每一条命令，`--rev`/`--sha256` 用于校验。
- **一个格子，适配所有系统。** 同一条命令可按平台区分（apt、pacman、brew……），drako 会选出合适的那一条。
- **批量启动。** 标记多个格子，在 tmux 或 herdr 中并排启动。
- **脚本与 AI 代理。** 通过 `drako ls`、`explain` 和 `run`，脚本或代理可以使用你整理好的命令，而不是自行编造。
- **没有意外。** 单个静态二进制文件，无遥测，无更新检查。只有在 summon 时 drako 才会访问网络。

完整文档（英文）：**[chronyx.xyz/docs/drako](https://chronyx.xyz/docs/drako/)**

https://github.com/user-attachments/assets/9be2906d-2389-4e4e-98c8-cbded653ff79

## 安装

Linux、macOS、FreeBSD：

```bash
curl -fsSL https://github.com/lucky7xz/drako/releases/latest/download/install.sh | sh
```

它会下载适合你系统的版本，校验 checksum，并把 `drako` 放到 `~/.local/bin`。不需要 root。

使用 Go 1.25 或更高版本：

```bash
go install github.com/lucky7xz/drako@latest
```

想尝鲜？`@dev` 安装 `dev` 分支上的最新改动：更新，但不那么稳定。

```bash
go install github.com/lucky7xz/drako@dev
```

然后启动它。第一次请使用完整路径，因为 `~/go/bin` 通常还不在你的 `PATH` 中：

```bash
~/go/bin/drako
```

要一劳永逸地解决，请在 Core profile 的 **🐉 Settings** 格子中运行 **Add drako to PATH**。它会把 `~/.local/bin` 和 `~/go/bin` 加入 `~/.bashrc`（macOS 上为 `~/.zshrc`，Windows 上为 PowerShell 配置文件）。打开一个新的 shell，直接输入 `drako` 即可。

二进制文件和 Linux 安装包（.deb、.rpm、.apk、Arch）在 [Releases](https://github.com/lucky7xz/drako/releases) 页面。[安装](https://chronyx.xyz/docs/drako/install/)页面（英文）介绍了所有这些方式，包括 Termux。

**Emoji：** profile 用 emoji 作为视觉标记。现代终端（Ghostty、WezTerm 等）开箱即可显示；较旧的终端可能需要 [Nerd Font](https://www.nerdfonts.com/) 或 emoji 字体包，例如 `fonts-noto-color-emoji`。

drako 以 Linux 为先。macOS 和 FreeBSD 可以使用，但测试较少。目前不支持 Windows：`go install` 可以构建，但体验还不完善。

## 可选依赖

drako 本身只是一个二进制文件。下列工具在某项功能需要时才会查找；缺少其中一个，只会失去那一项功能，绝不会崩溃。

| 工具 | 用途 | 缺少时 |
| --- | --- | --- |
| `bash` | 运行格子（默认 shell 为 `bash -lc`；Windows 上为 `powershell`） | 格子运行失败；可设置 `default_shell = "sh"` 或 `"zsh"`，随附的 profile 在这两者下同样能运行 |
| `git` | 从仓库 summon 面板 | 仍可 summon 单个文件或本地文件夹 |
| `tmux` 或 `herdr` | 批量启动 | 批量启动会提示缺少什么 |
| `wl-copy`、`xclip` 或 `xsel` | 复制命令（`y`） | drako 改用 OSC 52，许多终端都支持 |
| `xdg-open` | `drako open`、打开路径 | 该操作失败 |
| `$VISUAL` / `$EDITOR` | 在库存中编辑 profile（`e`） | drako 依次尝试 `nano`、`vim`、`vi` |

面板会用到各自的工具（`btop`、`mpv`、`distrobox` 等）。Core 的 **📦 Install Tools** 格子可以安装常用的那些。

## 在线试用

```bash
ssh chronyx.xyz
```

通过 SSH 提供的 drako，运行于 Glassroot 模式，为访客锁定了界面。此功能为实验性质且多人共用：请把它当作公共终端。背后的面板是 [groot demo](https://github.com/lucky7xz/groot_demo)。

## 面板

profile 是你载入 drako 的一张网格。面板（deck）是你分享的东西：一个装有若干 profile 的文件夹或仓库，外加它们附带的脚本和工具。（面板和 profile 有时可以互换使用。）

**随 drako 附带。** Core 在首次启动时装备；其余的在库存（`i`）中等待。

| Profile | 内容 |
| --- | --- |
| **Core** | 按发行版更新系统、清理与病毒扫描、进程监视器、文件查找、带宽、系统状态、工具安装器和设置 |
| **ssh-101** | 运行并加固 SSH 服务器：服务、防火墙、配置与日志、密钥、仅密钥登录、传输、隧道 |
| **tmux-101** | 会话、窗格与复制模式：tmux 基础，也是批量启动所依赖的引擎 |
| **jukebox** | 终端收音机：SomaFM、世界新闻、精选电台、离线噪音（mpv），以及你自己的音乐（cliamp） |
| **distrobox** | 运行其他发行版的容器：创建、进入、停止、升级、导出和删除，或启动一个退出即删除的临时容器。容器与你共享主目录：在里面删除的文件，在外面也就没了（!） |

https://github.com/user-attachments/assets/7fa86d77-3345-4103-be3e-26b545a11e4d

*一个 Core，两个发行版：同样的格子在 Arch 容器里运行 `pacman`，在 Fedora 容器里运行 `dnf`。*

**Summon 更多。** 使用 `drako summon <repo>`，或通过 Core 的 **📦 Install Tools** 格子。

| 面板 | 内容 |
| --- | --- |
| [**101 Series**](https://github.com/lucky7xz/101-deck) | 四个入门 profile：docker、git、网络和 unix |
| [**GGML**](https://github.com/lucky7xz/ggml-deck) | 用 llama.cpp 和 whisper.cpp 运行本地 AI：安装、模型、Web UI、语音转写 |
| [**groot demo**](https://github.com/lucky7xz/groot_demo) | `ssh chronyx.xyz` 背后的面板 |

你自己的面板：把一组 `.profile.toml` 文件放进 Git，在任何机器上 summon。

## 安全

面板是一组将在你的机器上运行的命令。请像对待可执行文件一样对待 summon 来的面板：只下载和运行你信任的内容！先阅读（`--show`）。不确定某条命令的作用？运行之前先问问懂行的人。

## 给 AI 代理的说明

`drako agents` 会输出一份简短说明，教 AI 代理如何使用你的面板。把它加入代理的指令文件。命令会被记录，但不在沙盒中运行。请确保你了解已装备的内容。

```bash
drako agents >> AGENTS.md
```

## 参与贡献

欢迎通过 issue 提出想法和报告缺陷。目前不接受 pull request；详见 [CONTRIBUTING](docs/CONTRIBUTING.md)。drako 处于测试（beta）阶段。

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

引擎采用 [AGPL-3.0](LICENCE) 许可。随附的 bootstrap profile 采用 MIT 许可。

<div align="center">

Tame the chaos.

</div>
