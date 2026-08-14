# Player Relationship Contract v1

状态：计划自 collector v1.3.1 起冻结。

本文定义真人玩家之间的定向互动关系事实。Player Relationship 只表达服务器能够可靠验证的游戏行为，不表达好友关系、社交关系、默契评分或任何主观评价。

## 1. 总体原则

- 只有通过 Steam 认证的真人玩家可以成为关系主体。
- Bot 不拥有玩家关系记录。
- 关系具有方向性：`A → B` 与 `B → A` 是两条不同关系。
- `actor_steam_id` 与 `target_steam_id` 必须不同。
- 每条关系事实必须归属于明确的 `round_id`。
- PvE 与 Versus Survivor 均可产生关系事实。
- Versus Infected 不产生本契约定义的定向支援关系。
- “并肩作战”不写入关系表，继续从 Player Segment 的同 Round、同阵营正重叠时间派生。
- 关系数据使用绝对快照写入，不使用数据库增量累计。
- 关系数据属于永久玩家档案事实，不使用 Incident retention。
- 不计算或持久化“关系分”“默契值”“好友度”等综合评分。

## 2. 数据模型

表：

`lps_player_round_relationship_stats`

主键：

`(round_id, actor_steam_id, target_steam_id)`

字段：

| 字段 | 含义 |
|---|---|
| `round_id` | 所属 Round |
| `actor_steam_id` | 行为发起真人 |
| `target_steam_id` | 行为目标真人 |
| `relationship_version` | v1 固定为 `1` |
| `incap_revives` | A 扶起普通倒地状态 B 的次数 |
| `ledge_rescues` | A 拉起挂边 B 的次数 |
| `defib_revives` | A 使用电击器复活 B 的次数 |
| `smoker_rescues` | A 从 Smoker 控制中解救 B 的次数 |
| `hunter_rescues` | A 从 Hunter 控制中解救 B 的次数 |
| `jockey_rescues` | A 从 Jockey 控制中解救 B 的次数 |
| `charger_rescues` | A 从 Charger 控制中解救 B 的次数 |
| `control_rescue_duration_ms` | 由 A 成功结束的 B 控制 Episode，在结束前已经持续的累计毫秒数 |
| `medkits_used` | A 成功对 B 使用医疗包的次数 |
| `medkit_healing` | 上述医疗包对 B 实际恢复的永久生命 |
| `black_white_restores` | A 使用医疗包成功把黑白状态 B 恢复为彩色的次数 |
| `friendly_fire_damage` | A 对 B 造成的实际有效友伤 |
| `last_saved_at` | 最近成功持久化 Unix 秒 |
| `revision` | 当前绝对快照版本 |

所有计数、治疗量、伤害量和持续时间不得为负数。

关系行至少有一个业务字段大于 0；不得预先为所有玩家组合创建全零关系行。

## 3. 普通扶起

`incap_revives` 使用现有 `revive_success` 且 `ledge_hang=false` 的成功语义。

必须满足：

- rescuer 是认证真人幸存者；
- subject 是另一名认证真人幸存者；
- 两者属于当前 Round；
- 不允许 `actor == target`。

一次成功扶起只增加一次。

## 4. 挂边救援

`ledge_rescues` 使用 `revive_success` 且 `ledge_hang=true`。

普通倒地扶起与挂边救援必须分别保存，不得合并为不可拆分的 `revives`。

## 5. 电击复活

`defib_revives` 使用成功的 `defibrillator_used`。

只有认证真人 A 成功复活认证真人 B 时产生关系事实。

## 6. 特感控制解救

适用职业：

- Smoker
- Hunter
- Jockey
- Charger

每个有效 Control Episode 最多给一名真人 rescuer 记录一次对应职业解救。

可以计为团队解救的情况包括：

- 游戏事件明确给出另一名真人 rescuer；
- 另一名真人击杀当前控制者并因此可靠结束控制。

以下情况不得计入 A → B 解救：

- B 自己挣脱；
- 控制自然释放；
- B 死亡；
- Round / Segment 生命周期结束；
- 无法可靠确定 rescuer；
- 重复的停止事件。

`control_rescue_duration_ms` 保存该次成功解救所结束的完整 Control Episode 已持续时间。

例如 A 在 B 被 Hunter 控制 1650ms 后成功解救：

- `hunter_rescues += 1`
- `control_rescue_duration_ms += 1650`

该字段不表示“减少了多少潜在控制时间”，不得如此解释。

## 7. 医疗包

