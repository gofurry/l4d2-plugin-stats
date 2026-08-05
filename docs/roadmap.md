# Roadmap

## Current Position

SourceMod 采集器的核心统计契约已经冻结，能够记录身份与 Session、PvE、装备、团队行为、技巧、Versus 双阵营职业数据和比赛结果。Dashboard 不修改这些采集表，也不执行采集器迁移；正常运行时 Stats DB 始终只读。

Dashboard 已完成 Go + 内嵌 React 地基、公开首页、多服务器 A2S、Steam OpenID、个人查询、单管理员后台、管理员运行监控、完整个人统计、全服排行榜和第一版日聚合读模型。当前开发版本为 `v0.9.1-dev`，下一阶段不再扩张统计口径，而是验证聚合正确性、完成安全保留工具并进行真实部署加固。

## Roadmap Strategy

1. 采集器数据库是唯一事实来源；Dashboard 聚合表必须可以完整重建。
2. 公开页面只读取统计字段，永远不返回 IP、boot ID、内部主键或数据库连接信息。
3. PvE（coop/realism）与 Versus 完全分开；Versus 的幸存者和感染者继续分开。
4. 趋势按 Session 或 Segment 的开始日期归属，以 UTC 天为聚合桶；这不是逐事件发生时间。
5. 排行榜只接受后端白名单指标，不允许客户端传入 SQL 字段或排序表达式。
6. 效率榜必须设置最低样本门槛；首版 PvE 默认为 5 小时，Versus 默认为 3 小时。
7. 身份、Run、Round、Segment 和核心统计永久保留；过期装备/职业明细、已关闭 Session 和比赛结果只有在聚合覆盖、预览复核、审计和管理员显式确认后才能清理。
8. `v1.0.0` 只用于首个真正稳定、可长期部署的版本；之前继续使用 `v0.x.x` 和稳定冻结 Alpha。

## Version Plan

### v0.8.0–v0.8.3 - Dashboard Foundation

**Status:** Implemented

**Scope:** Architecture / Backend / Frontend / Authentication

**Goal:** 建立单二进制 Dashboard、公开首页、个人身份查询与单管理员后台。

**Focus:** 工程边界、只读 Stats DB、A2S、Steam OpenID、JWT 管理后台。

#### Tasks

- [x] Go 1.26、Fiber v3、Cobra、Zap/Lumberjack 和 systemd 工具链
- [x] React 19、TypeScript、Vite、Ant Design、Tailwind、SCSS Modules 和内嵌构建
- [x] SQLite/MySQL/PostgreSQL Stats DB 只读适配与 sqlc 查询
- [x] Dashboard SQLite、Goose 迁移、首次设置、单管理员 JWT 和 CSRF
- [x] 多服务器 A2S 状态、玩家列表、服务器规则与后台服务器管理
- [x] Steam OpenID、手动 SteamID64、基础个人中心和管理员运行监控

#### Acceptance Criteria

- 单二进制可提供 React SPA 和 `/api/v1`，API 404 不回退 HTML
- Stats DB 使用只读账号即可运行，Dashboard DB 与采集器数据边界独立
- 未认证访客无法访问管理员 API 或运行监控

---

### v0.9.0 - Detailed Player Center and Leaderboards

**Status:** Implemented; awaiting visual and real-data review

**Scope:** Backend / Frontend / API

**Goal:** 将冻结的 PvE 和 Versus 契约完整转化为可浏览、可核对的个人统计与全服排行。

**Focus:** ECharts、精确明细、玩法隔离、筛选、排行白名单和有界查询。

#### Tasks

