package flags

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/civicos/identity-service/internal/audit"
	"github.com/civicos/identity-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Store interface {
	Find(f ListFilters) ([]domain.ContentFlag, error)
	FindByID(id string) (*domain.ContentFlag, error)
	Create(f *domain.ContentFlag) error
	Update(id string, updates map[string]any) error
	CountByStatus() (map[string]int64, error)
}

type Service struct {
	repo    Store
	auditor *audit.Auditor
}

func NewService(repo Store, auditor *audit.Auditor) *Service {
	return &Service{repo: repo, auditor: auditor}
}

type CreateInput struct {
	ContentType string  `json:"contentType" binding:"required"`
	ContentID   string  `json:"contentId" binding:"required,uuid"`
	Reason      string  `json:"reason" binding:"required"`
	Description *string `json:"description"`
}

type ResolveInput struct {
	Status         string  `json:"status" binding:"required"`
	ResolutionNote *string `json:"resolutionNote"`
}

func (s *Service) List(f ListFilters) ([]domain.ContentFlag, error) {
	return s.repo.Find(f)
}

func (s *Service) Counts() (map[string]int64, error) {
	return s.repo.CountByStatus()
}

func (s *Service) Get(id string) (*domain.ContentFlag, error) {
	f, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &AppError{Code: "FLAG_NOT_FOUND", Message: "Flag not found", Status: http.StatusNotFound}
	}
	return f, err
}

func (s *Service) Create(input CreateInput, reporterID, reporterName string) (*domain.ContentFlag, error) {
	ct := strings.ToUpper(input.ContentType)
	if !validFlaggable(ct) {
		return nil, &AppError{Code: "INVALID_CONTENT_TYPE", Message: "Unknown content type", Status: http.StatusBadRequest}
	}
	reason := strings.ToUpper(input.Reason)
	if !validReason(reason) {
		return nil, &AppError{Code: "INVALID_REASON", Message: "Unknown flag reason", Status: http.StatusBadRequest}
	}
	f := &domain.ContentFlag{
		ID:           uuid.New().String(),
		ContentType:  domain.FlaggableType(ct),
		ContentID:    input.ContentID,
		ReporterID:   reporterID,
		ReporterName: reporterName,
		Reason:       domain.FlagReason(reason),
		Description:  input.Description,
		Status:       domain.FlagStatusPending,
	}
	if err := s.repo.Create(f); err != nil {
		// GORM returns a unique-index violation when a user tries to flag the
		// same content twice. Map that to a friendly conflict rather than a
		// 500 so the client can show "you've already reported this".
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "idx_flag_dedup") {
			return nil, &AppError{Code: "ALREADY_FLAGGED", Message: "You've already flagged this content", Status: http.StatusConflict}
		}
		return nil, err
	}
	return f, nil
}

// Resolve moves a flag out of PENDING into HIDDEN or DISMISSED. Every
// resolution writes an AuditLog row so the moderator's decision is
// reviewable — the "Trust Before Automation" bar demands it. The audit
// write happens in the handler so this service stays request-agnostic.
func (s *Service) Resolve(id string, input ResolveInput, actor audit.Actor) (*domain.ContentFlag, error) {
	status := strings.ToUpper(input.Status)
	if status != string(domain.FlagStatusHidden) &&
		status != string(domain.FlagStatusDismissed) &&
		status != string(domain.FlagStatusReviewed) {
		return nil, &AppError{Code: "INVALID_STATUS", Message: "Status must be HIDDEN, DISMISSED, or REVIEWED", Status: http.StatusBadRequest}
	}
	existing, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if existing.Status != domain.FlagStatusPending {
		return nil, &AppError{Code: "ALREADY_RESOLVED", Message: "Flag has already been resolved", Status: http.StatusConflict}
	}
	now := time.Now().UTC()
	updates := map[string]any{
		"status":           status,
		"resolved_by_id":   actor.ID,
		"resolved_by_name": actor.Name,
		"resolved_at":      now,
	}
	if input.ResolutionNote != nil {
		updates["resolution_note"] = *input.ResolutionNote
	}
	if err := s.repo.Update(id, updates); err != nil {
		return nil, err
	}
	return s.Get(id)
}

func validReason(r string) bool {
	switch domain.FlagReason(r) {
	case domain.FlagReasonSpam, domain.FlagReasonAbuse, domain.FlagReasonMisinfo,
		domain.FlagReasonHate, domain.FlagReasonOther:
		return true
	}
	return false
}

func validFlaggable(t string) bool {
	switch domain.FlaggableType(t) {
	case domain.FlaggableIssue, domain.FlaggableIssueComment, domain.FlaggablePetition,
		domain.FlaggablePetitionComment, domain.FlaggableRepComment,
		domain.FlaggableAnnouncement, domain.FlaggableProgressUpdate:
		return true
	}
	return false
}

type AppError struct {
	Code    string
	Message string
	Status  int
}

func (e *AppError) Error() string { return e.Message }
