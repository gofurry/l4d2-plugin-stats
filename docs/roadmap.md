# Roadmap

## Current Position

项目已经完成数据库地基、真人身份/Session、Run/Round/Segment 生命周期，以及 v0.4.0～v0.5.3 PvE 统计的本地验收。v0.6.0 对抗核心统计、v0.6.2 感染者职业明细和 v0.6.3 控制与能力效果已经通过本地验证；v0.6.4 幸存者战斗明细完成实现和离线验证，等待本地数据验收。完整双方真人对抗、MySQL 和 PostgreSQL 实机兼容验证统一保留到 v0.7.0。

当前已采集 PvE 击杀、有效伤害、承伤、友伤、救援、治疗、临时生命、章节/战役成绩、设备、控制、技巧、目标互动和失能时长。Versus 使用独立统计模型，首版已覆盖幸存者职业战斗、Witch、官方消耗品、技巧、生存/救援/治疗，以及感染者职业出生、伤害、倒地、击杀、控制和能力效果。SourcePawn 采集器、数据库结构和未来 Go 服务的公共边界仍处于 pre-v1 阶段，可以按真实服务器验证结果调整。

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

**Status:** Completed (local validation scope)

**Scope:** User-facing / Correctness / Documentation

**Goal:** 补齐 PvE 治疗、临时生命和章节/战役成绩。

#### Tasks

- [x] 拆分医疗包自疗、他疗及实际真实生命恢复
- [x] 统计止痛药、肾上腺素和实际临时生命
- [x] 记录章节参与、完成时存活状态和战役完成
- [x] 添加事件字段和特殊边界测试文档
- [x] 在真实本地 PvE 对局中核对治疗、临时生命和章节/战役数值

#### Acceptance Criteria

- 医疗包真实生命与临时生命不混为同一指标
- 中途加入者不会获得加入前的章节数据
- 闲置、观战和 Bot 不获得个人章节完成记录

---

### v0.5.1 - PvE Statistics Expansion

**Status:** Completed (local validation scope)

**Scope:** User-facing / Architecture / Testing

**Goal:** 在固定内存与数据库边界内补齐可可靠归属的 PvE 设备、控制、参与和技巧统计。

#### Focus

- 固定 ID 的官方设备明细与 `Other Firearm`
- 特感职业、控制/解救和 Boss 参与
- 可验证技巧与补给部署

#### Tasks

- [x] 按六种普通特感拆分击杀和有效伤害
- [x] 记录官方枪械、官方近战和官方投掷物的固定设备行
- [x] 将未知/第三方枪械统一归入单一 `Other Firearm`，忽略自定义近战和投掷物
- [x] 记录四种控制次数、持续秒数和可归属的真人队友解救
- [x] 记录本人近战断舌、Tank 石头空爆、Witch 一击与单人击杀
- [x] 记录 Tank/Witch 遭遇、击杀参与和两种弹药升级包部署
- [x] 扩展三数据库初始结构、SQLite 集成测试和只读检查工具
- [x] 在真实本地 PvE 对局中完成 v0.5.1 测试清单

#### Acceptance Criteria

- 每个字段都有明确所有者、统计时机和排除规则
- 枪械射击/命中/弹药/换弹、普通感染者伤害、近战命中/斩首和激光瞄准器不进入数据库
- 设备类别和总计可从精确设备行无损聚合
- 常见 5×5～10×10 与大型 20×20 部署下不产生逐事件 SQL 或动态设备行

#### Notes

插件尚未发布，v0.5.1 直接重写 `0001_initial.sql`。验收前需清空或备份旧数据库；首次公开发布后不得再修改已经应用的迁移。

---

### v0.5.2 - PvE Interactions and State Durations

**Status:** Completed (local validation scope)

**Scope:** User-facing / Correctness / Performance / Testing

**Goal:** 补齐可可靠确认完成的目标互动、弹药补给、失能时长和黑白队友恢复统计。

#### Tasks

- [x] 以固定实体输出白名单记录成功目标互动，同一实体每个 Round 最多一次
- [x] 使用 `ammo_pickup` 记录从弹药堆实际补充弹药的次数
- [x] 分开累计真人 Segment 内倒地和挂边完整秒数
- [x] 验证医疗包成功将开始时为黑白状态的队友恢复为彩色
- [x] 将五项数据加入三数据库初始结构、绝对快照、SQLite 测试和检查工具
- [x] 明确排除 `player_use`、`finale_start`、普通开门和搬运目标
- [x] 在真实本地 PvE 对局中完成 v0.5.2 测试清单

#### Acceptance Criteria

- 未完成、取消或没有真人 activator 的互动不计数
- 同一目标实体的重复输出不会在同一 Round 重复计数
- 倒地和挂边时间互斥，不依赖每秒计时器，重复 flush 不重复累计
- 自疗、治疗中断和未真正解除黑白状态不会计入黑白队友恢复
- 新统计只写入 coop/realism 真人幸存者 Segment

