# L4D2 Stats Dashboard

`dashboard/` 是统计系统的展示端。Go + Fiber 提供 API，React 页面嵌入同一个二进制。Dashboard 自己的管理员、站点和服务器设置保存在独立的纯 Go SQLite 数据库中；日常查询与聚合只读 Stats DB，只有管理员明确确认原始数据清理时才临时打开维护连接。

## 已实现

- Go 1.26、Fiber v3、Cobra、Zap/Lumberjack、Goose、sqlc；
- React 19、TypeScript、Vite、Ant Design 6、Tailwind 4、SCSS Modules；
- Stats DB 只读支持 SQLite、MySQL 和 PostgreSQL；
- 首页历史指标、同优先级多服务器 A2S 状态列表和可选自定义页脚；
- 面向 L4D2 原生 MOTD WebView 的服务端渲染游戏内页面，无 JavaScript/API fan-out，提供当前服务器、在线玩家、公开个人摘要、排行榜、公告和本服文档；
- Steam OpenID 或手动 SteamID64 个人查询，支持时间、服务器、PvE 模式和游标分页；
- ECharts 个人趋势、PvE/装备/Versus 全量明细和独立的全服排行榜；
- 地图/战役、规则环境、标准化 Incident 时间线、Boss 生命周期和个人效率分析；
- 基于同 Round、同阵营正重叠时长的 Top-3 并肩作战预览；
- Achievement Contract v1 自动判定、历史补判、三枚徽章展示位、个人成就页和玩家预览主徽章；
- UTC 日增量聚合、月度/终身汇总读模型、可配置 15 分钟至 24 小时刷新周期和每批 500 行的可审计原始数据清理；
- 网页首次设置、easyhash bcrypt、8 小时 HS256 JWT、Fiber CSRF、登录限流和全站每 IP 每分钟 300 次的温和限流；
- 全局界面语言、背景图片、页脚链接、Steam 登录、服务器目录和账号安全后台；
- 仅管理员可访问的轻量运行监控，展示 Dashboard 进程、Go Runtime、宿主机资源和 HTTP 请求状态；
- systemd 安装、诊断、日志轮转和单二进制生产运行。

## 目录

```text
dashboard/
├─ cmd/l4d2-stats/              CLI 入口
├─ ingame/                      原生 MOTD SSR 模板、旧 WebKit CSS 与 PNG Atlas
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
  listen: "0.0.0.0:18848"

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
- 在“游戏内页面”设置全站默认外观、模块、三个亮点指标和固定缓存档位；展开“服务器管理”中的单台服务器可配置继承、覆盖或隐藏规则和本服文档；
- 在“安全设置”修改管理员用户名和密码。修改密码会让旧 JWT 立即失效。
- 通过“运行监控”在新标签页查看当前 Dashboard 和宿主机的短期实时状态。
- 通过“数据增长监控”查看数据库/日志占用、聚合水位与分析状态，分别配置聚合覆盖数据和 Incident 保留期并手动清理。

启用 Steam 登录后，需要填写玩家实际访问 Dashboard 时使用的完整地址，例如 `https://stats.example.com` 或 `http://203.0.113.10:18848`，供 Steam 验证后返回本站；不支持子路径。若服务器无法直连 Steam，可选填代理地址，例如 `http://127.0.0.1:7890`、`http://10.0.0.8:7890` 或 `socks5://proxy.example.com:1080`；省略协议时按 HTTP 处理，且只有 Steam OpenID 请求使用该代理。没有域名或不启用 Steam 登录都不影响手动 SteamID64 查询。

玩家查询使用的本地 SteamID 记录不参与权限判断。修改徽章展示位前必须重新完成一次 Steam OpenID 验证；服务端只签发 10 分钟有效、绑定本人 SteamID 的 HttpOnly 编辑凭据，写请求还必须来自站点设置中的公开地址。

Steam 登录玩家可在个人中心“设置”中按一级 Tab 控制访客可见内容，默认只公开概览、分析和玩家关系；本人始终可以查看全部栏目。可见性由服务端接口强制执行，不是仅在浏览器中隐藏导航。

