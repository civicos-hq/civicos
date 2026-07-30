package campaigns

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/civicos/organization-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Store interface {
	Find(f ListFilters) ([]domain.Campaign, error)
	FindByID(id string) (*domain.Campaign, error)
	FindBySlug(slug string) (*domain.Campaign, error)
	Create(c *domain.Campaign) error
	Update(id string, updates map[string]any) error
	Delete(id string) error
	SlugExists(slug string) (bool, error)
	Org(orgID string) (*domain.Organization, error)
	OrgNames(ids []string) (map[string]string, error)
	MilestonesFor(campaignID string) ([]domain.Milestone, error)
	CountMilestones(campaignID string) (int64, error)
	SumMilestoneTargets(campaignID string) (int64, error)
}

type Service struct {
	repo Store
	// platformFeeBps is carried here purely so the public projection can
	// disclose the rate. The donations package owns the actual arithmetic.
	platformFeeBps int64
}

func NewService(repo Store) *Service {
	return &Service{repo: repo}
}

// WithPlatformFee sets the rate disclosed on public campaign pages. A donor
// is entitled to know what reaches the organization before giving.
func (s *Service) WithPlatformFee(bps int64) *Service {
	s.platformFeeBps = bps
	return s
}

// ─── Transition guard ───────────────────────────────────────────────────
//
// Who may move a campaign, and where to. Two separate questions, both
// answered by one table so they cannot drift apart:
//
//   - Is this transition legal at all?
//   - Is this actor allowed to make it?
//
// An illegal transition is a 409 with CAMPAIGN_INVALID_TRANSITION, never a
// 500 and never a silent no-op. A legal transition attempted by the wrong
// actor is a 403.

// Actor is who is asking. Not a user role — a capability. An org admin and
// a platform admin can both act on a campaign but may make different moves,
// and some moves belong to neither.
type Actor uint8

const (
	// ActorOrg is an OWNER or ADMIN of the owning organization.
	ActorOrg Actor = 1 << iota
	// ActorPlatform is a CivicOS platform admin (the spec's "CivicOS
	// Administrator" role).
	ActorPlatform
	// ActorSystem is the platform itself, acting on ledger truth rather
	// than on anyone's intent. PUBLISHED → FUNDED is the only such move:
	// "goal reached" is a fact derived from settled donations, so no human
	// may assert it. There is deliberately no HTTP route that presents
	// ActorSystem — Phase 3 calls the service method directly.
	ActorSystem
)

func (a Actor) String() string {
	switch a {
	case ActorOrg:
		return "organization admin"
	case ActorPlatform:
		return "platform admin"
	case ActorSystem:
		return "system"
	}
	return "unknown"
}

// transitions is the single source of truth for the campaign lifecycle.
// Absent key or absent target == illegal. Statuses with no outgoing
// transitions (REJECTED, ARCHIVED) are terminal by omission.
var transitions = map[domain.CampaignStatus]map[domain.CampaignStatus]Actor{
	domain.CampaignDraft: {
		domain.CampaignPendingReview: ActorOrg,
	},
	domain.CampaignPendingReview: {
		// Only a platform admin decides. The spec's "Trust First" principle
		// is meaningless if the applicant can approve itself.
		domain.CampaignApproved:     ActorPlatform,
		domain.CampaignNeedsChanges: ActorPlatform,
		domain.CampaignRejected:     ActorPlatform,
	},
	domain.CampaignNeedsChanges: {
		domain.CampaignPendingReview: ActorOrg,
	},
	domain.CampaignApproved: {
		// Approval is permission to publish, not publication. The org
		// chooses the moment — matters for an emergency campaign that wants
		// to launch alongside an announcement.
		domain.CampaignPublished: ActorOrg,
	},
	domain.CampaignPublished: {
		domain.CampaignFunded:    ActorSystem,
		domain.CampaignCompleted: ActorOrg,
		domain.CampaignPaused:    ActorPlatform,
	},
	domain.CampaignFunded: {
		domain.CampaignCompleted: ActorOrg,
		domain.CampaignPaused:    ActorPlatform,
	},
	domain.CampaignPaused: {
		// Resume returns to PUBLISHED even if the goal was already met;
		// the system will re-derive FUNDED from the ledger.
		domain.CampaignPublished: ActorPlatform,
		domain.CampaignRejected:  ActorPlatform,
	},
	domain.CampaignCompleted: {
		domain.CampaignReported: ActorOrg,
	},
	domain.CampaignReported: {
		domain.CampaignArchived: ActorPlatform,
	},
}

