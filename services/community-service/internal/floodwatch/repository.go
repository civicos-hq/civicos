package floodwatch

import (
	"time"

	"github.com/civicos/community-service/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) CommunitiesWithCoordinates() ([]domain.Community, error) {
	var list []domain.Community
	return list, r.db.
		Where("latitude IS NOT NULL AND longitude IS NOT NULL").
		Find(&list).Error
}

// UpsertAlerts writes the current sweep.
//
// Conflict target is (community_id, gauge_id), so a pairing seen again is
// refreshed rather than duplicated. NotifiedSeverity and NotifiedAt are
// deliberately NOT in the update set: they record what citizens were told,
// which is not something a forecast refresh may overwrite. Losing them
// here would re-alarm everyone on the next sweep.
func (r *Repository) UpsertAlerts(alerts []domain.CommunityFloodAlert) error {
	if len(alerts) == 0 {
		return nil
	}
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "community_id"}, {Name: "gauge_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"severity", "trend", "river", "site_name", "distance_km",
			"gauge_latitude", "gauge_longitude",
			"issued_at", "forecast_start_at", "forecast_end_at",
			"last_seen_at", "updated_at",
		}),
	}).CreateInBatches(alerts, 200).Error
}

// PendingNotifications finds rows whose severity has risen above whatever
// citizens were last told.
//
// The comparison is on a severity ORDER, not equality: a forecast easing
// from EXTREME to SEVERE is still alarming but is not news, and pushing on
// it would train people to ignore the channel. Rows never notified qualify
// as soon as they are alerting at all.
func (r *Repository) PendingNotifications() ([]domain.CommunityFloodAlert, error) {
	const rank = `CASE severity
		WHEN 'EXTREME' THEN 3
		WHEN 'SEVERE' THEN 2
		WHEN 'ABOVE_NORMAL' THEN 1
		ELSE 0 END`
	const notifiedRank = `CASE notified_severity
		WHEN 'EXTREME' THEN 3
		WHEN 'SEVERE' THEN 2
		WHEN 'ABOVE_NORMAL' THEN 1
		ELSE 0 END`

	var list []domain.CommunityFloodAlert
	return list, r.db.
		Where(rank + " > 0").
		Where("notified_severity IS NULL OR " + rank + " > " + notifiedRank).
		Find(&list).Error
}

func (r *Repository) MarkNotified(id, severity string, at time.Time) error {
	return r.db.Model(&domain.CommunityFloodAlert{}).
		Where("id = ?", id).
		Updates(map[string]any{"notified_severity": severity, "notified_at": at}).Error
}

// ActiveForCommunity returns alerting forecasts still being reported.
//
// Rows older than staleBefore are excluded: an upstream that has stopped
// reporting tells us nothing, and a stale row on screen reads as current.
func (r *Repository) ActiveForCommunity(communityID string, staleBefore time.Time) ([]domain.CommunityFloodAlert, error) {
	var list []domain.CommunityFloodAlert
	return list, r.db.
		Where("community_id = ? AND last_seen_at >= ?", communityID, staleBefore).
		Where("severity IN ?", []string{"ABOVE_NORMAL", "SEVERE", "EXTREME"}).
		Order(`CASE severity WHEN 'EXTREME' THEN 0 WHEN 'SEVERE' THEN 1 ELSE 2 END, distance_km asc`).
		Find(&list).Error
}

// MemberIDs lists the users to notify for a community. Reads
// identity-service's membership table from the shared database, the same
// arrangement Discover and search already use.
func (r *Repository) MemberIDs(communityID string) ([]string, error) {
	var ids []string
	return ids, r.db.
		Table("user_community_memberships").
		Where("community_id = ?", communityID).
		Pluck("user_id", &ids).Error
}
