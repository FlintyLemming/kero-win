# 魔改台账（Modification Ledger）

本文件用于记录 `kero-win`（`origin/main`）相对上游 [egoist/kero](https://github.com/egoist/kero)（`Upstream/main`）的所有本地修改，以便定期手动同步上游代码时快速定位差异、解决冲突和追踪回归。

## 仓库与版本基线

| 项目 | 值 |
| --- | --- |
| 上游 Remote | `Upstream` (`https://github.com/egoist/kero.git`) |
| 下游 Remote | `origin` (`git@github.com:FlintyLemming/kero-win.git`) |
| 工作分支 | `main` |
| 当前上游同步基线 | `09eb1d42dfe24c8ffc07600024e152e096532519`（基于上游 `v2.1.35` 后 1 个提交） |
| 当前版本 / Tag | `2.1.35-patch.2026111518` / `v2.1.35-patch.2026111518` |

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
| P002 | 已应用 | 2026-11-15 | `v2.1.35-patch.2026111518` | 仓库治理 | 配置 `Upstream`、更新 `README.md`、建立台账 `PATCHES.md`、设定 `patch` 版本规则 |

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
