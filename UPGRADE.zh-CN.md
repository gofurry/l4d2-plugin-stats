# L4D2 Player Stats 升级与回滚

## 升级前备份

至少备份：

```text
left4dead2/addons/sourcemod/plugins/l4d2_player_stats.smx
left4dead2/cfg/sourcemod/l4d2_player_stats.cfg
left4dead2/addons/sourcemod/configs/databases.cfg
Stats 数据库
Dashboard 运行目录中的 l4d2-stats、config.yaml、dashboard.db
```

Dashboard 已运行时可先执行：

```sh
./l4d2-stats backup create --config ./config.yaml
```

该命令使用 SQLite 在线备份 API 同时保护 Dashboard DB 和 SQLite Stats DB，不会遗漏 WAL 数据。MySQL/PostgreSQL Stats DB 会在归档中标记为需要外部备份，必须同时使用数据库原生工具完成备份。

## 升级采集器

1. 将新发布包中的 `left4dead2` 文件夹合并到服务端同名目录；
2. 确认新迁移文件也已复制，不要只替换 `.smx`；
3. 保留已有 `cfg/sourcemod/l4d2_player_stats.cfg` 和 `databases.cfg`；
4. 在服务器控制台执行：

```text
sm plugins reload l4d2_player_stats
sm_lps_status
```

确认版本、数据库状态和 schema 版本符合新发布说明。

v1.2 的采集器会按顺序执行 `0001_initial.sql`、`0002_car_alarms_triggered.sql` 和 `0003_versus_objective_interactions.sql`，把 Stats schema 从 1 升至 3。`0002` 为 PvE 与对抗增加警报车计数，`0003` 为对抗幸存者增加机关互动计数。旧统计行的新字段保持 `NULL`（表示当时尚未采集），新快照写入明确的 0 或正整数；不要手工把历史 `NULL` 回填为 0。

v1.3 继续执行 `0004_analysis_foundation.sql`，把 Stats schema 升至 4 并新增 `lps_round_contexts` 与 `lps_incidents`。升级前的历史核心统计保持有效，但没有历史 Context/Incident 明细；Dashboard 必须把这些 Round 显示为“分析不可用”，不能解释为零事件。Dashboard DB 会从 schema 11 升至 12，以保存独立的 Incident 保留设置和清理审计。既有 `stats_version=1` 与 Aggregate Contract v1 不变。

v1.3.1 继续执行 `0005_relationships_and_assists.sql`，把 Stats schema 升至 5。该迁移新增真人幸存者定向关系表，并为 PvE/对抗幸存者加入可空的 Assist、Witch 参与和对抗黑白恢复字段。旧 Segment 中新字段保持 `NULL`（当时未采集），不得回填为 0；定向关系也不会从旧累计值反推。Dashboard schema 保持 13，`stats_version=1`、Aggregate Contract v1 和 Incident Contract v1 保持不变。

v1.3.2 继续执行 `0006_fall_deaths.sql`，把 Stats schema 升至 6，为 PvE 与对抗幸存者增加可空的坠落死亡计数。升级前的 Segment 保持 `NULL`，表示当时没有采集；新 Segment 从 0 开始，并满足 `0 <= fall_deaths <= deaths`。Dashboard schema 升至 14，用于保存 Achievement Contract v1 的永久解锁、自动判定/历史补判进度和玩家徽章展示位。成就由后台和访问个人资料时自动判定，不需要领取，也没有玩家或管理员手动刷新入口。`stats_version=1`、Aggregate Contract v1 与 Incident Contract v1 保持不变。

## 升级 Dashboard

Linux systemd：

```sh
sudo systemctl stop l4d2-stats
cp ./l4d2-stats ./l4d2-stats.previous
cp /path/to/new/l4d2-stats ./l4d2-stats
chmod +x ./l4d2-stats
./l4d2-stats doctor --config ./config.yaml
sudo systemctl start l4d2-stats
```

Windows：

1. 停止正在运行的 `l4d2-stats.exe`；
2. 将旧二进制保留为 `l4d2-stats.previous.exe`；
3. 替换为新二进制；
4. 使用原 `config.yaml` 执行 `doctor` 后再启动。

Dashboard 启动时会自动迁移自己的 `dashboard.db`。不要使用发布包中的示例配置覆盖现有 `config.yaml`。

升级后检查：

```sh
./l4d2-stats version
./l4d2-stats doctor --config ./config.yaml
./l4d2-stats doctor --deep --config ./config.yaml
```

并访问：

```text
/api/v1/health/live
/api/v1/health/ready
```

进入至少一个真人参与的完整 Round 后，还应检查 `/analysis`、个人页的 PvE、对抗、“玩家关系”和“成就”标签，以及管理后台数据运维页的只读成就引擎状态。`sm_lps_status` 应为 `version=1.3.2`、`schema=6/6`；深度检查不应报告 Context、Incident、Relationship、Assist 或坠落死亡契约错误。首次启动会以每批约 100 名玩家自动执行可恢复的历史成就补判，期间不需要人工操作。

## 回滚

如果新版本仅修改二进制且没有执行不可逆数据库迁移，可停止服务并恢复上一份二进制：

```sh
sudo systemctl stop l4d2-stats
cp ./l4d2-stats.previous ./l4d2-stats
sudo systemctl start l4d2-stats
```

如果新版本已经升级 Stats DB 或 `dashboard.db` 结构，旧二进制未必兼容新 schema。从 v1.3.2 回滚到 v1.3.1 时，安全边界是同时恢复 v1.3.1 Collector/Dashboard、升级前的 Stats schema 5 和 Dashboard schema 13 备份；不要只回退 Collector，也不要手工删除 schema 6 字段或成就表。此时应停止 Dashboard 服务，并同时恢复升级前数据库备份：

```sh
sudo systemctl stop l4d2-stats
./l4d2-stats backup restore ./backup-YYYYMMDDTHHMMSSZ.zip --config ./config.yaml
```

恢复会先校验归档成员、SHA-256、SQLite 完整性、Dashboard/Stats schema、聚合契约版本与当前 driver。当前数据库、配置和 SQLite sidecar 会保留为 `.pre-restore-*` 副本；MySQL/PostgreSQL Stats DB 必须使用原生工具另行恢复。不要手工修改迁移版本号。

原始数据清理是不可逆操作。执行清理前应确认聚合完成并备份 Stats DB；回滚二进制不会恢复被删除的原始记录。
