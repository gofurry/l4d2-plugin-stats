package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

const rankingCacheCapacity = 128

type AggregateService struct {
	dashboard  store.DashboardAggregateStore
	stats      store.StatsAggregateStore
	logger     *zap.Logger
	mu         sync.Mutex
	running    bool
	reschedule chan struct{}
}

func NewAggregateService(dashboard store.DashboardAggregateStore, stats store.StatsAggregateStore, logger *zap.Logger) *AggregateService {
	return &AggregateService{dashboard: dashboard, stats: stats, logger: logger, reschedule: make(chan struct{}, 1)}
}

func (s *AggregateService) Rebuild(ctx context.Context) error {
	retentionRuns, err := s.dashboard.RetentionRunCount(ctx)
	if err != nil {
		return fmt.Errorf("read retention history: %w", err)
	}
	if retentionRuns > 0 {
		return fmt.Errorf("full aggregate rebuild is unavailable after raw data cleanup; use incremental aggregation")
	}
	return s.run(ctx, true)
}

func (s *AggregateService) Sync(ctx context.Context) error {
	return s.run(ctx, false)
}

func (s *AggregateService) run(ctx context.Context, full bool) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("aggregate rebuild is already running")
	}
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()
	started := time.Now()
	after := int64(0)
	if !full {
		status, err := s.dashboard.AggregateStatus(ctx)
		if err != nil {
			return fmt.Errorf("read aggregate status: %w", err)
		}
		after = status.SourceWatermark
	}
	change, err := s.stats.AggregateChanges(ctx, after)
	if err != nil {
		return fmt.Errorf("read aggregate source: %w", err)
	}
	if full {
		change.Full = true
	}
	if err := s.dashboard.ApplyAggregateChanges(ctx, change); err != nil {
		return fmt.Errorf("apply aggregate changes: %w", err)
	}
	if s.logger != nil {
		s.logger.Info("statistics aggregate updated", zap.Bool("full", change.Full), zap.Int("rows", len(change.Rows)), zap.Int("days", len(change.Days)), zap.Duration("duration", time.Since(started)))
	}
	return nil
}

func (s *AggregateService) Start(ctx context.Context) {
	go func() {
		s.syncLogged(ctx)
		for {
			interval := 30 * time.Minute
			if settings, err := s.dashboard.DataMaintenanceSettings(ctx); err == nil && settings.AggregateIntervalMinutes > 0 {
				interval = time.Duration(settings.AggregateIntervalMinutes) * time.Minute
			}
			timer := time.NewTimer(interval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-s.reschedule:
				if !timer.Stop() {
					<-timer.C
				}
			case <-timer.C:
				s.syncLogged(ctx)
			}
		}
	}()
}

func (s *AggregateService) Reschedule() {
	select {
	case s.reschedule <- struct{}{}:
	default:
	}
}

func (s *AggregateService) syncLogged(ctx context.Context) {
	rebuildCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	if err := s.Sync(rebuildCtx); err != nil && s.logger != nil && ctx.Err() == nil {
		s.logger.Warn("statistics aggregate update failed", zap.Error(err))
	}
}

func (s *AggregateService) Status(ctx context.Context) (store.AggregateStatus, error) {
	return s.dashboard.AggregateStatus(ctx)
}

type rankingCacheEntry struct {
	value   store.RankingPage
	expires time.Time
	usedAt  time.Time
}

type RankingService struct {
	dashboard store.DashboardAggregateStore
	stats     store.StatsStore
	mu        sync.Mutex
	cache     map[string]rankingCacheEntry
	group     singleflight.Group
}

func NewRankingService(dashboard store.DashboardAggregateStore, stats store.StatsStore) *RankingService {
	return &RankingService{dashboard: dashboard, stats: stats, cache: make(map[string]rankingCacheEntry)}
}

