package petitions

import (
	"strings"
	"time"

	"github.com/civicos/community-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
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
	var list []domain.PetitionComment
	return list, r.db.Where("petition_id = ?", petitionID).
		Order("created_at asc").Find(&list).Error
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
		var sig domain.PetitionSignature
		if err := tx.Where("petition_id = ? AND user_id = ?", petitionID, userID).First(&sig).Error; err == nil {
			added = false
		} else {
			sig = domain.PetitionSignature{ID: uuid.New().String(), PetitionID: petitionID, UserID: userID, CreatedAt: time.Now()}
			if cerr := tx.Create(&sig).Error; cerr != nil {
				if strings.Contains(strings.ToLower(cerr.Error()), "duplicate") || strings.Contains(strings.ToLower(cerr.Error()), "unique") {
					added = false
				} else {
					return cerr
				}
			} else {
				added = true
				if uerr := tx.Model(&domain.Petition{}).Where("id = ?", petitionID).
					UpdateColumn("signature_count", gorm.Expr("signature_count + 1")).Error; uerr != nil {
					return uerr
				}
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
