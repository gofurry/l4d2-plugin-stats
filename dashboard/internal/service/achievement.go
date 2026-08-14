package service

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/gofurry/l4d2-plugin-stats/dashboard/internal/store"
	"go.uber.org/zap"
)

const (
	achievementBatchSize = int32(100)
	achievementInterval  = 10 * time.Minute
)

type achievementDashboard interface {
	store.DashboardAchievementStore
	store.DashboardAggregateStore
}

type AchievementCard struct {
	AchievementDefinition
	Unlocked         bool    `json:"unlocked"`
	CurrentValue     *int64  `json:"current_value,omitempty"`
	UnlockedAt       int64   `json:"unlocked_at,omitempty"`
	GrantKind        string  `json:"grant_kind,omitempty"`
	ValueAtUnlock    int64   `json:"value_at_unlock,omitempty"`
	EvidenceSteamID  string  `json:"evidence_steam_id,omitempty"`
	GlobalUnlockRate float64 `json:"global_unlock_rate"`
}

type AchievementBadge struct {
	Slot           int64  `json:"slot"`
	AchievementKey string `json:"achievement_key"`
	Title          string `json:"title"`
	ArtworkKey     string `json:"artwork_key"`
	Tier           int64  `json:"tier,omitempty"`
}

type AchievementOverview struct {
	Unlocked          int64                    `json:"unlocked"`
	Total             int64                    `json:"total"`
	CompletionPercent float64                  `json:"completion_percent"`
	EasterEggs        int64                    `json:"easter_eggs"`
	RecentUnlock      *store.AchievementUnlock `json:"recent_unlock,omitempty"`
	Badges            []AchievementBadge       `json:"badges"`
}

type PlayerAchievements struct {
	AchievementContractVersion int64               `json:"achievement_contract_version"`
	Overview                   AchievementOverview `json:"overview"`
	Items                      []AchievementCard   `json:"items"`
	UnseenLive                 []AchievementCard   `json:"unseen_live,omitempty"`
	UnseenBackfillCount        int64               `json:"unseen_backfill_count,omitempty"`
}

type PlayerBadges struct {
	AchievementContractVersion int64              `json:"achievement_contract_version"`
	Items                      []AchievementBadge `json:"items"`
}

type AchievementService struct {
	dashboard achievementDashboard
	stats     store.StatsAchievementStore
	logger    *zap.Logger
	mu        sync.Mutex
	running   bool
}

func NewAchievementService(dashboard achievementDashboard, stats store.StatsAchievementStore, logger *zap.Logger) *AchievementService {
	return &AchievementService{dashboard: dashboard, stats: stats, logger: logger}
}

func (s *AchievementService) Start(ctx context.Context) {
	go func() {
		s.runLogged(ctx)
		ticker := time.NewTicker(achievementInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runLogged(ctx)
			}
		}
	}()
}

func (s *AchievementService) runLogged(ctx context.Context) {
	runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err := s.RunOnce(runCtx); err != nil && ctx.Err() == nil && s.logger != nil {
		s.logger.Warn("achievement evaluation failed", zap.Error(err))
	}
}

func (s *AchievementService) RunOnce(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	state, err := s.dashboard.AchievementEngineState(ctx)
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	state.LastRunAt, state.UpdatedAt = now, now
	if !state.BackfillComplete {
		err = s.runBackfill(ctx, &state)
	} else {
		err = s.runIncremental(ctx, &state)
	}
	if err != nil {
		state.LastError = err.Error()
		state.UpdatedAt = time.Now().Unix()
		_ = s.dashboard.UpdateAchievementEngineState(context.Background(), state)
		return err
	}
	state.LastSuccessAt, state.LastError, state.UpdatedAt = time.Now().Unix(), "", time.Now().Unix()
	return s.dashboard.UpdateAchievementEngineState(ctx, state)
}

