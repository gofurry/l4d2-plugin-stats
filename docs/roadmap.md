# Roadmap

## Current Position

项目已经完成数据库地基、真人身份/Session、Run/Round/Segment 生命周期和 v0.4.0 PvE 核心战斗统计的本地验收。v0.5.0 的治疗、临时生命和章节/战役成绩已经完成实现，正在进行本地玩法数据验收。SQLite 已在真实 L4D2/SourceMod 环境中进入 `ready`；完整多人对抗、MySQL 和 PostgreSQL 实机兼容验证保留到 v0.7.0。

当前已采集 PvE 击杀、有效伤害、承伤、友伤、基础救援、治疗、临时生命和章节/战役成绩。更多 PvE 扩展字段将在 v0.5.x 讨论并冻结契约后再排期；Versus 仍使用后续独立统计模型。SourcePawn 采集器、数据库结构和未来 Go 服务的公共边界仍处于 pre-v1 阶段，可以按验证结果调整。

## Roadmap Strategy

优先完成会影响所有后续统计正确性的生命周期和持久化地基，再接入 PvE 与对抗玩法统计。Go 后端只在采集插件稳定后开始，并作为数据库只读消费者；迁移历史始终由 SourceMod 采集器拥有。`v1.0.0` 保留给第一版可稳定部署、可升级且有完整运维文档的正式版本。

## Version Plan

### v0.1.0 - Database Foundation

**Status:** Completed

**Scope:** Architecture / Stability / Developer-facing / Documentation

**Goal:** 建立可安全扩展的三数据库持久化地基。

#### Focus

- 异步数据库连接和迁移
- 有界重连与日志
- 构建、部署和打包

#### Tasks

- [x] 支持 SQLite、MySQL 和 PostgreSQL 驱动识别
- [x] 建立 0001 等价迁移、11 张表和必要索引
- [x] 实现服务器注册、`boot_id`、心跳和旧启动恢复
- [x] 实现 5/15/30/60 秒有上限重连和错误日志限流
- [x] 添加构建、部署、迁移校验和 SQLite 集成测试

#### Acceptance Criteria

- SourcePawn 编译无错误和警告
- SQLite 真实服务器加载状态为 `ready`、schema `1/1`
- 发布包同时包含 `.smx` 和三套运行时迁移
- 数据库故障不会踢玩家或创建无限本地日志

---

### v0.2.0 - Player Identity and Session

**Status:** Completed (local validation scope)

**Scope:** Architecture / Stability / Security / Testing

**Goal:** 在受支持模式中可靠保存真人身份和一次连续连接的 Session。

#### Focus

- SteamID64 真人身份
- Session 生命周期与隐私边界
- 连接时间和有效操作时间
- 绝对快照与断线队列

#### Tasks

- [x] 仅在 `coop`、`realism`、`versus` 中创建真人记录
- [x] 保存 SteamID64、最近昵称和不含端口的 IPv4/IPv6
- [x] 累计 `connected_seconds` 和 `active_play_seconds`
- [x] 支持中途加载、按 SteamID 跨图续接、真实断线和模式切换结束
- [x] 使用单一刷新事务保存 active Session 和有界 closed Session 队列
- [x] 数据库恢复后补写当前绝对快照，不使用数据库增量
- [x] 增加管理员状态信息与 SQLite Session 集成测试
- [x] 完成本地观战、闲置、跨图和模式切换验收
- [ ] 在独立服务器或多人服务器验证远端真人断线与重连（转移至 v0.7.0）

#### Acceptance Criteria

- Bot、未认证玩家和不支持模式不会创建玩家或 Session
- Session 可跨正常地图切换；listen server 换图产生的临时重连仍沿用同一 Session
- 非换图期间的真实断线会关闭 Session，再次连接创建新 Session
- 观战和闲置只增加连接时间，不增加有效操作时间
- 本地可验证的模式切换和周期刷新能够持久化
- 正常日志和管理员状态不输出 IP

#### Notes

本地 listen server 的房主断线会同时终止服务器，不能等价验证“远端真人断线但服务器继续运行”。换图时按 SteamID 暂存 Session 120 秒，成功重连沿用原 `session_id`；超时则以 `map_reconnect_timeout` 关闭。真实多人时序验收保留为 v0.7.0 发布加固任务。

---

### v0.3.0 - Run, Round, and Segment Lifecycle

**Status:** Completed (local validation scope)

**Scope:** Architecture / Correctness / Testing