func (s *RankingService) List(ctx context.Context, query store.RankingQuery) (store.RankingPage, error) {
	selectedIDs := append([]string(nil), query.SteamIDs...)
	sort.Strings(selectedIDs)
	key := fmt.Sprintf("%s|%s|%s|%d|%d|%d|%d|%s|%s", query.Mode, query.Metric, query.ServerKey, query.Cutoff, query.MinimumActiveSec, query.Limit, query.Offset, strings.Join(selectedIDs, ","), query.SubjectSteamID)
	now := time.Now()
	s.mu.Lock()
	cached, ok := s.cache[key]
	if ok && now.Before(cached.expires) {
		cached.usedAt = now
		s.cache[key] = cached
	}
	s.mu.Unlock()
	if ok && now.Before(cached.expires) {
		return cached.value, nil
	}
	result, err, _ := s.group.Do(key, func() (any, error) {
		page, err := s.load(ctx, query)
		if err != nil {
			return store.RankingPage{}, err
		}
		s.storeCache(key, page, time.Now())
		return page, nil
	})
	if err != nil {
		return store.RankingPage{}, err
	}
	return result.(store.RankingPage), nil
}

func (s *RankingService) storeCache(key string, page store.RankingPage, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for cacheKey, entry := range s.cache {
		if now.After(entry.expires) {
			delete(s.cache, cacheKey)
		}
	}
	for len(s.cache) >= rankingCacheCapacity {
		oldestKey := ""
		oldestAt := now
		for cacheKey, entry := range s.cache {
			if oldestKey == "" || entry.usedAt.Before(oldestAt) {
				oldestKey, oldestAt = cacheKey, entry.usedAt
			}
		}
		delete(s.cache, oldestKey)
	}
	s.cache[key] = rankingCacheEntry{value: page, expires: now.Add(60 * time.Second), usedAt: now}
}

func (s *RankingService) Servers(ctx context.Context) ([]string, error) {
	rows, err := s.dashboard.ListAggregateRows(ctx, store.AggregateFilter{Grain: store.AggregateGrainLifetime, Kinds: []string{"activity"}})
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{})
	for _, row := range rows {
		if row.ServerKey != "" {
			set[row.ServerKey] = struct{}{}
		}
	}
	servers := make([]string, 0, len(set))
	for key := range set {
		servers = append(servers, key)
	}
	sort.Strings(servers)
	return servers, nil
}

func (s *RankingService) SearchPlayers(ctx context.Context, query string) ([]store.PlayerIdentity, error) {
	return s.stats.SearchPlayers(ctx, strings.TrimSpace(query), 20)
}

type rankingAccumulator struct {
	metrics map[string]int64
	active  int64
}

func (s *RankingService) load(ctx context.Context, query store.RankingQuery) (store.RankingPage, error) {
	definition, ok := rankingDefinitions[query.Mode+":"+query.Metric]
	if !ok {
		return store.RankingPage{}, fmt.Errorf("unsupported ranking metric")
	}
	if definition.rawIncident {
		incidentStore, ok := s.stats.(store.StatsIncidentRankingStore)
		if !ok {
			return store.RankingPage{}, fmt.Errorf("incident rankings are unavailable")
		}
		entries, err := incidentStore.CarAlarmRanking(ctx, query)
		if err != nil {
			return store.RankingPage{}, err
		}
		return s.finishRanking(ctx, query, entries), nil
	}
	cutoffDay := int64(0)
	if query.Cutoff > 0 {
		cutoffDay = query.Cutoff / 86400
	}
	grain := store.AggregateGrainDaily
	if cutoffDay == 0 {
		grain = store.AggregateGrainLifetime
	}
	rows, err := s.dashboard.ListAggregateRows(ctx, store.AggregateFilter{Grain: grain, Kinds: definition.kinds, ServerKey: query.ServerKey, CutoffDay: cutoffDay})
	if err != nil {
		return store.RankingPage{}, err
	}
	players := make(map[string]*rankingAccumulator)
	for _, row := range rows {
		if !definition.accept(row) {
			continue
		}
		entry := players[row.SteamID]
		if entry == nil {
			entry = &rankingAccumulator{metrics: make(map[string]int64)}
			players[row.SteamID] = entry
		}
		if row.Kind == "activity" || row.Kind == "mode_activity" {
			entry.active += row.Metrics["active_play_seconds"]
		}
		for metric, value := range row.Metrics {
			entry.metrics[metric] += value
		}
	}
	minimum := query.MinimumActiveSec
	if minimum == 0 {
		minimum = definition.defaultMinimum
	}
	entries := make([]store.RankingEntry, 0, len(players))
	for steamID, accumulator := range players {
		if accumulator.active < minimum {
			continue
		}
		value := definition.value(accumulator.metrics)
		if definition.perHour {
			if accumulator.active == 0 {
				continue
			}
			value = value * 3600 / float64(accumulator.active)
		}
		if value <= 0 {
			continue
		}
		entries = append(entries, store.RankingEntry{SteamID: steamID, Value: value, ActiveSeconds: accumulator.active})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Value == entries[j].Value {
			return entries[i].SteamID < entries[j].SteamID
		}
		return entries[i].Value > entries[j].Value
	})
	for i := range entries {
		entries[i].Rank = int64(i + 1)
	}
	return s.finishRanking(ctx, query, entries), nil
}

