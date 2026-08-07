package organizations

import (
	"strings"

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
	if err := r.db.Where("organization_id = ?", orgID).Order("joined_at asc").Find(&list).Error; err != nil {
		return nil, err
	}
	// Names and platform roles are stored as snapshots; show what is true
	// now, not what was true when each person joined.
	return list, r.hydrateMembers(list)
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

// userRecord is a read-only view of `users`, owned by identity-service.
// Same shared-database pattern as the communities and audit packages: only
// the columns actually read are declared, TableName is pinned, and nothing
// here ever writes.
type userRecord struct {
	ID        string  `gorm:"type:uuid;primaryKey"`
	Email     string  `gorm:"column:email"`
	Name      string  `gorm:"column:name"`
	Role      string  `gorm:"column:role"`
	DeletedAt *string `gorm:"column:deleted_at"`
}

func (userRecord) TableName() string { return "users" }

// FindUserByEmail resolves the person an org owner is trying to add.
//
// Adding by email rather than by UUID is what makes member management
// usable at all: an org owner knows their colleague's email address, never
// their internal ID. Matching is case-insensitive because nobody types
// their own address the same way twice.
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

// hydrateMembers replaces the stored UserName/UserRole snapshots with what
// `users` currently says.
//
// Those columns are written once at join time and never updated, so a
// member who has since been renamed, or promoted from CITIZEN to
// REPRESENTATIVE, would otherwise be displayed with values that were true
// months ago. One grouped query per page rather than a join, to keep the
// cross-service read in one obvious place.
//
// A user row that has vanished leaves the snapshot untouched — it is the
// only record left of who that member was.
func (r *Repository) hydrateMembers(list []domain.OrgMember) error {
	if len(list) == 0 {
		return nil
	}
	ids := make([]string, 0, len(list))
	for i := range list {
		ids = append(ids, list[i].UserID)
	}

	var users []userRecord
	if err := r.db.Table("users").Select("id, email, name, role, deleted_at").
		Where("id IN ?", ids).Scan(&users).Error; err != nil {
		return err
	}

	byID := make(map[string]userRecord, len(users))
	for _, u := range users {
		byID[u.ID] = u
	}
	for i := range list {
		if u, ok := byID[list[i].UserID]; ok {
			list[i].UserName = u.Name
			list[i].UserRole = u.Role
		}
	}
	return nil
}

func (r *Repository) UpdateMemberTitle(orgID, userID string, title *string) error {
	return r.db.Model(&domain.OrgMember{}).
		Where("organization_id = ? AND user_id = ?", orgID, userID).
		Update("title", title).Error
}
