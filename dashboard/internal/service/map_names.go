package service

import (
	"strings"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
)

// MapNameResolver resolves raw map codes without coupling callers to the
// Dashboard database. Custom names take precedence over the built-in official
// campaign map catalog.
type MapNameResolver struct {
	custom map[string]string
}

func NewMapNameResolver(custom []store.IngameMapName) MapNameResolver {
	values := make(map[string]string, len(custom))
	for _, value := range custom {
		key := strings.ToLower(strings.TrimSpace(value.MapName))
		label := strings.TrimSpace(value.DisplayName)
		if key != "" && label != "" {
			values[key] = label
		}
	}
	return MapNameResolver{custom: values}
}

func (r MapNameResolver) DisplayName(mapName string) string {
	raw := strings.TrimSpace(mapName)
	key := strings.ToLower(raw)
	if value := r.custom[key]; value != "" {
		return value
	}
	if value := officialIngameMapNames[key]; value != "" {
		return value
	}
	return raw
}

var officialIngameMapNames = map[string]string{
	"c1m1_hotel": "死亡中心 1/4", "c1m2_streets": "死亡中心 2/4", "c1m3_mall": "死亡中心 3/4", "c1m4_atrium": "死亡中心 4/4",
	"c2m1_highway": "黑色狂欢节 1/5", "c2m2_fairgrounds": "黑色狂欢节 2/5", "c2m3_coaster": "黑色狂欢节 3/5", "c2m4_barns": "黑色狂欢节 4/5", "c2m5_concert": "黑色狂欢节 5/5",
	"c3m1_plankcountry": "沼泽激战 1/4", "c3m2_swamp": "沼泽激战 2/4", "c3m3_shantytown": "沼泽激战 3/4", "c3m4_plantation": "沼泽激战 4/4",
	"c4m1_milltown_a": "暴风骤雨 1/5", "c4m2_sugarmill_a": "暴风骤雨 2/5", "c4m3_sugarmill_b": "暴风骤雨 3/5", "c4m4_milltown_b": "暴风骤雨 4/5", "c4m5_milltown_escape": "暴风骤雨 5/5",
	"c5m1_waterfront": "教区 1/5", "c5m2_park": "教区 2/5", "c5m3_cemetery": "教区 3/5", "c5m4_quarter": "教区 4/5", "c5m5_bridge": "教区 5/5",
	"c6m1_riverbank": "消逝 1/3", "c6m2_bedlam": "消逝 2/3", "c6m3_port": "消逝 3/3",
	"c7m1_docks": "牺牲 1/3", "c7m2_barge": "牺牲 2/3", "c7m3_port": "牺牲 3/3",
	"c8m1_apartments": "毫不留情 1/5", "c8m2_subway": "毫不留情 2/5", "c8m3_sewers": "毫不留情 3/5", "c8m4_interior": "毫不留情 4/5", "c8m5_rooftops": "毫不留情 5/5",
	"c9m1_alleys": "坠机险途 1/2", "c9m2_lots": "坠机险途 2/2",
	"c10m1_caves": "死亡丧钟 1/5", "c10m2_drainage": "死亡丧钟 2/5", "c10m3_ranchhouse": "死亡丧钟 3/5", "c10m4_mainstreet": "死亡丧钟 4/5", "c10m5_houseboat": "死亡丧钟 5/5",
	"c11m1_greenhouse": "静寂时分 1/5", "c11m2_offices": "静寂时分 2/5", "c11m3_garage": "静寂时分 3/5", "c11m4_terminal": "静寂时分 4/5", "c11m5_runway": "静寂时分 5/5",
	"c12m1_hilltop": "血腥收获 1/5", "c12m2_traintunnel": "血腥收获 2/5", "c12m3_bridge": "血腥收获 3/5", "c12m4_barn": "血腥收获 4/5", "c12m5_cornfield": "血腥收获 5/5",
	"c13m1_alpinecreek": "刺骨寒溪 1/4", "c13m2_southpinestream": "刺骨寒溪 2/4", "c13m3_memorialbridge": "刺骨寒溪 3/4", "c13m4_cutthroatcreek": "刺骨寒溪 4/4",
	"c14m1_junkyard": "临死一搏 1/2", "c14m2_lighthouse": "临死一搏 2/2",
}
