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
	// gate is nil on deployments without campaigns wired up; a nil gate
	// refuses CAMPAIGN flags rather than waving them through.
	gate *CampaignGate
}

func NewService(repo Store, auditor *audit.Auditor, gate *CampaignGate) *Service {
	return &Service{repo: repo, auditor: auditor, gate: gate}
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

// DirectHideInput is the admin's proactive-moderation shortcut. The
// admin has already decided the content should be hidden; this creates
// the flag and immediately marks it HIDDEN in one call.
type DirectHideInput struct {
	ContentType    string  `json:"contentType" binding:"required"`
	ContentID      string  `json:"contentId" binding:"required,uuid"`
	Reason         string  `json:"reason" binding:"required"`
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
	if !validReasonFor(ct, reason) {
		return nil, &AppError{Code: "INVALID_REASON", Message: "Unknown flag reason for this content type", Status: http.StatusBadRequest}
	}

	if domain.FlaggableType(ct) == domain.FlaggableCampaign {
		// A concern about money must come with an account of what was seen.
		// A reason code alone tells a moderator nothing they can act on, and
		// the effort is itself a filter against drive-by reporting.
		if input.Description == nil || len(strings.TrimSpace(*input.Description)) < 20 {
			return nil, &AppError{
				Code:    "DESCRIPTION_REQUIRED",
				Message: "Describe what you have seen, in at least a sentence.",
				Status:  http.StatusBadRequest,
			}
		}
		if s.gate == nil {
			return nil, &AppError{Code: "INVALID_CONTENT_TYPE", Message: "Campaign reporting is not enabled", Status: http.StatusBadRequest}
		}
		if err := s.gate.Check(input.ContentID, reporterID); err != nil {
			return nil, err
		}
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
	if status == string(domain.FlagStatusHidden) && strings.TrimSpace(derefString(input.ResolutionNote)) == "" {
		return nil, &AppError{Code: "RESOLUTION_NOTE_REQUIRED", Message: "A resolution note is required when hiding content", Status: http.StatusBadRequest}
	}
	existing, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if existing.Status != domain.FlagStatusPending {
		return nil, &AppError{Code: "ALREADY_RESOLVED", Message: "Flag has already been resolved", Status: http.StatusConflict}
	}
	// A campaign concern can never be resolved by hiding something.
	//
	// HIDDEN is a content action: it takes a comment off a page. The
	// equivalent action for a campaign is to PAUSE it, which stops money
	// moving — and that must stay a separate, deliberate decision taken on
	// the campaign itself, with its own reason code and audit trail. If
	// clearing the queue could pause a fundraiser, a coordinated set of
	// reports would become a way to shut down a rival, which is exactly the
	// lever this design refuses to build.
	if existing.ContentType == domain.FlaggableCampaign && status == string(domain.FlagStatusHidden) {
		return nil, &AppError{
			Code: "USE_PAUSE_INSTEAD",
			Message: "A campaign concern cannot be resolved by hiding it. Mark it reviewed or dismissed, " +
				"and pause the campaign separately if that is warranted.",
			Status: http.StatusBadRequest,
		}
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

// DirectHide creates a flag and immediately resolves it as HIDDEN — the
// admin's one-shot proactive-moderation path. Actor is both reporter
// and resolver. Returns the created row so the handler can emit an
// audit entry with all the right metadata (see handler.go).
func (s *Service) DirectHide(input DirectHideInput, actor domain.User) (*domain.ContentFlag, error) {
	ct := strings.ToUpper(input.ContentType)
	if !validFlaggable(ct) {
		return nil, &AppError{Code: "INVALID_CONTENT_TYPE", Message: "Unknown content type", Status: http.StatusBadRequest}
	}
	// Same rule as Resolve, for the same reason: there is no such thing as
	// hiding a campaign. Pausing it is the real action and it lives on the
	// campaign, not in the moderation queue.
	if domain.FlaggableType(ct) == domain.FlaggableCampaign {
		return nil, &AppError{
			Code:    "USE_PAUSE_INSTEAD",
			Message: "Campaigns cannot be hidden. Pause the campaign instead.",
			Status:  http.StatusBadRequest,
		}
	}
	if strings.TrimSpace(derefString(input.ResolutionNote)) == "" {
		return nil, &AppError{Code: "RESOLUTION_NOTE_REQUIRED", Message: "A resolution note is required when hiding content", Status: http.StatusBadRequest}
	}
	reason := strings.ToUpper(input.Reason)
	if !validReasonFor(ct, reason) {
		return nil, &AppError{Code: "INVALID_REASON", Message: "Unknown flag reason", Status: http.StatusBadRequest}
	}
	now := time.Now().UTC()
	f := &domain.ContentFlag{
		ID:             uuid.New().String(),
		ContentType:    domain.FlaggableType(ct),
		ContentID:      input.ContentID,
		ReporterID:     actor.ID,
		ReporterName:   actor.Name,
		Reason:         domain.FlagReason(reason),
		Status:         domain.FlagStatusHidden,
		ResolvedByID:   &actor.ID,
		ResolvedByName: &actor.Name,
		ResolutionNote: input.ResolutionNote,
		ResolvedAt:     &now,
	}
	if err := s.repo.Create(f); err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "idx_flag_dedup") {
			return nil, &AppError{Code: "ALREADY_FLAGGED", Message: "This content is already flagged by this actor", Status: http.StatusConflict}
		}
		return nil, err
	}
	return f, nil
}

// validReasonFor keeps the two reason vocabularies apart. A CAMPAIGN takes
// only funding reasons; everything else takes only moderation reasons.
// OTHER is the one both share.
func validReasonFor(contentType, r string) bool {
	if domain.FlaggableType(contentType) == domain.FlaggableCampaign {
		for _, ok := range domain.FundingReasons() {
			if domain.FlagReason(r) == ok {
				return true
			}
		}
		return false
	}
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
		domain.FlaggableAnnouncement, domain.FlaggableProgressUpdate,
		domain.FlaggableCampaign:
		return true
	}
	return false
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

type AppError struct {
	Code    string
	Message string
	Status  int
}

func (e *AppError) Error() string { return e.Message }
