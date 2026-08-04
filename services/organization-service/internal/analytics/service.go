package analytics

import (
	"time"
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

const (
	defaultWeeks = 12
	maxWeeks     = 52
	topCampaigns = 8
)

// notesCommon are returned with every response.
//
// They are part of the payload, not the docs, because a number without its
// caveat travels — into a screenshot, a board pack, a funding application —
// and the caveat needs to travel with it.
func notesCommon(d DonorStats) []string {
	n := []string{
		"Money shown is what settled through CivicOS. Donations go straight to the organization's own bank account, so this is not what it holds, has spent, or has left.",
	}
	// Only worth saying when it actually bites.
	if d.TotalDonations > d.AttributableDonations {
		n = append(n, "Donor counts cover people who were signed in when they gave. Donations made while signed out cannot be linked to a person, so unique and repeat donor figures are a floor, not a total.")
	}
	n = append(n, "\"People helped\" is not shown. Nothing in the record measures it, and a number typed in by an organization would be a claim sitting among figures taken from a ledger.")
	return n
}

func clampWeeks(w int) int {
	if w <= 0 {
		return defaultWeeks
	}
	if w > maxWeeks {
		return maxWeeks
	}
	return w
}

func (s *Service) ForOrg(orgID string, weeks int) (*OrgAnalytics, error) {
	weeks = clampWeeks(weeks)
	out := &OrgAnalytics{OrganizationID: orgID, GeneratedAt: time.Now().UTC()}

	var err error
	if out.FundsRaised, err = s.repo.OrgFundsRaised(orgID); err != nil {
		return nil, err
	}
	if out.Donors, err = s.repo.OrgDonorStats(orgID); err != nil {
		return nil, err
	}
	if out.Campaigns, err = s.repo.OrgCampaignStats(orgID); err != nil {
		return nil, err
	}
	if out.Trend, err = s.repo.OrgTrend(orgID, weeks); err != nil {
		return nil, err
	}
	if out.TopCampaigns, err = s.repo.OrgTopCampaigns(orgID, topCampaigns); err != nil {
		return nil, err
	}
	out.Notes = notesCommon(out.Donors)
	return out, nil
}

func (s *Service) ForPlatform(weeks int) (*PlatformAnalytics, error) {
	weeks = clampWeeks(weeks)
	out := &PlatformAnalytics{GeneratedAt: time.Now().UTC()}

	var err error
	if out.FundsRaised, err = s.repo.PlatformFundsRaised(); err != nil {
		return nil, err
	}
	if out.Donors, err = s.repo.PlatformDonorStats(); err != nil {
		return nil, err
	}
	if out.Campaigns, err = s.repo.PlatformCampaignStats(); err != nil {
		return nil, err
	}
	out.TotalCampaigns = out.Campaigns.Total
	if out.Organizations, err = s.repo.OrgCounts(); err != nil {
		return nil, err
	}
	if out.Countries, err = s.repo.Countries(); err != nil {
		return nil, err
	}
	if out.Categories, err = s.repo.Categories(); err != nil {
		return nil, err
	}
	if out.Emergency, err = s.repo.Emergency(); err != nil {
		return nil, err
	}
	if out.Review, err = s.repo.Review(); err != nil {
		return nil, err
	}
	if out.Trend, err = s.repo.PlatformTrend(weeks); err != nil {
		return nil, err
	}
	out.Notes = notesCommon(out.Donors)
	return out, nil
}
