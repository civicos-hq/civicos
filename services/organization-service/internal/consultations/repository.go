package consultations

import (
	"github.com/civicos/organization-service/internal/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// ── Consultations ────────────────────────────────────────────────

// ListFilters is applied by the public list endpoint. Empty fields are
// ignored so callers can layer filters.
type ListFilters struct {
	OrganizationID string
	CommunityID    string
	Status         string
	Limit          int
	Offset         int
}

// FindAll lists consultations with optional filters. DRAFT items are
// still returned — access to drafts is gated at the handler layer (org
// members only). This keeps the repository layer policy-free.
func (r *Repository) FindAll(f ListFilters) ([]domain.Consultation, error) {
	q := r.db.Model(&domain.Consultation{})
	if f.OrganizationID != "" {
		q = q.Where("organization_id = ?", f.OrganizationID)
	}
	if f.CommunityID != "" {
		q = q.Where("community_id = ?", f.CommunityID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	q = q.Order("COALESCE(published_at, created_at) desc")
	if f.Limit > 0 {
		q = q.Limit(f.Limit)
	}
	if f.Offset > 0 {
		q = q.Offset(f.Offset)
	}
	var list []domain.Consultation
	return list, q.Find(&list).Error
}

func (r *Repository) FindByID(id string) (*domain.Consultation, error) {
	var c domain.Consultation
	return &c, r.db.First(&c, "id = ?", id).Error
}

func (r *Repository) Create(c *domain.Consultation) error {
	return r.db.Create(c).Error
}

func (r *Repository) Update(id string, updates map[string]any) error {
	return r.db.Model(&domain.Consultation{}).Where("id = ?", id).Updates(updates).Error
}

func (r *Repository) Delete(id string) error {
	// Deleting a draft cascades to its questions. Responses cannot exist
	// on a draft, so no cleanup there.
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&domain.ConsultationQuestion{}, "consultation_id = ?", id).Error; err != nil {
			return err
		}
		return tx.Delete(&domain.Consultation{}, "id = ?", id).Error
	})
}

// BumpResponseCount is called after a successful response submit. The
// denormalized counter powers the list page without a subquery.
func (r *Repository) BumpResponseCount(id string, delta int) error {
	return r.db.Model(&domain.Consultation{}).Where("id = ?", id).
		Update("response_count", gorm.Expr("response_count + ?", delta)).Error
}

// ── Questions ────────────────────────────────────────────────────

func (r *Repository) FindQuestions(consultationID string) ([]domain.ConsultationQuestion, error) {
	var qs []domain.ConsultationQuestion
	return qs, r.db.
		Where("consultation_id = ?", consultationID).
		Order("position asc").
		Find(&qs).Error
}

func (r *Repository) FindQuestionByID(id string) (*domain.ConsultationQuestion, error) {
	var q domain.ConsultationQuestion
	return &q, r.db.First(&q, "id = ?", id).Error
}

func (r *Repository) CreateQuestion(q *domain.ConsultationQuestion) error {
	return r.db.Create(q).Error
}

func (r *Repository) UpdateQuestion(id string, updates map[string]any) error {
	return r.db.Model(&domain.ConsultationQuestion{}).Where("id = ?", id).Updates(updates).Error
}

func (r *Repository) DeleteQuestion(id string) error {
	return r.db.Delete(&domain.ConsultationQuestion{}, "id = ?", id).Error
}

// NextQuestionPosition is used when a caller appends without specifying
// a position. Returns the current max+1 so the new question sits at the
// bottom.
func (r *Repository) NextQuestionPosition(consultationID string) (int, error) {
	var max int
	err := r.db.Model(&domain.ConsultationQuestion{}).
		Where("consultation_id = ?", consultationID).
		Select("COALESCE(MAX(position), 0)").
		Scan(&max).Error
	return max + 1, err
}

