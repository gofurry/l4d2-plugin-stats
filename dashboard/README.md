# L4D2 Stats Dashboard

`dashboard/` 是统计系统的只读展示端。它使用 Go + Fiber 提供 API，并把 React 首页嵌入同一个二进制；Dashboard 自己的站点设置保存在独立 SQLite 中，不会迁移或修改 SourceMod 采集器的 Stats DB。

## 已实现

- Go 1.26、Fiber v3、Cobra、Zap 与 Lumberjack 日志轮转；
- React 19、TypeScript、Vite、Ant Design 6、Tailwind 4 与 SCSS Modules；
- Stats DB 只读支持 SQLite、MySQL、PostgreSQL；
- Dashboard DB 使用纯 Go SQLite，Goose 自动迁移；
- sqlc 为四种存储目标生成类型安全查询；
- 首页展示全服历史指标、唯一主服务器 A2S 状态和自定义页脚；
- API 60 秒聚合缓存、A2S 30 秒缓存/5 分钟最后成功结果回退；
- systemd 安装、诊断、Bootstrap 重置和单二进制生产运行。

Steam OpenID、个人中心、JWT 管理后台不属于 v0.8.0–v0.8.1，见[路线图](../docs/roadmap.md)。

## 目录

```text
dashboard/
├─ cmd/l4d2-stats/              CLI 入口
├─ internal/                    配置、服务、存储、A2S、HTTP、systemd
├─ database/dashboard/          Dashboard SQLite 迁移与查询
├─ database/stats/              三种 Stats DB 方言的只读查询
├─ frontend/                    React 源码
├─ web/dist/                    生产前端产物，由 Go 嵌入
├─ config.example.yaml
├─ go.mod
└─ sqlc.yaml
```

`internal/store/sqlcgen/` 是生成代码，只能通过 `go tool sqlc generate` 更新，不能手工编辑。

## 本地开发

1. 将 `config.example.yaml` 复制为 `config.yaml`，填写现有 Stats DB 路径和主服务器地址。
2. 在 `frontend/` 执行 `pnpm install --frozen-lockfile`。
3. 后端执行 `go run ./cmd/l4d2-stats serve --config ./config.yaml`，监听 `127.0.0.1:18848`。
4. 前端执行 `pnpm dev`，访问 `http://127.0.0.1:10086`；Vite 会代理 `/api`。

VS Code 中可直接运行 `Dashboard: API Dev`、`Dashboard: Frontend Dev` 和 `Dashboard: Verify and Build`。

## Stats DB 配置

SQLite 路径相对于 `config.yaml`，应用会自动以 `mode=ro` 打开：

```yaml
stats_database:
  driver: "sqlite"
  dsn: "./l4d2_player_stats.sq3"
```

MySQL 应使用只有 `SELECT` 权限的账号：

```yaml
stats_database:
  driver: "mysql"
  dsn: "lps_reader:password@tcp(127.0.0.1:3306)/l4d2_stats?parseTime=true"
```

PostgreSQL 同样使用只读账号：

```yaml
stats_database:
  driver: "pgsql"
  dsn: "postgres://lps_reader:password@127.0.0.1:5432/l4d2_stats?sslmode=disable"
```

服务启动时只检查 `lps_schema_migrations` 兼容性，不会对 Stats DB 执行 Goose 或任何写查询。公开查询也不会选择 Session 中的 IP 地址。

## Bootstrap

首次启动时，如果 Dashboard DB 尚无站点或服务器记录，会导入 `bootstrap`。后续编辑配置不会默默覆盖数据库。

需要在管理后台完成前显式重置时执行：

```text
l4d2-stats bootstrap apply --replace --config ./config.yaml
```

页脚只接受纯文本和 `http/https` 链接，不渲染 HTML 或 Markdown。

## 构建与验证

Windows：

```powershell
.\scripts\build-dashboard.ps1
```

Linux：

```sh
sh ./scripts/build-dashboard.sh
```

脚本依次完成前端测试/检查/构建、sqlc 生成、Go 测试和单二进制编译。产物为 `dist/l4d2-stats(.exe)`。生产环境只需该二进制和 `config.yaml`；首次运行会在配置文件目录生成 `dashboard.db`、SQLite sidecar 与轮转日志。

详细部署、Nginx、健康检查和回滚步骤见[Dashboard 部署指南](../docs/dashboard-deployment.md)。

## CLI

```text
l4d2-stats serve
l4d2-stats install
l4d2-stats uninstall
l4d2-stats doctor
l4d2-stats version
l4d2-stats migrate status
l4d2-stats bootstrap apply --replace
```

所有命令均接受 `--config ./config.yaml`。

## 公开 API

```text
GET /api/v1/health/live
GET /api/v1/health/ready
GET /api/v1/site
GET /api/v1/dashboard/overview
GET /api/v1/servers/primary/status
```

API 统一返回 `data` 与 `request_id`；错误返回 `error` 与 `request_id`。不存在的 API 路由始终是 JSON 404，不会回退到 React 页面。