#### Notes

插件仍未发布，v0.5.2 继续重写 `0001_initial.sql`。验收前必须清空或备份旧数据库并由插件重建。

---

### v0.5.3 - Vomit Jar Action Reliability

**Status:** Completed (local validation scope)

**Scope:** Correctness / Testing

**Goal:** 在 `weapon_fire` 未提供可靠胆汁罐字段时仍准确记录成功投掷。

#### Tasks

- [x] 从 `vomitjar_projectile` 的真人投掷者记录一次 `Vomit Jar actions`
- [x] 胆汁罐不再进入原有 `weapon_fire` 动作路径，避免重复计数
- [x] 保持数据库结构和既有 v0.5.2 数据兼容
- [x] 在真实本地 PvE 对局投掷多个胆汁罐并确认每次只增加一次

#### Acceptance Criteria

- 每个真人幸存者成功投掷的胆汁罐只增加一次 `equipment_id=40 actions`
- Bot、无归属投射物和非 PvE Segment 不产生个人动作统计
- Molotov 与 Pipe Bomb 的既有动作统计不受影响

---

### v0.6.0 - Versus Statistics

**Status:** Completed (local validation scope)

**Scope:** User-facing / Architecture / Testing

**Goal:** 在独立模型中采集对抗幸存者和感染者统计。

#### Tasks

- [x] 拆分真人/Bot 特感和 Tank 击杀与有效伤害
- [x] 在独立表中采集对抗幸存者普通感染者击杀、感染者承伤和真人/Bot 友伤
- [x] 复用但隔离对抗幸存者倒地、死亡、救援、治疗和临时生命统计
- [x] 采集真人感染者有效出生，以及对真人/Bot 幸存者的伤害、倒地和击杀
- [x] 接入绝对快照、统一异步刷新事务、有界关闭队列和管理员状态诊断
- [x] 扩展 SQLite 集成测试与数据库检查工具
- [x] 使用 `versus + sb_all_bot_game 1` 完成单人上下半场统计归属验证
- [x] 修复第二半场结束后缺少可靠 `map_transition` 导致的 Run/Session 跨图断裂
- [x] 完成 c5m1 上下半场并进入 c5m2，确认同一 Run、同一 Session、`round_seq=3`、`map_seq=2`
- [ ] 在真实多人服务器完成双方真人、换队、Tank 交接、重连和中途加入测试（转移至 v0.7.0）

#### Acceptance Criteria

- Versus 数据不会进入 PvE 排行或聚合
- 幸存者和感染者统计写入不同表
- 第一阶段不推算稳定队伍、推进分、胜负或 MVP

#### Notes

v0.6.0 不新增迁移：三数据库 `0001_initial.sql` 已预留两张对抗统计表。单人本地对抗能够验证上下半场、Bot 目标、跨图和绝对快照，但无法覆盖两个 SteamID 同时在线的真人时序。真实多人部分按照 [v0.6.0 对抗测试清单](v0.6-test-checklist.md) 延期到 v0.7.0。

---

### v0.6.1 - Versus Production Stabilization

**Status:** Deferred to v0.7.0

**Scope:** Stability / Correctness / Testing

**Goal:** 根据真实服务器首轮数据修正多人时序、归属和生命周期问题，不扩张统计口径。

#### Focus

- 双方真人完整半场
- 换队、闲置、旁观、重连与中途加入
- Tank 控制权交接与半场重开

#### Tasks

- [ ] 在 3～7 天灰度期收集数据库检查结果和 SourceMod 错误日志
- [ ] 核对每个对抗 Segment 的阵营、Round、Session 和统计表唯一归属
- [ ] 修复真人/Bot 归属、重复出生、跨 Segment 污染和生命周期时序问题
- [ ] 完成 v0.6.0 测试清单中的全部真实服务器项目

#### Acceptance Criteria

- 数据库健康检查的 orphan、side mismatch、mode mismatch 和 dual stats 全部为 0
- 换队、重连和 Tank 交接不会把统计写给错误 SteamID 或 Segment
- 两个完整半场和半场重开均能生成可解释、可重复核对的数据

#### Notes

本阶段依赖至少两名真实玩家和持续运行的服务器，当前不作为 v0.6.2～v0.6.5 代码扩展的前置条件。任务与验收统一并入 v0.7.0 的多人/发布加固阶段，避免为了版本编号阻塞本地可验证的统计开发。

---

### v0.6.2 - Infected Class Breakdown

**Status:** Completed (local validation scope)

**Scope:** User-facing / Data model / Testing

**Goal:** 在核心归属稳定后补充对抗感染者职业明细。

#### Focus

- Smoker、Boomer、Hunter、Spitter、Jockey、Charger、Tank
- 职业出生、伤害、倒地和击杀
- 固定白名单与有界表结构

#### Tasks

