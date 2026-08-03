# 统计口径契约

状态：已确认

本文定义每一个统计值的业务含义。数据库字段、SourceMod事件和未来Go展示必须遵守相同口径。

## 1. 总体原则

- 只统计通过Steam认证的真人主体。
- Bot不拥有个人统计记录。
- 所有计数必须归属到明确的`segment_id`。
- PvE、对抗幸存者和对抗感染者使用独立统计表。
- 统计写入使用绝对快照，不使用可能在重试时重复累计的数据库增量。
- 第一阶段不保存逐击杀、逐伤害等原始事件流水。
- `coop`与`realism`可以在网页中组成PvE总览，但必须保留精确`game_mode`以供分别筛选。
- `versus`玩法统计不得与PvE统计直接合并排行。

## 2. 有效伤害

所有伤害统计只记录目标实际损失的生命值。

例如目标剩余10点生命，理论攻击为100点时，只记录10点。不得记录过量伤害。

伤害必须在游戏已经应用难度、护甲、减伤和友伤倍率之后计算。无法可靠确定实际生命损失的伤害不得猜测。

## 3. 通用排除项

以下伤害不计入玩家输出或承伤统计：

- 坠落；
- 溺水；
- 地图处决；
- 挂边掉落；
- 自伤；
- 无法找到可靠玩家所有者的环境火焰、爆炸或地图机关；
- 插件或管理员测试造成且被明确标记为调试的伤害。

如果火焰、酸液、爆炸或投掷物能够可靠追溯到真人所有者，则按实际攻击关系统计。

## 4. 击杀归属

第一阶段只记录最后击杀：

- 最后造成有效致死伤害的真人获得击杀；
- 其他参与者不获得助攻；
- 环境击杀不归属任何真人；
- Bot完成最后击杀时，不把击杀转移给伤害最高的真人；
- 自杀不计为玩家击杀。

助攻系统不属于第一阶段范围。

## 5. PvE统计

适用于`coop`和`realism`。

### 5.1 击杀

| 字段 | 含义 |
|---|---|
| `common_kills` | 普通感染者最后击杀 |
| `special_kills` | Smoker、Boomer、Hunter、Spitter、Jockey、Charger最后击杀 |
| `tank_kills` | Tank最后击杀 |
| `witch_kills` | Witch最后击杀 |

### 5.2 输出伤害

| 字段 | 含义 |
|---|---|
| `damage_to_special` | 对六种普通特感造成的有效伤害 |
| `damage_to_tank` | 对Tank造成的有效伤害 |
| `damage_to_witch` | 对Witch造成的有效伤害 |

第一阶段不统计对普通感染者的逐次伤害，只统计普通感染者击杀。

### 5.3 承伤和友伤

| 字段 | 含义 |
|---|---|
| `damage_taken_infected` | 来自感染者阵营的有效伤害 |
| `friendly_fire_to_humans` | 对真人幸存者造成的有效友伤 |
| `friendly_fire_to_bots` | 对幸存者Bot造成的有效友伤 |
| `friendly_fire_taken` | 自己受到的真人玩家有效友伤 |

真人和Bot友伤必须分开保存。

### 5.4 生存和救援

| 字段 | 含义 |
|---|---|
| `incapacitations` | 自己进入倒地状态的次数 |
| `deaths` | 自己真正死亡的次数 |
| `incap_revives` | 救起倒地队友的次数 |
| `ledge_rescues` | 拉起挂边队友的次数 |
| `defib_revives` | 使用电击器复活队友的次数 |
| `rescues_received` | 自己被上述方式救援的总次数 |

不同救援类型不得只保存成一个不可拆分的`rescues`字段。

### 5.5 治疗

| 字段 | 含义 |
|---|---|
| `medkits_used_self` | 对自己成功使用医疗包的次数 |
| `medkits_used_on_others` | 对队友成功使用医疗包的次数 |
| `medkit_healing_self` | 医疗包对自己实际恢复的真实生命 |
| `medkit_healing_others` | 医疗包对队友实际恢复的真实生命 |
| `pills_used` | 自己成功使用止痛药的次数 |
| `adrenaline_used` | 自己成功使用肾上腺素的次数 |
| `temporary_health_received` | 通过止痛药和肾上腺素实际获得的临时生命 |

网页中的“治疗量”默认只指医疗包实际恢复的真实生命。临时生命必须作为独立指标展示。

物品转交不等于治疗，第一阶段不记录医疗物资转交关系。

### 5.6 章节和战役

| 字段 | 含义 |
|---|---|
| `chapter_participations` | 满足模式契约完成归属条件的章节数 |
| `chapter_completions_alive` | 章节完成时仍存活的次数 |
| `chapter_completions_dead` | 章节完成时已死亡但仍属于参与者的次数 |
| `campaign_completions` | 参与并完成最终章节的战役次数 |

