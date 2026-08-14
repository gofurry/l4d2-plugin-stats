# Achievement Contract v1

状态：计划自 Dashboard v1.3.2 起冻结。

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

- slot 1 为主 Badge；
- 仅本人 Steam 登录后可修改；
- 只能选择已解锁 Achievement；
- 未手动设置时，fallback 最近解锁的 3 个 `counts_toward_completion=true`；
- Secret 不自动 fallback，但已解锁后可手动装备。

## 9. Evaluation

自动路径：

1. 后台增量 evaluator；
2. 玩家资料访问时按需补判；
3. 首次历史 Backfill。

不提供任何手动刷新。

一次玩家评估应先构造 typed metrics，再在 Go 内存里判断全部未解锁 Achievement。

## 10. Retention Safety

所有 Achievement resolver 必须读取永久事实或 retention-safe lifetime / aggregate。

原始明细 retention 不得使 progress 倒退。

## 11. 不撤销

普通 evaluator 只补发，不删除。

严重 bug 错发必须通过明确 repair / migration 处理。