- [x] 扩展个人 API，返回 PvE 战斗、生存、救援、治疗、章节、技巧和互动全量指标
- [x] 返回 40 个固定装备桶及动作、击杀、爆头和 Boss/特感伤害
- [x] 返回 Versus 幸存者总计、职业击杀/伤害，以及感染者职业控制、能力、击倒和击杀
- [x] 增加玩家日活跃趋势、连接/实际操作时长和服务器分布
- [x] 重构个人中心，使用 ECharts 展示趋势与职业结构，并保留紧凑精确表格
- [x] 增加 `/rankings` 页面、Top 10 图表、分页表格和玩家详情跳转
- [x] 增加时间范围、服务器、PvE 模式和最低有效时长筛选
- [x] 将活跃、PvE、Versus 幸存者和 Versus 感染者排行榜彻底分开
- [x] 为每小时效率榜设置默认样本门槛

#### Acceptance Criteria

- 每个展示字段都能追溯到冻结的 `stats_version = 1` 表字段或公开聚合公式
- coop/realism 不与 Versus 混合，Versus 双阵营不混合
- 排行指标来自固定白名单，分页上限为 100，缓存为 60 秒
- 图表最多展示有界时间点/Top 10，完整数值仍可在表格中核对

---

### v0.9.1 - Rebuildable Daily Aggregate and Retention Planning

**Status:** Implemented; deletion deliberately disabled

**Scope:** Data / Operations / Performance

**Goal:** 将高频公开查询从原始 Segment 明细迁移到可重建的日聚合，并建立不会误删数据的保留边界。

**Focus:** Dashboard DB 日聚合、定期刷新、重建、状态、保留预估。

#### Tasks

- [x] 在 Dashboard DB 建立按 UTC 日、服务器、玩家、模式和维度索引的聚合读模型
- [x] 聚合 Session 活跃、PvE 全字段、装备、Versus 幸存者/感染者及职业明细
- [x] 启动后及每 10 分钟事务性重建聚合快照，失败时保留上一份完整快照
- [x] 增加 `aggregate status` 和 `aggregate rebuild` 运维命令
- [x] 增加 `retention plan`，报告 180 天职业/装备明细和 365 天 Segment 候选量
- [x] 保持 Stats DB 只读；当前不提供删除命令，也不自动 VACUUM
- [x] 使用真实 SQLite 采集库完成聚合、排行和个人全量接口冒烟验证

#### Acceptance Criteria

- 聚合快照可从 Stats DB 完整重建，Dashboard DB 不是唯一事实来源
- 重建在单事务内替换，访客不会读取半份快照
- `retention plan` 始终返回 `deletion_enabled = false`
- 普通网页服务进程没有 Stats DB 写权限

---

### v0.9.2 - Incremental Aggregation and Safe Retention Apply

**Status:** Implemented; awaiting long-running real-server validation

**Scope:** Data / Reliability / CLI

**Goal:** 避免长期运行后每 10 分钟全表重建，并在可证明安全后提供显式、可审计的分批清理。

**Focus:** revision/watermark、受影响桶重算、校验清单、维护窗口。

#### Tasks

- [x] 使用 `last_saved_at` 水位，只重算受影响的 UTC 日桶
- [x] 聚合周期由 Dashboard DB 配置，默认 30 分钟并支持 15 分钟至 24 小时
- [x] 为每次聚合记录源水位、变更日期、行数、耗时和失败原因
- [x] 使用独立维护连接；常规网页、公开 API 和聚合仍只读 Stats DB
- [x] 管理后台提供数据增长监控、清理预览、二次确认和立即聚合
- [x] 清理过期装备/职业明细、已关闭 Session 和 Versus 比赛结果
- [x] 清理前强制刷新聚合并两次校验水位与预览 ID
- [x] 写入清理审计，记录范围、水位和实际删除数量
- [ ] 为三种数据库增加大数据量小批事务和真实运行验收
- [ ] 增加聚合与原始表按玩家/日期/指标抽样对账命令
- [x] 增加月度/终身汇总层，累计统计和全周期排行榜不再扫描全部日聚合

#### Acceptance Criteria

- 活跃 Segment 多次保存不会被重复累加
- 增量结果与同一水位的全量重建逐指标一致
- 未完成聚合、校验失败或水位落后时清理命令拒绝执行
- 身份、Run、Round、Segment 和核心统计不进入删除范围
- 清理后首页、个人累计和排行榜继续读取日聚合，旧逐条 Session/比赛结果按策略消失