func (s *RankingService) finishRanking(ctx context.Context, query store.RankingQuery, entries []store.RankingEntry) store.RankingPage {
	var self *store.RankingEntry
	if query.SubjectSteamID != "" {
		for i := range entries {
			if entries[i].SteamID == query.SubjectSteamID {
				copy := entries[i]
				self = &copy
				break
			}
		}
	}
	visibleEntries := entries
	if len(query.SteamIDs) > 0 {
		selected := make(map[string]struct{}, len(query.SteamIDs))
		for _, steamID := range query.SteamIDs {
			selected[steamID] = struct{}{}
		}
		visibleEntries = make([]store.RankingEntry, 0, len(query.SteamIDs))
		for _, entry := range entries {
			if _, ok := selected[entry.SteamID]; ok {
				visibleEntries = append(visibleEntries, entry)
			}
		}
	}
	total := len(visibleEntries)
	start := min(query.Offset, total)
	end := min(start+query.Limit, total)
	pageEntries := append([]store.RankingEntry(nil), visibleEntries[start:end]...)
	for i := range pageEntries {
		player, err := s.stats.PlayerSummary(ctx, pageEntries[i].SteamID)
		if err == nil && player != nil {
			pageEntries[i].PlayerName = player.LastName
		}
	}
	if self != nil {
		player, err := s.stats.PlayerSummary(ctx, self.SteamID)
		if err == nil && player != nil {
			self.PlayerName = player.LastName
		}
	}
	return store.RankingPage{Metric: query.Metric, Mode: query.Mode, Items: pageEntries, Total: int64(total), Self: self, GeneratedAt: time.Now().UTC()}
}

type rankingDefinition struct {
	kinds          []string
	defaultMinimum int64
	perHour        bool
	rawIncident    bool
	accept         func(store.AggregateRow) bool
	value          func(map[string]int64) float64
}

func sumMetrics(names ...string) func(map[string]int64) float64 {
	return func(metrics map[string]int64) float64 {
		var total int64
		for _, name := range names {
			total += metrics[name]
		}
		return float64(total)
	}
}

func modeRows(mode string, side string) func(store.AggregateRow) bool {
	return func(row store.AggregateRow) bool {
		if row.Kind == "mode_activity" {
			if side != "" && row.Dimension != side {
				return false
			}
			if mode == "pve" {
				return row.Mode == "coop" || row.Mode == "realism"
			}
			return row.Mode == mode
		}
		if mode == "pve" {
			return row.Mode == "coop" || row.Mode == "realism"
		}
		return row.Mode == mode || row.Kind == "activity"
	}
}

func definition(kinds []string, minimum int64, perHour bool, accept func(store.AggregateRow) bool, metrics ...string) rankingDefinition {
	return rankingDefinition{kinds: kinds, defaultMinimum: minimum, perHour: perHour, accept: accept, value: sumMetrics(metrics...)}
}

