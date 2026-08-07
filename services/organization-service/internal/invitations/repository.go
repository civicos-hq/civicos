package invitations

import (
	"strings"
	"time"

	"github.com/civicos/organization-service/internal/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// userRecord is a read-only view of `users`, owned by identity-service.
// Same shared-database pattern used elsewhere in this service: declare only
// the columns read, pin TableName, never write.
type userRecord struct {
	ID        string  `gorm:"type:uuid;primaryKey"`
	Email     string  `gorm:"column:email"`
	Name      string  `gorm:"column:name"`
	Role      string  `gorm:"column:role"`
	DeletedAt *string `gorm:"column:deleted_at"`
}

func (userRecord) TableName() string { return "users" }

func (r *Repository) FindUserByEmail(email string) (*userRecord, error) {
	var u userRecord
	if err := r.db.Where("LOWER(email) = LOWER(?)", strings.TrimSpace(email)).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) FindUserByID(id string) (*userRecord, error) {
	var u userRecord
	if err := r.db.Where("id = ?", id).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) FindOrganization(id string) (*domain.Organization, error) {
	var o domain.Organization
	if err := r.db.Where("id = ?", id).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *Repository) FindMember(orgID, userID string) (*domain.OrgMember, error) {
	var m domain.OrgMember
	if err := r.db.Where("organization_id = ? AND user_id = ?", orgID, userID).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *Repository) FindByTokenHash(hash string) (*domain.OrgInvitation, error) {
	var inv domain.OrgInvitation
	if err := r.db.Where("token_hash = ?", hash).First(&inv).Error; err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *Repository) FindByID(id string) (*domain.OrgInvitation, error) {
	var inv domain.OrgInvitation
	if err := r.db.Where("id = ?", id).First(&inv).Error; err != nil {
		return nil, err
	}
	return &inv, nil
}

// ListPending returns invitations that can still be accepted, newest first.
// Expired-but-unaccepted rows are included: an inviter needs to see that a
// link went stale, otherwise the person appears to have been invited and
// nothing ever explains why they never arrived.
func (r *Repository) ListPending(orgID string) ([]domain.OrgInvitation, error) {
	var list []domain.OrgInvitation
	return list, r.db.
		Where("organization_id = ? AND accepted_at IS NULL AND revoked_at IS NULL", orgID).
		Order("created_at desc").
		Find(&list).Error
}

// ReplacePending revokes any outstanding invitation for this address and
// creates the new one in a single transaction.
//
// Re-inviting is what an owner does when the first link expired or went to
// a typo'd address, and they expect it to supersede rather than to fail
// against the pending-uniqueness index. Revoking rather than deleting keeps
// the record of who was invited and by whom.
func (r *Repository) ReplacePending(inv *domain.OrgInvitation, revokedByID string) error {
	now := time.Now().UTC()
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&domain.OrgInvitation{}).
			Where("organization_id = ? AND email = ? AND accepted_at IS NULL AND revoked_at IS NULL",
				inv.OrganizationID, inv.Email).
			Updates(map[string]any{"revoked_at": now, "revoked_by_id": revokedByID}).Error; err != nil {
			return err
		}
		return tx.Create(inv).Error
	})
}

func (r *Repository) Revoke(id, byUserID string) error {
	now := time.Now().UTC()
	res := r.db.Model(&domain.OrgInvitation{}).
		Where("id = ? AND accepted_at IS NULL AND revoked_at IS NULL", id).
		Updates(map[string]any{"revoked_at": now, "revoked_by_id": byUserID})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Accept marks the invitation used and creates the membership together.
//
// One transaction because the two halves are meaningless apart: a consumed
// invitation with no membership locks the person out permanently, and a
// membership from an invitation still marked pending leaves a live token
// for an org that can raise money.
//
// The guard on accepted_at/revoked_at makes acceptance single-use even if
// two requests arrive at once — the loser updates zero rows and is told the
// invitation is no longer valid.
func (r *Repository) Accept(invID, userID string, member *domain.OrgMember) error {
	now := time.Now().UTC()
	return r.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&domain.OrgInvitation{}).
			Where("id = ? AND accepted_at IS NULL AND revoked_at IS NULL", invID).
			Updates(map[string]any{"accepted_at": now, "accepted_user_id": userID})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		if err := tx.Create(member).Error; err != nil {
			return err
		}
		return tx.Model(&domain.Organization{}).
			Where("id = ?", member.OrganizationID).
			UpdateColumn("member_count", gorm.Expr("member_count + 1")).Error
	})
}