Run和Round层面另行保存团队结果，不通过个人计数反推团队结果。

### 5.7 特感职业明细

六种普通特感分别保存击杀数和有效伤害：

```text
Smoker / Boomer / Hunter / Spitter / Jockey / Charger
```

职业击杀之和必须等于 `special_kills`，职业伤害之和必须等于
`damage_to_special`。Tank 与 Witch 继续使用独立字段，不并入普通特感。

### 5.8 武器、近战和投掷物

设备明细以固定数值 ID 写入 `lps_pve_segment_equipment_stats`。明细行为：

- 官方枪械分别记录普通感染者、普通特感、Tank、Witch 击杀，枪械爆头击杀，以及对普通特感、Tank、Witch 的有效伤害；
- 所有未知或第三方枪械只进入一个固定的 `Other Firearm` 桶，不按 classname 创建新行；
- 官方近战武器分别记录上述击杀与伤害，但不记录命中、挥砍、命中率或斩首；
- 自定义近战武器完全忽略；
- Molotov、Pipe Bomb 和 Vomit Jar 分别记录成功投掷次数，并记录能够可靠直接归属的击杀与伤害；
- 自定义投掷物完全忽略；
- 枪械不记录射击数、命中数、命中率、弹药消耗、换弹次数或对普通感染者伤害；
- 装备类别统计和全部装备总计由未来 Go 查询从精确设备行聚合，不在采集器中重复保存。

`actions` 当前只表示三种官方投掷物的成功投掷次数；其他设备行该字段为 0。

### 5.9 特感控制和团队解救

对 Smoker、Hunter、Jockey 和 Charger 分别保存：

- 真人幸存者被成功控制的次数；
- 从控制开始到结束的秒数；
- 真人队友结束该控制的解救次数。

解救者必须是另一个真人幸存者。由本人挣脱不算团队解救；无法可靠找到解救者的结束事件不猜测归属。击杀当前控制者和带有明确解救者字段的停止事件可以计入。

### 5.10 技巧统计

| 字段 | 精确定义 |
|---|---|
| `melee_tongue_self_cuts` | 被 Smoker 控制的本人使用官方近战武器切断自己的舌头；替队友断舌不计入 |
| `tank_rocks_destroyed` | 真人幸存者使用枪械在空中摧毁 `tank_rock` 实体 |
| `witch_oneshots` | `witch_killed.oneshot` 明确为真的 Witch 击杀 |
| `witch_solo_kills` | 从首次有效伤害到死亡只有同一名真人贡献者，且该玩家完成最后击杀 |

技巧统计只接受上述可验证信号，不通过时间窗口或动画状态猜测。

### 5.11 Boss 参与和弹药升级包

真人首次对某个 Tank/Witch 造成有效伤害时增加对应 `encounters`；该 Boss
死亡且该玩家仍可归属到真人 Segment 时增加对应 `kill_participations`。
最后击杀继续由 `tank_kills` / `witch_kills` 单独表达。

成功部署燃烧弹药包和高爆弹药包分别写入
`incendiary_packs_deployed` 与 `explosive_packs_deployed`。激光瞄准器获取不在统计范围内。

### 5.12 目标互动、补给和失能时长

| 字段 | 精确定义 |
|---|---|
| `objective_interactions` | 真人幸存者成功完成白名单目标实体的一次互动；同一实体每个 Round 最多归属一次 |
| `ammo_pile_uses` | 真人幸存者从 `weapon_ammo_spawn` 实际补充弹药并触发 `ammo_pickup` 的次数 |
| `incapacitated_seconds` | 真人幸存者 Segment 内处于普通倒地状态的累计整秒数，不含挂边时间 |
| `ledge_hanging_seconds` | 真人幸存者 Segment 内处于挂边状态的累计整秒数 |
| `black_white_teammates_restored` | 真人使用医疗包成功把开始治疗时为黑白状态的另一名幸存者恢复为彩色的次数 |

目标互动只接受可表达“已完成”的实体输出：

- `func_button` / `func_rot_button` 的 `OnIn`；
- `func_button_timed` / `func_buildable_button` 的 `OnTimeUp`；
- `momentary_rot_button` 的 `OnFullyClosed`；
- `point_script_use_target` 的 `OnUseFinished`。

不监听泛化的 `player_use`，不把 `finale_start`、普通开门、汽油桶/可乐等
`point_prop_use_target` 搬运目标自动算作目标互动。输出没有明确真人幸存者 activator
时不猜测归属。

