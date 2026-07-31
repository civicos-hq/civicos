package progress

import (
	"errors"
	"net/http"

	"github.com/civicos/organization-service/internal/domain"
	"github.com/civicos/organization-service/internal/organizations"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Store interface {
	Find(f ListFilters) ([]domain.ProgressUpdate, error)
	FindByID(id string) (*domain.ProgressUpdate, error)
	Create(p *domain.ProgressUpdate) error
	Delete(id string) error
}

// CampaignOwner resolves which organization a campaign belongs to.
type CampaignOwner interface {
	Get(campaignID string) (*domain.Campaign, error)
}

type Service struct {
	repo      Store
	orgs      *organizations.Service
	campaigns CampaignOwner
}

// WithCampaigns enables campaign-scoped updates. Without it, campaignId is
// refused outright rather than accepted unchecked — an unverifiable target
// is worse than an unsupported one.
func (s *Service) WithCampaigns(c CampaignOwner) *Service {
	s.campaigns = c
	return s
}

func NewService(repo Store, orgs *organizations.Service) *Service {
	return &Service{repo: repo, orgs: orgs}
}

type CreateInput struct {
	IssueID     *string  `json:"issueId"`
	ProjectID   *string  `json:"projectId"`
	CampaignID  *string  `json:"campaignId"`
	Title       *string  `json:"title" binding:"omitempty,max=200"`
	Body        string   `json:"body" binding:"required,min=2"`
	Attachments []string `json:"attachmentUrls" binding:"omitempty,max=12,dive,url"`
	IsPublic    *bool    `json:"isPublic"`
}

func (s *Service) List(f ListFilters) ([]domain.ProgressUpdate, error) {
	return s.repo.Find(f)
}

func (s *Service) Get(id string) (*domain.ProgressUpdate, error) {
	p, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &AppError{Code: "UPDATE_NOT_FOUND", Message: "Progress update not found", Status: http.StatusNotFound}
	}
	return p, err
}

func (s *Service) Create(orgID string, input CreateInput, authorID, authorName string) (*domain.ProgressUpdate, error) {
	// Exactly one target must be set — an update always hangs off an
	// assigned report, a project it is reporting on, or (Phase 4) the
	// campaign it is accounting for. An update belonging to two things at
	// once would appear on both feeds saying different things to different
	// audiences.
	targets := 0
	for _, t := range []*string{input.IssueID, input.ProjectID, input.CampaignID} {
		if t != nil {
			targets++
		}
	}
	if targets != 1 {
		return nil, &AppError{
			Code:    "INVALID_TARGET",
			Message: "Set exactly one of issueId, projectId or campaignId",
			Status:  http.StatusBadRequest,
		}
	}
	// A campaign target must belong to the organization being posted as.
	//
	// Authorisation upstream checks the caller administers orgID from the
	// URL; campaignId arrives in the BODY. Without this check an admin of
	// one organization could publish updates onto another organization's
	// campaign page, under that campaign's name, to that campaign's donors.
	if input.CampaignID != nil {
		if s.campaigns == nil {
			return nil, &AppError{Code: "INVALID_TARGET", Message: "Campaign updates are not available", Status: http.StatusBadRequest}
		}
		camp, err := s.campaigns.Get(*input.CampaignID)
		if err != nil || camp == nil {
			return nil, &AppError{Code: "CAMPAIGN_NOT_FOUND", Message: "Campaign not found", Status: http.StatusNotFound}
		}
		if camp.OrganizationID != orgID {
			// NOT_FOUND rather than FORBIDDEN: a 403 would confirm the
			// campaign exists, letting someone probe for ids.
			return nil, &AppError{Code: "CAMPAIGN_NOT_FOUND", Message: "Campaign not found", Status: http.StatusNotFound}
		}
	}

	public := true
	if input.IsPublic != nil {
		public = *input.IsPublic
	}
	p := &domain.ProgressUpdate{
		ID:             uuid.New().String(),
		OrganizationID: orgID,
		IssueID:        input.IssueID,
		ProjectID:      input.ProjectID,
		CampaignID:     input.CampaignID,
		Title:          input.Title,
		Body:           input.Body,
		// Never nil: an empty attachment list must serialise as [] so
		// clients can map over it without a guard.
		AttachmentURLs: append([]string{}, input.Attachments...),
		IsPublic:       public,
		AuthorID:       authorID,
		AuthorName:     authorName,
	}
	if err := s.repo.Create(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) Delete(id string) error {
	if _, err := s.Get(id); err != nil {
		return err
	}
	return s.repo.Delete(id)
}

type AppError struct {
	Code    string
	Message string
	Status  int
}

func (e *AppError) Error() string { return e.Message }
