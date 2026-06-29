package notifications

import (
	"github.com/civicos/community-service/internal/domain"
	"github.com/google/uuid"
)

type NotificationStore interface {
	ListByUser(userID string, limit int) ([]domain.Notification, error)
	CountUnread(userID string) (int64, error)
	Create(n *domain.Notification) error
	MarkRead(id, userID string) error
	MarkAllRead(userID string) error
}

type Service struct{ repo NotificationStore }

func NewService(repo NotificationStore) *Service { return &Service{repo: repo} }

func (s *Service) List(userID string, limit int) ([]domain.Notification, error) {
	return s.repo.ListByUser(userID, limit)
}

func (s *Service) UnreadCount(userID string) (int64, error) {
	return s.repo.CountUnread(userID)
}

func (s *Service) MarkRead(id, userID string) error {
	return s.repo.MarkRead(id, userID)
}

func (s *Service) MarkAllRead(userID string) error {
	return s.repo.MarkAllRead(userID)
}

// Emit creates a notification for the given user. Skips silently if userID is empty
// (e.g., when a user has no observers to notify or is the source of the event).
func (s *Service) Emit(userID string, t domain.NotificationType, title, body string, linkURL *string) error {
	if userID == "" {
		return nil
	}
	n := &domain.Notification{
		ID:      uuid.New().String(),
		Type:    t,
		Title:   title,
		Body:    body,
		LinkURL: linkURL,
		UserID:  userID,
	}
	return s.repo.Create(n)
}
