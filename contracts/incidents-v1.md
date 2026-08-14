# Incident Contract v1

状态：自 collector v1.3.0 起冻结。数值 ID、参与者语义、时间、坐标与完整性规则属于持久化公开契约，不得复用或静默改变。

## 稳定 ID

| ID | Incident |
|---:|---|
| 1 | `SURVIVOR_CONTROL` |
| 2 | `SURVIVOR_INCAP` |
| 3 | `SURVIVOR_DEATH` |
| 4 | `SURVIVOR_REVIVE` |
| 5 | `LEDGE_RESCUE` |
| 6 | `DEFIB_REVIVE` |
| 7 | `CAR_ALARM` |
| 8 | `TANK_SPAWN` |
| 9 | `TANK_DEATH` |
| 10 | `WITCH_SPAWN` |
| 11 | `WITCH_DEATH` |
| 12 | `WITCH_STARTLE` |
| 13 | `MEDKIT_HEAL` |
| 14 | `OBJECTIVE_COMPLETE` |

所有 v1.3 记录固定写入 `incident_version=1`。

ParticipantKind 固定为：0 `NONE`、1 `HUMAN_SURVIVOR`、2 `BOT_SURVIVOR`、3 `HUMAN_INFECTED`、4 `BOT_INFECTED`、5 `COMMON_INFECTED`、6 `WITCH`、7 `WORLD`、8 `OTHER`、9 `UNKNOWN`。`NONE` 与 `UNKNOWN` 不等价。真人身份复用 Session 已缓存的认证 SteamID；Bot 不创建玩家档案。

玩家 Incident 至少涉及一名已认证真人，纯 Bot 对 Bot 事件不保存。Boss 生命周期是 Round 级例外，但只有该 Round 已产生真人 Segment 后才保存。

## 序列、时间与坐标

- 每 Round 的 `incident_seq` 从 1 单调递增；每次语义事件生成尝试都先增加序号和期望数，队列溢出允许序号空洞。
- `(round_id, incident_seq)` 写入必须幂等；禁止宽泛 `INSERT IGNORE`。
- `occurred_at` 是 Unix 秒；`round_offset_ms` 是由 `GetGameTime()` 得出的 Round 相对毫秒。
- CONTROL 的时间和 `pos` 属于开始时刻，`duration_ms` 与 `end_pos` 属于结束时刻。
- 坐标四舍五入为整数 Source 单位；缺失写 `NULL`，不得用 `0,0,0`；不采样轨迹。

## CONTROL Episode

一次控制只保存一行完整 Episode。end reason 固定为：0 `UNKNOWN`、1 `RELEASED`、2 `RESCUED`、3 `SELF_FREED`、4 `CONTROLLER_KILLED`、5 `TARGET_DIED`、6 `LIFECYCLE_END`。只使用现有状态机可验证的事实，不依据最后伤害猜测因果。

## 事件边界

- INCAP 来源 `player_incapacitated`；DEATH 沿用累计死亡有效性且排除 `abort=true`。
- REVIVE/LEDGE 按 `revive_success.ledge_hang` 分开；DEFIB 来源 `defibrillator_used`。
- CAR_ALARM 来源 `triggered_car_alarm`，保持严格真人幸存者归属，坐标为触发玩家位置。
- Tank/Witch 使用独立 Spawn/Death。Death 可关联同 Round 成功入队的 Spawn；无法匹配为 0。
- Witch Death 的 `detail_flags` bit 0 表示引擎 `oneshot=true`，v1 其他位必须为 0。
- Witch Startle 来源 `witch_harasser_set`，只保存首次可验证惊扰；actor 为惊扰幸存者，target 为 Witch。
- Medkit Heal 来源成功的 `heal_success`；actor 为治疗者，target 为被治疗者。治疗量继续由 Core Stats 与 Relationship Stats 保存；自疗允许 actor 与 target 相同。
- Objective Complete 来源现有严格真人幸存者机关互动归属；actor 为完成互动的真人幸存者。

稳定 ID 只允许尾部追加。未知未来 ID 的读取端必须安全忽略或明确标记为未知，不得按已有类型猜测；`SPECIAL_DEATH` 不属于 Incident v1。

## 完整性与保留

- 禁用：`enabled=0, complete=0, expected=0, dropped=0`。
- 采集中或未最终持久化：`enabled=1, complete=0`。
- 只有 Round 有序结束、所有未丢弃 Incident 已持久化且 `dropped=0` 时才能写 `complete=1`。
- `dropped>0` 或进程崩溃的 Round 不得被解释为零事件完整样本。

Incident 默认保留 180 天。清理只删除已结束 Round 的旧 Incident，不删除 Round Context 或累计统计。
