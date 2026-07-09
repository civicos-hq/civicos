package organizations

import (
	"github.com/civicos/organization-service/internal/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

type ListFilters struct {
	Kind         string
	Jurisdiction string
	State        string
	LGA          string
	Search       string
}

func (r *Repository) FindAll(f ListFilters) ([]domain.Organization, error) {
	q := r.db.Model(&domain.Organization{})
	if f.Kind != "" {
		q = q.Where("kind = ?", f.Kind)
	}
	if f.Jurisdiction != "" {
		q = q.Where("jurisdiction = ?", f.Jurisdiction)
	}
	if f.State != "" {
		q = q.Where("state = ?", f.State)
	}
	if f.LGA != "" {
		q = q.Where("lga = ?", f.LGA)
	}
	if f.Search != "" {
		q = q.Where("LOWER(name) LIKE ?", "%"+f.Search+"%")
	}
	var list []domain.Organization
	return list, q.Order("verified desc, name asc").Find(&list).Error
}

func (r *Repository) FindByID(id string) (*domain.Organization, error) {
	var o domain.Organization
	return &o, r.db.Where("id = ?", id).First(&o).Error
}

func (r *Repository) FindBySlug(slug string) (*domain.Organization, error) {
	var o domain.Organization
	return &o, r.db.Where("slug = ?", slug).First(&o).Error
}

func (r *Repository) Create(o *domain.Organization) error {
	return r.db.Create(o).Error
}

func (r *Repository) Update(id string, updates map[string]any) error {
	return r.db.Model(&domain.Organization{}).Where("id = ?", id).Updates(updates).Error
}

// FindMember returns the caller's membership row for an org, or ErrRecordNotFound
// if they are not a member. Used to authorise write endpoints.
func (r *Repository) FindMember(orgID, userID string) (*domain.OrgMember, error) {
	var m domain.OrgMember
	return &m, r.db.Where("organization_id = ? AND user_id = ?", orgID, userID).First(&m).Error
}

// FindMembershipsByUser lists every membership row the caller holds
// across all orgs. Used by the `/me/organizations` endpoint so the
// frontend can render the "orgs you can act as" picker without doing
// N+1 lookups.
func (r *Repository) FindMembershipsByUser(userID string) ([]domain.OrgMember, error) {
	var list []domain.OrgMember
	return list, r.db.Where("user_id = ?", userID).Order("joined_at asc").Find(&list).Error
}

// FindByIDs returns the orgs for a set of IDs, preserving that set —
// callers use it to pair each membership with its org record.
func (r *Repository) FindByIDs(ids []string) ([]domain.Organization, error) {
	if len(ids) == 0 {
		return []domain.Organization{}, nil
	}
	var list []domain.Organization
	return list, r.db.Where("id IN ?", ids).Find(&list).Error
}

func (r *Repository) ListMembers(orgID string) ([]domain.OrgMember, error) {
	var list []domain.OrgMember
	return list, r.db.Where("organization_id = ?", orgID).Order("joined_at asc").Find(&list).Error
}

func (r *Repository) AddMember(m *domain.OrgMember) error {
	return r.db.Create(m).Error
}

func (r *Repository) UpdateMemberRole(orgID, userID string, role domain.OrgMemberRole) error {
	return r.db.Model(&domain.OrgMember{}).
		Where("organization_id = ? AND user_id = ?", orgID, userID).
		Update("role", role).Error
}

func (r *Repository) RemoveMember(orgID, userID string) error {
	return r.db.Where("organization_id = ? AND user_id = ?", orgID, userID).
		Delete(&domain.OrgMember{}).Error
}

// IncrementMemberCount and its siblings keep denormalised counters on the
// Organization row so the list page doesn't have to COUNT() per org.
func (r *Repository) BumpMemberCount(orgID string, delta int) error {
	return r.db.Model(&domain.Organization{}).Where("id = ?", orgID).
		Update("member_count", gorm.Expr("member_count + ?", delta)).Error
}
