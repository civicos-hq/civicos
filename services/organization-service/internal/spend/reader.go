package spend

import "github.com/civicos/organization-service/internal/campaigns"

// Reader adapts this package to the campaigns package's SpendReader, so the
// public campaign page can show reported spend without campaigns importing
// spend's storage.
type Reader struct{ svc *Service }

func NewReader(svc *Service) *Reader { return &Reader{svc: svc} }

// SummaryFor aggregates reported spend for one campaign against what the
// ledger says it received.
func (r *Reader) SummaryFor(campaignID string, receivedMinor int64, currency string) (*campaigns.SpendSummary, error) {
	records, err := r.svc.ListForCampaign(campaignID)
	if err != nil {
		return nil, err
	}
	s := Summarise(records, receivedMinor, currency)
	return &campaigns.SpendSummary{
		ReportedMinor:   s.ReportedMinor,
		UnreportedMinor: s.UnreportedMinor,
		ExceedsReceived: s.ExceedsReceived,
		RecordCount:     s.RecordCount,
		PerMilestone:    s.PerMilestone,
	}, nil
}
