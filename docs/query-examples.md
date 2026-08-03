# 读取查询示例

这些查询面向未来 Go 只读服务，也可用于管理员离线排查。示例使用命名参数
`:server_key`、`:steam_id` 和 `:limit` 表达输入；实际 Go 驱动应通过参数绑定并按数据库
驱动重写占位符，禁止拼接外部字符串。

## 1. 玩家对抗幸存者总览

```sql
SELECT
  s.steam_id,
  COALESCE(SUM(v.common_kills), 0) AS common_kills,
  COALESCE(SUM(v.human_special_kills), 0) AS human_special_kills,
  COALESCE(SUM(v.bot_special_kills), 0) AS bot_special_kills,
  COALESCE(SUM(v.damage_to_human_special), 0) AS damage_to_human_special,
  COALESCE(SUM(v.damage_to_bot_special), 0) AS damage_to_bot_special,
  COALESCE(SUM(v.incap_revives), 0) AS incap_revives,
  COALESCE(SUM(v.medkit_healing_others), 0) AS medkit_healing_others
FROM lps_player_segments s
JOIN lps_runs r ON r.run_id = s.run_id
JOIN lps_versus_survivor_stats v ON v.segment_id = s.segment_id
WHERE s.server_key = :server_key
  AND s.steam_id = :steam_id
  AND s.side = 'survivor'
  AND r.mode_family = 'versus'
  AND r.game_mode = 'versus'
  AND v.stats_version = 1
GROUP BY s.steam_id;
```

这里每个 Segment 只有一条当前绝对快照，因此跨 Segment 求和不会重复累计同一主键。

## 2. 玩家感染者职业总览

```sql
SELECT
  c.infected_class,
  SUM(c.spawn_count) AS spawn_count,
  SUM(c.damage_to_human_survivors) AS damage_to_human_survivors,
  SUM(c.damage_to_bot_survivors) AS damage_to_bot_survivors,
  SUM(c.human_survivor_controls) AS human_survivor_controls,
  SUM(c.bot_survivor_controls) AS bot_survivor_controls,
  SUM(c.human_survivor_control_seconds) AS human_survivor_control_seconds,
  SUM(c.bot_survivor_control_seconds) AS bot_survivor_control_seconds
FROM lps_player_segments s
JOIN lps_runs r ON r.run_id = s.run_id
JOIN lps_versus_infected_class_stats c ON c.segment_id = s.segment_id
WHERE s.server_key = :server_key
  AND s.steam_id = :steam_id
  AND s.side = 'infected'
  AND r.mode_family = 'versus'
  AND r.game_mode = 'versus'
  AND c.stats_version = 1
GROUP BY c.infected_class
ORDER BY c.infected_class;
```

职业名称必须通过固定 ID 1、2、3、4、5、6、8 映射，不能直接使用动态名称。

## 3. 最近完成的对抗比赛

```sql
SELECT
  r.run_id,
  r.campaign_key,
  r.started_at,
  r.ended_at,
  v.team_0_campaign_score,
  v.team_1_campaign_score,
  v.winner_team_slot
FROM lps_runs r
JOIN lps_versus_run_results v ON v.run_id = r.run_id
WHERE r.server_key = :server_key
  AND r.mode_family = 'versus'
  AND r.game_mode = 'versus'
  AND r.status = 'completed'
  AND v.result_status = 'completed'
  AND v.score_available = 1
  AND v.stats_version = 1
ORDER BY r.ended_at DESC
LIMIT :limit;
```

`winner_team_slot=2` 表示平局。`raw_winner_team` 不应作为网页胜方字段。

## 4. 一场比赛的半场比分

```sql
SELECT
  r.map_seq,
  r.round_seq,
  r.map_name,
  r.attempt_no,
  r.half_no,
  v.scoring_team_slot,
  v.team_0_map_score,
  v.team_1_map_score,
  v.team_0_campaign_score,
  v.team_1_campaign_score,
  v.result_status
FROM lps_rounds r
JOIN lps_versus_round_results v ON v.round_id = r.round_id
WHERE r.run_id = :run_id
  AND r.mode_family = 'versus'
  AND v.stats_version = 1
ORDER BY r.round_seq;
```

半场重开产生新的 Round；`abandoned` 尝试可以用于诊断，但正常比分页面应与
`completed` 结果分开展示。

## 5. 契约健康检查

跨数据库可移植的完整不变量查询位于：

```text
database/queries/versus_contract_checks.sql
```

每条查询都返回 `check_name` 和 `violation_count`。健康数据库的所有
`violation_count` 必须为 0。该文件不执行修复或写入，可由 Go 健康检查、运维任务和
集成测试共同复用。
