# Assist Contract v1

状态：计划自 collector v1.3.1 起冻结。

本文定义普通特感 Assist、Versus Tank Assist 以及 Boss Assist 的统一业务语义。

Assist 表示真人幸存者对最终死亡目标做出了可验证的有效伤害贡献，但没有获得该目标的最后击杀。

## 1. 总体原则

- Assist 只归属于通过 Steam 认证的真人幸存者。
- Assist 必须基于目标本次生命中的有效伤害贡献。
- 同一个目标的一次生命中，同一真人最多获得一次 Assist。
- 最后击杀者获得 Kill，不同时获得该目标的 Assist。
- 目标必须发生有效死亡；未死亡不产生 Assist。
- `abort=true` 的死亡不得结算 Assist。
- 不按伤害百分比设置最低门槛；至少 1 点有效伤害即可成为贡献者。
- 不根据时间窗口、最后一次命中时间或伤害排名猜测贡献。
- 不保存逐伤害事件流水。
- Assist 状态只存在于内存中的目标生命期贡献者集合，死亡时一次性结算。

## 2. 有效贡献

贡献伤害必须使用现有统计契约中的“有效伤害”定义：

- 只记录目标实际生命损失；
- 不记录溢出伤害；
- 不记录自伤；
- 排除现有通用排除伤害；
- 投掷物、火焰、酸液等只有能够可靠追溯真人所有者时才归属。

同一真人对同一目标造成多次有效伤害仍只登记一次 contributor。

不需要为了 Assist 保存该真人具体造成了多少伤害。

## 3. Contributor 生命周期

普通特感和 Tank 每次新的有效生命开始时必须清空上一生命的 contributor 状态。

状态不得因为客户端槽位或实体索引重用而串到下一只目标。

实现必须使用足以区分当前生命的运行时身份，例如：

- UserID；
- Segment generation；
- spawn generation；
- 等价的实体生命周期保护。

以下情况必须清理对应 contributor 状态：

- 新 spawn；
- 有效死亡并完成结算；
- 客户端断开；
- 目标生命周期结束；
- Round 结束；
- 模式退出支持范围。

## 4. PvE 普通特感 Assist

适用于：

- Smoker
- Boomer
- Hunter
- Spitter
- Jockey
- Charger

真人幸存者第一次对该普通特感当前生命造成有效伤害时登记为 contributor。

目标发生有效死亡时：

1. 确定最终击杀者；
2. 遍历当前目标 contributor；
3. 最终击杀者不获得 Assist；
4. 其他仍能可靠归属到当前 Round 真人幸存者 Segment 的 contributor 各获得一次 Assist。

### Bot 最后击杀

如果目标由 Bot 最后击杀：

- 真人 contributor 可以获得 Assist；
- 不把 Kill 转移给真人。

### 环境最后击杀

如果目标由环境完成最后击杀：

- 真人 contributor 可以获得 Assist；
- 不产生真人 Kill。

### 玩家离开

如果 contributor 在目标死亡前已经：

- 断开服务器；
- 结束当前 Segment；
- 离开幸存者统计身份；
- 无法再可靠关联认证真人；

则本次死亡不向已经关闭的旧 Segment 回写 Assist。

不得为了 Assist 修改已经关闭并进入异步持久化流程的历史 Segment。

## 5. PvE 字段

`lps_pve_segment_stats` 新增：

- `special_assists`
- `smoker_assists`
- `boomer_assists`
- `hunter_assists`
- `spitter_assists`
- `jockey_assists`
- `charger_assists`

必须满足：

`special_assists = smoker_assists + boomer_assists + hunter_assists + spitter_assists + jockey_assists + charger_assists`

Kill 与 Assist 相互独立：

- 最后一击真人：Kill +1
- 其他有效真人 contributor：Assist +1

## 6. PvE Tank / Witch Assist

PvE 不新增重复的 Boss Assist 持久化字段。

现有：

- `tank_kill_participations`
- `witch_kill_participations`
- `tank_kills`
- `witch_kills`

已经能够表达同一事实。

读取侧定义：

`tank_assists = tank_kill_participations - tank_kills`

`witch_assists = witch_kill_participations - witch_kills`

必须满足：

- `tank_kill_participations >= tank_kills`
- `witch_kill_participations >= witch_kills`

违反时属于数据质量错误，不得通过 `MAX(0, ...)` 静默修正。

如果 Boss 最终由 Bot 或环境击杀，则所有仍可可靠归属的真人 contributor 都属于 Assist，公式仍然成立。

## 7. Versus 普通特感 Assist

适用于幸存者攻击六种普通特感。

目标死亡时按照目标当时控制者分类：

- 真人控制：Human Controller
- Bot 控制：Bot Controller

分类描述的是死亡时目标感染者的控制状态，不描述 contributor。

例如某 Hunter：

1. 最初由真人控制；
2. A 对其造成有效伤害；
3. 控制者离开后由 Bot 接管；
4. 最终死亡；