// ReorderQuestions applies a full ordering in one transaction. The map
// is questionID → new position. Callers are expected to provide every
// question in the consultation — partial reorders are not supported here.
func (r *Repository) ReorderQuestions(consultationID string, ordering map[string]int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for id, pos := range ordering {
			if err := tx.Model(&domain.ConsultationQuestion{}).
				Where("id = ? AND consultation_id = ?", id, consultationID).
				Update("position", pos).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ── Responses + Answers ──────────────────────────────────────────

func (r *Repository) HasResponse(consultationID, userID string) (bool, error) {
	var count int64
	err := r.db.Model(&domain.ConsultationResponse{}).
		Where("consultation_id = ? AND user_id = ?", consultationID, userID).
		Count(&count).Error
	return count > 0, err
}

// CreateResponse writes the response header and all answers in a single
// transaction. Bump of the consultation's counter is left to the caller
// so it can be paired with a notification emit outside the tx.
func (r *Repository) CreateResponse(resp *domain.ConsultationResponse, answers []domain.ConsultationAnswer) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(resp).Error; err != nil {
			return err
		}
		if len(answers) == 0 {
			return nil
		}
		return tx.Create(&answers).Error
	})
}

func (r *Repository) FindResponses(consultationID string, limit, offset int) ([]domain.ConsultationResponse, int64, error) {
	var total int64
	if err := r.db.Model(&domain.ConsultationResponse{}).
		Where("consultation_id = ?", consultationID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []domain.ConsultationResponse
	q := r.db.Where("consultation_id = ?", consultationID).Order("submitted_at desc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}
	return list, total, q.Find(&list).Error
}

func (r *Repository) FindAnswers(responseIDs []string) ([]domain.ConsultationAnswer, error) {
	if len(responseIDs) == 0 {
		return []domain.ConsultationAnswer{}, nil
	}
	var list []domain.ConsultationAnswer
	return list, r.db.Where("response_id IN ?", responseIDs).Find(&list).Error
}

// FindMyResponseIDs returns the consultation IDs the caller has already
// responded to. Used by /me/consultations/responses so the frontend can
// mark "you've participated" in the list view.
func (r *Repository) FindMyResponseIDs(userID string) ([]string, error) {
	var ids []string
	err := r.db.Model(&domain.ConsultationResponse{}).
		Where("user_id = ?", userID).
		Order("submitted_at desc").
		Pluck("consultation_id", &ids).Error
	return ids, err
}

// ── Analytics ────────────────────────────────────────────────────

// AllAnswersForConsultation is the raw material for analytics. Small
// consultations can be aggregated in-memory in the service layer;
// larger ones will need a SQL rollup later.
func (r *Repository) AllAnswersForConsultation(consultationID string) ([]domain.ConsultationAnswer, error) {
	var list []domain.ConsultationAnswer
	return list, r.db.
		Table("consultation_answers AS a").
		Joins("JOIN consultation_responses AS r ON r.id = a.response_id").
		Where("r.consultation_id = ?", consultationID).
		Select("a.*").
		Scan(&list).Error
}

// ── Outcome ──────────────────────────────────────────────────────

func (r *Repository) FindOutcome(consultationID string) (*domain.ConsultationOutcome, error) {
	var o domain.ConsultationOutcome
	return &o, r.db.Where("consultation_id = ?", consultationID).First(&o).Error
}

func (r *Repository) CreateOrUpdateOutcome(o *domain.ConsultationOutcome) error {
	// Idempotent: one outcome per consultation. If it exists, update;
	// otherwise create. Enforced by the unique index on consultation_id.
	var existing domain.ConsultationOutcome
	err := r.db.Where("consultation_id = ?", o.ConsultationID).First(&existing).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return r.db.Create(o).Error
		}
		return err
	}
	return r.db.Model(&existing).Updates(map[string]any{
		"summary":      o.Summary,
		"decisions":    o.Decisions,
		"next_steps":   o.NextSteps,
		"author_id":    o.AuthorID,
		"author_name":  o.AuthorName,
		"published_at": o.PublishedAt,
	}).Error
}
