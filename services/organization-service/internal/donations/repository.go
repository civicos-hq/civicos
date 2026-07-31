package donations

import (
	"errors"
	"time"

	"github.com/civicos/organization-service/internal/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

var ErrDuplicate = errors.New("donation already exists")

func (r *Repository) Create(d *domain.Donation) error {
	err := r.db.Create(d).Error
	if err != nil && isUniqueViolation(err) {
		// The unique indexes on provider_ref and idempotency_key are doing
		// their job — a double-tapped donate button or a retried intent.
		return ErrDuplicate
	}
	return err
}

func (r *Repository) FindByRef(provider, ref string) (*domain.Donation, error) {
	var d domain.Donation
	return &d, r.db.Where("provider = ? AND provider_ref = ?", provider, ref).First(&d).Error
}

func (r *Repository) FindByIdempotencyKey(key string) (*domain.Donation, error) {
	var d domain.Donation
	return &d, r.db.Where("idempotency_key = ?", key).First(&d).Error
}

func (r *Repository) ListSettledForCampaign(campaignID string) ([]domain.Donation, error) {
	var list []domain.Donation
	return list, r.db.Where("campaign_id = ? AND status = ?", campaignID, domain.DonationSettled).
		Order("settled_at desc").Find(&list).Error
}

// ListStale returns donations sitting in a status longer than they should,
// oldest first. Reconciliation uses it to find payments whose webhook never
// arrived. Oldest first matters: if the limit truncates the sweep, the rows
// most likely to have been forgotten are the ones that get looked at.
func (r *Repository) ListStale(status domain.DonationStatus, olderThan time.Time, limit int) ([]domain.Donation, error) {
	var list []domain.Donation
	return list, r.db.Where("status = ? AND created_at < ?", status, olderThan.UTC()).
		Order("created_at asc").Limit(limit).Find(&list).Error
}

// ListSettledSince returns recently settled donations for re-checking
// against the provider. Bounded by a window because each row costs one
// provider call.
func (r *Repository) ListSettledSince(since time.Time, limit int) ([]domain.Donation, error) {
	var list []domain.Donation
	return list, r.db.Where("status = ? AND settled_at >= ?", domain.DonationSettled, since.UTC()).
		Order("settled_at asc").Limit(limit).Find(&list).Error
}

// MarkReceiptSent records that the donor was emailed. Deliberately a plain
// UPDATE with no status guard: it is only ever called after a send actually
// succeeded, and refusing to record a receipt that went out would cause a
// duplicate on the next sweep.
func (r *Repository) MarkReceiptSent(donationID string) error {
	return r.db.Model(&domain.Donation{}).Where("id = ?", donationID).
		Update("receipt_sent_at", gorm.Expr("NOW()")).Error
}

// ListSettledWithoutReceipt finds donors whose money we banked but who were
// never told. Oldest first, so the longest-waiting donor is served first if
// the limit truncates.
func (r *Repository) ListSettledWithoutReceipt(settledBefore time.Time, limit int) ([]domain.Donation, error) {
	var list []domain.Donation
	return list, r.db.Where(
		"status = ? AND receipt_sent_at IS NULL AND settled_at IS NOT NULL AND settled_at < ?",
		domain.DonationSettled, settledBefore.UTC(),
	).Order("settled_at asc").Limit(limit).Find(&list).Error
}

// Campaign is a plain read used to validate an intent before creating it.
func (r *Repository) Campaign(id string) (*domain.Campaign, error) {
	var c domain.Campaign
	return &c, r.db.Where("id = ?", id).First(&c).Error
}

func (r *Repository) Org(id string) (*domain.Organization, error) {
	var o domain.Organization
	return &o, r.db.Where("id = ?", id).First(&o).Error
}

// SettleResult reports what a settlement did, so the caller can decide
// whether a campaign reached its goal without re-reading.
type SettleResult struct {
	AlreadySettled bool
	Campaign       *domain.Campaign
}

// Settle marks a donation SETTLED and rebuilds the campaign's cached
// projection — in ONE transaction, and with the projection SUMMED from the
// ledger rather than incremented.
//
// Both of those matter:
//
//   - One transaction, because a crash between the two writes would leave a
//     settled donation that no total reflects. Money would be missing from
//     the public page with nothing to indicate it.
//   - SUM rather than `raised_minor = raised_minor + x`, because an
//     increment is only correct if it happens exactly once. Webhooks retry.
//     A SUM over settled rows is correct however many times it runs, so a
//     replayed delivery converges on the same number instead of inflating it.
//
// The row is re-read FOR UPDATE inside the transaction so two concurrent
// deliveries for the same reference cannot both pass the already-settled
// check.
func (r *Repository) Settle(donationID string, pspFeeMinor int64) (SettleResult, error) {
	var out SettleResult
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var d domain.Donation
		if err := tx.Clauses(lockForUpdate()).Where("id = ?", donationID).First(&d).Error; err != nil {
			return err
		}
		if d.Status == domain.DonationSettled {
			// Idempotent: a replayed webhook lands here and changes nothing.
			out.AlreadySettled = true
			return nil
		}

		if err := tx.Model(&domain.Donation{}).Where("id = ?", d.ID).Updates(map[string]any{
			"status":        domain.DonationSettled,
			"psp_fee_minor": pspFeeMinor,
			"settled_at":    gorm.Expr("NOW()"),
		}).Error; err != nil {
			return err
		}

		// Rebuild the projection from the ledger.
		var agg struct {
			Total  int64
			Donors int64
		}
		if err := tx.Model(&domain.Donation{}).
			Where("campaign_id = ? AND status = ?", d.CampaignID, domain.DonationSettled).
			Select("COALESCE(SUM(gross_minor),0) AS total, COUNT(*) AS donors").
			Scan(&agg).Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.Campaign{}).Where("id = ?", d.CampaignID).Updates(map[string]any{
			"raised_minor": agg.Total,
			"donor_count":  agg.Donors,
		}).Error; err != nil {
			return err
		}

		var c domain.Campaign
		if err := tx.Where("id = ?", d.CampaignID).First(&c).Error; err != nil {
			return err
		}
		out.Campaign = &c
		return nil
	})
	return out, err
}

