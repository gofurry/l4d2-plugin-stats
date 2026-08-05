# L4D2 Stats Dashboard

`dashboard/` 是统计系统的展示端。Go + Fiber 提供 API，React 页面嵌入同一个二进制。Dashboard 自己的管理员、站点和服务器设置保存在独立的纯 Go SQLite 数据库中；日常查询与聚合只读 Stats DB，只有管理员明确确认原始数据清理时才临时打开维护连接。

## 已实现

- Go 1.26、Fiber v3、Cobra、Zap/Lumberjack、Goose、sqlc；
- React 19、TypeScript、Vite、Ant Design 6、Tailwind 4、SCSS Modules；
- Stats DB 只读支持 SQLite、MySQL 和 PostgreSQL；
- 首页历史指标、同优先级多服务器 A2S 状态列表和可选自定义页脚；
- Steam OpenID 或手动 SteamID64 个人查询，支持时间、服务器、PvE 模式和游标分页；
- ECharts 个人趋势、PvE/装备/Versus 全量明细和独立的全服排行榜；
- UTC 日增量聚合读模型、可配置 15 分钟至 24 小时刷新周期和可审计原始数据清理；
- 网页首次设置、easyhash bcrypt、8 小时 HS256 JWT、Fiber CSRF 和登录限流；
- 全局界面语言、背景图片、页脚链接、Steam 登录、服务器目录和账号安全后台；
- 仅管理员可访问的轻量运行监控，展示 Dashboard 进程、Go Runtime、宿主机资源和 HTTP 请求状态；
- systemd 安装、诊断、日志轮转和单二进制生产运行。

## 目录

```text
dashboard/
├─ cmd/l4d2-stats/              CLI 入口
├─ internal/                    认证、服务、存储、A2S、HTTP、systemd
├─ database/dashboard/          Dashboard SQLite 初始结构与查询
├─ database/stats/              三种 Stats DB 方言的只读查询
├─ frontend/                    React 源码
├─ web/dist/                    前端生产产物，由 Go 嵌入
├─ config.example.yaml
├─ go.mod
└─ sqlc.yaml
```

`internal/store/sqlcgen/` 是 `go tool sqlc generate` 生成的代码，不能手工修改。

## 最小配置

```yaml
server:
  listen: "127.0.0.1:18848"

dashboard_database:
  path: "./dashboard.db"

stats_database:
  driver: "sqlite"
  dsn: "./l4d2_player_stats.sq3"

logging:
  file: "./logs/l4d2-stats.log"

monitor:
  enabled: true
```

相对路径以 `config.yaml` 所在目录为基准。超时、连接池和日志轮转参数均有安全默认值，可按 `config.example.yaml` 的注释选择性覆盖。

SQLite Stats DB 的常规连接会以只读模式打开。MySQL/PostgreSQL 若只使用展示与聚合，可继续使用只有 `SELECT` 权限的账号；若要从管理页执行清理，该 DSN 还需要对清理目标表具备 `DELETE` 权限。服务不会迁移 Stats DB；公开查询不会读取 Session IP。

## 首次设置

第一次启动且没有管理员时，程序会把 30 分钟有效的一次性令牌输出到 stderr；systemd 部署可通过 `journalctl -u l4d2-stats -n 100 --no-pager` 查看。打开 `/admin/setup`，填写令牌、管理员用户名和至少 12 字节的密码。

完成后登录 `/admin`：

- 在“站点设置”管理全局界面语言、在线背景图片、默认关闭的页脚链接和 Steam 登录；
- 在“服务器管理”只填写展示名称和 `host:port` 地址；系统自动生成 UUID，并在列表中完成启停、上下排序、编辑和删除；
- 在“安全设置”修改管理员用户名和密码。修改密码会让旧 JWT 立即失效。
- 通过“运行监控”在新标签页查看当前 Dashboard 和宿主机的短期实时状态。
- 通过“数据增长监控”查看数据库/日志占用、聚合水位，配置保留期并手动清理已聚合的过期数据。

启用 Steam 登录后，需要填写玩家实际访问 Dashboard 时使用的完整地址，例如 `https://stats.example.com` 或 `http://203.0.113.10:18848`，供 Steam 验证后返回本站；不支持子路径。没有域名或不启用 Steam 登录都不影响手动 SteamID64 查询。

> 项目尚未发布，v0.9.1 直接采用新的初始 Dashboard 结构。测试旧版后请先停止程序并删除旧 `dashboard.db`、`dashboard.db-shm`、`dashboard.db-wal`，再启动新版本。Stats DB 不要删除。

Dashboard 服务器 UUID 只标识网页中的实时服务器目录，不需要管理员填写。采集器的 `sm_lps_server_key` 仍是 Stats DB 中的数据来源标识，两者边界独立。L4D2 的加入链接和 A2S 状态查询统一使用同一个服务器地址。

## 本地开发

1. 复制 `config.example.yaml` 为 `config.yaml` 并填写 Stats DB 路径。
2. 在 `frontend/` 执行 `pnpm install --frozen-lockfile`。
3. 后端执行 `go run ./cmd/l4d2-stats serve --config ./config.yaml`。
4. 前端执行 `pnpm dev`，访问 `http://127.0.0.1:10086`；Vite 代理 `/api/v1` 到 `18848`。

## 构建与验证

Windows：`./scripts/build-dashboard.ps1`。Linux：`sh ./scripts/build-dashboard.sh`。

脚本依次执行前端测试、类型检查、Lint、生产构建、sqlc 生成和 Go 测试，然后输出 `dist/l4d2-stats(.exe)`。若 `dist/config.yaml` 不存在，脚本也会放入一份最小测试配置。

## CLI

```text
l4d2-stats serve
l4d2-stats install
l4d2-stats uninstall
l4d2-stats doctor
l4d2-stats version
l4d2-stats migrate status
l4d2-stats aggregate status
l4d2-stats aggregate rebuild
l4d2-stats retention plan
```

`aggregate rebuild` 仅适用于尚未清理原始数据的环境；清理后为保护历史聚合会拒绝全量重建。日常运行和管理页“立即聚合”使用增量模式。`retention plan` 仍是只读 CLI 预览；真正清理由管理员登录网页后在“数据增长监控”页二次确认执行。

统计读模型、永久保留范围和未来清理硬条件见[Dashboard 数据生命周期](../docs/dashboard-data-lifecycle.md)。

所有命令均接受 `--config ./config.yaml`。详细部署步骤见[Dashboard 部署指南](../docs/dashboard-deployment.md)。

## API 边界

- 公开：健康检查、站点、首页、服务器状态列表、Steam OpenID、玩家摘要/活动/PvE/Versus/Session/章节和排行榜；
- 首次设置：仅没有管理员时可用，并要求进程内一次性令牌；
- 管理：JWT Cookie + Fiber CSRF，覆盖站点、服务器和管理员账号；监控页面及其 JSON 快照同样要求管理员 JWT；
- 所有 `/api/v1/*` 错误保持 JSON，不会回退到 React HTML。
