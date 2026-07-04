// Package moderation contains cross-package helpers for content-hide
// enforcement — the read-side gate that translates HIDDEN flags in
// content_flags into invisible/placeholder content on the citizen
// surface.
package moderation

import "gorm.io/gorm"

// HiddenSet returns the subset of ids that have a HIDDEN moderator flag
// filed against them for the given content type. Empty input → empty
// output (no query). Failure to reach content_flags → empty output
// (fail open — a lookup outage shouldn't make hidden content vanish
// forever; the next query will catch it).
func HiddenSet(db *gorm.DB, contentType string, ids []string) map[string]struct{} {
	set := map[string]struct{}{}
	if len(ids) == 0 {
		return set
	}
	var hidden []string
	if err := db.
		Table("content_flags").
		Where("content_type = ? AND status = ? AND content_id IN ?", contentType, "HIDDEN", ids).
		Pluck("content_id", &hidden).Error; err != nil {
		return set
	}
	for _, id := range hidden {
		set[id] = struct{}{}
	}
	return set
}

// Placeholder is what replaces a HIDDEN comment's content on the
// citizen surface. Kept in one place so every content type surfaces
// the same phrasing.
const (
	PlaceholderContent    = "[Removed by moderator]"
	PlaceholderAuthorName = "[Removed]"
)
