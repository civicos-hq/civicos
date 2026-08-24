package issues

import (
	"errors"
	"net/http"

	"github.com/civicos/community-service/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IssueStore interface {
	FindAll(communityID, status, category string) ([]domain.Issue, error)
	FindByID(id string) (*domain.Issue, error)
	Create(issue *domain.Issue) error
	HasUserUpvoted(issueID, userID string) (bool, error)
	AddUpvote(issueID, userID string) (int, error)
	RemoveUpvote(issueID, userID string) (int, error)
	ListUpvotedIssueIDsByUser(userID string) ([]string, error)
	UpdateStatus(id string, status domain.IssueStatus) error
	ListComments(issueID string) ([]domain.IssueComment, error)
	AddComment(comment *domain.IssueComment) error
}

// OfficialRoles is the set of roles whose comments are flagged as official responses.
var OfficialRoles = map[string]bool{
	"REPRESENTATIVE":   true,
	"GOVERNMENT_ADMIN": true,
	"PLATFORM_ADMIN":   true,
	"NGO":              true,
	"MODERATOR":        true,
}

type Service struct{ repo IssueStore }

func NewService(repo IssueStore) *Service { return &Service{repo: repo} }

type CreateInput struct {
	Title       string               `json:"title" binding:"required,min=5"`
	Description string               `json:"description" binding:"required,min=10"`
	Category    domain.IssueCategory `json:"category" binding:"required"`
	CommunityID string               `json:"communityId" binding:"required"`
	Location    *string              `json:"location"`
	ImageURLs   []string             `json:"imageUrls"`
	// One video per issue, with its client-generated poster frame and byte
	// size alongside. The client uploads all three parts (video, poster,
	// size) before it posts the issue; we validate the shape here rather
	// than trusting the arrays to line up.
	VideoURLs       []string `json:"videoUrls" binding:"max=1"`
	VideoPosterURLs []string `json:"videoPosterUrls" binding:"max=1"`
	VideoSizeBytes  []int64  `json:"videoSizeBytes" binding:"max=1"`
}

// MaxVideosPerIssue caps video attachments. Videos are heavy for both storage
// and the reader's data plan; one clip is enough to show a pothole or a burst
// pipe, and a text-only report stays fully first-class.
const MaxVideosPerIssue = 1

type AppError struct {
	Code    string
	Message string
	Status  int
}

func (e *AppError) Error() string { return e.Message }

func (s *Service) List(communityID, status, category string) ([]domain.Issue, error) {
	return s.repo.FindAll(communityID, status, category)
}

func (s *Service) Get(id string) (*domain.Issue, error) {
	issue, err := s.repo.FindByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, &AppError{Code: "ISSUE_NOT_FOUND", Message: "Issue not found", Status: http.StatusNotFound}
	}
	return issue, err
}

func (s *Service) Create(input CreateInput, reportedByID string) (*domain.Issue, error) {
	images := input.ImageURLs
	if images == nil {
		images = []string{}
	}
	videos, posters, sizes, err := normalizeVideos(input)
	if err != nil {
		return nil, err
	}
	issue := &domain.Issue{
		ID:              uuid.New().String(),
		Title:           input.Title,
		Description:     input.Description,
		Category:        input.Category,
		Status:          domain.IssueStatusOpen,
		Location:        input.Location,
		ImageURLs:       images,
		VideoURLs:       videos,
		VideoPosterURLs: posters,
		VideoSizeBytes:  sizes,
		CommunityID:     input.CommunityID,
		ReportedByID:    reportedByID,
	}
	return issue, s.repo.Create(issue)
}

// normalizeVideos enforces the video cap and keeps the three positional
// arrays the same length, so a reader can always index posters[i]/sizes[i]
// for videos[i]. A missing poster is allowed (the player falls back to a
// neutral placeholder) — a mismatched one is not.
func normalizeVideos(input CreateInput) (videos, posters []string, sizes []int64, err error) {
	videos = input.VideoURLs
	if videos == nil {
		videos = []string{}
	}
	if len(videos) > MaxVideosPerIssue {
		return nil, nil, nil, &AppError{
			Code:    "TOO_MANY_VIDEOS",
			Message: "An issue can have at most one video",
			Status:  http.StatusBadRequest,
		}
	}

	posters = make([]string, len(videos))
	sizes = make([]int64, len(videos))
	for i := range videos {
		if videos[i] == "" {
			return nil, nil, nil, &AppError{
				Code:    "INVALID_VIDEO",
				Message: "Video attachment is missing its file reference",
				Status:  http.StatusBadRequest,
			}
		}
		if i < len(input.VideoPosterURLs) {
			posters[i] = input.VideoPosterURLs[i]
		}
		if i < len(input.VideoSizeBytes) {
			sizes[i] = input.VideoSizeBytes[i]
		}
	}
	return videos, posters, sizes, nil
}

// ToggleUpvote flips the caller's upvote on the issue. Returns the state
// they should now see (upvoted true/false) and the fresh counter. Prior
// behaviour blindly incremented per click — this dedups via the unique
// (issue_id, user_id) constraint on IssueUpvote.
func (s *Service) ToggleUpvote(issueID, userID string) (upvoted bool, count int, err error) {
	if _, err := s.Get(issueID); err != nil {
		return false, 0, err
	}
	has, err := s.repo.HasUserUpvoted(issueID, userID)
	if err != nil {
		return false, 0, err
	}
	if has {
		n, err := s.repo.RemoveUpvote(issueID, userID)
		return false, n, err
	}
	n, err := s.repo.AddUpvote(issueID, userID)
	return true, n, err
}

func (s *Service) ListUpvotedIssueIDs(userID string) ([]string, error) {
	return s.repo.ListUpvotedIssueIDsByUser(userID)
}

func (s *Service) UpdateStatus(id string, status domain.IssueStatus) error {
	return s.repo.UpdateStatus(id, status)
}

func (s *Service) ListComments(issueID string) ([]domain.IssueComment, error) {
	return s.repo.ListComments(issueID)
}

type CommentInput struct {
	Content string `json:"content" binding:"required,min=1,max=2000"`
}

func (s *Service) AddComment(issueID, authorID, authorName, authorRole, content string) (*domain.IssueComment, error) {
	comment := &domain.IssueComment{
		ID:                 uuid.New().String(),
		Content:            content,
		IssueID:            issueID,
		AuthorID:           authorID,
		AuthorName:         authorName,
		AuthorRole:         authorRole,
		IsOfficialResponse: OfficialRoles[authorRole],
	}
	return comment, s.repo.AddComment(comment)
}
