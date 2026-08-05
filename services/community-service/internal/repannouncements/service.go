// Package repannouncements lets a representative speak to their constituents
// directly, rather than only replying inside somebody else's thread.
//
// # Who may publish
//
// Exactly one account: the user the representative profile belongs to
// (`representatives.user_id`, set when their application is approved).
//
// This is deliberately NOT a role check. `REPRESENTATIVE` in a JWT says
// "this person is some representative", not "this person is THAT
// representative" — and a role check would let any approved rep publish in
// any other rep's name, to their followers, under their photograph. Platform
// admins are excluded too: an admin who could post as an elected official
// could put words in their mouth, and no audit trail undoes that once the
// notification has gone out.
//
// A profile with no linked account is unclaimed, and nobody can publish as it.
package repannouncements

import (
	"net/http"
	"strings"
	"time"

	"github.com/civicos/community-service/internal/domain"
	"github.com/google/uuid"
)

type AppError struct {
	Code    string
	Message string
	Status  int
}

func (e *AppError) Error() string { return e.Message }

type Store interface {
	Representative(id string) (*domain.Representative, error)
	Create(a *domain.RepresentativeAnnouncement) error
	FindByID(id string) (*domain.RepresentativeAnnouncement, error)
	Update(id string, fields map[string]any) error
	Delete(id string) error
	// ListPublic returns PUBLISHED announcements only, newest first.
	ListPublic(repID string) ([]domain.RepresentativeAnnouncement, error)
	// ListAll includes drafts and archived — owner's view.
	ListAll(repID string) ([]domain.RepresentativeAnnouncement, error)
	// FollowerIDs is the fan-out audience on publish.
	FollowerIDs(repID string) ([]string, error)
	ListComments(annID string) ([]domain.RepresentativeAnnouncementComment, error)
	AddComment(c *domain.RepresentativeAnnouncementComment) error
}

// Notifier is the subset of the notification service this package needs.
type Notifier interface {
	Emit(userID string, t domain.NotificationType, title, body string, linkURL *string) error
}

type Service struct {
	repo     Store
	notifier Notifier
}

func NewService(repo Store, notifier Notifier) *Service {
	return &Service{repo: repo, notifier: notifier}
}

type CreateInput struct {
	Title string `json:"title" binding:"required,min=4,max=200"`
	Body  string `json:"body" binding:"required,min=20"`
}

type UpdateInput struct {
	Title *string `json:"title"`
	Body  *string `json:"body"`
}

// requireOwner is the single gate every write goes through.
func (s *Service) requireOwner(repID, userID string) (*domain.Representative, error) {
	rep, err := s.repo.Representative(repID)
	if err != nil || rep == nil {
		return nil, &AppError{Code: "REPRESENTATIVE_NOT_FOUND", Message: "Representative not found", Status: http.StatusNotFound}
	}
	if rep.UserID == nil || *rep.UserID == "" {
		// Says what is wrong and who fixes it. An unclaimed profile is an
		// administrative gap, not the caller doing something forbidden.
		return nil, &AppError{
			Code:    "REPRESENTATIVE_UNCLAIMED",
			Message: "This representative profile is not linked to an account yet. A platform admin has to link it before anything can be published.",
			Status:  http.StatusConflict,
		}
	}
	if *rep.UserID != userID {
		return nil, &AppError{
			Code:    "NOT_YOUR_PROFILE",
			Message: "Only this representative can publish here",
			Status:  http.StatusForbidden,
		}
	}
	return rep, nil
}

