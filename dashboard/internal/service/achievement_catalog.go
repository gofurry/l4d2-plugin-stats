package service

import "fmt"

type AchievementDefinition struct {
	AchievementKey         string `json:"achievement_key"`
	GroupKey               string `json:"group_key"`
	Title                  string `json:"title"`
	Description            string `json:"description"`
	Category               string `json:"category"`
	MetricID               string `json:"metric_id,omitempty"`
	Threshold              int64  `json:"threshold,omitempty"`
	Tier                   int64  `json:"tier,omitempty"`
	Visibility             string `json:"visibility"`
	CountsTowardCompletion bool   `json:"counts_toward_completion"`
	ArtworkKey             string `json:"artwork_key,omitempty"`
}

var achievementCatalog = buildAchievementCatalog()

func AchievementCatalog() []AchievementDefinition {
	result := make([]AchievementDefinition, len(achievementCatalog))
	copy(result, achievementCatalog)
	return result
}

func buildAchievementCatalog() []AchievementDefinition {
	result := make([]AchievementDefinition, 0, 105)
	addTiered := func(group, title, description, category, metric, artwork string, thresholds ...int64) {
		for index, threshold := range thresholds {
			result = append(result, AchievementDefinition{
				AchievementKey: fmt.Sprintf("%s.%d", group, index+1), GroupKey: group,
				Title: title, Description: description, Category: category, MetricID: metric,
				Threshold: threshold, Tier: int64(index + 1), Visibility: "public",
				CountsTowardCompletion: true, ArtworkKey: artwork,
			})
		}
	}
	addSingle := func(key, title, description, category, metric, visibility string, threshold int64, counts bool) {
		result = append(result, AchievementDefinition{
			AchievementKey: key, GroupKey: key, Title: title, Description: description,
			Category: category, MetricID: metric, Threshold: threshold, Visibility: visibility,
			CountsTowardCompletion: counts, ArtworkKey: key,
		})
	}

	addTiered("career.veteran", "老兵", "累计实际游玩时长。", "career", "career.active_play_seconds", "career.veteran", 10*3600, 50*3600, 200*3600, 500*3600)
	addTiered("career.survivor", "生还者", "累计完成 PvE 战役。", "career", "pve.campaign_completions", "career.survivor", 10, 50, 200, 500)
	addTiered("combat.scavenger", "清道夫", "累计消灭普通感染者。", "combat", "pve.common_kills", "combat.scavenger", 10000, 50000, 200000, 500000)
	addTiered("combat.special_hunter", "特感猎手", "累计消灭特殊感染者。", "combat", "pve.special_kills", "combat.special_hunter", 1000, 5000, 20000, 50000)
	addTiered("combat.marksman", "神枪手", "累计使用受支持枪械爆头击杀。", "combat", "pve.firearm_headshot_kills", "combat.marksman", 5000, 25000, 100000, 250000)
	addTiered("combat.team_hunt", "协同猎杀", "累计参与特殊感染者助攻。", "combat", "pve.special_assists", "combat.team_hunt", 500, 2500, 10000, 25000)
	addTiered("support.steadfast", "坚毅不倒", "累计扶起普通倒地队友。", "support", "pve.incap_revives", "support.steadfast", 100, 500, 2000, 5000)
	addTiered("support.defuser", "拆火专家", "累计从 Smoker、Hunter、Jockey 或 Charger 控制中救下队友。", "support", "pve.special_rescues", "support.defuser", 100, 500, 2000)
	addTiered("support.field_medic", "战地医生", "累计为队友恢复医疗包生命值。", "support", "pve.medkit_healing_others", "support.field_medic", 10000, 50000, 200000)
	addTiered("boss.tank_hunter", "Tank 猎手", "累计击杀 Tank。", "boss", "pve.tank_kills", "boss.tank_hunter", 50, 200, 500)
	addTiered("boss.witch_hunter", "女巫猎人", "累计击杀 Witch。", "boss", "pve.witch_kills", "boss.witch_hunter", 50, 200, 500)
	addTiered("boss.boss_nemesis", "Boss 克星", "累计参与击杀但未取得最后一击的 Tank 与 Witch。", "boss", "pve.boss_assists", "boss.boss_nemesis", 100, 500, 2000)
	addTiered("versus.player_hunter", "玩家猎手", "累计击杀由真人控制的普通特殊感染者。", "versus", "versus.human_special_kills", "versus.player_hunter", 500, 2500, 10000)
	addTiered("versus.infected_master", "感染大师", "作为感染者累计对真人幸存者造成伤害。", "versus", "versus.infected_damage_to_human_survivors", "versus.infected_master", 50000, 250000, 1000000)
	addTiered("bond.comrade", "生死之交", "与同一位认证真人累计并肩作战。", "bond", "relationship.max_peer_shared_seconds", "bond.comrade", 10*3600, 50*3600, 100*3600)
	addTiered("weapon.throwable_expert", "投掷专家", "累计使用燃烧瓶、土制炸弹与胆汁罐。", "weapon", "survivor.throwables_used", "weapon.throwable_expert", 50, 250, 1000, 2500)
	addTiered("career.objective_master", "机关大师", "累计完成受支持的地图机关与目标互动。", "career", "survivor.objective_interactions", "career.objective_master", 25, 100, 500)
	addTiered("career.temp_health_addict", "药不能停", "累计使用止痛药与肾上腺素。", "career", "survivor.temp_health_items_used", "career.temp_health_addict", 100, 500, 2000, 5000)
	addTiered("support.firepower_upgrade", "火力升级", "累计部署燃烧弹药包与高爆弹药包。", "support", "survivor.upgrade_packs_deployed", "support.firepower_upgrade", 25, 100, 500)
	addTiered("weapon.single_shotgun", "单喷足以", "累计使用单发霰弹枪消灭感染者。", "weapon", "weapon.single_shotgun_kills", "weapon.single_shotgun", 1000, 5000, 20000, 50000)
	addTiered("weapon.chainsaw", "电锯狂魔", "累计使用电锯消灭感染者。", "weapon", "weapon.chainsaw_kills", "weapon.chainsaw", 250, 1000, 5000)
	addTiered("weapon.machine_gun", "枪林弹雨", "累计使用 M60 与固定机枪消灭感染者。", "weapon", "weapon.machine_gun_kills", "weapon.machine_gun", 500, 2500, 10000)
	addTiered("weapon.smg", "冲锋陷阵", "累计使用冲锋枪消灭感染者。", "weapon", "weapon.smg_kills", "weapon.smg", 1000, 5000, 20000, 50000)
	addTiered("weapon.bolt_sniper", "一发入魂", "累计使用 Scout 与 AWP 消灭感染者。", "weapon", "weapon.bolt_sniper_kills", "weapon.bolt_sniper", 250, 1000, 5000)
	addTiered("weapon.heavy_primary", "重火力", "累计使用重型主武器消灭感染者。", "weapon", "weapon.heavy_primary_kills", "weapon.heavy_primary", 1000, 5000, 20000, 50000)
	addTiered("weapon.grenade_launcher", "爆破专家", "累计使用榴弹发射器消灭感染者。", "weapon", "weapon.grenade_launcher_kills", "weapon.grenade_launcher", 500, 2500, 10000)
	addTiered("weapon.melee", "冷兵器大师", "累计使用近战武器消灭感染者。", "weapon", "weapon.melee_kills", "weapon.melee", 1000, 5000, 20000, 50000)

	addSingle("special.rock_breaker", "碎石机", "累计摧毁 Tank 投掷的石块。", "special", "tank_rocks_destroyed", "mystery", 5, true)
	addSingle("special.one_shot", "一击毙命", "累计一击击杀 Witch。", "special", "witch_oneshots", "mystery", 5, true)
	addSingle("special.witch_nemesis", "女巫克星", "累计单独击杀 Witch。", "special", "witch_solo_kills", "mystery", 5, true)
	addSingle("special.tongue_cutter", "砍舌达人", "累计用近战武器自行斩断 Smoker 舌头。", "special", "melee_tongue_self_cuts", "mystery", 5, true)
	addSingle("special.defib_rescuer", "起死回生", "累计使用电击器救活队友。", "special", "defib_revives", "public", 100, true)
	addSingle("special.miracle_healer", "妙手回春", "累计将黑白状态队友恢复为非黑白状态。", "special", "black_white_restores", "public", 100, true)

	addSingle("secret.crashed", "已坠机", "累计因坠落死亡。", "special", "survivor_fall_deaths", "secret", 5, false)
	addSingle("secret.see_u_again", "See u Again", "累计因坠落死亡。", "special", "survivor_fall_deaths", "secret", 100, false)
	addSingle("secret.dispatch", "出警", "累计触发警报车。", "special", "survivor_car_alarms", "secret", 100, false)
	addSingle("secret.ff_king", "黑枪王", "累计对真人队友造成友军伤害。", "special", "survivor_friendly_fire_to_humans", "secret", 10000, false)
	addSingle("secret.submissive", "已老实", "累计倒地。", "special", "survivor_incapacitations", "secret", 1000, false)
	return result
}

func achievementByKey(key string) (AchievementDefinition, bool) {
	for _, definition := range achievementCatalog {
		if definition.AchievementKey == key {
			return definition, true
		}
	}
	return AchievementDefinition{}, false
}
