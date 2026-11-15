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

### 3. Windows 专项适配

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