func (s *AchievementService) runBackfill(ctx context.Context, state *store.AchievementEngineState) error {
	players, err := s.stats.AchievementBackfillPlayers(ctx, state.BackfillCursor, achievementBatchSize)
	if err != nil {
		return err
	}
	for _, player := range players {
		if _, err := s.evaluatePlayer(ctx, player.SteamID, "backfill"); err != nil {
			return err
		}
		state.BackfillCursor = player.SteamID
	}
	if len(players) < int(achievementBatchSize) {
		state.BackfillComplete = true
		state.DirtyCursorWatermark = state.GlobalSourceWatermark
		state.DirtyCursorSteamID = "\uffff"
	}
	return nil
}

func (s *AchievementService) runIncremental(ctx context.Context, state *store.AchievementEngineState) error {
	if state.DirtyCursorWatermark < state.GlobalSourceWatermark || state.DirtyCursorSteamID == "" {
		state.DirtyCursorWatermark = state.GlobalSourceWatermark
		state.DirtyCursorSteamID = "\uffff"
	}
	players, err := s.stats.AchievementDirtyPlayers(ctx, state.DirtyCursorWatermark, state.DirtyCursorSteamID, achievementBatchSize)
	if err != nil {
		return err
	}
	for _, player := range players {
		if _, err := s.evaluatePlayer(ctx, player.SteamID, "live"); err != nil {
			return err
		}
		state.DirtyCursorWatermark, state.DirtyCursorSteamID = player.Watermark, player.SteamID
	}
	if len(players) < int(achievementBatchSize) {
		state.GlobalSourceWatermark = state.DirtyCursorWatermark
		state.DirtyCursorSteamID = "\uffff"
	}
	return nil
}

func (s *AchievementService) EnsurePlayer(ctx context.Context, steamID string) ([]store.AchievementUnlock, error) {
	state, err := s.dashboard.AchievementEngineState(ctx)
	if err != nil {
		return nil, err
	}
	grantKind := "live"
	if !state.BackfillComplete {
		grantKind = "backfill"
	}
	return s.evaluatePlayer(ctx, steamID, grantKind)
}

func (s *AchievementService) evaluatePlayer(ctx context.Context, steamID, grantKind string) ([]store.AchievementUnlock, error) {
	metrics, err := s.stats.PlayerAchievementMetrics(ctx, steamID)
	if err != nil {
		return nil, err
	}
	if err := s.addLifetimeHeadshots(ctx, &metrics); err != nil {
		return nil, err
	}
	evaluation, err := s.dashboard.AchievementEvaluationState(ctx, steamID)
	if err != nil {
		return nil, err
	}
	if evaluation.EvaluatedAt > 0 && evaluation.SourceWatermark >= metrics.Watermark {
		return nil, nil
	}
	existing, err := s.dashboard.ListAchievementUnlocks(ctx, steamID)
	if err != nil {
		return nil, err
	}
	unlocked := make(map[string]bool, len(existing))
	for _, item := range existing {
		unlocked[item.AchievementKey] = true
	}
	now := time.Now().Unix()
	candidates := make([]store.AchievementUnlock, 0)
	for _, definition := range achievementCatalog {
		if unlocked[definition.AchievementKey] {
			continue
		}
		metric, ok := metrics.Values[definition.MetricID]
		if !ok || !metric.Available || metric.Value < definition.Threshold {
			continue
		}
		candidates = append(candidates, store.AchievementUnlock{
			SteamID: steamID, AchievementKey: definition.AchievementKey,
			AchievementContractVersion: store.AchievementContractVersion,
			UnlockedAt:                 now, GrantKind: grantKind, ValueAtUnlock: metric.Value,
			EvidenceSteamID: metric.EvidenceSteamID,
		})
	}
	inserted, err := s.dashboard.InsertAchievementUnlocks(ctx, candidates)
	if err != nil {
		return nil, err
	}
	if err := s.dashboard.UpsertAchievementEvaluationState(ctx, store.AchievementEvaluationState{
		SteamID: steamID, AchievementContractVersion: store.AchievementContractVersion,
		SourceWatermark: metrics.Watermark, EvaluatedAt: now,
	}); err != nil {
		return nil, err
	}
	return inserted, nil
}

