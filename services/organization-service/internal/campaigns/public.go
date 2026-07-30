package campaigns

import (
	"time"

	"github.com/civicos/organization-service/internal/domain"
)

// PublicCampaign is what an unauthenticated visitor sees.
//
// Returning domain.Campaign directly would leak the review trail —
// `approvalStatus`, `reviewNote`, `reviewedById`, `submittedAt` — which is a
// conversation between the organization and the platform, not public record.
// `reviewNote` is the sharp one: "attach the works department quote, this
// looks inflated" is feedback the org is owed privately, and publishing it
// would defame organizations on the strength of an in-progress review.
//
// So this is an explicit allow-list, not a struct with a few fields hidden.
// New fields on domain.Campaign stay private until somebody deliberately
// adds them here — the safe direction for a surface that will eventually ask
// the public for money.
type PublicCampaign struct {
	ID       string                  `json:"id"`
	Slug     string                  `json:"slug"`
	Title    string                  `json:"title"`
	Summary  string                  `json:"summary"`
	Category domain.CampaignCategory `json:"category"`
	Status   domain.CampaignStatus   `json:"status"`

	Currency string `json:"currency"`
	// Goal and progress are public by design — this is the transparency the
	// feature exists for. In Phase 2 raised is always 0; Phase 3 recomputes
	// it from the donation ledger.
	GoalMinor   int64 `json:"goalMinor"`
	RaisedMinor int64 `json:"raisedMinor"`
	DonorCount  int   `json:"donorCount"`

	CoverImageURL *string `json:"coverImageUrl,omitempty"`
	State         *string `json:"state,omitempty"`
	LGA           *string `json:"lga,omitempty"`
	CommunityID   *string `json:"communityId,omitempty"`

	OrganizationID   string `json:"organizationId"`
	OrganizationName string `json:"organizationName,omitempty"`

	IsEmergency bool       `json:"isEmergency"`
	StartDate   *time.Time `json:"startDate,omitempty"`
	EndDate     *time.Time `json:"endDate,omitempty"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`

	// Paused campaigns stay visible with the reason CODE shown — a donor
	// looking at a stopped fundraiser deserves to know it was stopped and
	// broadly why. The free-text pause note is NOT included: it may name
	// individuals or describe an open investigation.
	PauseReasonCode *domain.PauseReason `json:"pauseReasonCode,omitempty"`

	// PlatformFeeBps is CivicOS's cut in basis points, disclosed publicly.
	// A donor is entitled to know what actually reaches the organization
	// BEFORE giving, not in a footnote afterwards.
	PlatformFeeBps int64 `json:"platformFeeBps"`
}

// PublicDetail adds the spend plan to the summary. Milestones are the heart
// of the Phase 2 exit criterion — a published campaign shows its goal and
// exactly how the money is meant to be spent, before it can take a naira.
type PublicDetail struct {
	PublicCampaign
	Description string            `json:"description"`
	Milestones  []PublicMilestone `json:"milestones"`
}

type PublicMilestone struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description *string                `json:"description,omitempty"`
	TargetMinor int64                  `json:"targetMinor"`
	Status      domain.MilestoneStatus `json:"status"`
	Position    int                    `json:"position"`
	CompletedAt *time.Time             `json:"completedAt,omitempty"`
}

func (s *Service) toPublic(c *domain.Campaign, orgName string) PublicCampaign {
	p := toPublic(c, orgName)
	p.PlatformFeeBps = s.platformFeeBps
	return p
}

func toPublic(c *domain.Campaign, orgName string) PublicCampaign {
	return PublicCampaign{
		ID:               c.ID,
		Slug:             c.Slug,
		Title:            c.Title,
		Summary:          c.Summary,
		Category:         c.Category,
		Status:           c.Status,
		Currency:         c.Currency,
		GoalMinor:        c.GoalMinor,
		RaisedMinor:      c.RaisedMinor,
		DonorCount:       c.DonorCount,
		CoverImageURL:    c.CoverImageURL,
		State:            c.State,
		LGA:              c.LGA,
		CommunityID:      c.CommunityID,
		OrganizationID:   c.OrganizationID,
		OrganizationName: orgName,
		IsEmergency:      c.IsEmergency,
		StartDate:        c.StartDate,
		EndDate:          c.EndDate,
		PublishedAt:      c.PublishedAt,
		CompletedAt:      c.CompletedAt,
		PauseReasonCode:  c.PauseReasonCode,
	}
}

func toPublicMilestones(ms []domain.Milestone) []PublicMilestone {
	// Never nil — an empty spend plan should serialise as [] so clients can
	// map over it without a guard.
	out := make([]PublicMilestone, 0, len(ms))
	for _, m := range ms {
		out = append(out, PublicMilestone{
			ID:          m.ID,
			Title:       m.Title,
			Description: m.Description,
			TargetMinor: m.TargetMinor,
			Status:      m.Status,
			Position:    m.Position,
			CompletedAt: m.CompletedAt,
		})
	}
	return out
}

// ListPublic returns campaigns a citizen may see. The status allow-list is
// applied in the repository (ListFilters.PublicOnly) rather than here, so
// there is exactly one place that decides what "public" means.
func (s *Service) ListPublic(f ListFilters) ([]PublicCampaign, error) {
	f.PublicOnly = true
	items, err := s.repo.Find(f)
	if err != nil {
		return nil, err
	}
	names, err := s.repo.OrgNames(orgIDsOf(items))
	if err != nil {
		return nil, err
	}
	out := make([]PublicCampaign, 0, len(items))
	for i := range items {
		out = append(out, s.toPublic(&items[i], names[items[i].OrganizationID]))
	}
	return out, nil
}

// GetPublicBySlug resolves the public campaign page. Slug rather than UUID
// because this is the citizen-facing URL.
//
// A campaign not in a public status returns NOT_FOUND, deliberately — not
// FORBIDDEN. A 403 would confirm that a campaign with that slug exists and
// is merely hidden, which leaks the existence of drafts and rejected
// applications to anyone willing to guess.
func (s *Service) GetPublicBySlug(slug string) (*PublicDetail, error) {
	c, err := s.GetBySlug(slug)
	if err != nil {
		return nil, err
	}
	if !isPublicStatus(c.Status) {
		return nil, notFound()
	}
	ms, err := s.repo.MilestonesFor(c.ID)
	if err != nil {
		return nil, err
	}
	names, err := s.repo.OrgNames([]string{c.OrganizationID})
	if err != nil {
		return nil, err
	}
	return &PublicDetail{
		PublicCampaign: s.toPublic(c, names[c.OrganizationID]),
		Description:    c.Description,
		Milestones:     toPublicMilestones(ms),
	}, nil
}

func isPublicStatus(s domain.CampaignStatus) bool {
	for _, ok := range publicStatuses {
		if s == ok {
			return true
		}
	}
	return false
}

func orgIDsOf(items []domain.Campaign) []string {
	seen := map[string]bool{}
	var ids []string
	for _, c := range items {
		if !seen[c.OrganizationID] {
			seen[c.OrganizationID] = true
			ids = append(ids, c.OrganizationID)
		}
	}
	return ids
}
