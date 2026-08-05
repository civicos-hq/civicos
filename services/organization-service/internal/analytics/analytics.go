// Package analytics answers "how is this going" for an organization and for
// the platform, over campaigns and the donation ledger.
//
// # What these numbers are, and are not
//
// Every money figure here is what settled THROUGH CivicOS. It is not what an
// organization holds, has spent, or has left. Donations settle straight to the
// organization's own bank account; CivicOS never held the money and has no
// view of it afterwards. The spec's transparency dashboard lists "Funds
// Withdrawn" and "Remaining Balance" — neither is computable here, and the
// funding plan already records that decision.
//
// Two metrics from the spec are deliberately absent, rather than approximated:
//
//   - **People Helped / Beneficiaries reached.** Nothing in the schema records
//     a beneficiary count. It could only be a number an organization typed in,
//     which is a claim, not a measurement — and putting it beside figures
//     derived from a ledger would lend it a precision it has not earned.
//   - **Funds Withdrawn / Remaining Balance.** See above.
//
// Where a number is knowably incomplete, the response says so in a sibling
// field rather than leaving the reader to assume it is whole. Repeat-donor
// counts are the important case: a donation made while signed out carries no
// user, so it cannot be tied to any other donation. The count is real but it
// is a floor, and `attributableDonations` is returned next to it.
// Every slice returned from this package is initialised empty rather than
// declared nil.
//
// A nil slice marshals to JSON `null`, not `[]`. That is invisible in
// development, where there is always demo data, and breaks on precisely the
// dataset a fresh deployment has: no campaigns, no donations. A client that
// iterates the field without guarding gets a TypeError and, in React, a blank
// page. Returning `[]` makes the empty case indistinguishable from the
// populated one for anyone consuming this.
package analytics

import (
	"time"

	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// MoneyByCurrency keeps totals separated by currency.
//
// Everything is NGN today and a campaign's currency is immutable after
// creation, so this is one entry in practice. Summing across currencies would
// nonetheless be a defect the moment a second one appears, and the cost of
// getting it right now is a map.
type MoneyByCurrency struct {
	Currency      string `json:"currency"`
	AmountMinor   int64  `json:"amountMinor"`
	DonationCount int64  `json:"donationCount"`
}

// TrendPoint is one bucket of the donation trend.
type TrendPoint struct {
	// PeriodStart is the first day of the bucket, UTC.
	PeriodStart time.Time `json:"periodStart"`
	AmountMinor int64     `json:"amountMinor"`
	Count       int64     `json:"count"`
}

// DonorStats covers the spec's "Repeat Donors" and "Average Donation".
type DonorStats struct {
	// UniqueDonors counts distinct signed-in donors. Donations made while
	// signed out are each their own person as far as the ledger knows, and
	// are NOT counted here — see AttributableDonations.
	UniqueDonors int64 `json:"uniqueDonors"`
	// RepeatDonors is signed-in donors with more than one settled donation.
	RepeatDonors int64 `json:"repeatDonors"`
	// AttributableDonations is how many settled donations carry a user at
	// all. Returned next to the two counts above because without it they read
	// as complete when they are a floor.
	AttributableDonations int64 `json:"attributableDonations"`
	TotalDonations        int64 `json:"totalDonations"`
	// AverageDonationMinor is over ALL settled donations, attributable or
	// not — an average does not need to know who gave.
	AverageDonation []MoneyByCurrency `json:"averageDonation"`
}

// CampaignStats covers "Campaign Performance" and "Completion Rate".
type CampaignStats struct {
	Total     int64            `json:"total"`
	ByStatus  map[string]int64 `json:"byStatus"`
	Published int64            `json:"everPublished"`
	Completed int64            `json:"completed"`
	Reported  int64            `json:"reported"`
	// CompletionRate is completed-or-reported over ever-published. Draft and
	// rejected campaigns are excluded from the denominator: a campaign that
	// never opened for donations cannot have failed to complete.
	CompletionRate float64 `json:"completionRate"`
	// ReportingRate is the spec's "percentage of campaigns publishing final
	// reports", over completed campaigns. The most telling number here: it is
	// the share of finished work that came with an account of the money.
	ReportingRate float64 `json:"reportingRate"`
}

// ─── Org-scoped ─────────────────────────────────────────────────────────

type OrgAnalytics struct {
	OrganizationID string            `json:"organizationId"`
	FundsRaised    []MoneyByCurrency `json:"fundsRaised"`
	Donors         DonorStats        `json:"donors"`
	Campaigns      CampaignStats     `json:"campaigns"`
	// Trend is weekly buckets over the requested window, oldest first, with
	// empty weeks included — a gap in giving is information, and a series
	// that silently skips it draws a misleading line.
	Trend []TrendPoint `json:"trend"`
	// TopCampaigns is per-campaign performance, best-raising first.
	TopCampaigns []CampaignPerformance `json:"topCampaigns"`
	GeneratedAt  time.Time             `json:"generatedAt"`
	// Notes carries the caveats a reader needs to interpret the numbers.
	Notes []string `json:"notes"`
}

type CampaignPerformance struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Slug        string  `json:"slug"`
	Status      string  `json:"status"`
	Currency    string  `json:"currency"`
	GoalMinor   int64   `json:"goalMinor"`
	RaisedMinor int64   `json:"raisedMinor"`
	DonorCount  int64   `json:"donorCount"`
	PercentGoal float64 `json:"percentOfGoal"`
}