func (s *AchievementService) addLifetimeHeadshots(ctx context.Context, metrics *store.PlayerAchievementMetrics) error {
	status, err := s.dashboard.AggregateStatus(ctx)
	if err != nil {
		return err
	}
	if status.State != "ready" || status.SourceWatermark < metrics.Watermark {
		metrics.Watermark = status.SourceWatermark
		return nil
	}
	rows, err := s.dashboard.ListAggregateRows(ctx, store.AggregateFilter{
		Grain: store.AggregateGrainLifetime, Kinds: []string{"pve_equipment"}, SteamID: metrics.SteamID,
	})
	if err != nil {
		return err
	}
	var total int64
	for _, row := range rows {
		total += row.Metrics["headshot_kills"]
	}
	if len(rows) > 0 {
		metrics.Values["pve.firearm_headshot_kills"] = store.AchievementMetricValue{Value: total, Available: true}
	}
	return nil
}

func (s *AchievementService) Player(ctx context.Context, steamID string, self bool) (PlayerAchievements, error) {
	if _, err := s.EnsurePlayer(ctx, steamID); err != nil {
		return PlayerAchievements{}, err
	}
	unlocks, err := s.dashboard.ListAchievementUnlocks(ctx, steamID)
	if err != nil {
		return PlayerAchievements{}, err
	}
	metrics, err := s.stats.PlayerAchievementMetrics(ctx, steamID)
	if err != nil {
		return PlayerAchievements{}, err
	}
	if err := s.addLifetimeHeadshots(ctx, &metrics); err != nil {
		return PlayerAchievements{}, err
	}
	eligible, err := s.stats.AchievementEligiblePlayerCount(ctx)
	if err != nil {
		return PlayerAchievements{}, err
	}
	rates, err := s.dashboard.AchievementUnlockRates(ctx)
	if err != nil {
		return PlayerAchievements{}, err
	}
	rateByKey := make(map[string]int64, len(rates))
	for _, rate := range rates {
		rateByKey[rate.AchievementKey] = rate.Unlocks
	}
	unlockByKey := make(map[string]store.AchievementUnlock, len(unlocks))
	for _, unlock := range unlocks {
		unlockByKey[unlock.AchievementKey] = unlock
	}
	result := PlayerAchievements{AchievementContractVersion: store.AchievementContractVersion, Items: make([]AchievementCard, 0, len(achievementCatalog))}
	for index, definition := range achievementCatalog {
		unlock, isUnlocked := unlockByKey[definition.AchievementKey]
		if definition.Visibility == "secret" && !isUnlocked {
			continue
		}
		card := AchievementCard{AchievementDefinition: definition, Unlocked: isUnlocked}
		if eligible > 0 {
			card.GlobalUnlockRate = float64(rateByKey[definition.AchievementKey]) * 100 / float64(eligible)
		}
		if isUnlocked {
			card.UnlockedAt, card.GrantKind, card.ValueAtUnlock = unlock.UnlockedAt, unlock.GrantKind, unlock.ValueAtUnlock
			card.EvidenceSteamID = unlock.EvidenceSteamID
		} else if definition.Visibility == "mystery" {
			card.AchievementDefinition = AchievementDefinition{
				AchievementKey: fmt.Sprintf("mystery.%d", index+1), GroupKey: "mystery",
				Title: "???", Description: "条件尚未发现", Category: "special",
				Visibility: "mystery", CountsTowardCompletion: true,
			}
		} else if metric, ok := metrics.Values[definition.MetricID]; ok && metric.Available {
			value := metric.Value
			card.CurrentValue = &value
		}
		result.Items = append(result.Items, card)
	}
	badges, err := s.resolveBadges(ctx, steamID, unlocks)
	if err != nil {
		return PlayerAchievements{}, err
	}
	result.Overview.Badges = badges
	for _, definition := range achievementCatalog {
		if definition.CountsTowardCompletion {
			result.Overview.Total++
			if _, ok := unlockByKey[definition.AchievementKey]; ok {
				result.Overview.Unlocked++
			}
		} else if _, ok := unlockByKey[definition.AchievementKey]; ok {
			result.Overview.EasterEggs++
		}
	}
	if result.Overview.Total > 0 {
		result.Overview.CompletionPercent = float64(result.Overview.Unlocked) * 100 / float64(result.Overview.Total)
	}
	if len(unlocks) > 0 {
		recent := unlocks[0]
		result.Overview.RecentUnlock = &recent
	}
	if self {
		for _, card := range result.Items {
			unlock, ok := unlockByKey[card.AchievementKey]
			if !ok || unlock.SeenAt != 0 {
				continue
			}
			if unlock.GrantKind == "backfill" {
				result.UnseenBackfillCount++
			} else {
				result.UnseenLive = append(result.UnseenLive, card)
			}
		}
		if len(result.UnseenLive) > 0 || result.UnseenBackfillCount > 0 {
			if err := s.dashboard.MarkAchievementUnlocksSeen(ctx, steamID, time.Now().Unix()); err != nil {
				return PlayerAchievements{}, err
			}
		}
	}
	return result, nil
}

