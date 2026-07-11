package consultations

import (
	"testing"
	"time"

	"github.com/civicos/organization-service/internal/domain"
	"github.com/google/uuid"
)

// TestConsultationLifecycle exercises the full happy path against a
// fake in-memory store — no DB. If the DB behaviour ever diverges from
// what the fake models here, we'll catch it in integration tests, not
// this smoke.
func TestConsultationLifecycle(t *testing.T) {
	svc := NewService(newFakeStore())
	authorID, authorName := uuid.NewString(), "Ada Admin"

	// 1. Create draft
	created, err := svc.Create(uuid.NewString(), CreateInput{
		Title:       "National Curriculum Review",
		Summary:     "How should the syllabus change for 2027?",
		Description: "We are collecting citizen input before finalizing the curriculum.",
	}, authorID, authorName)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Status != domain.ConsultationDraft {
		t.Fatalf("expected DRAFT, got %s", created.Status)
	}

	// 2. Try to publish with no questions — should refuse.
	if _, err := svc.Publish(created.ID); err == nil {
		t.Fatalf("expected publish-with-no-questions to fail")
	} else if appErr, ok := err.(*AppError); !ok || appErr.Code != "NO_QUESTIONS" {
		t.Fatalf("expected NO_QUESTIONS, got %v", err)
	}

	// 3. Add three questions.
	q1, err := svc.AddQuestion(created.ID, QuestionInput{
		Prompt: "Do you agree with the current syllabus?",
		Type:   domain.QuestionYesNo,
	})
	if err != nil {
		t.Fatalf("add q1: %v", err)
	}
	q2, err := svc.AddQuestion(created.ID, QuestionInput{
		Prompt:   "Which subjects deserve more focus?",
		Type:     domain.QuestionMultiChoice,
		Options:  []string{"Maths", "Science", "History", "Civic Ed"},
		Required: true,
	})
	if err != nil {
		t.Fatalf("add q2: %v", err)
	}
	q3, err := svc.AddQuestion(created.ID, QuestionInput{
		Prompt: "Anything else?",
		Type:   domain.QuestionLongText,
	})
	if err != nil {
		t.Fatalf("add q3: %v", err)
	}

	// Validation: single-choice with too many options.
	_, err = svc.AddQuestion(created.ID, QuestionInput{
		Prompt: "One choice", Type: domain.QuestionSingleChoice, Options: []string{"Only"},
	})
	if err == nil {
		t.Fatalf("expected OPTIONS_REQUIRED")
	} else if appErr, ok := err.(*AppError); !ok || appErr.Code != "OPTIONS_REQUIRED" {
		t.Fatalf("expected OPTIONS_REQUIRED, got %v", err)
	}

	// 4. Publish.
	published, err := svc.Publish(created.ID)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if published.Status != domain.ConsultationPublished {
		t.Fatalf("expected PUBLISHED, got %s", published.Status)
	}
	if published.PublishedAt == nil {
		t.Fatalf("expected publishedAt to be set")
	}

	// 5. After publish, questions are frozen.
	if _, err := svc.AddQuestion(created.ID, QuestionInput{Prompt: "Late", Type: domain.QuestionShortText}); err == nil {
		t.Fatalf("expected NOT_DRAFT when adding after publish")
	}

	// 6. Two different users respond. Required q2 must be answered.
	userA := uuid.NewString()
	yes := "YES"
	textA := "More civics please."
	_, err = svc.SubmitResponse(created.ID, userA, SubmitResponseInput{
		Answers: []AnswerInput{
			{QuestionID: q1.ID, Selections: []string{yes}},
			{QuestionID: q2.ID, Selections: []string{"Civic Ed", "Science"}},
			{QuestionID: q3.ID, TextValue: &textA},
		},
	})
	if err != nil {
		t.Fatalf("submit A: %v", err)
	}

	// Same user resubmitting — one-per-user enforcement.
	if _, err := svc.SubmitResponse(created.ID, userA, SubmitResponseInput{
		Answers: []AnswerInput{{QuestionID: q2.ID, Selections: []string{"Maths"}}},
	}); err == nil {
		t.Fatalf("expected ALREADY_RESPONDED on second submit")
	} else if appErr, ok := err.(*AppError); !ok || appErr.Code != "ALREADY_RESPONDED" {
		t.Fatalf("expected ALREADY_RESPONDED, got %v", err)
	}

	// Missing required question — should refuse.
	userB := uuid.NewString()
	if _, err := svc.SubmitResponse(created.ID, userB, SubmitResponseInput{
		Answers: []AnswerInput{{QuestionID: q1.ID, Selections: []string{"NO"}}},
	}); err == nil {
		t.Fatalf("expected REQUIRED_UNANSWERED")
	} else if appErr, ok := err.(*AppError); !ok || appErr.Code != "REQUIRED_UNANSWERED" {
		t.Fatalf("expected REQUIRED_UNANSWERED, got %v", err)
	}

	// Second user submits a valid answer set.
	_, err = svc.SubmitResponse(created.ID, userB, SubmitResponseInput{
		Answers: []AnswerInput{
			{QuestionID: q1.ID, Selections: []string{"NO"}},
			{QuestionID: q2.ID, Selections: []string{"Civic Ed"}},
		},
	})
	if err != nil {
		t.Fatalf("submit B: %v", err)
	}

	// 7. Analytics rollup.
	agg, err := svc.Analytics(created.ID)
	if err != nil {
		t.Fatalf("analytics: %v", err)
	}
	if len(agg) != 3 {
		t.Fatalf("expected 3 aggregates, got %d", len(agg))
	}
	for _, a := range agg {
		switch a.QuestionID {
		case q1.ID:
			if a.OptionCounts["YES"] != 1 || a.OptionCounts["NO"] != 1 {
				t.Fatalf("q1 rollup wrong: %+v", a.OptionCounts)
			}
		case q2.ID:
			if a.OptionCounts["Civic Ed"] != 2 || a.OptionCounts["Science"] != 1 {
				t.Fatalf("q2 rollup wrong: %+v", a.OptionCounts)
			}
		case q3.ID:
			if len(a.TextValues) != 1 || a.TextValues[0] != textA {
				t.Fatalf("q3 rollup wrong: %+v", a.TextValues)
			}
		}
	}

	// 8. Cannot publish outcome before closing.
	if _, err := svc.PublishOutcome(created.ID, OutcomeInput{
		Summary: "s", Decisions: "d", NextSteps: "n",
	}, authorID, authorName); err == nil {
		t.Fatalf("expected NOT_CLOSED when outcome before close")
	}

	// 9. Close.
	closed, err := svc.Close(created.ID)
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	if closed.Status != domain.ConsultationClosed {
		t.Fatalf("expected CLOSED, got %s", closed.Status)
	}

	// New responses rejected after close.
	if _, err := svc.SubmitResponse(created.ID, uuid.NewString(), SubmitResponseInput{
		Answers: []AnswerInput{{QuestionID: q1.ID, Selections: []string{"YES"}}, {QuestionID: q2.ID, Selections: []string{"Maths"}}},
	}); err == nil {
		t.Fatalf("expected submit after close to fail")
	}

	// 10. Publish outcome.
	outcome, err := svc.PublishOutcome(created.ID, OutcomeInput{
		Summary:   "Citizens want more civic education.",
		Decisions: "We will introduce weekly civic ed classes.",
		NextSteps: "Rollout starts Q3 2027.",
	}, authorID, authorName)
	if err != nil {
		t.Fatalf("publish outcome: %v", err)
	}
	if outcome.Decisions == "" {
		t.Fatalf("outcome decisions empty")
	}
	if outcome.PublishedAt.IsZero() {
		t.Fatalf("outcome publishedAt not set")
	}

	// Republishing overwrites (idempotent).
	outcome2, err := svc.PublishOutcome(created.ID, OutcomeInput{
		Summary:   "Updated summary.",
		Decisions: "Refined decisions.",
		NextSteps: "Refined next steps.",
	}, authorID, authorName)
	if err != nil {
		t.Fatalf("republish outcome: %v", err)
	}
	if outcome2.Summary != "Updated summary." {
		t.Fatalf("outcome not overwritten")
	}

	// 11. Responder IDs — two unique.
	ids, err := svc.ResponderIDs(created.ID)
	if err != nil {
		t.Fatalf("responder ids: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 responders, got %d", len(ids))
	}
}