**Goal:** 为章节、战役、对抗半场和玩家参赛身份建立可靠归属。

#### Focus

- PvE 与 Versus Run
- 章节尝试和对抗半场 Round
- 真人阵营与闲置 Segment

#### Tasks

- [x] 实现 PvE 战役推进、团灭重试和结局完成语义
- [x] 实现 Versus 两个半场及半场重开语义
- [x] 处理手动换图、模式切换、插件重载和异常恢复
- [x] 处理阵营切换、观战、闲置和重新接管 Segment
- [x] 为生命周期状态机增加固定事件回放测试清单
- [x] 完成 PvE 通关、团灭重试、手动换图和本地对抗首半场验收
- [x] 复测修复后的跨图 Session，确认一次连续连接只保留一个 `session_id`

#### Acceptance Criteria

- 每个统计对象都能归属到确定的 Session、Run、Round 和 Segment
- PvE 团灭不会错误创建新 Run
- Versus 两个半场不会合并成同一个 Round
- 不正常换图留下 `abandoned`，不伪装为正常完成

---

### v0.4.0 - PvE Core Statistics

**Status:** Completed (local validation scope)

**Scope:** User-facing / Correctness / Testing

**Goal:** 按契约采集 coop 与 realism 的第一批核心统计。

#### Focus

- 击杀与有效伤害
- 承伤和真人/Bot 友伤拆分
- 倒地、死亡和基础救援

#### Tasks

- [x] 采集普通感染者、特感、Tank 和 Witch 最后击杀
- [x] 采集对特感、Tank 和 Witch 的实际生命损失
- [x] 采集感染者承伤与三类友伤
- [x] 采集倒地、死亡、倒地救援、挂边救援和电击器复活
- [x] 在代码与 SQLite 集成测试中验证数据只写入 PvE Segment 统计表
- [x] 在真实本地 `coop` 对局中核对首批统计数值，并确认 `realism` 复用相同 PvE 采集路径

#### Acceptance Criteria

- 不记录溢出伤害、自伤和无法可靠归属的环境伤害
- coop 与 realism 可分别过滤但能组成 PvE 总览
- Bot 不拥有个人统计记录

---

### v0.5.0 - Healing and Chapter Results

**Status:** Implementation complete; local gameplay validation pending

**Scope:** User-facing / Correctness / Documentation

**Goal:** 补齐 PvE 治疗、临时生命和章节/战役成绩。

#### Tasks

- [x] 拆分医疗包自疗、他疗及实际真实生命恢复
- [x] 统计止痛药、肾上腺素和实际临时生命
- [x] 记录章节参与、完成时存活状态和战役完成
- [x] 添加事件字段和特殊边界测试文档
- [ ] 在真实本地 PvE 对局中核对治疗、临时生命和章节/战役数值

#### Acceptance Criteria

- 医疗包真实生命与临时生命不混为同一指标
- 中途加入者不会获得加入前的章节数据
- 闲置、观战和 Bot 不获得个人章节完成记录

---

### v0.5.x - PvE Statistics Expansion

**Status:** Deferred (scope discussion pending)

**Scope:** User-facing / Architecture / Testing

**Goal:** 在不提前锁定字段的前提下，讨论并筛选下一批有可靠事件来源和明确展示价值的 PvE 数据扩展。

#### Focus

- PvE 扩展指标的使用场景和展示价值
- 事件来源、可归属性和作弊/三方图干扰风险
- 是否需要提高 `stats_version` 或增加数据库迁移

#### Tasks

- [ ] 讨论候选指标及精确定义
- [ ] 核对 SourceMod/L4D2 事件和可测试边界
- [ ] 确认数据库兼容方案与聚合方式
- [ ] 契约确认后再拆分为正式 patch 版本排期

#### Acceptance Criteria

- 每个进入排期的字段都有明确所有者、统计时机和排除规则
- 不把无法可靠归属的推测数据写入永久明细
- 在讨论完成前不改变 schema、`stats_version` 或发布计划

#### Notes

本节仅是讨论入口，不代表已经承诺具体 v0.5.x 发布内容。

---

### v0.6.0 - Versus Statistics

**Status:** Planned

**Scope:** User-facing / Architecture / Testing

**Goal:** 在独立模型中采集对抗幸存者和感染者统计。

#### Tasks

