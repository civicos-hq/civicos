package organizations

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/civicos/organization-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// representativeRecord is a read-only view of the `representatives` table,
// which community-service owns. Same shared-database pattern the audit and
// communities packages use: redeclare only the columns actually read, pin
// TableName, never write.
type representativeRecord struct {
	ID           string  `gorm:"type:uuid;primaryKey"`
	Name         string  `gorm:"column:name"`
	Title        string  `gorm:"column:title"`
	Position     string  `gorm:"column:position"`
	Constituency string  `gorm:"column:constituency"`
	Email        *string `gorm:"column:email"`
	Phone        *string `gorm:"column:phone"`
	Website      *string `gorm:"column:website"`
	AvatarURL    *string `gorm:"column:avatar_url"`
	CommunityID  string  `gorm:"column:community_id;type:uuid"`
	UserID       *string `gorm:"column:user_id;type:uuid"`
}

func (representativeRecord) TableName() string { return "representatives" }

// communityRecord is the same arrangement for `communities`, read only to
// place an office in a state and LGA.
type communityRecord struct {
	ID    string `gorm:"type:uuid;primaryKey"`
	Name  string
	State string
	LGA   string
}

func (communityRecord) TableName() string { return "communities" }

// FindRepresentativeByUserID returns the representative profile CLAIMED by
// this account, if any. An unclaimed profile — one whose user_id is NULL —
// is deliberately not matched: the claim is what ties an office to a named
// human, and it is the substitute for a registration number when the
// office asks to raise money.
func (r *Repository) FindRepresentativeByUserID(userID string) (*representativeRecord, error) {
	var rec representativeRecord
	if err := r.db.Where("user_id = ?", userID).First(&rec).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *Repository) FindCommunity(id string) (*communityRecord, error) {
	var rec communityRecord
	if err := r.db.Where("id = ?", id).First(&rec).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *Repository) FindOfficeByRepresentativeID(repID string) (*domain.Organization, error) {
	var o domain.Organization
	if err := r.db.Where("representative_id = ?", repID).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// CreateOffice writes the organization and its owner membership in one
// transaction. A half-created office — a row nobody can administer, or a
// membership pointing at nothing — would need manual repair in the
// database, so neither is allowed to exist alone.
func (r *Repository) CreateOffice(org *domain.Organization, owner *domain.OrgMember) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(org).Error; err != nil {
			return err
		}
		if err := tx.Create(owner).Error; err != nil {
			return err
		}
		return tx.Model(&domain.Organization{}).Where("id = ?", org.ID).
			Update("member_count", 1).Error
	})
}

// ─── Service ──────────────────────────────────────────────────────────

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	out := nonSlugChars.ReplaceAllString(strings.ToLower(strings.TrimSpace(s)), "-")
	return strings.Trim(out, "-")
}

// officeName reads as the office, not the person: "Office of Senator Ada
// Okafor". The holder is recorded separately in RepresentativeName, which
// is the field FundingEligible checks for an accountable human — offices
// change hands and the name on a donation receipt should say which office
// took the money.
func officeName(rec *representativeRecord) string {
	title := strings.TrimSpace(rec.Title)
	name := strings.TrimSpace(rec.Name)
	// Profiles vary: some store "Senator Ada Okafor" in Name with Title
	// empty, some split them. Don't produce "Office of Senator Senator Ada".
	if title != "" && !strings.HasPrefix(strings.ToLower(name), strings.ToLower(title)) {
		return fmt.Sprintf("Office of %s %s", title, name)
	}
	if name == "" {
		return "Representative office"
	}
	return "Office of " + name
}

