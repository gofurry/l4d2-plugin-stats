package store

import (
	"context"
	"database/sql"
	"fmt"
)

func (s *dashboardStore) AchievementEngineState(ctx context.Context) (AchievementEngineState, error) {
	var state AchievementEngineState
	var backfill int64
	err := s.db.QueryRowContext(ctx, `SELECT achievement_contract_version,global_source_watermark,
dirty_cursor_watermark,dirty_cursor_steam_id,backfill_cursor,backfill_complete,
last_run_at,last_success_at,last_error,updated_at
FROM achievement_engine_state WHERE singleton_id=1`).Scan(
		&state.AchievementContractVersion, &state.GlobalSourceWatermark,
		&state.DirtyCursorWatermark, &state.DirtyCursorSteamID, &state.BackfillCursor, &backfill,
		&state.LastRunAt, &state.LastSuccessAt, &state.LastError, &state.UpdatedAt,
	)
	if err != nil {
		return state, fmt.Errorf("read achievement engine state: %w", err)
	}
	if state.AchievementContractVersion != AchievementContractVersion {
		return state, fmt.Errorf("unsupported Achievement Contract version %d; expected %d", state.AchievementContractVersion, AchievementContractVersion)
	}
	state.BackfillComplete = backfill != 0
	return state, nil
}

func (s *dashboardStore) UpdateAchievementEngineState(ctx context.Context, state AchievementEngineState) error {
	if state.AchievementContractVersion != AchievementContractVersion {
		return fmt.Errorf("unsupported Achievement Contract version %d; expected %d", state.AchievementContractVersion, AchievementContractVersion)
	}
	_, err := s.db.ExecContext(ctx, `UPDATE achievement_engine_state SET
achievement_contract_version=?,global_source_watermark=?,dirty_cursor_watermark=?,
dirty_cursor_steam_id=?,backfill_cursor=?,backfill_complete=?,last_run_at=?,
last_success_at=?,last_error=?,updated_at=? WHERE singleton_id=1`,
		state.AchievementContractVersion, state.GlobalSourceWatermark, state.DirtyCursorWatermark,
		state.DirtyCursorSteamID, state.BackfillCursor, boolInt64(state.BackfillComplete), state.LastRunAt,
		state.LastSuccessAt, state.LastError, state.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update achievement engine state: %w", err)
	}
	return nil
}

func (s *dashboardStore) AchievementEvaluationState(ctx context.Context, steamID string) (AchievementEvaluationState, error) {
	var state AchievementEvaluationState
	err := s.db.QueryRowContext(ctx, `SELECT steam_id,achievement_contract_version,source_watermark,evaluated_at
FROM achievement_evaluation_state WHERE steam_id=?`, steamID).Scan(
		&state.SteamID, &state.AchievementContractVersion, &state.SourceWatermark, &state.EvaluatedAt,
	)
	if err == sql.ErrNoRows {
		return AchievementEvaluationState{SteamID: steamID, AchievementContractVersion: AchievementContractVersion}, nil
	}
	if err != nil {
		return state, fmt.Errorf("read achievement evaluation state: %w", err)
	}
	if state.AchievementContractVersion != AchievementContractVersion {
		return state, fmt.Errorf("unsupported Achievement Contract version %d; expected %d", state.AchievementContractVersion, AchievementContractVersion)
	}
	return state, nil
}

