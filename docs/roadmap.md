# Roadmap

## Current Position

当前版本为 `v1.3.4`。SourceMod 采集器使用 Stats schema 6 / `stats_version = 1`，支持 SQLite、MySQL 和 PostgreSQL；Dashboard 使用 schema 17，Aggregate Contract 与 Achievement Contract 均为 v1。Achievement Catalog 现包含 105 个底层项目和 38 个 artwork key。详细版本变化统一记录在 [`CHANGELOG.md`](../CHANGELOG.md)。

系统已经具备身份与生命周期、PvE/装备、Versus 双阵营/职业和比赛结果采集，以及 Assist、真人定向互动关系、Round Context、低频 Incident、自动成就与徽章、地图/Boss/玩家分析、公开首页、个人中心、排行榜、Steam OpenID、单管理员后台、多服务器 A2S、原生 MOTD 游戏内轻量页面、日/月/终身聚合、安全保留、深度数据检查、备份恢复和脱敏诊断导出。

## Compatibility Principles

1. Stats DB 是采集事实来源；Dashboard 不迁移 Stats DB，正常查询和聚合保持只读。
2. PvE（`coop`/`realism`）与 Versus 完全分开，Versus 幸存者与感染者分别统计。
3. Stats schema、`stats_version` 或 Aggregate Contract 的未知版本必须明确拒绝，不能静默解释或自动升级。
4. 公开页面与诊断输出不得泄露 Session IP、DSN、管理员密钥、JWT、Cookie 或内部敏感标识。
5. 清理必须经过聚合覆盖、契约版本、水位、预览 ID、管理员确认和审计校验；身份、Run、Round、Segment 与核心统计永久保留。
6. 破坏性 API、配置、数据库或统计语义变化必须提供兼容策略、迁移说明和回滚边界。

## Near-term Priorities

- 在首次推送后运行 SQLite、MySQL 8.4 和 PostgreSQL 17 数据库契约矩阵，并处理真实方言差异。
- 使用中型和大型真实服务器数据评估个人页、排行榜、聚合与 deep doctor 的 P50/P95/P99。
- 测量 2048 Incident 队列恢复、256 条批量 SQL 构造、SQLite 写锁时长和真实 Incident 行增长。
- 验证长时间运行时的断线恢复、Dashboard DB 锁等待、并发聚合读取和原始数据分批清理。
- 完成桌面与移动端真实数据视觉回归、空数据和局部故障状态检查。
- 持续审计公开 API、日志、备份和诊断包的隐私边界。

## Medium-term Priorities

- 为聚合与原始表增加按玩家、日期和指标的抽样对账工具。
- 增强管理员监控中的聚合耗时、查询延迟、缓存命中率和数据库连接池可见性。
- 根据真实运行数据评估永久增长的核心表、索引和 SQLite 维护窗口。
- 在不改变 Stats v1 语义的前提下改善可访问性、国际化和运维体验。

## Deferred

- 玩家账号密码体系、多管理员和细粒度 RBAC。
- 微服务、队列、分布式缓存或多实例聚合锁。
- 高频事件流水、比赛回放、轨迹/逐时刻推进曲线、伪坐标热力图和当前契约无法可靠表达的个人胜率。
- Aggregate Contract v2、Context 维度排行榜、综合技能分和完整社交关系图。
- 未经独立契约设计和迁移方案的新玩法统计或 Aggregate Contract v2。
