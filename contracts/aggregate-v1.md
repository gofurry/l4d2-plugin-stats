# Aggregate Contract v1

Aggregate Contract v1 freezes the Dashboard aggregate representation used by L4D2 Player Stats 1.2.0. The only supported value of `aggregate_version` is `1`. A reader, writer, rollup, or retention operation must fail closed when it encounters another value; v1.2.0 does not upgrade aggregate contracts automatically.

## Common representation

Every row has `kind`, `server_key`, `steam_id`, `mode`, `dimension`, `metrics_json`, and `aggregate_version`. Daily rows use `day = floor(started_at / 86400)`. Monthly rows use the UTC first day of the month, expressed with the same day-number unit. Lifetime rows have no stored period and are exposed with `day = 0`.

All eleven kinds support the `daily`, `monthly`, and `lifetime` grains. Monthly and lifetime values are exact sums of their daily source rows. Identifiers are opaque strings. Metric values are signed 64-bit integers in storage but valid v1 producers emit non-negative counts, seconds, scores, and damage totals. Empty dimensions use `""`.

## Kinds

### `activity`

- Dimensions: `server_key`, `steam_id`; `mode` and `dimension` are empty.
- Source: player sessions grouped by session start day.
- Metrics: `session_count` counts sessions; `connected_seconds` is connected duration; `active_play_seconds` is actual play duration.

### `mode_activity`

- Dimensions: `server_key`, `steam_id`, `mode`, `dimension`.
- `mode`: `coop`, `realism`, or `versus`. `dimension`: `survivor` or `infected`; PvE only emits `survivor`.
- Source: eligible player segments grouped by segment start day.
- Metrics: `chapter_count` counts segments; `active_play_seconds` is segment active duration.

### `run_result`

- Dimensions: `server_key`, `mode`; `steam_id` is empty. `dimension` is the mode family (`pve` or `versus`).
- Metrics: `run_count` counts eligible runs; `completed_runs` counts runs whose status is `completed`.

### `versus_result`

- Dimensions: `server_key`; `mode` is `versus`, `steam_id` is empty, and `dimension` is `round` or `run`.
- Metrics: `completed_results` counts version-1 result rows; `score_available` sums the result rows whose score was available.

### `pve_combat`

- Dimensions: `server_key`, `steam_id`, `mode`; `dimension` is empty. `mode` is `coop` or `realism`.
- Metrics: `common_kills`, `special_kills`, `tank_kills`, `witch_kills`; `damage_to_special`, `damage_to_tank`, `damage_to_witch`, `damage_taken_infected`; `friendly_fire_to_humans`, `friendly_fire_to_bots`, `friendly_fire_taken`; `incapacitations`, `deaths`; `incap_revives`, `ledge_rescues`, `defib_revives`, `rescues_received`; `medkits_used_self`, `medkits_used_on_others`, `medkit_healing_self`, `medkit_healing_others`; `pills_used`, `adrenaline_used`, `temporary_health_received`; `chapter_participations`, `chapter_completions_alive`, `chapter_completions_dead`, `campaign_completions`.
- Kill, use, participation, rescue, incap, and death metrics are event counts. Damage and healing metrics are effective-health points. Duration is not included in this kind.

### `pve_detail`

- Dimensions are identical to `pve_combat`.
- Per-class metrics: `{smoker|boomer|hunter|spitter|jockey|charger}_kills` and `damage_to_{class}`; `{smoker|hunter|jockey|charger}_controls_received`, `{class}_controlled_seconds`, and `{class}_saves` for controlling classes.
- Interaction metrics: `melee_tongue_self_cuts`, `tank_rocks_destroyed`, `witch_oneshots`, `witch_solo_kills`, `tank_encounters`, `tank_kill_participations`, `witch_encounters`, `witch_kill_participations`, `incendiary_packs_deployed`, `explosive_packs_deployed`, `objective_interactions`, `ammo_pile_uses`, `incapacitated_seconds`, `ledge_hanging_seconds`, `black_white_teammates_restored`.
- Counts are event or encounter counts; damage is effective-health points; fields ending in `_seconds` are whole seconds.