- [ ] 拆分真人/Bot 特感和 Tank 击杀与伤害
- [ ] 复用但隔离对抗幸存者生存、救援和治疗统计
- [ ] 采集感染者出生、伤害、倒地和击杀
- [ ] 完成半场切换、换队和中途加入测试

#### Acceptance Criteria

- Versus 数据不会进入 PvE 排行或聚合
- 幸存者和感染者统计写入不同表
- 第一阶段不推算稳定队伍、推进分、胜负或 MVP

---

### v0.7.0 - Multi-database and Release Hardening

**Status:** Planned

**Scope:** Stability / Performance / CI / Release

**Goal:** 在开始网页侧之前证明采集器可长期运行和安全升级。

#### Tasks

- [ ] 在真实 MySQL 与 PostgreSQL 测试迁移、重连和 upsert
- [ ] 在独立服务器或多人服务器验证远端真人断线、重连及数据库故障期间断线补写
- [ ] 使用双方真人完成 Versus 两个半场、换队、半场重开和中途退出验收
- [ ] 增加 SQLite/MySQL/PostgreSQL 自动兼容测试
- [ ] 验证长时间运行、队列上限、事务大小和地图切换压力
- [ ] 建立 GitHub Actions 编译、迁移校验和发布产物流程
- [ ] 完成数据库最小权限、升级和回滚文档

#### Acceptance Criteria

- 三种数据库通过相同场景验收
- 24 小时压力测试无无限内存或磁盘增长
- CI 可从干净仓库生成可安装包

---

### v0.8.0 - Read-only Go API Foundation

**Status:** Planned

**Scope:** Architecture / Security / Developer-facing

**Goal:** 建立只读 Go 服务、查询边界和内嵌前端地基。

#### Tasks

- [ ] 初始化 Go 模块、配置、健康检查和数据库只读连接
- [ ] 定义不暴露 IP 的内部查询与公开 DTO
- [ ] 实现服务器、玩家、Session 和基础统计查询
- [ ] 建立 Go 单元测试与三数据库查询兼容测试
- [ ] 初始化可嵌入 Go 二进制的前端构建流程

#### Acceptance Criteria

- Go 服务不执行迁移和写入采集表
- 未授权公开响应不包含 IP
- 单一二进制可以提供 API 和静态前端

---

### v0.9.0 - Dashboard and Operations

**Status:** Planned

**Scope:** User-facing / Security / Documentation / Release

**Goal:** 提供可部署的统计浏览体验和管理边界。

#### Tasks

- [ ] 实现服务器、模式、玩家和章节筛选页面
- [ ] 明确 PvE 与 Versus 展示和排行边界
- [ ] 增加分页、缓存、查询超时和大数据量验证
- [ ] 增加私有管理接口的身份验证和 IP 查询审计
- [ ] 完成备份、恢复、升级和隐私说明

#### Acceptance Criteria

- 公开页面默认不显示敏感数据
- 大数据量查询有分页和明确上限
- 服务器管理员可以按文档完成部署和恢复

---

### v1.0.0-alpha.1 - Stability Freeze

**Status:** Planned

**Scope:** Stability / Testing / Documentation / Release

**Goal:** 冻结第一版架构候选并进行真实服务器反馈测试。

#### Tasks

- [ ] 冻结采集口径、数据库兼容边界和 Go 公共 API 候选
- [ ] 完成多人、长战役、对抗和数据库故障回归
- [ ] 完成安装、升级、隐私、备份和故障排查文档
- [ ] 收集至少一轮外部服务器反馈

#### Acceptance Criteria

- 没有已知会破坏数据正确性的 blocker
- 升级路径在三种数据库上可重复验证
- API 和架构变化进入兼容性审查

---

### v1.0.0 - First Stable Release

**Status:** Planned

**Scope:** User-facing / Stability / Security / Release

**Goal:** 发布第一版可长期部署、可升级和有明确支持边界的稳定系统。

#### Acceptance Criteria

- SourceMod 采集器、迁移、Go API 和前端均有稳定版本边界
- 三种数据库、支持模式和核心生命周期通过回归测试
- 发布包、校验值、变更日志和升级说明完整
- 没有已知高优先级数据损坏、隐私泄露或无限资源增长问题

## Deferred Beyond v1

- 原始事件流水和逐事件回放
- 助攻、控制链及高级技术动作
- 对抗稳定队伍归属、推进分、最终胜负和 MVP
- 可靠聚合表完成前的 Session/Segment 明细删除
- 跨服务器账号系统和公开写入 API
