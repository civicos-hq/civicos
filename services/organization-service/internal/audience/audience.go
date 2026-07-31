// Package audience resolves who should hear about a campaign event.
//
// Campaign notifications are emitted from three different packages —
// campaigns (approved, completed), milestones (completed) and donations
// (received, goal reached) — and they all need the same two questions
// answered: who runs this organization, and who paid for this campaign.
// Resolving that here keeps one definition of "the people with a stake in
// this campaign" rather than three that drift.
package audience

import (
	"log"

	"github.com/civicos/organization-service/internal/domain"
	"gorm.io/gorm"
)

type Resolver struct{ db *gorm.DB }

func New(db *gorm.DB) *Resolver { return &Resolver{db: db} }

// OrgMembers returns the user ids of everyone in the organization.
func (r *Resolver) OrgMembers(orgID string) []string {
	if r == nil || orgID == "" {
		return nil
	}
	var ids []string
	if err := r.db.Model(&domain.OrgMember{}).
		Where("organization_id = ?", orgID).
		Pluck("user_id", &ids).Error; err != nil {
		log.Printf("audience: org members lookup failed org=%s: %v", orgID, err)
		return nil
	}
	return ids
}

// Donors returns the user ids of everyone who has successfully funded this
// campaign.
//
// Two deliberate choices:
//
//   - **Only SETTLED donations count.** A pending intent is someone who
//     opened a checkout page, not a donor, and telling them about the spend
//     of money they never sent would be wrong.
//   - **Anonymous donors are included.** Anonymity governs the PUBLIC donor
//     list, not whether a person hears what happened to their own money.
//     Same reasoning as receipts.
//
// Guest donors have no account and therefore no user id; they are absent
// here by construction and were told by email instead.
func (r *Resolver) Donors(campaignID string) []string {
	if r == nil || campaignID == "" {
		return nil
	}
	var ids []string
	if err := r.db.Model(&domain.Donation{}).
		Where("campaign_id = ? AND status = ? AND donor_user_id IS NOT NULL",
			campaignID, domain.DonationSettled).
		Distinct().Pluck("donor_user_id", &ids).Error; err != nil {
		log.Printf("audience: donor lookup failed campaign=%s: %v", campaignID, err)
		return nil
	}
	return ids
}

// Stakeholders is donors plus org members, deduplicated.
//
// The overlap is real — someone at the organization may well have donated —
// and without the dedupe they would get two identical rows in their tray for
// one event.
func (r *Resolver) Stakeholders(campaignID, orgID string) []string {
	return Dedupe(append(r.Donors(campaignID), r.OrgMembers(orgID)...))
}

// Dedupe removes repeats while preserving order, so the first-listed
// audience stays first.
func Dedupe(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
