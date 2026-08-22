# Dashboard 数据生命周期

## 数据边界

- Stats DB 是采集事实来源。正常查询、聚合和网页请求只使用只读连接。
- Dashboard DB 保存站点配置、管理员、服务器目录、UTC 日聚合、维护设置、清理审计，以及 Achievement Contract v1 的永久解锁、判定状态和徽章展示位。
- 只有管理员明确确认“原始数据清理”时，程序才临时打开独立维护连接；不会后台自动删除。
- 公开 API 不读取或返回 Session IP、boot ID、内部数据库主键、DSN 或管理员数据。
- PvE 只包含 `coop`/`realism` 幸存者数据；Versus 的幸存者和感染者分别聚合。
- Round Context 是永久事实；Incident 是可保留的低频分析明细。不完整或历史无 Incident 的 Round 不按零事件处理。
- Player Relationship 是永久的真人定向互动事实，不使用 Incident 保留策略；Assist 属于永久 Core Stats。
- Achievement 解锁一经自动确认便永久保留；自动判定和历史 Backfill 只读取 retention-safe 事实，不提供手动领取、刷新或重建。
- Chat Audit 是独立数据域：Stats DB 只保存默认 72 小时的传输 outbox，最终正文进入 Dashboard 管理的 `chat-audit.db`，默认保留 30 天且只允许管理员查询。
- GeoIP 只在 Dashboard 管理域工作。原始 IP 仍只来自 Stats Session；Dashboard DB 仅保存 HMAC-SHA256 键和近似城市结果，不复制原始 IP。

## 增量聚合

首次运行创建完整 UTC 日聚合，并同步生成月度与终身汇总。后续使用事实表的 `last_saved_at` 水位定位发生变化的日期，只在一个 Dashboard SQLite 事务中替换这些日期并差量修正对应月份和终身键。默认每 30 分钟执行一次，管理员可选择 15/30/60 分钟或 3/5/12/24 小时，也可在“数据增长监控”页立即执行。

聚合覆盖 Session 活动、PvE 全字段与装备、Versus 双阵营及职业、Run 完成量和 Versus 半场/比赛结果数量。这些口径由 [Aggregate Contract v1](../contracts/aggregate-v1.md) 冻结，日、月、终身聚合及状态/清理记录都携带 `aggregate_version = 1`。读到未知版本时系统会拒绝读取或清理，不执行自动升级。

每次聚合记录源水位、变更日期数、聚合行数、耗时和错误。清理发生后禁止全量重建，因为已清理的历史只能由现存日聚合提供；增量聚合仍可继续。

## 可清理范围

管理员可以分别配置明细、已关闭 Session 和比赛结果的保留天数（30–3650 天）。点击清理前，系统会先增量聚合，生成包含聚合版本、源水位与候选数量的预览 ID，校验聚合覆盖当前 Stats DB，再在执行前复核版本、水位和预览都没有变化。删除使用独立写连接，每 500 行一个短事务分批提交，结果写入 `retention_runs` 审计表。

| 数据 | 默认保留 | 历史如何继续展示 |
|---|---:|---|
| PvE 装备明细 | 180 天 | `pve_equipment` 日聚合 |
| Versus 双阵营职业明细 | 180 天 | 职业日聚合 |
| 已关闭 Session | 365 天 | `activity` 日聚合保留总量和趋势；旧逐条 Session 不再展示 |
| Versus 半场/比赛结果 | 365 天 | `versus_result` 日聚合保留完成量；旧逐条结果不再展示 |

不会清理玩家身份、Run、Round、Player Segment、Player Relationship、PvE/Versus 核心总计或正在进行的 Session。IP 随被清理的旧 Session 一并消失，程序不会单独导出或展示它。

Stats schema 7 的五个幸存者指标属于永久核心快照，但保持可空：历史 `NULL` 表示当时没有采集，不能解释为 0。它们不进入 Aggregate Contract v1。保护队友、Skeet、Level 排行榜直接读取永久原始核心行；挂边和被石块命中不提供公开排行榜。

