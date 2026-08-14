package store

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

const relationshipContractVersion int64 = 1

func (s *statsStore) PlayerRelationships(ctx context.Context, steamID string, query PlayerRelationshipQuery) (PlayerRelationshipPage, error) {
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	query = normalizeRelationshipQuery(query)

	peers := make(map[string]*PlayerRelationship)
	if err := s.loadCompanionRelationships(queryCtx, steamID, query.PlayerFilter, peers); err != nil {
		return PlayerRelationshipPage{}, err
	}
	if err := s.loadDirectedRelationships(queryCtx, steamID, query.PlayerFilter, peers); err != nil {
		return PlayerRelationshipPage{}, err
	}

	items := make([]PlayerRelationship, 0, len(peers))
	for _, peer := range peers {
		finalizeRelationshipDirection(&peer.Outgoing)
		finalizeRelationshipDirection(&peer.Incoming)
		peer.MutualSupport = peer.Outgoing.SupportActions + peer.Incoming.SupportActions
		items = append(items, *peer)
	}
	summaries := relationshipSummaries(items)
	sortRelationships(items, query.Sort, query.Order)

	result := PlayerRelationshipPage{
		RelationshipVersion: relationshipContractVersion,
		Page:                query.Page, PageSize: query.PageSize, Total: int64(len(items)),
		Summaries: summaries, Items: make([]PlayerRelationship, 0),
	}
	start := (query.Page - 1) * query.PageSize
	if start >= int64(len(items)) {
		return result, nil
	}
	end := start + query.PageSize
	if end > int64(len(items)) {
		end = int64(len(items))
	}
	result.Items = append(result.Items, items[start:end]...)
	return result, nil
}

func normalizeRelationshipQuery(query PlayerRelationshipQuery) PlayerRelationshipQuery {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 || query.PageSize > 100 {
		query.PageSize = 20
	}
	allowed := map[string]bool{
		"player_name": true, "shared_rounds": true, "shared_seconds": true,
		"outgoing_support": true, "incoming_support": true, "mutual_support": true,
		"outgoing_healing": true, "incoming_healing": true,
		"outgoing_friendly_fire": true, "incoming_friendly_fire": true,
	}
	if !allowed[query.Sort] {
		query.Sort = "shared_rounds"
	}
	if query.Order != "asc" && query.Order != "desc" {
		query.Order = "desc"
	}
	return query
}

func (s *statsStore) loadCompanionRelationships(ctx context.Context, steamID string, filter PlayerFilter, peers map[string]*PlayerRelationship) error {
	args := []any{steamID}
	where := "s.steam_id=" + s.bind(len(args))
	where = s.relationshipRoundFilter(where, "r", filter, &args)
	subjectEnd := "COALESCE(s.ended_at,s.last_saved_at)"
	peerEnd := "COALESCE(p.ended_at,p.last_saved_at)"
	latestStart := "CASE WHEN s.started_at>p.started_at THEN s.started_at ELSE p.started_at END"
	earliestEnd := "CASE WHEN " + subjectEnd + "<" + peerEnd + " THEN " + subjectEnd + " ELSE " + peerEnd + " END"
	overlap := "(" + earliestEnd + "-" + latestStart + ")"
	statement := `SELECT p.steam_id,MAX(identity.last_name),COUNT(DISTINCT s.round_id),COALESCE(SUM(` + overlap + `),0)
FROM lps_player_segments s
JOIN lps_rounds r ON r.round_id=s.round_id
JOIN lps_player_segments p ON p.round_id=s.round_id AND p.side=s.side AND p.steam_id<>s.steam_id
JOIN lps_players identity ON identity.steam_id=p.steam_id
WHERE ` + where + ` AND ` + earliestEnd + `>` + latestStart + `
GROUP BY p.steam_id`
	rows, err := s.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var peer PlayerRelationship
		if err := rows.Scan(&peer.PeerSteamID, &peer.PeerName, &peer.SharedRounds, &peer.SharedSeconds); err != nil {
			return err
		}
		copy := peer
		peers[peer.PeerSteamID] = &copy
	}
	return rows.Err()
}

