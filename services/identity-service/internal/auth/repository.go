package auth

import (
	"github.com/civicos/identity-service/internal/domain"
	"gorm.io/gorm"
)

// Repository handles all User database queries for the auth domain.
// It is injected into AuthService — never instantiated directly inside services.
type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindByEmail(email string) (*domain.User, error) {
	var user domain.User
	result := r.db.Where("email = ?", email).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func (r *Repository) FindByID(id string) (*domain.User, error) {
	var user domain.User
	result := r.db.Where("id = ?", id).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func (r *Repository) Create(user *domain.User) error {
	return r.db.Create(user).Error
}

func (r *Repository) UpdateCommunity(userID, communityID string) error {
	return r.db.Model(&domain.User{}).Where("id = ?", userID).Update("community_id", communityID).Error
}
