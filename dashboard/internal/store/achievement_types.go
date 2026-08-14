package store

import "context"

const AchievementContractVersion int64 = 1

type AchievementMetricValue struct {
	Value           int64
	Available       bool
	EvidenceSteamID string
	EvidenceRounds  int64
}

type PlayerAchievementMetrics struct {
	SteamID   string
	Watermark int64
	Values    map[string]AchievementMetricValue
}

type AchievementSourcePlayer struct {
	SteamID   string
	Watermark int64
}

type AchievementUnlock struct {
	SteamID                    string `json:"steam_id,omitempty"`
	AchievementKey             string `json:"achievement_key"`
	AchievementContractVersion int64  `json:"achievement_contract_version"`
	UnlockedAt                 int64  `json:"unlocked_at"`
	GrantKind                  string `json:"grant_kind"`
	ValueAtUnlock              int64  `json:"value_at_unlock"`
	EvidenceSteamID            string `json:"evidence_steam_id,omitempty"`
	SeenAt                     int64  `json:"seen_at,omitempty"`
}

type AchievementEvaluationState struct {
	SteamID                    string
	AchievementContractVersion int64
	SourceWatermark            int64
	EvaluatedAt                int64
}

type AchievementEngineState struct {
	AchievementContractVersion int64  `json:"achievement_contract_version"`
	GlobalSourceWatermark      int64  `json:"global_source_watermark"`
	DirtyCursorWatermark       int64  `json:"dirty_cursor_watermark"`
	DirtyCursorSteamID         string `json:"dirty_cursor_steam_id"`
	BackfillCursor             string `json:"backfill_cursor"`
	BackfillComplete           bool   `json:"backfill_complete"`
	LastRunAt                  int64  `json:"last_run_at"`
	LastSuccessAt              int64  `json:"last_success_at"`
	LastError                  string `json:"last_error,omitempty"`
	UpdatedAt                  int64  `json:"updated_at"`
}

type BadgeShowcaseSlot struct {
	Slot           int64  `json:"slot"`
	AchievementKey string `json:"achievement_key"`
	UpdatedAt      int64  `json:"updated_at,omitempty"`
}

type AchievementUnlockRate struct {
	AchievementKey string
	Unlocks        int64
}

type DashboardAchievementStore interface {
	AchievementEngineState(context.Context) (AchievementEngineState, error)
	UpdateAchievementEngineState(context.Context, AchievementEngineState) error
	AchievementEvaluationState(context.Context, string) (AchievementEvaluationState, error)
	UpsertAchievementEvaluationState(context.Context, AchievementEvaluationState) error
	ListAchievementUnlocks(context.Context, string) ([]AchievementUnlock, error)
	InsertAchievementUnlocks(context.Context, []AchievementUnlock) ([]AchievementUnlock, error)
	MarkAchievementUnlocksSeen(context.Context, string, int64) error
	AchievementUnlockRates(context.Context) ([]AchievementUnlockRate, error)
	AchievementEvaluatedPlayerCount(context.Context) (int64, error)
	BadgeShowcase(context.Context, string) ([]BadgeShowcaseSlot, error)
	ReplaceBadgeShowcase(context.Context, string, []BadgeShowcaseSlot, int64) error
}

type StatsAchievementStore interface {
	PlayerAchievementMetrics(context.Context, string) (PlayerAchievementMetrics, error)
	PlayerAchievementWatermark(context.Context, string) (int64, error)
	AchievementDirtyPlayers(context.Context, int64, string, int32) ([]AchievementSourcePlayer, error)
	AchievementBackfillPlayers(context.Context, string, int32) ([]AchievementSourcePlayer, error)
	AchievementEligiblePlayerCount(context.Context) (int64, error)
}
