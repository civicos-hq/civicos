package consultations

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/civicos/organization-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Store is the interface the service consumes. Keeping it narrow lets
// tests substitute a fake without pulling in GORM.
type Store interface {
	FindAll(f ListFilters) ([]domain.Consultation, error)
	FindByID(id string) (*domain.Consultation, error)
	Create(c *domain.Consultation) error
	Update(id string, updates map[string]any) error
	Delete(id string) error
	BumpResponseCount(id string, delta int) error

	FindQuestions(consultationID string) ([]domain.ConsultationQuestion, error)
	FindQuestionByID(id string) (*domain.ConsultationQuestion, error)
	CreateQuestion(q *domain.ConsultationQuestion) error
	UpdateQuestion(id string, updates map[string]any) error
	DeleteQuestion(id string) error
	NextQuestionPosition(consultationID string) (int, error)
	ReorderQuestions(consultationID string, ordering map[string]int) error

	HasResponse(consultationID, userID string) (bool, error)
	CreateResponse(resp *domain.ConsultationResponse, answers []domain.ConsultationAnswer) error
	FindResponses(consultationID string, limit, offset int) ([]domain.ConsultationResponse, int64, error)
	FindAnswers(responseIDs []string) ([]domain.ConsultationAnswer, error)
	FindMyResponseIDs(userID string) ([]string, error)
	AllAnswersForConsultation(consultationID string) ([]domain.ConsultationAnswer, error)

	FindOutcome(consultationID string) (*domain.ConsultationOutcome, error)
	CreateOrUpdateOutcome(o *domain.ConsultationOutcome) error
}

type Service struct{ repo Store }

func NewService(repo Store) *Service { return &Service{repo: repo} }

// AppError is the service-level error type. Handlers convert it to the
// {code, message, status} response envelope.
type AppError struct {
	Code    string
	Message string
	Status  int
}

func (e *AppError) Error() string { return e.Message }

// ── Input DTOs ───────────────────────────────────────────────────

type CreateInput struct {
	Title         string  `json:"title" binding:"required,min=5,max=200"`
	Summary       string  `json:"summary" binding:"required,min=10,max=500"`
	Description   string  `json:"description" binding:"required,min=10"`
	CoverImageURL *string `json:"coverImageUrl"`
	// CommunityID is nullable — nil = whole-org audience.
	CommunityID *string    `json:"communityId" binding:"omitempty,uuid4"`
	OpensAt     *time.Time `json:"opensAt"`
	ClosesAt    *time.Time `json:"closesAt"`
}

type UpdateInput struct {
	Title         *string    `json:"title" binding:"omitempty,min=5,max=200"`
	Summary       *string    `json:"summary" binding:"omitempty,min=10,max=500"`
	Description   *string    `json:"description" binding:"omitempty,min=10"`
	CoverImageURL *string    `json:"coverImageUrl"`
	CommunityID   *string    `json:"communityId" binding:"omitempty,uuid4"`
	OpensAt       *time.Time `json:"opensAt"`
	ClosesAt      *time.Time `json:"closesAt"`
}

type QuestionInput struct {
	Prompt   string              `json:"prompt" binding:"required,min=3,max=500"`
	HelpText *string             `json:"helpText"`
	Type     domain.QuestionType `json:"type" binding:"required"`
	Options  []string            `json:"options"`
	Required bool                `json:"required"`
	// Position is optional — nil means append. When re-editing an
	// existing question, the client can move it explicitly.
	Position *int `json:"position"`
}

type ReorderInput struct {
	// Map of questionID → new zero-based position. Every question in the
	// consultation must appear.
	Ordering map[string]int `json:"ordering" binding:"required"`
}

// AnswerInput is one entry in the response body. Exactly one of
// TextValue or Selections must be populated based on question type.
type AnswerInput struct {
	QuestionID string   `json:"questionId" binding:"required,uuid4"`
	TextValue  *string  `json:"textValue"`
	Selections []string `json:"selections"`
}

