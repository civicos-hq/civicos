// Package notifications is a thin writer to the shared notifications table.
// The schema is owned by community-service — this service only INSERTs.
// Same shared-DB pattern as the audit package.
//
// If services move to isolated databases later, replace the direct DB
// insert with a NATS publish or an HTTP call to a notifications endpoint.
package notifications

import (
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationType mirrors the enum in community-service/internal/domain.
// Keep the string values in sync — community-service owns the schema.
type NotificationType string

const (
	TypeConsultationUpdate NotificationType = "CONSULTATION_UPDATE"
	TypeAnnouncementUpdate NotificationType = "ANNOUNCEMENT_UPDATE"
)

// Notification is a re-declaration of community-service's model with
// TableName pinned to notifications so GORM writes into the right table
// without an FK cycle.
type Notification struct {
	ID        string           `gorm:"type:uuid;primaryKey" json:"id"`
	Type      NotificationType `gorm:"type:varchar(40);not null" json:"type"`
	Title     string           `gorm:"not null" json:"title"`
	Body      string           `gorm:"not null" json:"body"`
	Read      bool             `gorm:"default:false;index" json:"read"`
	LinkURL   *string          `json:"linkUrl,omitempty"`
	UserID    string           `gorm:"type:uuid;not null;index" json:"userId"`
	CreatedAt time.Time        `json:"createdAt"`
}

func (Notification) TableName() string { return "notifications" }

// Notifier is the abstraction consumers (handlers, services) depend on.
// Implementations must be safe to call from any goroutine — the direct-DB
// writer below is, since GORM's *DB is concurrency-safe.
type Notifier interface {
	Emit(userID string, t NotificationType, title, body string, linkURL *string) error
	EmitMany(userIDs []string, t NotificationType, title, body string, linkURL *string)
}

// DBNotifier writes notifications directly to the shared notifications
// table. Failures are logged but never returned to the caller — a
// notification miss should not block the primary action (publish a
// consultation, etc.).
type DBNotifier struct{ db *gorm.DB }

func NewDBNotifier(db *gorm.DB) *DBNotifier { return &DBNotifier{db: db} }

func (n *DBNotifier) Emit(userID string, t NotificationType, title, body string, linkURL *string) error {
	if userID == "" || title == "" {
		return nil
	}
	row := &Notification{
		ID:        uuid.New().String(),
		Type:      t,
		Title:     title,
		Body:      body,
		LinkURL:   linkURL,
		UserID:    userID,
		CreatedAt: time.Now().UTC(),
	}
	if err := n.db.Create(row).Error; err != nil {
		log.Printf("⚠️  notify: write failed userId=%s type=%s err=%v", userID, t, err)
		return err
	}
	return nil
}

// EmitMany fans out to a slice of recipients. Individual failures are
// logged and skipped — one bad row must not stop the rest.
func (n *DBNotifier) EmitMany(userIDs []string, t NotificationType, title, body string, linkURL *string) {
	for _, uid := range userIDs {
		_ = n.Emit(uid, t, title, body, linkURL)
	}
}