v1.3 为 Incident 引入独立保留策略，默认 180 天。它不依赖 Aggregate 覆盖，只能删除
已结束 Round 且早于截止时间的 Incident；候选空间出现未知 Incident Contract 版本时
必须拒绝清理。删除仍按每批 500 行短事务执行，并使用独立确认与审计。Round Context、
核心统计和所有生命周期事实不受影响。

## 仍会增长的数据

- 玩家身份、Run、Round、Player Segment、Player Relationship 和核心统计永久增长；
- Round Context 永久增长；Incident 按管理员配置的保留窗口增长和清理；
- UTC 日聚合按活跃玩家、服务器和维度线性增长；
- 成就解锁、判定状态、徽章展示位、清理审计、公告和站点配置随玩家或管理员操作增长；
- SQLite 删除行后文件大小不保证立即回落，页面展示的是实际数据库文件占用；
- 应用日志由 Lumberjack 按大小、份数和天数轮转，内存缓存均有容量与 TTL 上限。

## Chat Audit 生命周期

- `sm_lps_chat_audit_enabled` 默认开启，并独立于 gameplay 模式白名单；Survival、Scavenge、Mutation 等不受支持玩法仍可审计真人聊天，但不会创建 gameplay Session/Stats。
- Collector 先给每条符合条件的消息分配 boot 内单调序列，再尝试进入 1024 条有界队列；队列满会留下可诊断序列缺口，不阻断游戏。
- 批量异步写入 Stats `lps_chat_outbox`，Collector 以每批最多 256 行删除超过 72 小时的传输记录；Dashboard 只读 Stats 并以 boot 游标幂等摄取。
- `chat-audit.db` 默认保留 30 天，可选择 7/14/30/60/90/180/365 天或永久；缩短窗口会先预览影响并要求管理员确认。确认后先持久化新策略，再按 500 行分批删除；中途失败保留新策略并报告清理待继续，后续每小时清理可幂等完成剩余数据，采集关闭也不阻止保留策略生效。
- 常规备份排除 `chat-audit.db`。SQLite Stats 备份副本会删除 outbox 并清空 Session IP，Dashboard 副本会清空百度 AK、GeoIP cache secret 和依赖旧 secret 的缓存；所有清理只发生在在线快照，live DB 不变，恢复启动时重新生成 secret。诊断包不含聊天正文或原始聊天数据库。MySQL/PostgreSQL 原生备份可能包含 `lps_sessions.ip_address` 与 `lps_chat_outbox`，管理员必须按隐私策略排除、加密或单独到期。

## GeoIP 生命周期与隐私

- GeoIP 没有独立启用开关，只支持管理员配置的百度普通 IP 定位 AK；没有 AK 就不会请求 provider，Collector 始终不发 HTTP 请求。
- 仅公网 IP 会在缓存未命中时进入 256 条有界队列；管理员可设置 1-3 QPS（默认 2），后台解析与测试请求共用同一无突发限速器。私有、回环、链路本地、文档保留地址不会发给 provider。
- 成功结果默认缓存 30 天；网络或 provider 错误只缓存 1 小时。结果是近似城市级位置，不代表精确玩家位置，海外/IPv6 能力由 provider 与账户决定。
- 过期缓存由后台按固定周期、固定批次数量清理；相同 IP 在有效缓存期内直接复用，队列按 HMAC 键去重，不会并发重复消耗额度。
- 连接审计没有位置条件时只读取一个原始 keyset page；有位置条件时按每批最多 200 条、单次最多 2,000 条连续扫描，返回游标始终指向最后实际扫描的原始连接行。预算耗尽但仍有原始数据时继续返回下一页游标；缓存未命中只排入现有异步队列，不同步调用 provider，并提示管理员部分位置仍在解析、稍后刷新。
- 原始 IP、位置与 provider 状态只出现在管理员连接审计中，不进入公开 API、个人页、排行榜或 `/ingame`；AK 只以掩码形式返回管理前端，也不进入日志、备份诊断或错误文本。

“数据增长监控”页展示 Stats DB、Dashboard DB/WAL、Chat Audit DB/WAL、轮转日志、聚合状态、聊天数量/窗口/摄取滞后/缺口/丢弃计数、GeoIP 缓存/队列/状态、候选清理数量和历史清理次数，用于决定是否缩短保留期或安排数据库维护窗口。