type SubmitResponseInput struct {
	Answers []AnswerInput `json:"answers" binding:"required,min=1"`
}

type OutcomeInput struct {
	Summary   string `json:"summary" binding:"required,min=10"`
	Decisions string `json:"decisions" binding:"required,min=10"`
	NextSteps string `json:"nextSteps" binding:"required,min=10"`
}

// ── Consultation lifecycle ───────────────────────────────────────

func (s *Service) List(f ListFilters) ([]domain.Consultation, error) {
	return s.repo.FindAll(f)
}

func (s *Service) Get(id string) (*domain.Consultation, error) {
	c, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &AppError{Code: "CONSULTATION_NOT_FOUND", Message: "Consultation not found", Status: http.StatusNotFound}
	}
	return c, err
}

func (s *Service) Create(orgID string, in CreateInput, authorID, authorName string) (*domain.Consultation, error) {
	c := &domain.Consultation{
		ID:             uuid.New().String(),
		OrganizationID: orgID,
		CommunityID:    in.CommunityID,
		Title:          strings.TrimSpace(in.Title),
		Summary:        strings.TrimSpace(in.Summary),
		Description:    in.Description,
		CoverImageURL:  in.CoverImageURL,
		Status:         domain.ConsultationDraft,
		OpensAt:        in.OpensAt,
		ClosesAt:       in.ClosesAt,
		AuthorID:       authorID,
		AuthorName:     authorName,
	}
	if err := s.repo.Create(c); err != nil {
		return nil, err
	}
	return c, nil
}

// Update is DRAFT-only. Editing a published consultation would change
// what earlier responders saw, undermining the record.
func (s *Service) Update(id string, in UpdateInput) (*domain.Consultation, error) {
	c, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if c.Status != domain.ConsultationDraft {
		return nil, &AppError{Code: "NOT_DRAFT", Message: "Only drafts can be edited", Status: http.StatusConflict}
	}
	updates := map[string]any{}
	if in.Title != nil {
		updates["title"] = strings.TrimSpace(*in.Title)
	}
	if in.Summary != nil {
		updates["summary"] = strings.TrimSpace(*in.Summary)
	}
	if in.Description != nil {
		updates["description"] = *in.Description
	}
	if in.CoverImageURL != nil {
		updates["cover_image_url"] = *in.CoverImageURL
	}
	if in.CommunityID != nil {
		updates["community_id"] = *in.CommunityID
	}
	if in.OpensAt != nil {
		updates["opens_at"] = *in.OpensAt
	}
	if in.ClosesAt != nil {
		updates["closes_at"] = *in.ClosesAt
	}
	if len(updates) == 0 {
		return c, nil
	}
	if err := s.repo.Update(id, updates); err != nil {
		return nil, err
	}
	return s.Get(id)
}

// Publish flips DRAFT → PUBLISHED. Requires at least one question — an
// empty form is almost certainly a mistake.
func (s *Service) Publish(id string) (*domain.Consultation, error) {
	c, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if c.Status == domain.ConsultationPublished {
		return c, nil
	}
	if c.Status != domain.ConsultationDraft {
		return nil, &AppError{Code: "INVALID_TRANSITION", Message: "Only drafts can be published", Status: http.StatusConflict}
	}
	qs, err := s.repo.FindQuestions(id)
	if err != nil {
		return nil, err
	}
	if len(qs) == 0 {
		return nil, &AppError{Code: "NO_QUESTIONS", Message: "Add at least one question before publishing", Status: http.StatusBadRequest}
	}
	now := time.Now().UTC()
	if err := s.repo.Update(id, map[string]any{
		"status":       domain.ConsultationPublished,
		"published_at": now,
	}); err != nil {
		return nil, err
	}
	return s.Get(id)
}

