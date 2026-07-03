package issues

import (
	"errors"

	"github.com/civicos/community-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) FindAll(communityID, status, category string) ([]domain.Issue, error) {
	var list []domain.Issue
	q := r.db.Order("created_at desc")
	if communityID != "" {
		q = q.Where("community_id = ?", communityID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if category != "" {
		q = q.Where("category = ?", category)
	}
	return list, q.Find(&list).Error
}

func (r *Repository) FindByID(id string) (*domain.Issue, error) {
	var issue domain.Issue
	return &issue, r.db.Where("id = ?", id).First(&issue).Error
}

func (r *Repository) Create(issue *domain.Issue) error {
	return r.db.Create(issue).Error
}

// HasUserUpvoted reports whether the user already has an IssueUpvote row for
// this issue. Callers use this to decide add-vs-remove without a race.
func (r *Repository) HasUserUpvoted(issueID, userID string) (bool, error) {
	var count int64
	err := r.db.Model(&domain.IssueUpvote{}).
		Where("issue_id = ? AND user_id = ?", issueID, userID).
		Count(&count).Error
	return count > 0, err
}

// AddUpvote inserts the (issue, user) row and bumps upvote_count in a single
// transaction. If the row already exists (idx_issue_upvoter unique) we treat
// the operation as a no-op — matches the "click twice, land in the same
// state" contract the toggle handler needs.
func (r *Repository) AddUpvote(issueID, userID string) (int, error) {
	var count int
	err := r.db.Transaction(func(tx *gorm.DB) error {
		vote := &domain.IssueUpvote{
			ID: uuid.New().String(), IssueID: issueID, UserID: userID,
		}
		if err := tx.Create(vote).Error; err != nil {
			// Unique-index collision means someone else already voted for this
			// (issueID, userID). Read the current counter and return.
			if isUniqueViolation(err) {
				return tx.Model(&domain.Issue{}).Select("upvote_count").
					Where("id = ?", issueID).Scan(&count).Error
			}
			return err
		}
		if err := tx.Model(&domain.Issue{}).Where("id = ?", issueID).
			UpdateColumn("upvote_count", gorm.Expr("upvote_count + 1")).Error; err != nil {
			return err
		}
		return tx.Model(&domain.Issue{}).Select("upvote_count").
			Where("id = ?", issueID).Scan(&count).Error
	})
	return count, err
}

// RemoveUpvote deletes the (issue, user) row and decrements the counter. If
// no row exists this is a no-op (returns the current count untouched).
func (r *Repository) RemoveUpvote(issueID, userID string) (int, error) {
	var count int
	err := r.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Where("issue_id = ? AND user_id = ?", issueID, userID).
			Delete(&domain.IssueUpvote{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected > 0 {
			// Guard against negative counts if a prior bug left the counter
			// out of sync — clamp at zero.
			if err := tx.Model(&domain.Issue{}).Where("id = ? AND upvote_count > 0", issueID).
				UpdateColumn("upvote_count", gorm.Expr("upvote_count - 1")).Error; err != nil {
				return err
			}
		}
		return tx.Model(&domain.Issue{}).Select("upvote_count").
			Where("id = ?", issueID).Scan(&count).Error
	})
	return count, err
}

// ListUpvotedIssueIDsByUser returns every issue this user has an active
// upvote on. The list is small (usually << 1000) so we don't paginate.
func (r *Repository) ListUpvotedIssueIDsByUser(userID string) ([]string, error) {
	var ids []string
	err := r.db.Model(&domain.IssueUpvote{}).
		Where("user_id = ?", userID).
		Pluck("issue_id", &ids).Error
	return ids, err
}

// isUniqueViolation matches Postgres's SQLSTATE 23505 by string sniff so we
// don't have to import pgconn just for one branch. GORM wraps the error but
// keeps the driver's Code() reachable through errors.Is on a subset of drivers;
// checking the message is the portable option here.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// Common markers across pgx/lib-pq wrappers.
	msg := err.Error()
	return errors.Is(err, gorm.ErrDuplicatedKey) ||
		containsAny(msg, "duplicate key", "SQLSTATE 23505", "unique constraint")
}

func containsAny(s string, needles ...string) bool {
	for _, n := range needles {
		if len(s) >= len(n) && indexOf(s, n) >= 0 {
			return true
		}
	}
	return false
}

func indexOf(s, sub string) int {
	// Small local helper so the file stays free of the strings import churn.
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func (r *Repository) UpdateStatus(id string, status domain.IssueStatus) error {
	return r.db.Model(&domain.Issue{}).Where("id = ?", id).
		Update("status", status).Error
}

func (r *Repository) ListComments(issueID string) ([]domain.IssueComment, error) {
	var list []domain.IssueComment
	return list, r.db.Where("issue_id = ?", issueID).
		Order("created_at asc").Find(&list).Error
}

func (r *Repository) AddComment(comment *domain.IssueComment) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(comment).Error; err != nil {
			return err
		}
		return tx.Model(&domain.Issue{}).Where("id = ?", comment.IssueID).
			UpdateColumn("comment_count", gorm.Expr("comment_count + 1")).Error
	})
}