func (s *statsStore) loadDirectedRelationships(ctx context.Context, steamID string, filter PlayerFilter, peers map[string]*PlayerRelationship) error {
	args := make([]any, 0, 8)
	branch := func(outgoing bool) string {
		args = append(args, steamID)
		matchColumn, peerColumn, direction := "rel.actor_steam_id", "rel.target_steam_id", 0
		if !outgoing {
			matchColumn, peerColumn, direction = "rel.target_steam_id", "rel.actor_steam_id", 1
		}
		where := matchColumn + "=" + s.bind(len(args)) + " AND rel.relationship_version=1"
		where = s.relationshipRoundFilter(where, "r", filter, &args)
		return fmt.Sprintf(`SELECT %s peer_steam_id,%d direction,
rel.incap_revives,rel.ledge_rescues,rel.defib_revives,rel.smoker_rescues,rel.hunter_rescues,rel.jockey_rescues,rel.charger_rescues,
rel.control_rescue_duration_ms,rel.medkits_used,rel.medkit_healing,rel.black_white_restores,rel.friendly_fire_damage
FROM lps_player_round_relationship_stats rel JOIN lps_rounds r ON r.round_id=rel.round_id WHERE %s`, peerColumn, direction, where)
	}
	union := branch(true) + " UNION ALL " + branch(false)
	statement := `SELECT directed.peer_steam_id,MAX(identity.last_name),directed.direction,
SUM(directed.incap_revives),SUM(directed.ledge_rescues),SUM(directed.defib_revives),SUM(directed.smoker_rescues),SUM(directed.hunter_rescues),SUM(directed.jockey_rescues),SUM(directed.charger_rescues),
SUM(directed.control_rescue_duration_ms),SUM(directed.medkits_used),SUM(directed.medkit_healing),SUM(directed.black_white_restores),SUM(directed.friendly_fire_damage)
FROM (` + union + `) directed JOIN lps_players identity ON identity.steam_id=directed.peer_steam_id
GROUP BY directed.peer_steam_id,directed.direction`
	rows, err := s.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var peerID, peerName string
		var direction int
		var value PlayerRelationshipDirection
		if err := rows.Scan(&peerID, &peerName, &direction, &value.IncapRevives, &value.LedgeRescues, &value.DefibRevives, &value.SmokerRescues, &value.HunterRescues, &value.JockeyRescues, &value.ChargerRescues, &value.ControlRescueDurationMS, &value.MedkitsUsed, &value.MedkitHealing, &value.BlackWhiteRestores, &value.FriendlyFireDamage); err != nil {
			return err
		}
		peer := peers[peerID]
		if peer == nil {
			peer = &PlayerRelationship{PeerSteamID: peerID, PeerName: peerName}
			peers[peerID] = peer
		} else if peer.PeerName == "" {
			peer.PeerName = peerName
		}
		if direction == 0 {
			peer.Outgoing = value
		} else {
			peer.Incoming = value
		}
	}
	return rows.Err()
}

func (s *statsStore) relationshipRoundFilter(where, alias string, filter PlayerFilter, args *[]any) string {
	if filter.Cutoff > 0 {
		*args = append(*args, filter.Cutoff)
		where += " AND " + alias + ".started_at>=" + s.bind(len(*args))
	}
	if filter.ServerKey != "" {
		*args = append(*args, filter.ServerKey)
		where += " AND " + alias + ".server_key=" + s.bind(len(*args))
	}
	if filter.GameMode == "pve" || filter.GameMode == "versus" {
		*args = append(*args, filter.GameMode)
		where += " AND " + alias + ".mode_family=" + s.bind(len(*args))
	}
	return where
}

func finalizeRelationshipDirection(value *PlayerRelationshipDirection) {
	value.SpecialRescues = value.SmokerRescues + value.HunterRescues + value.JockeyRescues + value.ChargerRescues
	value.SupportActions = value.IncapRevives + value.LedgeRescues + value.DefibRevives + value.SpecialRescues + value.MedkitsUsed
	if value.SpecialRescues > 0 {
		average := float64(value.ControlRescueDurationMS) / float64(value.SpecialRescues)
		value.AverageControlRescueMS = &average
	}
}

func relationshipSummaries(items []PlayerRelationship) PlayerRelationshipSummaries {
	var result PlayerRelationshipSummaries
	for index := range items {
		item := &items[index]
		updateRelationshipSummary(&result.MostCompanion, item, item.SharedRounds, item.SharedSeconds)
		updateRelationshipSummary(&result.MostSupported, item, item.Outgoing.SupportActions, 0)
		updateRelationshipSummary(&result.MostSupportedBy, item, item.Incoming.SupportActions, 0)
		updateRelationshipSummary(&result.MostMutual, item, item.MutualSupport, 0)
	}
	if result.MostCompanion != nil {
		result.MostCompanion.SupportActions = 0
	}
	return result
}

func updateRelationshipSummary(current **PlayerRelationshipSummary, item *PlayerRelationship, primary, secondary int64) {
	if primary <= 0 {
		return
	}
	candidate := &PlayerRelationshipSummary{PeerSteamID: item.PeerSteamID, PeerName: item.PeerName, SharedRounds: item.SharedRounds, SharedSeconds: item.SharedSeconds, SupportActions: primary}
	if *current == nil {
		*current = candidate
		return
	}
	currentPrimary, currentSecondary := (*current).SupportActions, int64(0)
	if (*current).SharedRounds > 0 && secondary > 0 {
		currentPrimary, currentSecondary = (*current).SharedRounds, (*current).SharedSeconds
	}
	if primary > currentPrimary || (primary == currentPrimary && (secondary > currentSecondary || (secondary == currentSecondary && item.PeerSteamID < (*current).PeerSteamID))) {
		*current = candidate
	}
}

func sortRelationships(items []PlayerRelationship, field, order string) {
	value := func(item PlayerRelationship) int64 {
		switch field {
		case "shared_rounds":
			return item.SharedRounds
		case "shared_seconds":
			return item.SharedSeconds
		case "outgoing_support":
			return item.Outgoing.SupportActions
		case "incoming_support":
			return item.Incoming.SupportActions
		case "mutual_support":
			return item.MutualSupport
		case "outgoing_healing":
			return item.Outgoing.MedkitHealing
		case "incoming_healing":
			return item.Incoming.MedkitHealing
		case "outgoing_friendly_fire":
			return item.Outgoing.FriendlyFireDamage
		case "incoming_friendly_fire":
			return item.Incoming.FriendlyFireDamage
		}
		return 0
	}
	sort.SliceStable(items, func(i, j int) bool {
		if field == "player_name" {
			left, right := strings.ToLower(items[i].PeerName), strings.ToLower(items[j].PeerName)
			if left != right {
				return (left < right) == (order == "asc")
			}
		} else if value(items[i]) != value(items[j]) {
			return (value(items[i]) < value(items[j])) == (order == "asc")
		}
		return items[i].PeerSteamID < items[j].PeerSteamID
	})
}