// Close flips PUBLISHED → CLOSED. Idempotent — closing an already-closed
// consultation returns it unchanged so the frontend doesn't need special
// error handling for concurrent closes.
func (s *Service) Close(id string) (*domain.Consultation, error) {
	c, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if c.Status == domain.ConsultationClosed {
		return c, nil
	}
	if c.Status != domain.ConsultationPublished {
		return nil, &AppError{Code: "INVALID_TRANSITION", Message: "Only published consultations can be closed", Status: http.StatusConflict}
	}
	now := time.Now().UTC()
	if err := s.repo.Update(id, map[string]any{
		"status":    domain.ConsultationClosed,
		"closed_at": now,
	}); err != nil {
		return nil, err
	}
	return s.Get(id)
}

// Delete is DRAFT-only. Once published, the record must survive so
// responders can revisit their submissions and outcomes stay linked.
func (s *Service) Delete(id string) error {
	c, err := s.Get(id)
	if err != nil {
		return err
	}
	if c.Status != domain.ConsultationDraft {
		return &AppError{Code: "NOT_DRAFT", Message: "Only drafts can be deleted", Status: http.StatusConflict}
	}
	return s.repo.Delete(id)
}

// ── Questions ────────────────────────────────────────────────────

var validQuestionTypes = map[domain.QuestionType]bool{
	domain.QuestionShortText:    true,
	domain.QuestionLongText:     true,
	domain.QuestionSingleChoice: true,
	domain.QuestionMultiChoice:  true,
	domain.QuestionYesNo:        true,
}

func choiceTypesRequireOptions(t domain.QuestionType) bool {
	return t == domain.QuestionSingleChoice || t == domain.QuestionMultiChoice
}

func (s *Service) ListQuestions(consultationID string) ([]domain.ConsultationQuestion, error) {
	return s.repo.FindQuestions(consultationID)
}

func (s *Service) AddQuestion(consultationID string, in QuestionInput) (*domain.ConsultationQuestion, error) {
	c, err := s.Get(consultationID)
	if err != nil {
		return nil, err
	}
	if c.Status != domain.ConsultationDraft {
		return nil, &AppError{Code: "NOT_DRAFT", Message: "Only draft consultations can be edited", Status: http.StatusConflict}
	}
	if !validQuestionTypes[in.Type] {
		return nil, &AppError{Code: "INVALID_QUESTION_TYPE", Message: "Unknown question type", Status: http.StatusBadRequest}
	}
	if choiceTypesRequireOptions(in.Type) && len(in.Options) < 2 {
		return nil, &AppError{Code: "OPTIONS_REQUIRED", Message: "Choice questions need at least two options", Status: http.StatusBadRequest}
	}
	pos := 0
	if in.Position != nil {
		pos = *in.Position
	} else {
		pos, err = s.repo.NextQuestionPosition(consultationID)
		if err != nil {
			return nil, err
		}
	}
	q := &domain.ConsultationQuestion{
		ID:             uuid.New().String(),
		ConsultationID: consultationID,
		Position:       pos,
		Prompt:         strings.TrimSpace(in.Prompt),
		HelpText:       in.HelpText,
		Type:           in.Type,
		Options:        normalizeOptions(in.Options),
		Required:       in.Required,
	}
	if err := s.repo.CreateQuestion(q); err != nil {
		return nil, err
	}
	return q, nil
}

func (s *Service) UpdateQuestion(id string, in QuestionInput) (*domain.ConsultationQuestion, error) {
	q, err := s.getQuestion(id)
	if err != nil {
		return nil, err
	}
	c, err := s.Get(q.ConsultationID)
	if err != nil {
		return nil, err
	}
	if c.Status != domain.ConsultationDraft {
		return nil, &AppError{Code: "NOT_DRAFT", Message: "Only draft consultations can be edited", Status: http.StatusConflict}
	}
	if !validQuestionTypes[in.Type] {
		return nil, &AppError{Code: "INVALID_QUESTION_TYPE", Message: "Unknown question type", Status: http.StatusBadRequest}
	}
	if choiceTypesRequireOptions(in.Type) && len(in.Options) < 2 {
		return nil, &AppError{Code: "OPTIONS_REQUIRED", Message: "Choice questions need at least two options", Status: http.StatusBadRequest}
	}
	updates := map[string]any{
		"prompt":    strings.TrimSpace(in.Prompt),
		"help_text": in.HelpText,
		"type":      in.Type,
		"options":   normalizeOptions(in.Options),
		"required":  in.Required,
	}
	if in.Position != nil {
		updates["position"] = *in.Position
	}
	if err := s.repo.UpdateQuestion(id, updates); err != nil {
		return nil, err
	}
	return s.getQuestion(id)
}

