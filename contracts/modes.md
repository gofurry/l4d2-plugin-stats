# 模式与生命周期契约

状态：已确认

本文定义采集范围，以及 Session、Run、Round 和 Segment 的开始、延续与结束规则。实现不得通过临时事件处理逻辑改变这些语义。

## 1. 支持范围

插件只在 `mp_gamemode` 精确等于以下值时采集数据：

| 游戏模式 | 统计族 |
|---|---|
| `coop` | `pve` |
| `realism` | `pve` |
| `versus` | `versus` |

其他模式完全不记录，包括玩家身份、IP、连接时间和玩法统计。

当服务器从受支持模式切换到不受支持模式时，插件必须结束当前活动的 Session、Run、Round 和 Segment。重新切回受支持模式时必须创建新的生命周期对象，不得延续旧对象。

## 2. 真人和 Bot

- 只有通过 Steam 认证并能取得 SteamID64 的真人才是统计主体。
- Bot 不得拥有 `player`、`session` 或 `segment` 记录。
- 真人对 Bot 造成的行为可以按统计契约计入真人数据。
- Bot 对真人造成的行为可以计入真人的承伤或被控制数据，但不得为 Bot 创建攻击者身份。
- 无法取得有效 SteamID64 时不得创建临时玩家身份，也不得退回使用客户端槽位、昵称或 IP 作为主键。

## 3. 时间定义

每名玩家同时保存两种时间：

### 3.1 连接时间

`connected_seconds`表示玩家在受支持模式中保持已连接状态的时间，包括：

- 正常参赛；
- 死亡等待；
- 感染者复活等待；
- 观战；
- 闲置且角色由 Bot 接管。

### 3.2 有效参赛时间

`active_play_seconds`只计算玩家被分配到幸存者或感染者阵营、且没有处于闲置 Bot 接管状态的时间。

死亡等待和感染者复活等待仍属于有效参赛时间。纯观战和闲置时间不属于有效参赛时间。

所有时间以Unix秒保存。持续时间不得通过简单的`ended_at - started_at`替代，必须根据生命周期累计，以排除闲置和不受支持模式。

## 4. Session

Session表示真人玩家在受支持模式中的一次连续连接记录。

### 4.1 开始

同时满足以下条件时创建Session：

1. 玩家已经通过Steam认证；
2. 玩家是真人；
3. 当前模式受支持；
4. 当前客户端槽位尚无活动Session。

玩家已经在线时，如果服务器从不受支持模式切换到受支持模式，也应在模式确认后创建Session。

### 4.2 结束

以下情况结束Session：

- 玩家断开；
- 切换到不受支持模式；
- 插件正常卸载或重载时尽力关闭；
- 服务器正常关闭时尽力关闭；
- 无法继续确认Steam身份。

断线重连必须创建新Session。Session可以跨正常地图切换延续。

正常卸载或关闭时如果数据库写入未能完成，下次启动仍按异常恢复规则标记，不得假设关闭操作一定已经持久化。

### 4.3 异常恢复

服务器崩溃或进程被强制终止时，Session可能没有`ended_at`。

插件下次启动并连接数据库后，必须将相同`server_key`下、属于旧`boot_id`且仍为`active`的Session标记为：

```text
status = abandoned
ended_at = last_saved_at
```

不得把异常恢复伪装成正常断开。

## 5. Run

Run表示一次完整战役流程。每个Run只能属于一个统计族和一个精确游戏模式。

### 5.1 PvE Run

在`coop`或`realism`中，以下情况创建新Run：

- 进入受支持模式且不存在可延续Run；
- 没有正常`map_transition`依据的新地图开始；
- 游戏模式从`coop`切换为`realism`或反向切换；
- 上一个Run已经完成或放弃；
- 插件重载或服务器重启后重新开始采集。

以下情况延续当前Run：

- 正常`map_transition`进入下一章节；
- `mission_lost`后重试当前章节。

以下情况结束Run：

