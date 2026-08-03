# 数据库结构契约

状态：已确认

本文定义SQLite、MySQL和PostgreSQL必须共同表达的数据模型。具体DDL由后续`database/migrations/<driver>/`实现。

## 1. 总体规则

- SQLite、MySQL和PostgreSQL必须产生语义等价的表和字段。
- 表名固定使用`lps_`前缀，不允许通过服务器配置更改。
- 不使用数据库自增ID。
- 不依赖数据库日期函数，时间由插件以Unix秒写入。
- 不使用JSON作为第一阶段核心字段。
- 不保存逐事件流水，只保存生命周期记录和聚合快照。
- 所有动态字符串必须通过当前数据库驱动转义。
- 所有玩法写入必须异步执行。
- 数据永久保留；第一阶段不提供自动删除Session、Segment或IP的任务。

## 2. 通用字段类型

| 语义 | 统一类型 |
|---|---|
| SteamID64 | `VARCHAR(32)` |
| 生命周期ID | `VARCHAR(128)` |
| server_key | `VARCHAR(64)` |
| 玩家昵称 | `VARCHAR(128)` |
| IPv4/IPv6 | `VARCHAR(45)` |
| 状态和枚举 | `VARCHAR(32)` |
| Unix时间 | `BIGINT` |
| 持续秒数 | `BIGINT` |
| 计数和revision | `BIGINT` |
| 布尔值 | `INTEGER`，仅允许0或1 |

SteamID64必须作为字符串处理，不得转为SourcePawn整数或数据库数值主键。

## 3. ID生成

每台服务器配置唯一`server_key`。每次插件启动生成：

```text
boot_id = <server_key>:<启动Unix时间>:<随机十六进制>
```

其他ID使用`boot_id`和进程内单调序列生成：

```text
session_id = <boot_id>:session:<sequence>
run_id     = <boot_id>:run:<sequence>
round_id   = <boot_id>:round:<sequence>
segment_id = <boot_id>:segment:<sequence>
```

ID不是安全令牌，但必须在共享数据库中全局唯一。所有ID在发送异步插入前就必须确定。

## 4. 表结构

### 4.1 `lps_schema_migrations`

| 字段 | 说明 |
|---|---|
| `version` | 迁移版本，主键 |
| `name` | 不可变的迁移名称 |
| `applied_at` | 应用时间 |

已经发布并应用的迁移文件不得修改。结构变化必须增加更高版本的新迁移。

### 4.2 `lps_servers`

| 字段 | 说明 |
|---|---|
| `server_key` | 服务器唯一键，主键 |
| `display_name` | 服务器显示名称 |
| `first_seen_at` | 首次注册时间 |
| `last_seen_at` | 最后心跳时间 |

多个服务器共用数据库时，`server_key`不得重复。插件不得根据IP和端口自动推导该值。

### 4.3 `lps_server_boots`

| 字段 | 说明 |
|---|---|
| `boot_id` | 启动实例ID，主键 |
| `server_key` | 所属服务器 |
| `started_at` | 启动时间 |
| `ended_at` | 正常结束时间，可空 |
| `last_heartbeat_at` | 最近保存或心跳时间 |
| `status` | `active`、`closed`、`abandoned` |

新boot完成迁移后，必须把同一`server_key`下其他仍为`active`的boot及其开放生命周期恢复为`abandoned`。

### 4.4 `lps_players`

| 字段 | 说明 |
|---|---|
| `steam_id` | SteamID64，主键 |
| `last_name` | 最近观察到的昵称 |
| `first_seen_at` | 首次出现时间 |
| `last_seen_at` | 最后出现时间 |

玩家主表不重复保存`last_ip`。最近IP可以通过该玩家最新Session查询。

### 4.5 `lps_sessions`

| 字段 | 说明 |
|---|---|
| `session_id` | 主键 |
| `boot_id` | 所属启动实例 |
| `server_key` | 所属服务器 |
| `steam_id` | 玩家SteamID64 |
| `player_name` | Session开始时的昵称快照 |
| `ip_address` | 服务器观察到的IP，不含端口 |
| `started_at` | 开始时间 |
| `ended_at` | 结束时间，可空 |
| `last_saved_at` | 最近成功持久化时间 |
| `connected_seconds` | 连接时间 |
| `active_play_seconds` | 有效参赛时间 |
| `status` | `active`、`closed`、`abandoned` |
| `disconnect_reason` | 规范化结束原因 |
| `revision` | 绝对快照版本 |