// CanTransition reports whether `actor` may move a campaign from → to.
// Exported so the lifecycle is testable without a DB or an HTTP request.
func CanTransition(from, to domain.CampaignStatus, actor Actor) error {
	targets, ok := transitions[from]
	if !ok {
		return &AppError{
			Code:    "CAMPAIGN_INVALID_TRANSITION",
			Message: fmt.Sprintf("A campaign in %s is final and cannot change status", from),
			Status:  http.StatusConflict,
		}
	}
	allowed, ok := targets[to]
	if !ok {
		return &AppError{
			Code:    "CAMPAIGN_INVALID_TRANSITION",
			Message: fmt.Sprintf("Cannot move a campaign from %s to %s", from, to),
			Status:  http.StatusConflict,
		}
	}
	if allowed&actor == 0 {
		return &AppError{
			Code:    "CAMPAIGN_TRANSITION_FORBIDDEN",
			Message: fmt.Sprintf("Moving a campaign from %s to %s is done by the %s", from, to, allowed),
			Status:  http.StatusForbidden,
		}
	}
	return nil
}

// ─── Validation ─────────────────────────────────────────────────────────

// maxGoalMinor caps a funding goal at 1e15 minor units. Not a judgement
// about ambition — headroom so that summing many campaigns' goals for
// platform analytics cannot overflow int64 (max ≈ 9.2e18). An
// implausible-but-legal goal is a fraud signal for CivicAI to score in
// Phase 6, not a validation error here.
const maxGoalMinor int64 = 1_000_000_000_000_000

// supportedCurrencies is an allow-list. The spec's worked example uses £
// and the platform is Nigeria-first, so both must work — but an arbitrary
// 3-letter string must not, or Phase 3 inherits campaigns denominated in
// currencies no payment provider can settle.
var supportedCurrencies = map[string]bool{
	"NGN": true,
	"GBP": true,
	"USD": true,
	"EUR": true,
}

func validCategory(c string) bool {
	switch domain.CampaignCategory(c) {
	case domain.CategoryEmergencyRelief, domain.CategoryCommunityDevelopment,
		domain.CategoryEducation, domain.CategoryHealthcare,
		domain.CategoryEnvironment, domain.CategoryAgriculture,
		domain.CategoryOther:
		return true
	}
	return false
}

var slugUnsafe = regexp.MustCompile(`[^a-z0-9]+`)

// slugify produces a URL-safe slug. Phase 1 has no public route, but the
// slug is minted at create time so the eventual public URL is stable from
// the campaign's first moment rather than assigned at publish.
func slugify(title string) string {
	s := slugUnsafe.ReplaceAllString(strings.ToLower(title), "-")
	s = strings.Trim(s, "-")
	if len(s) > 80 {
		s = strings.Trim(s[:80], "-")
	}
	if s == "" {
		s = "campaign"
	}
	return s
}