黑白转彩色只给施救真人记账；被治疗队友可以是真人或 Bot，但自疗不计入。
插件在 `heal_begin` 捕获黑白状态，并在同一次 `heal_success` 后验证状态确实解除。
治疗中断、目标不匹配或无法验证均不记录。

倒地和挂边时长由状态开始/结束事件结算，不使用每秒轮询。周期保存会先把已经过去的
完整秒数并入绝对快照，并保留不足一秒的余数；重复保存不得重复累计。

## 6. 对抗幸存者统计

只统计玩家作为幸存者参与对抗半场时的数据。

### 6.1 击杀

| 字段 | 含义 |
|---|---|
| `common_kills` | 普通感染者击杀 |
| `human_special_kills` | 真人控制普通特感击杀 |
| `bot_special_kills` | Bot控制普通特感击杀 |
| `human_tank_kills` | 真人控制Tank击杀 |
| `bot_tank_kills` | Bot控制Tank击杀 |

对抗中的真人和Bot目标必须分开保存，网页可以在查询时合并。

### 6.2 伤害

| 字段 | 含义 |
|---|---|
| `damage_to_human_special` | 对真人控制普通特感的有效伤害 |
| `damage_to_bot_special` | 对Bot控制普通特感的有效伤害 |
| `damage_to_human_tank` | 对真人控制Tank的有效伤害 |
| `damage_to_bot_tank` | 对Bot控制Tank的有效伤害 |
| `damage_taken_infected` | 来自感染者阵营的有效伤害 |
| `friendly_fire_to_humans` | 对真人队友造成的有效伤害 |
| `friendly_fire_to_bots` | 对Bot队友造成的有效伤害 |
| `friendly_fire_taken` | 自己受到的真人玩家有效友伤 |

### 6.3 生存、救援和治疗

对抗幸存者复用PvE的以下定义：

- `incapacitations`；
- `deaths`；
- `incap_revives`；
- `ledge_rescues`；
- `defib_revives`；
- 医疗包和临时生命相关字段。

这些数据只进入对抗幸存者统计表，不进入PvE统计表。

### 6.4 特感职业战斗明细

Smoker、Boomer、Hunter、Spitter、Jockey、Charger、Tank 使用固定职业 ID 保存：

- 击杀真人控制感染者次数；
- 击杀 Bot 控制感染者次数；
- 对真人控制感染者造成的有效伤害；
- 对 Bot 控制感染者造成的有效伤害。

每个幸存者 Segment 最多七个职业行。未知职业不创建动态 ID 或名称行。1～6 职业
明细必须严格聚合回普通特感真人/Bot 总计，Tank 职业明细必须严格聚合回 Tank
真人/Bot 总计。总计与明细在同一次内存更新中共同增加，检查工具将任何不一致视为
数据健康错误。

### 6.5 Witch、投掷物、升级包和技巧

对抗幸存者总表额外保存：

| 字段 | 含义 |
|---|---|
| `witch_kills` | 对 Witch 完成最后击杀的次数 |
| `damage_to_witch` | 对 Witch 造成的有效生命损失 |
| `molotovs_thrown` | 投出官方 Molotov 的次数 |
| `pipe_bombs_thrown` | 投出官方 Pipe Bomb 的次数 |
| `vomit_jars_thrown` | 投出官方 Vomit Jar 的次数 |
| `incendiary_packs_deployed` | 成功部署燃烧弹药包的次数 |
| `explosive_packs_deployed` | 成功部署高爆弹药包的次数 |
| `melee_tongue_self_cuts` | 使用官方近战武器切断控制自己的 Smoker 舌头次数 |
| `tank_rocks_destroyed` | 摧毁飞行中 Tank 石头的次数 |
| `witch_oneshots` | 单次伤害击杀此前满血 Witch 的次数 |
| `witch_solo_kills` | 本次 Witch 击杀仅有该真人幸存者造成过有效伤害的次数 |

投掷物和升级包仅接受固定官方类型，未知/第三方名称不创建维度或记录。Witch 参与者
使用受服务器最大客户端数约束的内存槽位；伤害、投掷和技巧事件只更新绝对快照，
不直接访问数据库。

### 6.6 明确不采集的幸存者明细

对抗首版不创建逐武器统计表，也不记录：

- 枪械射击数、命中数、命中率、弹药消耗和换弹；
- 对普通感染者的武器伤害；
- 近战命中数、命中率和斩首；
- 自定义武器、近战或投掷物动态名称；
- Skeet、Level、Deadstop 等难以稳定归属的技术动作。

## 7. 对抗感染者统计

只统计玩家作为感染者参与对抗半场时的数据。

### 7.1 总计字段

