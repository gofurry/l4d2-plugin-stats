# 数据库地基 v0.1.0

## 1. 安装文件

部署脚本会安装：

```text
left4dead2/addons/sourcemod/plugins/l4d2_player_stats.smx
left4dead2/addons/sourcemod/configs/l4d2_player_stats/migrations/sqlite/0001_initial.sql
left4dead2/addons/sourcemod/configs/l4d2_player_stats/migrations/mysql/0001_initial.sql
left4dead2/addons/sourcemod/configs/l4d2_player_stats/migrations/pgsql/0001_initial.sql
```

迁移 SQL 是运行时依赖，不能只复制 `.smx`。

## 2. 配置数据库

把 `config/databases.cfg.example` 中选定的数据库块合并到服务器：

```text
left4dead2/addons/sourcemod/configs/databases.cfg
```

配置名称默认必须是 `l4d2_player_stats`。SQLite 无需外部服务，数据库文件由 SourceMod 建立在：

```text
left4dead2/addons/sourcemod/data/sqlite/l4d2_player_stats.sq3
```

MySQL 使用 `mysql` 驱动；PostgreSQL 使用 SourceMod 驱动标识 `pgsql`。不要把真实密码复制回仓库。

## 3. 配置采集器

插件首次加载会生成：

```text
left4dead2/cfg/sourcemod/l4d2_player_stats.cfg
```

必须修改：

```text
sm_lps_server_key "my-l4d2-server-01"
```

`server_key` 在共享数据库内必须唯一，只允许字母、数字、点、下划线和连字符，长度 1～64。默认值 `change-me` 会让采集器停在配置错误状态，避免不同服务器意外共用身份。

常用配置：

```text
sm_lps_database_config "l4d2_player_stats"
sm_lps_server_name ""
sm_lps_flush_interval "300"
sm_lps_flush_jitter "60"
sm_lps_retry_maximum "60"
sm_lps_log_suppression_window "300"
sm_lps_closed_session_queue_limit "256"
sm_lps_closed_lifecycle_queue_limit "512"
sm_lps_closed_equipment_queue_limit "4096"
sm_lps_versus_stats_enabled "1"
```

`server_name` 留空时使用游戏服务器的 `hostname`。`sm_lps_versus_stats_enabled` 只控制对抗幸存者/感染者玩法统计；身份、Session 和生命周期仍按支持模式契约工作。修改该开关后以新 Segment 为边界生效。

修改连接、驱动、总启用状态或 `server_key` 后需要重载插件，或者在配置有效时执行 `sm_lps_reconnect`。

## 4. SQLite 首次验证

1. 运行 `scripts/deploy.ps1`。
2. 合并 SQLite 数据库配置。
3. 加载插件，让它生成 cfg；填写唯一 `server_key`。
4. 执行 `sm plugins reload l4d2_player_stats`。
5. 执行 `sm_lps_status`。

预期状态：

```text
state=ready
driver=sqlite
schema=1/1
```

当前首次连接会建立 14 张表、9 个查询索引、服务器记录和当前 `boot_id`。数据库仍使用未发布阶段的 `schema=1/1` 完整初始结构。

## 5. 故障验证

- 临时写错数据库配置并重载：状态应进入 `waiting-to-retry`，间隔为 5、15、30、60 秒后保持上限。
- 连续同类错误只会首次立即写入 SourceMod 日志，之后在抑制窗口内合并计数。
- 修复配置后执行 `sm_lps_reconnect`：应恢复到 `ready`，并输出一次恢复消息。
- `sm_lps_flush` 在 ready 状态执行两条异步心跳更新；并发请求会合并，不启动同一份并行刷新。

插件不创建专属日志文件，也不会记录数据库密码、连接字符串或玩家 IP。