// uniqueSlug suffixes until free. Bounded so a pathological collision run
// cannot spin: after 25 tries fall back to a UUID fragment, which is ugly
// but always unique.
func (s *Service) uniqueSlug(base string) (string, error) {
	candidate := base
	for i := 2; i < 27; i++ {
		exists, err := s.repo.SlugExists(candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
	return fmt.Sprintf("%s-%s", base, uuid.NewString()[:8]), nil
}

// ─── Inputs ─────────────────────────────────────────────────────────────

type CreateInput struct {
	Title         string     `json:"title" binding:"required,min=4,max=160"`
	Summary       string     `json:"summary" binding:"required,min=10,max=300"`
	Description   string     `json:"description" binding:"required,min=40"`
	Category      string     `json:"category" binding:"required"`
	Currency      string     `json:"currency"`
	GoalMinor     int64      `json:"goalMinor" binding:"required"`
	CoverImageURL *string    `json:"coverImageUrl"`
	CommunityID   *string    `json:"communityId"`
	ProjectID     *string    `json:"projectId"`
	State         *string    `json:"state"`
	LGA           *string    `json:"lga"`
	StartDate     *time.Time `json:"startDate"`
	EndDate       *time.Time `json:"endDate"`
	IsEmergency   bool       `json:"isEmergency"`
}

// UpdateInput deliberately omits Currency and Status.
//
// Currency is immutable: re-denominating a campaign silently reinterprets
// every amount attached to it. Status is not editable as a field — it moves
// only through the guarded transition endpoints, so there is no path where
// a PATCH body can skip review.
type UpdateInput struct {
	Title         *string    `json:"title"`
	Summary       *string    `json:"summary"`
	Description   *string    `json:"description"`
	Category      *string    `json:"category"`
	GoalMinor     *int64     `json:"goalMinor"`
	CoverImageURL *string    `json:"coverImageUrl"`
	CommunityID   *string    `json:"communityId"`
	ProjectID     *string    `json:"projectId"`
	State         *string    `json:"state"`
	LGA           *string    `json:"lga"`
	StartDate     *time.Time `json:"startDate"`
	EndDate       *time.Time `json:"endDate"`
	IsEmergency   *bool      `json:"isEmergency"`
}

// ─── Reads ──────────────────────────────────────────────────────────────

func (s *Service) List(f ListFilters) ([]domain.Campaign, error) {
	return s.repo.Find(f)
}

func (s *Service) Get(id string) (*domain.Campaign, error) {
	c, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notFound()
	}
	return c, err
}

func (s *Service) GetBySlug(slug string) (*domain.Campaign, error) {
	c, err := s.repo.FindBySlug(slug)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notFound()
	}
	return c, err
}

// ─── Writes ─────────────────────────────────────────────────────────────

func (s *Service) Create(orgID string, input CreateInput, createdByID, createdByName string) (*domain.Campaign, error) {
	if !validCategory(input.Category) {
		return nil, &AppError{Code: "INVALID_CATEGORY", Message: "Unknown campaign category", Status: http.StatusBadRequest}
	}
	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	if currency == "" {
		currency = "NGN"
	}
	if !supportedCurrencies[currency] {
		return nil, &AppError{Code: "UNSUPPORTED_CURRENCY", Message: "Campaign currency is not supported", Status: http.StatusBadRequest}
	}
	if err := validateGoal(input.GoalMinor); err != nil {
		return nil, err
	}
	if err := validateDates(input.StartDate, input.EndDate); err != nil {
		return nil, err
	}

	slug, err := s.uniqueSlug(slugify(input.Title))
	if err != nil {
		return nil, err
	}

	c := &domain.Campaign{
		ID:             uuid.New().String(),
		OrganizationID: orgID,
		Title:          input.Title,
		Slug:           slug,
		Summary:        input.Summary,
		Description:    input.Description,
		Category:       domain.CampaignCategory(input.Category),
		Status:         domain.CampaignDraft,
		CoverImageURL:  input.CoverImageURL,
		Currency:       currency,
		GoalMinor:      input.GoalMinor,
		RaisedMinor:    0,
		DonorCount:     0,
		CommunityID:    input.CommunityID,
		ProjectID:      input.ProjectID,
		State:          input.State,
		LGA:            input.LGA,
		StartDate:      input.StartDate,
		EndDate:        input.EndDate,
		IsEmergency:    input.IsEmergency,
		ApprovalStatus: "NONE",
		CreatedByID:    createdByID,
		CreatedByName:  createdByName,
	}
	if err := s.repo.Create(c); err != nil {
		return nil, err
	}
	return c, nil
}