| 字段 | 含义 |
|---|---|
| `spawn_count` | 以可操控感染者成功出生的次数 |
| `damage_to_human_survivors` | 对真人幸存者造成的有效伤害 |
| `damage_to_bot_survivors` | 对Bot幸存者造成的有效伤害 |
| `human_survivor_incaps` | 造成真人幸存者倒地的次数 |
| `bot_survivor_incaps` | 造成Bot幸存者倒地的次数 |
| `human_survivor_kills` | 造成真人幸存者死亡的次数 |
| `bot_survivor_kills` | 造成Bot幸存者死亡的次数 |

Tank属于对抗感染者身份。上述总计是读取侧无需聚合即可使用的权威快照。

### 7.2 职业明细

Smoker、Boomer、Hunter、Spitter、Jockey、Charger、Tank 使用固定职业 ID
分别保存与总计相同的七项指标：

- 出生次数；
- 对真人/Bot 幸存者的有效伤害；
- 造成真人/Bot 幸存者倒地的次数；
- 造成真人/Bot 幸存者死亡的次数。

每个感染者 Segment 最多七个职业行。未知职业不创建动态 ID 或名称行，也不记录
职业明细。正常数据中，每一项职业明细之和必须严格等于对应总计；检查工具将不一致
视为数据健康错误。Bot 感染者仍不拥有个人 Segment 或职业统计。

### 7.3 控制与能力效果

Smoker、Hunter、Jockey 和 Charger 分别保存对真人/Bot 幸存者成功控制的次数与控制
秒数。控制从对应成功事件开始，持续到释放、获救、受害者或控制者死亡、换队、Segment
结束或 Round 结束。Charger 的搬运转入捶打属于同一次控制，不重复计数。重复开始或结束
事件必须幂等。

Boomer 使用 `player_now_it.by_boomer` 记录实际被胆汁命中的真人/Bot 幸存者人数；普通
胆汁罐不归给 Boomer。Spitter 只记录 `insect_swarm` 对真人/Bot 幸存者造成的有效生命
损失，酸液池在 Spitter 死亡后仍归给原所有者。抓、骑、撞击、喷吐和酸液事件只更新
固定内存快照，不直接访问数据库。

以下指标继续延期：

- Hunter 扑击距离；
- Jockey 骑乘距离；
- Charger 冲锋距离；
- 控制链；
- Skeet、Level 和 Deadstop 等技术动作。

## 8. 对抗比赛结果

对抗结果直接读取 L4D2 `CTerrorGameRules` 的权威整数值，不根据玩家坐标、存活人数或
自定义公式重新计算。结果使用 Run 内部的逻辑队伍槽位 `team_0` / `team_1`，它们不是
跨 Run 稳定的战队身份，也不能被解释为当前幸存者/感染者阵营。

每个已结束半场在 `lps_versus_round_results` 保存：

| 字段 | 含义 |
|---|---|
| `scoring_team_slot` | 本半场担任幸存者并获得地图分的逻辑队伍槽位，0 或 1 |
| `teams_flipped` | 半场开始时引擎的队伍翻转标记，诊断字段 |
| `team_0_map_score` / `team_1_map_score` | 引擎当前保存的本章节/半场得分；尚未出场可以是 -1 |
| `team_0_campaign_score` / `team_1_campaign_score` | 该时刻的战役累计分 |
| `score_available` | 本半场计分队伍的地图分是否可用 |
| `raw_winner_team` | `round_end` 的原始引擎阵营编号，只供诊断 |
| `result_status` | `completed` 或因重开/异常结束修正为 `abandoned` |

每个对抗 Run 在 `lps_versus_run_results` 保存累计分和结果。半场结束后先写入 `active`
快照；比赛正常结束时写入 `completed`，比较两边累计分得到 `winner_team_slot`：0/1 为
获胜槽位，2 为平局，-1 为未知。异常结束写入 `abandoned`，不宣布胜者。插件异常退出
后，下次启动会把遗留的 `active` 结果恢复为 `abandoned`。

`raw_winner_team` 保存 `versus_match_finished` 的原始值，只用于排查引擎或第三方模式差异；
网页不得直接用它代替 `winner_team_slot`。`score_available=0` 时不得把 -1 当作真实分数。

第一阶段仍不保存或推算：

- 跨 Run 稳定队伍或战队成员关系；
- 逐时刻推进距离、进度曲线或逐玩家推进贡献；
- 第三方计分插件的额外分项；
- MVP归属。

## 9. 统计版本

每张玩法统计表必须包含`stats_version`。第一阶段固定为：

```text
stats_version = 1
```

以后统计口径发生不兼容变化时必须增加版本，不得悄悄改变旧数据含义。
