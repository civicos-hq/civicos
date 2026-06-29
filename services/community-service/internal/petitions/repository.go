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

func (r *Repository) AddSignature(petitionID, userID string) error {
	// Use transaction to ensure unique signature and count consistency
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Check if user already signed
		var sig domain.PetitionSignature
		if err := tx.Where("petition_id = ? AND user_id = ?", petitionID, userID).First(&sig).Error; err == nil {
			return nil // already signed
		}
		// Create signature
		sig = domain.PetitionSignature{ID: uuid.New().String(), PetitionID: petitionID, UserID: userID, CreatedAt: time.Now()}
		if err := tx.Create(&sig).Error; err != nil {
			// If a concurrent transaction created the same signature between
			// the existence check and create, the DB will return a unique
			// constraint error. Treat that as idempotent and return nil.
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "unique") {
				return nil
			}
			return err
		}
		// Increment signature count
		if err := tx.Model(&domain.Petition{}).Where("id = ?", petitionID).
			UpdateColumn("signature_count", gorm.Expr("signature_count + 1")).Error; err != nil {
			return err
		}
		return nil
	})
}

// uuid returns a pseudo-UUID string using database-side generation when available.
// (no additional helpers)
