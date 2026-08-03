# Versus v1 读取契约

状态：冻结（collector v0.6.6，`stats_version=1`）

本文是 SourceMod 采集器、三种数据库迁移和未来 Go 读取侧共同遵守的第一版对抗契约。
字段业务口径以本文为准，精确字段清单以
[`versus-schema-v1.json`](versus-schema-v1.json) 为准。

## 1. 支持边界

- 只统计 `mp_gamemode=versus`；突变、清道夫、生还者模式和第三方模式不进入本契约。
- 只有通过 Steam 认证的真人拥有 Player、Session 和 Segment。
- Bot 没有个人统计行，但真人与 Bot 目标之间可可靠归属的行为会写入真人行。
- 幸存者和感染者使用不同总表；一个 Segment 不得同时拥有两种总表。
- 所有玩法数值都是当前 Segment 的绝对快照，重复 upsert 不得累加。
- 伤害单位是目标实际损失的整数生命值，不含溢出伤害。
- 所有 `*_seconds` 是累计整秒，所有时间戳是 Unix 秒。
- 所有计数、伤害和时长都不得为负数。

## 2. 读取关系

对抗个人统计必须沿下面的关系读取：

```text
lps_players
  └─ lps_sessions
      └─ lps_player_segments
          ├─ lps_versus_survivor_stats
          │   └─ lps_versus_survivor_infected_class_stats
          └─ lps_versus_infected_stats
              └─ lps_versus_infected_class_stats

lps_runs
  ├─ lps_versus_run_results
  └─ lps_rounds
      └─ lps_versus_round_results
```

读取侧必须同时验证：

- `lps_runs.mode_family='versus'`；
- `lps_runs.game_mode='versus'`；
- 幸存者总表对应 `segment.side='survivor'`；
- 感染者总表对应 `segment.side='infected'`；
- 玩法表的 `stats_version=1`。

总表缺失表示该 Segment 没有可用的 v1 统计快照，不能直接解释为全零。总表存在时，
某个职业子行缺失可以解释为该职业所有指标均为 0。

## 3. 通用快照字段

所有对抗玩法表都包含：

| 字段 | 单位 | 读取规则 |
|---|---|---|
| `stats_version` | 版本号 | v1 固定为 1；读取侧必须显式筛选 |
| `last_saved_at` | Unix 秒 | 最近一次成功持久化时间，不是事件发生时间 |
| `revision` | 单调整数 | 同一主键的新快照覆盖旧值；不可相加 |

统计表主键为 `segment_id`，职业表主键为 `(segment_id, infected_class)`。职业 ID 固定为：

| ID | 职业 |
|---:|---|
| 1 | Smoker |
| 2 | Boomer |
| 3 | Hunter |
| 4 | Spitter |
| 5 | Jockey |
| 6 | Charger |
| 8 | Tank |

未知 ID 不属于 v1，读取侧不得自行命名或合并到其他职业。

## 4. 幸存者总表

表：`lps_versus_survivor_stats`

所有者始终是 `segment_id` 对应的真人幸存者。

### 4.1 击杀与输出

| 字段 | 单位 | 精确定义 |
|---|---|---|
| `common_kills` | 次 | 真人完成最后击杀的普通感染者 |
| `human_special_kills` | 次 | 真人最后击杀当时由真人控制的六种普通特感 |
| `bot_special_kills` | 次 | 真人最后击杀当时由 Bot 控制的六种普通特感 |
| `human_tank_kills` | 次 | 真人最后击杀当时由真人控制的 Tank |
| `bot_tank_kills` | 次 | 真人最后击杀当时由 Bot 控制的 Tank |
| `damage_to_human_special` | 生命值 | 对真人控制普通特感造成的有效伤害 |
| `damage_to_bot_special` | 生命值 | 对 Bot 控制普通特感造成的有效伤害 |
| `damage_to_human_tank` | 生命值 | 对真人控制 Tank 造成的有效伤害 |
| `damage_to_bot_tank` | 生命值 | 对 Bot 控制 Tank 造成的有效伤害 |
| `witch_kills` | 次 | 真人完成最后击杀的 Witch |
| `damage_to_witch` | 生命值 | 对 Witch 造成的有效伤害 |

`human` / `bot` 描述受害感染者在本次伤害或死亡发生时的控制者，不描述统计行所有者。
助攻、环境最后击杀和 Bot 自己的击杀不转移给真人。

### 4.2 承伤、友伤与生存

| 字段 | 单位 | 精确定义 |
|---|---|---|
| `damage_taken_infected` | 生命值 | 该真人受到感染者阵营的有效伤害 |
| `friendly_fire_to_humans` | 生命值 | 该真人对真人幸存者造成的有效友伤 |
| `friendly_fire_to_bots` | 生命值 | 该真人对幸存者 Bot 造成的有效友伤 |
| `friendly_fire_taken` | 生命值 | 该真人受到其他真人幸存者的有效友伤 |
| `incapacitations` | 次 | 该真人进入普通倒地状态 |
| `deaths` | 次 | 该真人真正死亡 |

