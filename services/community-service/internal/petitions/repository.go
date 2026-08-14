package petitions

import (
	"time"

	"github.com/civicos/community-service/internal/domain"
	"github.com/civicos/community-service/internal/moderation"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository implements PetitionStore using GORM.
type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) FindAll(communityID, status string) ([]domain.Petition, error) {
	var list []domain.Petition
	q := r.db.Order("created_at desc")
	if communityID != "" {
		q = q.Where("community_id = ?", communityID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	return list, q.Find(&list).Error
}

func (r *Repository) FindByID(id string) (*domain.Petition, error) {
	var p domain.Petition
	return &p, r.db.Where("id = ?", id).First(&p).Error
}

func (r *Repository) Create(p *domain.Petition) error {
	p.CreatedAt = time.Now()
	return r.db.Create(p).Error
}

func (r *Repository) ListComments(petitionID string) ([]domain.PetitionComment, error) {
	// See issues/repository.go for the hide-placeholder rationale.
	var list []domain.PetitionComment
	if err := r.db.
		Where("petition_id = ?", petitionID).
		Order("created_at asc").
		Find(&list).Error; err != nil {
		return nil, err
	}
	ids := make([]string, len(list))
	for i, c := range list {
		ids[i] = c.ID
	}
	hidden := moderation.HiddenSet(r.db, "PETITION_COMMENT", ids)
	for i := range list {
		if _, ok := hidden[list[i].ID]; ok {
			list[i].IsHidden = true
			list[i].Content = moderation.PlaceholderContent
			list[i].AuthorName = moderation.PlaceholderAuthorName
		}
	}
	return list, nil
}

func (r *Repository) AddComment(comment *domain.PetitionComment) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(comment).Error; err != nil {
			return err
		}
		return tx.Model(&domain.Petition{}).Where("id = ?", comment.PetitionID).
			UpdateColumn("comment_count", gorm.Expr("comment_count + 1")).Error
	})
}

// AddSignature is idempotent. Returns added=true only when a new signature row
// was actually created (so the caller can fire milestone notifications without
// duplicating them on repeat sign attempts). newCount reflects the post-commit
// signature_count for the petition.
func (r *Repository) AddSignature(petitionID, userID string) (added bool, newCount int, err error) {
	err = r.db.Transaction(func(tx *gorm.DB) error {
		// Let the database resolve the conflict instead of inserting and
		// catching.
		//
		// This previously read the signature first and, if absent,
		// inserted and treated a duplicate-key error as "already signed".
		// That is check-then-act: two concurrent signs both see no row and
		// both insert, so one violates idx_petition_user.
		//
		// Catching that error did not help, it hurt. Postgres aborts the
		// entire transaction the moment any statement in it fails, so the
		// SELECT below then failed with "current transaction is aborted"
		// and the request 500'd — a confusing failure two statements away
		// from its cause. The handler looked idempotent and was not.
		//
		// ON CONFLICT means no statement ever errors, so the transaction
		// stays usable and RowsAffected says whether we were the inserter.
		//
		// The conflict target is named rather than a bare DO NOTHING: if
		// idx_petition_user ever went missing this fails loudly, instead
		// of silently writing duplicate signatures and inflating the count
		// that the whole petition rests on.
		sig := domain.PetitionSignature{
			ID:         uuid.New().String(),
			PetitionID: petitionID,
			UserID:     userID,
			CreatedAt:  time.Now(),
		}
		res := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "petition_id"}, {Name: "user_id"}},
			DoNothing: true,
		}).Create(&sig)
		if res.Error != nil {
			return res.Error
		}

		// Exactly one concurrent caller sees 1 here, which is what stops
		// milestone notifications firing once per attempt.
		added = res.RowsAffected > 0
		if added {
			if uerr := tx.Model(&domain.Petition{}).Where("id = ?", petitionID).
				UpdateColumn("signature_count", gorm.Expr("signature_count + 1")).Error; uerr != nil {
				return uerr
			}
		}

		var p domain.Petition
		if perr := tx.Select("signature_count").Where("id = ?", petitionID).First(&p).Error; perr != nil {
			return perr
		}
		newCount = p.SignatureCount
		return nil
	})
	return added, newCount, err
}

// uuid returns a pseudo-UUID string using database-side generation when available.
// (no additional helpers)