// MarkFailed records a terminal non-success. Deliberately does NOT touch the
// projection: a failed donation never counted toward it.
func (r *Repository) MarkFailed(donationID string, status domain.DonationStatus) error {
	return r.db.Model(&domain.Donation{}).Where("id = ?", donationID).Updates(map[string]any{
		"status":    status,
		"failed_at": gorm.Expr("NOW()"),
	}).Error
}

// SetCampaignFunded flips a campaign to FUNDED. Called only after the goal
// is met, and only by the settlement path — see campaigns.ActorSystem: the
// platform asserts this from ledger truth, never a user.
func (r *Repository) SetCampaignFunded(campaignID string) error {
	return r.db.Model(&domain.Campaign{}).
		Where("id = ? AND status = ?", campaignID, domain.CampaignPublished).
		Update("status", domain.CampaignFunded).Error
}

// RecordWebhook stores every delivery, verified or not. Returns false if this
// event id was already recorded — the second dedupe layer, in front of the
// donation-level one, so a replay is cheap to detect.
func (r *Repository) RecordWebhook(e *domain.WebhookEvent) (bool, error) {
	err := r.db.Create(e).Error
	if err != nil {
		if isUniqueViolation(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *Repository) MarkWebhookHandled(id string, note *string) error {
	return r.db.Model(&domain.WebhookEvent{}).Where("id = ?", id).
		Updates(map[string]any{"handled": true, "note": note}).Error
}

// ConnectSubaccount records the org's payout destination. Stores the code,
// the bank name and the last four digits only — never the account number.
func (r *Repository) ConnectSubaccount(orgID, provider, code, bankName, last4 string) error {
	return r.db.Model(&domain.Organization{}).Where("id = ?", orgID).Updates(map[string]any{
		"psp_provider":        provider,
		"psp_subaccount_code": code,
		"psp_bank_name":       bankName,
		"psp_account_last4":   last4,
		"psp_connected_at":    gorm.Expr("NOW()"),
	}).Error
}