func (s *dashboardStore) UpsertAchievementEvaluationState(ctx context.Context, state AchievementEvaluationState) error {
	if state.AchievementContractVersion != AchievementContractVersion {
		return fmt.Errorf("unsupported Achievement Contract version %d; expected %d", state.AchievementContractVersion, AchievementContractVersion)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO achievement_evaluation_state
(steam_id,achievement_contract_version,source_watermark,evaluated_at) VALUES (?,?,?,?)
ON CONFLICT(steam_id) DO UPDATE SET achievement_contract_version=excluded.achievement_contract_version,
source_watermark=excluded.source_watermark,evaluated_at=excluded.evaluated_at`,
		state.SteamID, state.AchievementContractVersion, state.SourceWatermark, state.EvaluatedAt,
	)
	if err != nil {
		return fmt.Errorf("update achievement evaluation state: %w", err)
	}
	return nil
}

func (s *dashboardStore) ListAchievementUnlocks(ctx context.Context, steamID string) ([]AchievementUnlock, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT steam_id,achievement_key,achievement_contract_version,
unlocked_at,grant_kind,value_at_unlock,evidence_steam_id,seen_at
FROM achievement_unlocks WHERE steam_id=? ORDER BY unlocked_at DESC,achievement_key`, steamID)
	if err != nil {
		return nil, fmt.Errorf("list achievement unlocks: %w", err)
	}
	defer rows.Close()
	result := make([]AchievementUnlock, 0)
	for rows.Next() {
		var item AchievementUnlock
		if err := rows.Scan(&item.SteamID, &item.AchievementKey, &item.AchievementContractVersion,
			&item.UnlockedAt, &item.GrantKind, &item.ValueAtUnlock, &item.EvidenceSteamID, &item.SeenAt); err != nil {
			return nil, err
		}
		if item.AchievementContractVersion != AchievementContractVersion {
			return nil, fmt.Errorf("achievement %s uses unsupported contract version %d", item.AchievementKey, item.AchievementContractVersion)
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *dashboardStore) InsertAchievementUnlocks(ctx context.Context, unlocks []AchievementUnlock) ([]AchievementUnlock, error) {
	if len(unlocks) == 0 {
		return nil, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	inserted := make([]AchievementUnlock, 0, len(unlocks))
	for _, item := range unlocks {
		if item.AchievementContractVersion != AchievementContractVersion {
			return nil, fmt.Errorf("unsupported Achievement Contract version %d", item.AchievementContractVersion)
		}
		result, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO achievement_unlocks
(steam_id,achievement_key,achievement_contract_version,unlocked_at,grant_kind,value_at_unlock,evidence_steam_id,seen_at)
VALUES (?,?,?,?,?,?,?,?)`, item.SteamID, item.AchievementKey, item.AchievementContractVersion,
			item.UnlockedAt, item.GrantKind, item.ValueAtUnlock, item.EvidenceSteamID, item.SeenAt)
		if err != nil {
			return nil, fmt.Errorf("insert achievement unlock %s: %w", item.AchievementKey, err)
		}
		if affected, err := result.RowsAffected(); err == nil && affected == 1 {
			inserted = append(inserted, item)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return inserted, nil
}

func (s *dashboardStore) MarkAchievementUnlocksSeen(ctx context.Context, steamID string, seenAt int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE achievement_unlocks SET seen_at=?
WHERE steam_id=? AND seen_at=0`, seenAt, steamID)
	return err
}

func (s *dashboardStore) AchievementUnlockRates(ctx context.Context) ([]AchievementUnlockRate, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT achievement_key,COUNT(*) FROM achievement_unlocks GROUP BY achievement_key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]AchievementUnlockRate, 0)
	for rows.Next() {
		var item AchievementUnlockRate
		if err := rows.Scan(&item.AchievementKey, &item.Unlocks); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *dashboardStore) AchievementEvaluatedPlayerCount(ctx context.Context) (int64, error) {
	var count int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM achievement_evaluation_state
WHERE achievement_contract_version=?`, AchievementContractVersion).Scan(&count); err != nil {
		return 0, fmt.Errorf("count evaluated achievement players: %w", err)
	}
	return count, nil
}

func (s *dashboardStore) BadgeShowcase(ctx context.Context, steamID string) ([]BadgeShowcaseSlot, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT slot,achievement_key,updated_at
FROM player_badge_showcase WHERE steam_id=? ORDER BY slot`, steamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]BadgeShowcaseSlot, 0, 3)
	for rows.Next() {
		var slot BadgeShowcaseSlot
		if err := rows.Scan(&slot.Slot, &slot.AchievementKey, &slot.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, slot)
	}
	return result, rows.Err()
}

func (s *dashboardStore) BadgeShowcaseConfigured(ctx context.Context, steamID string) (bool, error) {
	var configured int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM player_badge_showcase_state WHERE steam_id=?`, steamID).Scan(&configured)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read badge showcase state: %w", err)
	}
	return true, nil
}

func (s *dashboardStore) ReplaceBadgeShowcase(ctx context.Context, steamID string, slots []BadgeShowcaseSlot, updatedAt int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	seenSlots := make(map[int64]bool, len(slots))
	seenKeys := make(map[string]bool, len(slots))
	for _, slot := range slots {
		if slot.Slot < 1 || slot.Slot > 3 || seenSlots[slot.Slot] || seenKeys[slot.AchievementKey] {
			return fmt.Errorf("invalid badge showcase selection")
		}
		seenSlots[slot.Slot], seenKeys[slot.AchievementKey] = true, true
		var exists int
		if err := tx.QueryRowContext(ctx, `SELECT 1 FROM achievement_unlocks
WHERE steam_id=? AND achievement_key=?`, steamID, slot.AchievementKey).Scan(&exists); err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("achievement %s is not unlocked", slot.AchievementKey)
			}
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM player_badge_showcase WHERE steam_id=?`, steamID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO player_badge_showcase_state (steam_id,configured_at)
VALUES (?,?) ON CONFLICT(steam_id) DO UPDATE SET configured_at=excluded.configured_at`, steamID, updatedAt); err != nil {
		return err
	}
	for _, slot := range slots {
		if _, err := tx.ExecContext(ctx, `INSERT INTO player_badge_showcase
(steam_id,slot,achievement_key,updated_at) VALUES (?,?,?,?)`, steamID, slot.Slot, slot.AchievementKey, updatedAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func boolInt64(value bool) int64 {
	if value {
		return 1
	}
	return 0
}
