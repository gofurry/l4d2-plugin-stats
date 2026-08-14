# Analysis Derived Contract v1

状态：自 Dashboard v1.3.0 起冻结。本文定义从生命周期、累计统计、Round Context 与 Incident 派生分析结果的口径；Aggregate Contract 保持 v1。

## 标准化与样本门槛

- `per_hour = metric / active_play_seconds * 3600`；分母为零时不可用，不显示为 0。
- Boss 参与率为 participations / encounters；每出生指标为 metric / spawns。
- 每小时排行至少 7200 秒；Boss 参与率至少 5 次遭遇；每出生至少 20 次出生；平均控制时长至少 10 次控制。
- 排行元数据区分 `higher_is_better` 与 `lower_is_better`。Context 不加入 Aggregate v1 维度。

## Round 与 Incident 分母

- 地图分析只使用至少有一个 Player Segment 的 Round。
- PvE 完成率分母为 `completed + failed`，排除 `active/abandoned`。
- Incident 派生率只使用 `capture_enabled=1, capture_complete=1, dropped_count=0` 的 Round。
- 历史无 Context/Incident 或不完整 Round 显示为不可用覆盖，不解释为零事件。

Timeline 使用 60 秒 offset 桶，每桶为 `事件数 * 100 / 持续到桶起点的完整 Incident Round 数`。Boss 时长只取同 Round 已匹配 Spawn/Death 的 offset 差；未匹配 Death 计入死亡数但不进入时长样本。

## 同步控制

CONTROL 使用半开区间 `[start, start + duration)`：相同时间先 end 后 start；同时数按不同幸存者 target 去重；每个 Episode 记录存续期间最大同时数；最大值至少 2/3/4 时分别计入 2-cap/3-cap/4-cap，一个 Episode 对每个阈值最多一次。

## 并肩作战

两名认证真人须共享 `round_id` 与 `side` 且 SteamID 不同，Segment 墙钟区间存在正重叠：`end=COALESCE(ended_at,last_saved_at)`，`overlap=max(0,min(endA,endB)-max(startA,startB))`。按 peer 汇总 shared seconds 与不同 shared rounds，用于玩家关系页的并肩数据与“最常并肩”摘要。它不表示好友关系，也不再出现在首页玩家预览卡。

坐标只供未来经过校准的地图 artwork 使用；v1.3 公共 UI 不展示伪热力图。