// Update edits campaign content. Only permitted while the campaign is
// still the org's to change — DRAFT or NEEDS_CHANGES. Once it is in review
// or published, content is frozen: a donor who reads a campaign and gives
// money must be giving to the thing they read.
func (s *Service) Update(id string, input UpdateInput) (*domain.Campaign, error) {
	c, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if c.Status != domain.CampaignDraft && c.Status != domain.CampaignNeedsChanges {
		return nil, &AppError{
			Code:    "CAMPAIGN_NOT_EDITABLE",
			Message: "Only draft campaigns or ones sent back for changes can be edited",
			Status:  http.StatusConflict,
		}
	}

	updates := map[string]any{}
	if input.Title != nil {
		updates["title"] = *input.Title
	}
	if input.Summary != nil {
		updates["summary"] = *input.Summary
	}
	if input.Description != nil {
		updates["description"] = *input.Description
	}
	if input.Category != nil {
		if !validCategory(*input.Category) {
			return nil, &AppError{Code: "INVALID_CATEGORY", Message: "Unknown campaign category", Status: http.StatusBadRequest}
		}
		updates["category"] = *input.Category
	}
	if input.GoalMinor != nil {
		if err := validateGoal(*input.GoalMinor); err != nil {
			return nil, err
		}
		updates["goal_minor"] = *input.GoalMinor
	}
	if input.CoverImageURL != nil {
		updates["cover_image_url"] = *input.CoverImageURL
	}
	if input.CommunityID != nil {
		updates["community_id"] = *input.CommunityID
	}
	if input.ProjectID != nil {
		updates["project_id"] = *input.ProjectID
	}
	if input.State != nil {
		updates["state"] = *input.State
	}
	if input.LGA != nil {
		updates["lga"] = *input.LGA
	}
	if input.StartDate != nil || input.EndDate != nil {
		start, end := c.StartDate, c.EndDate
		if input.StartDate != nil {
			start = input.StartDate
		}
		if input.EndDate != nil {
			end = input.EndDate
		}
		if err := validateDates(start, end); err != nil {
			return nil, err
		}
		if input.StartDate != nil {
			updates["start_date"] = *input.StartDate
		}
		if input.EndDate != nil {
			updates["end_date"] = *input.EndDate
		}
	}
	if input.IsEmergency != nil {
		updates["is_emergency"] = *input.IsEmergency
	}
	if len(updates) == 0 {
		return c, nil
	}
	if err := s.repo.Update(id, updates); err != nil {
		return nil, err
	}
	return s.Get(id)
}

// Submit moves DRAFT | NEEDS_CHANGES → PENDING_REVIEW.
//
// This is the gate the spec's "Trust First" principle lives behind, so the
// checks are server-side and not negotiable from the client:
//
//   - the owning organization must be verified
//   - there must be at least one milestone (a spend plan)
//   - milestone targets must not exceed the goal
func (s *Service) Submit(id string) (*domain.Campaign, error) {
	c, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if err := CanTransition(c.Status, domain.CampaignPendingReview, ActorOrg); err != nil {
		return nil, err
	}

	org, err := s.repo.Org(c.OrganizationID)
	if err != nil {
		return nil, err
	}
	// The error names exactly what is missing. "Your organization is not
	// eligible" with no detail leaves an org with no way to fix it.
	if ok, missing := org.FundingEligible(); !ok {
		return nil, &AppError{
			Code:    "ORG_NOT_FUNDING_ELIGIBLE",
			Message: "Your organization cannot raise funds yet — still needed: " + strings.Join(missing, ", "),
			Status:  http.StatusConflict,
		}
	}

	count, err := s.repo.CountMilestones(id)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, &AppError{
			Code:    "NO_MILESTONES",
			Message: "Add at least one milestone describing how the funds will be spent",
			Status:  http.StatusBadRequest,
		}
	}

	total, err := s.repo.SumMilestoneTargets(id)
	if err != nil {
		return nil, err
	}
	if total > c.GoalMinor {
		return nil, &AppError{
			Code:    "MILESTONES_EXCEED_GOAL",
			Message: "Milestone targets add up to more than the funding goal",
			Status:  http.StatusBadRequest,
		}
	}

	now := time.Now().UTC()
	if err := s.repo.Update(id, map[string]any{
		"status":          domain.CampaignPendingReview,
		"approval_status": "PENDING",
		"submitted_at":    now,
		// Clear any previous reviewer verdict so the queue shows this as a
		// fresh decision rather than carrying a stale "needs changes" note.
		"review_note":    nil,
		"reviewed_by_id": nil,
		"reviewed_at":    nil,
	}); err != nil {
		return nil, err
	}
	return s.Get(id)
}

