# Round Context Contract v1

状态：自 collector v1.3.0 起冻结。Round Context 保存 Round 开始时的有限规则快照，以及这些值是否在 Round 中发生过变化。

`context_version=1`。字段来源：`collector_version=LPS_VERSION`、`ruleset_name=sm_lps_ruleset_name`、`difficulty=z_difficulty`、`survivor_limit`、`max_player_zombies=z_max_player_zombies`、`common_limit=z_common_limit`、`tank_health=z_tank_health`、`witch_health=z_witch_health`。规则集名称去除首尾空白并限制 64 字符。可选 CVar 缺失时字符串写 `''`、整数写 `-1`，不得因此禁用采集器。

值在 Round 开始时捕获，之后永不覆盖。存在的 CVar 发生变化时只设置 `change_mask`、增加 revision 并标记 dirty；即使改回原值也不清除。

| Bit | 字段 |
|---:|---|
| 0 | `ruleset_name` |
| 1 | `difficulty` |
| 2 | `survivor_limit` |
| 3 | `max_player_zombies` |
| 4 | `common_limit` |
| 5 | `tank_health` |
| 6 | `witch_health` |

`sm_lps_incidents_enabled` 在 Round 开始时锁存，中途修改只影响下一 Round。Context 永久保留。

读取侧按 `context_version, ruleset_name, difficulty, survivor_limit, max_player_zombies, common_limit, tank_health, witch_health` 的确定性规范表示计算 SHA-256，界面显示 `ctx-<前 12 个十六进制字符>`。不得加入 collector 版本、时间戳、change mask 或 Incident 字段。只有 `change_mask=0` 能进入严格稳定 Context 分组。