func (s *Service) Create(repID, userID, userName string, in CreateInput) (*domain.RepresentativeAnnouncement, error) {
	rep, err := s.requireOwner(repID, userID)
	if err != nil {
		return nil, err
	}
	a := &domain.RepresentativeAnnouncement{
		ID:               uuid.New().String(),
		RepresentativeID: rep.ID,
		CommunityID:      rep.CommunityID,
		Title:            strings.TrimSpace(in.Title),
		Body:             strings.TrimSpace(in.Body),
		// Always starts as a draft. Nothing reaches a constituent because
		// somebody clicked Save.
		Status:     domain.AnnouncementDraft,
		AuthorID:   userID,
		AuthorName: userName,
	}
	if err := s.repo.Create(a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *Service) Update(repID, annID, userID string, in UpdateInput) (*domain.RepresentativeAnnouncement, error) {
	if _, err := s.requireOwner(repID, userID); err != nil {
		return nil, err
	}
	a, err := s.owned(repID, annID)
	if err != nil {
		return nil, err
	}
	// Published text is frozen. Constituents were notified about specific
	// words; silently editing them afterwards turns a public statement into
	// something nobody can rely on having read. Archive and publish anew.
	if a.Status != domain.AnnouncementDraft {
		return nil, &AppError{
			Code:    "ALREADY_PUBLISHED",
			Message: "A published announcement cannot be edited. Archive it and publish a new one.",
			Status:  http.StatusConflict,
		}
	}
	fields := map[string]any{}
	if in.Title != nil && strings.TrimSpace(*in.Title) != "" {
		fields["title"] = strings.TrimSpace(*in.Title)
	}
	if in.Body != nil && strings.TrimSpace(*in.Body) != "" {
		fields["body"] = strings.TrimSpace(*in.Body)
	}
	if len(fields) == 0 {
		return a, nil
	}
	if err := s.repo.Update(annID, fields); err != nil {
		return nil, err
	}
	return s.repo.FindByID(annID)
}

// Publish makes it visible and tells the representative's followers.
//
// The fan-out is best effort: a notification that fails to send must not roll
// back a statement the representative has already made public. Followers can
// still see it on the profile.
func (s *Service) Publish(repID, annID, userID string) (*domain.RepresentativeAnnouncement, error) {
	rep, err := s.requireOwner(repID, userID)
	if err != nil {
		return nil, err
	}
	a, err := s.owned(repID, annID)
	if err != nil {
		return nil, err
	}
	if a.Status == domain.AnnouncementPublished {
		return a, nil // idempotent; a double-click must not notify twice
	}
	now := time.Now().UTC()
	if err := s.repo.Update(annID, map[string]any{
		"status":       domain.AnnouncementPublished,
		"published_at": now,
	}); err != nil {
		return nil, err
	}
	s.notifyFollowers(rep, a)
	return s.repo.FindByID(annID)
}

func (s *Service) notifyFollowers(rep *domain.Representative, a *domain.RepresentativeAnnouncement) {
	if s.notifier == nil {
		return
	}
	ids, err := s.repo.FollowerIDs(rep.ID)
	if err != nil {
		return
	}
	link := "/representatives/" + rep.ID
	for _, uid := range ids {
		// The representative does not need telling about their own post.
		if uid == a.AuthorID {
			continue
		}
		_ = s.notifier.Emit(uid, domain.NotificationRepresentativeAnnouncement,
			rep.Name+": "+a.Title, truncate(a.Body, 140), &link)
	}
}

// Archive takes a published announcement out of public view without deleting
// it. The record that the statement was made survives; only its visibility
// changes.
func (s *Service) Archive(repID, annID, userID string) (*domain.RepresentativeAnnouncement, error) {
	if _, err := s.requireOwner(repID, userID); err != nil {
		return nil, err
	}
	if _, err := s.owned(repID, annID); err != nil {
		return nil, err
	}
	if err := s.repo.Update(annID, map[string]any{"status": domain.AnnouncementArchived}); err != nil {
		return nil, err
	}
	return s.repo.FindByID(annID)
}

// Delete removes a draft. Anything that was ever published is archived, never
// deleted — a statement a constituency was notified about should not be able
// to vanish.
func (s *Service) Delete(repID, annID, userID string) error {
	if _, err := s.requireOwner(repID, userID); err != nil {
		return err
	}
	a, err := s.owned(repID, annID)
	if err != nil {
		return err
	}
	if a.Status != domain.AnnouncementDraft {
		return &AppError{
			Code:    "ARCHIVE_INSTEAD",
			Message: "Only a draft can be deleted. Archive a published announcement instead.",
			Status:  http.StatusConflict,
		}
	}
	return s.repo.Delete(annID)
}

func (s *Service) ListPublic(repID string) ([]domain.RepresentativeAnnouncement, error) {
	return s.repo.ListPublic(repID)
}

// ListMine is the owner's view: drafts and archived included.
func (s *Service) ListMine(repID, userID string) ([]domain.RepresentativeAnnouncement, error) {
	if _, err := s.requireOwner(repID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListAll(repID)
}

// owned resolves an announcement and confirms it belongs to this
// representative — so an id from another profile cannot be operated on by
// pairing it with a profile the caller does own.
func (s *Service) owned(repID, annID string) (*domain.RepresentativeAnnouncement, error) {
	a, err := s.repo.FindByID(annID)
	if err != nil || a == nil {
		return nil, &AppError{Code: "ANNOUNCEMENT_NOT_FOUND", Message: "Announcement not found", Status: http.StatusNotFound}
	}
	if a.RepresentativeID != repID {
		return nil, &AppError{Code: "ANNOUNCEMENT_NOT_FOUND", Message: "Announcement not found", Status: http.StatusNotFound}
	}
	return a, nil
}

func truncate(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return strings.TrimSpace(string(r[:n])) + "…"
}

// ─── Comments ───────────────────────────────────────────────────────────

// OfficialRoles mirrors the set used on issues, petitions and representative
// profiles, so the "official response" badge means the same thing wherever a
// citizen sees it.
var OfficialRoles = map[string]bool{
	"REPRESENTATIVE":   true,
	"GOVERNMENT_ADMIN": true,
	"PLATFORM_ADMIN":   true,
	"NGO":              true,
	"MODERATOR":        true,
}

type CommentInput struct {
	Content string `json:"content" binding:"required,min=2,max=2000"`
}

// ListComments is public: the thread under a published announcement is part
// of the public record, readable by anyone who can read the announcement.
func (s *Service) ListComments(repID, annID string) ([]domain.RepresentativeAnnouncementComment, error) {
	a, err := s.owned(repID, annID)
	if err != nil {
		return nil, err
	}
	// A draft has no public thread — it has never been visible, so there is
	// nothing anyone could have replied to.
	if a.Status == domain.AnnouncementDraft {
		return []domain.RepresentativeAnnouncementComment{}, nil
	}
	return s.repo.ListComments(annID)
}

// AddComment lets a verified citizen reply.
//
// Only on a PUBLISHED announcement: a draft is not public, and an archived one
// has been withdrawn — reopening a thread on something the representative has
// taken down would put words under a statement they have retracted.
func (s *Service) AddComment(repID, annID, authorID, authorName, authorRole, content string) (*domain.RepresentativeAnnouncementComment, error) {
	a, err := s.owned(repID, annID)
	if err != nil {
		return nil, err
	}
	if a.Status != domain.AnnouncementPublished {
		return nil, &AppError{
			Code:    "NOT_OPEN_FOR_COMMENT",
			Message: "This announcement is not open for comments",
			Status:  http.StatusConflict,
		}
	}
	c := &domain.RepresentativeAnnouncementComment{
		ID:                 uuid.New().String(),
		AnnouncementID:     annID,
		Content:            strings.TrimSpace(content),
		AuthorID:           authorID,
		AuthorName:         authorName,
		AuthorRole:         authorRole,
		IsOfficialResponse: OfficialRoles[authorRole],
	}
	if err := s.repo.AddComment(c); err != nil {
		return nil, err
	}
	return c, nil
}
