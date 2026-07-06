package adminmetrics

import (
	"time"

	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// Metrics is the platform-wide snapshot shown on the admin Overview
// page. Every count is a fresh COUNT(*) — cheap because every parent
// table is small and every FK column is indexed. If usage grows past
// six figures we swap this for a nightly materialized view.
type Metrics struct {
	Users           UsersMetrics      `json:"users"`
	Communities     CountOnly         `json:"communities"`
	Issues          IssueMetrics      `json:"issues"`
	Petitions       PetitionMetrics   `json:"petitions"`
	Representatives CountOnly         `json:"representatives"`
	Organizations   OrgMetrics        `json:"organizations"`
	Moderation      ModerationMetrics `json:"moderation"`
}

type UsersMetrics struct {
	Total        int64 `json:"total"`
	NewToday     int64 `json:"newToday"`
	NewThisWeek  int64 `json:"newThisWeek"`
	VerifiedRate int   `json:"verifiedRate"` // 0-100
	BannedTotal  int64 `json:"bannedTotal"`
}

type CountOnly struct {
	Total int64 `json:"total"`
}

type IssueMetrics struct {
	Total        int64            `json:"total"`
	ByStatus     map[string]int64 `json:"byStatus"`
	ResponseRate int              `json:"responseRate"` // 0-100 — issues with ≥1 official response or org progress update
}

type PetitionMetrics struct {
	Total              int64 `json:"total"`
	SignaturesTotal    int64 `json:"signaturesTotal"`
	SignaturesThisWeek int64 `json:"signaturesThisWeek"`
}

type OrgMetrics struct {
	Total    int64 `json:"total"`
	Verified int64 `json:"verified"`
}

type ModerationMetrics struct {
	PendingFlags    int64 `json:"pendingFlags"`
	HiddenAllTime   int64 `json:"hiddenAllTime"`
	AuditLogEntries int64 `json:"auditLogEntries"`
}

func (r *Repository) Snapshot() (*Metrics, error) {
	m := &Metrics{}
	now := time.Now().UTC()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	weekStart := todayStart.AddDate(0, 0, -7)

	// Users
	if err := r.db.Raw("SELECT COUNT(*) FROM users").Scan(&m.Users.Total).Error; err != nil {
		return nil, err
	}
	if err := r.db.Raw("SELECT COUNT(*) FROM users WHERE created_at >= ?", todayStart).Scan(&m.Users.NewToday).Error; err != nil {
		return nil, err
	}
	if err := r.db.Raw("SELECT COUNT(*) FROM users WHERE created_at >= ?", weekStart).Scan(&m.Users.NewThisWeek).Error; err != nil {
		return nil, err
	}
	if err := r.db.Raw("SELECT COUNT(*) FROM users WHERE banned_at IS NOT NULL").Scan(&m.Users.BannedTotal).Error; err != nil {
		return nil, err
	}
	var verified int64
	if err := r.db.Raw("SELECT COUNT(*) FROM users WHERE email_verified = true").Scan(&verified).Error; err != nil {
		return nil, err
	}
	if m.Users.Total > 0 {
		m.Users.VerifiedRate = int(verified * 100 / m.Users.Total)
	}

	// Communities / Representatives — simple totals
	_ = r.db.Raw("SELECT COUNT(*) FROM communities").Scan(&m.Communities.Total).Error
	_ = r.db.Raw("SELECT COUNT(*) FROM representatives").Scan(&m.Representatives.Total).Error

	// Issues
	_ = r.db.Raw("SELECT COUNT(*) FROM issues").Scan(&m.Issues.Total).Error
	m.Issues.ByStatus = map[string]int64{}
	rows, err := r.db.Raw("SELECT status, COUNT(*) FROM issues GROUP BY status").Rows()
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s string
			var n int64
			if err := rows.Scan(&s, &n); err == nil {
				m.Issues.ByStatus[s] = n
			}
		}
	}
	// Response rate — an issue "has a response" if it received at least
	// one official comment (is_official_response=true) OR at least one
	// public progress update from an assigned organization.
	if m.Issues.Total > 0 {
		var responded int64
		_ = r.db.Raw(`
			SELECT COUNT(*) FROM issues i
			WHERE EXISTS (SELECT 1 FROM issue_comments ic WHERE ic.issue_id = i.id AND ic.is_official_response = true)
			   OR EXISTS (SELECT 1 FROM progress_updates pu WHERE pu.issue_id = i.id AND pu.is_public = true)
		`).Scan(&responded).Error
		m.Issues.ResponseRate = int(responded * 100 / m.Issues.Total)
	}

	// Petitions
	_ = r.db.Raw("SELECT COUNT(*) FROM petitions").Scan(&m.Petitions.Total).Error
	_ = r.db.Raw("SELECT COUNT(*) FROM petition_signatures").Scan(&m.Petitions.SignaturesTotal).Error
	_ = r.db.Raw("SELECT COUNT(*) FROM petition_signatures WHERE created_at >= ?", weekStart).Scan(&m.Petitions.SignaturesThisWeek).Error

	// Organizations
	_ = r.db.Raw("SELECT COUNT(*) FROM organizations").Scan(&m.Organizations.Total).Error
	_ = r.db.Raw("SELECT COUNT(*) FROM organizations WHERE verified = true").Scan(&m.Organizations.Verified).Error

	// Moderation
	_ = r.db.Raw("SELECT COUNT(*) FROM content_flags WHERE status = 'PENDING'").Scan(&m.Moderation.PendingFlags).Error
	_ = r.db.Raw("SELECT COUNT(*) FROM content_flags WHERE status = 'HIDDEN'").Scan(&m.Moderation.HiddenAllTime).Error
	_ = r.db.Raw("SELECT COUNT(*) FROM audit_logs").Scan(&m.Moderation.AuditLogEntries).Error

	return m, nil
}

// CommunityStats is the per-community drill-down the Communities admin
// page needs. Fresh COUNT(*)s again — same "cheap because everything
// is indexed" rationale.
type CommunityStats struct {
	CitizenCount        int64            `json:"citizenCount"`
	IssueTotal          int64            `json:"issueTotal"`
	IssuesByStatus      map[string]int64 `json:"issuesByStatus"`
	PetitionTotal       int64            `json:"petitionTotal"`
	RepresentativeTotal int64            `json:"representativeTotal"`
}

func (r *Repository) CommunityStats(communityID string) (*CommunityStats, error) {
	s := &CommunityStats{IssuesByStatus: map[string]int64{}}
	_ = r.db.Raw("SELECT COUNT(*) FROM users WHERE community_id = ?", communityID).Scan(&s.CitizenCount).Error
	_ = r.db.Raw("SELECT COUNT(*) FROM issues WHERE community_id = ?", communityID).Scan(&s.IssueTotal).Error
	rows, err := r.db.Raw("SELECT status, COUNT(*) FROM issues WHERE community_id = ? GROUP BY status", communityID).Rows()
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var status string
			var n int64
			if err := rows.Scan(&status, &n); err == nil {
				s.IssuesByStatus[status] = n
			}
		}
	}
	_ = r.db.Raw("SELECT COUNT(*) FROM petitions WHERE community_id = ?", communityID).Scan(&s.PetitionTotal).Error
	_ = r.db.Raw("SELECT COUNT(*) FROM representatives WHERE community_id = ?", communityID).Scan(&s.RepresentativeTotal).Error
	return s, nil
}