坠落、溺水、地图处决、自伤和无法可靠归属的环境伤害不进入伤害字段。

### 4.3 救援与治疗

| 字段 | 单位 | 精确定义 |
|---|---|---|
| `incap_revives` | 次 | 救起普通倒地目标 |
| `ledge_rescues` | 次 | 拉起挂边目标 |
| `defib_revives` | 次 | 成功用电击器复活目标 |
| `rescues_received` | 次 | 本人被上述三种方式成功救援的总次数 |
| `medkits_used_self` | 次 | 成功对自己使用医疗包 |
| `medkits_used_on_others` | 次 | 成功对其他幸存者使用医疗包 |
| `medkit_healing_self` | 真实生命 | 医疗包实际恢复给自己的永久生命 |
| `medkit_healing_others` | 真实生命 | 医疗包实际恢复给其他人的永久生命 |
| `pills_used` | 次 | 成功使用止痛药 |
| `adrenaline_used` | 次 | 成功使用肾上腺素 |
| `temporary_health_received` | 临时生命 | 止痛药和肾上腺素实际增加的临时生命 |

真人可以从救援或治疗 Bot 的行为获得相应输出统计；Bot 仍不产生自己的个人统计行。

### 4.4 消耗品与技巧

| 字段 | 单位 | 精确定义 |
|---|---|---|
| `molotovs_thrown` | 次 | 成功投出官方燃烧瓶 |
| `pipe_bombs_thrown` | 次 | 成功投出官方土制炸弹 |
| `vomit_jars_thrown` | 次 | 成功创建有真人所有者的官方胆汁罐投射物 |
| `incendiary_packs_deployed` | 次 | 成功部署燃烧弹药包 |
| `explosive_packs_deployed` | 次 | 成功部署高爆弹药包 |
| `melee_tongue_self_cuts` | 次 | 使用官方近战武器切断控制自己的 Smoker 舌头 |
| `tank_rocks_destroyed` | 次 | 真人幸存者使用受支持枪械摧毁飞行中的 Tank 石头 |
| `witch_oneshots` | 次 | 引擎明确标记为 one-shot 的 Witch 击杀 |
| `witch_solo_kills` | 次 | 从首次有效伤害到死亡只有该真人贡献且由其最后击杀 |

未知/第三方投掷物和升级包不创建动态维度。对抗 v1 不建立逐武器统计表。

## 5. 幸存者职业明细

表：`lps_versus_survivor_infected_class_stats`

| 字段 | 单位 | 精确定义 |
|---|---|---|
| `infected_class` | 固定 ID | 本行对应的感染者职业 |
| `human_controller_kills` | 次 | 击杀该职业真人控制者 |
| `bot_controller_kills` | 次 | 击杀该职业 Bot 控制者 |
| `damage_to_human_controllers` | 生命值 | 对该职业真人控制者的有效伤害 |
| `damage_to_bot_controllers` | 生命值 | 对该职业 Bot 控制者的有效伤害 |

职业 1～6 的和必须分别等于幸存者总表的普通特感真人/Bot 击杀与伤害；职业 8 必须
分别等于 Tank 真人/Bot 击杀与伤害。Witch 不属于可操控职业，不进入该表。

## 6. 感染者总表

表：`lps_versus_infected_stats`

所有者始终是 `segment_id` 对应的真人感染者。字段中的 `human_survivor` / `bot_survivor`
描述目标幸存者身份。

| 字段 | 单位 | 精确定义 |
|---|---|---|
| `spawn_count` | 次 | 真人以可操控感染者成功实体化出生 |
| `damage_to_human_survivors` | 生命值 | 对真人幸存者造成的有效伤害 |
| `damage_to_bot_survivors` | 生命值 | 对幸存者 Bot 造成的有效伤害 |
| `human_survivor_incaps` | 次 | 造成真人幸存者倒地 |
| `bot_survivor_incaps` | 次 | 造成幸存者 Bot 倒地 |
| `human_survivor_kills` | 次 | 造成真人幸存者真正死亡 |
| `bot_survivor_kills` | 次 | 造成幸存者 Bot 真正死亡 |

Tank 属于该表。幽灵状态、重复出生事件和无法确认真人控制者的行为不计入。

## 7. 感染者职业明细与能力

表：`lps_versus_infected_class_stats`

`spawn_count`、两类伤害、两类倒地和两类击杀与感染者总表同义，但限定为当前
`infected_class`。七个职业行的每个字段之和必须严格等于感染者总表。

能力字段：

| 字段 | 单位 | 有效职业与含义 |
|---|---|---|
| `human_survivor_controls` | 人次 | Smoker/Hunter/Jockey/Charger 成功控制真人幸存者 |
| `bot_survivor_controls` | 人次 | 上述四职业成功控制幸存者 Bot |
| `human_survivor_control_seconds` | 秒 | 对真人目标的累计完整控制时间 |
| `bot_survivor_control_seconds` | 秒 | 对 Bot 目标的累计完整控制时间 |
| `human_survivor_ability_hits` | 人次 | Boomer 胆汁实际命中的真人幸存者人数 |
| `bot_survivor_ability_hits` | 人次 | Boomer 胆汁实际命中的 Bot 幸存者人数 |
| `human_survivor_ability_damage` | 生命值 | Spitter 酸液对真人造成的有效伤害 |
| `bot_survivor_ability_damage` | 生命值 | Spitter 酸液对 Bot 造成的有效伤害 |

