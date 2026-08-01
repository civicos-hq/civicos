package flags

import (
	"net/http"
	"strings"

	"gorm.io/gorm"
)

// Who may raise a concern about a campaign.
//
// Every other flaggable thing on CivicOS is open to any verified citizen,
// because the cost of a bad report is a moderator's minute. A campaign is
// different in one direction and not the other: the campaign was reviewed by
// an admin before it published and is locked afterwards, so a report is not a
// claim that the platform approved something it shouldn't have — it is a
// claim about conduct that happened later, out where nobody at CivicOS can
// see. Money has already moved and cannot be clawed back.
//
// That cuts both ways. It means a concern is worth taking seriously; it also
// means an open report button is a lever a rival organization or a political
// opponent would be glad to have. So eligibility is limited to the two groups
// who can actually know something:
//
//   - **Donors.** They have standing and a receipt. If anyone is owed an
//     answer about where ₦50,000 went, it is the person who sent it.
//   - **Citizens in the campaign's own LGA.** They have the ground truth. An
//     admin in Abuja cannot know whether the Zaria culvert was rebuilt; a
//     neighbour can, and is very often someone who did not donate precisely
//     because it looked wrong.
//
// Note what this deliberately excludes: everyone else in the country,
// including someone in the same state but a different LGA. State-wide is too
// coarse to mean local knowledge — the whole of Kaduna is not "there".
//
// This is the first place in the product where a community tag acts as a
// gate rather than an audience label. That is a real departure and it is
// confined to this one decision on purpose.

// campaign is a read-only view of organization-service's table. Same
// shared-DB arrangement the discover feed and search use; the alternative is
// a synchronous cross-service call on a path that must not fail open.
type campaign struct {
	ID             string  `gorm:"type:uuid;primaryKey"`
	OrganizationID string  `gorm:"type:uuid"`
	CommunityID    *string `gorm:"type:uuid"`
	State          *string
	LGA            *string
	Status         string
}

func (campaign) TableName() string { return "campaigns" }

// donation is read only to answer "did this user give to this campaign".
// Deliberately not the full ledger model — nothing here needs amounts, and a
// gate that does not read money cannot leak it.
type donation struct {
	CampaignID  string  `gorm:"type:uuid"`
	DonorUserID *string `gorm:"type:uuid"`
	Status      string
}

func (donation) TableName() string { return "donations" }

// membershipCommunity is the caller's community, resolved to a place.
type membershipCommunity struct {
	ID    string `gorm:"type:uuid;primaryKey"`
	State string
	LGA   string
}

func (membershipCommunity) TableName() string { return "communities" }

// CampaignGate answers whether a user may raise a concern about a campaign.
type CampaignGate struct{ db *gorm.DB }

func NewCampaignGate(db *gorm.DB) *CampaignGate { return &CampaignGate{db: db} }

// Check returns nil when the user may report the campaign.
//
// Failure modes are deliberately distinguishable to the caller but NOT to the
// reporter: a 404 for a campaign that is not publicly visible, so this
// endpoint cannot be used to probe for draft or rejected campaigns by id.
func (g *CampaignGate) Check(campaignID, userID string) error {
	var c campaign
	if err := g.db.Where("id = ?", campaignID).First(&c).Error; err != nil {
		return &AppError{Code: "CAMPAIGN_NOT_FOUND", Message: "Campaign not found", Status: http.StatusNotFound}
	}
	// Mirrors organization-service's publicStatuses. A campaign with no
	// public page cannot be reported, because the reporter could not have
	// seen it — and confirming it exists would leak the review outcome.
	switch c.Status {
	case "PUBLISHED", "FUNDED", "COMPLETED", "REPORTED":
	default:
		return &AppError{Code: "CAMPAIGN_NOT_FOUND", Message: "Campaign not found", Status: http.StatusNotFound}
	}

	donor, err := g.isDonor(campaignID, userID)
	if err != nil {
		return err
	}
	if donor {
		return nil
	}

	local, err := g.isLocal(c, userID)
	if err != nil {
		return err
	}
	if local {
		return nil
	}

	return &AppError{
		Code: "NOT_ELIGIBLE_TO_REPORT",
		Message: "Only people who donated to this campaign, or who live in the area it serves, " +
			"can raise a concern about it.",
		Status: http.StatusForbidden,
	}
}

// isDonor reports whether the user has a SETTLED donation to this campaign.
//
// SETTLED only: a PENDING row means a payment was started, which anyone can
// do by opening a checkout and walking away. Standing comes from money that
// actually arrived.
func (g *CampaignGate) isDonor(campaignID, userID string) (bool, error) {
	var n int64
	err := g.db.Model(&donation{}).
		Where("campaign_id = ? AND donor_user_id = ? AND status = ?", campaignID, userID, "SETTLED").
		Count(&n).Error
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// isLocal reports whether the user belongs to a community in the same LGA as
// the campaign.
//
// The campaign's own state/lga wins over its community anchor, because a
// campaign is often raised for a specific ward rather than for the community
// record it happens to be filed under — the same reason the discover feed
// tiers campaigns by place first.
func (g *CampaignGate) isLocal(c campaign, userID string) (bool, error) {
	state, lga := placeOf(g.db, c)
	if state == "" || lga == "" {
		// A campaign with no resolvable place has no locals. Donors can still
		// report it; nobody else can. Failing closed is the right direction
		// for a gate.
		return false, nil
	}

	var mine []membershipCommunity
	err := g.db.Table("communities").
		Joins("JOIN user_community_memberships m ON m.community_id = communities.id").
		Where("m.user_id = ?", userID).
		Find(&mine).Error
	if err != nil {
		return false, err
	}
	for _, m := range mine {
		if strings.EqualFold(m.State, state) && strings.EqualFold(m.LGA, lga) {
			return true, nil
		}
	}
	return false, nil
}

// placeOf resolves a campaign to a (state, lga) pair: its own fields first,
// then its community anchor, then the owning organization's registered place.
// Returns empty strings when none of the three yields a place.
func placeOf(db *gorm.DB, c campaign) (string, string) {
	if c.State != nil && c.LGA != nil && *c.State != "" && *c.LGA != "" {
		return *c.State, *c.LGA
	}
	if c.CommunityID != nil {
		var comm membershipCommunity
		if err := db.Where("id = ?", *c.CommunityID).First(&comm).Error; err == nil {
			return comm.State, comm.LGA
		}
	}
	var org struct {
		State *string
		LGA   *string
	}
	if err := db.Table("organizations").Select("state, lga").
		Where("id = ?", c.OrganizationID).Scan(&org).Error; err == nil {
		if org.State != nil && org.LGA != nil {
			return *org.State, *org.LGA
		}
	}
	return "", ""
}
