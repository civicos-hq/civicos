package communities

import (
	"strings"

	"github.com/civicos/community-service/internal/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// SearchParams narrows the community list. Every zero value means "no
// constraint on this field", so one query serves all three callers: the
// onboarding wizard searching by name, its state/LGA drill-down fallback,
// and the browse page listing everything.
type SearchParams struct {
	Query string
	State string
	LGA   string
	// IDs restricts the result to a specific set. The browse page needs it
	// to render the communities a user has joined: those are known by ID
	// from their memberships and are not necessarily on the page of search
	// results currently being shown.
	IDs    []string
	Limit  int
	Offset int
}

// Search returns one page of communities plus the total matching count so
// the caller can paginate.
//
// Name search exists because the drill-down alone could not find a
// university: a student looking for "University of Abuja" had to already
// know it sits in Gwagwalada, while Nile, Baze and AUN are in AMAC. Nobody
// thinks about their campus in terms of local government areas.
func (r *Repository) Search(p SearchParams) ([]domain.Community, int64, error) {
	q := r.db.Model(&domain.Community{})
	if len(p.IDs) > 0 {
		q = q.Where("id IN ?", p.IDs)
	}
	if p.State != "" {
		q = q.Where("state = ?", p.State)
	}
	if p.LGA != "" {
		q = q.Where("lga = ?", p.LGA)
	}

	term := strings.TrimSpace(p.Query)
	if term != "" {
		like := "%" + escapeLike(term) + "%"
		q = q.Where("name ILIKE ? OR description ILIKE ? OR lga ILIKE ? OR state ILIKE ?", like, like, like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Prefix matches lead. Typing "uni" should surface "University of
	// Abuja" above "Federal Capital Territory University Village", which a
	// plain alphabetical sort would invert.
	if term != "" {
		q = q.Order(gorm.Expr("CASE WHEN name ILIKE ? THEN 0 ELSE 1 END", escapeLike(term)+"%"))
	}
	q = q.Order("name asc")

	if p.Limit > 0 {
		q = q.Limit(p.Limit)
	}
	if p.Offset > 0 {
		q = q.Offset(p.Offset)
	}

	var list []domain.Community
	if err := q.Find(&list).Error; err != nil {
		return nil, 0, err
	}
	if err := r.attachMemberCounts(list); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *Repository) FindByID(id string) (*domain.Community, error) {
	var c domain.Community
	if err := r.db.Where("id = ?", id).First(&c).Error; err != nil {
		return nil, err
	}
	one := []domain.Community{c}
	if err := r.attachMemberCounts(one); err != nil {
		return nil, err
	}
	return &one[0], nil
}

// FindByIDs loads a specific set of communities, used to validate a batch
// join before any membership is written.
func (r *Repository) FindByIDs(ids []string) ([]domain.Community, error) {
	if len(ids) == 0 {
		return []domain.Community{}, nil
	}
	var list []domain.Community
	if err := r.db.Where("id IN ?", ids).Order("name asc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, r.attachMemberCounts(list)
}

func (r *Repository) Create(c *domain.Community) error {
	return r.db.Create(c).Error
}

// attachMemberCounts fills MemberCount for a page of communities with one
// grouped query rather than a correlated subquery per row. Pages are capped
// at maxPageSize, so the IN list stays small.
func (r *Repository) attachMemberCounts(list []domain.Community) error {
	if len(list) == 0 {
		return nil
	}
	ids := make([]string, 0, len(list))
	for i := range list {
		ids = append(ids, list[i].ID)
	}

	var rows []struct {
		CommunityID string
		Total       int
	}
	if err := r.db.
		Table("user_community_memberships").
		Select("community_id, COUNT(*) AS total").
		Where("community_id IN ?", ids).
		Group("community_id").
		Scan(&rows).Error; err != nil {
		return err
	}

	counts := make(map[string]int, len(rows))
	for _, row := range rows {
		counts[row.CommunityID] = row.Total
	}
	for i := range list {
		list[i].MemberCount = counts[list[i].ID]
	}
	return nil
}

// escapeLike neutralises the LIKE wildcards so a user typing "100%" searches
// for that literal string instead of matching every row.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	return strings.ReplaceAll(s, `_`, `\_`)
}
