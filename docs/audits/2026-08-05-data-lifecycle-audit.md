# Dashboard 数据生命周期与有界性审计

> 审计日期：2026-08-05  
> 范围：SourceMod 采集器保存节奏、Dashboard 聚合、缓存、A2S、日志和数据保留  
> 结论：未发现 P0/P1 问题；清理执行尚未实装，长期增长与全量重建为主要运维风险。

## 执行摘要

- SourceMod 默认每 `300s + 0–60s` 保存一次，并在关卡、回合、Session、Segment 等关键生命周期主动请求保存。
- 数据库不可用时的内存队列有硬上限；超限后会丢弃新的已关闭快照，不会无限占用服务器内存。
- Dashboard 启动时立即构建聚合，之后每 10 分钟全量重建一次；单次最长 3 分钟，同一时刻只允许一次重建。
- Dashboard 聚合快照在一个 SQLite 事务里整体替换，失败会保留上一份完整快照。
- `retention plan` 仅预估可清理行，`deletion_enabled` 始终为 `false`；当前没有定时清理、`retention apply`、自动 `VACUUM` 或历史压缩。
- Go 应用日志由 Lumberjack 轮转，默认单文件 50 MB、最多 10 份备份、最长 30 天、压缩保存。

## 严重性统计

| 等级 | 数量 |
|---|---:|
| P0 | 0 |
| P1 | 0 |
| P2 | 3 |
| P3 | 3 |

## 发现

### P2-001：Stats DB 和 Dashboard 日聚合尚无实际清理路径

- **状态**：Open（已在 roadmap 延后）
- **证据**：`dashboard/internal/cli/root.go` 中只有 `retention plan`，且强制 `DeletionEnabled = false`；`docs/roadmap.md` 将执行清理安排在 v0.9.2。
- **影响**：`lps_sessions`、Run、Round、Segment、PvE/Versus 细分表会随游戏时间持续增长；`aggregate_rows` 也会随“天数 × 玩家 × 服务器 × 维度”增长。
- **建议**：按 v0.9.2 实装水位、校验清单、小批事务和显式确认；在此之前将 Stats DB 视为永久增长的事实库，监控磁盘并定期整库备份。

### P2-002：每 10 分钟全量重建所有历史聚合

- **状态**：Open
- **证据**：`dashboard/internal/service/aggregate.go` 固定每 10 分钟调用 `AggregateRows`；`dashboard/internal/store/stats_aggregate.go` 遍历全部聚合步骤；服务端定时重建超时为 3 分钟。
- **影响**：成本与全历史行数成长；长期运行后可能超过 3 分钟，导致后台持续保留旧快照，排行榜不再更新。
- **建议**：改为按 `last_saved_at + revision + source ID` 的增量水位重算受影响日桶，保留手动全量重建仅用于恢复和对账。

### P2-003：排行榜和个人查询缓存不是严格的条目上限

- **状态**：Open
- **证据**：`RankingService` 只在条目超过 128 时删除已过期项，然后仍写入新项；`PlayerService` 限制的是 256 个 SteamID 拥有者，不是缓存键总数。
- **影响**：在 60 秒 TTL 内制造大量不同筛选组合时，内存可以短时间突增；服务器筛选值若不限定为已登记 key，还会扩大键空间。
- **建议**：将两者改为硬上限 LRU/TTL 缓存，对排行榜限制总条目数，对个人查询同时限制玩家数和总键数，并将 `server_key` 校验为已知服务器。

### P3-001：A2S 缓存不会清除已删除或改址址的服务器键

- **状态**：Open
- **证据**：`dashboard/internal/a2s/provider.go` 的 `entries` 以 `UUID + address` 为键，只有写入和读取，无定期清理或服务器删除钩子。
- **影响**：管理员反复删除服务器、重建 UUID 或修改地址时，旧缓存留到进程重启。该操作频率低，所以风险较低。
- **建议**：在列出服务器后顺便删除不在当前 UUID/address 集合中的缓存键，或在 CRUD 成功后显式失效。

### P3-002：SourceMod 日志目录的长期保留不由本插件管理

- **状态**：Accepted risk / 运维配置
- **证据**：采集器使用 SourceMod `LogMessage`/`LogError`；`logging.inc` 仅对同类数据库错误做默认 300 秒抑制，无权删除 SourceMod 全局日志。
- **影响**：插件自身不会高频写正常事件，但 `addons/sourcemod/logs` 中所有插件和 SourceMod 的旧日志是否删除，取决于服务器运维。
- **建议**：在部署文档中加入 SourceMod 日志保留任务，例如保留 30–90 天；不要让插件删除整个共享日志目录。

### P3-003：聚合状态中的 `source_rows` 实际填入了输出聚合行数

- **状态**：Open
- **证据**：`dashboard/internal/service/aggregate.go` 在完成 `AggregateRows` 后以 `int64(len(rows))` 作为 `sourceRows` 传入 `ReplaceAggregateRows`，因此它与 `aggregate_rows` 相同，不是 Stats DB 被扫描的原始行数。
- **影响**：`aggregate status` 的运维信息容易被误读，未来不能用该字段判断源数据覆盖率或清理安全性。
- **建议**：要么由聚合层返回真实源行计数，要么在增量水位实装前将字段明确重命名为 `output_rows`。

## 已经有界或不会持久增长的部分

| 系统 | 当前约定 | 有界性 |
|---|---|---|
| 采集定时保存 | 默认 300 秒 + 0–60 秒随机扰动 | 单定时器，不重入；飞行中再次请求只合并为一次 queued flush |
| 采集断线重连 | 5/15/30/60 秒退避，上限可配，默认 60 秒 | 常数状态 |
| 采集离线队列 | Session 256；Run/Round/Segment 合计 512；装备 4096 | 硬上限；超限告警并丢弃追加快照 |
| 概览缓存 | 全局单份、60 秒、singleflight | 严格有界 |
| 个人缓存 | 60 秒，最多 256 个 SteamID 拥有者 | 玩家数有界，但键总数不是硬上限 |
| 排行榜缓存 | 60 秒、singleflight，128 后尝试清过期项 | 软限制，不是硬上限 |
| A2S | 2 秒/次，最多 4 并发；刷新 5–60 秒；扰动 0/2/5 秒；失败重试 1–3 次；旧成功结果最多回退 5 分钟 | 单服务器快照有界，历史服务器键未清理 |
| 登录/初始化限流 | 15 分钟窗口；登录 5 次/键、最多 1024 键；初始化 10 次/键、最多 256 键 | 硬容量，满后清过期并驱逐一项 |
| Go 应用日志 | 50 MB/文件、10 备份、30 天、压缩 | 应用文件有界；`also_console` 副本由 journald 保留策略决定 |
| monitor | 默认 2 秒刷新、60 样本窗口，仅管理员可访问 | 进程内短窗口，无持久历史库 |
| 页脚/站点文档 | 页脚最多 20 条；站点文档固定 3 份，单份最多 100 KiB | 有界 |
| 公告 | 单条 Markdown 最多 100 KiB，数量无上限 | 管理员控制的持久增长 |

## 优先级建议

1. **先做可观测性**：监控 Stats DB、Dashboard DB/WAL、SourceMod 日志和 Go 日志目录大小；对聚合持续失败或快照超时告警。
2. **再做增量聚合**：这是长期性能的首要工作，比直接删数据更安全。
3. **然后收紧内存缓存**：改成硬上限 LRU，并限定可用的服务器筛选键。
4. **最后启用清理**：仅在水位、对账、清单、备份和 dry-run 均可验收后，再开放显式 `retention apply`。
