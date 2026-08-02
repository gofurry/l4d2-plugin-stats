# L4D2 Player Stats

一个面向《求生之路 2》服务器的玩家身份、会话和玩法统计系统。

项目采用 monorepo。当前实现为数据库地基 v0.1.0；玩家身份、会话和玩法采集将在后续版本按已确认契约逐步接入。插件稳定后再开发 Go 后端及其内嵌前端。

## 当前状态

v0.1.0 已实现：

- 一个模块化 SourcePawn 插件，最终编译为 `l4d2_player_stats.smx`；
- SQLite、MySQL 和 PostgreSQL 三套等价迁移；
- 异步连接、自动迁移、服务器启动实例注册和异常启动恢复；
- 300 秒加随机抖动的服务器心跳；
- 5/15/30/60 秒有上限重连，以及重复错误日志限流；
- 仅限 root 管理员的状态、重连和立即刷新命令；
- 构建、部署、迁移校验和发布打包脚本。

v0.1.0 **尚不创建玩家 Session，也不采集击杀、伤害或治疗数据**。

已经确认的基础边界：

- 数据库支持 SQLite、MySQL 和 PostgreSQL（SourceMod 驱动名为 `pgsql`）。
- 只采集 `coop`、`realism` 和 `versus`；其他模式完全不记录。
- `coop` 与 `realism` 属于 PvE 统计族，`versus` 使用独立的 PvP 统计模型。
- 只为通过 Steam 认证的真人玩家建立身份和统计，不为 Bot 建立玩家记录。
- 同时保存连接时间和有效参赛时间。
- 保存服务器观察到的玩家 IP 地址，不含端口；IP 不得由公开网页默认展示。
- 身份、会话和统计明细默认永久保留。
- 常规保存周期为 300 秒加可配置随机抖动，并在关键生命周期事件发生时保存。
- 第一阶段数据库故障只使用有界内存状态恢复，不创建无限增长的本地日志或队列文件。

## 契约

- [模式与生命周期](contracts/modes.md)
- [统计口径](contracts/statistics.md)
- [数据库结构](database/schema.md)

## Monorepo 边界

```text
collector/     SourceMod 数据采集插件
database/      三种数据库的结构和迁移
contracts/     插件与未来 Go 服务共同遵守的行为定义
dashboard/     未来的 Go 后端与内嵌前端
docs/          架构、部署和测试文档
scripts/       仓库级构建与发布脚本
```

详细说明见[架构文档](docs/architecture.md)。

后续版本按[开发路线图](docs/roadmap.md)推进。

## 本地构建

1. 将 `scripts/config.example.ps1` 复制为 `scripts/config.local.ps1`，填写本机 SourceMod 路径。
2. 运行 `scripts/build.ps1`；产物位于 `dist/l4d2_player_stats.smx`。
3. 运行 `scripts/deploy.ps1`，插件和三套迁移会复制到本机 SourceMod 环境。

VS Code 可以直接执行 `L4D2 Stats: Build` 或 `L4D2 Stats: Build and Deploy` 任务。

## 服务器配置与验证

数据库、ConVar 和 SQLite 首次验证步骤见[数据库地基部署](docs/database-foundation.md)。数据库密码只应存在于服务器自己的 `addons/sourcemod/configs/databases.cfg`，不得提交到仓库。

管理员命令：

| 命令 | 权限 | 用途 |
|---|---|---|
| `sm_lps_status` | root | 查看连接、驱动、迁移和最近错误 |
| `sm_lps_reconnect` | root | 立即重新连接并重新检查迁移 |
| `sm_lps_flush` | root | 立即执行服务器心跳/刷新请求 |

## License

[MIT](LICENSE)