func (s *Service) DeleteQuestion(id string) error {
	q, err := s.getQuestion(id)
	if err != nil {
		return err
	}
	c, err := s.Get(q.ConsultationID)
	if err != nil {
		return err
	}
	if c.Status != domain.ConsultationDraft {
		return &AppError{Code: "NOT_DRAFT", Message: "Only draft consultations can be edited", Status: http.StatusConflict}
	}
	return s.repo.DeleteQuestion(id)
}

func (s *Service) ReorderQuestions(consultationID string, in ReorderInput) error {
	c, err := s.Get(consultationID)
	if err != nil {
		return err
	}
	if c.Status != domain.ConsultationDraft {
		return &AppError{Code: "NOT_DRAFT", Message: "Only draft consultations can be edited", Status: http.StatusConflict}
	}
	return s.repo.ReorderQuestions(consultationID, in.Ordering)
}

func (s *Service) getQuestion(id string) (*domain.ConsultationQuestion, error) {
	q, err := s.repo.FindQuestionByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &AppError{Code: "QUESTION_NOT_FOUND", Message: "Question not found", Status: http.StatusNotFound}
	}
	return q, err
}

// ── Response submission ──────────────────────────────────────────

func (s *Service) SubmitResponse(consultationID, userID string, in SubmitResponseInput) (*domain.ConsultationResponse, error) {
	c, err := s.Get(consultationID)
	if err != nil {
		return nil, err
	}
	if c.Status != domain.ConsultationPublished {
		return nil, &AppError{Code: "NOT_ACCEPTING_RESPONSES", Message: "This consultation isn't accepting responses", Status: http.StatusConflict}
	}
	if c.ClosesAt != nil && time.Now().UTC().After(*c.ClosesAt) {
		return nil, &AppError{Code: "PAST_DEADLINE", Message: "Response window has closed", Status: http.StatusConflict}
	}
	has, err := s.repo.HasResponse(consultationID, userID)
	if err != nil {
		return nil, err
	}
	if has {
		return nil, &AppError{Code: "ALREADY_RESPONDED", Message: "You've already responded to this consultation", Status: http.StatusConflict}
	}
	questions, err := s.repo.FindQuestions(consultationID)
	if err != nil {
		return nil, err
	}
	byID := map[string]domain.ConsultationQuestion{}
	for _, q := range questions {
		byID[q.ID] = q
	}
	// Validate answers against the questions. Every required question
	// must have a non-empty answer; every submitted answer must reference
	// a question that belongs to this consultation.
	suppliedRequired := map[string]bool{}
	answers := make([]domain.ConsultationAnswer, 0, len(in.Answers))
	for _, a := range in.Answers {
		q, ok := byID[a.QuestionID]
		if !ok {
			return nil, &AppError{Code: "UNKNOWN_QUESTION", Message: "Answer references a question outside this consultation", Status: http.StatusBadRequest}
		}
		if err := validateAnswer(q, a); err != nil {
			return nil, err
		}
		answers = append(answers, domain.ConsultationAnswer{
			ID:         uuid.New().String(),
			QuestionID: a.QuestionID,
			TextValue:  a.TextValue,
			Selections: a.Selections,
		})
		if q.Required {
			suppliedRequired[q.ID] = true
		}
	}
	for _, q := range questions {
		if q.Required && !suppliedRequired[q.ID] {
			return nil, &AppError{Code: "REQUIRED_UNANSWERED", Message: "One or more required questions were not answered", Status: http.StatusBadRequest}
		}
	}
	now := time.Now().UTC()
	resp := &domain.ConsultationResponse{
		ID:             uuid.New().String(),
		ConsultationID: consultationID,
		UserID:         userID,
		SubmittedAt:    now,
	}
	for i := range answers {
		answers[i].ResponseID = resp.ID
	}
	if err := s.repo.CreateResponse(resp, answers); err != nil {
		return nil, err
	}
	_ = s.repo.BumpResponseCount(consultationID, 1)
	return resp, nil
}