func (s *AchievementService) Badges(ctx context.Context, steamID string) (PlayerBadges, error) {
	if _, err := s.EnsurePlayer(ctx, steamID); err != nil {
		return PlayerBadges{}, err
	}
	unlocks, err := s.dashboard.ListAchievementUnlocks(ctx, steamID)
	if err != nil {
		return PlayerBadges{}, err
	}
	badges, err := s.resolveBadges(ctx, steamID, unlocks)
	return PlayerBadges{AchievementContractVersion: store.AchievementContractVersion, Items: badges}, err
}

func (s *AchievementService) SetBadges(ctx context.Context, steamID string, slots []store.BadgeShowcaseSlot) (PlayerBadges, error) {
	if len(slots) > 3 {
		return PlayerBadges{}, fmt.Errorf("at most three badges may be selected")
	}
	if err := s.dashboard.ReplaceBadgeShowcase(ctx, steamID, slots, time.Now().Unix()); err != nil {
		return PlayerBadges{}, err
	}
	return s.Badges(ctx, steamID)
}

func (s *AchievementService) resolveBadges(ctx context.Context, steamID string, unlocks []store.AchievementUnlock) ([]AchievementBadge, error) {
	explicit, err := s.dashboard.BadgeShowcase(ctx, steamID)
	if err != nil {
		return nil, err
	}
	keys := make([]store.BadgeShowcaseSlot, 0, 3)
	if len(explicit) > 0 {
		keys = explicit
	} else {
		for _, unlock := range unlocks {
			definition, ok := achievementByKey(unlock.AchievementKey)
			if !ok || !definition.CountsTowardCompletion {
				continue
			}
			keys = append(keys, store.BadgeShowcaseSlot{Slot: int64(len(keys) + 1), AchievementKey: unlock.AchievementKey})
			if len(keys) == 3 {
				break
			}
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].Slot < keys[j].Slot })
	result := make([]AchievementBadge, 0, len(keys))
	for _, slot := range keys {
		definition, ok := achievementByKey(slot.AchievementKey)
		if !ok {
			continue
		}
		result = append(result, AchievementBadge{Slot: slot.Slot, AchievementKey: definition.AchievementKey, Title: definition.Title, ArtworkKey: definition.ArtworkKey, Tier: definition.Tier})
	}
	return result, nil
}

func (s *AchievementService) EngineState(ctx context.Context) (store.AchievementEngineState, error) {
	return s.dashboard.AchievementEngineState(ctx)
}