Dashboard DB 使用内嵌 Goose migration 自动升级，当前 schema 为 17；Stats schema 为 6，`stats_version` 仍为 1，Aggregate Contract 与 Achievement Contract 均为 v1。升级前仍应停止服务并同时备份 Dashboard DB、Stats DB 与配置文件；不要通过删除 Stats DB 的方式处理版本变化。

Dashboard 服务器 UUID 只标识网页中的实时服务器目录，不需要管理员填写。采集器的 `sm_lps_server_key` 仍是 Stats DB 中的数据来源标识，两者边界独立。L4D2 的加入链接和 A2S 状态查询统一使用同一个服务器地址。

## 游戏内页面

后台“MOTD 部署帮助”会从服务器已经持久化的 A2S 规则中读取 `sm_lps_server_key`，生成跳转到 `/ingame?server=<server_key>` 的 `motd.txt`。Dashboard/Collector 不会写入游戏服务器文件，`host.txt` 仍只能放普通文本。页面请求只使用已有 A2S 快照，不会同步查服。

游戏内 Home、Player、Rankings、公告/文档分别使用批准的缓存档位：Home 10/30/60/120 秒，Player 30/60/120/300 秒，Rankings 60/120/300/600 秒，内容 60/300/600/1800 秒。这里的 View Cache 与“站点设置 → 服务”的 A2S 刷新周期相互独立；缓存命中不会重新查 Stats DB 或 A2S，设置、公告、文档、服务器和个人可见性更新会主动失效相应缓存。Visual v2 使用接近 MOTD viewport 全宽的 Survival Overlay 布局：Banner 仅用于 Home Hero，Player、Rankings、公告和文档使用紧凑服务器 Header；页面保持 SSR-only 与 0 JavaScript。

Background、Banner 和完整网站目的地只接受不含账号密码的绝对 HTTP/HTTPS URL。服务端不会下载、代理、探测或生成外部图片缩略图；更换图片时应使用新 URL 或查询版本。Markdown 仅渲染安全的轻量子集，不允许脚本、iframe、音视频、样式、复杂表格和外部图片，经过校验的 HTTP/HTTPS 外链统一交给 Steam 外部浏览器打开。所有 `/ingame` 玩家数据都按匿名公开可见性查询，即使请求携带 Steam 或管理员会话也不会提升权限。内嵌 CSS/PNG 使用基于内容 SHA-256 的稳定短指纹并保持一年 immutable cache；HTML 继续使用 `no-cache`。

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
l4d2-stats doctor --deep
l4d2-stats version
l4d2-stats migrate status
l4d2-stats aggregate status
l4d2-stats aggregate rebuild
l4d2-stats retention plan
l4d2-stats backup create
l4d2-stats backup restore <file>
l4d2-stats diagnostics export
```

`aggregate rebuild` 仅适用于尚未清理原始数据的环境；清理后为保护历史聚合会拒绝全量重建。日常运行和管理页“立即聚合”使用增量模式。`retention plan` 仍是只读 CLI 预览；真正清理由管理员登录网页后在“数据增长监控”页二次确认执行。备份和诊断归档默认写入当前目录；恢复前必须停止 Dashboard 服务。

统计读模型、永久保留范围和未来清理硬条件见[Dashboard 数据生命周期](../docs/dashboard-data-lifecycle.md)。

所有命令均接受 `--config ./config.yaml`。详细部署步骤见[Dashboard 部署指南](../docs/dashboard-deployment.md)。

## API 边界

- 公开：健康检查、站点、首页、服务器状态列表、Steam OpenID、玩家摘要/活动/PvE/Versus/Session/章节、排行榜、战局分析，以及服务端渲染的 `/ingame` 页面；
- 首次设置：仅没有管理员时可用，并要求进程内一次性令牌；
- 管理：JWT Cookie + Fiber CSRF，覆盖站点、服务器、游戏内页面默认值/单服覆盖/文档和管理员账号；监控页面及其 JSON 快照同样要求管理员 JWT；
- 所有 `/api/v1/*` 错误保持 JSON，不会回退到 React HTML。