// validateAnswer ensures the submitted answer matches the question type.
// Empty text for a required question is caught by the required-check
// loop, so here we just check shape.
func validateAnswer(q domain.ConsultationQuestion, a AnswerInput) error {
	switch q.Type {
	case domain.QuestionShortText, domain.QuestionLongText:
		if a.TextValue != nil && strings.TrimSpace(*a.TextValue) == "" {
			// treat empty string as no answer; validated by required-check
			return nil
		}
	case domain.QuestionSingleChoice:
		if len(a.Selections) > 1 {
			return &AppError{Code: "SINGLE_CHOICE_ONLY", Message: "Single-choice questions accept exactly one selection", Status: http.StatusBadRequest}
		}
		if len(a.Selections) == 1 && !stringInSlice(a.Selections[0], q.Options) {
			return &AppError{Code: "UNKNOWN_OPTION", Message: "Selected option is not on the question", Status: http.StatusBadRequest}
		}
	case domain.QuestionMultiChoice:
		for _, sel := range a.Selections {
			if !stringInSlice(sel, q.Options) {
				return &AppError{Code: "UNKNOWN_OPTION", Message: "Selected option is not on the question", Status: http.StatusBadRequest}
			}
		}
	case domain.QuestionYesNo:
		if len(a.Selections) > 1 {
			return &AppError{Code: "SINGLE_CHOICE_ONLY", Message: "Yes/No questions accept exactly one selection", Status: http.StatusBadRequest}
		}
		if len(a.Selections) == 1 && a.Selections[0] != "YES" && a.Selections[0] != "NO" {
			return &AppError{Code: "INVALID_YES_NO", Message: "Yes/No answers must be YES or NO", Status: http.StatusBadRequest}
		}
	}
	return nil
}

// ── Response reads ───────────────────────────────────────────────

// ResponseWithAnswers bundles a response and its answers for admin viewing.
type ResponseWithAnswers struct {
	Response domain.ConsultationResponse `json:"response"`
	Answers  []domain.ConsultationAnswer `json:"answers"`
}

func (s *Service) ListResponses(consultationID string, limit, offset int) ([]ResponseWithAnswers, int64, error) {
	responses, total, err := s.repo.FindResponses(consultationID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	if len(responses) == 0 {
		return []ResponseWithAnswers{}, total, nil
	}
	ids := make([]string, len(responses))
	for i, r := range responses {
		ids[i] = r.ID
	}
	answers, err := s.repo.FindAnswers(ids)
	if err != nil {
		return nil, 0, err
	}
	byResponse := map[string][]domain.ConsultationAnswer{}
	for _, a := range answers {
		byResponse[a.ResponseID] = append(byResponse[a.ResponseID], a)
	}
	out := make([]ResponseWithAnswers, len(responses))
	for i, r := range responses {
		out[i] = ResponseWithAnswers{Response: r, Answers: byResponse[r.ID]}
	}
	return out, total, nil
}

func (s *Service) MyResponseIDs(userID string) ([]string, error) {
	ids, err := s.repo.FindMyResponseIDs(userID)
	if err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []string{}
	}
	return ids, nil
}

// ── Analytics ────────────────────────────────────────────────────

// Aggregate is the per-question rollup returned by GET /analytics.
type Aggregate struct {
	QuestionID   string              `json:"questionId"`
	Prompt       string              `json:"prompt"`
	Type         domain.QuestionType `json:"type"`
	AnswerCount  int                 `json:"answerCount"`
	OptionCounts map[string]int      `json:"optionCounts,omitempty"`
	// TextValues is populated for SHORT_TEXT and LONG_TEXT — capped so
	// a large consultation doesn't blow up the response body.
	TextValues []string `json:"textValues,omitempty"`
}

