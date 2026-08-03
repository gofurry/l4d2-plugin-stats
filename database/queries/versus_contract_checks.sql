SELECT 'orphan_versus_round_results' AS check_name, COUNT(*) AS violation_count
FROM lps_versus_round_results v
LEFT JOIN lps_rounds r ON r.round_id = v.round_id
WHERE r.round_id IS NULL;
-- statement-breakpoint
SELECT 'orphan_versus_run_results' AS check_name, COUNT(*) AS violation_count
FROM lps_versus_run_results v
LEFT JOIN lps_runs r ON r.run_id = v.run_id
WHERE r.run_id IS NULL;
-- statement-breakpoint
SELECT 'versus_result_status_mismatches' AS check_name, COUNT(*) AS violation_count
FROM (
  SELECT v.round_id AS object_id
  FROM lps_versus_round_results v
  JOIN lps_rounds r ON r.round_id = v.round_id
  WHERE r.mode_family <> 'versus' OR r.status <> v.result_status
  UNION ALL
  SELECT v.run_id AS object_id
  FROM lps_versus_run_results v
  JOIN lps_runs r ON r.run_id = v.run_id
  WHERE r.mode_family <> 'versus' OR r.status <> v.result_status
) mismatches;
-- statement-breakpoint
SELECT 'versus_scoring_slot_mismatches' AS check_name, COUNT(*) AS violation_count
FROM lps_versus_round_results v
JOIN lps_rounds r ON r.round_id = v.round_id
WHERE r.mode_family = 'versus'
  AND r.half_no IN (1, 2)
  AND v.scoring_team_slot <> r.half_no - 1;
-- statement-breakpoint
SELECT 'versus_survivor_context_mismatches' AS check_name, COUNT(*) AS violation_count
FROM (
  SELECT v.segment_id AS object_id
  FROM lps_versus_survivor_stats v
  JOIN lps_player_segments s ON s.segment_id = v.segment_id
  JOIN lps_runs r ON r.run_id = s.run_id
  WHERE s.side <> 'survivor' OR r.mode_family <> 'versus' OR v.stats_version <> 1
  UNION ALL
  SELECT c.segment_id AS object_id
  FROM lps_versus_survivor_infected_class_stats c
  JOIN lps_player_segments s ON s.segment_id = c.segment_id
  JOIN lps_runs r ON r.run_id = s.run_id
  LEFT JOIN lps_versus_survivor_stats v ON v.segment_id = c.segment_id
  WHERE s.side <> 'survivor' OR r.mode_family <> 'versus'
    OR c.stats_version <> 1 OR c.infected_class NOT IN (1, 2, 3, 4, 5, 6, 8)
    OR v.segment_id IS NULL
) mismatches;
-- statement-breakpoint
SELECT 'versus_infected_context_mismatches' AS check_name, COUNT(*) AS violation_count
FROM (
  SELECT v.segment_id AS object_id
  FROM lps_versus_infected_stats v
  JOIN lps_player_segments s ON s.segment_id = v.segment_id
  JOIN lps_runs r ON r.run_id = s.run_id
  WHERE s.side <> 'infected' OR r.mode_family <> 'versus' OR v.stats_version <> 1
  UNION ALL
  SELECT c.segment_id AS object_id
  FROM lps_versus_infected_class_stats c
  JOIN lps_player_segments s ON s.segment_id = c.segment_id
  JOIN lps_runs r ON r.run_id = s.run_id
  LEFT JOIN lps_versus_infected_stats v ON v.segment_id = c.segment_id
  WHERE s.side <> 'infected' OR r.mode_family <> 'versus'
    OR c.stats_version <> 1 OR c.infected_class NOT IN (1, 2, 3, 4, 5, 6, 8)
    OR v.segment_id IS NULL
) mismatches;
-- statement-breakpoint
SELECT 'dual_versus_stats' AS check_name, COUNT(*) AS violation_count
FROM lps_versus_survivor_stats s
JOIN lps_versus_infected_stats i ON i.segment_id = s.segment_id;
-- statement-breakpoint
SELECT 'versus_survivor_class_total_mismatches' AS check_name, COUNT(*) AS violation_count
FROM lps_versus_survivor_stats v
WHERE
  v.human_special_kills <> COALESCE((SELECT SUM(c.human_controller_kills) FROM lps_versus_survivor_infected_class_stats c WHERE c.segment_id = v.segment_id AND c.infected_class BETWEEN 1 AND 6), 0)
  OR v.bot_special_kills <> COALESCE((SELECT SUM(c.bot_controller_kills) FROM lps_versus_survivor_infected_class_stats c WHERE c.segment_id = v.segment_id AND c.infected_class BETWEEN 1 AND 6), 0)
  OR v.damage_to_human_special <> COALESCE((SELECT SUM(c.damage_to_human_controllers) FROM lps_versus_survivor_infected_class_stats c WHERE c.segment_id = v.segment_id AND c.infected_class BETWEEN 1 AND 6), 0)
  OR v.damage_to_bot_special <> COALESCE((SELECT SUM(c.damage_to_bot_controllers) FROM lps_versus_survivor_infected_class_stats c WHERE c.segment_id = v.segment_id AND c.infected_class BETWEEN 1 AND 6), 0)
  OR v.human_tank_kills <> COALESCE((SELECT SUM(c.human_controller_kills) FROM lps_versus_survivor_infected_class_stats c WHERE c.segment_id = v.segment_id AND c.infected_class = 8), 0)
  OR v.bot_tank_kills <> COALESCE((SELECT SUM(c.bot_controller_kills) FROM lps_versus_survivor_infected_class_stats c WHERE c.segment_id = v.segment_id AND c.infected_class = 8), 0)
  OR v.damage_to_human_tank <> COALESCE((SELECT SUM(c.damage_to_human_controllers) FROM lps_versus_survivor_infected_class_stats c WHERE c.segment_id = v.segment_id AND c.infected_class = 8), 0)
  OR v.damage_to_bot_tank <> COALESCE((SELECT SUM(c.damage_to_bot_controllers) FROM lps_versus_survivor_infected_class_stats c WHERE c.segment_id = v.segment_id AND c.infected_class = 8), 0);
