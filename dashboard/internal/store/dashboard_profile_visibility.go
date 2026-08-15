package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

func (s *dashboardStore) PlayerProfileVisibility(ctx context.Context, steamID string) (PlayerProfileVisibility, error) {
	var raw string
	var updatedAt int64
	err := s.db.QueryRowContext(ctx, `SELECT visible_sections_json,updated_at FROM player_profile_visibility WHERE steam_id=?`, steamID).Scan(&raw, &updatedAt)
	if err == sql.ErrNoRows {
		return PlayerProfileVisibility{VisibleSections: cloneProfileSections(DefaultPlayerProfileSections)}, nil
	}
	if err != nil {
		return PlayerProfileVisibility{}, fmt.Errorf("get player profile visibility: %w", err)
	}
	var sections []PlayerProfileSection
	if err := json.Unmarshal([]byte(raw), &sections); err != nil {
		return PlayerProfileVisibility{}, fmt.Errorf("decode player profile visibility: %w", err)
	}
	sections, err = normalizeProfileSections(sections)
	if err != nil {
		return PlayerProfileVisibility{}, fmt.Errorf("validate player profile visibility: %w", err)
	}
	return PlayerProfileVisibility{VisibleSections: sections, UpdatedAt: updatedAt}, nil
}

func (s *dashboardStore) ReplacePlayerProfileVisibility(ctx context.Context, steamID string, sections []PlayerProfileSection, updatedAt int64) (PlayerProfileVisibility, error) {
	normalized, err := normalizeProfileSections(sections)
	if err != nil {
		return PlayerProfileVisibility{}, err
	}
	raw, err := json.Marshal(normalized)
	if err != nil {
		return PlayerProfileVisibility{}, fmt.Errorf("encode player profile visibility: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO player_profile_visibility (steam_id,visible_sections_json,updated_at)
VALUES (?,?,?) ON CONFLICT(steam_id) DO UPDATE SET visible_sections_json=excluded.visible_sections_json,updated_at=excluded.updated_at`, steamID, string(raw), updatedAt)
	if err != nil {
		return PlayerProfileVisibility{}, fmt.Errorf("replace player profile visibility: %w", err)
	}
	return PlayerProfileVisibility{VisibleSections: normalized, UpdatedAt: updatedAt}, nil
}

func normalizeProfileSections(sections []PlayerProfileSection) ([]PlayerProfileSection, error) {
	requested := make(map[PlayerProfileSection]bool, len(sections))
	for _, section := range sections {
		if requested[section] {
			return nil, fmt.Errorf("duplicate profile section %q", section)
		}
		requested[section] = true
	}
	normalized := make([]PlayerProfileSection, 0, len(sections))
	for _, section := range PlayerProfileSections {
		if requested[section] {
			normalized = append(normalized, section)
			delete(requested, section)
		}
	}
	for section := range requested {
		return nil, fmt.Errorf("unknown profile section %q", section)
	}
	return normalized, nil
}

func cloneProfileSections(sections []PlayerProfileSection) []PlayerProfileSection {
	return append([]PlayerProfileSection(nil), sections...)
}