`medkits_used` 只在成功的 `heal_success` 后增加。

`medkit_healing` 使用与现有 Core Stats 相同的 `health_restored` 实际永久生命口径。

只有：

`真人 A → 另一名真人 B`

才写入关系表。

自疗继续进入个人 Core Stats，但不产生玩家关系。

## 8. 黑白恢复

`black_white_restores` 必须使用可验证的黑白恢复语义：

1. 治疗开始时确认目标为黑白状态；
2. 治疗成功；
3. 治疗完成后确认目标确实已经解除黑白状态。

一次医疗包同时：

- `medkits_used += 1`
- `medkit_healing += actual`
- 可选 `black_white_restores += 1`

`black_white_restores` 是医疗行为属性，不代表额外的一次支援行为。

PvE 与 Versus Survivor 应使用相同语义。

## 9. 友伤

`friendly_fire_damage` 使用现有有效友伤口径：

- 只记录目标实际损失生命；
- 已应用游戏实际友伤倍率；
- 不记录溢出伤害；
- 不记录自伤；
- 不记录无法可靠归属的环境伤害。

只有真人幸存者 A 对另一名真人幸存者 B 的友伤进入关系表。

Bot 友伤继续保留在个人 Core Stats 中，但不产生玩家关系。

## 10. 并肩作战

Player Relationship 表不重复保存并肩关系。

Dashboard 继续从 Player Segment 派生：

- 相同 `round_id`；
- 相同 `side`；
- SteamID 不同；
- Segment 墙钟区间存在正重叠。

可以派生：

- `shared_rounds`
- `shared_seconds`

该数据表示“同阵营共同参赛”，不表示好友关系。

## 11. 派生指标

以下指标只在读取侧计算，不写入 Stats DB。

### 特感解救总数

`special_rescues`：

`smoker_rescues + hunter_rescues + jockey_rescues + charger_rescues`

### 支援行为次数

`support_actions`：

`incap_revives + ledge_rescues + defib_revives + special_rescues + medkits_used`

`black_white_restores` 不再次加入，以避免一次医疗包重复计数。

### 平均特感解救响应

存在至少一次特感解救时：

`average_control_rescue_ms = control_rescue_duration_ms / special_rescues`

该指标越低表示已完成解救的响应时间越短。

### 双向支援

A 与 B 的双向支援量：

`support_actions(A→B) + support_actions(B→A)`

只展示事实总量，不转换为关系分数。

## 12. 生命周期与写入

关系事实首先更新当前 Round 的内存绝对快照。

首次发生有效 A → B 行为时才创建该 pair snapshot。

后续行为只修改：

- 对应字段；
- `revision`；
- dirty 状态。

数据库写入沿用 Core Stats 的异步绝对快照流程。

不得：

- 每次救援立即执行同步 SQL；
- 每次友伤立即执行 SQL；
- 使用 `column = column + value` 的数据库增量；
- 使用 Incident Analysis Queue 保存永久关系事实。

Round 或相关生命周期结束时应确保最终关系快照进入可靠持久化流程。

## 13. 历史数据

v1.3.1 之前没有保存 actor → target 维度，因此不得从旧累计统计反推：

- 谁救过谁；
- 谁治疗过谁；
- 谁误伤过谁；
- 谁从特感控制中救过谁。

旧历史的并肩作战仍可由 Segment 推导。

Dashboard 必须明确说明：

> 定向互动关系统计仅覆盖服务器启用 Player Relationship Contract v1 后产生的数据。

缺少历史关系行不得解释为“历史上从未发生互动”。

## 14. 保留策略

Player Relationship 属于永久玩家档案事实。

不得使用 Incident 默认 180 天 retention 删除。

未来如果关系数据规模需要压缩，应另行定义 Relationship Aggregate Contract，不得静默改变 v1 原始关系语义。

## 15. 数据库索引

建议：

- 主键 `(round_id, actor_steam_id, target_steam_id)`
- 索引 `(actor_steam_id, round_id)`
- 索引 `(target_steam_id, round_id)`

不得为每个统计字段单独建立排序索引。

## 16. 一致性检查

`doctor --deep` 至少检查：

- `relationship_version = 1`
- `actor_steam_id != target_steam_id`
- actor / target 均能关联认证玩家
- `round_id` 存在
- 所有数值非负
- 每行至少一个业务字段大于 0
- 不存在 Bot Steam 身份
- 不存在感染者 → 感染者的本契约支援记录

未知 `relationship_version` 必须报告为不支持版本，不得按 v1 强行解释。