// Review is the platform-admin decision on a PENDING_REVIEW campaign.
// `decision` is one of APPROVED, NEEDS_CHANGES, REJECTED. A note is
// required for anything other than approval — telling an organization its
// flood-relief campaign is rejected without saying why is not a usable
// product.
func (s *Service) Review(id, decision string, note *string, reviewerID string) (*domain.Campaign, error) {
	c, err := s.Get(id)
	if err != nil {
		return nil, err
	}

	var target domain.CampaignStatus
	switch strings.ToUpper(decision) {
	case "APPROVED":
		target = domain.CampaignApproved
	case "NEEDS_CHANGES":
		target = domain.CampaignNeedsChanges
	case "REJECTED":
		target = domain.CampaignRejected
	default:
		return nil, &AppError{
			Code:    "INVALID_DECISION",
			Message: "Decision must be APPROVED, NEEDS_CHANGES or REJECTED",
			Status:  http.StatusBadRequest,
		}
	}
	if err := CanTransition(c.Status, target, ActorPlatform); err != nil {
		return nil, err
	}
	if target != domain.CampaignApproved && (note == nil || strings.TrimSpace(*note) == "") {
		return nil, &AppError{
			Code:    "REVIEW_NOTE_REQUIRED",
			Message: "Explain what needs to change or why the campaign was rejected",
			Status:  http.StatusBadRequest,
		}
	}

	now := time.Now().UTC()
	updates := map[string]any{
		"status":          target,
		"approval_status": string(target),
		"reviewed_by_id":  reviewerID,
		"reviewed_at":     now,
	}
	if note != nil {
		updates["review_note"] = strings.TrimSpace(*note)
	}
	if err := s.repo.Update(id, updates); err != nil {
		return nil, err
	}
	return s.Get(id)
}

// Publish moves APPROVED → PUBLISHED. From here the campaign is publicly
// visible; in Phase 3 it also becomes able to accept donations.
func (s *Service) Publish(id string) (*domain.Campaign, error) {
	c, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if err := CanTransition(c.Status, domain.CampaignPublished, ActorOrg); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if err := s.repo.Update(id, map[string]any{
		"status":       domain.CampaignPublished,
		"published_at": now,
	}); err != nil {
		return nil, err
	}
	return s.Get(id)
}