- [x] 设计固定七职业子表并完成三数据库等价 DDL
- [x] 按职业拆分真人感染者出生、伤害、倒地和击杀，并保留真人/Bot 目标口径
- [x] 保留现有感染者总计作为可直接查询的权威快照
- [x] 增加职业总和与总计一致性检查和 SQLite 绝对快照测试
- [x] 按 [v0.6.2 本地测试清单](v0.6.2-test-checklist.md) 验证真实职业行与总计一致

#### Acceptance Criteria

- 未知职业不会创建动态列或无限增长的名称维度
- 职业明细能够无歧义聚合回现有感染者总计
- 不记录 Bot 自己的个人统计

#### Notes

插件尚未公开发布，v0.6.2 继续重写 `0001_initial.sql`，不保留测试数据库升级包袱。测试前必须备份后清空旧数据库，由插件按当时的 13 表、8 索引结构重新建立。

---

### v0.6.3 - Control and Ability Effectiveness

**Status:** Completed (local validation)

**Scope:** User-facing / Correctness / Performance

**Goal:** 记录对抗特感控制和关键技能效果，而不保存逐事件流水。

#### Focus

- 控制成功次数和持续时间
- 可可靠归属的技能命中/效果
- 低频绝对快照

#### Tasks

- [x] 定义 Smoker、Hunter、Jockey、Charger 控制口径及重复事件去重
- [x] 记录 Boomer 胆汁命中人数和 Spitter 酸液有效伤害，并拆分真人/Bot 目标
- [x] 将新增指标并入固定职业内存状态和周期绝对快照
- [x] 静态验证事件热路径不产生逐事件 SQL、动态职业行或无界分配
- [x] 按 [v0.6.3 本地测试清单](v0.6.3-test-checklist.md) 验证控制封口与能力归属

#### Acceptance Criteria

- 每项能力指标都有明确所有者、开始、结束和排除条件
- 救援、死亡、换队和 Round 结束能正确封口控制时长
- 高频伤害事件仍只修改内存，不直接访问数据库

---

### v0.6.4 - Versus Survivor Combat Detail

**Status:** Implementation complete; local validation pending

**Scope:** User-facing / Data model / Testing

**Goal:** 在不复制 PvE 模型的前提下补充对抗幸存者有价值的战斗明细。

#### Focus

- 特感职业击杀与伤害
- 对抗设备使用边界
- 与感染者职业明细的交叉校验

#### Tasks

- [x] 按职业拆分幸存者对真人/Bot 特感的击杀与有效伤害
- [x] 记录官方投掷物、弹药升级包、本人近战断舌、Tank 石头空爆和 Witch 技巧
- [x] 保留现有幸存者总计并增加明细合计一致性检查
- [x] 明确不采集枪械命中率、逐发命中、弹药、换弹及逐武器对抗表
- [x] 完成 SQLite 集成、三数据库迁移等价性、检查器、编译和打包离线验证
- [ ] 按 [v0.6.4 本地测试清单](v0.6.4-test-checklist.md) 验证游戏事件归属和数据库快照

#### Acceptance Criteria

- 幸存者职业明细可无歧义聚合回现有真人/Bot 总计
- PvE 与 Versus 设备数据仍由表和查询边界彻底隔离
- 新增统计不会导致事件级 SQL 或不受控维度增长

---

### v0.6.5 - Versus Contract Freeze

**Status:** Planned

**Scope:** Architecture / Stability / Documentation

**Goal:** 冻结 Go 查询侧开始前的第一版对抗统计契约。

#### Focus

- 字段语义与兼容性
- 三数据库迁移
- 读取侧聚合边界

#### Tasks

- [ ] 审核 v0.6.x 每个字段的所有者、单位、归属、排除和 Bot 语义
- [ ] 完成 SQLite/MySQL/PostgreSQL 等价迁移与回归查询
- [ ] 更新统计契约、数据库结构、运维和查询示例
- [ ] 明确推进分、稳定队伍、最终胜负和 MVP 继续延期

#### Acceptance Criteria

- Go 读取侧无需猜测字段语义或重新推断真人/Bot 归属
- 三种数据库对同一绝对快照产生等价结果
- v0.6.x 之后对已发布字段的破坏性修改必须通过新迁移处理

---

### v0.7.0 - Multi-database and Release Hardening

**Status:** Planned

**Scope:** Stability / Performance / CI / Release

**Goal:** 在开始网页侧之前证明采集器可长期运行和安全升级。

#### Tasks

- [ ] 在真实 MySQL 与 PostgreSQL 测试迁移、重连和 upsert
- [ ] 在独立服务器或多人服务器验证远端真人断线、重连及数据库故障期间断线补写
- [ ] 使用双方真人完成 Versus 两个半场、换队、半场重开和中途退出验收
- [ ] 完成 3～7 天 Versus 灰度，核对 Tank 交接、真人/Bot 归属、重复出生和跨 Segment 污染
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