// TestAddQuestion_OptionsAlwaysEmptySlice locks in the API contract that
// caused the citizen-side render crash reported earlier: a question type
// that doesn't take options (SHORT_TEXT, LONG_TEXT, YES_NO) must have
// Options == []string{} on read, never nil. If this ever regresses the
// downstream JSON serializes as `null`, and any client doing
// `q.options.length` crashes on mount.
func TestAddQuestion_OptionsAlwaysEmptySlice(t *testing.T) {
	svc := NewService(newFakeStore())
	authorID, authorName := uuid.NewString(), "Ada"

	c, err := svc.Create(uuid.NewString(), CreateInput{
		Title:       "Options contract",
		Summary:     "Verifying that empty options come back as [] not nil.",
		Description: "This test documents the write-path invariant.",
	}, authorID, authorName)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Every non-choice type must round-trip with Options == []string{}.
	cases := []struct {
		name string
		typ  domain.QuestionType
	}{
		{"short_text", domain.QuestionShortText},
		{"long_text", domain.QuestionLongText},
		{"yes_no", domain.QuestionYesNo},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q, err := svc.AddQuestion(c.ID, QuestionInput{
				Prompt: "Any prompt",
				Type:   tc.typ,
				// deliberately nil — the caller may or may not pass an
				// empty slice; either way the stored value must be [].
			})
			if err != nil {
				t.Fatalf("add: %v", err)
			}
			if q.Options == nil {
				t.Fatalf("Options was nil after AddQuestion — should be []string{}")
			}
			if len(q.Options) != 0 {
				t.Fatalf("expected empty options, got %v", q.Options)
			}
		})
	}
}