func incidentDefinition() rankingDefinition {
	return rankingDefinition{rawIncident: true}
}

var rankingDefinitions = func() map[string]rankingDefinition {
	all := func(store.AggregateRow) bool { return true }
	pve := modeRows("pve", "survivor")
	vs := modeRows("versus", "survivor")
	vi := modeRows("versus", "infected")
	definitions := map[string]rankingDefinition{
		"activity:active_time":                    definition([]string{"activity"}, 0, false, all, "active_play_seconds"),
		"activity:sessions":                       definition([]string{"activity"}, 0, false, all, "session_count"),
		"pve:common_kills":                        definition([]string{"mode_activity", "pve_combat"}, 0, false, pve, "common_kills"),
		"pve:special_kills":                       definition([]string{"mode_activity", "pve_combat"}, 0, false, pve, "special_kills"),
		"pve:boss_kills":                          definition([]string{"mode_activity", "pve_combat"}, 0, false, pve, "tank_kills", "witch_kills"),
		"pve:special_damage":                      definition([]string{"mode_activity", "pve_combat"}, 0, false, pve, "damage_to_special", "damage_to_tank", "damage_to_witch"),
		"pve:rescues":                             definition([]string{"mode_activity", "pve_combat"}, 0, false, pve, "incap_revives", "ledge_rescues", "defib_revives"),
		"pve:healing":                             definition([]string{"mode_activity", "pve_combat"}, 0, false, pve, "medkit_healing_self", "medkit_healing_others"),
		"pve:campaign_completions":                definition([]string{"mode_activity", "pve_combat"}, 0, false, pve, "campaign_completions"),
		"pve:tongue_self_cuts":                    definition([]string{"mode_activity", "pve_detail"}, 0, false, pve, "melee_tongue_self_cuts"),
		"pve:rocks_destroyed":                     definition([]string{"mode_activity", "pve_detail"}, 0, false, pve, "tank_rocks_destroyed"),
		"pve:car_alarms_triggered":                incidentDefinition(),
		"pve:common_kills_per_hour":               definition([]string{"mode_activity", "pve_combat"}, 5*3600, true, pve, "common_kills"),
		"pve:special_kills_per_hour":              definition([]string{"mode_activity", "pve_combat"}, 5*3600, true, pve, "special_kills"),
		"versus_survivor:human_si_kills":          definition([]string{"mode_activity", "versus_survivor"}, 0, false, vs, "human_special_kills", "human_tank_kills"),
		"versus_survivor:damage":                  definition([]string{"mode_activity", "versus_survivor"}, 0, false, vs, "damage_to_human_special", "damage_to_human_tank"),
		"versus_survivor:rescues":                 definition([]string{"mode_activity", "versus_survivor"}, 0, false, vs, "incap_revives", "ledge_rescues", "defib_revives"),
		"versus_survivor:car_alarms_triggered":    incidentDefinition(),
		"versus_survivor:human_si_kills_per_hour": definition([]string{"mode_activity", "versus_survivor"}, 3*3600, true, vs, "human_special_kills", "human_tank_kills"),
		"versus_infected:damage":                  definition([]string{"mode_activity", "versus_infected"}, 0, false, vi, "damage_to_human_survivors"),
		"versus_infected:incaps":                  definition([]string{"mode_activity", "versus_infected"}, 0, false, vi, "human_survivor_incaps"),
		"versus_infected:kills":                   definition([]string{"mode_activity", "versus_infected"}, 0, false, vi, "human_survivor_kills"),
		"versus_infected:controls":                definition([]string{"mode_activity", "versus_infected_class"}, 0, false, vi, "human_survivor_controls"),
		"versus_infected:damage_per_hour":         definition([]string{"mode_activity", "versus_infected"}, 3*3600, true, vi, "damage_to_human_survivors"),
	}
	return definitions
}()

func ValidRanking(mode, metric string) bool {
	_, ok := rankingDefinitions[strings.TrimSpace(mode)+":"+strings.TrimSpace(metric)]
	return ok
}
