package analytics

import "time"

// Platform-wide analytics, covering the spec's list: Total Campaigns, Total
// Funds Raised, Verified Organisations, Countries, Categories, and Emergency
// Response Metrics — plus the Success Metrics list, minus the two that have no
// data behind them (see the package comment).

type PlatformAnalytics struct {
	TotalCampaigns int64             `json:"totalCampaigns"`
	FundsRaised    []MoneyByCurrency `json:"fundsRaised"`
	Donors         DonorStats        `json:"donors"`
	Campaigns      CampaignStats     `json:"campaigns"`
	Organizations  OrgCounts         `json:"organizations"`
	Countries      []CountryCount    `json:"countries"`
	Categories     []CategoryCount   `json:"categories"`
	Emergency      EmergencyStats    `json:"emergency"`
	Review         ReviewStats       `json:"review"`
	Trend          []TrendPoint      `json:"trend"`
	GeneratedAt    time.Time         `json:"generatedAt"`
	Notes          []string          `json:"notes"`
}

type OrgCounts struct {
	Total int64 `json:"total"`
	// Verified carries the badge. FundingEligible is the stricter one: an
	// organization can be verified and still unable to take money, because
	// eligibility also needs a connected payout account. The gap between the
	// two is the onboarding drop-off, which is why both are here.
	Verified        int64 `json:"verified"`
	FundingEligible int64 `json:"fundingEligible"`
	WithCampaigns   int64 `json:"withPublishedCampaigns"`
}

type CountryCount struct {
	Country       string `json:"country"`
	Organizations int64  `json:"organizations"`
}

type CategoryCount struct {
	Category    string `json:"category"`
	Campaigns   int64  `json:"campaigns"`
	RaisedMinor int64  `json:"raisedMinor"`
	Currency    string `json:"currency"`
}

// EmergencyStats is the spec's "Emergency Response Metrics".
//
// The spec does not define them, so this reads the phrase the way an
// emergency appeal makes it matter: how many there were, how much reached
// them, and how quickly. MedianHoursToFirstDonation is the responsiveness
// figure — the gap between opening an appeal and the first naira arriving.
type EmergencyStats struct {
	Campaigns                  int64             `json:"campaigns"`
	FundsRaised                []MoneyByCurrency `json:"fundsRaised"`
	MedianHoursToFirstDonation *float64          `json:"medianHoursToFirstDonation"`
	// FundedWithin7Days is emergency campaigns that reached their goal inside
	// a week of publishing.
	FundedWithin7Days int64 `json:"fundedWithin7Days"`
}

// ReviewStats is the spec's "average verification time", measured on
// campaigns rather than organizations: campaigns carry submitted_at and
// reviewed_at, so the figure is exact. Organization verification runs through
// identity-service's application queue and is not visible from here.
type ReviewStats struct {
	Reviewed           int64    `json:"reviewed"`
	AverageHours       *float64 `json:"averageHours"`
	MedianHours        *float64 `json:"medianHours"`
	AwaitingReview     int64    `json:"awaitingReview"`
	OldestWaitingHours *float64 `json:"oldestWaitingHours"`
}

func (r *Repository) PlatformFundsRaised() ([]MoneyByCurrency, error) {
	out := []MoneyByCurrency{}
	err := r.db.Raw(`
		SELECT currency,
		       COALESCE(SUM(gross_minor), 0) AS amount_minor,
		       COUNT(*)                      AS donation_count
		FROM donations WHERE status = 'SETTLED'
		GROUP BY currency ORDER BY amount_minor DESC`).Scan(&out).Error
	return out, err
}