// ── fake store ───────────────────────────────────────────────────

type fakeStore struct {
	consultations map[string]*domain.Consultation
	questions     map[string]*domain.ConsultationQuestion
	responses     map[string]*domain.ConsultationResponse
	answers       map[string]*domain.ConsultationAnswer
	outcomes      map[string]*domain.ConsultationOutcome
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		consultations: map[string]*domain.Consultation{},
		questions:     map[string]*domain.ConsultationQuestion{},
		responses:     map[string]*domain.ConsultationResponse{},
		answers:       map[string]*domain.ConsultationAnswer{},
		outcomes:      map[string]*domain.ConsultationOutcome{},
	}
}

func (f *fakeStore) FindAll(filters ListFilters) ([]domain.Consultation, error) {
	out := make([]domain.Consultation, 0, len(f.consultations))
	for _, c := range f.consultations {
		out = append(out, *c)
	}
	return out, nil
}

func (f *fakeStore) FindByID(id string) (*domain.Consultation, error) {
	c, ok := f.consultations[id]
	if !ok {
		return nil, notFoundErr{}
	}
	// return a copy so the caller can mutate without affecting the store
	cp := *c
	return &cp, nil
}

func (f *fakeStore) Create(c *domain.Consultation) error {
	c.CreatedAt = time.Now().UTC()
	c.UpdatedAt = c.CreatedAt
	cp := *c
	f.consultations[c.ID] = &cp
	return nil
}

