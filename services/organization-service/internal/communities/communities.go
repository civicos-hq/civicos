// Package communities is a read-only view into the community-membership
// table owned by identity-service. Same shared-DB pattern as the audit
// and notifications packages: we redeclare the model here with
// TableName pinned so GORM can read it without owning the schema.
//
// If services move to isolated databases later, replace the direct DB
// read with an HTTP call to identity-service (e.g. GET
// /v1/communities/:id/members) or a NATS-fed materialized view.
package communities

import (
	"time"

	"gorm.io/gorm"
)

// userCommunityMembership mirrors the identity-service model. Only
// the fields we actually read here are declared; other columns exist
// on the row and GORM will happily leave them untouched.
type userCommunityMembership struct {
	ID          string    `gorm:"type:uuid;primaryKey"`
	UserID      string    `gorm:"type:uuid;not null"`
	CommunityID string    `gorm:"type:uuid;not null"`
	JoinedAt    time.Time `gorm:"not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (userCommunityMembership) TableName() string { return "user_community_memberships" }

// Reader is the interface consumers depend on. Kept narrow so tests
// can substitute a fake without pulling in GORM.
type Reader interface {
	FindMemberIDs(communityID string) ([]string, error)
}

type dbReader struct{ db *gorm.DB }

// NewReader wires a GORM-backed community-membership reader.
func NewReader(db *gorm.DB) Reader { return &dbReader{db: db} }

// FindMemberIDs returns every user ID that has joined the given
// community. Empty slice when the community has no members — callers
// use the length to decide whether to skip the notification fan-out.
//
// Note: we don't filter out deleted or banned users here. The
// notification writer (community-service via the shared table) will
// eventually skip banned readers on read; the write cost is trivial.
// A cleaner filter needs a JOIN into `users` — deferred.
func (r *dbReader) FindMemberIDs(communityID string) ([]string, error) {
	if communityID == "" {
		return []string{}, nil
	}
	var ids []string
	err := r.db.Model(&userCommunityMembership{}).
		Where("community_id = ?", communityID).
		Pluck("user_id", &ids).Error
	if ids == nil {
		ids = []string{}
	}
	return ids, err
}