IP保存规则：

- 使用服务器实际观察到的远端地址；
- 去除端口；
- IPv4和IPv6都保存为规范字符串；
- 不进行地理位置推断；
- 不写入正常运行日志；
- Go公开API和前端默认不得返回或展示IP；
- IP永久保留，直到未来明确引入明细清理政策。

### 4.6 `lps_runs`

| 字段 | 说明 |
|---|---|
| `run_id` | 主键 |
| `boot_id` | 创建它的启动实例 |
| `server_key` | 所属服务器 |
| `mode_family` | `pve`或`versus` |
| `game_mode` | `coop`、`realism`或`versus` |
| `campaign_key` | 可识别时保存战役键，否则为空字符串 |
| `started_at` | 开始时间 |
| `ended_at` | 结束时间，可空 |
| `last_saved_at` | 最近保存时间 |
| `status` | `active`、`completed`、`abandoned` |
| `round_count` | 已创建Round数 |
| `completed_round_count` | 完成Round数 |
| `failed_round_count` | 失败Round数 |
| `revision` | 绝对快照版本 |

### 4.7 `lps_rounds`

| 字段 | 说明 |
|---|---|
| `round_id` | 主键 |
| `run_id` | 所属Run |
| `server_key` | 所属服务器 |
| `mode_family` | `pve`或`versus` |
| `map_name` | 地图内部名称 |
| `round_seq` | Run内递增序号 |
| `map_seq` | Run内章节序号 |
| `attempt_no` | 当前章节或半场尝试次数 |
| `half_no` | 对抗为1或2，PvE为0 |
| `started_at` | 开始时间 |
| `ended_at` | 结束时间，可空 |
| `last_saved_at` | 最近保存时间 |
| `status` | `active`、`completed`、`failed`、`abandoned` |
| `revision` | 绝对快照版本 |

第一阶段不加入对抗队伍分数和胜负字段。

### 4.8 `lps_player_segments`

| 字段 | 说明 |
|---|---|
| `segment_id` | 主键 |
| `session_id` | 所属Session |
| `run_id` | 所属Run |
| `round_id` | 所属Round |
| `server_key` | 所属服务器 |
| `steam_id` | 玩家SteamID64 |
| `side` | `survivor`或`infected` |
| `started_at` | 开始时间 |
| `ended_at` | 结束时间，可空 |
| `last_saved_at` | 最近保存时间 |
| `active_play_seconds` | 本Segment有效参赛时间 |
| `status` | `active`、`closed`、`abandoned` |
| `revision` | 绝对快照版本 |

观战和闲置不创建独立Segment。

### 4.9 `lps_pve_segment_stats`

主键为`segment_id`，并包含：

```text
stats_version
last_saved_at
common_kills
special_kills
tank_kills
witch_kills
damage_to_special
damage_to_tank
damage_to_witch
damage_taken_infected
friendly_fire_to_humans
friendly_fire_to_bots
friendly_fire_taken
incapacitations
deaths
incap_revives
ledge_rescues
defib_revives
rescues_received
medkits_used_self
medkits_used_on_others
medkit_healing_self
medkit_healing_others
pills_used
adrenaline_used
temporary_health_received
chapter_participations
chapter_completions_alive
chapter_completions_dead
campaign_completions
smoker_kills
boomer_kills
hunter_kills
spitter_kills
jockey_kills
charger_kills
damage_to_smoker
damage_to_boomer
damage_to_hunter
damage_to_spitter
damage_to_jockey
damage_to_charger
smoker_controls_received
hunter_controls_received
jockey_controls_received
charger_controls_received
smoker_controlled_seconds
hunter_controlled_seconds
jockey_controlled_seconds
charger_controlled_seconds
smoker_saves
hunter_saves
jockey_saves
charger_saves
melee_tongue_self_cuts
tank_rocks_destroyed
witch_oneshots
witch_solo_kills
tank_encounters
tank_kill_participations
witch_encounters
witch_kill_participations
incendiary_packs_deployed
explosive_packs_deployed
objective_interactions
ammo_pile_uses
incapacitated_seconds
ledge_hanging_seconds
black_white_teammates_restored
revision
```

