# Kero (kero-win)

本仓库是 [egoist/kero](https://github.com/egoist/kero) 的**自用魔改版本**，定位为一款轻量、纯粹的跨平台终端（重点面向 **Windows** 日常使用体验进行适配与定制）。

- **上游仓库（`Upstream`）**：`https://github.com/egoist/kero`
- **自用分支（`main`）**：在跟踪上游代码的基础上维护本地定制补丁，定期手动同步 `Upstream/main`
- **当前版本**：`2.1.35-patch.2026111518`（基于上游 `2.1.35` / commit `09eb1d4`）

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
