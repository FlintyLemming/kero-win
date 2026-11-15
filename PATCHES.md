# 魔改台账（Modification Ledger）

本文件用于记录 `kero-win`（`origin/main`）相对上游 [egoist/kero](https://github.com/egoist/kero)（`Upstream/main`）的所有本地修改，以便定期手动同步上游代码时快速定位差异、解决冲突和追踪回归。

## 仓库与版本基线

| 项目 | 值 |
| --- | --- |
| 上游 Remote | `Upstream` (`https://github.com/egoist/kero.git`) |
| 下游 Remote | `origin` (`git@github.com:FlintyLemming/kero-win.git`) |
| 工作分支 | `main` |
| 当前上游同步基线 | `09eb1d42dfe24c8ffc07600024e152e096532519`（基于上游 `v2.1.35` 后 1 个提交） |
| 当前版本 / Tag | `2.1.35-patch.2026111521` / `v2.1.35-patch.2026111521` |

## 版本号与 Tag 规则

- **格式**：
  - `mygo.json` `"version"`：`<upstream_version>-patch.<YYYYMMDDHH>`
  - Git Tag：`v<upstream_version>-patch.<YYYYMMDDHH>`
- **命名说明**：
  - `<upstream_version>` 取当前已同步的上游 `mygo.json` 版本号（如 `2.1.35`）。
  - `<YYYYMMDDHH>` 取打包/发布时的本地日期与小时（24 小时制，如 `2026111518`）。
- **兼容性**：
  - 符合 SemVer 2.0.0 预发布格式；
  - 满足 `internal/update/update.go` 中 `ValidVersion`（`^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$`）；
  - 满足 `scripts/install-server.ps1`、`scripts/install-server.sh` 与 `.github/workflows/release.yml` 的 `v$(jq -r .version mygo.json)` 校验；
  - 满足 `mygo` Windows 资源版本解析（在 `-` 处截断主版本号）。

---

## 修改台账总览

| ID | 状态 | 日期 | 提交 | 类别 | 摘要 |
| --- | --- | --- | --- | --- | --- |
| P001 | 已应用 | 2026-11-13 | `3b36bed` | 功能精简 | 移除全部 AI / Coding Agent 功能，还原为通用终端 |
| P002 | 已应用 | 2026-11-15 | `ee669f1` | 仓库治理 | 配置 `Upstream`、更新 `README.md`、建立台账 `PATCHES.md`、设定 `patch` 版本规则 |
| P003 | 已应用 | 2026-11-15 | `c4c5218` | UI 与项目配置 | 恢复 1.x 风格项目侧边栏 + 顶部横向标签页（`top-tabs`），新增项目级默认 Shell 与默认命令 |
| P004 | 已应用 | 2026-11-15 | `b057a97` | UI 与交互 | 为顶部会话标签页增加鼠标中键关闭功能 |
| P005 | 已应用 | 2026-11-15 | `ee6979f` | 更新源与 CI/CD | 更新源与公钥切换到 `FlintyLemming/kero-win`；发布改为纯手动 Windows 桌面端；CI/测试限定 Windows 并修复 Windows 测试兼容性 |
| P006 | 已应用 | 2026-11-15 | `e775b2f` | UI 与工作区管理 | 左上角 `+` 改为新建工作区弹窗（支持无目录创建、Shell/命令预配置），全量统一 Workspace 术语 |
| P007 | 已应用 | 2026-11-15 | `7cccddc` | Windows UI 与交互 | 最大化或全屏时将主窗口与设置窗口关闭按钮热区延伸至最右上角盲点 |
| P008 | 已应用 | 2026-11-15 | `7cccddc` | UI 与会话创建 | Workspace 顶部标签栏右键 `+` 支持选择 Shell 新建标签页，且不应用默认文件夹与默认命令 |
| P009 | 已应用 | 2026-11-15 | `bf7369e` | 快捷键与设置 | 在快捷键设置、配置文件、Edit 菜单与终端中新增可自定义/解绑的复制（`edit.copy`）与粘贴（`edit.paste`）快捷键（含 `Insert` 键录制） |
| P010 | 已应用 | 2026-11-15 | 待提交 | Windows 测试与发布 | 修复 `windows-latest` Runner 下 `TestSaveKeepsASymlinkedConfig` 与 `TestConsoleInheritCursor/system` 兼容性，并补充 `CHANGELOG.md` 发版说明 |

---

## 修改明细

### P001：移除全部 AI / Coding Agent 功能

- **Commit**：`3b36bed` (`Remove the AI features`)
- **目的**：去掉 Coding Agent 专属逻辑，仅保留纯净的会话终端、守护进程持久化、CLI 与远程主机能力。
- **改动清单**：
  1. **状态检测与协议**：
     - 删除 `internal/detect/`（含所有 `manifests/*.toml` 与测试）以及 `agents.toml` 支持；
     - 删除 OSC `9;4` 进度识别，仅保留标题 Spinner 与 `kero report`（OSC `7501`）。
  2. **CLI 与 Agent 插件**：
     - 删除 `internal/cli/skill.go`、`internal/cli/skill_test.go`、`internal/cli/hook.go`、`internal/cli/hook_test.go`、`internal/cli/hook_windows_test.go`；
     - 删除 `internal/cli/agents/`（Pi / OMP / OpenCode 插件）；
     - 精简 `internal/cli/cli.go`、`internal/cli/config.go`、`internal/cli/events.go`、`internal/cli/remote.go` 中的 Agent 命令与文案。
  3. **会话历史与恢复**：
     - 删除 `internal/daemon/conversations.go`、`internal/daemon/conversations_test.go`、`ended.go`；
     - 精简 `internal/daemon/session.go`、`sessions.go`、`store.go`、`rpc.go`、`git.go`、`internal/proto/proto.go`，移除对话记录与恢复逻辑，侧边栏不再显示 `Ended`/`Archived`；
     - 恢复会话时直接启动默认 Shell。
  4. **Changes / Diff 精简**：
     - 精简 `internal/git/diff.go`、`internal/git/git.go`、`view.go` 等，移除 Agent Turn 快照、`Last Turn`、`Revert`/`Keep`、`kero diff --turn`。
  5. **UI、设置、图标与本地化**：
     - 调整 `app.go`、`side.go`、`palette.go`、`settings_view.go`、`settings_search.go`、`awake.go`、`icons.go` 等：Settings 中 `Agents` 改为 `Command Line`，移除 `Resume agents` / `Session history`；
     - 删除 `icons/si-*.svg` 与 `icons/f7-arrow-uturn-left.svg`；
     - 删除 `docs/agents.md`、`docs/ja/agents.md`、`docs/zh/agents.md` 并更新 `docs/sidebar.json`、`README.md`；
     - 更新 `locales/en.json`、`locales/ja.json`、`locales/zh-Hans.json` 及 `locales/catalog_gen.go`。
- **同步上游时的冲突处理原则**：
  - 若上游修改了上述已删除目录/文件（如 `internal/detect/**`、`internal/cli/hook*`、`internal/cli/skill*`、`internal/daemon/conversations*`、`ended.go`、`docs/*/agents.md`），保持删除状态（`git rm`）。
  - 若上游在 `side.go`、`settings_view.go`、`internal/daemon/session.go`、`internal/cli/cli.go`、`locales/*.json` 等共享文件中新增或调整了非 AI 功能，保留上游新功能并剔除其中引用已删 Agent 模块的代码；如涉及翻译生成，同步检查 `locales/catalog_gen.go`。

### P002：仓库治理与版本规范

- **日期**：2026-11-15
- **改动清单**：
  - 添加上游远程仓库 `Upstream` -> `https://github.com/egoist/kero.git`；
  - 重写 [README.md](README.md)，说明仓库用途、已改动点与版本规则；
  - 新增本台账 [PATCHES.md](PATCHES.md)；
  - 更新 [mygo.json](mygo.json) 的 `"version"` 为 `2.1.35-patch.2026111518`。

### P003：1.x 风格顶部会话标签页、项目默认 Shell 与默认命令

- **Commit**：`c4c5218` (`Feat: add top session tabs, project default command, and project default shell`)
- **日期**：2026-11-15
- **目的**：找回上游 `1.x` 左侧项目导航 + 顶部横向标签页布局（可通过设置切换），并支持在项目右键菜单中为不同项目独立指定默认 Shell 与默认启动命令。
- **改动清单**：
  1. **设置与本地化**：
     - 在 `internal/settings/settings.go` 与 `internal/settings/config.go` 新增 `TopTabs bool`（配置项 `top-tabs`，默认 `false`），并在 `internal/settings/config_test.go` 增加配置往返测试；
     - 在 `settings_view.go`（General 页）和 `settings_search.go` 暴露 **Top session tabs / 顶部会话标签页** 开关与检索关键词；
     - 在 `locales/en.json`、`locales/ja.json`、`locales/zh-Hans.json` 增加对应词条并重新生成 `locales/catalog_gen.go`。
  2. **项目切换、标签排序与快捷键**：
     - 在 `app.go`、`hosts.go`、`palette.go` 维护当前选中项目（`selectedProject`）与各项目上次会话记忆（`projectSession`），支持 `Cmd/Ctrl+1..9` 切换项目、`Ctrl+Tab` / 上下一会话在当前项目标签内切换，以及关闭最后一个会话时留在空项目。
  3. **顶部横向标签栏与项目侧边栏 UI**：
     - 新增 `tabs.go`（`projectGroupView`、`projectRow`、`topTabsBar`、`sessionTabItem` 等）与 `tabs_test.go` UI 测试；
     - 调整 `view.go`、`panes.go`、`side.go`，在开启 `TopTabs` 时渲染顶部横向标签栏及空项目提示，并避免窗格/右侧面板头部重复占用顶部控制区。
  4. **项目级默认 Shell 与默认命令**：
     - 新增 `internal/shell/`（`list.go`、`list_windows.go`、`list_unix.go`、`list_test.go`）及改进 `internal/shell/account_windows.go`，用于探测主机可用 Shell（`pwsh`、Windows PowerShell、`cmd`、WSL 发行版、Git Bash 及 Unix shells）；
     - 新增 `project_shell.go`（`fetchShells`、`projectShellMenu`、`setProjectShell` 等），在 `view.go`（`projectContextMenu`、`projectCommandView`）为项目右键菜单增加“默认 Shell”子菜单与“默认命令…”弹窗；
     - 在 `internal/proto/proto.go`、`hosts.go`、`internal/daemon/`（`rpc.go`、`sessions.go`、`session.go`、`shellcmd.go`、`store.go`、`project_test.go`）实现 `host.shells`、`project.shell`、`project.command` RPC 与持久化恢复（并在 `project_order_test.go` 补充相关测试）。
- **同步上游时的冲突处理原则**：
  - `tabs.go`、`tabs_test.go`、`project_shell.go`、`internal/daemon/shellcmd.go`、`internal/shell/list*.go` 为独立新增文件，仅需随上游接口小幅适配；
  - `view.go`、`panes.go`、`side.go`、`app.go`、`hosts.go`、`settings_view.go` 及 `internal/daemon/*` 合并冲突时，保留上游新功能并保持 `a.set.TopTabs` 分支以及 `Project.Shell` / `Project.Command` 字段的持久化路径。

### P004：顶部会话标签页支持鼠标中键关闭

- **日期**：2026-11-15
- **目的**：在开启顶部横向标签页（`top-tabs`）时，支持使用鼠标中键直接关闭任意会话标签页（不切换当前激活标签与终端焦点，并沿用 `ConfirmClose` 与 Worktree 关闭确认）。
- **改动清单**：
  - `app.go`、`view.go`：维护并每帧清理 `tabMiddleClose` 状态；
  - `tabs.go`（`sessionTabItem`）：通过 `HandleInput` 捕获中键按下与抬起（`ev.Button == 2`），仅在非重命名、非拖拽且指针仍悬停在目标标签上释放时触发 `closeSession(sid, true)`；
  - `tabs_test.go`：新增 `TestTopTabsMiddleClickClose` 覆盖中键关闭、拖出取消、`CancelPress`、重命名中忽略及 `ConfirmClose` 确认逻辑。
- **同步上游时的冲突处理原则**：
  - `tabs.go` 与 `tabs_test.go` 为独立新增文件；合并 `app.go` 与 `view.go` 时保留 `tabMiddleClose` 字段及每帧清理逻辑即可。

### P005：自用更新源、纯手动 Windows 发布与 Windows 测试适配

- **日期**：2026-11-15
- **目的**：使桌面端/服务端更新检查与下载指向 `FlintyLemming/kero-win` 并使用自有密钥签名；工作流仅保留纯手动 Windows 桌面端发布，并将测试环境限定为 Windows 且确保 `go test ./internal/...` 在 Windows 下全绿。
- **改动清单**：
  1. **更新仓库与签名公钥**：
     - 修改 `mygo.json`（`updates.github`、`updates.publicKey`）、`internal/update/update.go`（`Repo`、`PublicKey`）以及 `scripts/install-server.sh`、`scripts/install-server.ps1`；
     - 使用 `go tool mygo keygen` 生成独立 Ed25519 密钥对，公钥写入仓库配置，私钥由用户配置到 GitHub Secret `MYGO_UPDATER_PRIVATE_KEY`。
  2. **GitHub Actions 工作流精简**：
     - 修改 `.github/workflows/release.yml`：移除 `push: tags: ["v*"]` 自动触发（仅保留 `workflow_dispatch`）；仅构建 `windows/amd64,windows/arm64` 桌面端，移除 `macOS`、`Linux` 及 `server` / `server-sign` 任务，将 `test` 任务限定为 `windows-latest`，并更新 `assets` 校验清单；
     - 修改 `.github/workflows/ci.yml`：将 `test` 任务改为 `runs-on: windows-latest`，保留 `go vet ./...`、`go test ./internal/...` 与 Windows `go build ./...`，移除跨平台/`kero-server` 构建、Server 安装脚本检查及自动调用 `server.yml` 的 `server` 任务；
     - 删除 `.github/workflows/server.yml`。
  3. **Windows 测试兼容性修复**：
     - `internal/daemon/fs_test.go`（`TestFSWrite`）与 `internal/git/git_test.go`（`TestWrite`）：改为断言写入前后权限位一致，兼容 Windows NTFS 无 POSIX `0755` 权限位；
     - `internal/daemon/project_test.go`（`TestProjectCreateNormalizesRelativeDirectory`、`TestResolveShellAndArgs`）：使用同一 `t.TempDir()` 下的 `t.Chdir` 避免 Windows 跨盘符 `filepath.Rel` 失败，并将 `wsl:Ubuntu` 用例收入 `runtime.GOOS == "windows"` 分支；
     - `internal/daemon/worktree_test.go`（`TestWorktreePath`）：在 Windows 下为虚拟绝对路径加上 `C:` 盘符；
     - `internal/git/git_test.go`（`TestLocal`）：将符号链接测试拆分子测试并在 Windows 无符号链接权限时跳过。
- **同步上游时的冲突处理原则**：
  - `mygo.json`、`internal/update/update.go`、`scripts/install-server.*` 中保持 `FlintyLemming/kero-win` 与自有 `publicKey` 不被上游覆盖；
  - `.github/workflows/release.yml` 与 `.github/workflows/ci.yml` 保持纯手动 `workflow_dispatch` 发布、Windows-only 构建与测试策略；若上游更新 `.github/workflows/server.yml`，保持删除状态（`git rm`）；
  - 上述 `internal/*/*_test.go` 合并时保留 Windows 兼容断言。

### P006：左上角 `+` 改为新建工作区与 Workspace 术语统一

- **日期**：2026-11-15
- **目的**：将左上角 `+` 从直接新建会话改为“新建工作区”，允许用户不选目录仅凭名称（或可选关联文件夹）并预设默认 Shell / 命令创建工作区，同时统一桌面端术语为 Workspace / 工作区 / ワークスペース。
- **改动清单**：
  1. **新建工作区弹窗与逻辑（`workspace.go`、`app.go`、`picker.go`）**：
     - 在 `App` 新增 `wsAsk *workspaceAsk`，新增 `workspace.go` 实现 `startNewWorkspace`、`workspaceView`、`submitNewWorkspace`、`chooseWorkspaceFolder` 与 `createWorkspace`；
     - 支持名称创建（无目录）及可选目录创建（本地 `mygo.Dialog.Open`，远程通过 `picker.go` 新增的 `folderPicker.onPick` / `pickFolderOn` 回调选目录并返回弹窗，保持表单状态）；
     - 支持多主机切换、默认 Shell 选择（`hostShellOptions`）、默认命令输入，并在创建后自动在其中启动初始会话并选中。
  2. **按钮重定义与界面适配（`view.go`、`panes.go`、`palette.go`）**：
     - 将展开侧边栏与标准模式折叠侧边栏的左上角 `+` 按钮从 `session.new` 改为 `project.new`（`startNewWorkspace(nil)`）；TopTabs 标签栏右侧 `+` 与左下角打开目录按钮保持原功能；
     - 在 `view.go`（拖拽防护、TopTabs 空工作区图标 `terminalIcon`）与 `palette.go`（远程无目录工作区详情格式、`wsAsk` 打开时禁用全局快捷键）补齐边界处理。
  3. **动作、Daemon 与本地化（`internal/actions/actions.go`、`actions.go`、`settings_search.go`、`internal/daemon/rpc.go`、`locales/*.json`、`locales/catalog_gen.go`）**：
     - 新增 `project.new`（`CmdOrCtrl+Shift+N` / `Ctrl+Shift+N`）并将 `project.open` 文案改为 `Open Workspace…`（保留原 ID）；
     - `internal/daemon/rpc.go` 的 `project.create` 支持在创建时直接保存 `Command` 与 `Shell`；
     - 统一 `locales/en.json`、`locales/zh-Hans.json`、`locales/ja.json` 中的 `Project / 项目 / プロジェクト` 术语为 Workspace，并新增弹窗词条，重新生成 `locales/catalog_gen.go`。
  4. **测试（`workspace_test.go`、`internal/daemon/project_test.go`）**：
     - 新增 `workspace_test.go` 与 `TestProjectCreateCommandAndShell` 覆盖上述交互、校验、RPC 参数与远程目录回填。
- **同步上游时的冲突处理原则**：
  - `workspace.go` 与 `workspace_test.go` 为独立新增文件，合入上游后按需适配内部接口即可；
  - 合并 `view.go`、`panes.go`、`picker.go`、`actions.go`、`internal/actions/actions.go`、`internal/daemon/rpc.go` 与 `locales/*.json` 时，保留左上角 `+` 的 `project.new` 绑定、`pickFolderOn` 回调、`project.create` 的 `Command`/`Shell` 字段保存及 `locales` 中的 Workspace 文案，并用 `go tool mygo generate` 重新生成 `locales/catalog_gen.go`。

### P007：最大化 / 全屏时右上角盲点关闭窗口

- **日期**：2026-11-15
- **目的**：在 Windows 等自绘窗口控制按钮（`showControls`）环境下，当窗口最大化或进入全屏时，遵循 Fitts's Law 将右上角关闭按钮的点击/悬停热区延伸至窗口最右上角边缘（`Top(0)`、`Right(0)`），使鼠标直接甩到最右上角盲点即可高亮并点击关闭窗口。
- **改动清单**：
  1. **通用角标控制栏布局（`platform.go`、`app.go`、`side.go`）**：
     - 抽象 `windowControlTarget` 接口与 `cornerControls` 布局辅助函数；在 `IsMaximized() || IsFullScreen()` 时将控制栏锚定到 `Right(0).Top(0)`，通过 `Margin` 保持最小化/最大化按钮与原有位置完全一致，同时将 `close-window` 按钮热区扩展到最右上角并通过 `Draw` 保留原 `28×26`、`Radius(6)` 悬停红底视觉；
     - 在 `App` 增加仅测试使用的 `testWin` / `testSetWin` 替身字段并统一通过 `resolveWindowTarget` 解析。
  2. **设置窗口同步支持（`settings_view.go`）**：
     - 将设置窗口控制按钮从被 `card.Clip()` 裁剪的卡片标题栏内移至与主窗口一致的顶层角落覆盖层（`settingsControls`），并在原标题栏右侧保留 `controlsW + 6` 占位宽度，使设置窗口在最大化或全屏时同样支持右上角盲点关闭。
  3. **测试（`platform_test.go`）**：
     - 新增 `TestCornerCloseButton`（覆盖 `TopTabs` 开/关与右侧边栏开/关的 4 种组合，验证普通窗口不误触、最大化/全屏最右上角悬停高亮与点击/滑动释放/双击关闭、不溢出外角背景及最小化/还原按钮正常）与 `TestSettingsCornerCloseButton`。
- **同步上游时的冲突处理原则**：
  - 合并 `platform.go`、`settings_view.go`、`side.go` 时，保留 `cornerControls`、`settingsControls` 及设置页卡片外部控件层（及标题栏占位），若上游调整 `barMid`、`cardGap` 或 `controlButtons` 尺寸需同步更新对应常量。

### P008：Workspace 顶部标签栏右键 `+` 选择 Shell 新建标签页

- **日期**：2026-11-15
- **目的**：在 Workspace 顶部标签栏中，保留左键点击 `+` 使用默认设置新建标签页的同时，支持右键 `+` 菜单选择特定 Shell 新建标签页，并显式跳过任何默认文件夹与默认命令设置。
- **改动清单**：
  1. **协议与守护进程（`internal/proto/proto.go`、`internal/daemon/sessions.go`）**：
     - 在 `proto.CreateParams` 新增可选字段 `NoDefaults bool`（`json:"noDefaults,omitempty"`），保持协议版本兼容；
     - 在 `Daemon.create` 中，当 `p.NoDefaults` 为 `true` 时跳过工作区默认目录（`pr.Dir`）、默认命令（`pr.Command`）与默认 Shell（`pr.Shell`）回退，令 `Dir` 为空以由 `pane.start` 回退至 `proto.Home()`。
  2. **客户端创建与右键菜单（`app.go`、`view.go`、`project_shell.go`、`tabs.go`）**：
     - 在 `App` 新增 `newSessionWithShell(h *host, projectID, shellID string)`，绕过 `NewSessionDir` / `NewSessionPath` 等默认目录解析，以 `NoDefaults: true` 与指定 `Shell` 在当前主机的当前工作区新建会话（同主机有活动会话时沿用 `After` / `Cols` / `Rows`）；
     - `iconButton` 返回 `ui.Element`；在 `project_shell.go` 新增 `App.newTabShellMenu`，列出目标主机可用 Shell（`hostShellOptions`）及未重复的本地自定义 `a.set.Shell`，并在 `tabs.go` 顶部标签栏的 `new-tab` `+` 按钮（含空工作区与有会话工作区分支）挂载该右键菜单。
  3. **测试（`internal/daemon/project_test.go`、`tabs_test.go`）**：
     - 新增 `TestCreateNoDefaultsSkipsWorkspaceDefaults` 与 `TestTopTabsNewTabRightClickShellMenu`，覆盖 `NoDefaults` 跳过工作区默认目录/命令/Shell 与默认 `NewSessionDir`、各分支右键菜单选择及自定义 Shell 条目。
- **同步上游时的冲突处理原则**：
  - 合并 `internal/proto/proto.go`、`internal/daemon/sessions.go` 与 `tabs.go` 时，保留 `CreateParams.NoDefaults` 字段及其在 `Daemon.create` 的短路逻辑，以及 `topTabsBar` 中 `new-tab` 按钮的 `a.newTabShellMenu` 绑定；
  - 若上游修改 `hostShellOptions` 或 `newSessionOn` 参数结构，需同步调整 `newTabShellMenu` 与 `newSessionWithShell`。

### P009：快捷键设置支持复制与粘贴（Copy / Paste）

- **日期**：2026-11-15
- **目的**：在“设置 → 快捷键”（`Settings → Keyboard`）与 `keybind` 配置中补齐“复制（Copy）”和“粘贴（Paste）”的可自定义/解绑快捷键项，并使其在菜单栏、终端窗格、Quick Terminal、文本输入框及浏览器窗格中协同生效。
- **改动清单**：
  1. **动作定义与国际化（`internal/actions/actions.go`、`locales/{en,zh-Hans,ja}.json`、`locales/catalog_gen.go`、`settings_search.go`、`palette.go`）**：
     - 在 `actions.All` 的 `Edit` 菜单分组中新增 `edit.copy`（macOS 默认 `CmdOrCtrl+C`，`pcKeys` 默认 `Ctrl+Shift+C`）与 `edit.paste`（macOS 默认 `CmdOrCtrl+V`，`pcKeys` 默认 `Ctrl+Shift+V`），并在 `words` 补充 `insert` 显示名；
     - 在 `locales/{en,zh-Hans,ja}.json` 新增 `actionCopy` / `actionPaste` 并运行 `go tool mygo generate` 重新生成 `locales/catalog_gen.go`；在 `settings_search.go` 补充检索关键词；在 `palette.go` 过滤这两个依赖当前焦点的 Edit 角色动作。
  2. **快捷键录制与配置序列化（`settings_view.go`、`internal/settings/config.go`）**：
     - `settings_view.go` 的 `keyNames` 新增 `ui.KeyInsert: "Insert"`，并在 `recordKey` 允许仅带 `Shift` 修饰的 `Shift+Insert` 录制；
     - `internal/settings/config.go` 的 `accelerator` 支持 `"insert"` 规范化为 `"Insert"`。
  3. **Edit 菜单与终端键位联动（`actions.go`、`app.go`）**：
     - 在 `actions.go` 新增 `App.editRoleItem`（复用 `RoleCopy` / `RolePaste`，在未配置或解绑时传入 `"none"` 避免 `mygo` 回落为默认 `CmdOrCtrl+C/V`）、共享 `parseAccelerator` 辅助函数（供 `menuShortcut` 与终端键位共用）及 `termKeybinds`（在自定义或解绑时移除主非 performable 默认绑定、按需注入新绑定并保留 Windows 下 `Ctrl+C` 的 performable SIGINT 退避行为）；
     - 在 `app.go` 的 `termOptions` 注入 `termKeybinds` 并在 `apply` 的 `anew` 条件监听 `edit.copy` / `edit.paste` 变更以刷新在线终端实例。
  4. **测试（`internal/settings/config_test.go`、`settings_test.go`）**：
     - 在 `TestKeybinds` 补充 `edit.copy` / `edit.paste`（含 `shift+insert`）配置往返测试；新增 `TestCopyPasteShortcuts` 覆盖设置页展示、设置搜索、`Shift+Insert`/`Alt+C` 录制持久化、菜单角色加速键、`termKeybinds` 及解绑行为；同步调整 `TestRecordShortcut` 测试视口高度（`1200`）以容纳新增行。
- **同步上游时的冲突处理原则**：
  - 合并 `internal/actions/actions.go`、`actions.go`、`app.go` 与 `locales/*.json` 时，保留 `edit.copy` / `edit.paste`、`editRoleItem` 与 `termKeybinds`；
  - 若上游修改 `Edit` 菜单项构造或 `termOptions` 参数，需保持 `RoleCopy` / `RolePaste` 对 `a.accelerator` 的引用，并在 `anew` 中继续追踪 `edit.copy` / `edit.paste` 变更，然后重新执行 `go tool mygo generate`。

### P010：Windows Runner 测试兼容性与发版日志

- **日期**：2026-11-15
- **目的**：修复在 GitHub `windows-latest`（开启 Developer Mode 符号链接权限的 Windows Server 2022）上暴露的两个上游测试环境假定，并补齐 `mygo build -upload` 要求的 `CHANGELOG.md` 版本说明。
- **改动清单**：
  1. `internal/settings/config_test.go`（`TestSaveKeepsASymlinkedConfig`）：保存前记录 `os.Stat(dotfiles)` 权限位，保存后断言前后权限位一致，兼容 Windows NTFS（`-rw-rw-rw-`）。
  2. `internal/daemon/pty_windows_test.go`（`TestConsoleInheritCursor`）：参照同文件 `TestConsoleClearScreen`，在 `cp.clear == nil && windows.RtlGetVersion().BuildNumber < 26100` 时跳过 Windows 11 24H2 之前旧版系统自带 ConPTY 子测试，保留 `bundled` ConPTY 验证。
  3. `CHANGELOG.md`：在顶部增加 `## 2.1.35-patch.2026111521` 章节以通过 `mygo build -upload` 校验。
- **同步上游时的冲突处理原则**：
  - 合并 `internal/settings/config_test.go` 与 `internal/daemon/pty_windows_test.go` 时保留上述 Windows 兼容断言/跳过条件；`CHANGELOG.md` 顶部保留当前发行版本章节。

---

## 定期同步上游操作流程

每次需要同步 `Upstream/main` 时，按以下步骤执行：

1. **拉取上游更新**：
   ```sh
   git fetch Upstream --tags
   git log --oneline HEAD..Upstream/main
   ```
2. **合并上游代码到 `main`**：
   ```sh
   git merge Upstream/main
   ```
   - 遇到冲突时，优先参照上方台账中的「冲突处理原则」解决。
   - 若上游更新了 `mygo.json` 中的版本号（例如从 `2.1.35` 升至 `2.1.36`），请以新的上游版本号作为前缀。
3. **运行测试验证**：
   ```sh
   go test ./internal/...
   ```
   （如修改涉及根目录 UI/App 代码，同步执行 `go test ./...`）
4. **更新版本号、台账并打 Tag**：
   - 将 `mygo.json` 中的 `"version"` 设为 `<新上游版本>-patch.<YYYYMMDDHH>`；
   - 在本文件「仓库与版本基线」更新 `当前上游同步基线` commit 与版本，并在台账中登记新增改动或同步记录；
   - 提交变更并创建对应 Tag：
     ```sh
     git commit -am "Sync upstream and bump version to <version>"
     git tag v<version>
     ```
5. **（如需）推送到个人仓库**：
   ```sh
   git push origin main
   git push origin v<version>
   ```