### 4.10 `lps_pve_segment_equipment_stats`

复合主键为 `(segment_id, equipment_id)`，并包含：

```text
stats_version
last_saved_at
actions
common_kills
special_kills
tank_kills
witch_kills
headshot_kills
damage_to_special
damage_to_tank
damage_to_witch
revision
```

`equipment_id` 是公开后不可复用的稳定数值标识。当前 ID：

| 范围 | 内容 |
|---|---|
| 1 | `Other Firearm`，所有未知/第三方枪械共享 |
| 2–4 | 单手枪、双持手枪、马格南 |
| 5–7 | Uzi、消音冲锋枪、MP5 |
| 8–11 | 木喷、Chrome、Auto、SPAS |
| 12–15 | M16、AK-47、SCAR、SG552 |
| 16–19 | Hunting Rifle、Military Sniper、Scout、AWP |
| 20–24 | 榴弹发射器、M60、电锯、固定机枪、Minigun |
| 25–37 | 13 种官方近战脚本 |
| 38–40 | Molotov、Pipe Bomb、Vomit Jar |

采集器只写精确设备行。类别聚合和全部设备总计由读取侧计算，避免重复数据漂移。

### 4.11 `lps_versus_survivor_stats`

主键为`segment_id`，并包含：

```text
stats_version
last_saved_at
common_kills
human_special_kills
bot_special_kills
human_tank_kills
bot_tank_kills
damage_to_human_special
damage_to_bot_special
damage_to_human_tank
damage_to_bot_tank
damage_taken_infected
friendly_fire_to_humans
friendly_fire_to_bots
friendly_fire_taken
incapacitations
deaths
incap_revives
ledge_rescues
defib_revives
rescues_received
medkits_used_self
medkits_used_on_others
medkit_healing_self
medkit_healing_others
pills_used
adrenaline_used
temporary_health_received
witch_kills
damage_to_witch
molotovs_thrown
pipe_bombs_thrown
vomit_jars_thrown
incendiary_packs_deployed
explosive_packs_deployed
melee_tongue_self_cuts
tank_rocks_destroyed
witch_oneshots
witch_solo_kills
revision
```

### 4.12 `lps_versus_survivor_infected_class_stats`

复合主键为 `(segment_id, infected_class)`。每个幸存者 Segment 最多产生七行，
并包含：

```text
stats_version
last_saved_at
human_controller_kills
bot_controller_kills
damage_to_human_controllers
damage_to_bot_controllers
revision
```

`infected_class` 使用 1～6 和 8 七个固定职业 ID，定义与感染者职业表一致。
未知职业不创建行。1～6 职业行的四项合计必须分别等于幸存者总表中的普通特感
真人/Bot 击杀与伤害；职业 8 必须等于对应 Tank 总计。

### 4.13 `lps_versus_infected_stats`

主键为`segment_id`，并包含：

```text
stats_version
last_saved_at
spawn_count
damage_to_human_survivors
damage_to_bot_survivors
human_survivor_incaps
bot_survivor_incaps
human_survivor_kills
bot_survivor_kills
revision
```

### 4.14 `lps_versus_infected_class_stats`

复合主键为 `(segment_id, infected_class)`。每个感染者 Segment 最多产生七行，
并包含：

```text
stats_version
last_saved_at
spawn_count
damage_to_human_survivors
damage_to_bot_survivors
human_survivor_incaps
bot_survivor_incaps
human_survivor_kills
bot_survivor_kills
human_survivor_controls
bot_survivor_controls
human_survivor_control_seconds
bot_survivor_control_seconds
human_survivor_ability_hits
bot_survivor_ability_hits
human_survivor_ability_damage
bot_survivor_ability_damage
revision
```

`infected_class` 使用游戏稳定职业编号：

| ID | 职业 |
|---:|---|
| 1 | Smoker |
| 2 | Boomer |
| 3 | Hunter |
| 4 | Spitter |
| 5 | Jockey |
| 6 | Charger |
| 8 | Tank |

未知职业不创建行。职业行是同一绝对快照的有界明细，其中出生、伤害、倒地和击杀
合计必须分别等于 `lps_versus_infected_stats` 中对应总计。