则 A 的 Assist 属于 `bot_controller_assists`。

该规则与现有 Versus Kill 分类保持一致。

## 8. Versus 普通特感字段

`lps_versus_survivor_stats` 新增：

- `human_special_assists`
- `bot_special_assists`

`lps_versus_survivor_infected_class_stats` 新增：

- `human_controller_assists`
- `bot_controller_assists`

六种普通特感职业行必须满足：

六职业 `human_controller_assists` 之和
= `human_special_assists`

六职业 `bot_controller_assists` 之和
= `bot_special_assists`

## 9. Versus Tank Assist

Versus Tank 使用与普通特感相同的 contributor 原则，但保持独立 Boss 统计。

`lps_versus_survivor_stats` 新增：

- `human_tank_assists`
- `bot_tank_assists`

职业 ID 8 的 `lps_versus_survivor_infected_class_stats` 同样使用：

- `human_controller_assists`
- `bot_controller_assists`

并满足：

Tank 职业行 `human_controller_assists`
= `human_tank_assists`

Tank 职业行 `bot_controller_assists`
= `bot_tank_assists`

分类仍以 Tank 死亡时真人/Bot 控制状态为准。

最终击杀真人不得同时获得该 Tank Assist。

## 10. Versus Witch Assist

Witch 不属于可控制感染者，不使用 human/bot controller 分类。

Versus Survivor 总表新增：

- `witch_encounters`
- `witch_kill_participations`

第一次对某只 Witch 造成有效伤害时：

`witch_encounters += 1`

该 Witch 最终有效死亡且 contributor 仍可可靠归属时：

`witch_kill_participations += 1`

读取侧派生：

`witch_assists = witch_kill_participations - witch_kills`

并要求：

`witch_kill_participations >= witch_kills`

不额外持久化 `witch_assists`。

## 11. Kill、Assist、Participation 的关系

### 普通特感

每名真人对同一目标最终只能属于：

- Killer；
- Assistant；
- 无归属。

Killer 与 Assistant 互斥。

### Boss

Boss 使用：

- Encounter：本次生命中至少造成过一次有效伤害；
- Kill Participation：该玩家是 contributor，且 Boss 最终有效死亡；
- Kill：最后击杀；
- Assist：`Kill Participation - Kill`。

因此 Boss Assist 是 Participation 的派生子集，而不是额外持久化事实。

## 12. Incident 边界

Assist 属于 Core Stats，不为每名 Assistant 创建独立 Incident。

不得增加：

- `SPECIAL_ASSIST`
- `TANK_ASSIST`
- `WITCH_ASSIST`

之类逐贡献者 Incident。

Assist 的存在不要求保存逐伤害流水。

如未来需要逐目标贡献审计，应另行定义 Encounter / Contributor Contract，不得通过扩充 Assist v1 偷渡逐伤害日志。

## 13. 历史字段

Stats schema 5 引入的新 Assist 字段，对历史已经存在的 Segment 必须保持 `NULL`。

`NULL` 表示：

> 当时没有采集该指标。

不得迁移为 `0`。

v1.3.1 之后新创建并启用 Assist Contract v1 的统计行使用：

- `0` 表示已采集但没有 Assist；
- 正整数表示真实采集结果。

Dashboard 不得把历史 `NULL` 显示为真实 0。

## 14. stats_version 与 Aggregate

本契约增加新指标，但不改变现有字段语义。

因此：

- 现有 gameplay `stats_version` 继续保持 `1`；
- Aggregate Contract 继续保持 v1；
- v1.3.1 不要求为了 Assist 修改 Aggregate Contract。

Assist 数据从永久保留的 Core Stats 读取。

未来若加入 Assist 日/月/终身聚合，应另行升级 Aggregate Contract，不得静默改变 Aggregate v1。

## 15. 性能约束

Assist contributor tracking 必须：

- 复用现有有效伤害路径；
- 不新增每帧轮询；
- 不新增同步数据库操作；
- 不保存逐 hit 记录；
- contributor 登记应为常数级内存操作；
- 目标死亡时允许对有限玩家槽位进行一次 bounded scan。

Assist 不得依赖 Analysis Incident Queue。

## 16. 一致性检查

`doctor --deep` 至少检查：

### PvE

- Assist 字段非负或历史 `NULL`
- `special_assists` 等于六职业 Assist 之和
- `tank_kill_participations >= tank_kills`
- `witch_kill_participations >= witch_kills`

### Versus

- Assist 字段非负或历史 `NULL`
- 六职业 Human Assist 之和等于 `human_special_assists`
- 六职业 Bot Assist 之和等于 `bot_special_assists`
- Tank ID 8 Human Assist 等于 `human_tank_assists`
- Tank ID 8 Bot Assist 等于 `bot_tank_assists`
- `witch_kill_participations >= witch_kills`

未知未来 Assist Contract 语义不得按 v1 猜测解释。