func (r *Repository) OrgFundsRaised(orgID string) ([]MoneyByCurrency, error) {
	out := []MoneyByCurrency{}
	err := r.db.Raw(`
		SELECT currency,
		       COALESCE(SUM(gross_minor), 0) AS amount_minor,
		       COUNT(*)                      AS donation_count
		FROM donations
		WHERE organization_id = ? AND status = 'SETTLED'
		GROUP BY currency
		ORDER BY amount_minor DESC`, orgID).Scan(&out).Error
	return out, err
}

func (r *Repository) OrgDonorStats(orgID string) (DonorStats, error) {
	var s DonorStats
	// One pass for the counts that share a WHERE clause.
	var row struct {
		Unique        int64
		Repeat        int64
		Attributable  int64
		TotalDonation int64
	}
	err := r.db.Raw(`
		SELECT
		  COUNT(DISTINCT donor_user_id) FILTER (WHERE donor_user_id IS NOT NULL) AS unique,
		  COUNT(*) FILTER (WHERE donor_user_id IS NOT NULL)                      AS attributable,
		  COUNT(*)                                                               AS total_donation,
		  (SELECT COUNT(*) FROM (
		     SELECT donor_user_id FROM donations
		     WHERE organization_id = ? AND status = 'SETTLED' AND donor_user_id IS NOT NULL
		     GROUP BY donor_user_id HAVING COUNT(*) > 1) AS rd)                  AS repeat
		FROM donations
		WHERE organization_id = ? AND status = 'SETTLED'`, orgID, orgID).Scan(&row).Error
	if err != nil {
		return s, err
	}
	s.UniqueDonors, s.RepeatDonors = row.Unique, row.Repeat
	s.AttributableDonations, s.TotalDonations = row.Attributable, row.TotalDonation

	avg := []MoneyByCurrency{}
	err = r.db.Raw(`
		SELECT currency,
		       COALESCE(AVG(gross_minor), 0)::bigint AS amount_minor,
		       COUNT(*)                              AS donation_count
		FROM donations
		WHERE organization_id = ? AND status = 'SETTLED'
		GROUP BY currency`, orgID).Scan(&avg).Error
	s.AverageDonation = avg
	return s, err
}

func (r *Repository) OrgCampaignStats(orgID string) (CampaignStats, error) {
	return r.campaignStats("organization_id = ?", orgID)
}

// campaignStats is shared by the org and platform paths. `where` is a fixed
// fragment written here, never caller-supplied text.
func (r *Repository) campaignStats(where string, args ...any) (CampaignStats, error) {
	out := CampaignStats{ByStatus: map[string]int64{}}
	var rows []struct {
		Status string
		N      int64
	}
	if err := r.db.Raw(`SELECT status, COUNT(*) AS n FROM campaigns WHERE `+where+` GROUP BY status`, args...).
		Scan(&rows).Error; err != nil {
		return out, err
	}
	for _, row := range rows {
		out.ByStatus[row.Status] = row.N
		out.Total += row.N
	}
	// "Ever published" includes campaigns that have since moved on — a
	// completed campaign was published once, and counting only currently
	// PUBLISHED rows would make the completion rate rise as work finishes.
	var everPublished int64
	if err := r.db.Raw(`SELECT COUNT(*) FROM campaigns WHERE `+where+` AND published_at IS NOT NULL`, args...).
		Scan(&everPublished).Error; err != nil {
		return out, err
	}
	out.Published = everPublished
	out.Completed = out.ByStatus["COMPLETED"] + out.ByStatus["REPORTED"]
	out.Reported = out.ByStatus["REPORTED"]
	if out.Published > 0 {
		out.CompletionRate = round2(float64(out.Completed) / float64(out.Published) * 100)
	}
	if out.Completed > 0 {
		out.ReportingRate = round2(float64(out.Reported) / float64(out.Completed) * 100)
	}
	return out, nil
}

// OrgTrend returns weekly donation buckets for the last `weeks` weeks.
//
// generate_series fills empty weeks with zeros. Without it a chart would join
// the two weeks either side of a silence and draw a line straight through it.
func (r *Repository) OrgTrend(orgID string, weeks int) ([]TrendPoint, error) {
	out := []TrendPoint{}
	err := r.db.Raw(`
		WITH buckets AS (
		  SELECT generate_series(
		    date_trunc('week', NOW() AT TIME ZONE 'UTC') - make_interval(weeks => ?::int - 1),
		    date_trunc('week', NOW() AT TIME ZONE 'UTC'),
		    '1 week') AS period_start
		)
		SELECT b.period_start,
		       COALESCE(SUM(d.gross_minor), 0) AS amount_minor,
		       COUNT(d.id)                     AS count
		FROM buckets b
		LEFT JOIN donations d
		  ON date_trunc('week', d.settled_at AT TIME ZONE 'UTC') = b.period_start
		 AND d.organization_id = ? AND d.status = 'SETTLED'
		GROUP BY b.period_start
		ORDER BY b.period_start`, weeks, orgID).Scan(&out).Error
	return out, err
}

func (r *Repository) OrgTopCampaigns(orgID string, limit int) ([]CampaignPerformance, error) {
	out := []CampaignPerformance{}
	err := r.db.Raw(`
		SELECT id, title, slug, status, currency, goal_minor, raised_minor, donor_count
		FROM campaigns
		WHERE organization_id = ? AND published_at IS NOT NULL
		ORDER BY raised_minor DESC, created_at DESC
		LIMIT ?`, orgID, limit).Scan(&out).Error
	for i := range out {
		if out[i].GoalMinor > 0 {
			out[i].PercentGoal = round2(float64(out[i].RaisedMinor) / float64(out[i].GoalMinor) * 100)
		}
	}
	return out, err
}

func round2(f float64) float64 {
	return float64(int64(f*100+0.5)) / 100
}
