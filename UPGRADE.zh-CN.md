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

`chat-audit.db` 含管理员敏感聊天历史，常规 `backup create` 默认有意排除。需要保留时应停服后使用权限受限的独立备份和独立保留政策，不要把它混入普通诊断归档。

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

v1.3.2 继续执行 `0006_fall_deaths.sql`，把 Stats schema 升至 6，为 PvE 与对抗幸存者增加可空的坠落死亡计数。升级前的 Segment 保持 `NULL`，表示当时没有采集；新 Segment 从 0 开始，并满足 `0 <= fall_deaths <= deaths`。Dashboard schema 升至 15，用于保存 Achievement Contract v1 的永久解锁、自动判定/历史补判进度、玩家徽章展示位和个人中心公开可见性。成就由后台和访问个人资料时自动判定，不需要领取，也没有玩家或管理员手动刷新入口。`stats_version=1`、Aggregate Contract v1 与 Incident Contract v1 保持不变。

v1.3.3 不修改 Stats schema，采集器继续使用 schema 6 和 `stats_version=1`。Dashboard schema 升至 16，用于区分“从未设置徽章展示位”和“明确取消全部展示”；Achievement Contract v1 以兼容方式扩充 Catalog，历史自动补判继续由后台执行，无需领取或手动刷新。

v1.3.4 不修改任何 gameplay 数据、Stats schema 或冻结契约；Collector 只统一版本号。Dashboard schema 从 16 升至 17 时新增独立的游戏内页面表，再升至 18，把覆盖设置和服务器文档迁移为 `server_key` 服务器组范围并新增介绍/状态模块开关；schema 19 增加服务器组快速链接，schema 20 增加全局自定义地图名称，schema 21 移除试用阶段放弃的服务器组短描述字段。同组旧记录按更新时间和 server ID 确定性折叠，无法从持久化 A2S 快照映射的记录不会被任意归组。升级后默认启用轻量 `/ingame` 路由，但仍需管理员配置 `public_origin`、确认 A2S 已识别 `sm_lps_server_key`，并手工部署 `motd.txt`。迁移不会修改现有站点、服务器、公告、玩家可见性或统计数据。

v1.3.5 自动执行 `0007_high_value_telemetry_chat.sql`，把 Stats schema 升至 7：PvE/对抗幸存者增加保护队友、挂边、被 Tank 石块造成伤害、Hunter Skeet 和 Charger Level 五个可空字段，旧 Segment 保持 `NULL`；同一迁移增加默认 72 小时的聊天传输 outbox 与完整性状态。Dashboard schema 自动升至 23，保存默认开启/30 天保留的 Chat Audit 设置、导出审计，以及百度 GeoIP 凭据、1-3 QPS 请求策略和 HMAC 缓存；最终聊天库使用独立 Chat Audit schema 1，并在配置目录旁生成 `chat-audit.db`。Gameplay `stats_version=1` 和 Aggregate/Achievement/Incident/Relationship Contract v1 均不变，也不需要 Left4DHooks 或第三方 SourceMod 插件。

升级前已有 `config.yaml` 未写 `chat_audit` 时会使用安全默认路径 `./chat-audit.db`（相对配置目录）。Chat Audit capture 默认开启；如所在地区/政策不允许采集聊天，应在加载新 Collector 前设置 `sm_lps_chat_audit_enabled 0`。百度 AK 是可选项：没有 AK 就不会发出 provider 请求，保存 AK 后默认限制为 2 QPS，可在审计后台改为 1-3 QPS；清除 AK 即停止新的解析请求。

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

进入至少一个真人参与的完整 Round 后，还应检查 `/analysis`、个人页的 PvE、对抗、“玩家关系”和“成就”标签，以及管理后台数据运维页。`sm_lps_status` 应为 `version=1.3.5`、`schema=7/7`；深度检查不应报告 Context、Incident、Relationship、Assist、新 telemetry 或聊天完整性契约错误。后台“审计”应只对管理员开放，聊天默认最近 24 小时，未配置 GeoIP AK 时不得请求 provider。首次启动会继续自动执行可恢复的历史成就补判，无需人工领取或刷新。

Skeet/Level 的自动化与 SourcePawn 编译不能替代真实引擎顺序验证。发布到正式服前按 [`docs/v1.3.5-technique-validation.md`](docs/v1.3.5-technique-validation.md) 在 L4D2 build 10097 执行全部正反例；完成前不要宣称两个判定器已实机验证。

v1.3.4 还应在后台“游戏内页面”完成一次专项检查：保存全站默认值和固定缓存预设；验证服务器组继承/覆盖/隐藏、快速链接、服务器文档和地图友好名称；确认 A2S server key 后复制 `motd.txt`；最后用生产所用的 Windows Steam + L4D2 客户端验证按 H 打开 Home、Player、Rankings、公告/文档，以及“完整网站”“加入游戏”和快速链接均显示纯文本操作提示卡。游戏内请求不应触发即时 A2S 查询，页面源代码不应包含 React、JavaScript、`fetch` 或 XHR。

## 回滚

如果新版本仅修改二进制且没有执行不可逆数据库迁移，可停止服务并恢复上一份二进制：

```sh
sudo systemctl stop l4d2-stats
cp ./l4d2-stats.previous ./l4d2-stats
sudo systemctl start l4d2-stats
```

如果新版本已经升级 Stats DB 或 `dashboard.db` 结构，旧二进制未必兼容新 schema。从 v1.3.5 回滚必须恢复升级前 Stats schema 6、Dashboard schema 21 的完整备份，并单独处理/保留 `chat-audit.db`；不能只替换旧二进制，也不要手工删除 schema 7/23 的结构。更早版本同样必须按对应发布的 Stats/Dashboard schema 一起恢复。此时应停止 Dashboard 服务，并恢复升级前数据库备份：

```sh
sudo systemctl stop l4d2-stats
./l4d2-stats backup restore ./backup-YYYYMMDDTHHMMSSZ.zip --config ./config.yaml
```

恢复会先校验归档成员、SHA-256、SQLite 完整性、Dashboard/Stats schema、聚合契约版本与当前 driver。当前数据库、配置和 SQLite sidecar 会保留为 `.pre-restore-*` 副本；MySQL/PostgreSQL Stats DB 必须使用原生工具另行恢复。不要手工修改迁移版本号。

原始数据清理是不可逆操作。执行清理前应确认聚合完成并备份 Stats DB；回滚二进制不会恢复被删除的原始记录。