---

### v0.9.3 - Multi-database and Production Validation

**Status:** Planned

**Scope:** Testing / Operations / Security / UX

**Goal:** 在真实服务器和三种数据库上验证长期运行、性能、隐私和界面质量。

**Focus:** MySQL/PostgreSQL、查询计划、并发、故障恢复和真实玩家反馈。

#### Tasks

- [ ] 使用真实 MySQL 与 PostgreSQL 完成全量个人查询、聚合、排行和保留预估验收
- [ ] 使用中小型与大型服务器数据量回放查询并记录 P50/P95/P99
- [ ] 验证聚合刷新期间的并发读取、Stats DB 断线、Dashboard DB 锁等待和进程中断恢复
- [ ] 为聚合状态、查询耗时、缓存命中率和数据库连接池补充管理员监控指标
- [ ] 完成个人中心与排行榜桌面/移动端视觉检查、空数据和部分失败状态
- [ ] 检查公开 API 不泄露 IP、内部 ID、DSN、JWT 或管理员信息
- [ ] 完成反向代理、HTTPS、备份恢复、升级和回滚文档

#### Acceptance Criteria

- 三种数据库在相同冻结夹具上返回等价统计
- 常用个人页和排行榜查询有明确、可重复的性能上限
- 数据库或 A2S 局部故障不会拖垮无关页面
- 没有无限增长的应用日志、缓存或后台认证状态

---

### v1.0.0-alpha.1 - Stability Freeze

**Status:** Planned

**Scope:** Release / Compatibility / Documentation

**Goal:** 冻结首个公开版本的 API、配置、聚合口径和部署方式。

**Focus:** 兼容性、真实部署回归、发布材料。

#### Tasks

- [ ] 冻结公开 API v1 DTO、错误码、筛选与分页约定
- [ ] 冻结 Dashboard DB 升级策略和 Stats DB schema v1 兼容边界
- [ ] 完成 Steam OpenID、管理员 JWT/CSRF、monitor 权限和反向代理安全回归
- [ ] 完成隐私说明、数据保留策略、安装、升级、备份和故障排查文档
- [ ] 收集至少一轮真实综合服务器运行反馈并关闭高优先级问题

#### Acceptance Criteria

- 没有已知的高优先级数据误读、越权、敏感信息泄露或资源无限增长问题
- 从干净仓库可以生成可部署的单二进制与最小配置
- Alpha 后任何破坏性 API、配置或聚合变化都进入兼容性审查

---

### v1.0.0 - First Stable Release

**Status:** Planned

**Scope:** Stable Release

**Goal:** 发布第一版可长期部署的 L4D2 Stats 采集与展示系统。

**Focus:** 稳定性、可运维性、文档和发布完整性。

#### Tasks

- [ ] 发布采集器、Dashboard 单二进制、最小配置和校验值
- [ ] 发布完整变更日志、安装、升级、回滚和备份说明
- [ ] 建立稳定分支与后续兼容性政策

#### Acceptance Criteria

- SQLite、MySQL、PostgreSQL 均有完成验收的部署路径
- 首页、个人中心、排行榜、Steam OpenID、后台、A2S 和运行监控通过真实环境验收
- 聚合和保留工具可审计、可恢复且不会在网页进程内自动删除原始数据

## Deferred Beyond v1

- 玩家账号、角色、密码、跨服务器账号体系和公开写入 API
- 多管理员、细粒度 RBAC 和组织空间
- 浏览器直接 A2S、任意访客地址探测和 UDP 代理
- 逐事件流水、比赛回放和逐时刻推进曲线
- 跨 Run 稳定队伍身份与个人胜率（当前采集契约不足以可靠表达）
- 多实例 Dashboard 的分布式聚合锁、跨实例缓存和密钥轮换协调
