# Achievement Contract v1

状态：自 Dashboard v1.3.2 起冻结；v1.3.3 按本契约兼容扩充 Catalog。

## 1. 定位

Achievement 是 Dashboard 根据 Stats DB 永久事实或 retention-safe 派生数据自动确认的永久荣誉。

- 不由 Collector 直接发放；
- 不需要玩家手动领取；
- 不存在玩家或管理员“手动刷新”；
- 已解锁后正常运行不自动撤销；
- Badge 是 Achievement 的展示资产，不拥有独立进度。

## 2. 条件模型

Contract v1 只支持单调指标：

```text
metric_value >= threshold
```

不支持：

- `<`
- `==`
- ratio / percentage
- AND / OR
- 时间窗口
- 任意 SQL
- 自定义表达式
- 会下降的效率指标

`metric_id` 必须由 Go 代码白名单 resolver 实现。

Catalog 可以在不修改已发布 key、metric、threshold、可见性与 Backfill 语义的前提下做加法扩充；这类扩充不升级 Contract 版本。

## 3. 可见性

`visibility`：

- `public`：未解锁时正常显示名称、条件、进度；
- `mystery`：未解锁显示 `??? / 隐藏成就`，不显示条件和进度；
- `secret`：未解锁完全不出现在公共 UI/API。

`counts_toward_completion`：

- 正向 public / mystery：`true`
- 搞笑/负面 secret：`false`

Secret 不计正常完成度，也不泄露总数量。

## 4. Tier

成长系列每个 Tier 都是独立不可变 `achievement_key`，UI 通过 `group_key` 合并。

发布后：

- key 不得复用；
- metric 不得改变；
- threshold 不得静默改变；
- 标题、文案、artwork 可以做不改变业务语义的修正。

同一系列共用一个基础 artwork；Tier 通过 CSS 外框表现。

## 5. 解锁

满足条件时幂等插入：

```text
(steam_id, achievement_key)
```

`unlocked_at`：

> Dashboard 首次确认满足条件的时间。

`grant_kind`：

- `live`
- `backfill`

## 6. 历史与 NULL

新采集字段的历史 `NULL`：

> 未采集

不得解释为真实 0。

只要存在系统已经确认的新版本累计值，达到阈值即可解锁；但 UI 不得声称该数值覆盖升级前完整历史。

## 7. Evidence

仅需要关联人的成就保存 `evidence_steam_id`。

`生死之交` tie-break：

1. shared_seconds DESC
2. shared_rounds DESC
3. SteamID64 ASC

解锁后 evidence 不自动漂移。

## 8. Badge Showcase

最多 3 个展示位：

- slot 1 为主 Badge，并作为仅支持单徽章的旧客户端兼容值；
- 仅本人 Steam 登录后可修改；
- 只能选择已解锁 Achievement；
- 未手动设置时，fallback 最近解锁的 3 个 `counts_toward_completion=true`；
- Secret 不自动 fallback，但已解锁后可手动装备。
- 玩家卡片按 slot 升序展示全部已配置或 fallback 的 Badge，最多 3 个；不得只展示 slot 1；
- 成就 Tab 不公开时，公开玩家卡片不得返回或展示 Badge。

玩家卡片 API 使用 `badges` 返回全部展示位；兼容字段 `main_badge` 在存在展示项时镜像 `badges[0]`。

## 9. Evaluation

自动路径：

1. 后台增量 evaluator；
2. 玩家资料访问时按需补判；
3. 首次历史 Backfill。

不提供任何手动刷新。

一次玩家评估应先构造 typed metrics，再在 Go 内存里判断全部未解锁 Achievement。

Catalog 做兼容加法扩充时，evaluator 必须使用内部 Catalog revision 触发一次可恢复的全量 Backfill，不得因 Contract 版本与原数据水位未变而跳过新 key。Catalog revision 不对外改变 Achievement Contract 版本。

## 10. Retention Safety

所有 Achievement resolver 必须读取永久事实或 retention-safe lifetime / aggregate。

原始明细 retention 不得使 progress 倒退。

v1.3.3 武器专精只使用 Dashboard 终身 `pve_equipment` 聚合，并以下式计算：

```text
family_kills = common_kills + special_kills + tank_kills + witch_kills
```

武器家族映射冻结为：

- `single_shotgun`：PumpShotgun、ChromeShotgun；
- `chainsaw`：Chainsaw；
- `machine_gun`：M60、MountedGun、Minigun；
- `smg`：SMG、SilencedSMG、MP5；
- `bolt_sniper`：Scout、AWP；
- `heavy_primary`：AutoShotgun、SPAS、HuntingRifle、MilitarySniper、M16、AK47、SCAR、SG552；
- `grenade_launcher`：GrenadeLauncher；
- `melee`：BaseballBat、CricketBat、Crowbar、ElectricGuitar、FireAxe、FryingPan、GolfClub、Katana、Knife、Machete、Pitchfork、Shovel、Tonfa。

Equipment ID 不得跨家族重复；`OtherFirearm`、投掷物不进入武器击杀专精，Chainsaw 不进入 melee。

## 11. 不撤销

普通 evaluator 只补发，不删除。

严重 bug 错发必须通过明确 repair / migration 处理。

## 12. v1.3.3 Catalog 扩充

Catalog 从 63 个底层 Achievement 扩充为 105 个：100 个计入完成度，5 个 Secret 不计入完成度，共享 38 个 artwork key。新增 Category `weapon`。

| group_key | metric_id | Tier threshold | Category |
| --- | --- | --- | --- |
| `weapon.throwable_expert` | `survivor.throwables_used` | 50 / 250 / 1000 / 2500 | weapon |
| `career.objective_master` | `survivor.objective_interactions` | 25 / 100 / 500 | career |
| `career.temp_health_addict` | `survivor.temp_health_items_used` | 100 / 500 / 2000 / 5000 | career |
| `support.firepower_upgrade` | `survivor.upgrade_packs_deployed` | 25 / 100 / 500 | support |
| `weapon.single_shotgun` | `weapon.single_shotgun_kills` | 1000 / 5000 / 20000 / 50000 | weapon |
| `weapon.chainsaw` | `weapon.chainsaw_kills` | 250 / 1000 / 5000 | weapon |
| `weapon.machine_gun` | `weapon.machine_gun_kills` | 500 / 2500 / 10000 | weapon |
| `weapon.smg` | `weapon.smg_kills` | 1000 / 5000 / 20000 / 50000 | weapon |
| `weapon.bolt_sniper` | `weapon.bolt_sniper_kills` | 250 / 1000 / 5000 | weapon |
| `weapon.heavy_primary` | `weapon.heavy_primary_kills` | 1000 / 5000 / 20000 / 50000 | weapon |
| `weapon.grenade_launcher` | `weapon.grenade_launcher_kills` | 500 / 2500 / 10000 | weapon |
| `weapon.melee` | `weapon.melee_kills` | 1000 / 5000 / 20000 / 50000 | weapon |

以上 42 个 Tier 全部为 `public` 且 `counts_toward_completion=true`。投掷、机关、药物与弹药包指标合并 PvE 与 Versus Survivor 生涯事实；投掷物的 PvE 部分使用终身装备 actions。
