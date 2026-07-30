package milestones

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/civicos/organization-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Store interface {
	FindByCampaign(campaignID string) ([]domain.Milestone, error)
	FindByID(id string) (*domain.Milestone, error)
	Create(m *domain.Milestone) error
	Update(id string, updates map[string]any) error
	Delete(id string) error
	NextPosition(campaignID string) (int, error)
	SumTargetsExcluding(campaignID, excludeID string) (int64, error)
	Campaign(campaignID string) (*domain.Campaign, error)
}

type Service struct {
	repo Store
}

func NewService(repo Store) *Service {
	return &Service{repo: repo}
}

type CreateInput struct {
	Title       string  `json:"title" binding:"required,min=3,max=160"`
	Description *string `json:"description"`
	TargetMinor int64   `json:"targetMinor" binding:"required"`
	Position    *int    `json:"position"`
}

type UpdateInput struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	TargetMinor *int64  `json:"targetMinor"`
	Position    *int    `json:"position"`
	Status      *string `json:"status"`
}

// editableStatuses are the campaign statuses in which the spend plan may
// still change. Once a campaign is in review or live, its milestones are
// part of what donors and reviewers were shown.
//
// The one exception is marking a milestone COMPLETED, which is progress
// reporting rather than editing the plan — see SetStatus.
func editable(s domain.CampaignStatus) bool {
	return s == domain.CampaignDraft || s == domain.CampaignNeedsChanges
}

func (s *Service) List(campaignID string) ([]domain.Milestone, error) {
	if _, err := s.campaign(campaignID); err != nil {
		return nil, err
	}
	return s.repo.FindByCampaign(campaignID)
}

// Campaign is exported so the handler can authorize against the owning
// organization without a second round-trip.
func (s *Service) Campaign(campaignID string) (*domain.Campaign, error) {
	return s.campaign(campaignID)
}

func (s *Service) campaign(campaignID string) (*domain.Campaign, error) {
	c, err := s.repo.Campaign(campaignID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &AppError{Code: "CAMPAIGN_NOT_FOUND", Message: "Campaign not found", Status: http.StatusNotFound}
	}
	return c, err
}

func (s *Service) Get(id string) (*domain.Milestone, error) {
	m, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &AppError{Code: "MILESTONE_NOT_FOUND", Message: "Milestone not found", Status: http.StatusNotFound}
	}
	return m, err
}

func (s *Service) Create(campaignID string, input CreateInput) (*domain.Milestone, error) {
	c, err := s.campaign(campaignID)
	if err != nil {
		return nil, err
	}
	if !editable(c.Status) {
		return nil, notEditable()
	}
	if input.TargetMinor <= 0 {
		return nil, &AppError{Code: "INVALID_TARGET", Message: "Milestone target must be greater than zero", Status: http.StatusBadRequest}
	}

	// The plan may not promise to spend more than the campaign is asking
	// for. Checked here as well as at submit time so the org gets the error
	// while adding the milestone, not minutes later at the review gate.
	existing, err := s.repo.SumTargetsExcluding(campaignID, "")
	if err != nil {
		return nil, err
	}
	if existing+input.TargetMinor > c.GoalMinor {
		return nil, exceedsGoal()
	}

	position := 0
	if input.Position != nil {
		position = *input.Position
	} else {
		position, err = s.repo.NextPosition(campaignID)
		if err != nil {
			return nil, err
		}
	}

	m := &domain.Milestone{
		ID:          uuid.New().String(),
		CampaignID:  campaignID,
		Title:       input.Title,
		Description: input.Description,
		TargetMinor: input.TargetMinor,
		Status:      domain.MilestonePlanned,
		Position:    position,
	}
	if err := s.repo.Create(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *Service) Update(id string, input UpdateInput) (*domain.Milestone, error) {
	m, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	c, err := s.campaign(m.CampaignID)
	if err != nil {
		return nil, err
	}

	updates := map[string]any{}

	// Status is progress reporting, allowed on a live campaign. Everything
	// else is editing the plan and is gated on the campaign being editable.
	if input.Status != nil {
		st, err := parseStatus(*input.Status)
		if err != nil {
			return nil, err
		}
		updates["status"] = st
		if st == domain.MilestoneCompleted {
			updates["completed_at"] = time.Now().UTC()
		} else {
			updates["completed_at"] = nil
		}
	}

	planChange := input.Title != nil || input.Description != nil ||
		input.TargetMinor != nil || input.Position != nil
	if planChange {
		if !editable(c.Status) {
			return nil, notEditable()
		}
		if input.Title != nil {
			updates["title"] = *input.Title
		}
		if input.Description != nil {
			updates["description"] = *input.Description
		}
		if input.Position != nil {
			updates["position"] = *input.Position
		}
		if input.TargetMinor != nil {
			if *input.TargetMinor <= 0 {
				return nil, &AppError{Code: "INVALID_TARGET", Message: "Milestone target must be greater than zero", Status: http.StatusBadRequest}
			}
			others, err := s.repo.SumTargetsExcluding(m.CampaignID, m.ID)
			if err != nil {
				return nil, err
			}
			if others+*input.TargetMinor > c.GoalMinor {
				return nil, exceedsGoal()
			}
			updates["target_minor"] = *input.TargetMinor
		}
	}

	if len(updates) == 0 {
		return m, nil
	}
	if err := s.repo.Update(id, updates); err != nil {
		return nil, err
	}
	return s.Get(id)
}

// SetStatus is the progress-reporting path, usable while a campaign is
// live. Kept separate from Update so a handler can expose "mark this
// milestone done" without also exposing plan edits.
func (s *Service) SetStatus(id, status string) (*domain.Milestone, error) {
	st, err := parseStatus(status)
	if err != nil {
		return nil, err
	}
	if _, err := s.Get(id); err != nil {
		return nil, err
	}
	updates := map[string]any{"status": st}
	if st == domain.MilestoneCompleted {
		updates["completed_at"] = time.Now().UTC()
	} else {
		updates["completed_at"] = nil
	}
	if err := s.repo.Update(id, updates); err != nil {
		return nil, err
	}
	return s.Get(id)
}

func (s *Service) Delete(id string) error {
	m, err := s.Get(id)
	if err != nil {
		return err
	}
	c, err := s.campaign(m.CampaignID)
	if err != nil {
		return err
	}
	if !editable(c.Status) {
		return notEditable()
	}
	return s.repo.Delete(id)
}

func parseStatus(v string) (domain.MilestoneStatus, *AppError) {
	switch domain.MilestoneStatus(strings.ToUpper(strings.TrimSpace(v))) {
	case domain.MilestonePlanned:
		return domain.MilestonePlanned, nil
	case domain.MilestoneInProgress:
		return domain.MilestoneInProgress, nil
	case domain.MilestoneCompleted:
		return domain.MilestoneCompleted, nil
	}
	return "", &AppError{Code: "INVALID_STATUS", Message: "Unknown milestone status", Status: http.StatusBadRequest}
}

func notEditable() *AppError {
	return &AppError{
		Code:    "CAMPAIGN_NOT_EDITABLE",
		Message: "The spend plan can only change while the campaign is a draft or has been sent back for changes",
		Status:  http.StatusConflict,
	}
}

func exceedsGoal() *AppError {
	return &AppError{
		Code:    "MILESTONES_EXCEED_GOAL",
		Message: "Milestone targets would add up to more than the funding goal",
		Status:  http.StatusBadRequest,
	}
}

type AppError struct {
	Code    string
	Message string
	Status  int
}

func (e *AppError) Error() string { return e.Message }