func (f *fakeStore) Update(id string, updates map[string]any) error {
	c, ok := f.consultations[id]
	if !ok {
		return notFoundErr{}
	}
	if v, ok := updates["title"]; ok {
		c.Title = v.(string)
	}
	if v, ok := updates["summary"]; ok {
		c.Summary = v.(string)
	}
	if v, ok := updates["description"]; ok {
		c.Description = v.(string)
	}
	if v, ok := updates["status"]; ok {
		c.Status = v.(domain.ConsultationStatus)
	}
	if v, ok := updates["published_at"]; ok {
		t := v.(time.Time)
		c.PublishedAt = &t
	}
	if v, ok := updates["closed_at"]; ok {
		t := v.(time.Time)
		c.ClosedAt = &t
	}
	c.UpdatedAt = time.Now().UTC()
	return nil
}

func (f *fakeStore) Delete(id string) error {
	delete(f.consultations, id)
	for qid, q := range f.questions {
		if q.ConsultationID == id {
			delete(f.questions, qid)
		}
	}
	return nil
}

func (f *fakeStore) BumpResponseCount(id string, delta int) error {
	c, ok := f.consultations[id]
	if !ok {
		return notFoundErr{}
	}
	c.ResponseCount += delta
	return nil
}

func (f *fakeStore) FindQuestions(consultationID string) ([]domain.ConsultationQuestion, error) {
	out := []domain.ConsultationQuestion{}
	for _, q := range f.questions {
		if q.ConsultationID == consultationID {
			out = append(out, *q)
		}
	}
	// stable order for the test — by Position
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[i].Position > out[j].Position {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

func (f *fakeStore) FindQuestionByID(id string) (*domain.ConsultationQuestion, error) {
	q, ok := f.questions[id]
	if !ok {
		return nil, notFoundErr{}
	}
	cp := *q
	return &cp, nil
}

func (f *fakeStore) CreateQuestion(q *domain.ConsultationQuestion) error {
	q.CreatedAt = time.Now().UTC()
	q.UpdatedAt = q.CreatedAt
	cp := *q
	f.questions[q.ID] = &cp
	return nil
}

func (f *fakeStore) UpdateQuestion(id string, updates map[string]any) error {
	q, ok := f.questions[id]
	if !ok {
		return notFoundErr{}
	}
	if v, ok := updates["prompt"]; ok {
		q.Prompt = v.(string)
	}
	if v, ok := updates["type"]; ok {
		q.Type = v.(domain.QuestionType)
	}
	if v, ok := updates["options"]; ok {
		q.Options = v.([]string)
	}
	if v, ok := updates["required"]; ok {
		q.Required = v.(bool)
	}
	if v, ok := updates["position"]; ok {
		q.Position = v.(int)
	}
	q.UpdatedAt = time.Now().UTC()
	return nil
}

func (f *fakeStore) DeleteQuestion(id string) error {
	delete(f.questions, id)
	return nil
}

func (f *fakeStore) NextQuestionPosition(consultationID string) (int, error) {
	max := 0
	for _, q := range f.questions {
		if q.ConsultationID == consultationID && q.Position > max {
			max = q.Position
		}
	}
	return max + 1, nil
}

func (f *fakeStore) ReorderQuestions(consultationID string, ordering map[string]int) error {
	for id, pos := range ordering {
		q, ok := f.questions[id]
		if !ok || q.ConsultationID != consultationID {
			continue
		}
		q.Position = pos
	}
	return nil
}

func (f *fakeStore) HasResponse(consultationID, userID string) (bool, error) {
	for _, r := range f.responses {
		if r.ConsultationID == consultationID && r.UserID == userID {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeStore) CreateResponse(resp *domain.ConsultationResponse, answers []domain.ConsultationAnswer) error {
	resp.CreatedAt = time.Now().UTC()
	cp := *resp
	f.responses[resp.ID] = &cp
	for _, a := range answers {
		acp := a
		f.answers[a.ID] = &acp
	}
	return nil
}

func (f *fakeStore) FindResponses(consultationID string, limit, offset int) ([]domain.ConsultationResponse, int64, error) {
	out := []domain.ConsultationResponse{}
	for _, r := range f.responses {
		if r.ConsultationID == consultationID {
			out = append(out, *r)
		}
	}
	return out, int64(len(out)), nil
}

func (f *fakeStore) FindAnswers(responseIDs []string) ([]domain.ConsultationAnswer, error) {
	set := map[string]bool{}
	for _, id := range responseIDs {
		set[id] = true
	}
	out := []domain.ConsultationAnswer{}
	for _, a := range f.answers {
		if set[a.ResponseID] {
			out = append(out, *a)
		}
	}
	return out, nil
}

func (f *fakeStore) FindMyResponseIDs(userID string) ([]string, error) {
	out := []string{}
	for _, r := range f.responses {
		if r.UserID == userID {
			out = append(out, r.ConsultationID)
		}
	}
	return out, nil
}

func (f *fakeStore) AllAnswersForConsultation(consultationID string) ([]domain.ConsultationAnswer, error) {
	respIDs := map[string]bool{}
	for _, r := range f.responses {
		if r.ConsultationID == consultationID {
			respIDs[r.ID] = true
		}
	}
	out := []domain.ConsultationAnswer{}
	for _, a := range f.answers {
		if respIDs[a.ResponseID] {
			out = append(out, *a)
		}
	}
	return out, nil
}

func (f *fakeStore) FindOutcome(consultationID string) (*domain.ConsultationOutcome, error) {
	for _, o := range f.outcomes {
		if o.ConsultationID == consultationID {
			cp := *o
			return &cp, nil
		}
	}
	return nil, notFoundErr{}
}

func (f *fakeStore) CreateOrUpdateOutcome(o *domain.ConsultationOutcome) error {
	for id, existing := range f.outcomes {
		if existing.ConsultationID == o.ConsultationID {
			f.outcomes[id] = &domain.ConsultationOutcome{
				ID:             existing.ID,
				ConsultationID: existing.ConsultationID,
				Summary:        o.Summary,
				Decisions:      o.Decisions,
				NextSteps:      o.NextSteps,
				AuthorID:       o.AuthorID,
				AuthorName:     o.AuthorName,
				PublishedAt:    o.PublishedAt,
				CreatedAt:      existing.CreatedAt,
				UpdatedAt:      time.Now().UTC(),
			}
			return nil
		}
	}
	cp := *o
	cp.CreatedAt = time.Now().UTC()
	cp.UpdatedAt = cp.CreatedAt
	f.outcomes[o.ID] = &cp
	return nil
}

// notFoundErr mirrors gorm.ErrRecordNotFound so service code's
// errors.Is check works.
type notFoundErr struct{}

func (notFoundErr) Error() string { return "record not found" }
func (notFoundErr) Is(target error) bool {
	return target != nil && target.Error() == "record not found"
}

// TestDedupeUserIDs covers the audience-merge helper that the publish
// fan-out uses. Org and community memberships often overlap in
// practice — the same citizen joins their local NGO and the community
// the NGO serves — and we don't want them to receive the same
// notification twice.
func TestDedupeUserIDs(t *testing.T) {
	cases := []struct {
		name          string
		first, second []string
		want          []string
	}{
		{
			name:   "no overlap keeps both, org first",
			first:  []string{"a", "b"},
			second: []string{"c", "d"},
			want:   []string{"a", "b", "c", "d"},
		},
		{
			name:   "overlap keeps the org-side row",
			first:  []string{"a", "b"},
			second: []string{"b", "c"},
			want:   []string{"a", "b", "c"},
		},
		{
			name:   "empty inputs are safe",
			first:  nil,
			second: nil,
			want:   []string{},
		},
		{
			name:   "empty strings are dropped",
			first:  []string{"", "a"},
			second: []string{"", "b"},
			want:   []string{"a", "b"},
		},
		{
			name:   "internal duplicates in one slice are collapsed",
			first:  []string{"a", "a", "b"},
			second: []string{"c", "c"},
			want:   []string{"a", "b", "c"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := dedupeUserIDs(tc.first, tc.second)
			if len(got) != len(tc.want) {
				t.Fatalf("length mismatch: got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("at %d: got %q, want %q (full: %v)", i, got[i], tc.want[i], got)
				}
			}
		})
	}
}
