# L4D2 Player Stats

一个面向《求生之路 2》服务器的玩家身份、会话和玩法统计系统。

项目采用 monorepo。当前实现为 v0.6.5 统计采集器：在真人身份、Session、Run、Round 和 Segment 生命周期之上，已接入 `coop`、`realism` 的完整 PvE 统计，以及独立的 `versus` 幸存者/感染者统计、双方职业明细、控制/能力效果和引擎权威比分结果。插件稳定后再开发 Go 后端及其内嵌前端。

## 当前状态

当前已实现：

- 一个模块化 SourcePawn 插件，最终编译为 `l4d2_player_stats.smx`；
- SQLite、MySQL 和 PostgreSQL 三套等价迁移；
- 异步连接、自动迁移、服务器启动实例注册和异常启动恢复；
- 300 秒加随机抖动的服务器心跳；
- 5/15/30/60 秒有上限重连，以及重复错误日志限流；
- 仅限 root 管理员的状态、重连和立即刷新命令；
- 构建、部署、迁移校验和发布打包脚本。
- 仅在 `coop`、`realism`、`versus` 中保存通过 Steam 认证的真人；
- 保存 SteamID64、昵称、IP 和一次连续连接的 Session；
- 分开累计连接时间与实际操作时间；
- Session 按 SteamID 跨正常地图切换延续；换图重连窗口为 120 秒，真实断线或离开支持模式时关闭；
- 数据库故障期间使用有上限的内存 closed Session 队列。
- 建立 PvE 战役、章节尝试和 Versus 半场的 Run / Round 归属；
- 处理正常过图、团灭重试、手动换图、结局和模式切换；
- 按真人幸存者/感染者身份创建 Segment，观战和闲置会结束当前 Segment；
- Run、Round 和 Segment 使用绝对快照、单一异步事务和有界 closed 队列持久化。
- 按 PvE Segment 统计普通感染者、六种普通特感、Tank 和 Witch 的最后击杀；
- 统计对特感、Tank 和 Witch 造成的实际生命损失，不记录溢出伤害；
- 统计感染者造成的承伤，并拆分对真人、对 Bot 和真人承受的友伤；
- 统计倒地、死亡、倒地救援、挂边救援、电击器复活和被救援次数；
- 拆分医疗包自疗、治疗队友次数及实际恢复的真实生命；
- 统计止痛药、肾上腺素使用次数和实际获得的临时生命；
- 统计章节参与、完成时存活/死亡状态及战役完成；
- 拆分六种普通特感的击杀、伤害、受控次数/时长和队友解救；
- 以固定 ID 记录官方枪械与近战明细，未知/第三方枪械只进入 `Other Firearm`；
- 记录三种官方投掷物、Boss 参与、本人近战断舌、Tank 石头空爆和 Witch 技巧；
- 记录燃烧与高爆弹药升级包部署，不记录激光瞄准器；
- 记录成功目标互动、从弹药堆补充弹药、倒地/挂边累计时间和把黑白队友治疗回彩色；
- PvE 统计使用绝对快照与有界关闭队列，不为 Bot 建立个人统计。
- 对抗幸存者侧拆分真人/Bot 特感与 Tank 的击杀和有效伤害；
- 对抗幸存者侧按七种固定感染者职业保存真人/Bot 控制目标的击杀与有效伤害明细；
- 对抗幸存者侧记录普通感染者击杀、感染者承伤、真人/Bot 友伤、生存、救援、治疗和临时生命；
- 对抗幸存者侧记录 Witch、三种官方投掷物、两种弹药升级包、本人近战断舌、Tank 石头空爆和 Witch 技巧；
- 对抗感染者侧记录有效出生、对真人/Bot 幸存者的伤害、倒地和击杀；
- 对抗感染者侧按 Smoker、Boomer、Hunter、Spitter、Jockey、Charger、Tank 保存同口径职业明细；
- 对抗感染者侧记录四种控制次数/时长、Boomer 胆汁命中人数和 Spitter 酸液有效伤害，并拆分真人/Bot 目标；
- 对抗半场记录逻辑队伍地图分与累计分，对抗 Run 记录最终累计分、平局/胜负和异常结束状态；
- 对抗幸存者与感染者使用不同数据库表、绝对快照和有界关闭队列，不能进入 PvE 聚合。