| 情况 | Run状态 |
|---|---|
| `finale_win` | `completed` |
| 手动换图且无正常过图依据 | `abandoned` |
| 切换模式 | `abandoned` |
| 插件重载或服务器正常关闭 | `abandoned` |
| 崩溃后恢复旧记录 | `abandoned` |

团灭重试属于同一个Run。

### 5.2 Versus Run

在`versus`中，一整场对抗战役属于一个Run。

- 正常地图推进延续Run；
- 同一地图的两个半场属于同一个Run；
- 半场重开仍属于同一个Run；
- 最终章节的第二半场正常结束时可以将Run标记为`completed`；
- 第一阶段不计算稳定的队伍A/队伍B归属，也不计算最终胜负；
- 手动换图、切换模式、插件重载和服务器终止将Run标记为`abandoned`。

## 6. Round

Round表示一次可独立结算的玩法阶段。

### 6.1 PvE Round

在PvE中，一张地图的一次尝试就是一个Round。

- 地图开始并进入可玩状态时创建；
- `mission_lost`将当前Round标记为`failed`；
- 重试同一章节时创建新Round，并增加`attempt_no`；
- 正常过图将Round标记为`completed`；
- 非正常换图、模式切换或进程终止将Round标记为`abandoned`。

### 6.2 Versus Round

在对抗中，一张地图的一个半场就是一个Round。

Round至少保存：

```text
round_seq
map_seq
attempt_no
half_no
map_name
started_at
ended_at
status
```

- `round_seq`在Run内单调递增；
- `half_no`为1或2；
- 半场重开创建新Round并增加`attempt_no`；
- 第一阶段不保存对抗队伍总分、推进距离和最终胜负。

## 7. Segment

Segment表示一个真人在某个Round中，以同一统计身份连续参与的一段时间。

### 7.1 开始

以下情况创建Segment：

- 真人进入幸存者阵营；
- 真人进入感染者阵营；
- 中途加入当前Round；
- 从闲置恢复并重新接管角色；
- 从观战进入可参赛阵营；
- 阵营切换后开始新的统计身份。

### 7.2 结束

以下情况结束Segment：

- 玩家断开；
- 玩家进入观战；
- 玩家闲置并由Bot接管；
- 幸存者和感染者阵营互换；
- Round结束；
- 模式变为不受支持；
- 插件或服务器终止。

死亡、感染者复活等待和幸存者等待救援不会单独结束Segment。

### 7.3 中途加入

中途加入者只统计Segment开始后的数据和时间。不得推算加入前的Round数据，也不得把完整Round时长计给中途加入者。

## 8. 章节完成归属

PvE章节完成时，符合以下条件的真人获得个人章节参与完成记录：

1. 当前仍连接服务器；
2. 当前属于幸存者阵营；
3. 当前Round中存在有效Segment。

玩家在章节完成时已经死亡仍算参与完成，同时保存：

```text
alive_at_completion = 0
```

存活玩家保存：

```text
alive_at_completion = 1
```

观战者、闲置者和Bot不获得个人章节完成记录。

## 9. 标识符和启动实例

每次插件启动生成新的`boot_id`。所有Session、Run、Round和Segment都必须关联：

```text
server_key
boot_id
```

`server_key`由服主手动配置，并且在共享数据库中的所有服务器之间必须唯一。

推荐约束：

```text
长度：1～64
字符：A-Z a-z 0-9 . _ -
```

## 10. 保存触发点

常规保存由服务器级单一计时器触发：

```text
300秒 + 0～60秒随机抖动
```

以下情况请求特殊保存：

- 玩家断开；
- Session、Run、Round或Segment结束；
- `map_transition`；
- `mission_lost`；
- `finale_win`；
- 对抗半场结束；
- 玩家切换阵营；
- 玩家进入或离开闲置；
- 服务器进入休眠；
- 管理员手动刷新。

已有保存事务执行时不得并行创建第二个相同事务；必须标记补充刷新，并在当前事务完成后再次检查dirty数据。