-- statement-breakpoint
SELECT 'versus_infected_class_total_mismatches' AS check_name, COUNT(*) AS violation_count
FROM lps_versus_infected_stats v
WHERE
  v.spawn_count <> COALESCE((SELECT SUM(c.spawn_count) FROM lps_versus_infected_class_stats c WHERE c.segment_id = v.segment_id), 0)
  OR v.damage_to_human_survivors <> COALESCE((SELECT SUM(c.damage_to_human_survivors) FROM lps_versus_infected_class_stats c WHERE c.segment_id = v.segment_id), 0)
  OR v.damage_to_bot_survivors <> COALESCE((SELECT SUM(c.damage_to_bot_survivors) FROM lps_versus_infected_class_stats c WHERE c.segment_id = v.segment_id), 0)
  OR v.human_survivor_incaps <> COALESCE((SELECT SUM(c.human_survivor_incaps) FROM lps_versus_infected_class_stats c WHERE c.segment_id = v.segment_id), 0)
  OR v.bot_survivor_incaps <> COALESCE((SELECT SUM(c.bot_survivor_incaps) FROM lps_versus_infected_class_stats c WHERE c.segment_id = v.segment_id), 0)
  OR v.human_survivor_kills <> COALESCE((SELECT SUM(c.human_survivor_kills) FROM lps_versus_infected_class_stats c WHERE c.segment_id = v.segment_id), 0)
  OR v.bot_survivor_kills <> COALESCE((SELECT SUM(c.bot_survivor_kills) FROM lps_versus_infected_class_stats c WHERE c.segment_id = v.segment_id), 0);
-- statement-breakpoint
SELECT 'versus_infected_ability_mismatches' AS check_name, COUNT(*) AS violation_count
FROM lps_versus_infected_class_stats
WHERE
  (infected_class NOT IN (1, 3, 5, 6) AND (human_survivor_controls > 0 OR bot_survivor_controls > 0 OR human_survivor_control_seconds > 0 OR bot_survivor_control_seconds > 0))
  OR (infected_class <> 2 AND (human_survivor_ability_hits > 0 OR bot_survivor_ability_hits > 0))
  OR (infected_class <> 4 AND (human_survivor_ability_damage > 0 OR bot_survivor_ability_damage > 0))
  OR (human_survivor_control_seconds > 0 AND human_survivor_controls = 0)
  OR (bot_survivor_control_seconds > 0 AND bot_survivor_controls = 0)
  OR human_survivor_ability_damage > damage_to_human_survivors
  OR bot_survivor_ability_damage > damage_to_bot_survivors;
