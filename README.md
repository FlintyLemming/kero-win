# Kero (kero-win)

本仓库是 [egoist/kero](https://github.com/egoist/kero) 的**自用魔改版本**，定位为一款轻量、纯粹的跨平台终端（重点面向 **Windows** 日常使用体验进行适配与定制）。

- **上游仓库（`Upstream`）**：`https://github.com/egoist/kero`
- **自用分支（`main`）**：在跟踪上游代码的基础上维护本地定制补丁，定期手动同步 `Upstream/main`
- **当前版本**：`2.1.35-patch.2026111521`（基于上游 `2.1.35` / commit `09eb1d4`）

详细改动明细、涉及文件与上游同步流程见 [修改台账（PATCHES.md）](PATCHES.md)。

## 已修改内容

### 1. 移除 AI / Coding Agent 相关功能（`3b36bed`）

将 Kero 精简为纯粹的会话式终端复用器，移除所有面向 Coding Agent 的耦合逻辑：

- **状态检测精简**：移除 `internal/detect`、`agents.toml` 与 OSC `9;4` 进度识别；窗格状态仅通过终端标题 Spinner 或 `kero report`（OSC `7501`）主动上报判定。
- **CLI 与插件清理**：移除 `kero skill`、`kero hook` 命令以及 Pi / OMP / OpenCode 相关集成插件和文档。
- **对话历史与恢复清理**：移除 Agent 对话持久化、`kero history`、`kero resume`、侧边栏 `Ended` / `Archived` 列表，恢复会话时直接重新启动 Shell。
- **变更对比精简**：移除 Changes 面板中的 Agent Turn 快照、`Last Turn` 基准、`Revert`/`Keep` 操作及 `kero diff --turn`。
- **界面与文案调整**：设置页将 `Agents` 调整为 `Command Line`，移除 `Resume agents` 与 `Session history` 等开关，移除 Agent 品牌图标、配色与多语言文案。

### 2. 分支治理、台账与版本规范

- **独立台账**：通过 [PATCHES.md](PATCHES.md) 登记所有本地魔改项及对应上游同步冲突处理建议。
- **版本与 Tag 约定**：见下文规则，`mygo.json` 中的 `"version"` 与 Release Tag 保持严格一致。

### 3. 恢复 1.x 风格项目侧边栏与顶部横向标签页（`c4c5218`）

参考上游 `1.x` 的界面布局，恢复“左侧边栏管理 Projects + 顶部横向标签页管理当前项目会话”的经典模式（默认关闭以保持 `2.x` 默认布局）：

- **设置与配置开关**：在 **Settings → General** 中新增 **Top session tabs / 顶部会话标签页** 开关，并对应配置文件键 `top-tabs = true`（支持设置搜索）。
- **左侧项目管理**：开启后侧边栏仅列出项目行，支持项目汇总状态、会话数显示、`Cmd/Ctrl+1..9` 快速切换项目、拖拽排序、右键菜单及记忆各项目上次打开的会话；空项目也会保持选中并提示新建会话。
- **顶部横向标签栏**：在主区域上方横向展示当前项目的会话标签（支持横向滚动、悬停关闭、鼠标中键关闭、双击重命名、`+` 新建会话、水平拖拽重排顺序、右键关闭其他/右侧会话，`Ctrl+Tab` 在当前项目标签内切换），并自动协调侧边栏折叠按钮与系统窗口控制按钮位置。

<img width="914" height="321" alt="图片" src="https://github.com/user-attachments/assets/9b70e51c-2180-4022-8a7f-02435ce60fef" />

### 4. 项目级右键默认 Shell 与默认命令（`c4c5218`）

在侧边栏项目右键菜单（标准模式与顶部标签页模式共用）中新增项目级启动配置，并随 `kerod` 状态持久化：

- **默认 Shell（Default Shell）子菜单**：通过 `host.shells` 与 `internal/shell` 自动探测主机可用终端（Windows 下支持 PowerShell 7 / `pwsh`、Windows PowerShell、命令提示符 `cmd`、已安装的 WSL 发行版如 `Ubuntu`、Git Bash；Unix 下列出系统可用 Shell），可通过 `project.shell` 为每个项目单独指定默认 Shell 或切回“自动（跟随主机默认）”。

  <img width="715" height="316" alt="图片" src="https://github.com/user-attachments/assets/cf4c37b4-9993-44e9-8675-b29ae85afa69" />
  
- **默认命令（Default Command…）**：通过 `project.command` 弹窗为指定项目设置创建新会话时自动执行的启动命令，并在项目 Tooltip 中展示当前配置。

  <img width="418" height="298" alt="图片" src="https://github.com/user-attachments/assets/5ba42b63-08ab-4f13-a406-5cdfb163a421" />

  <img width="736" height="376" alt="图片" src="https://github.com/user-attachments/assets/ada3a071-8766-491e-b0c6-b91e2f70131e" />

### 5. Windows 专项适配

> 后续针对 Windows 的终端行为、快捷键、窗口表现、ConPTY / CLI 安装等改进将陆续合入 `main`，并同步登记至 [PATCHES.md](PATCHES.md)。

### 6. 自用更新源与纯手动 Windows 发布工作流

- **独立更新源与签名校验**：将 [mygo.json](mygo.json)、[internal/update/update.go](internal/update/update.go) 及服务端安装脚本（[scripts/install-server.sh](scripts/install-server.sh)、[scripts/install-server.ps1](scripts/install-server.ps1)）中的 Release 仓库改为 `FlintyLemming/kero-win`，并替换为独立的 Ed25519 更新签名公钥（私钥通过 GitHub Actions Secret `MYGO_UPDATER_PRIVATE_KEY` 注入）。
- **纯手动 Windows 桌面端发布**：[.github/workflows/release.yml](.github/workflows/release.yml) 取消 `push: tags` 自动触发，仅保留 `workflow_dispatch` 手动触发；仅构建并校验 Windows 桌面端（`windows/amd64`、`windows/arm64`），移除 macOS / Linux 桌面端与 `kero-server` 打包签名任务，并将 `test` 任务限定为 `windows-latest`。[.github/workflows/ci.yml](.github/workflows/ci.yml) 将自动编译与测试限定为 `windows-latest`（仅执行 `go vet ./...`、`go test ./internal/...` 与 Windows `go build ./...`），移除跨平台 / `kero-server` 构建、安装脚本检查与自动调用 `server.yml` 的打包流程，并删除 `.github/workflows/server.yml`。同时修复 `internal/daemon` 与 `internal/git` 测试在 Windows 下的权限位、跨盘符路径、盘符前缀与符号链接兼容性。

### 7. 左上角加号改为“新建工作区（Workspace）”并支持无目录创建

- **新建工作区弹窗（`New Workspace…`）**：将侧边栏左上角 `+` 按钮（含展开侧边栏与标准模式下折叠侧边栏，对应新增动作 `project.new`、快捷键 `CmdOrCtrl+Shift+N` / `Ctrl+Shift+N`）从“新建会话”重定义为“新建工作区”，支持：
  - **仅填名称无目录创建**：只需输入名称即可创建工作区，无需绑定文件夹；
  - **可选文件夹**：可直接输入路径（支持本地 `~` 展开与校验），或通过`选择…`调用系统目录选择器（本地） / 内置目录选择器（远程主机），选定后在名称为空时自动回填目录名；
  - **主机、默认 Shell 与默认命令**：支持在新建时直接选择目标主机（多主机环境）、默认 Shell 与默认启动命令，`project.create` 同步持久化 `Shell` 与 `Command`；
  - **创建后自动进入工作状态**：工作区创建完成后自动在其中启动一个初始会话并选中。
- **既有快捷入口保持不变**：左下角文件夹按钮保留为快捷“打开工作区（选目录）”，顶部标签页模式下标签栏右侧的 `+` 按钮保持为“在当前工作区新建会话”。
- **全局文案统一**：将菜单、弹窗、侧边栏、设置页、设置搜索及目录选择器中原本面向用户的 `Project / 项目 / プロジェクト` 统一为 `Workspace / 工作区 / ワークスペース`（保留 `project.open` 动作 ID 以兼容既有快捷键覆盖）。

### 8. 最大化 / 全屏时支持右上角盲点关闭窗口

- **主窗口与设置窗口角落关闭热区**：在自绘窗口控制按钮（Windows / Linux）处于最大化（`IsMaximized`）或全屏（`IsFullScreen`）状态时，将右上角关闭按钮（`close-window`）的交互热区无缝延伸至窗口最右上角边缘（`Top(0)`、`Right(0)`），移动鼠标到屏幕最右上角盲点即可直接高亮关闭按钮并点击关闭窗口，不再误触窗口拖拽或无响应。
- **视觉与普通窗口保持一致**：关闭按钮视觉上的圆角胶囊高亮与图标位置保持不变；还原为普通浮动窗口时自动恢复紧凑热区，不影响右上角拖拽与调整窗口大小。

### 9. 顶部标签栏右键 `+` 选择 Shell 新建标签页（不应用默认目录/命令）

- **右键菜单选择 Shell**：在顶部标签页（`top-tabs`）模式下，右键点击 Workspace 标签栏右侧的 `+` 新建标签页按钮，会弹出目标主机可用的 Shell 菜单（包含自动发现的 PowerShell / Command Prompt / Bash / WSL / macOS / Linux Shell 等，以及本地自定义 `Shell` 设置项）。
- **纯净启动，不应用默认文件夹与默认命令**：通过该菜单选择 Shell 新建标签页时，会显式跳过应用级默认目录设置（`NewSessionDir` / `NewSessionPath`）以及 Workspace 级默认文件夹（`Dir`）、默认命令（`Command`）和默认 Shell（`Shell`），直接以所选 Shell 在主目录中启动新会话；左键点击 `+` 仍按原有默认配置创建标签页。

### 10. 快捷键设置支持复制与粘贴（Copy / Paste）

- **Keyboard 设置页新增复制 / 粘贴配置项**：在 `internal/actions/actions.go` 的 `Edit` 分组中新增 `edit.copy`（复制，macOS 默认 `CmdOrCtrl+C`、Windows / Linux 默认 `Ctrl+Shift+C`）与 `edit.paste`（粘贴，macOS 默认 `CmdOrCtrl+V`、Windows / Linux 默认 `Ctrl+Shift+V`），支持在 **设置 → 快捷键**、设置搜索及配置文件 `keybind` 中自由修改、重置、解绑，并新增 `Insert` 键（含 `Shift+Insert`、`Ctrl+Insert`）的录制、显示与配置序列化支持。
- **全场景协同生效**：
  - **终端窗格与 Quick Terminal**：自定义复制/粘贴快捷键会同步写入 `terminal.Options.Keybinds`（`termKeybinds`）覆盖默认主快捷键并即时刷新菜单与终端上下文菜单显示，同时在 Windows 上保留仅选中文本时才复制、未选中时仍发送 SIGINT 的 `Ctrl+C` 降级行为；
  - **菜单栏、文本输入框与浏览器窗格**：Edit 菜单的 `RoleCopy` / `RolePaste` 自动跟随所配快捷键（解绑时不回落回系统默认键位，也不拦截正在录制中的快捷键输入）。

## 版本号与 Tag 规则

为了在定期同步上游的同时清晰标识每次自用构建，并兼容 `mygo` 与 `.github/workflows/release.yml` / `internal/update` 的格式校验，版本与 Tag 统一采用：

- **版本号（`mygo.json`）**：`<上游版本>-patch.<YYYYMMDDHH>`
  - 示例：`2.1.35-patch.2026111518`
- **Git Tag**：`v<上游版本>-patch.<YYYYMMDDHH>`
  - 示例：`v2.1.35-patch.2026111518`

## 常用维护命令

```sh
# 拉取上游最新代码
git fetch Upstream

# 合并上游 main 到本地 main（定期手动同步）
git merge Upstream/main

# 运行测试（Windows/全平台 internal 包）
go test ./internal/...
```

## 原版功能与许可

保留上游 Kero 的核心终端能力（`kerod` 守护进程会话持久化、`kero` CLI 脚本化控制、SSH/WSL 远程主机管理、基于 libghostty 的终端渲染）。

原版文档见 [docs/index.md](docs/index.md)。本项目遵循 [LICENSE](LICENSE)（MIT）。