// jurisdictionFor maps a representative's community to the reach of their
// office. Best-effort and deliberately conservative: COMMUNITY is the
// narrowest option, so a wrong guess under-claims reach rather than
// letting an office advertise itself as national.
//
// Position strings are free text entered by an admin or applicant, so this
// is a heuristic, not a taxonomy. An admin can correct it afterwards
// through the normal organization update.
func jurisdictionFor(position string) domain.OrgJurisdiction {
	p := strings.ToLower(position)
	switch {
	case strings.Contains(p, "senator"), strings.Contains(p, "senate"),
		strings.Contains(p, "house of representatives"), strings.Contains(p, "federal"),
		strings.Contains(p, "president"):
		return domain.JurisdictionNational
	case strings.Contains(p, "governor"), strings.Contains(p, "state assembly"),
		strings.Contains(p, "commissioner"), strings.Contains(p, "state"):
		return domain.JurisdictionState
	case strings.Contains(p, "chairman"), strings.Contains(p, "councilor"),
		strings.Contains(p, "councillor"), strings.Contains(p, "local government"),
		strings.Contains(p, "lga"):
		return domain.JurisdictionLGA
	default:
		return domain.JurisdictionCommunity
	}
}

// ProvisionRepresentativeOffice returns the caller's constituency office,
// creating it on first call.
//
// Idempotent by design rather than by convention: a representative reaches
// this every time they open their dashboard, so it must be safe to call
// repeatedly. The database also carries a unique index on
// representative_id, because the check-then-create below is not atomic and
// two concurrent calls would otherwise both create one.
//
// The office starts UNVERIFIED with no payout account, exactly as a newly
// registered organization does. Provisioning grants the ability to draft;
// it grants nothing about money. FundingEligible still has to pass before
// a campaign can leave review, and that needs a platform admin.
func (s *Service) ProvisionRepresentativeOffice(userID, userRole string) (*domain.Organization, error) {
	if userRole != "REPRESENTATIVE" {
		return nil, &AppError{
			Code:    "NOT_A_REPRESENTATIVE",
			Message: "Only an approved representative has a constituency office",
			Status:  http.StatusForbidden,
		}
	}

	rec, err := s.repo.FindRepresentativeByUserID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Says what is wrong and who fixes it. The account is a
			// representative but no profile names it as the holder — an
			// administrative gap, not the caller doing something wrong.
			return nil, &AppError{
				Code:    "REPRESENTATIVE_UNCLAIMED",
				Message: "Your account is not linked to a representative profile yet. A platform admin has to link it before your office can be created.",
				Status:  http.StatusConflict,
			}
		}
		return nil, err
	}

	existing, err := s.repo.FindOfficeByRepresentativeID(rec.ID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	name := officeName(rec)
	org := &domain.Organization{
		ID:               uuid.New().String(),
		Name:             name,
		Slug:             slugify(name) + "-" + strings.Split(rec.ID, "-")[0],
		Kind:             domain.OrgKindRepresentativeOffice,
		Jurisdiction:     jurisdictionFor(rec.Position),
		Email:            rec.Email,
		Phone:            rec.Phone,
		Website:          rec.Website,
		LogoURL:          rec.AvatarURL,
		RepresentativeID: &rec.ID,
		// The accountable human behind anything this office publishes.
		RepresentativeName: &rec.Name,
		// Unverified, no payout account: same starting position as any
		// other new organization.
		Verified:    false,
		CreatedByID: userID,
	}

	// Place the office where the representative serves, so its campaigns
	// and projects tier correctly in Discover.
	if community, cErr := s.repo.FindCommunity(rec.CommunityID); cErr == nil {
		org.State = &community.State
		org.LGA = &community.LGA
	}

	owner := &domain.OrgMember{
		ID:             uuid.New().String(),
		OrganizationID: org.ID,
		UserID:         userID,
		UserName:       rec.Name,
		UserRole:       "REPRESENTATIVE",
		Role:           domain.MemberRoleOwner,
		JoinedAt:       time.Now().UTC(),
	}

	if err := s.repo.CreateOffice(org, owner); err != nil {
		// Lost a race with a concurrent provision. The unique index did its
		// job; return the office that won rather than an error, since the
		// caller wanted an office and there now is one.
		if won, findErr := s.repo.FindOfficeByRepresentativeID(rec.ID); findErr == nil {
			return won, nil
		}
		return nil, err
	}
	return org, nil
}