v0.6.0 的核心归属、单人 Bot 上下半场和跨图 Run/Session 连续性已经通过本地验收；v0.6.2 职业明细和 v0.6.3 控制与能力效果也已通过本地验收。v0.6.4 幸存者战斗明细完成实现和离线验证，等待本地对抗数据验收；v0.6.5 已接入引擎半场/战役分数和最终结果，等待本地结果验收。完整的双方真人、换队、Tank 交接、重连和半场重开统一延期到 v0.7.0。当前不推算跨 Run 稳定队伍、逐时刻推进曲线或 MVP；对抗首版不建立逐武器统计表。

已经确认的基础边界：

- 数据库支持 SQLite、MySQL 和 PostgreSQL（SourceMod 驱动名为 `pgsql`）。
- 只采集 `coop`、`realism` 和 `versus`；其他模式完全不记录。
- `coop` 与 `realism` 属于 PvE 统计族，`versus` 使用独立的 PvP 统计模型。
- 只为通过 Steam 认证的真人玩家建立身份和统计，不为 Bot 建立玩家记录。
- 同时保存连接时间和有效参赛时间。
- 保存服务器观察到的玩家 IP 地址，不含端口；IP 不得由公开网页默认展示。
- 身份、会话和统计明细默认永久保留。
- 常规保存周期为 300 秒加可配置随机抖动，并在关键生命周期事件发生时保存。
- 第一阶段数据库故障只使用有界内存状态恢复，不创建无限增长的本地日志或队列文件。

## 契约

- [模式与生命周期](contracts/modes.md)
- [统计口径](contracts/statistics.md)
- [数据库结构](database/schema.md)

## Monorepo 边界

```text
collector/     SourceMod 数据采集插件
database/      三种数据库的结构和迁移
contracts/     插件与未来 Go 服务共同遵守的行为定义
dashboard/     未来的 Go 后端与内嵌前端
docs/          架构、部署和测试文档
scripts/       仓库级构建与发布脚本
```

详细说明见[架构文档](docs/architecture.md)。

后续版本按[开发路线图](docs/roadmap.md)推进。

## 本地构建

1. 将 `scripts/config.example.ps1` 复制为 `scripts/config.local.ps1`，填写本机 SourceMod 路径。
2. 运行 `scripts/build.ps1`；产物位于 `dist/l4d2_player_stats.smx`。
3. 运行 `scripts/deploy.ps1`，插件和三套迁移会复制到本机 SourceMod 环境。

VS Code 可以直接执行 `L4D2 Stats: Build` 或 `L4D2 Stats: Build and Deploy` 任务。

## 服务器配置与验证

数据库、ConVar 和 SQLite 首次验证步骤见[数据库地基部署](docs/database-foundation.md)。数据库密码只应存在于服务器自己的 `addons/sourcemod/configs/databases.cfg`，不得提交到仓库。

v0.3 的生命周期验收见 [Run / Round / Segment 测试清单](docs/v0.3-test-checklist.md)，v0.4 的战斗统计验收见 [PvE 核心统计测试清单](docs/v0.4-test-checklist.md)，v0.5 的治疗与章节统计验收见 [PvE 治疗与章节测试清单](docs/v0.5-test-checklist.md)，v0.5.1 的扩展统计验收见 [PvE 扩展统计测试清单](docs/v0.5.1-test-checklist.md)，v0.5.2 的互动与状态验收见 [PvE 互动与状态测试清单](docs/v0.5.2-test-checklist.md)，v0.6 的核心验收见 [Versus 核心统计测试清单](docs/v0.6-test-checklist.md)，v0.6.2 的职业明细验收见 [Versus 感染者职业测试清单](docs/v0.6.2-test-checklist.md)，v0.6.3 的能力效果验收见 [Versus 控制与能力测试清单](docs/v0.6.3-test-checklist.md)，v0.6.4 的幸存者明细验收见 [Versus 幸存者战斗明细测试清单](docs/v0.6.4-test-checklist.md)，v0.6.5 的比分与结果验收见 [Versus 比分与结果测试清单](docs/v0.6.5-test-checklist.md)。

管理员命令：

| 命令 | 权限 | 用途 |
|---|---|---|
| `sm_lps_status` | root | 查看连接、驱动、迁移和最近错误 |
| `sm_lps_reconnect` | root | 立即重新连接并重新检查迁移 |
| `sm_lps_flush` | root | 立即执行服务器心跳/刷新请求 |

## License

[MIT](LICENSE)