能力列在不适用职业必须为 0：控制只允许职业 1/3/5/6，命中人数只允许职业 2，能力
伤害只允许职业 4。Spitter 死亡后仍存在的酸液可归给原真人所有者。Charger 搬运转入
捶打仍属于一次连续控制。

## 8. 半场和比赛结果

### 8.1 `lps_versus_round_results`

所有者是已经结束的 `round_id`，每个半场最多一条当前权威快照。

| 字段 | 单位/枚举 | 读取规则 |
|---|---|---|
| `scoring_team_slot` | 0/1 | 本半场担任幸存者并取得地图分的 Run 内逻辑队伍 |
| `teams_flipped` | 0/1 | 原始 GameRules 诊断值，不用于单独推断计分槽位 |
| `team_0_map_score` | 分 | 逻辑队伍 0 的本半场地图分，未知为 -1 |
| `team_1_map_score` | 分 | 逻辑队伍 1 的本半场地图分，未知为 -1 |
| `team_0_campaign_score` | 分 | 半场结束时逻辑队伍 0 的累计分 |
| `team_1_campaign_score` | 分 | 半场结束时逻辑队伍 1 的累计分 |
| `raw_winner_team` | 引擎整数 | `round_end` 原始诊断值，不是逻辑槽位 |
| `score_available` | 0/1 | 计分槽位对应地图分是否可用 |
| `result_status` | completed/abandoned | 必须与父 Round 当前状态一致 |
| `finalized_at` | Unix 秒 | 半场结果首次封口时间 |

第一半场计分槽位为 0，第二半场为 1。采集器优先用地图分变化判断，零分或变化不明确
时按半场编号回退。半场重开通过同一结果行的更高 `revision` 将旧状态修正为
`abandoned`。

### 8.2 `lps_versus_run_results`

| 字段 | 单位/枚举 | 读取规则 |
|---|---|---|
| `team_0_campaign_score` | 分 | 当前已观察到的逻辑队伍 0 累计分 |
| `team_1_campaign_score` | 分 | 当前已观察到的逻辑队伍 1 累计分 |
| `winner_team_slot` | -1/0/1/2 | 未知/队伍0/队伍1/平局 |
| `raw_winner_team` | 引擎整数 | `versus_match_finished` 原始诊断值 |
| `score_available` | 0/1 | 两边累计分是否可用 |
| `result_status` | active/completed/abandoned | 必须与父 Run 当前状态一致 |
| `finalized_at` | Unix 秒或 NULL | active 为 NULL，其他状态非空 |

只有 `result_status='completed' AND score_available=1` 才能产生 0/1/2 胜方；其他情况
`winner_team_slot` 必须是 -1。逻辑队伍槽位只在单个 Run 内稳定，不能跨战役追踪战队。

## 9. 读取侧不变量

Go 读取侧和离线检查必须遵守：

1. 聚合时对不同 Segment 行求和，但同一主键永远只读取当前绝对快照一次。
2. 不对 `revision`、`last_saved_at`、原始 winner 或翻转标记求和。
3. 职业明细用于分组展示和总计校验；总表仍是无需聚合即可读取的权威快照。
4. `score_available=0` 时不得把 -1 展示成真实 0 分。
5. `abandoned` 不等于失败方，不能进入正常胜率分母。
6. 公开 API 不得从 Session 连接中返回 IP；管理查询必须另设权限边界。
7. SQL 健康检查以 `database/queries/versus_contract_checks.sql` 为公共回归基线。

## 10. 明确延期

v1 不采集或推断：

- 跨 Run 稳定队伍、战队或成员关系；
- 连续推进曲线、逐时刻推进分或玩家推进贡献；
- 第三方计分插件的额外分项；
- MVP；
- 逐武器对抗统计、命中率、弹药、换弹；
- Skeet、Level、Deadstop、控制链和距离类技巧。

## 11. 兼容性规则

从 v0.6.6 起，三个 `0001_initial.sql` 中的 Versus v1 结构视为冻结：

- 不得修改已经存在字段的名称、类型、单位、所有者、归属或 Bot 语义；
- 不得复用职业 ID、状态枚举或特殊值；
- 任何结构变化必须新增 `0002` 或更高编号迁移，并同时提供 SQLite/MySQL/PostgreSQL；
- 只增加索引属于结构兼容变化，仍需新迁移，但不要求提高 `stats_version`；
- 字段改名、删除、拆分、合并或含义变化属于统计破坏性变化，必须保留 v1 数据并写入
  新字段/新表，同时提高到新的 `stats_version`；
- Go 侧只能读取明确支持的 `stats_version`，不得猜测未知版本。