// Pause suspends a live campaign. The reason code is one of the spec's five
// Governance triggers (or OTHER); the note is required and carries the
// specifics.
//
// A code AND a note, not one or the other: the code makes pauses countable
// and filterable ("how often do we pause for fraud?"), while the note is
// what the organization is actually owed as an explanation.
func (s *Service) Pause(id, reasonCode, note string) (*domain.Campaign, error) {
	c, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if err := CanTransition(c.Status, domain.CampaignPaused, ActorPlatform); err != nil {
		return nil, err
	}
	code := strings.ToUpper(strings.TrimSpace(reasonCode))
	if !domain.ValidPauseReason(code) {
		return nil, &AppError{
			Code:    "INVALID_PAUSE_REASON",
			Message: "Pause reason must be one of FRAUD_DETECTED, VERIFICATION_EXPIRED, MISUSE_REPORTED, ORGANIZATION_SUSPENDED, FALSE_INFORMATION, OTHER",
			Status:  http.StatusBadRequest,
		}
	}
	if strings.TrimSpace(note) == "" {
		return nil, &AppError{
			Code:    "PAUSE_NOTE_REQUIRED",
			Message: "Record what specifically prompted this pause",
			Status:  http.StatusBadRequest,
		}
	}
	if err := s.repo.Update(id, map[string]any{
		"status":            domain.CampaignPaused,
		"pause_reason_code": domain.PauseReason(code),
		"pause_note":        strings.TrimSpace(note),
	}); err != nil {
		return nil, err
	}
	return s.Get(id)
}

// Resume returns a paused campaign to PUBLISHED. Phase 3 re-derives
// FUNDED from the ledger, so a campaign whose goal was met while paused
// does not need a special case here.
func (s *Service) Resume(id string) (*domain.Campaign, error) {
	c, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if err := CanTransition(c.Status, domain.CampaignPublished, ActorPlatform); err != nil {
		return nil, err
	}
	if err := s.repo.Update(id, map[string]any{
		"status":            domain.CampaignPublished,
		"pause_reason_code": nil,
		"pause_note":        nil,
	}); err != nil {
		return nil, err
	}
	return s.Get(id)
}

// Transition is the generic mover for the remaining org/platform steps
// (COMPLETED, REPORTED, ARCHIVED, and rejecting a paused campaign). The
// guard table is consulted exactly as it is for the named methods above —
// this is a thinner API over the same rules, not a way around them.
func (s *Service) Transition(id string, to domain.CampaignStatus, actor Actor) (*domain.Campaign, error) {
	c, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if err := CanTransition(c.Status, to, actor); err != nil {
		return nil, err
	}
	updates := map[string]any{"status": to}
	if to == domain.CampaignCompleted {
		updates["completed_at"] = time.Now().UTC()
	}
	if err := s.repo.Update(id, updates); err != nil {
		return nil, err
	}
	return s.Get(id)
}

// Delete removes a campaign that never left DRAFT. Anything that has been
// through review is part of the public record of what this organization
// asked for, and is archived rather than deleted.
func (s *Service) Delete(id string) error {
	c, err := s.Get(id)
	if err != nil {
		return err
	}
	if c.Status != domain.CampaignDraft {
		return &AppError{
			Code:    "CAMPAIGN_NOT_DELETABLE",
			Message: "Only draft campaigns can be deleted; submitted campaigns are archived instead",
			Status:  http.StatusConflict,
		}
	}
	return s.repo.Delete(id)
}

// ─── Helpers ────────────────────────────────────────────────────────────

func validateGoal(goal int64) *AppError {
	if goal <= 0 {
		return &AppError{Code: "INVALID_GOAL", Message: "Funding goal must be greater than zero", Status: http.StatusBadRequest}
	}
	if goal > maxGoalMinor {
		return &AppError{Code: "INVALID_GOAL", Message: "Funding goal is implausibly large", Status: http.StatusBadRequest}
	}
	return nil
}

func validateDates(start, end *time.Time) *AppError {
	if start != nil && end != nil && !end.After(*start) {
		return &AppError{Code: "INVALID_DATES", Message: "End date must be after the start date", Status: http.StatusBadRequest}
	}
	return nil
}

func notFound() *AppError {
	return &AppError{Code: "CAMPAIGN_NOT_FOUND", Message: "Campaign not found", Status: http.StatusNotFound}
}

type AppError struct {
	Code    string
	Message string
	Status  int
}

func (e *AppError) Error() string { return e.Message }