### `pve_equipment`

- Dimensions: the PvE dimensions above plus `dimension = equipment_id` as its base-10 integer string.
- Metrics: `actions`, `common_kills`, `special_kills`, `tank_kills`, `witch_kills`, `headshot_kills`, `damage_to_special`, `damage_to_tank`, `damage_to_witch`.
- `actions` is use/attack activity attributed to the equipment; other metrics retain the kill-count and effective-damage meanings above.

### `versus_survivor`

- Dimensions: `server_key`, `steam_id`, `mode = versus`; `dimension` is empty.
- Combat metrics: `common_kills`, `human_special_kills`, `bot_special_kills`, `human_tank_kills`, `bot_tank_kills`, `damage_to_human_special`, `damage_to_bot_special`, `damage_to_human_tank`, `damage_to_bot_tank`, `damage_taken_infected`, `friendly_fire_to_humans`, `friendly_fire_to_bots`, `friendly_fire_taken`, `incapacitations`, `deaths`.
- Team/support metrics: `incap_revives`, `ledge_rescues`, `defib_revives`, `rescues_received`, `medkits_used_self`, `medkits_used_on_others`, `medkit_healing_self`, `medkit_healing_others`, `pills_used`, `adrenaline_used`, `temporary_health_received`.
- Other metrics: `witch_kills`, `damage_to_witch`, `molotovs_thrown`, `pipe_bombs_thrown`, `vomit_jars_thrown`, `incendiary_packs_deployed`, `explosive_packs_deployed`, `melee_tongue_self_cuts`, `tank_rocks_destroyed`, `witch_oneshots`, `witch_solo_kills`.
- Human/Bot qualifiers describe the controlled target or attacker represented by the field. Damage and healing are effective-health points; other values are counts.

### `versus_survivor_class`

- Dimensions: Versus survivor dimensions plus `dimension = infected_class` as its base-10 class ID.
- Metrics: `human_controller_kills`, `bot_controller_kills`, `damage_to_human_controllers`, `damage_to_bot_controllers`.
- Controller metrics classify the killed or damaged special infected by whether a human or Bot controlled it.

### `versus_infected`

- Dimensions: `server_key`, `steam_id`, `mode = versus`; `dimension` is empty.
- Metrics: `spawn_count`, `damage_to_human_survivors`, `damage_to_bot_survivors`, `human_survivor_incaps`, `bot_survivor_incaps`, `human_survivor_kills`, `bot_survivor_kills`.
- Human/Bot qualifiers describe the survivor target. Damage is effective-health points; all other values are counts.

### `versus_infected_class`

- Dimensions: Versus infected dimensions plus `dimension = infected_class` as its base-10 class ID.
- Metrics: all `versus_infected` metrics plus `human_survivor_controls`, `bot_survivor_controls`, `human_survivor_control_seconds`, `bot_survivor_control_seconds`, `human_survivor_ability_hits`, `bot_survivor_ability_hits`, `human_survivor_ability_damage`, `bot_survivor_ability_damage`.
- Controls and hits are counts, control durations are whole seconds, and damage fields are effective-health points.

## Compatibility rules

- Producers write `aggregate_version = 1` to daily, monthly, lifetime, state, and retention audit rows.
- Consumers validate the version before decoding or summing metrics.
- Cleanup preview IDs include the aggregate version. Cleanup requires a ready state, matching version, and a source watermark that covers the preview watermark; all three are checked again immediately before deletion.
- Adding or removing a kind, dimension, eligible source mode, metric, unit, or semantic meaning requires a new aggregate contract version.
- Stats schema 2's `car_alarms_triggered` field is deliberately not an Aggregate Contract v1 metric. Player totals and full-server rankings read it from raw Stats rows; retention must therefore treat availability of that ranking like any other raw-detail feature.
- Stats schema 3's Versus-survivor `objective_interactions` field is also outside Aggregate Contract v1 and is read from raw Stats rows for player detail only.