const maxTextSamplesPerQuestion = 100

func (s *Service) Analytics(consultationID string) ([]Aggregate, error) {
	if _, err := s.Get(consultationID); err != nil {
		return nil, err
	}
	questions, err := s.repo.FindQuestions(consultationID)
	if err != nil {
		return nil, err
	}
	answers, err := s.repo.AllAnswersForConsultation(consultationID)
	if err != nil {
		return nil, err
	}
	byQuestion := map[string][]domain.ConsultationAnswer{}
	for _, a := range answers {
		byQuestion[a.QuestionID] = append(byQuestion[a.QuestionID], a)
	}
	out := make([]Aggregate, 0, len(questions))
	for _, q := range questions {
		agg := Aggregate{
			QuestionID: q.ID,
			Prompt:     q.Prompt,
			Type:       q.Type,
		}
		list := byQuestion[q.ID]
		agg.AnswerCount = len(list)
		switch q.Type {
		case domain.QuestionSingleChoice, domain.QuestionMultiChoice, domain.QuestionYesNo:
			counts := map[string]int{}
			for _, a := range list {
				for _, sel := range a.Selections {
					counts[sel]++
				}
			}
			agg.OptionCounts = counts
		case domain.QuestionShortText, domain.QuestionLongText:
			samples := make([]string, 0, min(len(list), maxTextSamplesPerQuestion))
			for _, a := range list {
				if a.TextValue == nil {
					continue
				}
				samples = append(samples, *a.TextValue)
				if len(samples) >= maxTextSamplesPerQuestion {
					break
				}
			}
			agg.TextValues = samples
		}
		out = append(out, agg)
	}
	return out, nil
}

// ── Outcome ──────────────────────────────────────────────────────

func (s *Service) GetOutcome(consultationID string) (*domain.ConsultationOutcome, error) {
	o, err := s.repo.FindOutcome(consultationID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &AppError{Code: "OUTCOME_NOT_FOUND", Message: "No outcome has been published yet", Status: http.StatusNotFound}
	}
	return o, err
}

func (s *Service) PublishOutcome(consultationID string, in OutcomeInput, authorID, authorName string) (*domain.ConsultationOutcome, error) {
	c, err := s.Get(consultationID)
	if err != nil {
		return nil, err
	}
	if c.Status != domain.ConsultationClosed {
		return nil, &AppError{Code: "NOT_CLOSED", Message: "Close the consultation before publishing an outcome", Status: http.StatusConflict}
	}
	now := time.Now().UTC()
	o := &domain.ConsultationOutcome{
		ID:             uuid.New().String(),
		ConsultationID: consultationID,
		Summary:        in.Summary,
		Decisions:      in.Decisions,
		NextSteps:      in.NextSteps,
		AuthorID:       authorID,
		AuthorName:     authorName,
		PublishedAt:    now,
	}
	if err := s.repo.CreateOrUpdateOutcome(o); err != nil {
		return nil, err
	}
	return s.GetOutcome(consultationID)
}

// ── Helpers ──────────────────────────────────────────────────────

// ResponderIDs returns the set of user IDs who have submitted a response
// to this consultation. Used by notification fan-out on close +
// outcome-published.
func (s *Service) ResponderIDs(consultationID string) ([]string, error) {
	responses, _, err := s.repo.FindResponses(consultationID, 0, 0)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(responses))
	seen := map[string]bool{}
	for _, r := range responses {
		if seen[r.UserID] {
			continue
		}
		seen[r.UserID] = true
		ids = append(ids, r.UserID)
	}
	return ids, nil
}

func normalizeOptions(opts []string) []string {
	if len(opts) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(opts))
	for _, o := range opts {
		t := strings.TrimSpace(o)
		if t == "" {
			continue
		}
		out = append(out, t)
	}
	return out
}

func stringInSlice(v string, list []string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