能力字段按职业限定：Smoker、Hunter、Jockey、Charger 使用控制次数和控制秒数；
Boomer 使用能力命中人数；Spitter 使用酸液有效伤害。每项均拆分真人与 Bot 幸存者
目标。其他职业对应字段必须为 0，Spitter 能力伤害不得大于该职业的总有效伤害。

## 5. 逻辑关联和外键

第一阶段只定义逻辑外键，不要求数据库建立物理`FOREIGN KEY`约束。

原因：

- 三种数据库的约束和级联行为存在差异；
- 异步写入必须严格控制父子插入顺序；
- 数据暂不删除，不依赖级联删除；
- 物理外键可以在三种驱动稳定后通过新迁移评估。

必须为以下查询建立普通索引：

```text
lps_sessions(server_key, started_at)
lps_sessions(steam_id, started_at)
lps_runs(server_key, started_at)
lps_rounds(run_id, round_seq)
lps_player_segments(round_id, steam_id)
lps_player_segments(steam_id, started_at)
lps_pve_segment_equipment_stats(equipment_id, segment_id)
lps_versus_survivor_infected_class_stats(infected_class, segment_id)
lps_versus_infected_class_stats(infected_class, segment_id)
```

不同数据库允许使用不同建索引语法，但索引语义必须一致。

## 6. 幂等写入和revision

所有可更新记录使用绝对快照。

内存对象维护：

```text
current_revision
persisted_revision
dirty
```

统计发生变化时增加`current_revision`。开始异步保存时捕获本次revision。

回调成功后：

- 如果当前revision等于已提交revision，则清除dirty；
- 如果写入期间数据又发生变化，保持dirty并等待下一次保存；
- 查询失败不得推进`persisted_revision`。

禁止使用无法安全重试的数据库增量，例如：

```sql
UPDATE ... SET kills = kills + 1
```

## 7. 保存调度

默认配置：

```text
base_interval = 300秒
jitter = 0～60秒
```

一台服务器只使用一个常规刷新计时器。每次刷新只提交dirty记录，并尽量组成一个异步事务。

特殊保存时机由模式契约定义。保存进行中再次收到刷新请求时，只设置补充刷新标志，不并发提交相同对象。

## 8. 数据库故障

第一阶段使用内存dirty状态：

- 数据库不可用时继续在内存中统计；
- 使用有上限的指数退避重连；
- 连接恢复后补写当前绝对快照；
- 不踢玩家；
- 不创建无限增长的本地失败日志；
- 数据库故障期间如果服务器崩溃，允许丢失尚未持久化的数据。

推荐重连间隔：

```text
5秒、15秒、30秒、60秒，之后保持60秒
```

有界本地spool属于后续阶段。未来实现时必须限制文件数、总容量和记录数。

## 9. 日志和磁盘边界

第一阶段不创建插件专属常规日志文件。

只通过SourceMod日志记录：

- 启动和驱动识别；
- 数据库连接失败与恢复；
- 迁移失败与成功；
- 事务失败；
- 内存队列达到边界；
- 管理员操作。

不得记录：

- IP地址；
- 每次击杀和伤害；
- 每个玩家的周期保存成功；
- 每条SQL成功；
- 数据库密码和连接字符串。

相同错误必须限流：第一次立即记录，之后最多每5分钟输出一次合并摘要，恢复时输出一次恢复消息。

如果以后增加专属诊断日志，只能写入插件自己的子目录，并同时实现保留天数和最大总容量。插件不得清理SourceMod共享日志。

## 10. 数据保留与隐私

- 玩家身份、昵称快照、IP、Session、Run、Round、Segment和统计默认永久保留。
- 第一阶段不实现自动清理。
- Go侧建立可靠的永久聚合表之前，不得删除Session和Segment明细。
- 未来引入清理时必须先完成可验证聚合，再按时间范围分批删除明细。
- IP只能用于服务器管理用途；未来公开API和前端默认不得返回IP。
- 数据库账号应遵循最小权限。未来Go服务使用只读账号，SourceMod采集器使用写入账号。

## 11. 迁移所有权

数据库迁移的唯一来源是：

```text
database/migrations/sqlite/
database/migrations/mysql/
database/migrations/pgsql/
```

SourceMod采集器负责检查和执行迁移。未来Go服务是数据库只读消费者，不得维护另一份独立迁移历史。
