package flags

import (
	"github.com/civicos/identity-service/internal/domain"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

type ListFilters struct {
	Status      string
	ContentType string
	ReporterID  string
	Limit       int
	Offset      int
}

func (r *Repository) Find(f ListFilters) ([]domain.ContentFlag, error) {
	q := r.db.Model(&domain.ContentFlag{})
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.ContentType != "" {
		q = q.Where("content_type = ?", f.ContentType)
	}
	if f.ReporterID != "" {
		q = q.Where("reporter_id = ?", f.ReporterID)
	}
	if f.Limit <= 0 {
		f.Limit = 50
	}
	var list []domain.ContentFlag
	return list, q.
		Order("created_at desc").
		Limit(f.Limit).
		Offset(f.Offset).
		Find(&list).Error
}

func (r *Repository) FindByID(id string) (*domain.ContentFlag, error) {
	var f domain.ContentFlag
	return &f, r.db.Where("id = ?", id).First(&f).Error
}

func (r *Repository) Create(f *domain.ContentFlag) error {
	return r.db.Create(f).Error
}

func (r *Repository) Update(id string, updates map[string]any) error {
	return r.db.Model(&domain.ContentFlag{}).Where("id = ?", id).Updates(updates).Error
}

func (r *Repository) CountByStatus() (map[string]int64, error) {
	rows, err := r.db.Model(&domain.ContentFlag{}).
		Select("status, COUNT(*) as n").
		Group("status").
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]int64{}
	for rows.Next() {
		var status string
		var n int64
		if err := rows.Scan(&status, &n); err != nil {
			return nil, err
		}
		counts[status] = n
	}
	return counts, nil
}