func (r *Repository) PlatformDonorStats() (DonorStats, error) {
	var s DonorStats
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
		     WHERE status = 'SETTLED' AND donor_user_id IS NOT NULL
		     GROUP BY donor_user_id HAVING COUNT(*) > 1) AS rd)                  AS repeat
		FROM donations WHERE status = 'SETTLED'`).Scan(&row).Error
	if err != nil {
		return s, err
	}
	s.UniqueDonors, s.RepeatDonors = row.Unique, row.Repeat
	s.AttributableDonations, s.TotalDonations = row.Attributable, row.TotalDonation

	avg := []MoneyByCurrency{}
	err = r.db.Raw(`
		SELECT currency, COALESCE(AVG(gross_minor), 0)::bigint AS amount_minor, COUNT(*) AS donation_count
		FROM donations WHERE status = 'SETTLED' GROUP BY currency`).Scan(&avg).Error
	s.AverageDonation = avg
	return s, err
}

func (r *Repository) PlatformCampaignStats() (CampaignStats, error) {
	return r.campaignStats("TRUE")
}

func (r *Repository) OrgCounts() (OrgCounts, error) {
	var c OrgCounts
	err := r.db.Raw(`
		SELECT
		  COUNT(*)                                          AS total,
		  COUNT(*) FILTER (WHERE verified)                  AS verified,
		  COUNT(*) FILTER (WHERE verified AND bank_account_verified
		                     AND psp_subaccount_code IS NOT NULL
		                     AND psp_subaccount_code <> '') AS funding_eligible
		FROM organizations`).Scan(&c).Error
	if err != nil {
		return c, err
	}
	err = r.db.Raw(`
		SELECT COUNT(DISTINCT organization_id) FROM campaigns WHERE published_at IS NOT NULL`).
		Scan(&c.WithCampaigns).Error
	return c, err
}

func (r *Repository) Countries() ([]CountryCount, error) {
	out := []CountryCount{}
	err := r.db.Raw(`
		SELECT COALESCE(NULLIF(TRIM(country), ''), 'Unspecified') AS country,
		       COUNT(*) AS organizations
		FROM organizations
		GROUP BY 1 ORDER BY organizations DESC, country`).Scan(&out).Error
	return out, err
}

// Categories counts only campaigns that reached the public, so the breakdown
// describes what citizens could actually give to rather than what was drafted.
func (r *Repository) Categories() ([]CategoryCount, error) {
	out := []CategoryCount{}
	err := r.db.Raw(`
		SELECT category,
		       COUNT(*)                        AS campaigns,
		       COALESCE(SUM(raised_minor), 0)  AS raised_minor,
		       COALESCE(MIN(currency), 'NGN')  AS currency
		FROM campaigns
		WHERE published_at IS NOT NULL
		GROUP BY category ORDER BY raised_minor DESC`).Scan(&out).Error
	return out, err
}

func (r *Repository) Emergency() (EmergencyStats, error) {
	var e EmergencyStats
	if err := r.db.Raw(`
		SELECT COUNT(*) FROM campaigns WHERE is_emergency AND published_at IS NOT NULL`).
		Scan(&e.Campaigns).Error; err != nil {
		return e, err
	}
	if err := r.db.Raw(`
		SELECT d.currency,
		       COALESCE(SUM(d.gross_minor), 0) AS amount_minor,
		       COUNT(*)                        AS donation_count
		FROM donations d
		JOIN campaigns c ON c.id = d.campaign_id
		WHERE d.status = 'SETTLED' AND c.is_emergency
		GROUP BY d.currency`).Scan(&e.FundsRaised).Error; err != nil {
		return e, err
	}
	// Median rather than mean: one appeal that sat unfunded for a month would
	// drag an average far away from the typical experience.
	var median *float64
	if err := r.db.Raw(`
		SELECT PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY hrs) FROM (
		  SELECT EXTRACT(EPOCH FROM (MIN(d.settled_at) - c.published_at)) / 3600 AS hrs
		  FROM campaigns c
		  JOIN donations d ON d.campaign_id = c.id AND d.status = 'SETTLED'
		  WHERE c.is_emergency AND c.published_at IS NOT NULL
		  GROUP BY c.id, c.published_at) AS t`).Scan(&median).Error; err != nil {
		return e, err
	}
	e.MedianHoursToFirstDonation = round2p(median)
	err := r.db.Raw(`
		SELECT COUNT(*) FROM campaigns c
		WHERE c.is_emergency AND c.published_at IS NOT NULL
		  AND c.raised_minor >= c.goal_minor
		  AND EXISTS (
		    SELECT 1 FROM donations d
		    WHERE d.campaign_id = c.id AND d.status = 'SETTLED'
		      AND d.settled_at <= c.published_at + INTERVAL '7 days')`).
		Scan(&e.FundedWithin7Days).Error
	return e, err
}

// Review measures how long a campaign waits for a decision — the spec's
// "average verification time", and the number a reviewer's own backlog shows
// up in. OldestWaitingHours is the one to watch: an average stays comfortable
// while a single campaign sits for a fortnight.
func (r *Repository) Review() (ReviewStats, error) {
	var s ReviewStats
	var row struct {
		N   int64
		Avg *float64
		Med *float64
	}
	if err := r.db.Raw(`
		SELECT COUNT(*) AS n,
		       AVG(EXTRACT(EPOCH FROM (reviewed_at - submitted_at)) / 3600)  AS avg,
		       PERCENTILE_CONT(0.5) WITHIN GROUP (
		         ORDER BY EXTRACT(EPOCH FROM (reviewed_at - submitted_at)) / 3600) AS med
		FROM campaigns
		WHERE reviewed_at IS NOT NULL AND submitted_at IS NOT NULL
		  AND reviewed_at >= submitted_at`).Scan(&row).Error; err != nil {
		return s, err
	}
	s.Reviewed, s.AverageHours, s.MedianHours = row.N, round2p(row.Avg), round2p(row.Med)

	var wait struct {
		N      int64
		Oldest *float64
	}
	err := r.db.Raw(`
		SELECT COUNT(*) AS n,
		       MAX(EXTRACT(EPOCH FROM (NOW() - submitted_at)) / 3600) AS oldest
		FROM campaigns
		WHERE status = 'PENDING_REVIEW' AND submitted_at IS NOT NULL`).Scan(&wait).Error
	s.AwaitingReview, s.OldestWaitingHours = wait.N, round2p(wait.Oldest)
	return s, err
}

func (r *Repository) PlatformTrend(weeks int) ([]TrendPoint, error) {
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
		 AND d.status = 'SETTLED'
		GROUP BY b.period_start ORDER BY b.period_start`, weeks).Scan(&out).Error
	return out, err
}

func round2p(f *float64) *float64 {
	if f == nil {
		return nil
	}
	v := round2(*f)
	return &v
}
